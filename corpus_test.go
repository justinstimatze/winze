// Package winze tests treat the KB's own source code as a dataset.
//
// The same .go files that go build type-checks are also walked by go/ast
// to verify invariants the type system cannot express. The KB tests itself:
// a non-executable knowledge base where the test suite asserts properties
// about the source code that IS the knowledge.
//
// These tests run against the REAL corpus — no fixtures, no mocks. If an
// entity in tunguska.go is missing a Brief, TestEntityBriefCompleteness
// fails. If two entities in different files share an ID,
// TestEntityIDUniqueness fails. The test suite is a superset of go build
// as a consistency checker.
package winze

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justinstimatze/winze/internal/astutil"
)

// roleTypes are the role wrapper types that mark entities.
// A-shape roles (concrete entity-relation) and B-shape roles (design-doc
// authorial claims) are both validated — they share the Entity atom.
var testRoleTypes = map[string]bool{
	// A-shape roles (roles.go)
	"Person": true, "Organization": true, "Concept": true,
	"Hypothesis": true, "Event": true, "Place": true,
	"Instrument": true, "Facility": true, "Substance": true,
	// B-shape roles (design_roles.go) — provisioned for creative-work
	// ingests (--pkm). No corpus entities currently use these types.
	// Included so that when B-shape entities ARE ingested, they're
	// immediately validated for Brief completeness, ID uniqueness,
	// and orphan detection.
	"CreativeWork": true, "DesignLayer": true, "Phase": true,
	"ProtectedLine": true, "NeverAnswered": true,
	"AuthorialPolicy": true, "Reading": true,
}

// claimPredicates are predicate types that reference entities as Subject/Object.
var testClaimPredicates = map[string]bool{
	"Proposes": true, "Disputes": true, "ProposesOrg": true, "DisputesOrg": true,
	"Accepts": true, "AcceptsOrg": true, "EarlyFormulationOf": true,
	"TheoryOf": true, "HypothesisExplains": true,
	"BelongsTo": true, "DerivedFrom": true, "InfluencedBy": true,
	"LocatedIn": true, "LocatedNear": true, "OccurredAt": true,
	"WorksFor": true, "AffiliatedWith": true, "InvestigatedBy": true,
	"Authored": true, "AuthoredOrg": true, "CommentaryOn": true,
	"AppearsIn": true, "LedExpedition": true, "FundedBy": true,
	"Predicts": true, "ResolvedAs": true, "Credence": true,
	"FormedAt": true, "EnergyEstimate": true, "EnglishTranslationOf": true,
	"IsCognitiveBias": true, "IsPolyvalentTerm": true,
	"IsFictionalWork": true, "IsFictional": true,
	"CorrectsCommonMisconception": true,
	"GrantsBroadAuthorityOverWinze": true, "PrefersTerseResponses": true,
	"PushesBackOnOverengineering": true, "PrefersOrganicSchemaGrowth": true,
	// Main-schema predicates with no current corpus usage (Tunguska-domain)
	"CausedEvent": true, "Contaminates": true, "HoldsContractWith": true,
	"MonitoredBy": true, "MonitoredByOrg": true, "Operates": true,
	"Released": true, "RunsFacility": true, "ShipsSamplesTo": true,
	// Design-shape predicates (private corpus slices)
	"AppliesToWork": true, "WorkHasLayer": true, "WorkHasPhase": true,
	"WorkHasProtectedLine": true, "WorkCommitsToNeverAnswering": true,
	"LineHasReadingAtLayer": true, "ReadingAtLayer": true, "ReadingAtPhase": true,
}

// corpusDir returns the path to the corpus root (this package's directory).
func corpusDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return dir
}

// walkCorpusFiles parses all non-test .go files in the corpus root.
func walkCorpusFiles(t *testing.T, dir string) map[string]*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		files[e.Name()] = f
	}
	return files
}

// extractStringField extracts a string literal field value from a composite literal.
func extractStringField(cl *ast.CompositeLit, fieldName string) string {
	return astutil.ExtractStringField(cl, fieldName)
}

// TestEntityBriefCompleteness verifies every entity has a non-empty Brief.
// The type system enforces that Brief is a string field, but not that it's
// populated. An entity without a Brief is a knowledge atom with no description —
// technically valid Go but epistemically empty.
func TestEntityBriefCompleteness(t *testing.T) {
	dir := corpusDir(t)
	files := walkCorpusFiles(t, dir)

	for name, f := range files {
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
					typeName := compositeTypeName(cl)
					if !testRoleTypes[typeName] {
						continue
					}
					brief := extractStringField(cl, "Brief")
					if brief == "" {
						t.Errorf("%s: entity %s (%s) has empty Brief", name, nameIdent.Name, typeName)
					}
				}
			}
		}
	}
}

// TestEntityIDUniqueness verifies no two entities share the same ID.
// The type system doesn't prevent two entities from having ID: "consciousness" —
// only the test suite catches this copy-paste error across files.
func TestEntityIDUniqueness(t *testing.T) {
	dir := corpusDir(t)
	files := walkCorpusFiles(t, dir)

	seen := map[string]string{} // ID → "file:varName"
	for name, f := range files {
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
					typeName := compositeTypeName(cl)
					if !testRoleTypes[typeName] {
						continue
					}
					id := extractStringField(cl, "ID")
					if id == "" {
						continue
					}
					loc := name + ":" + nameIdent.Name
					if prev, exists := seen[id]; exists {
						t.Errorf("duplicate entity ID %q: %s and %s", id, prev, loc)
					}
					seen[id] = loc
				}
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("no entity IDs found — test infrastructure may be broken")
	}
	t.Logf("checked %d unique entity IDs", len(seen))
}

// TestProvenanceCompleteness verifies every Provenance var has non-empty Origin and Quote.
// A provenance without Origin is an untraceable claim. A provenance without Quote
// is an unverifiable claim. Both violate the mirror-source-commitments principle.
func TestProvenanceCompleteness(t *testing.T) {
	dir := corpusDir(t)
	files := walkCorpusFiles(t, dir)

	count := 0
	for name, f := range files {
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
					typeName := compositeTypeName(cl)
					if typeName != "Provenance" {
						continue
					}
					count++
					origin := extractStringField(cl, "Origin")
					quote := extractStringField(cl, "Quote")
					if origin == "" {
						t.Errorf("%s: provenance %s has empty Origin", name, nameIdent.Name)
					}
					if quote == "" {
						ingestedAt := extractStringField(cl, "IngestedAt")
						// Quote became mandatory with metabolism cycle 6 (QuoteMandateDate).
						// Earlier provenance vars were created before the requirement
						// existed, and some use string concatenation for Quote which
						// extractStringField can't parse. Enforce only for post-mandate.
						if ingestedAt >= QuoteMandateDate {
							t.Errorf("%s: provenance %s has empty Quote (IngestedAt %s, post-mandate)", name, nameIdent.Name, ingestedAt)
						} else {
							t.Logf("%s: provenance %s has empty Quote (advisory, pre-mandate)", name, nameIdent.Name)
						}
					}
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("no Provenance vars found — test infrastructure may be broken")
	}
	t.Logf("checked %d provenance records", count)
}

// TestNoOrphanEntities verifies every entity appears as Subject or Object in at least
// one claim. An orphan entity is a knowledge atom that exists but isn't connected to
// anything — it compiles but contributes nothing to the knowledge graph.
func TestNoOrphanEntities(t *testing.T) {
	dir := corpusDir(t)
	files := walkCorpusFiles(t, dir)

	// Known exceptions: bootstrap entities that describe the project itself
	// (not external knowledge claims).
	exceptions := map[string]bool{
		"Winze": true, "Defn": true, "Dolt": true, "GasTown": true,
		"ClaudeCode": true, "CursorAI": true,
		// Kirk/Campbell/Nagel wired via EarlyFormulationOf.
		// RCZaehner and WalterTerenceStace removed (mirror-source violation).
	}

	// Collect all entity var names
	entities := map[string]string{} // var name → file
	for name, f := range files {
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
					typeName := compositeTypeName(cl)
					if testRoleTypes[typeName] {
						entities[nameIdent.Name] = name
					}
				}
			}
		}
	}

	// Collect all entity references in claims (Subject/Object fields)
	referenced := map[string]bool{}
	for _, f := range files {
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
					typeName := compositeTypeName(cl)
					if !testClaimPredicates[typeName] {
						continue
					}
					// Extract Subject and Object ident references
					for _, elt := range cl.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok {
							continue
						}
						if key.Name == "Subject" || key.Name == "Object" {
							if ident, ok := kv.Value.(*ast.Ident); ok {
								referenced[ident.Name] = true
							}
						}
					}
				}
			}
		}
	}

	orphans := 0
	for entity, file := range entities {
		if exceptions[entity] {
			continue
		}
		if !referenced[entity] {
			t.Errorf("%s: entity %s is an orphan (not referenced by any claim)", file, entity)
			orphans++
		}
	}
	t.Logf("checked %d entities, %d orphans found", len(entities), orphans)
}

// TestClaimReferencesExist is the inverse of TestNoOrphanEntities: it verifies
// that every Subject and Object in a claim refers to a var that actually exists
// in the corpus. A claim referencing a deleted entity compiles (the var is
// just unused) but is structurally broken — a dangling edge in the knowledge graph.
func TestClaimReferencesExist(t *testing.T) {
	dir := corpusDir(t)
	files := walkCorpusFiles(t, dir)

	// Collect all top-level var names (not just entities — claims can reference
	// hypotheses, events, concepts, etc.)
	allVars := map[string]bool{}
	for _, f := range files {
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
				for _, nameIdent := range vs.Names {
					allVars[nameIdent.Name] = true
				}
			}
		}
	}

	// Check every claim's Subject/Object references resolve to existing vars
	dangling := 0
	for name, f := range files {
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
					typeName := compositeTypeName(cl)
					if !testClaimPredicates[typeName] {
						continue
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
						if key.Name == "Subject" || key.Name == "Object" {
							if ident, ok := kv.Value.(*ast.Ident); ok {
								if !allVars[ident.Name] {
									t.Errorf("%s: claim %s.%s references non-existent var %s",
										name, nameIdent.Name, key.Name, ident.Name)
									dangling++
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("checked claim references, %d dangling found", dangling)
}

// TestCorpusCompiles runs go build ./... and verifies the knowledge base compiles.
// This is redundant with running go build directly, but having it as an explicit
// test documents the invariant: compilation IS consistency checking for this KB.
func TestCorpusCompiles(t *testing.T) {
	dir := corpusDir(t)
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./... failed:\n%s", out)
	}
}

func compositeTypeName(cl *ast.CompositeLit) string {
	return astutil.CompositeTypeName(cl)
}
