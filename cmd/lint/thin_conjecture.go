package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
)

// thinConjectureRule flags a Conjecture literal whose Rationale is the empty
// string (including the key omitted entirely, which is the same zero value).
// Conjecture is winze's own generated-claim backing (corpus/schema.go) —
// deliberately uncitable (no Quote field, closed by the compiler) but its
// honesty depends on Rationale actually carrying winze's reasoning for the
// connection. An empty Rationale is generated knowledge with no reasoning
// recorded at all, and nothing else in the toolchain catches it: Go accepts
// the zero value for a string field, and none of the other ten lint rules
// inspect this type.
//
// Conjecture literals are authored NESTED inside the claim they attribute —
// `TheoryOf{..., Prov: Conjecture{...}}` — never as a standalone top-level
// var (cmd/add's --conjecture mode inlines it the same way --quote/--origin
// inline a Provenance; only --provenance-var has a reuse mode, and Conjecture
// has no equivalent). So this walks every expression under each var decl
// looking for a Conjecture literal at any depth, rather than checking only
// the var's own top-level type the way lexiconFenceRule does for Provenance —
// a shallower walk here found zero hits against the real corpus despite a
// dozen-plus trip-promoted files that plainly have them (metabolism_cycle*.go).
//
// Deterministic and narrow on purpose: only the empty string is flagged. A
// short-but-present Rationale ("TBD") is a real quality question too, but
// there is no forcing-function instance of it yet to say what "thin" means in
// practice — per this project's own schema-accretion discipline, this rule
// waits for one rather than inventing a threshold.
func thinConjectureRule(dir string) int {
	fset, files, err := corpusFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[thin-conjecture] error: %v\n", err)
		return 2
	}

	scanned := 0
	var hits []thinConjectureHit
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Values) == 0 || len(vs.Names) == 0 {
					continue
				}
				varName := vs.Names[0].Name
				ast.Inspect(vs.Values[0], func(n ast.Node) bool {
					cl, ok := n.(*ast.CompositeLit)
					if !ok {
						return true
					}
					typeIdent, ok := cl.Type.(*ast.Ident)
					if !ok || typeIdent.Name != "Conjecture" {
						return true
					}
					scanned++
					hasRationale := false
					for _, elt := range cl.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok || key.Name != "Rationale" {
							continue
						}
						if resolveStringExpr(kv.Value) != "" {
							hasRationale = true
						}
					}
					if !hasRationale {
						pos := fset.Position(cl.Pos())
						hits = append(hits, thinConjectureHit{
							varName: varName,
							file:    filepath.Base(pos.Filename),
							line:    pos.Line,
						})
					}
					return true
				})
			}
		}
	}

	fmt.Printf("[thin-conjecture] %d conjecture literal(s) scanned, %d with no Rationale\n", scanned, len(hits))
	if len(hits) == 0 {
		fmt.Println("  every conjecture records its own reasoning")
		return 0
	}

	for _, h := range hits {
		fmt.Printf("    %s has no Rationale (%s:%d)\n", h.varName, h.file, h.line)
	}
	fmt.Println("  a Conjecture with no Rationale is generated knowledge with no recorded reasoning —")
	fmt.Println("  fill in Rationale with winze's actual reasoning for the connection.")
	return 1
}

// thinConjectureHit is one Conjecture literal with no recorded reasoning.
type thinConjectureHit struct {
	varName string
	file    string
	line    int
}
