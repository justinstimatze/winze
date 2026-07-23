package main

// set-brief — replace an entity's Brief (and optionally its Name) in place.
//
// `add` appends and `rename`/`merge` restructure, but neither can revise the
// prose of an existing entity. A memory whose fact changed, or a Brief that was
// wrong, has to be editable without hand-touching the .go file — otherwise the
// dedup guard's "update that memory instead" advice dead-ends. This is the
// revise primitive: find the var's Brief string literal by AST, splice a new
// value at its byte offsets, run the same gofmt+build+vet gate as every other
// mutation, revert on failure. The caller (winze_update) commits.

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"

	"github.com/justinstimatze/winze/internal/astutil"
	"github.com/justinstimatze/winze/internal/corpuslock"
)

func cmdSetBrief(args []string) int {
	fs := flag.NewFlagSet("set-brief", flag.ExitOnError)
	varName := fs.String("var", "", "entity var name to revise (e.g. RecallHookRelevanceGate)")
	brief := fs.String("brief", "", "new Brief prose")
	name := fs.String("name", "", "optional: also replace the display Name")
	root := fs.String("root", ".", "winze repo root")
	dryRun := fs.Bool("dry-run", false, "report the target and new value, write nothing")
	_ = fs.Parse(args)

	if *varName == "" || *brief == "" {
		fmt.Fprintln(os.Stderr, "set-brief: --var and --brief are required")
		return 2
	}

	file, edits, err := planSetBrief(*root, *varName, *brief, *name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set-brief: %v\n", err)
		return 1
	}

	if *dryRun {
		fmt.Printf("would revise %s in %s (%d field(s))\n", *varName, file, len(edits))
		return 0
	}

	unlock, err := corpuslock.Acquire(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set-brief: lock: %v\n", err)
		return 1
	}
	defer unlock()

	// Re-plan under the lock: another writer may have moved offsets between the
	// preview plan and acquiring the lock.
	file, edits, err = planSetBrief(*root, *varName, *brief, *name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set-brief: %v\n", err)
		return 1
	}

	src, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set-brief: read %s: %v\n", file, err)
		return 1
	}
	if err := os.WriteFile(file, applyEdits(src, edits), 0o644); err != nil {
		_ = os.WriteFile(file, src, 0o644)
		fmt.Fprintf(os.Stderr, "set-brief: write %s: %v (reverted)\n", file, err)
		return 1
	}

	steps := [][]string{
		{"gofmt", "-w", file},
		{"go", "build", "."},
		{"go", "vet", "."},
	}
	for _, step := range steps {
		if out, err := runCmd(*root, step[0], step[1:]...); err != nil {
			_ = os.WriteFile(file, src, 0o644)
			fmt.Fprintf(os.Stderr, "%s failed (reverted):\n%s\n", step[0], out)
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "revised %s in %s (build gate passed)\n", *varName, file)
	return 0
}

// planSetBrief finds the target var's Brief (and optionally Name) string-literal
// value nodes and returns the file plus the byte-offset edits to apply.
func planSetBrief(root, varName, brief, name string) (string, []edit, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, root, astutil.GoFileFilter, parser.ParseComments)
	if err != nil {
		return "", nil, fmt.Errorf("parse %s: %w", root, err)
	}

	for _, pkg := range pkgs {
		for path, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) == 0 || len(vs.Values) == 0 {
						continue
					}
					if vs.Names[0].Name != varName {
						continue
					}
					fields := entityFields(vs.Values[0])
					briefLit := fields["Brief"]
					if briefLit == nil {
						return "", nil, fmt.Errorf("%s has no Brief field", varName)
					}
					var edits []edit
					edits = append(edits, litEdit(fset, briefLit, briefLiteral(brief)))
					if name != "" {
						if nameLit := fields["Name"]; nameLit != nil {
							edits = append(edits, litEdit(fset, nameLit, strconv.Quote(name)))
						}
					}
					return path, edits, nil
				}
			}
		}
	}
	return "", nil, fmt.Errorf("var %s not found under %s", varName, root)
}

// entityFields returns the string-literal value nodes of an entity composite
// literal keyed by field name, descending through any role wrapper
// (Concept{&Entity{...}}) and the &Entity address-of.
func entityFields(v ast.Expr) map[string]*ast.BasicLit {
	out := map[string]*ast.BasicLit{}
	ast.Inspect(v, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}
		if lit, ok := kv.Value.(*ast.BasicLit); ok {
			out[key.Name] = lit
		}
		return true
	})
	return out
}

func litEdit(fset *token.FileSet, lit *ast.BasicLit, repl string) edit {
	start := fset.Position(lit.Pos()).Offset
	end := fset.Position(lit.End()).Offset
	return edit{offset: start, length: end - start, repl: repl}
}

// briefLiteral renders a brief as a Go string literal, preferring a raw string
// (the corpus convention) unless the text contains a backtick, which a raw
// string cannot hold — then fall back to a quoted string.
func briefLiteral(s string) string {
	if !containsByte(s, '`') {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}
