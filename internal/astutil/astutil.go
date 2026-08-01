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
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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
//
// Two departures from the parser.ParseDir this replaces, both measured on a
// 250k-line corpus where ParseDir cost 198ms single-threaded:
//
//   - Object resolution is skipped. It builds ast.Object graphs and package
//     scopes that nothing in winze reads — every consumer walks composite
//     literals by shape. Worth 28% on its own.
//   - Files are parsed in parallel. Parsing is CPU-bound and per-file
//     independent, and a corpus is dozens to hundreds of files.
//
// Comments are still parsed: pragmas (//winze:contested and friends) are real
// corpus content, not decoration.
//
// The returned map is keyed by package name and each package's Files map is
// keyed by path, matching ParseDir. Files that fail to parse are reported in
// the error but do not prevent the rest from being returned — a read-side tool
// should still answer over the files that are valid.
func ParseCorpus(dir string) (map[string]*ast.Package, *token.FileSet, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || !GoFileFilter(info) {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	sort.Strings(paths)

	// One FileSet shared across goroutines: token.FileSet.AddFile is mutex-
	// guarded internally, so concurrent parses may share it, and positions
	// stay comparable across files the way ParseDir's callers expect.
	fset := token.NewFileSet()
	files := make([]*ast.File, len(paths))
	errs := make([]error, len(paths))

	workers := runtime.GOMAXPROCS(0)
	if workers > len(paths) {
		workers = len(paths)
	}
	var wg sync.WaitGroup
	ch := make(chan int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				files[i], errs[i] = parser.ParseFile(fset, paths[i], nil,
					parser.ParseComments|parser.SkipObjectResolution)
			}
		}()
	}
	for i := range paths {
		ch <- i
	}
	close(ch)
	wg.Wait()

	pkgs := map[string]*ast.Package{}
	var firstErr error
	for i, f := range files {
		if errs[i] != nil {
			if firstErr == nil {
				firstErr = errs[i]
			}
			continue
		}
		name := f.Name.Name
		pkg := pkgs[name]
		if pkg == nil {
			pkg = &ast.Package{Name: name, Files: map[string]*ast.File{}}
			pkgs[name] = pkg
		}
		pkg.Files[paths[i]] = f
	}
	return pkgs, fset, firstErr
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
			subj = subjectObjectIdent(kv.Value)
		case "Object":
			obj = subjectObjectIdent(kv.Value)
		}
	}
	return
}

// subjectObjectIdent resolves a Subject/Object value to the entity var it
// names. A plain ident (Subject: Alice) is the var directly. A selector
// (Subject: Survivor.Entity) names the var in its X half — the shape
// winze-edit merge's audit claim writes to reach the embedded *Entity of a
// role-typed survivor whose concrete type is not known at claim-render time.
func subjectObjectIdent(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		if id, ok := v.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
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

// EmbedsEntityPointer reports whether a struct embeds *Entity (an anonymous
// *Entity field) — the winze role-type marker. lint and internal/defndb each
// re-implemented this walk (a calque scan flagged the twin); one definition now.
func EmbedsEntityPointer(st *ast.StructType) bool {
	if st == nil || st.Fields == nil {
		return false
	}
	for _, field := range st.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if ident, ok := star.X.(*ast.Ident); ok && ident.Name == "Entity" {
			return true
		}
	}
	return false
}
