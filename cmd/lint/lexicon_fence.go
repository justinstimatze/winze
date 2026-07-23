package main

// lexicon-fence: keep lexicon's private, non-redistributable content out of the
// public corpus. Lexicon is a *stimulus* winze reads to spark connections among
// its own entities — never a *source* whose text winze quotes. The sanctioned
// reference is a locator inside a Conjecture (`Conjecture{From: "lexicon:lex-0165",
// Rationale: ...}`); Conjecture has no Quote field by design, so a lexicon-sparked
// claim structurally cannot carry lexicon text.
//
// The one hole the compiler can't close: `Provenance.Origin` and `.Quote` are
// free strings, so nothing stops someone pasting a lexicon gloss or an acquired
// primary-source quote into a Provenance and pushing it to a public repo. This
// rule is that fence — a Provenance that references a lexicon locator is a hard
// failure, because the correct attribution for anything lexicon-derived is a
// Conjecture, which cannot quote. Deterministic; the semantic layer the build
// gate can't reach.

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// lexiconLocator matches a lexicon atom reference: the atom-id form (lex-0165)
// or an explicit lexicon: locator prefix. Deliberately specific so the English
// word "lexicon" in ordinary prose doesn't trip it — only a locator does.
var lexiconLocator = regexp.MustCompile(`(?i)\blex-\d{3,4}\b|\blexicon:`)

type lexiconLeak struct {
	varName string
	field   string // "Origin" or "Quote"
	value   string
	file    string
	line    int
}

func lexiconFenceRule(dir string) int {
	fset, files, err := corpusFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[lexicon-fence] error: %v\n", err)
		return 2
	}

	scanned := 0
	var leaks []lexiconLeak
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Values) == 0 {
					continue
				}
				cl, ok := vs.Values[0].(*ast.CompositeLit)
				if !ok {
					continue
				}
				typeIdent, ok := cl.Type.(*ast.Ident)
				if !ok || typeIdent.Name != "Provenance" {
					continue
				}
				scanned++
				for _, elt := range cl.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok || (key.Name != "Origin" && key.Name != "Quote") {
						continue
					}
					val := resolveStringExpr(kv.Value)
					if lexiconLocator.MatchString(val) {
						pos := fset.Position(vs.Names[0].Pos())
						leaks = append(leaks, lexiconLeak{
							varName: vs.Names[0].Name,
							field:   key.Name,
							value:   val,
							file:    filepath.Base(pos.Filename),
							line:    pos.Line,
						})
					}
				}
			}
		}
	}

	fmt.Printf("[lexicon-fence] %d provenance vars scanned, %d lexicon reference(s) in a quoted source\n", scanned, len(leaks))
	if len(leaks) == 0 {
		fmt.Println("  lexicon stays a stimulus, not a quoted source")
		return 0
	}

	for _, l := range leaks {
		fmt.Printf("    %s %s references lexicon (%s:%d): %q\n", l.varName, l.field, l.file, l.line, truncateFence(l.value))
	}
	fmt.Println("  lexicon content must not enter the public corpus. Attribute a lexicon-sparked")
	fmt.Println("  claim with Conjecture{From: \"lexicon:lex-NNNN\", Rationale: ...} — never a quoted")
	fmt.Println("  Provenance. A Conjecture carries no Quote by design, so nothing leaks.")
	return 1
}

func truncateFence(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
