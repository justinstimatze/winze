package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// selfRecallN is how many sessions the replay writes. Each write runs winze's
// build gate (~2s), so this is the knob between a minute of wall clock and ten.
const selfRecallN = 20

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
	queryBin := filepath.Join(bin, "winze-query")
	runQuery := func(args ...string) (string, error) {
		cmd := exec.Command(queryBin, args...)
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
	varSets, noteSets, rejected, attempted := writeSessions(t, run, picked)

	// Give the store real winze_link claim edges before probing -- see
	// linkRelatedSessions's doc comment (linkbuilder_test.go) for why this runs
	// once over the complete store rather than incrementally per write.
	linked := linkRelatedSessions(t, run, runQuery, store, picked, varSets, noteSets)
	t.Logf("LINK PASS: %d claim edge(s) created across %d sessions (floor %.2f, cap %d)",
		linked, len(picked), linkSuggestScoreMirror, linkSuggestMaxMirror)

	// Flag later-ask candidates that recur, paraphrased, across other
	// sessions (a personal ritual like "pick a name, check for collisions")
	// before probing -- see flagBoilerplateAsks's doc for why those can't
	// fairly test this session's recall.
	boiler := flagBoilerplateAsks(picked)
	if len(boiler) > 0 {
		t.Logf("BOILERPLATE FILTER: %d later-ask candidate(s) flagged as cross-session ritual, excluded from probing", len(boiler))
	}

	// The store is now at full size, with real claim edges. Probe each session
	// twice; see probeAll's doc comment for what TITLE and LATER each
	// establish, and for why a session can own more than one var under
	// WINZE_NOTE_SHAPE=claims.
	title, later, noLater := probeAll(t, run, picked, varSets, noteSets, boiler, os.Getenv("WINZE_SELFRECALL_MANIFEST"))
	if title.found == 0 {
		t.Fatalf("no note was recalled by its own title at any rank -- %d missing", title.miss)
	}
	t.Logf("TITLE PROBE: %d/%d recalled, mean rank %.2f, median rank %.1f, hit@5 %.0f%%, %d never surfaced",
		title.found, title.found+title.miss, title.meanRank(), title.medianRank(), 100*title.hitRateWithin(5), title.miss)
	if later.found == 0 {
		t.Logf("LATER PROBE: nothing surfaced across %d probes (%d sessions had no second ask)",
			later.miss, noLater)
	} else {
		t.Logf("LATER PROBE: %d/%d recalled from text never written into a note, mean rank %.2f, median rank %.1f, hit@5 %.0f%%, "+
			"%d never surfaced, %d sessions had no second ask",
			later.found, later.found+later.miss, later.meanRank(), later.medianRank(), 100*later.hitRateWithin(5), later.miss, noLater)
	}
	t.Logf("write-rejection rate %d/%d (%.0f%%) at %d attempted writes for %d sessions",
		len(rejected), attempted, 100*float64(len(rejected))/float64(attempted), attempted, len(picked))
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

// noteFor renders the memory note for a session, in one of three shapes
// chosen by $WINZE_NOTE_SHAPE: "open" (default, openNote), "arc" (arcNote),
// or "outcome" (outcomeNote, falling back to openNote when no assistant
// turn precedes the probe). All three cost nothing: every word is already
// on disk. See each helper's own doc comment for what it tests and why.
func noteFor(s *transcriptSession) string {
	switch os.Getenv("WINZE_NOTE_SHAPE") {
	case "outcome":
		if note := outcomeNote(s); note != "" {
			return note
		}
		return openNote(s)
	case "arc":
		return arcNote(s)
	default:
		return openNote(s)
	}
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
		VarName string  `json:"var_name"`
		Score   float64 `json:"score"`
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

type probeStats struct {
	found, miss, rankSum int
	ranks                []int // rank of every found hit -- feeds medianRank/hitRateWithin, which meanRank alone can't answer
}

func (p *probeStats) record(rank int) {
	if rank == 0 {
		p.miss++
	} else {
		p.found++
		p.rankSum += rank
		p.ranks = append(p.ranks, rank)
	}
}

func (p probeStats) meanRank() float64 {
	if p.found == 0 {
		return 0
	}
	return float64(p.rankSum) / float64(p.found)
}

func probeAll(t *testing.T, run func(args ...string) (string, error), picked []*transcriptSession, varSets, noteSets [][]string, boiler map[string]bool, manifestPath string) (title, later probeStats, noLater int) {
	t.Helper()
	probe := func(query string, want []string) (int, error) {
		payload := fmt.Sprintf(`{"query":%s,"limit":%d,"brief_chars":0}`, mustJSON(query), len(picked)*6)
		out, err := run("call", "winze_recall", payload)
		if err != nil {
			return 0, fmt.Errorf("%v\n%s", err, out)
		}
		var hits recallHits
		if err := json.Unmarshal([]byte(out), &hits); err != nil {
			return 0, fmt.Errorf("unparseable JSON: %v\n%s", err, out)
		}
		return bestRankOf(hits, want), nil
	}

	// WINZE_SELFRECALL_MANIFEST, when set, dumps one JSON line per session with
	// the exact query text used for each probe and its var name -- the detail a
	// summary line can't carry, needed to replay one session's probe by hand
	// against the persistent store (WINZE_SELFRECALL_STORE) after the test exits.
	// "note" is the exact text written for the session (noteSets[i][0] for
	// "open"/"arc"/"outcome"; joined for "claims"' multi-fact shape) -- a
	// coverage-vs-retrieval miss audit needs it to check whether a miss's
	// LaterAsk content ever reached the note at all, not just that it ranked
	// poorly. Used 2026-09-08 for exactly that: 39 coverage misses vs 4
	// retrieval misses out of 56, on the outcome+rerank N=150 run.
	var manifest *os.File
	if manifestPath != "" {
		var err error
		manifest, err = os.Create(manifestPath)
		if err != nil {
			t.Fatalf("creating manifest %s: %v", manifestPath, err)
		}
		defer manifest.Close()
	}

	t.Logf("%-4s %-12s %-6s %-4s %-6s %-6s %s", "idx", "date", "after", "n", "title", "later", "session")
	for i, s := range picked {
		if len(varSets[i]) == 0 {
			continue
		}
		titleRank, err := probe(s.Title, varSets[i])
		if err != nil {
			t.Errorf("title probe %d: %v", i, err)
			continue
		}
		title.record(titleRank)

		laterRank, laterLabel := 0, "n/a"
		if q := s.laterAskAvoiding(boiler); q != "" {
			if len(q) > 400 {
				q = q[:400]
			}
			laterRank, err = probe(q, varSets[i])
			if err != nil {
				t.Errorf("later probe %d: %v", i, err)
				continue
			}
			laterLabel = rankLabel(laterRank)
			later.record(laterRank)
		} else {
			noLater++
		}
		t.Logf("%-4d %-12s %-6d %-4d %-6s %-6s %s", i, s.Start.Format("2006-01-02"),
			len(picked)-1-i, len(varSets[i]), rankLabel(titleRank), laterLabel, s.Title)
		if manifest != nil {
			laterQ := s.laterAskAvoiding(boiler)
			if len(laterQ) > 400 {
				laterQ = laterQ[:400]
			}
			rec, _ := json.Marshal(map[string]any{
				"idx": i, "date": s.Start.Format("2006-01-02"), "title": s.Title, "vars": varSets[i],
				"title_rank": titleRank, "later_rank": laterRank, "later_ask": laterQ,
				"note": strings.Join(noteSets[i], "\n---\n"),
			})
			manifest.Write(append(rec, '\n'))
		}
	}
	return title, later, noLater
}

// writeSessions replays picked sessions oldest-first through winze_remember,
// pulled out of TestSelfRecallDecaysWithCorpusGrowth so the write phase and
// the probe phase are each one readable function instead of one long one.
//
// noteSets tracks every attempted note/fact's exact text alongside varSets'
// entity vars -- linkRelatedSessions (linkbuilder_test.go) uses each
// session's first note as the query text for its --semantic link-suggestion
// pass.
func writeSessions(t *testing.T, run func(args ...string) (string, error), picked []*transcriptSession) (varSets [][]string, noteSets [][]string, rejected []string, attempted int) {
	t.Helper()
	varSets = make([][]string, len(picked))
	noteSets = make([][]string, len(picked))
	writeStart := time.Now()

	if os.Getenv("WINZE_NOTE_SHAPE") == "claims" {
		client, ok := newAnthropicClientFromEnv()
		if !ok {
			t.Skip("WINZE_NOTE_SHAPE=claims needs ANTHROPIC_API_KEY (one Haiku call per session, ~700 input + ~150 output tokens each) — skipping")
		}
		var inTok, cacheReadTok, outTok int64
		for i, s := range picked {
			facts, usage, err := extractFacts(client, s)
			inTok += usage.InputTokens
			cacheReadTok += usage.CacheReadInputTokens
			outTok += usage.OutputTokens
			if err != nil {
				t.Errorf("extracting facts for %d (%s): %v", i, s.ID[:8], err)
				continue
			}
			for _, fact := range facts {
				attempted++
				noteSets[i] = append(noteSets[i], fact)
				out, err := run("call", "winze_remember", `{"note":`+mustJSON(fact)+`}`)
				if err != nil {
					t.Errorf("write %d fact %q failed to execute: %v\n%s", i, fact, err, out)
					continue
				}
				if v := createdVar(out); v != "" {
					varSets[i] = append(varSets[i], v)
				} else {
					rejected = append(rejected, fmt.Sprintf("[%d] %s %q — %s",
						i, s.Start.Format("2006-01-02"), fact, dedupReason(out)))
				}
			}
		}
		t.Logf("%d sessions, %d facts extracted and attempted in %s; %d stored, %d rejected before storage",
			len(picked), attempted, time.Since(writeStart).Round(time.Second),
			attempted-len(rejected), len(rejected))
		t.Logf("extraction cost: %d input tokens (%d cache-read), %d output tokens across %d Haiku calls",
			inTok, cacheReadTok, outTok, len(picked))
		for _, r := range rejected {
			t.Logf("  rejected: %s", r)
		}
		return varSets, noteSets, rejected, attempted
	}

	for i, s := range picked {
		attempted++
		note := noteFor(s)
		noteSets[i] = []string{note}
		out, err := run("call", "winze_remember", `{"note":`+mustJSON(note)+`}`)
		if err != nil {
			t.Errorf("write %d (%s) failed to execute: %v\n%s", i, s.ID[:8], err, out)
			continue
		}
		if v := createdVar(out); v != "" {
			varSets[i] = []string{v}
		} else {
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
	return varSets, noteSets, rejected, attempted
}

// bestRankOf returns the best (lowest) 1-based rank among vars in a recall
// result, or 0 if none of them were surfaced. Needed once a session can own
// more than one entity (the "claims" note shape): the question "was this
// session found" becomes "was ANY of its facts found", not "was its one note
// found".
func bestRankOf(hits recallHits, vars []string) int {
	best := 0
	for _, v := range vars {
		if v == "" {
			continue
		}
		if r := rankOf(hits, v); r != 0 && (best == 0 || r < best) {
			best = r
		}
	}
	return best
}

// hitRateWithin reports the fraction of all probes (found and missed) that
// ranked at or within k. k=5 matches recallDefaultLimit (cmd/agent/mcp.go) --
// the number that answers whether a live winze_recall caller, not this
// harness's much larger measurement window, would actually have seen the hit.
func (p probeStats) hitRateWithin(k int) float64 {
	total := p.found + p.miss
	if total == 0 {
		return 0
	}
	within := 0
	for _, r := range p.ranks {
		if r <= k {
			within++
		}
	}
	return float64(within) / float64(total)
}

// medianRank returns the median of every found rank. meanRank alone can't
// tell a symmetric spread from a few very-deep hits dragging the average up
// while most hits already rank low -- exactly the ambiguity a costrel
// consult (2026-09-07) flagged as unresolved before recommending a build.
func (p probeStats) medianRank() float64 {
	if len(p.ranks) == 0 {
		return 0
	}
	sorted := append([]int(nil), p.ranks...)
	sort.Ints(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return float64(sorted[mid-1]+sorted[mid]) / 2
	}
	return float64(sorted[mid])
}

// arcNote adds the session's later asks, minus the one LaterAsk holds out as
// the probe. That holdout is the whole point: dropping every ask into the
// note would make the later probe a title probe in different clothes, and
// the run would report a retrieval win that was really the answer being
// written into the question. Comparing this shape against openNote at the
// same store size separates "the store cannot bridge unseen wording" from
// "the note did not describe the session" -- which the first run could not
// tell apart.
func arcNote(s *transcriptSession) string {
	note := openNote(s)
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

// openNote is title plus the operator's first ask -- the cheapest note that
// could work, and the shape the first 140-session run measured.
func openNote(s *transcriptSession) string {
	ask := s.OpeningAsk()
	if len(ask) > 1200 {
		ask = ask[:1200] + "…"
	}
	return fmt.Sprintf("Session %s (%s): %s\n\nOpened with: %s",
		s.Start.Format("2006-01-02"), s.ID[:8], s.Title, ask)
}

// outcomeNote replaces the opening ask with the assistant's own most recent
// substantial response from strictly before the probe turn (midpointOutcome)
// -- an already-reached conclusion rather than the question that started the
// session. Real winze-memory Briefs measured 2026-09-08 (median 571 chars,
// dense with dates/commits/outcomes) read nothing like a raw opening
// question, so openNote's own shape may be a pessimistic proxy for what a
// real winze_remember call actually writes -- this tests that without
// re-feeding the transcript to a model (same zero-extra-cost principle
// openNote/arcNote already use: the text is already on disk). Returns ""
// when no assistant turn precedes the probe, so callers can fall back to
// openNote rather than emit an empty note.
func outcomeNote(s *transcriptSession) string {
	out := s.midpointOutcome()
	if out == "" {
		return ""
	}
	if len(out) > 1200 {
		out = out[:1200] + "…"
	}
	return fmt.Sprintf("Session %s (%s): %s\n\n%s",
		s.Start.Format("2006-01-02"), s.ID[:8], s.Title, out)
}
