package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// tripCycleClaimRE matches var names of trip-cycle-promoted claims:
// TripCycleNN<...> (the actual cycle promotion, e.g.
// TripCycle25SurvivorshipBiasCommentaryOnSuperiorPatternProcessing).
// Deliberately does NOT match TripLint/TripBuild/TripLLM/TripFunctional
// auto-generated reify machinery, which are predictions ABOUT trip cycles
// rather than trip promotions themselves.
var tripCycleClaimRE = regexp.MustCompile(`^TripCycle\d`)

func isTripGenerated(claimVar string) bool {
	return tripCycleClaimRE.MatchString(claimVar)
}

// entity is a corpus entity declaration: a var whose value is a role-type
// composite literal wrapping &Entity{Name, Brief, Aliases, ...}.
type entity struct {
	varName  string // Go var name, e.g. "KlausConrad"
	roleType string // role type, e.g. "Person", "Concept"
	id       string // Entity.ID slot
	name     string // Entity.Name slot
	brief    string // Entity.Brief slot
	aliases  []string
	file     string
}

// claim is a typed predicate instance: var X = PredicateType{Subject: A, Object: B, Prov: ...}.
// Unary claims have empty objectVar.
type claim struct {
	varName         string // claim var name
	predicateType   string // predicate type, e.g. "Proposes"
	subjectVar      string // subject var ref (Ident name)
	objectVar       string // object var ref; empty if unary
	file            string
	tripGenerated   bool   // true if the claim was promoted by a trip cycle (heuristic: TripCycleNN... prefix)
}

// parseCorpus walks the root .go files (non-test) and extracts entities + claims.
// Skips cmd/ subdirectories. Conservative: silently skips files it can't parse,
// silently skips vars whose value shape doesn't match the entity/claim pattern.
func parseCorpus(dir string) ([]entity, []claim, error) {
	var entities []entity
	var claims []claim

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

// tryParseEntity recognises the entity shape: RoleIdent{&Entity{...}}.
func tryParseEntity(varName string, cl *ast.CompositeLit, file string) (entity, bool) {
	roleType := typeIdent(cl.Type)
	if roleType == "" {
		return entity{}, false
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
		innerType := typeIdent(inner.Type)
		if innerType != "Entity" {
			continue
		}
		ent := entity{varName: varName, roleType: roleType, file: file}
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
				ent.id = stringLit(kv.Value)
			case "Name":
				ent.name = stringLit(kv.Value)
			case "Brief":
				ent.brief = stringLit(kv.Value)
			case "Aliases":
				ent.aliases = sliceLit(kv.Value)
			}
		}
		if ent.name == "" {
			return entity{}, false
		}
		return ent, true
	}
	return entity{}, false
}

// tryParseClaim recognises the claim shape: PredicateType{Subject: X, Object: Y, Prov: ...}.
// Unary claims (UnaryClaim derivatives) have only Subject and Prov.
func tryParseClaim(varName string, cl *ast.CompositeLit, file string) (claim, bool) {
	predType := typeIdent(cl.Type)
	if predType == "" {
		return claim{}, false
	}
	c := claim{varName: varName, predicateType: predType, file: file, tripGenerated: isTripGenerated(varName)}
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
			c.subjectVar = identName(kv.Value)
		case "Object":
			c.objectVar = identName(kv.Value)
		case "Prov":
			hasProv = true
		}
	}
	if c.subjectVar == "" || !hasProv {
		return claim{}, false
	}
	return c, true
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
		// Best-effort: concatenations like "foo" + "bar" are not flattened
		// here because rot-probe is read-only and the Brief lint already
		// flags split provenance quotes; if we miss one its rot-probe
		// finding will simply lack Brief context.
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

// unquote strips Go string literal quoting (raw or interpreted). The
// rot-probe doesn't need full strconv.Unquote correctness — readability is
// the only consumer.
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
