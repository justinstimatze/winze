package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ast retriever: hand-written go/ast queries that represent the ceiling
// of what typed storage enables. Each question maps to a function that
// walks the source files and returns exact answers.

func astRetrieve(dir string, q Question) []string {
	switch q.ID {
	// Lexical
	case "lex-01":
		return astFindEntitiesByTypeAndFile(dir, "Person", "hard_problem.go", "Chalmers")
	case "lex-02":
		return astFindEntitiesByTypeAndFile(dir, "Person", "godel_incompleteness.go", "Godel")
	case "lex-03":
		return astFindEntityByName(dir, "Apophenia")
	case "lex-04":
		return astFindEntitiesByTypeAndFile(dir, "Person", "quantum_thief.go", "Rajaniemi")
	case "lex-05":
		return astFindEntitiesByTypeAndFile(dir, "Person", "demon_haunted.go", "Sagan")
	case "lex-06":
		return astFindEntityByName(dir, "Kulik")

	// Aggregation
	case "agg-01":
		claims := astCollectClaimsByPredicate(dir, "HypothesisExplains")
		return []string{fmt.Sprintf("%d", len(claims))}
	case "agg-02":
		return astCollectUnarySubjectsByPredicate(dir, "IsCognitiveBias")
	case "agg-03":
		entities := astCollectEntitiesByRole(dir, "Person")
		return []string{fmt.Sprintf("%d", len(entities))}
	case "agg-04":
		claims := astCollectClaimsByPredicate(dir, "TheoryOf")
		return []string{fmt.Sprintf("%d", len(claims))}
	case "agg-05":
		return astCollectClaimVarsByPredicate(dir, "InfluencedBy")
	case "agg-06":
		return astCollectClaimVarsByPredicate(dir, "Disputes")

	// Multi-hop
	case "hop-01":
		return astTheoryOfTarget(dir, "Consciousness")
	case "hop-02":
		return astInfluencedBySubject(dir, "HannuRajaniemi")
	case "hop-03":
		return astDisputersOfHypothesesExplaining(dir, "TunguskaEvent")
	case "hop-04":
		return astFictionalEntitiesInFile(dir, "quantum_thief.go")
	case "hop-05":
		return astProposersOfTheoriesAbout(dir, "MathematicalFoundations")
	case "hop-06":
		return astInfluencedBySubject(dir, "StevenPinker")

	// Contested
	case "con-01":
		return astContestedTargetsWithMinSubjects(dir, 3)
	case "con-02":
		return astContestedTargetsWithMinSubjects(dir, 2)
	case "con-03":
		count := astCountKnownDisputes(dir)
		return []string{fmt.Sprintf("%d", count)}
	case "con-04":
		return astTheoryOfSubjects(dir, "Schizophrenia")
	case "con-05":
		return astTheoryOfSubjects(dir, "Apophenia")
	case "con-06":
		return astCollectUnarySubjectsByPredicate(dir, "IsPolyvalentTerm")

	default:
		return nil
	}
}

// --- helpers (ported from cmd/lint/main.go) ---

type claimInfo struct {
	varName       string
	predicateType string
	subject       string
	object        string
}

func astExprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.UnaryExpr:
		return v.Op.String() + astExprString(v.X)
	case *ast.SelectorExpr:
		return astExprString(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + astExprString(v.X)
	default:
		return fmt.Sprintf("<expr@%T>", e)
	}
}

func astExtractClaim(cl *ast.CompositeLit) (predType, subject, object string, ok bool) {
	typeIdent, typeOK := cl.Type.(*ast.Ident)
	if !typeOK {
		return "", "", "", false
	}
	var haveSubject, haveObject bool
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
			haveSubject = true
			subject = astExprString(kv.Value)
		case "Object":
			haveObject = true
			object = astExprString(kv.Value)
		}
	}
	if !haveSubject || !haveObject {
		return "", "", "", false
	}
	return typeIdent.Name, subject, object, true
}

func walkVarDecls(dir string, fn func(name string, cl *ast.CompositeLit)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
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
					fn(nameIdent.Name, cl)
				}
			}
		}
	}
	return nil
}

func walkVarDeclsWithFile(dir string, fn func(name string, cl *ast.CompositeLit, file string)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
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
					fn(nameIdent.Name, cl, e.Name())
				}
			}
		}
	}
	return nil
}

// --- query implementations ---

func astCollectAllClaims(dir string) []claimInfo {
	var out []claimInfo
	_ = walkVarDecls(dir, func(name string, cl *ast.CompositeLit) {
		pred, subj, obj, ok := astExtractClaim(cl)
		if ok {
			out = append(out, claimInfo{varName: name, predicateType: pred, subject: subj, object: obj})
		}
	})
	return out
}

func astCollectClaimsByPredicate(dir string, predicate string) []claimInfo {
	all := astCollectAllClaims(dir)
	var out []claimInfo
	for _, c := range all {
		if c.predicateType == predicate {
			out = append(out, c)
		}
	}
	return out
}

func astCollectClaimVarsByPredicate(dir string, predicate string) []string {
	claims := astCollectClaimsByPredicate(dir, predicate)
	var out []string
	for _, c := range claims {
		out = append(out, c.varName)
	}
	sort.Strings(out)
	return out
}

func astCollectEntitiesByRole(dir string, role string) []string {
	var out []string
	_ = walkVarDecls(dir, func(name string, cl *ast.CompositeLit) {
		typeIdent, ok := cl.Type.(*ast.Ident)
		if ok && typeIdent.Name == role {
			out = append(out, name)
		}
	})
	sort.Strings(out)
	return out
}

func astCollectUnarySubjectsByPredicate(dir string, predicate string) []string {
	var out []string
	_ = walkVarDecls(dir, func(name string, cl *ast.CompositeLit) {
		typeIdent, ok := cl.Type.(*ast.Ident)
		if !ok || typeIdent.Name != predicate {
			return
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if key.Name == "Subject" {
				out = append(out, astExprString(kv.Value))
			}
		}
	})
	sort.Strings(out)
	return out
}

func astFindEntitiesByTypeAndFile(dir string, roleType, file, nameContains string) []string {
	var out []string
	_ = walkVarDeclsWithFile(dir, func(name string, cl *ast.CompositeLit, f string) {
		if f != file {
			return
		}
		typeIdent, ok := cl.Type.(*ast.Ident)
		if !ok || typeIdent.Name != roleType {
			return
		}
		if strings.Contains(name, nameContains) {
			out = append(out, name)
		}
	})
	return out
}

func astFindEntityByName(dir string, target string) []string {
	var out []string
	_ = walkVarDecls(dir, func(name string, cl *ast.CompositeLit) {
		if name == target {
			out = append(out, name)
		}
	})
	return out
}

func astTheoryOfTarget(dir string, target string) []string {
	claims := astCollectClaimsByPredicate(dir, "TheoryOf")
	var out []string
	for _, c := range claims {
		if c.object == target {
			out = append(out, c.varName)
		}
	}
	sort.Strings(out)
	return out
}

func astTheoryOfSubjects(dir string, target string) []string {
	claims := astCollectClaimsByPredicate(dir, "TheoryOf")
	var out []string
	for _, c := range claims {
		if c.object == target {
			out = append(out, c.subject)
		}
	}
	sort.Strings(out)
	return out
}

func astInfluencedBySubject(dir string, person string) []string {
	claims := astCollectClaimsByPredicate(dir, "InfluencedBy")
	var out []string
	for _, c := range claims {
		if c.subject == person {
			out = append(out, c.object)
		}
	}
	sort.Strings(out)
	return out
}

func astDisputersOfHypothesesExplaining(dir string, event string) []string {
	explains := astCollectClaimsByPredicate(dir, "HypothesisExplains")
	hypotheses := map[string]bool{}
	for _, c := range explains {
		if c.object == event {
			hypotheses[c.subject] = true
		}
	}

	disputes := astCollectClaimsByPredicate(dir, "Disputes")
	disputers := map[string]bool{}
	for _, c := range disputes {
		if hypotheses[c.object] {
			disputers[c.subject] = true
		}
	}

	var out []string
	for p := range disputers {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func astFictionalEntitiesInFile(dir string, targetFile string) []string {
	fictional := map[string]bool{}
	_ = walkVarDeclsWithFile(dir, func(name string, cl *ast.CompositeLit, file string) {
		if file != targetFile {
			return
		}
		typeIdent, ok := cl.Type.(*ast.Ident)
		if !ok || typeIdent.Name != "IsFictional" {
			return
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if key.Name == "Subject" {
				fictional[astExprString(kv.Value)] = true
			}
		}
	})

	var out []string
	for e := range fictional {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

func astProposersOfTheoriesAbout(dir string, target string) []string {
	theories := astCollectClaimsByPredicate(dir, "TheoryOf")
	theorySubjects := map[string]bool{}
	for _, c := range theories {
		if c.object == target {
			theorySubjects[c.subject] = true
		}
	}

	proposes := astCollectClaimsByPredicate(dir, "Proposes")
	proposers := map[string]bool{}
	for _, c := range proposes {
		if theorySubjects[c.object] {
			proposers[c.subject] = true
		}
	}

	var out []string
	for p := range proposers {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func astContestedTargetsWithMinSubjects(dir string, minSubjects int) []string {
	claims := astCollectClaimsByPredicate(dir, "TheoryOf")
	targetSubjects := map[string]map[string]bool{}
	for _, c := range claims {
		if targetSubjects[c.object] == nil {
			targetSubjects[c.object] = map[string]bool{}
		}
		targetSubjects[c.object][c.subject] = true
	}

	var out []string
	for target, subjects := range targetSubjects {
		if len(subjects) >= minSubjects {
			out = append(out, target)
		}
	}
	sort.Strings(out)
	return out
}

func astCountKnownDisputes(dir string) int {
	count := 0
	_ = walkVarDecls(dir, func(name string, cl *ast.CompositeLit) {
		typeIdent, ok := cl.Type.(*ast.Ident)
		if ok && typeIdent.Name == "KnownDispute" {
			count++
		}
	})
	return count
}
