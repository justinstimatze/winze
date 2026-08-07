// Command longmemeval runs winze's machinery against LongMemEval-S: it extracts
// typed personal-memory facts from each question's haystack, stores them as a
// standalone typed Go corpus, indexes and retrieves through defn, answers from
// the retrieval, and judges against gold. It reports accuracy alongside a
// per-hop timing breakdown that separates the winze-via-defn machinery
// (build + sync + retrieve) from the LLM hops (extract + answer + judge).
//
// The memory schema here is deliberately NOT winze's epistemology schema — it's
// a single Fact struct. That's the point: the substrate (build gate as
// consistency check, defn as index) is schema-agnostic, so a new domain is a
// new struct, not a rewrite.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type runner struct {
	client   anthropic.Client
	cacheDir string
	workDir  string
	stats    *usageStats
}

func nowNS() int64 { return time.Now().UnixNano() }

func main() {
	var (
		dataset    = flag.String("dataset", "", "path to longmemeval_s.json (required)")
		cacheDir   = flag.String("cache", "", "extraction cache dir (default: <work>/cache)")
		workDir    = flag.String("work", "", "working dir for generated stores (required)")
		nTemporal  = flag.Int("temporal", 2, "number of temporal-reasoning questions")
		nKnowledge = flag.Int("knowledge", 2, "number of knowledge-update questions")
		nSingle    = flag.Int("single", 1, "number of single-session-user questions")
		nMulti     = flag.Int("multi", 0, "number of multi-session questions")
		nAsst      = flag.Int("assistant", 0, "number of single-session-assistant questions")
		nPref      = flag.Int("preference", 0, "number of single-session-preference questions")
		topK       = flag.Int("k", 60, "retrieval top-k facts fed to the answerer. Was 15, with no recorded rationale; the 2026-08-06 sweep scored 44/45/47 of 60 at k=15/30/60, monotone and concentrated in multi-session (7->8->9), the type with the most facts competing for the window. 27 of 60 questions produce more facts than 15 slots admit, so 15 was binding on nearly half the set. Not swept past 60 — no ceiling was found, so a larger value may still pay. Costs answerer input tokens only; retrieval searches the whole store either way.")
		dryRun     = flag.Bool("dry-run", false, "select subset and report shape only; no API calls")
		probe      = flag.Bool("probe", false, "report whether gold answer turns survive renderSession truncation; no API calls")
		conc       = flag.Int("concurrency", 8, "questions run at once. The loop is ~99% blocked on the API — 12.5s extract + 2.5s answer + 1.1s judge against 0.9s of winze machinery per question — so this is close to a linear speedup until the API rate limit or the per-question `go build` becomes the constraint. 1 restores the old serial behaviour, which is what a concurrency bug should be diffed against.")
		only       = flag.String("only", "", "comma-separated question ids (prefixes ok) to run instead of the per-type quota. For testing a hypothesis about specific failures without paying for the whole subset — a lensVersion bump makes every question cold, so a six-question check costs six extractions rather than sixty.")
		baseline   = flag.String("baseline", "", "write per-question outcomes (qid, gold, answer, verdict) as JSONL to this path, for diffing the next configuration against this one question by question. Omits timings on purpose — they churn every row on every run.")
	)
	flag.Parse()

	if *dataset == "" || *workDir == "" {
		fmt.Fprintln(os.Stderr, "usage: longmemeval --dataset <path> --work <dir> [flags]")
		os.Exit(2)
	}
	cache := *cacheDir
	if cache == "" {
		cache = filepath.Join(*workDir, "cache")
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir cache: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*workDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir work: %v\n", err)
		os.Exit(1)
	}

	quota := map[string]int{
		"temporal-reasoning":        *nTemporal,
		"knowledge-update":          *nKnowledge,
		"single-session-user":       *nSingle,
		"multi-session":             *nMulti,
		"single-session-assistant":  *nAsst,
		"single-session-preference": *nPref,
	}
	var questions []Question
	var err error
	if *only != "" {
		var ids []string
		for _, id := range strings.Split(*only, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
		questions, err = selectByID(*dataset, ids)
	} else {
		questions, err = selectSubset(*dataset, quota)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "select subset: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("selected %d questions: %s\n", len(questions), quotaSummary(questions))

	if *dryRun {
		for _, q := range questions {
			fmt.Printf("  %s [%s] %d sessions — %q\n", q.QuestionID, q.QuestionType, len(q.HaystackSessions), q.Question)
		}
		return
	}

	if *probe {
		probeTruncation(questions)
		return
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = loadEnvKey("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY required (env or ./.env)")
		os.Exit(1)
	}

	r := &runner{
		client:   anthropic.NewClient(option.WithAPIKey(apiKey)),
		cacheDir: cache,
		workDir:  *workDir,
		stats:    &usageStats{perModel: map[string]*modelUsage{}},
	}

	rows, errored := runAll(r, questions, *topK, *conc)

	report(rows, errored, r.stats)

	if *baseline != "" {
		if err := writeBaseline(*baseline, questions, rows); err != nil {
			fmt.Fprintf(os.Stderr, "write baseline: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nbaseline written: %s (%d rows)\n", *baseline, len(rows))
	}
}

// runQuestion runs the full loop for one question, timing each hop.
func (r *runner) runQuestion(q Question, k int) (resultRow, error) {
	row := resultRow{qid: q.QuestionID, qtype: q.QuestionType, sessions: len(q.HaystackSessions)}

	// Map session id -> date for attaching temporal context to facts.
	dateOf := map[string]string{}
	for i, sid := range q.HaystackSessionIDs {
		if i < len(q.HaystackDates) {
			dateOf[sid] = q.HaystackDates[i]
		}
	}

	// Extract (per session, cached).
	tExtract := nowNS()
	var facts []Fact
	for i, sess := range q.HaystackSessions {
		sid := ""
		if i < len(q.HaystackSessionIDs) {
			sid = q.HaystackSessionIDs[i]
		}
		ex, err := r.extractSession(sid, sess)
		if err != nil {
			return row, err
		}
		if ex.Truncated {
			row.truncated++
		}
		if ex.Retried {
			row.retried++
		}
		for _, f := range ex.Facts {
			facts = append(facts, Fact{
				Attribute: f.Attribute, Value: f.Value, Kind: f.Kind,
				Date: dateOf[sid], Session: sid, Quote: f.Quote,
			})
		}
	}
	row.extractNS = nowNS() - tExtract
	row.facts = len(facts)

	// Build the typed store (build gate as consistency check).
	tBuild := nowNS()
	dir, err := r.buildStore(q.QuestionID, facts)
	if err != nil {
		return row, err
	}
	row.buildNS = nowNS() - tBuild

	// Sync through defn + retrieve.
	retrieved, syncNS, retrieveNS, err := r.syncAndRetrieve(dir, q.Question, k)
	if err != nil {
		return row, err
	}
	row.syncNS, row.retrieveNS, row.retrieved = syncNS, retrieveNS, len(retrieved)

	// Answer.
	tAns := nowNS()
	ans, err := r.answer(q.Question, q.QuestionDate, retrieved)
	if err != nil {
		return row, err
	}
	row.answerNS = nowNS() - tAns
	row.answer = ans

	// Judge.
	tJudge := nowNS()
	correct, err := r.judge(q.Question, string(q.Answer), ans)
	if err != nil {
		return row, err
	}
	row.judgeNS = nowNS() - tJudge
	row.correct = correct
	return row, nil
}

func quotaSummary(qs []Question) string {
	counts := map[string]int{}
	for _, q := range qs {
		counts[q.QuestionType]++
	}
	var parts []string
	for t, n := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", n, t))
	}
	return strings.Join(parts, ", ")
}

// loadEnvKey reads a single key from ./.env (KEY=VALUE), for local runs.
func loadEnvKey(key string) string {
	f, err := os.Open(".env")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// selectByID streams the dataset and returns exactly the named questions, in
// dataset order, ignoring the per-type quota entirely.
//
// This exists so a hypothesis about six failing questions can be tested by
// running six questions. The quota path cannot express that: it selects the
// first N of each type in file order, so reaching one specific failure means
// paying for every question ahead of it. A lensVersion bump then makes the
// whole thing cold. Two of today's measurements spent a full sixty-question
// re-extraction to learn something four questions would have shown.
//
// IDs may be given as prefixes, matching how the report and baselines print
// them.
func selectByID(path string, ids []string) ([]Question, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("read opening token: %w", err)
	}

	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []Question
	seen := map[string]bool{}
	for dec.More() && len(out) < len(ids) {
		var q Question
		if err := dec.Decode(&q); err != nil {
			return nil, fmt.Errorf("decode record: %w", err)
		}
		for id := range want {
			if seen[id] || !strings.HasPrefix(q.QuestionID, id) {
				continue
			}
			seen[id] = true
			out = append(out, q)
			break
		}
	}
	// A typo in an id is silent otherwise — the run just comes up short and
	// looks like a smaller subset was intended.
	var missing []string
	for _, id := range ids {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("--only: no question matched %s", strings.Join(missing, ", "))
	}
	return out, nil
}
