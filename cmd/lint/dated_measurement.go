package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"

	"github.com/justinstimatze/winze/internal/cliutil"
)

func datedMeasurementRule(dir string) int {
	fset, files, err := corpusFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[dated-measurement] error: %v\n", err)
		return 2
	}

	scanned := 0
	var hits []datedMeasurementHit
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
					kv, ok := n.(*ast.KeyValueExpr)
					if !ok {
						return true
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok || !datedFieldNames[key.Name] {
						return true
					}
					value := resolveStringExpr(kv.Value)
					if value == "" {
						return true
					}
					scanned++
					if !measurementPattern.MatchString(value) && !statusPattern.MatchString(value) {
						return true
					}
					if isoDatePattern.MatchString(value) {
						return true
					}
					pos := fset.Position(kv.Pos())
					hits = append(hits, datedMeasurementHit{
						varName: varName,
						field:   key.Name,
						value:   value,
						file:    filepath.Base(pos.Filename),
						line:    pos.Line,
					})
					return true
				})
			}
		}
	}

	fmt.Printf("[dated-measurement] %d Brief/Rationale/Quote string(s) scanned, %d measurement-shaped with no date\n", scanned, len(hits))
	if len(hits) == 0 {
		fmt.Println("  every measurement-shaped claim carries its own date")
		return 0
	}

	for _, h := range hits {
		fmt.Printf("    %s.%s (%s:%d): %s\n", h.varName, h.field, h.file, h.line, cliutil.Truncate(h.value, 100))
	}
	fmt.Println("  a claim about a number or present state with no ISO date reads as live and rots silently —")
	fmt.Println("  add the date it was true, or the date it was measured.")
	fmt.Println("  (advisory — see FEEDBACK-2026-09-02.md#4 for why this doesn't gate the build yet)")
	return 0
}

type datedMeasurementHit struct {
	varName string
	field   string
	value   string
	file    string
	line    int
}

// datedMeasurementRule flags a Brief, Rationale, or Quote string that reads
// as a measurement or a claim about present state but carries no date to say
// when it was true — README known-problem 5, made concrete: internal/defndb's
// package doc asserted defn was Dolt-backed and too heavy to link, that
// stopped being true, and nothing noticed until an agent read the stale claim
// as a live measurement and built ~500 lines on it. This is the "lint rule
// flagging measurement-shaped prose carrying no date" the README asked for.
//
// A first cut, not a precise one: measurementPattern catches a bare
// number-plus-unit (24GB, 97%, 12ms) and statusPattern catches present-tense
// status words (currently, now, takes, ...) that read as "this is how it is"
// rather than "this is how it was." Either match with no ISO date (\d{4}-\d{2}-\d{2})
// anywhere in the same string is a hit. Deliberately loose on purpose — see
// FEEDBACK-2026-09-02.md#4 for the exact ask — and advisory rather than
// blocking (return 0 regardless of hits, like briefCheckRule) until a real
// run against the corpus says whether the false-positive rate is low enough
// to gate on, the same replay-before-wiring discipline onsetter asks of its
// own asks.
var (
	measurementPattern = regexp.MustCompile(`\b\d+(\.\d+)?\s*(ms|s|KLOC|lines|%|stars|commits|MB|GB)\b`)
	statusPattern      = regexp.MustCompile(`\b(currently|now|is backed by|links in|takes)\b`)
	isoDatePattern     = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	datedFieldNames    = map[string]bool{"Brief": true, "Rationale": true, "Quote": true}
)
