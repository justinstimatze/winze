// Package defndb provides typed queries over the corpus: role types, entity
// vars, composite-literal fields, and pragma comments.
//
// It was originally backed by defn's SQL database (github.com/justinstimatze/defn/db),
// which linked the full Dolt engine — go-mysql-server, vitess, four cloud SDKs,
// OpenTelemetry, gRPC — into every winze binary: ~190 indirect modules for a
// 282-line query wrapper. That made `go build`, the load-bearing consistency
// gate this project is built around, cost minutes and peak 1-2 GB RSS at link.
//
// The corpus is a few dozen files of declarative Go. go/ast reads all of it in
// milliseconds, so the database was buying nothing the parser doesn't give.
// The public API is unchanged from the defn-backed version; only the engine
// underneath is different.
package defndb

import (
	"errors"
	"go/ast"
	"go/token"
	"os"
	"strings"
	"sync"

	"github.com/justinstimatze/winze/internal/astutil"
)

// ErrNotAvailable indicates the corpus directory could not be read.
var ErrNotAvailable = errors.New("defndb: corpus not available")

// Client answers typed queries about a corpus directory. The corpus is parsed
// once, lazily, on first query and cached for the Client's lifetime.
type Client struct {
	dir  string
	once sync.Once
	idx  *index
	err  error
}

type index struct {
	roleTypes  []RoleType
	entityVars []VarRoleInfo
	literals   []LiteralField
	pragmas    []Pragma
	defs       []SearchResult
	roleSet    map[string]bool
}

// New creates a Client for the given corpus directory. Returns ErrNotAvailable
// if the directory does not exist or is not a directory.
func New(dir string) (*Client, error) {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return nil, ErrNotAvailable
	}
	return &Client{dir: dir}, nil
}

// Close releases resources. Retained for API compatibility; the AST-backed
// implementation holds nothing that needs releasing.
func (c *Client) Close() error { return nil }

func (c *Client) load() (*index, error) {
	c.once.Do(func() {
		c.idx, c.err = buildIndex(c.dir)
	})
	return c.idx, c.err
}

// RoleType is a type that embeds *Entity.
type RoleType struct {
	Name       string
	SourceFile string
}

// LiteralField is a field from a composite literal initializer. TypeName is
// the type of the immediately-enclosing literal, so a Provenance literal
// inlined inside a claim reports TypeName "Provenance", not the claim's
// predicate type. DefName is always the enclosing top-level var.
type LiteralField struct {
	DefName    string
	TypeName   string
	FieldName  string
	FieldValue string
	SourceFile string
	Line       int
}

// Pragma represents a parsed pragma comment (e.g., //winze:contested).
type Pragma struct {
	DefName    string
	SourceFile string
	Line       int
	Key        string
	Value      string
}

// SearchResult is a definition found by search.
type SearchResult struct {
	Name       string
	Kind       string
	SourceFile string
	Line       int
}

// VarRoleInfo is a var with its role type and source file.
type VarRoleInfo struct {
	VarName    string
	RoleType   string
	SourceFile string
}

// RoleTypes returns all types embedding *Entity.
func (c *Client) RoleTypes() ([]RoleType, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	return idx.roleTypes, nil
}

// RoleTypeSet returns role type names as a set for quick lookup.
func (c *Client) RoleTypeSet() (map[string]bool, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	return idx.roleSet, nil
}

// EntityVarsWithRoles returns entity vars with their role types resolved.
func (c *Client) EntityVarsWithRoles() ([]VarRoleInfo, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	return idx.entityVars, nil
}

// ClaimFields returns Subject/Object/Prov fields from claim composite literals.
func (c *Client) ClaimFields() ([]LiteralField, error) {
	return c.fieldsByName("Subject", "Object", "Prov")
}

// EntityFields returns Name/Brief/ID/Origin/Quote fields from entity literals.
func (c *Client) EntityFields() ([]LiteralField, error) {
	return c.fieldsByName("Name", "Brief", "ID", "Origin", "Quote")
}

// LiteralFieldsForType returns all literal fields whose enclosing literal type
// contains typePattern as a substring.
func (c *Client) LiteralFieldsForType(typePattern string) ([]LiteralField, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	var out []LiteralField
	for _, f := range idx.literals {
		if strings.Contains(f.TypeName, typePattern) {
			out = append(out, f)
		}
	}
	return out, nil
}

func (c *Client) fieldsByName(names ...string) ([]LiteralField, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	var out []LiteralField
	for _, f := range idx.literals {
		if want[f.FieldName] {
			out = append(out, f)
		}
	}
	return out, nil
}

// Pragmas returns all pragma comments whose key starts with prefix.
func (c *Client) Pragmas(prefix string) ([]Pragma, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	var out []Pragma
	for _, p := range idx.pragmas {
		if strings.HasPrefix(p.Key, prefix) {
			out = append(out, p)
		}
	}
	return out, nil
}

// Search returns definitions whose name contains pattern (case-insensitive).
func (c *Client) Search(pattern string) ([]SearchResult, error) {
	idx, err := c.load()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.Trim(pattern, "%"))
	var out []SearchResult
	for _, d := range idx.defs {
		if strings.Contains(strings.ToLower(d.Name), needle) {
			out = append(out, d)
		}
	}
	return out, nil
}

func buildIndex(dir string) (*index, error) {
	pkgs, fset, err := astutil.ParseCorpus(dir)
	if err != nil {
		return nil, ErrNotAvailable
	}
	idx := &index{roleSet: map[string]bool{}}

	// Pass 1: role types. Needed before vars so entity vars can be classified.
	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
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
					if embedsEntity(ts) {
						idx.roleTypes = append(idx.roleTypes, RoleType{Name: ts.Name.Name, SourceFile: fname})
						idx.roleSet[ts.Name.Name] = true
					}
					// Pragmas on type declarations (e.g. //winze:functional
					// on a predicate type) attribute to the type name.
					idx.pragmas = append(idx.pragmas,
						collectPragmas(gd.Doc, ts.Name.Name, fname, fset)...)
					idx.pragmas = append(idx.pragmas,
						collectPragmas(ts.Doc, ts.Name.Name, fname, fset)...)
				}
			}
		}
	}

	// Pass 2: var declarations with composite-literal initializers.
	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
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
					cl, ok := vs.Values[0].(*ast.CompositeLit)
					if !ok {
						continue
					}
					name := vs.Names[0].Name
					idx.defs = append(idx.defs, SearchResult{
						Name:       name,
						Kind:       "var",
						SourceFile: fname,
						Line:       fset.Position(vs.Pos()).Line,
					})
					if typeName := astutil.CompositeTypeName(cl); idx.roleSet[typeName] {
						idx.entityVars = append(idx.entityVars, VarRoleInfo{
							VarName:    name,
							RoleType:   typeName,
							SourceFile: fname,
						})
					}
					idx.literals = flattenLiteral(cl, name, fname, fset, idx.literals, 100)
					idx.pragmas = append(idx.pragmas, collectPragmas(gd.Doc, name, fname, fset)...)
					idx.pragmas = append(idx.pragmas, collectPragmas(vs.Doc, name, fname, fset)...)
					idx.pragmas = append(idx.pragmas, collectPragmas(vs.Comment, name, fname, fset)...)
				}
			}
		}
	}
	return idx, nil
}

func embedsEntity(ts *ast.TypeSpec) bool {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return false
	}
	return astutil.EmbedsEntityPointer(st)
}

// flattenLiteral records every keyed field in a composite literal tree,
// attributing each to the enclosing top-level var but tagging it with the
// type of its immediately-enclosing literal.
func flattenLiteral(cl *ast.CompositeLit, defName, file string, fset *token.FileSet, out []LiteralField, depth int) []LiteralField {
	if depth <= 0 {
		return out
	}
	typeName := astutil.CompositeTypeName(cl)
	for _, elt := range cl.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			out = append(out, LiteralField{
				DefName:    defName,
				TypeName:   typeName,
				FieldName:  key.Name,
				FieldValue: fieldValue(kv.Value),
				SourceFile: file,
				Line:       fset.Position(kv.Pos()).Line,
			})
			if nested := asCompositeLit(kv.Value); nested != nil {
				out = flattenLiteral(nested, defName, file, fset, out, depth-1)
			}
			continue
		}
		// Embedded (unkeyed) element, e.g. RoleType{&Entity{...}}.
		if nested := asCompositeLit(elt); nested != nil {
			out = flattenLiteral(nested, defName, file, fset, out, depth-1)
		}
	}
	return out
}

func asCompositeLit(e ast.Expr) *ast.CompositeLit {
	switch v := e.(type) {
	case *ast.CompositeLit:
		return v
	case *ast.UnaryExpr:
		if cl, ok := v.X.(*ast.CompositeLit); ok {
			return cl
		}
	}
	return nil
}

// fieldValue renders a field's value. String literals (including concatenated
// ones) resolve to their unquoted text; everything else resolves to an
// identifier name, so Subject/Object read as the var they reference and an
// inline &Provenance{...} reads as "Provenance".
func fieldValue(e ast.Expr) string {
	if s := astutil.ResolveStringExpr(e); s != "" {
		return s
	}
	return astutil.ExprIdent(e)
}

// collectPragmas extracts //key:value or //key comments from a comment group.
// Only comments whose text has no spaces before the first colon are treated as
// pragmas, so ordinary prose comments are ignored.
func collectPragmas(cg *ast.CommentGroup, defName, file string, fset *token.FileSet) []Pragma {
	if cg == nil {
		return nil
	}
	var out []Pragma
	for _, c := range cg.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		if text == "" || strings.ContainsAny(text, " \t") {
			continue
		}
		if !strings.Contains(text, ":") {
			continue
		}
		key, value := text, ""
		// A pragma is namespace:name[=value]; keep the namespaced key intact
		// because consumers match on the full "winze:functional" form.
		if eq := strings.Index(text, "="); eq >= 0 {
			key, value = text[:eq], text[eq+1:]
		}
		out = append(out, Pragma{
			DefName:    defName,
			SourceFile: file,
			Line:       fset.Position(c.Pos()).Line,
			Key:        key,
			Value:      value,
		})
	}
	return out
}
