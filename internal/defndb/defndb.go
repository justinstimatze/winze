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
	"go/parser"
	"go/token"
	"os"
	"runtime"
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
	literals   [][]LiteralField // per source file, in sorted path order
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
	n := 0
	idx.eachLiteral(func(f *LiteralField) bool {
		if strings.Contains(f.TypeName, typePattern) {
			n++
		}
		return true
	})
	out := make([]LiteralField, 0, n)
	idx.eachLiteral(func(f *LiteralField) bool {
		if strings.Contains(f.TypeName, typePattern) {
			out = append(out, *f)
		}
		return true
	})
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
	// Count before allocating: ClaimFields selects ~90k of ~130k records on a
	// large corpus, and growing into that from nil copies the result set a
	// dozen times over.
	n := 0
	idx.eachLiteral(func(f *LiteralField) bool {
		if want[f.FieldName] {
			n++
		}
		return true
	})
	out := make([]LiteralField, 0, n)
	idx.eachLiteral(func(f *LiteralField) bool {
		if want[f.FieldName] {
			out = append(out, *f)
		}
		return true
	})
	return out, nil
}

// eachLiteral walks every literal field in index order across all files.
func (idx *index) eachLiteral(fn func(*LiteralField) bool) {
	for _, group := range idx.literals {
		for i := range group {
			if !fn(&group[i]) {
				return
			}
		}
	}
}

// EachField calls fn for every literal field whose name is in names, in index
// order, stopping early if fn returns false.
//
// This is the allocation-free form of ClaimFields/EntityFields. Those build a
// filtered copy — ~90k records, around 9 MB, on a large corpus — which the
// caller then walks exactly once to group into its own structures. The copy is
// pure waste in that pattern, and it was the single largest remaining cost on
// the warm query path. The slice-returning methods stay for callers that need
// to hold or re-scan the result.
func (c *Client) EachField(names []string, fn func(*LiteralField) bool) error {
	idx, err := c.load()
	if err != nil {
		return err
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	idx.eachLiteral(func(f *LiteralField) bool {
		if !want[f.FieldName] {
			return true
		}
		return fn(f)
	})
	return nil
}

// EachFieldOfType calls fn for every literal field whose enclosing literal type
// contains typePattern. The iterator counterpart to LiteralFieldsForType.
func (c *Client) EachFieldOfType(typePattern string, fn func(*LiteralField) bool) error {
	idx, err := c.load()
	if err != nil {
		return err
	}
	idx.eachLiteral(func(f *LiteralField) bool {
		if !strings.Contains(f.TypeName, typePattern) {
			return true
		}
		return fn(f)
	})
	return nil
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

// buildIndex assembles the corpus index, reusing cached per-file fragments for
// files whose size and mtime are unchanged and parsing only the rest. Fragments
// are merged in sorted path order, so the index's slice order — which becomes
// query output order — is stable across runs. The previous implementation
// iterated the package and file maps directly and inherited Go's randomised map
// order.
func buildIndex(dir string) (*index, error) {
	paths, keys, err := scanFiles(dir)
	if err != nil {
		return nil, ErrNotAvailable
	}

	cached := loadCache(dir)
	frags := make(map[string]fragment, len(paths))
	var stale []string
	for _, p := range paths {
		if f, ok := cached[p]; ok && f.Key == keys[p] {
			frags[p] = f
			continue
		}
		stale = append(stale, p)
	}

	if len(stale) > 0 {
		parsed, err := parseFragments(stale, keys)
		if err != nil {
			return nil, err
		}
		for p, f := range parsed {
			frags[p] = f
		}
		storeCache(dir, paths, frags)
	} else if len(cached) != len(frags) {
		// Files were deleted since the cache was written; prune them.
		storeCache(dir, paths, frags)
	}

	return mergeFragments(paths, frags), nil
}

// mergeFragments assembles the queryable index. Entity-var classification lives
// here rather than in the fragments because it needs the corpus-wide role set:
// a role type declared in roles.go decides how a var in another file is read,
// so caching that decision per file would go stale invisibly when roles change.
func mergeFragments(paths []string, frags map[string]fragment) *index {
	idx := &index{roleSet: map[string]bool{}}

	// Size every destination exactly before filling it. A corpus of this shape
	// carries ~130k literal fields; letting append grow that slice from nothing
	// costs more than decoding the whole cache did, because each regrow copies
	// every record written so far.
	var nRole, nDef, nLit, nPrag, nCand int
	for _, p := range paths {
		f := frags[p]
		nRole += len(f.RoleTypes)
		nDef += len(f.Defs)
		nLit += len(f.Literals)
		nPrag += len(f.Pragmas)
		nCand += len(f.Candidates)
	}
	idx.roleTypes = make([]RoleType, 0, nRole)
	idx.defs = make([]SearchResult, 0, nDef)
	idx.pragmas = make([]Pragma, 0, nPrag)
	idx.entityVars = make([]VarRoleInfo, 0, nCand)
	// Literals are kept per file rather than concatenated. Every consumer
	// iterates them; flattening ~130k records into one slice allocated 13 MB
	// and cost more than decoding the cache, to produce a layout nothing needs.
	idx.literals = make([][]LiteralField, 0, len(paths))
	_ = nLit

	for _, p := range paths {
		for _, rt := range frags[p].RoleTypes {
			idx.roleTypes = append(idx.roleTypes, rt)
			idx.roleSet[rt.Name] = true
		}
	}
	for _, p := range paths {
		f := frags[p]
		idx.defs = append(idx.defs, f.Defs...)
		if len(f.Literals) > 0 {
			idx.literals = append(idx.literals, f.Literals)
		}
		idx.pragmas = append(idx.pragmas, f.Pragmas...)
		for _, c := range f.Candidates {
			if idx.roleSet[c.RoleType] {
				idx.entityVars = append(idx.entityVars, c)
			}
		}
	}
	return idx
}

// parseFragments parses the given files in parallel and extracts each one's
// fragment.
func parseFragments(paths []string, keys map[string]fileKey) (map[string]fragment, error) {
	fset := token.NewFileSet()
	out := make([]fragment, len(paths))
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
				f, err := parser.ParseFile(fset, paths[i], nil, parser.ParseComments|parser.SkipObjectResolution)
				if err != nil {
					errs[i] = err
					continue
				}
				out[i] = fragmentFor(f, paths[i], fset)
				out[i].Key = keys[paths[i]]
			}
		}()
	}
	for i := range paths {
		ch <- i
	}
	close(ch)
	wg.Wait()

	res := make(map[string]fragment, len(paths))
	for i, p := range paths {
		if errs[i] != nil {
			// A corpus mid-edit does not parse. Serving the rest is right for a
			// read-side tool, and the file is left out of the cache so the next
			// run retries it.
			continue
		}
		res[p] = out[i]
	}
	return res, nil
}

// fragmentFor extracts one file's contribution: the role types it declares, its
// top-level composite-literal vars, every keyed field inside them, and its
// pragmas.
func fragmentFor(f *ast.File, fname string, fset *token.FileSet) fragment {
	var fr fragment
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		switch gd.Tok {
		case token.TYPE:
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if embedsEntity(ts) {
					fr.RoleTypes = append(fr.RoleTypes, RoleType{Name: ts.Name.Name, SourceFile: fname})
				}
				// Pragmas on type declarations (e.g. //winze:functional on a
				// predicate type) attribute to the type name.
				fr.Pragmas = append(fr.Pragmas, collectPragmas(gd.Doc, ts.Name.Name, fname, fset)...)
				fr.Pragmas = append(fr.Pragmas, collectPragmas(ts.Doc, ts.Name.Name, fname, fset)...)
			}
		case token.VAR:
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
				fr.Defs = append(fr.Defs, SearchResult{
					Name:       name,
					Kind:       "var",
					SourceFile: fname,
					Line:       fset.Position(vs.Pos()).Line,
				})
				if typeName := astutil.CompositeTypeName(cl); typeName != "" {
					fr.Candidates = append(fr.Candidates, VarRoleInfo{
						VarName:    name,
						RoleType:   typeName,
						SourceFile: fname,
					})
				}
				fr.Literals = flattenLiteral(cl, name, fname, fset, fr.Literals, 100)
				fr.Pragmas = append(fr.Pragmas, collectPragmas(gd.Doc, name, fname, fset)...)
				fr.Pragmas = append(fr.Pragmas, collectPragmas(vs.Doc, name, fname, fset)...)
				fr.Pragmas = append(fr.Pragmas, collectPragmas(vs.Comment, name, fname, fset)...)
			}
		}
	}
	return fr
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
