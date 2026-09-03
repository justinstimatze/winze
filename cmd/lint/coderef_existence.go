package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

// codeRefExistenceRule is milestone 2 of CodeRef.Client (see coderef.go,
// corpus/schema.go): a Go-to-Go existence check for a Client-only citation
// (Client != "" && Span == nil) via golang.org/x/tools/go/packages —
// deliberately not hash-based; go/packages correctly ignores harmless
// reformatting/comment drift a hash would false-positive on.
//
// Known, deliberate limitation: Types.Scope().Names() only sees package-level
// declarations, not methods ((*Foo).Bar) — fine for the motivating case (a
// plain function), a real gap if a method citation shows up later; not
// solved speculatively.
func codeRefExistenceRule(dir string, clients map[string]string) int {
	sites, err := collectCodeRefs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[coderef-existence] error: %v\n", err)
		return 2
	}
	type target struct{ entity, path string }
	byClient := map[string][]target{}
	for _, s := range sites {
		if s.ref.Client == "" || s.ref.Span != nil {
			continue
		}
		byClient[s.ref.Client] = append(byClient[s.ref.Client], target{s.entity, s.ref.Path})
	}
	if len(byClient) == 0 {
		fmt.Println("[coderef-existence] no Client-only CodeRefs to check")
		return 0
	}
	if len(clients) == 0 {
		fmt.Println("[coderef-existence] skipped (use --clients to enable)")
		return 0
	}

	var bad []string
	checked := 0
	for client, targets := range byClient {
		repoPath, ok := clients[client]
		if !ok {
			for _, tg := range targets {
				bad = append(bad, fmt.Sprintf("%s: Client %q has no configured path (--clients)", tg.entity, client))
			}
			continue
		}
		cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes, Dir: repoPath}
		pkgs, err := packages.Load(cfg, "./...")
		if err != nil {
			for _, tg := range targets {
				bad = append(bad, fmt.Sprintf("%s: loading client %q at %s: %v", tg.entity, client, repoPath, err))
			}
			continue
		}
		exists := map[string]bool{}
		for _, p := range pkgs {
			if p.Types == nil {
				continue
			}
			scope := p.Types.Scope()
			for _, name := range scope.Names() {
				exists[p.PkgPath+"."+name] = true
			}
		}
		for _, tg := range targets {
			checked++
			if !exists[tg.path] {
				bad = append(bad, fmt.Sprintf("%s: %q not found in client %q at %s", tg.entity, tg.path, client, repoPath))
			}
		}
	}

	fmt.Printf("[coderef-existence] %d Client-only CodeRef(s) checked, %d unresolved\n", checked, len(bad))
	if len(bad) == 0 {
		return 0
	}
	for _, b := range bad {
		fmt.Println("   ", b)
	}
	return 1
}
