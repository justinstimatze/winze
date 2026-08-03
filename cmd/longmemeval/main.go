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
		topK       = flag.Int("k", 15, "retrieval top-k facts fed to the answerer")
		dryRun     = flag.Bool("dry-run", false, "select subset and report shape only; no API calls")
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
	questions, err := selectSubset(*dataset, quota)
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

	var rows []resultRow
	for i, q := range questions {
		fmt.Printf("\n[%d/%d] %s [%s]\n  Q: %s\n", i+1, len(questions), q.QuestionID, q.QuestionType, q.Question)
		row, err := r.runQuestion(q, *topK)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
			continue
		}
		mark := "✗"
		if row.correct {
			mark = "✓"
		}
		fmt.Printf("  gold: %q\n  ans:  %q  %s\n", q.Answer, row.answer, mark)
		rows = append(rows, row)
	}

	report(rows, r.stats)
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
		for _, f := range ex {
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
