// Command rot-probe samples a random subset of corpus entities and asks an
// LLM to flag potential rot signals — near-duplicates, contradictions, or
// brief drift. Findings are surfaced for human review only; the tool never
// auto-fixes. Output is appended to .metabolism-rot-probe.jsonl for
// time-series, and pretty-printed to stderr.
//
// Phase 3 of wi-vr66. The whole point of this tool is to make winze's
// typed-substrate value visible: until rot surfacing actually surfaces
// things, the "is the typed gate worth its friction" question lives on
// faith. Periodic rot-probe runs convert that question into evidence.
//
// Usage:
//
//	go run ./cmd/rot-probe .                          # 10 entities, Haiku, default log
//	go run ./cmd/rot-probe --n 20 --model sonnet .    # bigger sample, deeper model
//	go run ./cmd/rot-probe --dry-run --seed 42 .      # deterministic preview, no API call
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// rotProbeRun is one full invocation written as a single JSONL row.
type rotProbeRun struct {
	Timestamp  string    `json:"timestamp"`
	SampleSize int       `json:"sample_size"`
	Seed       uint64    `json:"seed"`
	Model      string    `json:"model"`
	Sampled    []string  `json:"sampled_entities"` // var names
	Findings   []finding `json:"findings"`
}

func main() {
	var (
		dir     = flag.String("dir", ".", "corpus root (directory containing predicates.go)")
		n       = flag.Int("n", 10, "number of entities to sample")
		model   = flag.String("model", "haiku", "anthropic model tier: haiku | sonnet")
		out     = flag.String("out", ".metabolism-rot-probe.jsonl", "JSONL output path (relative to --dir)")
		seedHex = flag.Int64("seed", 0, "PRNG seed (0 = time-based)")
		dryRun  = flag.Bool("dry-run", false, "print the prompt and sampled entities; do not call API or write log")
	)
	flag.Parse()

	if *n < 1 {
		fmt.Fprintln(os.Stderr, "error: --n must be >= 1")
		os.Exit(2)
	}
	if *model != "haiku" && *model != "sonnet" {
		fmt.Fprintln(os.Stderr, "error: --model must be 'haiku' or 'sonnet'")
		os.Exit(2)
	}

	entities, claims, err := parseCorpus(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse corpus: %v\n", err)
		os.Exit(1)
	}
	if len(entities) == 0 {
		fmt.Fprintf(os.Stderr, "no entities found in %s\n", *dir)
		os.Exit(1)
	}

	hoods := buildNeighborhoods(entities, claims)

	// Only sample entities that have at least one claim connection — naked
	// entities with no edges have nothing for the rot-probe to reason about.
	hoods = filterConnected(hoods)
	if len(hoods) == 0 {
		fmt.Fprintf(os.Stderr, "no connected entities in %s\n", *dir)
		os.Exit(1)
	}

	seed := uint64(*seedHex)
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	samples := sample(hoods, *n, seed)

	sampledNames := make([]string, 0, len(samples))
	for _, s := range samples {
		sampledNames = append(sampledNames, s.ent.varName)
	}

	if *dryRun {
		fmt.Fprintf(os.Stderr, "--- dry-run: %d entities sampled (seed=%d) ---\n", len(samples), seed)
		for _, name := range sampledNames {
			fmt.Fprintln(os.Stderr, "  -", name)
		}
		fmt.Println(buildPrompt(samples))
		return
	}

	loadDotEnv(*dir)
	loadDotEnv(".")

	findings, err := runProbe(samples, *model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rot-probe API call failed: %v\n", err)
		os.Exit(1)
	}

	run := rotProbeRun{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		SampleSize: len(samples),
		Seed:       seed,
		Model:      *model,
		Sampled:    sampledNames,
		Findings:   findings,
	}

	logPath := filepath.Join(*dir, *out)
	if err := appendJSONL(logPath, run); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to append to %s: %v\n", logPath, err)
	}

	printFindings(samples, findings)
}

func filterConnected(hoods []neighborhood) []neighborhood {
	out := hoods[:0]
	for _, h := range hoods {
		if len(h.asSubj) > 0 || len(h.asObj) > 0 {
			out = append(out, h)
		}
	}
	return out
}

func sample(hoods []neighborhood, n int, seed uint64) []neighborhood {
	if n >= len(hoods) {
		return hoods
	}
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	idx := r.Perm(len(hoods))[:n]
	sort.Ints(idx)
	out := make([]neighborhood, 0, n)
	for _, i := range idx {
		out = append(out, hoods[i])
	}
	return out
}

func buildNeighborhoods(entities []entity, claims []claim) []neighborhood {
	subjMap := map[string][]claim{}
	objMap := map[string][]claim{}
	for _, c := range claims {
		subjMap[c.subjectVar] = append(subjMap[c.subjectVar], c)
		if c.objectVar != "" {
			objMap[c.objectVar] = append(objMap[c.objectVar], c)
		}
	}
	out := make([]neighborhood, 0, len(entities))
	for _, e := range entities {
		out = append(out, neighborhood{
			ent:    e,
			asSubj: subjMap[e.varName],
			asObj:  objMap[e.varName],
		})
	}
	return out
}

func appendJSONL(path string, v any) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	enc := json.NewEncoder(f)
	return enc.Encode(v)
}

func printFindings(samples []neighborhood, findings []finding) {
	fmt.Fprintf(os.Stderr, "rot-probe: sampled %d entities, %d finding(s)\n", len(samples), len(findings))
	if len(findings) == 0 {
		fmt.Fprintln(os.Stderr, "  (sample looks clean)")
		return
	}
	for i, f := range findings {
		fmt.Fprintf(os.Stderr, "\n[%d] %s (%s confidence)\n", i+1, f.Kind, f.Confidence)
		fmt.Fprintf(os.Stderr, "    entities: %v\n", f.Entities)
		fmt.Fprintf(os.Stderr, "    %s\n", f.Rationale)
	}
}
