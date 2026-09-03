package main

import (
	"fmt"
	"os"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// codeRefSite is one CodeRef literal plus the entity var name carrying it —
// built once from parseCorpusCached's shared entity parse, not a separate
// AST walk, and shared by every coderef-* rule.
type codeRefSite struct {
	entity string
	ref    corpusparse.CodeRef
}

// codeRefMutualExclusionRule flags a CodeRef literal that sets both a Symbol
// and a Client — see corpus/schema.go's CodeRef doc for why that's a
// structural contradiction, not a style preference: a citation is either
// this store's own module (compiler-checked) or an external client
// (lint-checked), never ambiguously either. Always on: zero cost (no client
// resolution, no file I/O), so it runs unconditionally like naming-oracle.
func codeRefMutualExclusionRule(dir string) int {
	sites, err := collectCodeRefs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[coderef-exclusive] error: %v\n", err)
		return 2
	}
	var bad []string
	for _, s := range sites {
		if s.ref.HasSymbol && s.ref.Client != "" {
			bad = append(bad, fmt.Sprintf("%s: %q sets both Symbol and Client=%q", s.entity, s.ref.Path, s.ref.Client))
		}
	}
	fmt.Printf("[coderef-exclusive] %d CodeRef(s) scanned, %d violating Symbol/Client mutual exclusion\n", len(sites), len(bad))
	if len(bad) == 0 {
		return 0
	}
	for _, b := range bad {
		fmt.Println("   ", b)
	}
	fmt.Println("  a CodeRef is either a same-module citation (Symbol) or an external-client citation (Client), never both.")
	return 1
}

func collectCodeRefs(dir string) ([]codeRefSite, error) {
	entities, _, err := parseCorpusCached(dir)
	if err != nil {
		return nil, err
	}
	var out []codeRefSite
	for _, e := range entities {
		for _, r := range e.Refs {
			out = append(out, codeRefSite{entity: e.VarName, ref: r})
		}
	}
	return out, nil
}
