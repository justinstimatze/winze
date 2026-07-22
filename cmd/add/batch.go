package main

// Batch mode appends many claims under a single build gate. The gate
// (go build . && go vet .) is ~91ms warm and dominates the per-claim cost;
// gofmt-ing one file is ~2.5ms and a raw append is sub-millisecond. Running
// the gate once for K claims turns K*(2.5+91)ms into K*2.5 + 91ms — five
// claims drop from ~475ms to ~104ms. This is the burst-write path the
// per-session shared-KB shape needs: a session that learned several things
// in one turn commits them together, not one expensive gate at a time.
//
// Atomicity matches the single-claim path: every file the batch touches is
// backed up before the first write, and ANY failure (parse, append, gofmt,
// build, vet) reverts every touched file. A batch is all-or-nothing — a
// partial append that failed the gate would leave the corpus in a state that
// does not build, which is exactly what the gate exists to prevent.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// claimSpec is one claim in a batch file. Fields mirror the single-claim
// flags so a batch record and a command line describe the same thing. JSONL
// (one object per line) is the corpus's native log shape (.metabolism-*.jsonl)
// and streams without loading the whole batch into a nested document.
type claimSpec struct {
	To         string `json:"to"`
	Name       string `json:"name"`
	Predicate  string `json:"predicate"`
	Subject    string `json:"subject"`
	Object     string `json:"object"`
	Quote      string `json:"quote"`
	Origin     string `json:"origin"`
	IngestedBy string `json:"ingested_by"`
	ProvVar    string `json:"provenance_var"`
	Unary      bool   `json:"unary"`
}

func (s claimSpec) ingestedByOrDefault() string {
	if s.IngestedBy == "" {
		return "winze-add"
	}
	return s.IngestedBy
}

// readBatch parses JSONL from a path, or from stdin when path is "-". Blank
// lines are skipped so a batch file can be visually grouped.
func readBatch(path string) ([]claimSpec, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}

	var specs []claimSpec
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // claims can carry long quotes
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var spec claimSpec
		if err := json.Unmarshal([]byte(text), &spec); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		specs = append(specs, spec)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return specs, nil
}

// runBatch appends every claim in the batch under one build gate, reverting
// all touched files if any step fails. Returns a process exit code.
func runBatch(batchPath, repoRoot string, dryRun bool) int {
	specs, err := readBatch(batchPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read batch: %v\n", err)
		return 2
	}
	if len(specs) == 0 {
		fmt.Fprintln(os.Stderr, "error: batch is empty")
		return 2
	}

	// Validate every record before touching a single file — a bad record in
	// the batch should fail the whole thing before any state changes.
	for i, s := range specs {
		if err := validateFlags(s.Predicate, s.Subject, s.Object, s.Quote, s.Origin, s.ProvVar, s.To, s.Name, s.Unary); err != nil {
			fmt.Fprintf(os.Stderr, "batch record %d (%q): %v\n", i+1, s.Name, err)
			return 2
		}
	}

	if dryRun {
		for _, s := range specs {
			fmt.Printf("--- would append to %s ---\n", s.To)
			fmt.Println(renderClaim(s.Predicate, s.Subject, s.Object, s.Quote, s.Origin, s.ingestedByOrDefault(), s.ProvVar, s.Name, s.Unary))
		}
		return 0
	}

	// Back up every unique target before the first write; touched preserves
	// insertion order so gofmt gets a stable, minimal file list.
	backups := map[string][]byte{}
	var touched []string
	for _, s := range specs {
		p := filepath.Join(repoRoot, s.To)
		if _, ok := backups[p]; ok {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			revertAll(backups)
			fmt.Fprintf(os.Stderr, "read %s: %v (reverted)\n", p, err)
			return 1
		}
		backups[p] = b
		touched = append(touched, p)
	}

	for _, s := range specs {
		p := filepath.Join(repoRoot, s.To)
		decl := renderClaim(s.Predicate, s.Subject, s.Object, s.Quote, s.Origin, s.ingestedByOrDefault(), s.ProvVar, s.Name, s.Unary)
		if err := appendDecl(p, decl); err != nil {
			revertAll(backups)
			fmt.Fprintf(os.Stderr, "append to %s failed (reverted): %v\n", p, err)
			return 1
		}
	}

	// Format only the files this batch touched. gofmt -w on the repo root
	// would recurse and sweep unrelated drift into the change.
	gofmtArgs := append([]string{"-w"}, touched...)
	steps := [][]string{
		append([]string{"gofmt"}, gofmtArgs...),
		{"go", "build", "."},
		{"go", "vet", "."},
	}
	for _, step := range steps {
		if out, err := runCmd(repoRoot, step[0], step[1:]...); err != nil {
			revertAll(backups)
			fmt.Fprintf(os.Stderr, "%s failed (all %d files reverted):\n%s\n", step[0], len(touched), out)
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "added %d claims across %d files (one build gate)\n", len(specs), len(touched))
	return 0
}

func revertAll(backups map[string][]byte) {
	for path, content := range backups {
		_ = os.WriteFile(path, content, 0o644)
	}
}
