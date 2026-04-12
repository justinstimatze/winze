// Command lint runs winze's deterministic lint rules against the root
// package. The lint binary is a separate package main; package winze
// itself remains non-executable (dec-non-executable). The lint tool reads
// winze's source files via go/ast and imports winze as a library so it
// can read authoritative data like ExternalTerms without duplicating it.
//
// Rules implemented (v0):
//   1. naming-oracle      — role types must be grounded in ExternalTerms
//   2. orphan-report      — entity vars with zero claim references (advisory)
//   3. value-conflict     — functional-predicate value conflicts (advisory)
//   4. contested-concept  — multiple TheoryOf subjects per concept (advisory)
//   5. llm-contradiction  — LLM-detected semantic contradictions (--llm, advisory)
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

	"github.com/justinstimatze/winze"
)

type roleType struct {
	name string
	file string
	line int
}

func collectRoleTypes(dir string) ([]roleType, error) {
	fset := token.NewFileSet()
	var out []roleType

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
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
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				if embedsEntityPointer(st) {
					pos := fset.Position(ts.Pos())
					out = append(out, roleType{
						name: ts.Name.Name,
						file: filepath.Base(pos.Filename),
						line: pos.Line,
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func embedsEntityPointer(st *ast.StructType) bool {
	for _, field := range st.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		ident, ok := star.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "Entity" {
			return true
		}
	}
	return false
}

// namingOracleRule reports role types whose names do not appear in
// winze.ExternalTerms. Exits nonzero when any role is ungrounded.
func namingOracleRule(dir string) int {
	roles, err := collectRoleTypes(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "naming-oracle: %v\n", err)
		return 2
	}

	known := map[string]string{}
	for _, t := range winze.ExternalTerms {
		known[t.Name] = t.Source
	}

	var grounded, ungrounded []roleType
	for _, r := range roles {
		if _, ok := known[r.name]; ok {
			grounded = append(grounded, r)
		} else {
			ungrounded = append(ungrounded, r)
		}
	}

	fmt.Printf("[naming-oracle] %d role types, %d grounded, %d ungrounded\n",
		len(roles), len(grounded), len(ungrounded))

	if len(grounded) > 0 {
		fmt.Println("  grounded:")
		for _, r := range grounded {
			fmt.Printf("    %-16s %s:%d   (%s)\n", r.name, r.file, r.line, known[r.name])
		}
	}
	if len(ungrounded) > 0 {
		fmt.Println("  ungrounded — rename or add to ExternalTerms:")
		for _, r := range ungrounded {
			fmt.Printf("    %-16s %s:%d\n", r.name, r.file, r.line)
		}
		return 1
	}
	return 0
}

// valueConflict is a single (predicate type, subject) group with two or
// more claims whose Object expressions differ. Detected by the
// value-conflict rule below — the first deterministic lint rule that
// catches semantic contradiction rather than structural breakage.
type valueConflict struct {
	predicateType string
	subject       string
	claims        []claimSite
}

type claimSite struct {
	name          string
	predicateType string
	subject       string
	object        string
	file          string
	line          int
}

// collectClaims walks every .go file in dir and returns every var
// declaration whose initializer is a composite literal of a named type
// with both Subject: and Object: keyed fields. That pattern uniquely
// identifies a BinaryRelation predicate instantiation under the current
// schema; predicates are named distinct types over BinaryRelation so
// their instantiations always spell Subject/Object explicitly.
//
// It also collects predicate-type pragmas (//winze:functional,
// //winze:disjoint, ...) from the doc comments of type declarations,
// so downstream rules can consult them without a second pass.
//
// The rule intentionally does NOT use go/types. Go/ast is enough to
// identify predicate claims by shape, and avoiding go/types keeps the
// lint binary cheap enough to run on every commit.
func collectClaims(dir string) ([]claimSite, map[claimKey][]claimSite, map[string]bool, map[string]bool, map[claimKey]string, error) {
	fset := token.NewFileSet()
	var all []claimSite
	groups := map[claimKey][]claimSite{}
	functional := map[string]bool{}
	contested := map[string]bool{}
	suppressed := map[claimKey]string{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gen.Tok == token.TYPE {
				isFunctional := hasPragma(gen.Doc, "winze:functional")
				isContested := hasPragma(gen.Doc, "winze:contested")
				if isFunctional || isContested {
					for _, spec := range gen.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							if isFunctional {
								functional[ts.Name.Name] = true
							}
							if isContested {
								contested[ts.Name.Name] = true
							}
						}
					}
				}
				continue
			}
			if gen.Tok != token.VAR {
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
					typeIdent, typeOK := cl.Type.(*ast.Ident)
					if typeOK && typeIdent.Name == "KnownDispute" {
						subj, pred, rationale, ok := extractKnownDispute(cl)
						if ok {
							suppressed[claimKey{pred, subj}] = rationale
						}
						continue
					}
					predType, subj, obj, ok := extractPredicateClaim(cl)
					if !ok {
						continue
					}
					pos := fset.Position(nameIdent.Pos())
					site := claimSite{
						name:          nameIdent.Name,
						predicateType: predType,
						subject:       subj,
						object:        obj,
						file:          filepath.Base(pos.Filename),
						line:          pos.Line,
					}
					all = append(all, site)
					groups[claimKey{predType, subj}] = append(groups[claimKey{predType, subj}], site)
				}
			}
		}
	}
	return all, groups, functional, contested, suppressed, nil
}

// collectUnarySubjects walks every .go file and returns the Subject
// field identifier from every composite literal that has a Subject
// keyed field but no Object keyed field (UnaryClaim shape). This
// captures references that orphan-report would otherwise miss; it is
// deliberately NOT merged into collectClaims because the value-conflict
// rule keys on (predicate, subject, object) triples and would be
// confused by rows missing an object.
func collectUnarySubjects(dir string) ([]string, error) {
	fset := token.NewFileSet()
	var out []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
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
				for i := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					var subject string
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
							subject = exprString(kv.Value)
						case "Object":
							haveObject = true
						}
					}
					if haveSubject && !haveObject && subject != "" {
						out = append(out, subject)
					}
				}
			}
		}
	}
	return out, nil
}

// extractKnownDispute inspects a KnownDispute composite literal and
// returns the SubjectRef identifier, PredicateType string value, and
// Rationale string value. Matches the go/ast shape of the literal used
// in the ingest; anything else gets ok=false and is silently ignored.
func extractKnownDispute(cl *ast.CompositeLit) (subject, predicate, rationale string, ok bool) {
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
		case "SubjectRef":
			subject = exprString(kv.Value)
		case "PredicateType":
			if bl, ok := kv.Value.(*ast.BasicLit); ok {
				predicate = strings.Trim(bl.Value, "\"")
			}
		case "Rationale":
			if bl, ok := kv.Value.(*ast.BasicLit); ok {
				rationale = strings.Trim(bl.Value, "\"")
			}
		}
	}
	if subject == "" || predicate == "" {
		return "", "", "", false
	}
	return subject, predicate, rationale, true
}

type claimKey struct {
	predicateType string
	subject       string
}

// hasPragma returns true if any line of the given doc comment group
// ends with the named pragma (e.g. "winze:functional"). The convention
// is //winze:<tag> alone on its own comment line.
func hasPragma(doc *ast.CommentGroup, pragma string) bool {
	if doc == nil {
		return false
	}
	want := "//" + pragma
	for _, c := range doc.List {
		if strings.TrimSpace(c.Text) == want {
			return true
		}
	}
	return false
}

// extractPredicateClaim inspects a composite literal and returns the
// predicate type name, subject identifier, and object expression string
// if the literal has both Subject: and Object: keyed fields. Returns
// ok=false for any literal that does not match the predicate claim shape.
func extractPredicateClaim(cl *ast.CompositeLit) (predType, subject, object string, ok bool) {
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
			subject = exprString(kv.Value)
		case "Object":
			haveObject = true
			object = exprString(kv.Value)
		}
	}
	if !haveSubject || !haveObject {
		return "", "", "", false
	}
	return typeIdent.Name, subject, object, true
}

// exprString renders an AST expression as a compact identifier form
// suitable for equality comparison across claim sites. It covers the
// expression shapes that currently appear in winze's ingest files:
// plain identifiers, &Ident, Ident.Field, and package.Ident. Richer
// shapes (function calls, struct literals nested inside claim values)
// fall through to a positional placeholder so they compare unequal to
// anything but themselves, which is the safe default.
func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	case *ast.UnaryExpr:
		return v.Op.String() + exprString(v.X)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	default:
		return fmt.Sprintf("<expr@%T>", e)
	}
}

// valueConflictRule flags cases where two or more claims share the same
// predicate type and subject identifier but have different object
// expressions. This is winze's first deterministic *semantic* lint rule
// — it catches the contradiction shape the Tunguska Lake Cheko age
// dispute surfaced, where the real disagreement lives in claim values
// rather than claim types.
//
// This is advisory: a flagged group may be an intentional record of a
// live dispute (as Lake Cheko is), in which case the correct response
// is an AuthorialPolicy marking the conflict as load-bearing, or a
// temporal split, or a reconciliation once the underlying question
// settles. The rule does not fail the build; it prints the groups so
// ingest workers can decide.
func valueConflictRule(dir string) int {
	_, groups, functional, _, suppressed, err := collectClaims(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "value-conflict: %v\n", err)
		return 2
	}

	var conflicts []valueConflict
	var recorded []valueConflict
	for k, sites := range groups {
		if !functional[k.predicateType] {
			continue
		}
		if len(sites) < 2 {
			continue
		}
		distinctObjects := map[string]bool{}
		for _, s := range sites {
			distinctObjects[s.object] = true
		}
		if len(distinctObjects) < 2 {
			continue
		}
		vc := valueConflict{
			predicateType: k.predicateType,
			subject:       k.subject,
			claims:        sites,
		}
		if _, ok := suppressed[k]; ok {
			recorded = append(recorded, vc)
		} else {
			conflicts = append(conflicts, vc)
		}
	}
	sortConflicts := func(xs []valueConflict) {
		sort.Slice(xs, func(i, j int) bool {
			if xs[i].predicateType != xs[j].predicateType {
				return xs[i].predicateType < xs[j].predicateType
			}
			return xs[i].subject < xs[j].subject
		})
	}
	sortConflicts(conflicts)
	sortConflicts(recorded)

	fmt.Printf("[value-conflict] %d functional predicates, %d KnownDispute annotations, %d unresolved conflicts, %d recorded disputes\n",
		len(functional), len(suppressed), len(conflicts), len(recorded))
	fmt.Println("  (only //winze:functional predicates are checked — most are legitimately one-to-many.")
	fmt.Println("   unresolved conflicts need a rename, a temporal split, or a KnownDispute annotation.")
	fmt.Println("   recorded disputes are load-bearing and shown for audit only.)")
	if len(conflicts) > 0 {
		fmt.Println("  unresolved:")
		for _, c := range conflicts {
			fmt.Printf("    %s (%s):\n", c.predicateType, c.subject)
			for _, s := range c.claims {
				fmt.Printf("      %-40s %s:%d   object=%s\n", s.name, s.file, s.line, s.object)
			}
		}
	}
	if len(recorded) > 0 {
		fmt.Println("  recorded disputes:")
		for _, c := range recorded {
			fmt.Printf("    %s (%s):\n", c.predicateType, c.subject)
			for _, s := range c.claims {
				fmt.Printf("      %-40s %s:%d   object=%s\n", s.name, s.file, s.line, s.object)
			}
		}
	}
	return 0
}

// entitySite is a var whose initializer is a composite literal of a
// known role type (Person{...}, Place{...}, Event{...}, etc).
// orphanReportRule flags these when nothing claims about them.
type entitySite struct {
	name     string
	roleType string
	file     string
	line     int
}

// collectEntityVars walks every .go file in dir and returns vars whose
// initializer is a composite literal of a type named in roleTypes. This
// is the go/ast equivalent of "find every instance of a role-typed
// entity" — what was previously answered by a defn query against the
// reference graph, now answered in-process without the defn dependency.
//
// Vars of plain *Entity (bootstrap.go self-tracking entities like Stope
// or Dolt) are intentionally NOT collected: those are project-description
// vars and are expected never to be claim-subjects, so including them
// would drown the useful signal.
func collectEntityVars(dir string, roleTypes map[string]bool) ([]entitySite, error) {
	fset := token.NewFileSet()
	var out []entitySite

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
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
					typeIdent, ok := cl.Type.(*ast.Ident)
					if !ok {
						continue
					}
					if !roleTypes[typeIdent.Name] {
						continue
					}
					pos := fset.Position(nameIdent.Pos())
					out = append(out, entitySite{
						name:     nameIdent.Name,
						roleType: typeIdent.Name,
						file:     filepath.Base(pos.Filename),
						line:     pos.Line,
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// orphanReportRule flags role-typed entity vars that no claim references
// as either Subject or Object. This is the pure go/ast replacement for
// the earlier defn-query version: walking the claim composite literals
// already done by collectClaims gives every subject/object expression,
// and set subtraction from the declared entity vars identifies orphans.
// Eliminating the defn shell-out removes the CLI/MCP lock collision and
// makes the lint binary fully self-contained.
func orphanReportRule(dir string) int {
	roles, err := collectRoleTypes(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orphan-report: %v\n", err)
		return 2
	}
	roleSet := map[string]bool{}
	for _, r := range roles {
		roleSet[r.name] = true
	}

	entities, err := collectEntityVars(dir, roleSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orphan-report: %v\n", err)
		return 2
	}

	_, groups, _, _, _, err := collectClaims(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orphan-report: %v\n", err)
		return 2
	}

	referenced := map[string]bool{}
	for k, sites := range groups {
		referenced[k.subject] = true
		for _, s := range sites {
			referenced[s.object] = true
		}
	}

	// Also collect Subject references from UnaryClaim instantiations,
	// which collectClaims skips (it requires both Subject and Object
	// keyed fields to identify a BinaryRelation claim). Without this
	// pass, any entity mentioned only in unary claims — like the
	// user.go style-claim seed — looks falsely orphaned.
	unarySubjects, err := collectUnarySubjects(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orphan-report: %v\n", err)
		return 2
	}
	for _, s := range unarySubjects {
		referenced[s] = true
	}

	var orphans []entitySite
	for _, e := range entities {
		if !referenced[e.name] {
			orphans = append(orphans, e)
		}
	}

	fmt.Printf("[orphan-report] %d role-typed entities declared, %d referenced by claims, %d orphaned\n",
		len(entities), len(referenced), len(orphans))
	fmt.Println("  (actionable: entities created during ingest that no claim mentions as Subject or Object.")
	fmt.Println("   wire them up with a claim, or delete them if the ingest was overzealous)")
	if len(orphans) > 0 {
		byRole := map[string][]entitySite{}
		for _, o := range orphans {
			byRole[o.roleType] = append(byRole[o.roleType], o)
		}
		roleNames := make([]string, 0, len(byRole))
		for r := range byRole {
			roleNames = append(roleNames, r)
		}
		sort.Strings(roleNames)
		for _, r := range roleNames {
			sites := byRole[r]
			fmt.Printf("  %s (%d):\n", r, len(sites))
			for _, s := range sites {
				fmt.Printf("    %-40s %s:%d\n", s.name, s.file, s.line)
			}
		}
	}
	return 0
}

// contestedConceptRule groups claims by (predicateType, object) for any
// predicate type marked with the //winze:contested pragma, counts the
// distinct subject identifiers pointing at each object, and emits an
// advisory report when a group has two or more distinct subjects. This
// is the mirror axis of value-conflict: value-conflict groups by
// (predicate, subject) and asks "are the objects in conflict?";
// contested-concept groups by (predicate, object) and asks "are
// multiple subjects theorising this target?"
//
// Unlike value-conflict, contested-concept is NOT a failure condition.
// Multiple theories of the same concept are a normal state of affairs
// wherever a field has live intellectual disagreement, and forcing
// their resolution would be a category error — winze's job is to
// surface the landscape of disagreement, not to adjudicate. The rule
// returns 0 always; its output is an information surface for ingest
// workers and for downstream queries that want to find contested
// targets without a full graph walk.
//
// The rule was motivated by the Nondualism ingest, where three authors
// (Murti, Loy, Volker) proposed incompatible typologies of the same
// concept via TheoryOf. The pattern was structurally visible in the
// defn reference graph before any dedicated rule existed; landing the
// rule closes the gap between "visible via ad-hoc query" and "surfaced
// automatically at every lint run."
func contestedConceptRule(dir string) int {
	all, _, _, contested, _, err := collectClaims(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "contested-concept: %v\n", err)
		return 2
	}

	type contestedGroup struct {
		predicateType string
		object        string
		claims        []claimSite
	}

	byKey := map[claimKey][]claimSite{}
	for _, s := range all {
		if !contested[s.predicateType] {
			continue
		}
		k := claimKey{s.predicateType, s.object}
		byKey[k] = append(byKey[k], s)
	}

	var groups []contestedGroup
	for k, sites := range byKey {
		distinctSubjects := map[string]bool{}
		for _, s := range sites {
			distinctSubjects[s.subject] = true
		}
		if len(distinctSubjects) < 2 {
			continue
		}
		groups = append(groups, contestedGroup{
			predicateType: k.predicateType,
			object:        k.subject,
			claims:        sites,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].predicateType != groups[j].predicateType {
			return groups[i].predicateType < groups[j].predicateType
		}
		return groups[i].object < groups[j].object
	})

	fmt.Printf("[contested-concept] %d contested predicates, %d contested targets\n",
		len(contested), len(groups))
	fmt.Println("  (advisory: //winze:contested predicates with two or more distinct subjects per object.")
	fmt.Println("   multiple theories of the same target are a normal state of affairs; this is an")
	fmt.Println("   information surface, not a failure. unresolved disagreement is the point.)")
	for _, g := range groups {
		sort.Slice(g.claims, func(i, j int) bool {
			return g.claims[i].subject < g.claims[j].subject
		})
		fmt.Printf("    %s (%s): %d subjects\n", g.predicateType, g.object, len(uniqueSubjects(g.claims)))
		for _, s := range g.claims {
			fmt.Printf("      %-40s %s:%d   subject=%s\n", s.name, s.file, s.line, s.subject)
		}
	}
	return 0
}

func uniqueSubjects(sites []claimSite) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range sites {
		if seen[s.subject] {
			continue
		}
		seen[s.subject] = true
		out = append(out, s.subject)
	}
	return out
}

func main() {
	llmFlag := false
	llmModel := "haiku"
	llmMaxCalls := 0
	llmMaxTokens := 1024

	dir := "."
	var args []string
	for _, arg := range os.Args[1:] {
		switch {
		case arg == "--llm":
			llmFlag = true
		case strings.HasPrefix(arg, "--llm-model="):
			llmModel = strings.TrimPrefix(arg, "--llm-model=")
		case strings.HasPrefix(arg, "--llm-max-calls="):
			if n, err := fmt.Sscanf(arg, "--llm-max-calls=%d", &llmMaxCalls); n != 1 || err != nil {
				llmMaxCalls = 0
			}
		case strings.HasPrefix(arg, "--llm-max-tokens="):
			if n, err := fmt.Sscanf(arg, "--llm-max-tokens=%d", &llmMaxTokens); n != 1 || err != nil {
				llmMaxTokens = 1024
			}
		default:
			args = append(args, arg)
		}
	}
	if len(args) > 0 {
		dir = args[0]
	}

	rc1 := namingOracleRule(dir)
	fmt.Println()
	rc2 := orphanReportRule(dir)
	fmt.Println()
	rc3 := valueConflictRule(dir)
	fmt.Println()
	rc4 := contestedConceptRule(dir)
	fmt.Println()
	rc5 := llmContradictionRule(dir, llmBudget{
		enabled:        llmFlag,
		model:          llmModel,
		maxCallsPerRun: llmMaxCalls,
		maxTokens:      llmMaxTokens,
	})

	worst := rc1
	for _, rc := range []int{rc2, rc3, rc4, rc5} {
		if rc > worst {
			worst = rc
		}
	}
	os.Exit(worst)
}
