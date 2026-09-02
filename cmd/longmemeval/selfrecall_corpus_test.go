package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// selfRecallN is how many sessions the replay writes. Each write runs winze's
// build gate (~2s), so this is the knob between a minute of wall clock and ten.
const selfRecallN = 20

// TestSelfRecallDecaysWithCorpusGrowth is Phase 3b's first measurement, and it
// spends nothing: no answerer, no judge, no API calls.
//
// docs/agent-identity-integration.md's open question is whether a memory an
// agent wrote about its own session stays findable once the store keeps
// growing around it. That is a retrieval-rank question before it is a
// comprehension question, and rank is deterministic. Adding an LLM answerer
// here would put a second, noisier system between the write and the number,
// and would cost real money to discover a ranking bug a sort could have shown.
// Promote to answer+judge only if this curve bends.
//
// Two probes per note, because one of them is too easy. Querying by the
// session title asks the store to find a note by the note's own opening words,
// which is close to a lookup; it is kept as the control. The probe that
// carries the result is a mid-session user turn (LaterAsk) that was never
// written into any note -- real text from the same session, in wording the
// store has never seen, which is the shape a cold agent actually arrives with.
//
// A dedup rejection is DATA, not an error. The first run of this measured 20
// writes and 2 rejections at cosine 0.73-0.74 -- against other session notes
// in the same replay, not against semantic duplicates -- while every note that
// did land came back at rank 1. Counting a rejection as a test failure would
// bury the one number worth having behind a red result.
//
// Skips without the corpus or the built binaries, so it is an instrument
// rather than a CI gate -- the same status TestAskOnceReplayAgainstTheRealLog
// carries.
func TestSelfRecallDecaysWithCorpusGrowth(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	dir := os.Getenv("WINZE_TRANSCRIPT_DIR")
	if dir == "" {
		dir = filepath.Join(home, ".claude", "projects", "-home-gas6amus-Documents")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no transcript corpus at %s", dir)
	}
	bin := os.Getenv("WINZE_BIN")
	if bin == "" {
		bin = filepath.Join(home, "Documents", "winze", "bin")
	}
	agent := filepath.Join(bin, "winze-agent")
	if _, err := os.Stat(agent); err != nil {
		t.Skipf("no winze-agent at %s (run `make build`)", agent)
	}

	all, err := readProjectTranscripts(dir, 4)
	if err != nil {
		t.Fatalf("reading corpus: %v", err)
	}
	var usable []*transcriptSession
	for _, s := range all {
		if s.Title != "" && s.OpeningAsk() != "" {
			usable = append(usable, s)
		}
	}
	if len(usable) < 4 {
		t.Skipf("only %d sessions carry both a title and an opening ask", len(usable))
	}
	picked := stratify(usable, selfRecallCount())
	t.Logf("replaying %d of %d usable sessions, %s .. %s",
		len(picked), len(usable),
		picked[0].Start.Format("2006-01-02"), picked[len(picked)-1].Start.Format("2006-01-02"))

	store := os.Getenv("WINZE_SELFRECALL_STORE")
	if store == "" {
		store = filepath.Join(t.TempDir(), "store")
	} else if err := os.RemoveAll(store); err != nil {
		t.Fatalf("clearing persistent store %s: %v", store, err)
	}
	env := append(os.Environ(),
		"WINZE_STORE="+store, "WINZE_BIN="+bin,
		"GIT_AUTHOR_NAME=selfrecall", "GIT_AUTHOR_EMAIL=selfrecall@localhost",
		"GIT_COMMITTER_NAME=selfrecall", "GIT_COMMITTER_EMAIL=selfrecall@localhost",
	)
	run := func(args ...string) (string, error) {
		cmd := exec.Command(agent, args...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}
	if out, err := run("init", store, "--from", repo); err != nil {
		t.Fatalf("scaffolding store: %v\n%s", err, out)
	}

	// Replay oldest-first so each note is written into a store holding every
	// earlier note and none of the later ones -- the growth the design doc asks for.
	vars, rejected := writeSessions(t, run, picked)
	assertRawTier(t, store, picked, rejected)

	// The store is now at full size. Probe each stored note twice; see probeAll's
	// doc comment for what TITLE and LATER each establish.
	title, later, noLater := probeAll(t, run, picked, vars, os.Getenv("WINZE_SELFRECALL_MANIFEST"))
	if title.found == 0 {
		t.Fatalf("no note was recalled by its own title at any rank -- %d missing", title.miss)
	}
	t.Logf("TITLE PROBE: %d/%d recalled, mean rank %.2f, %d never surfaced",
		title.found, title.found+title.miss, title.meanRank(), title.miss)
	if later.found == 0 {
		t.Logf("LATER PROBE: nothing surfaced across %d probes (%d sessions had no second ask)",
			later.miss, noLater)
	} else {
		t.Logf("LATER PROBE: %d/%d recalled from text never written into a note, mean rank %.2f, "+
			"%d never surfaced, %d sessions had no second ask",
			later.found, later.found+later.miss, later.meanRank(), later.miss, noLater)
	}
	t.Logf("write-rejection rate %d/%d (%.0f%%) at store size %d",
		len(rejected), len(picked), 100*float64(len(rejected))/float64(len(picked)), len(picked))
}

// createdVar scrapes the entity name out of winze-agent's write confirmation,
// which reads "created entity SomeName (Concept) in memory.go (...)".
func createdVar(out string) string {
	for _, line := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "created entity ")
		if !ok {
			continue
		}
		if name, _, ok := strings.Cut(rest, " "); ok {
			return name
		}
	}
	return ""
}

// mustJSON encodes a string as a JSON literal for embedding in a tool payload.
func mustJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

// noteFor renders the memory note for a session, in one of two shapes chosen
// by $WINZE_NOTE_SHAPE.
//
// "open" (default) is title plus the operator's first ask -- the cheapest note
// that could work, and the shape the first 140-session run measured.
//
// "arc" adds the session's later asks, minus the one LaterAsk holds out as the
// probe. That holdout is the whole point: dropping every ask into the note
// would make the later probe a title probe in different clothes, and the run
// would report a retrieval win that was really the answer being written into
// the question. Comparing the two shapes at the same store size separates "the
// store cannot bridge unseen wording" from "the note did not describe the
// session" -- which the first run could not tell apart.
//
// Both shapes cost nothing: every word is already on disk.
func noteFor(s *transcriptSession) string {
	ask := s.OpeningAsk()
	if len(ask) > 1200 {
		ask = ask[:1200] + "…"
	}
	note := fmt.Sprintf("Session %s (%s): %s\n\nOpened with: %s",
		s.Start.Format("2006-01-02"), s.ID[:8], s.Title, ask)
	if os.Getenv("WINZE_NOTE_SHAPE") != "arc" {
		return note
	}
	held := s.LaterAsk()
	var arc []string
	budget := 1500
	for _, a := range s.ArcAsks() {
		if a == held || budget <= 0 {
			continue
		}
		if len(a) > 300 {
			a = a[:300] + "…"
		}
		arc = append(arc, a)
		budget -= len(a)
	}
	if len(arc) == 0 {
		return note
	}
	return note + "\n\nWent on to: " + strings.Join(arc, " / ")
}

// rankLabel renders a rank, with 0 meaning the recall never surfaced the note.
func rankLabel(rank int) string {
	if rank == 0 {
		return "MISS"
	}
	return strconv.Itoa(rank)
}

// rankOf returns the 1-based position of varName in a recall result, or 0 when
// the recall did not surface it at all.
func rankOf(hits recallHits, varName string) int {
	for i, h := range hits.Hits {
		if h.VarName == varName {
			return i + 1
		}
	}
	return 0
}

// stratify picks n sessions spread evenly across a chronological slice.
//
// Evenly across the calendar, not the most recent n: age is the independent
// variable. Taking the newest n would hold corpus-growth-since-write nearly
// constant and measure nothing, which is the same shape of mistake as the N=1
// same-day trial this whole phase exists to improve on.
func stratify(sessions []*transcriptSession, n int) []*transcriptSession {
	if n >= len(sessions) {
		return sessions
	}
	out := make([]*transcriptSession, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, sessions[i*len(sessions)/n])
	}
	return out
}

// recallHits is the shape winze_recall returns.
type recallHits struct {
	Matched int `json:"matched"`
	Hits    []struct {
		VarName string `json:"var_name"`
	} `json:"hits"`
}

// dedupReason pulls the cosine score and the blocking memory out of a
// "NOT stored" response, so a rejection reads as a measurement rather than as
// an unexplained absence.
func dedupReason(out string) string {
	var score, against string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if _, after, ok := strings.Cut(line, "(cosine "); ok {
			score, _, _ = strings.Cut(after, ")")
		}
		if strings.HasPrefix(line, "Session ") && strings.Contains(line, "] — ") {
			against, _, _ = strings.Cut(line, " — ")
		}
	}
	if score == "" {
		return "no entity created, and no dedup message to explain it"
	}
	if against == "" {
		return "blocked at cosine " + score
	}
	return "blocked at cosine " + score + " against " + against
}

// selfRecallCount is how many sessions the replay writes, overridable with
// $WINZE_SELFRECALL_N. The default keeps the committed test to about a minute;
// the override is what a real measurement run uses, since store size is the
// independent variable and 20 notes is a small store to draw a curve through.
func selfRecallCount() int {
	if v := os.Getenv("WINZE_SELFRECALL_N"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return selfRecallN
}

// probeStats tallies one probe's outcomes across the replay -- pulled out of
// TestSelfRecallDecaysWithCorpusGrowth so the per-session loop has one thing
// to update instead of three counters threaded through by hand.
type probeStats struct {
	found, miss, rankSum int
}

func (p *probeStats) record(rank int) {
	if rank == 0 {
		p.miss++
	} else {
		p.found++
		p.rankSum += rank
	}
}

func (p probeStats) meanRank() float64 {
	if p.found == 0 {
		return 0
	}
	return float64(p.rankSum) / float64(p.found)
}

// assertRawTier is Phase 3a's whole claim: appendRawLog runs on the fourth
// line of handleRemember, before the dedup gate, so a note the gate refuses
// is meant to survive anyway. Pulled out of TestSelfRecallDecaysWithCorpusGrowth
// alongside writeSessions, which is the only caller that has a rejected list
// worth checking this against.
func assertRawTier(t *testing.T, store string, picked []*transcriptSession, rejected []string) {
	t.Helper()
	rawPath := filepath.Join(store, "raw.jsonl")
	rawBytes, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("raw tier absent at %s: %v", rawPath, err)
	}
	rawLines := strings.Count(strings.TrimSpace(string(rawBytes)), "\n") + 1
	if rawLines != len(picked) {
		t.Errorf("raw tier holds %d entries, want %d (one per attempted write)", rawLines, len(picked))
	}
	for _, r := range rejected {
		title, _, _ := strings.Cut(strings.TrimPrefix(r[11:], `"`), `"`)
		if !strings.Contains(string(rawBytes), title) {
			t.Errorf("rejected note %q is not in the raw tier — it is genuinely lost", title)
		}
	}
	t.Logf("raw tier: %d entries for %d attempted writes; all %d rejected notes recoverable",
		rawLines, len(picked), len(rejected))
}

// probeAll runs both probes (title, then LaterAsk) against every stored
// session and optionally dumps a manifest -- pulled out of
// TestSelfRecallDecaysWithCorpusGrowth, which had grown to interleave the
// probe calls, the manifest dump, and three counters in one loop.
//
// TITLE is the note's own first line, so a hit says the store can find a
// memory by its own words -- necessary, but nearer a lookup than a recall.
// LATER is a mid-session user turn that was never written into any note, so
// a hit says retrieval bridged from wording the store has never seen. That
// second number is the one worth having; the first is its control.
func probeAll(t *testing.T, run func(args ...string) (string, error), picked []*transcriptSession, vars []string, manifestPath string) (title, later probeStats, noLater int) {
	t.Helper()
	probe := func(query, want string) (int, error) {
		payload := fmt.Sprintf(`{"query":%s,"limit":%d,"brief_chars":0}`, mustJSON(query), len(picked))
		out, err := run("call", "winze_recall", payload)
		if err != nil {
			return 0, fmt.Errorf("%v\n%s", err, out)
		}
		var hits recallHits
		if err := json.Unmarshal([]byte(out), &hits); err != nil {
			return 0, fmt.Errorf("unparseable JSON: %v\n%s", err, out)
		}
		return rankOf(hits, want), nil
	}

	// WINZE_SELFRECALL_MANIFEST, when set, dumps one JSON line per session with
	// the exact query text used for each probe and its var name -- the detail a
	// summary line can't carry, needed to replay one session's probe by hand
	// against the persistent store (WINZE_SELFRECALL_STORE) after the test exits.
	var manifest *os.File
	if manifestPath != "" {
		var err error
		manifest, err = os.Create(manifestPath)
		if err != nil {
			t.Fatalf("creating manifest %s: %v", manifestPath, err)
		}
		defer manifest.Close()
	}

	t.Logf("%-4s %-12s %-6s %-6s %-6s %s", "idx", "date", "after", "title", "later", "session")
	for i, s := range picked {
		if vars[i] == "" {
			continue
		}
		titleRank, err := probe(s.Title, vars[i])
		if err != nil {
			t.Errorf("title probe %d: %v", i, err)
			continue
		}
		title.record(titleRank)

		laterRank, laterLabel := 0, "n/a"
		if q := s.LaterAsk(); q != "" {
			if len(q) > 400 {
				q = q[:400]
			}
			laterRank, err = probe(q, vars[i])
			if err != nil {
				t.Errorf("later probe %d: %v", i, err)
				continue
			}
			laterLabel = rankLabel(laterRank)
			later.record(laterRank)
		} else {
			noLater++
		}
		t.Logf("%-4d %-12s %-6d %-6s %-6s %s", i, s.Start.Format("2006-01-02"),
			len(picked)-1-i, rankLabel(titleRank), laterLabel, s.Title)
		if manifest != nil {
			laterQ := s.LaterAsk()
			if len(laterQ) > 400 {
				laterQ = laterQ[:400]
			}
			rec, _ := json.Marshal(map[string]any{
				"idx": i, "date": s.Start.Format("2006-01-02"), "title": s.Title, "var": vars[i],
				"title_rank": titleRank, "later_rank": laterRank, "later_ask": laterQ,
			})
			manifest.Write(append(rec, '\n'))
		}
	}
	return title, later, noLater
}

// writeSessions replays picked sessions oldest-first through winze_remember,
// pulled out of TestSelfRecallDecaysWithCorpusGrowth so the write phase and
// the probe phase are each one readable function instead of one long one.
func writeSessions(t *testing.T, run func(args ...string) (string, error), picked []*transcriptSession) (vars []string, rejected []string) {
	t.Helper()
	vars = make([]string, len(picked))
	writeStart := time.Now()
	for i, s := range picked {
		out, err := run("call", "winze_remember", `{"note":`+mustJSON(noteFor(s))+`}`)
		if err != nil {
			t.Errorf("write %d (%s) failed to execute: %v\n%s", i, s.ID[:8], err, out)
			continue
		}
		if vars[i] = createdVar(out); vars[i] == "" {
			rejected = append(rejected, fmt.Sprintf("%s %q — %s",
				s.Start.Format("2006-01-02"), s.Title, dedupReason(out)))
		}
	}
	t.Logf("%d writes in %s; %d stored, %d rejected before storage",
		len(picked), time.Since(writeStart).Round(time.Second),
		len(picked)-len(rejected), len(rejected))
	for _, r := range rejected {
		t.Logf("  rejected: %s", r)
	}
	return vars, rejected
}
