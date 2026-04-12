// Package astutil provides shared AST walking utilities for winze's cmd/
// tools and tests. Extracted to eliminate ~13 instances of duplicated AST
// parsing boilerplate across cmd/metabolism, cmd/query, cmd/lint, cmd/topology,
// and corpus_test.go.
package astutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

// GoFileFilter is a parser.ParseDir filter that accepts all .go files.
func GoFileFilter(info os.FileInfo) bool {
	return strings.HasSuffix(info.Name(), ".go")
}

// IsInfraFile returns true for schema/role/predicate files that should be
// skipped when scanning for entity or claim var declarations.
func IsInfraFile(name string) bool {
	infra := map[string]bool{
		"schema.go":       true,
		"roles.go":        true,
		"predicates.go":   true,
		"design_roles.go": true,
	}
	return infra[name] || strings.HasSuffix(name, "_test.go")
}

// ParseCorpus parses all .go files in dir and returns the package map.
func ParseCorpus(dir string) (map[string]*ast.Package, *token.FileSet, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, GoFileFilter, parser.ParseComments)
	return pkgs, fset, err
}

// ResolveStringExpr extracts a string value from an AST expression,
// handling both simple string literals and concatenated strings ("a" + "b").
// Recursion is bounded to 100 levels to prevent stack overflow on
// pathological input.
func ResolveStringExpr(e ast.Expr) string {
	return resolveStringExprDepth(e, 100)
}

func resolveStringExprDepth(e ast.Expr, depth int) string {
	if depth <= 0 {
		return ""
	}
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return ""
		}
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			return v.Value
		}
		return s
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return ""
		}
		return resolveStringExprDepth(v.X, depth-1) + resolveStringExprDepth(v.Y, depth-1)
	default:
		return ""
	}
}

// Unquote extracts a string value from an AST expression.
// Delegates to ResolveStringExpr for concatenation support.
func Unquote(e ast.Expr) string {
	return ResolveStringExpr(e)
}

// CompositeTypeName extracts the type name from a composite literal.
// Handles both simple types (Foo{}) and generic types (Foo[Bar]{}).
func CompositeTypeName(cl *ast.CompositeLit) string {
	switch t := cl.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// ExtractStringField extracts a named string field value from a composite
// literal. Recurses into RoleType{&Entity{...}} patterns. Uses
// ResolveStringExpr to handle concatenated strings. Recursion is bounded
// to 100 levels to prevent stack overflow on pathological input.
func ExtractStringField(cl *ast.CompositeLit, fieldName string) string {
	return extractStringFieldDepth(cl, fieldName, 100)
}

func extractStringFieldDepth(cl *ast.CompositeLit, fieldName string, depth int) string {
	if depth <= 0 {
		return ""
	}
	for _, elt := range cl.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == fieldName {
				return ResolveStringExpr(kv.Value)
			}
			continue
		}
		// Nested: RoleType{&Entity{...}}
		if uexpr, ok := elt.(*ast.UnaryExpr); ok {
			if nested, ok := uexpr.X.(*ast.CompositeLit); ok {
				if v := extractStringFieldDepth(nested, fieldName, depth-1); v != "" {
					return v
				}
			}
		}
	}
	return ""
}

// ExtractEntityBrief extracts the Brief field from an entity composite literal.
// Handles both direct Entity{Brief: "..."} and RoleType{&Entity{Brief: "..."}} patterns.
func ExtractEntityBrief(cl *ast.CompositeLit) string {
	for _, elt := range cl.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Brief" {
				return Unquote(kv.Value)
			}
			continue
		}
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, innerElt := range inner.Elts {
			if kv, ok := innerElt.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Brief" {
					return Unquote(kv.Value)
				}
			}
		}
	}
	return ""
}

// ExprIdent extracts an identifier name from an AST expression.
// Handles *ast.Ident, *ast.UnaryExpr (&Struct{}), *ast.SelectorExpr (pkg.Name),
// and *ast.CompositeLit (Type{}).
func ExprIdent(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.UnaryExpr:
		if cl, ok := v.X.(*ast.CompositeLit); ok {
			return CompositeTypeName(cl)
		}
		if id, ok := v.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.SelectorExpr:
		if id, ok := v.X.(*ast.Ident); ok {
			return id.Name + "." + v.Sel.Name
		}
	case *ast.CompositeLit:
		return CompositeTypeName(v)
	}
	return ""
}

// CollectRoleTypes finds type names that embed *Entity (role types like
// Person, Concept, Hypothesis, etc.) by scanning type declarations.
func CollectRoleTypes(pkgs map[string]*ast.Package) map[string]bool {
	roleTypes := map[string]bool{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
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
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range st.Fields.List {
						if len(field.Names) > 0 {
							continue
						}
						star, ok := field.Type.(*ast.StarExpr)
						if !ok {
							continue
						}
						ident, ok := star.X.(*ast.Ident)
						if ok && ident.Name == "Entity" {
							roleTypes[ts.Name.Name] = true
						}
					}
				}
			}
		}
	}
	return roleTypes
}

// ExtractSubjectObject extracts Subject and Object identifier names from
// a claim composite literal.
func ExtractSubjectObject(cl *ast.CompositeLit) (subj, obj string) {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Subject":
			if id, ok := kv.Value.(*ast.Ident); ok {
				subj = id.Name
			}
		case "Object":
			if id, ok := kv.Value.(*ast.Ident); ok {
				obj = id.Name
			}
		}
	}
	return
}

// VarDecl represents a top-level var declaration with its composite literal.
type VarDecl struct {
	Name     string
	TypeName string
	Lit      *ast.CompositeLit
	File     string // base filename
}

// WalkVarDecls iterates over all top-level var declarations in the packages,
// calling fn for each one that has a composite literal initializer.
func WalkVarDecls(pkgs map[string]*ast.Package, fn func(VarDecl)) {
	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
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
					fn(VarDecl{
						Name:     vs.Names[0].Name,
						TypeName: CompositeTypeName(cl),
						Lit:      cl,
						File:     fname,
					})
				}
			}
		}
	}
}
