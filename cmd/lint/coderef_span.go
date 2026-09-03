package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// codeRefSpanRule is milestone 1 of CodeRef.Client/.Span (corpus/schema.go,
// docs/typed-citation.md, FEEDBACK-2026-09-02.md#1): a content-hash check for
// the only mechanism available to a non-Go citation, since Symbol has no
// compile-time reach outside the store's own module and none into non-Go
// source at all.
func codeRefSpanRule(dir string, clients map[string]string) int {
	sites, err := collectCodeRefs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[coderef-span] error: %v\n", err)
		return 2
	}
	var spanSites []codeRefSite
	for _, s := range sites {
		if s.ref.Span != nil {
			spanSites = append(spanSites, s)
		}
	}
	if len(spanSites) == 0 {
		fmt.Println("[coderef-span] no Span-based CodeRefs in this corpus")
		return 0
	}
	// Client == "" span refs (a content-hash citation to a file in the store's
	// OWN checkout) never needed --clients in the first place, so they're
	// always checked. Only Client != "" refs are gated by --clients, mirroring
	// the llm.go "skip entirely, cost nothing" precedent for the external case
	// this feature is built for.
	needsClients := false
	for _, s := range spanSites {
		if s.ref.Client != "" {
			needsClients = true
		}
	}
	if needsClients && len(clients) == 0 {
		fmt.Printf("[coderef-span] skipped: %d Span-based CodeRef(s) need --clients, none configured\n", len(spanSites))
		return 0
	}

	// An in-store citation's Path is relative to the store's own root, the
	// same convention corpus.CodeRef.Path already uses for a same-module Go
	// symbol ("internal/corpuslock.Acquire") — not the process's cwd, which
	// only accidentally matches when the tool happens to run from the repo
	// root. dir is the corpus subdirectory (see cmd/agent/init.go's
	// goDirective for the same filepath.Dir(corpusDir) convention).
	storeRoot := filepath.Dir(dir)

	var bad []string
	checked := 0
	for _, s := range spanSites {
		root := storeRoot
		if s.ref.Client != "" {
			p, ok := clients[s.ref.Client]
			if !ok {
				bad = append(bad, fmt.Sprintf("%s: Client %q has no configured path (--clients)", s.entity, s.ref.Client))
				continue
			}
			root = p
		}
		line, err := lineAt(filepath.Join(root, s.ref.Path), s.ref.Span.Line)
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s: %s:%d: %v", s.entity, s.ref.Path, s.ref.Span.Line, err))
			continue
		}
		checked++
		if hashLine(line) != s.ref.Span.Hash {
			bad = append(bad, fmt.Sprintf("%s: %s:%d content changed (hash mismatch)", s.entity, s.ref.Path, s.ref.Span.Line))
		}
	}

	fmt.Printf("[coderef-span] %d Span-based CodeRef(s), %d checked, %d stale/unresolved\n", len(spanSites), checked, len(bad))
	if len(bad) == 0 {
		return 0
	}
	for _, b := range bad {
		fmt.Println("   ", b)
	}
	return 1
}

func hashLine(line string) string {
	sum := sha256.Sum256([]byte(line))
	return hex.EncodeToString(sum[:])
}

// lineAt returns the exact text of 1-indexed line n in path (no trailing
// newline, matching bufio.Scanner's ScanLines split — the same semantics an
// author must reproduce by hand when computing Span.Hash; see docs/lint-rules.md).
// A genuine scan error (e.g. a line exceeding bufio.Scanner's token limit) is
// reported distinctly from "line past EOF", rather than folding both into one
// message.
func lineAt(path string, n int) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("line %d out of range", n)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	i := 0
	for scanner.Scan() {
		i++
		if i == n {
			return scanner.Text(), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanning %s: %w", path, err)
	}
	return "", fmt.Errorf("line %d out of range (file has %d lines)", n, i)
}
