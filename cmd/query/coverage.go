package main

// Documentation-coverage gate: every binary under cmd/ must be named in a doc.
//
// The rot the docs-recall split is meant to prevent isn't a stale sentence —
// it's a tool nobody wrote a sentence about. Absence is the one defect no drift
// checker sees: a linter, an LLM reading the diff, and a tree-sitter anchor all
// need something present to flag, and an undocumented binary is nothing to look
// at. This is the set difference the other lanes can't compute: what the code
// exposes, minus what the docs mention.
//
// Scoped to binaries deliberately. Their names (winze-meld, cmd/observatory)
// are distinctive, so a substring test has no false positives. MCP tool names
// (claims, search, stats) collide with common prose words, so an automated
// mention test on them would be noise — left out rather than shipped flaky.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runDocsCoverage enumerates cmd/* binaries and checks each is named in a doc
// (as winze-<name> or cmd/<name>). Prints the undocumented ones and exits 1 if
// any — the gate — else prints a clean line and exits 0.
func runDocsCoverage(dir string) {
	bins, err := listBinaries(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs-coverage: %v\n", err)
		os.Exit(2)
	}
	docText, err := gatherDocText(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs-coverage: %v\n", err)
		os.Exit(2)
	}

	var undocumented []string
	for _, b := range bins {
		if !strings.Contains(docText, "winze-"+b) && !strings.Contains(docText, "cmd/"+b) {
			undocumented = append(undocumented, b)
		}
	}

	if len(undocumented) == 0 {
		fmt.Printf("docs-coverage: all %d binaries documented\n", len(bins))
		return
	}
	fmt.Printf("docs-coverage: %d of %d binaries have no doc mention:\n", len(undocumented), len(bins))
	for _, b := range undocumented {
		fmt.Printf("  cmd/%s — add a `winze-%s` or `cmd/%s` mention to a doc\n", b, b, b)
	}
	os.Exit(1)
}

// listBinaries returns the base name of every cmd/* subdirectory (each is a
// main package that builds to bin/winze-<name>).
func listBinaries(dir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "cmd"))
	if err != nil {
		return nil, fmt.Errorf("read cmd/: %w", err)
	}
	var bins []string
	for _, e := range entries {
		if e.IsDir() {
			bins = append(bins, e.Name())
		}
	}
	sort.Strings(bins)
	return bins, nil
}

// gatherDocText concatenates the project's documentation: top-level *.md and
// everything under docs/. CLAUDE.md is included here (unlike docs-recall, which
// excludes it) — a binary mentioned only in the resident core still counts as
// documented.
func gatherDocText(dir string) (string, error) {
	var b strings.Builder
	add := func(path string) {
		if data, err := os.ReadFile(path); err == nil {
			b.Write(data)
			b.WriteByte('\n')
		}
	}
	// Top-level markdown.
	top, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range top {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			add(filepath.Join(dir, e.Name()))
		}
	}
	// docs/ tree.
	_ = filepath.WalkDir(filepath.Join(dir, "docs"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
			add(path)
		}
		return nil
	})
	return b.String(), nil
}
