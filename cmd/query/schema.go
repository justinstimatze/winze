package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
)

// runSchema introspects the corpus's own type declarations and prints the model
// an author needs to add to it: the roles, the predicate signatures, and the
// two attribution modes. It reads the live schema via go/ast, so it can never
// drift from the code the way a hand-written SCHEMA.md would.

type predSig struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`   // BinaryRelation | UnaryClaim
	Params string `json:"params"` // e.g. "Person, Hypothesis"
}

type schemaModel struct {
	Roles       []string  `json:"roles"`
	Predicates  []predSig `json:"predicates"`
	Attribution []string  `json:"attribution"` // Provenance, Conjecture — whichever the corpus defines
}

func runSchema(dir string, jsonOut bool) {
	m := extractSchema(dir)
	if jsonOut {
		out, _ := json.MarshalIndent(m, "", "  ")
		fmt.Println(string(out))
		return
	}
	printSchema(dir, m)
}

func extractSchema(dir string) schemaModel {
	fset := token.NewFileSet()
	entries, _ := os.ReadDir(dir)
	var roles []string
	var preds []predSig
	attribution := map[string]bool{}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				name := ts.Name.Name
				switch t := ts.Type.(type) {
				case *ast.StructType:
					if name == "Provenance" || name == "Conjecture" {
						attribution[name] = true
					} else if isRoleStruct(t) {
						roles = append(roles, name)
					}
				case *ast.IndexExpr: // Foo[X] — a one-parameter generic (UnaryClaim[S])
					if base, ok := t.X.(*ast.Ident); ok && isPredicateBase(base.Name) {
						preds = append(preds, predSig{name, base.Name, exprStr(fset, t.Index)})
					}
				case *ast.IndexListExpr: // Foo[X, Y] — a two-parameter generic (BinaryRelation[S, O])
					if base, ok := t.X.(*ast.Ident); ok && isPredicateBase(base.Name) {
						preds = append(preds, predSig{name, base.Name, joinExprs(fset, t.Indices)})
					}
				}
			}
		}
	}
	sort.Strings(roles)
	sort.Slice(preds, func(i, j int) bool { return preds[i].Name < preds[j].Name })
	var attr []string
	for _, a := range []string{"Provenance", "Conjecture"} {
		if attribution[a] {
			attr = append(attr, a)
		}
	}
	return schemaModel{Roles: roles, Predicates: preds, Attribution: attr}
}

func isPredicateBase(name string) bool {
	return name == "BinaryRelation" || name == "UnaryClaim"
}

// isRoleStruct reports whether a struct is a role: a single embedded *Entity.
func isRoleStruct(s *ast.StructType) bool {
	if s.Fields == nil || len(s.Fields.List) != 1 {
		return false
	}
	fld := s.Fields.List[0]
	if len(fld.Names) != 0 { // embedded fields have no names
		return false
	}
	star, ok := fld.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "Entity"
}

func exprStr(fset *token.FileSet, e ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

func joinExprs(fset *token.FileSet, es []ast.Expr) string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = exprStr(fset, e)
	}
	return strings.Join(parts, ", ")
}

func printSchema(dir string, m schemaModel) {
	fmt.Printf("winze schema · %s\n\n", dir)

	fmt.Println("SHAPES")
	fmt.Println("  entity     var X = <Role>{&Entity{ID, Name, Kind, Brief}}   (roles embed *Entity)")
	fmt.Println("  claim      var C = <Predicate>{Subject: X, Object: Y, Prov: <Attribution>}")
	fmt.Println("  predicate  type Foo BinaryRelation[S, O]   or   UnaryClaim[S]")
	fmt.Println()

	if len(m.Attribution) > 0 {
		fmt.Println("ATTRIBUTION")
		for _, a := range m.Attribution {
			switch a {
			case "Provenance":
				fmt.Println("  Provenance  sourced — Origin + Quote (exact source text) + IngestedAt/By")
			case "Conjecture":
				fmt.Println("  Conjecture  winze's own generation — GeneratedBy + Rationale; NO Quote by design")
			}
		}
		fmt.Println()
	}

	fmt.Printf("ROLES (%d)\n  %s\n\n", len(m.Roles), strings.Join(m.Roles, "  "))

	fmt.Printf("PREDICATES (%d)\n", len(m.Predicates))
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, p := range m.Predicates {
		fmt.Fprintf(w, "  %s\t%s[%s]\n", p.Name, p.Kind, p.Params)
	}
	w.Flush()
}
