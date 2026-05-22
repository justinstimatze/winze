// Command predicates-suggest reads .metabolism-trip-isolated.jsonl —
// the log of trip-cycle generations whose rationale explicitly states no
// existing predicate captures the connection — and asks an LLM to cluster
// them by predicate-shape. For each cluster of >= --min-cluster entries,
// it proposes a candidate predicate (name, slot types, sample claims,
// rationale). The human reviewer decides yes/no — the tool preserves the
// project-wide "do not invent predicates speculatively" discipline by
// surfacing candidates rather than auto-promoting them.
//
// Phase 2 of wi-78v1. The trip-isolated log is the corpus's own
// predicate-ontology gap signal: generations the metabolism wanted to
// encode but couldn't. Until cmd/predicates-suggest existed, the log was
// write-only and the gap signal was invisible.
//
// Usage:
//
//	go run ./cmd/predicates-suggest .
//	go run ./cmd/predicates-suggest --min-cluster 4 --model sonnet .
//	go run ./cmd/predicates-suggest --dry-run .                       # print prompt, no API
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justinstimatze/winze/internal/dotenv"
)

func filterValidCandidates(in []predicateCandidate, minCluster int) []predicateCandidate {
	out := make([]predicateCandidate, 0, len(in))
	for _, c := range in {
		if len(c.SampleEntries) < minCluster {
			continue
		}
		if looksLikeSkipMarker(c) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func looksLikeSkipMarker(c predicateCandidate) bool {
	if strings.EqualFold(strings.TrimSpace(c.Name), "SKIP") {
		return true
	}
	lr := strings.ToLower(c.Rationale)
	if strings.Contains(lr, "skip — absorbed") || strings.Contains(lr, "skip - absorbed") || strings.Contains(lr, "skip, absorbed") {
		return true
	}
	for _, s := range c.SampleClaims {
		if strings.EqualFold(strings.TrimSpace(s), "SKIP") {
			return true
		}
	}
	return false
}

// tripIsolated mirrors the JSONL row shape written by cmd/metabolism's
// --trip path when a generated connection has no matching predicate.
type tripIsolated struct {
	Timestamp   string `json:"timestamp"`
	EntityA     string `json:"entity_a"`
	EntityB     string `json:"entity_b"`
	ClusterA    int    `json:"cluster_a"`
	ClusterB    int    `json:"cluster_b"`
	Connection  string `json:"connection"`
	Rationale   string `json:"rationale"`
	Score       int    `json:"score"`
	PromptType  string `json:"prompt_type"`
	Temperature float64 `json:"temperature"`
}

// candidateOutput is the JSON shape written alongside stderr output, so
// the proposals are durable for review and can be diffed across runs.
type candidateOutput struct {
	Timestamp           string             `json:"timestamp"`
	MinClusterSize      int                `json:"min_cluster_size"`
	MinScore            int                `json:"min_score"`
	Model               string             `json:"model"`
	SourceEntryCount    int                `json:"source_entry_count"`
	FilteredEntryCount  int                `json:"filtered_entry_count"`
	ExistingPredicates  []string           `json:"existing_predicates"`
	Candidates          []predicateCandidate `json:"candidates"`
}

func main() {
	var (
		dir          = flag.String("dir", ".", "corpus root (directory containing .metabolism-trip-isolated.jsonl)")
		minCluster   = flag.Int("min-cluster", 3, "minimum entries to constitute a promotable cluster")
		minScore     = flag.Int("min-score", 3, "minimum trip score to include (skip score=2 noise)")
		model        = flag.String("model", "sonnet", "anthropic model tier: haiku | sonnet")
		logName      = flag.String("log", ".metabolism-trip-isolated.jsonl", "trip-isolated JSONL filename (relative to --dir)")
		out          = flag.String("out", ".metabolism-predicate-candidates.json", "candidates JSON output (relative to --dir)")
		dryRun       = flag.Bool("dry-run", false, "print the prompt; do not call API or write output")
	)
	flag.Parse()

	if *minCluster < 2 {
		fmt.Fprintln(os.Stderr, "error: --min-cluster must be >= 2")
		os.Exit(2)
	}
	if *model != "haiku" && *model != "sonnet" {
		fmt.Fprintln(os.Stderr, "error: --model must be 'haiku' or 'sonnet'")
		os.Exit(2)
	}

	logPath := filepath.Join(*dir, *logName)
	entries, err := readTripIsolated(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", logPath, err)
		os.Exit(1)
	}
	sourceCount := len(entries)

	filtered := filterByScore(entries, *minScore)
	if len(filtered) == 0 {
		fmt.Fprintf(os.Stderr, "no entries with score >= %d in %s (source had %d total)\n", *minScore, logPath, sourceCount)
		os.Exit(0)
	}

	existing, err := loadExistingPredicates(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not enumerate existing predicates: %v\n", err)
	}

	if *dryRun {
		fmt.Fprintf(os.Stderr, "--- dry-run: %d entries (filtered from %d), %d existing predicates ---\n",
			len(filtered), sourceCount, len(existing))
		fmt.Println(buildPrompt(filtered, existing, *minCluster))
		return
	}

	dotenv.Load(*dir)
	dotenv.Load(".")

	candidates, err := proposeCandidates(filtered, existing, *minCluster, *model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "predicates-suggest API call failed: %v\n", err)
		os.Exit(1)
	}

	// Forced tool_use occasionally produces placeholder "SKIP" candidates
	// when the model wants to return empty but feels obliged to fill the
	// schema. Drop candidates whose sample size is below threshold or whose
	// content reads as a SKIP marker; the next run's tightened prompt will
	// shrink this set further. Filtering here keeps the JSON output and
	// stderr aligned.
	candidates = filterValidCandidates(candidates, *minCluster)

	output := candidateOutput{
		Timestamp:          nowRFC3339(),
		MinClusterSize:     *minCluster,
		MinScore:           *minScore,
		Model:              *model,
		SourceEntryCount:   sourceCount,
		FilteredEntryCount: len(filtered),
		ExistingPredicates: existing,
		Candidates:         candidates,
	}

	outPath := filepath.Join(*dir, *out)
	if err := writeJSON(outPath, output); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", outPath, err)
	}

	printCandidates(candidates, len(filtered))
}

func readTripIsolated(path string) ([]tripIsolated, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	var out []tripIsolated
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var t tripIsolated
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			continue // skip malformed; trip-isolated may pre-date current schema
		}
		out = append(out, t)
	}
	return out, scanner.Err()
}

func filterByScore(entries []tripIsolated, minScore int) []tripIsolated {
	out := entries[:0]
	for _, e := range entries {
		if e.Score >= minScore {
			out = append(out, e)
		}
	}
	return out
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printCandidates(candidates []predicateCandidate, filteredN int) {
	fmt.Fprintf(os.Stderr, "predicates-suggest: %d candidate predicate(s) proposed from %d trip-isolated entries\n",
		len(candidates), filteredN)
	if len(candidates) == 0 {
		fmt.Fprintln(os.Stderr, "  (no clusters reached min-cluster threshold; gap may be absorbed by existing predicates or noise-shaped)")
		return
	}
	for i, c := range candidates {
		fmt.Fprintf(os.Stderr, "\n[%d] %s  (slots: %s -> %s; cluster size: %d)\n",
			i+1, c.Name, c.SubjectSlot, c.ObjectSlot, len(c.SampleEntries))
		fmt.Fprintf(os.Stderr, "    rationale: %s\n", c.Rationale)
		fmt.Fprintln(os.Stderr, "    sample claims it would encode:")
		for j, s := range c.SampleClaims {
			if j >= 3 {
				fmt.Fprintf(os.Stderr, "      (... %d more)\n", len(c.SampleClaims)-3)
				break
			}
			fmt.Fprintf(os.Stderr, "      %s\n", s)
		}
	}
	fmt.Fprintln(os.Stderr, "\nHuman review required before promotion. Reject candidates that:")
	fmt.Fprintln(os.Stderr, "  - duplicate an existing predicate the LLM missed")
	fmt.Fprintln(os.Stderr, "  - widen scope past what the sample entries justify")
	fmt.Fprintln(os.Stderr, "  - collapse genuinely distinct relations into one slot shape")
}
