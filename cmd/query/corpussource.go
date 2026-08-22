package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justinstimatze/winze/internal/astutil"
	"github.com/justinstimatze/winze/internal/defndb"
)

// corpusSource is the read side's view of a corpus. Every query mode reaches
// its data through these calls and nothing else, so a second implementation is
// the whole cost of reading a corpus defn cannot index.
//
// *defndb.Client satisfies it as written; astSource below is the other one.
type corpusSource interface {
	RoleTypeSet() (map[string]bool, error)
	EntityVarsWithRoles() ([]defndb.VarRoleInfo, error)
	EachField(names []string, fn func(*defndb.LiteralField) bool) error
	EachFieldOfType(typePattern string, fn func(*defndb.LiteralField) bool) error
	EachFieldWithValuePrefix(valuePrefix string, names []string, fn func(*defndb.LiteralField) bool) error
	EachFieldForDefs(defNames map[string]bool, names []string, fn func(*defndb.LiteralField) bool) error
	Close() error
}

// openCorpus picks the backend for dir: the parse-tree reader for a meld, defn
// for everything else.
//
// A meld is the case defn structurally cannot serve. Its ingest runs
// packages.Load, which wants a module that type-checks, and a meld is built not
// to be one — no go.mod, duplicate identifiers across the melded stores (which
// is exactly what makes a meld read-only), and a corpus file importing
// winze/internal from outside the winze module. None of the three stops a
// parser.
//
// The choice reads the meld manifest rather than trying defn and falling back.
// A fallback would turn every real defn problem — a stale index, an ingest
// failure, a missing binary — into a silent switch to the slower path with no
// symptom but different answers.
func openCorpus(dir string) (corpusSource, error) {
	if _, err := os.Stat(filepath.Join(dir, meldManifestName)); err == nil {
		return newASTSource(dir)
	}
	return defndb.New(dir)
}

// meldManifestName marks a directory as a winze-meld. cmd/meld writes it; this
// is the only thing the read side needs from it, so it is duplicated rather
// than imported — the two commands share no package, and a constant is a
// cheaper coupling than one invented for it.
const meldManifestName = ".winze-meld.json"

// astSource answers the corpusSource calls from the parse tree instead of from
// defn's index.
//
// It reads the corpus once at open and holds the same LiteralField stream
// defn's client hands out, so the six query methods are filters over one slice
// and every read path above them is untouched. Building the same stream rather
// than the same records is what keeps this to one file: --theories, --claims,
// --provenance and --disputes each walk defn's field stream directly instead of
// going through the index, so a backend at the record level would have had to
// rewrite four modes as well.
type astSource struct {
	fields []defndb.LiteralField
	roles  map[string]bool
	vars   []defndb.VarRoleInfo
}

func newASTSource(dir string) (*astSource, error) {
	files, fset, err := astutil.ParseCorpus(dir)
	// A file that will not parse costs its own claims and nothing else. A meld
	// is a union of stores pinned at different commits, so refusing the whole
	// read over one bad file would make the good stores unreadable too.
	if len(files) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, os.ErrNotExist
	}

	s := &astSource{roles: astutil.CollectRoleTypes(files)}
	for path, f := range files {
		base := filepath.Base(path)
		// The file's own bytes, so a field's value can be the exact source text
		// defn stores rather than a reprinted equivalent. A failed read leaves
		// src nil and fieldValue falls back to the printer, which differs only
		// in the whitespace inside an inline literal.
		src, _ := os.ReadFile(path)
		eachTopLevelVar(f, func(name string, value ast.Expr) {
			cl, typeName := compositeOf(value)
			if cl == nil {
				return
			}
			if s.roles[typeName] {
				s.vars = append(s.vars, defndb.VarRoleInfo{
					VarName: name, RoleType: typeName, SourceFile: base,
				})
			}
			fileWalk{s: s, fset: fset, src: src, file: base, def: name}.walk(typeName, cl)
		})
	}

	// Both orders are the ones the read paths call "index order" and depend on
	// for reproducible output; SortLiteralFields is defn's own so the two
	// backends cannot drift apart on it.
	defndb.SortLiteralFields(s.fields)
	sort.Slice(s.vars, func(i, j int) bool { return s.vars[i].VarName < s.vars[j].VarName })
	return s, nil
}

// eachTopLevelVar calls fn for every `var NAME = <expr>` at the top level of f.
// The corpus is nothing but these, and lifting the decl/spec/name walk out of
// newASTSource leaves the part that decides anything three levels shallower.
func eachTopLevelVar(f *ast.File, fn func(name string, value ast.Expr)) {
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i < len(vs.Values) {
					fn(name.Name, vs.Values[i])
				}
			}
		}
	}
}

// fileWalk carries the per-file context down a nested literal: which file's
// bytes to slice a value out of, and which top-level var every field below
// belongs to.
type fileWalk struct {
	s    *astSource
	fset *token.FileSet
	src  []byte
	file string
	def  string
}

// walk emits one LiteralField per keyed field in cl, then recurses into any
// literal nested inside it.
//
// typeName is the innermost enclosing literal's type, matching defn's rule that
// a Provenance inlined in a claim reports Provenance rather than the claim's
// predicate. The def name stays the top-level var the whole way down, which is
// what makes an inline Conjecture findable as that claim's provenance.
func (w fileWalk) walk(typeName string, cl *ast.CompositeLit) {
	if typeName == "" {
		// An untyped literal — both the []*ExternalTerm{...} in external.go and
		// each {Name: …} element inside it. defn indexes neither, and matching
		// that keeps the two field streams identical. Nothing is lost: an entity
		// or a claim always spells its type out, which is what the corpus build
		// gate is checking.
		return
	}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			// A positional element, which in this corpus means the wrapped
			// entity in Concept{&Entity{...}}. It has no field name to report,
			// so descend without emitting a row: the Entity's own fields are
			// what the read paths are after, tagged Entity rather than Concept.
			if inner, innerType := compositeOf(elt); inner != nil {
				w.walk(innerType, inner)
			}
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		w.s.fields = append(w.s.fields, defndb.LiteralField{
			DefName:    w.def,
			TypeName:   typeName,
			FieldName:  key.Name,
			FieldValue: w.fieldValue(kv.Value),
			SourceFile: w.file,
			Line:       w.fset.Position(kv.Pos()).Line,
		})
		if inner, innerType := compositeOf(kv.Value); inner != nil {
			w.walk(innerType, inner)
		}
	}
}

// compositeOf unwraps &T{...} and T{...} to the literal and its type name,
// returning nil for anything else.
func compositeOf(e ast.Expr) (*ast.CompositeLit, string) {
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return nil, ""
	}
	return cl, astutil.CompositeTypeName(cl)
}

// fieldValue renders a field the way defn stores it: a string literal unquoted,
// anything else as the exact source text it was written as.
//
// The unquoting is load-bearing at one place and invisible everywhere else.
// Single-entity lookups find a var's claims by value prefix over Subject and
// Object, which hold bare identifiers — Apophenia, DragonInGarageArgument.Entity
// — so a quoted rendering there matches nothing and the entity reads as having
// no claims at all.
//
// Slicing the source rather than reprinting matters only for an inline literal
// (a Conjecture written inside its claim), where go/printer re-aligns the
// fields with tabs and the value stops being byte-identical to defn's. Nothing
// matches on that text, but "both backends answer identically" is a far easier
// property to test than "identically modulo whitespace".
func (w fileWalk) fieldValue(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			if s, err := strconv.Unquote(v.Value); err == nil {
				return s
			}
		}
	case *ast.BinaryExpr:
		// "a" + "b", which the corpus uses for long Briefs.
		if s := astutil.ResolveStringExpr(v); s != "" {
			return s
		}
	}
	if s, ok := w.sourceText(e); ok {
		return s
	}
	var b strings.Builder
	if err := printer.Fprint(&b, w.fset, e); err != nil {
		return ""
	}
	return b.String()
}

// sourceText returns e exactly as it was written, dedented, reporting false
// when the file's bytes are not available to slice.
func (w fileWalk) sourceText(e ast.Expr) (string, bool) {
	if w.src == nil {
		return "", false
	}
	lo := w.fset.Position(e.Pos()).Offset
	hi := w.fset.Position(e.End()).Offset
	if lo < 0 || hi > len(w.src) || lo >= hi {
		return "", false
	}
	return dedent(w.src, lo, string(w.src[lo:hi])), true
}

// dedent strips the indentation of the line an expression starts on from every
// later line of its text, which is how defn stores a multi-line value: an
// inline Conjecture reads the same whether it was written one level deep or
// three. Without it the two backends print the same literal at different
// indents and only that differs.
func dedent(src []byte, lo int, text string) string {
	if !strings.Contains(text, "\n") {
		return text
	}
	bol := bytes.LastIndexByte(src[:lo], '\n') + 1
	// The line up to the expression is `\tProv: ` for an inline literal, so
	// take only the leading whitespace run, not the field name with it.
	n := 0
	for n < lo-bol && (src[bol+n] == '\t' || src[bol+n] == ' ') {
		n++
	}
	prefix := string(src[bol : bol+n])
	if prefix == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.TrimPrefix(lines[i], prefix)
	}
	return strings.Join(lines, "\n")
}

func (s *astSource) RoleTypeSet() (map[string]bool, error)              { return s.roles, nil }
func (s *astSource) EntityVarsWithRoles() ([]defndb.VarRoleInfo, error) { return s.vars, nil }
func (s *astSource) Close() error                                       { return nil }

// each hands fn every field keep accepts, in index order, stopping when fn
// says so. The four Each* methods below are this with a different predicate.
func (s *astSource) each(keep func(*defndb.LiteralField) bool, fn func(*defndb.LiteralField) bool) error {
	for i := range s.fields {
		f := &s.fields[i]
		if !keep(f) {
			continue
		}
		if !fn(f) {
			return nil
		}
	}
	return nil
}

// nameSet turns a field-name list into a lookup. An empty list means every
// field, which is how defn's filter reads an empty FieldNames.
func nameSet(names []string) func(string) bool {
	if len(names) == 0 {
		return func(string) bool { return true }
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	return func(n string) bool { return want[n] }
}

func (s *astSource) EachField(names []string, fn func(*defndb.LiteralField) bool) error {
	want := nameSet(names)
	return s.each(func(f *defndb.LiteralField) bool { return want(f.FieldName) }, fn)
}

func (s *astSource) EachFieldOfType(typePattern string, fn func(*defndb.LiteralField) bool) error {
	return s.each(func(f *defndb.LiteralField) bool {
		return strings.Contains(f.TypeName, typePattern)
	}, fn)
}

func (s *astSource) EachFieldWithValuePrefix(valuePrefix string, names []string, fn func(*defndb.LiteralField) bool) error {
	want := nameSet(names)
	return s.each(func(f *defndb.LiteralField) bool {
		return want(f.FieldName) && strings.HasPrefix(f.FieldValue, valuePrefix)
	}, fn)
}

func (s *astSource) EachFieldForDefs(defNames map[string]bool, names []string, fn func(*defndb.LiteralField) bool) error {
	want := nameSet(names)
	return s.each(func(f *defndb.LiteralField) bool {
		return want(f.FieldName) && defNames[f.DefName]
	}, fn)
}
