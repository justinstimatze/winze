// Package corpusparse is a minimal go/ast walker for winze corpus
// files. It extracts entities (RoleType{&Entity{...}}), claims
// (PredicateType{Subject, Object, Prov}), and predicate type
// declarations (BinaryRelation / UnaryClaim derivatives), and exposes
// the trip-cycle promotion heuristic.
//
// This package is the third-occurrence extraction: cmd/rot-probe and
// cmd/predicates-suggest both needed corpus-shape parsing, and the
// third client (cmd/lint's brief-check and related) could plug in here
// too — though cmd/lint's heavier machinery (roleType discipline,
// pragma handling, defndb integration, value-conflict claimKey) stays
// in cmd/lint for now. This package is the smaller shared core, not
// the full lint pipeline.
//
// Conservatism: the parser silently skips files it cannot parse and
// vars whose shape does not match. Read-only by design — callers are
// tools surfacing findings, not editors mutating the corpus.
package corpusparse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Entity is a corpus entity declaration: a var whose value is a
// role-type composite literal wrapping &Entity{...}.
type Entity struct {
	VarName  string
	RoleType string
	ID       string
	Name     string
	Brief    string
	Aliases  []string
	File     string
}

// Claim is a typed predicate instance: var X = PredicateType{Subject: A, Object: B, Prov: ...}.
// Unary claims have empty ObjectVar.
type Claim struct {
	VarName       string
	PredicateType string
	SubjectVar    string
	ObjectVar     string
	File          string
	TripGenerated bool // matches IsTripGenerated(VarName)
}

// tripCycleClaimRE matches var names of trip-cycle-promoted claims:
// TripCycleNN<...> (e.g. TripCycle25SurvivorshipBiasCommentaryOnSPP).
// Deliberately does NOT match TripLint/TripBuild/TripLLM/TripFunctional
// auto-generated reify machinery, which are predictions ABOUT trip cycles
// rather than trip promotions themselves. The producer convention lives
// in cmd/metabolism (the --trip path); if it changes, this regex needs
// to follow.
var tripCycleClaimRE = regexp.MustCompile(`^TripCycle\d`)

// IsTripGenerated reports whether a claim var name follows the metabolism
// --trip promotion naming convention. False positives are possible if a
// human-authored claim coincidentally matches; in practice the prefix is
// reserved for the trip pipeline.
func IsTripGenerated(claimVar string) bool {
	return tripCycleClaimRE.MatchString(claimVar)
}

// reifyMachineryRE matches var names of the auto-generated metabolism
// machinery: TripLint*, TripBuild*, TripLLM*, TripFunctional* check
// entities (from cmd/metabolism --reify) and EvidenceSearch* placeholder
// events. These have no meaningful rot-signal content — their Briefs
// are templated, their claims are auto-promotion ResolvedAs / Predicts
// chains. Surfaced by IsReifyMachinery so tools can exclude or down-weight.
var reifyMachineryRE = regexp.MustCompile(`^(TripLint|TripBuild|TripLLM|TripFunctional|EvidenceSearch)`)

// IsReifyMachinery reports whether a var name belongs to the
// metabolism-reify auto-generated families. Used by rot-probe to skip
// these from sampling — they consume LLM budget without offering
// human-actionable rot signal.
func IsReifyMachinery(varName string) bool {
	return reifyMachineryRE.MatchString(varName)
}

// ParseCorpus walks the root .go files in dir (skipping subdirectories,
// _test.go files) and returns the extracted entities and claims. Silently
// skips files it cannot parse and vars whose composite-literal shape
// doesn't match the entity-or-claim pattern.
func ParseCorpus(dir string) ([]Entity, []Claim, error) {
	var entities []Entity
	var claims []Claim

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
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
				for i, nameIdent := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					if ent, ok := tryParseEntity(nameIdent.Name, cl, e.Name()); ok {
						entities = append(entities, ent)
						continue
					}
					if c, ok := tryParseClaim(nameIdent.Name, cl, e.Name()); ok {
						claims = append(claims, c)
					}
				}
			}
		}
	}
	return entities, claims, nil
}

// LoadPredicates walks dir/predicates.go and returns all top-level type
// names derived from BinaryRelation[...] or UnaryClaim[...]. Used by
// cmd/predicates-suggest to populate the existing-predicate exclusion
// list in its prompt. Sorted lexicographically.
func LoadPredicates(dir string) ([]string, error) {
	path := filepath.Join(dir, "predicates.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if isPredicateType(ts.Type) {
				out = append(out, ts.Name.Name)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// tryParseEntity recognises the entity shape: RoleIdent{&Entity{...}}.
func tryParseEntity(varName string, cl *ast.CompositeLit, file string) (Entity, bool) {
	roleType := typeIdent(cl.Type)
	if roleType == "" {
		return Entity{}, false
	}
	for _, elt := range cl.Elts {
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		if typeIdent(inner.Type) != "Entity" {
			continue
		}
		ent := Entity{VarName: varName, RoleType: roleType, File: file}
		for _, ie := range inner.Elts {
			kv, ok := ie.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "ID":
				ent.ID = stringLit(kv.Value)
			case "Name":
				ent.Name = stringLit(kv.Value)
			case "Brief":
				ent.Brief = stringLit(kv.Value)
			case "Aliases":
				ent.Aliases = sliceLit(kv.Value)
			}
		}
		if ent.Name == "" {
			return Entity{}, false
		}
		return ent, true
	}
	return Entity{}, false
}

func tryParseClaim(varName string, cl *ast.CompositeLit, file string) (Claim, bool) {
	predType := typeIdent(cl.Type)
	if predType == "" {
		return Claim{}, false
	}
	c := Claim{VarName: varName, PredicateType: predType, File: file, TripGenerated: IsTripGenerated(varName)}
	hasProv := false
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
			c.SubjectVar = identName(kv.Value)
		case "Object":
			c.ObjectVar = identName(kv.Value)
		case "Prov":
			hasProv = true
		}
	}
	if c.SubjectVar == "" || !hasProv {
		return Claim{}, false
	}
	return c, true
}

func isPredicateType(e ast.Expr) bool {
	idx, ok := e.(*ast.IndexExpr)
	if ok {
		return identNameIs(idx.X, "BinaryRelation") || identNameIs(idx.X, "UnaryClaim")
	}
	list, ok := e.(*ast.IndexListExpr)
	if ok {
		return identNameIs(list.X, "BinaryRelation") || identNameIs(list.X, "UnaryClaim")
	}
	return false
}

func identNameIs(e ast.Expr, want string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == want
}

func typeIdent(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeIdent(t.X)
	}
	return ""
}

func identName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

func stringLit(e ast.Expr) string {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return ""
	}
	return unquote(bl.Value)
}

func sliceLit(e ast.Expr) []string {
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(cl.Elts))
	for _, elt := range cl.Elts {
		if s := stringLit(elt); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '`' && last == '`') {
		return s[1 : len(s)-1]
	}
	return s
}
