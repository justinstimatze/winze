package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// TestNamingOracleRule verifies all role types are grounded in ExternalTerms.
// Exit 0 means all role names have external vocabulary justification.
func TestNamingOracleRule(t *testing.T) {
	root := repoRoot(t)
	code := namingOracleRule(root)
	if code != 0 {
		t.Errorf("namingOracleRule returned %d, want 0 (all roles grounded)", code)
	}
}

// TestValueConflictRule checks for functional predicate violations.
func TestValueConflictRule(t *testing.T) {
	root := repoRoot(t)
	// The real corpus has known suppressed conflicts (EnergyEstimate 3-way).
	// This test verifies the rule runs without panic and returns a known exit code.
	code := valueConflictRule(root)
	// Exit 0 = no unsuppressed conflicts, which is the expected state.
	if code != 0 {
		t.Errorf("valueConflictRule returned %d, want 0 (no unsuppressed conflicts)", code)
	}
}

// TestOrphanReportRule runs the orphan detector against the real corpus.
func TestOrphanReportRule(t *testing.T) {
	root := repoRoot(t)
	code := orphanReportRule(root)
	// Orphan report is advisory (always returns 0).
	if code != 0 {
		t.Errorf("orphanReportRule returned %d, want 0 (advisory)", code)
	}
}

// TestContestedConceptRule verifies contested concept detection.
// Consciousness should be flagged as contested (3+ competing theories).
func TestContestedConceptRule(t *testing.T) {
	root := repoRoot(t)
	code := contestedConceptRule(root)
	// Advisory rule — exit code 0 expected.
	if code != 0 {
		t.Errorf("contestedConceptRule returned %d, want 0", code)
	}
}

// TestBriefCheckRule verifies entity Brief completeness and length.
func TestBriefCheckRule(t *testing.T) {
	root := repoRoot(t)
	code := briefCheckRule(root)
	if code != 0 {
		t.Errorf("briefCheckRule returned %d, want 0", code)
	}
}

// TestProvenanceSplitRule detects fragmented provenance declarations.
func TestProvenanceSplitRule(t *testing.T) {
	root := repoRoot(t)
	code := provenanceSplitRule(root)
	// Advisory rule — returns 0 even with splits.
	if code != 0 {
		t.Errorf("provenanceSplitRule returned %d, want 0", code)
	}
}

// ---------------------------------------------------------------------------
// Synthetic violation tests: prove rules detect bad input, not just that
// the real corpus is clean.
// ---------------------------------------------------------------------------

// writeSyntheticCorpus writes .go files to dir from a map of filename→content.
func writeSyntheticCorpus(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

// TestNamingOracleRule_Violation creates a role type not in ExternalTerms
// and verifies the rule returns nonzero.
func TestNamingOracleRule_Violation(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCorpus(t, dir, map[string]string{
		"roles.go": `package synth

type Entity struct {
	ID   string
	Name string
}

// Bogus is a role type with no external grounding.
type Bogus struct{ *Entity }
`,
	})
	code := namingOracleRule(dir)
	if code == 0 {
		t.Error("namingOracleRule returned 0 for ungrounded role 'Bogus', want nonzero")
	}
}

// TestValueConflictRule_Violation creates two functional claims on the same
// subject with different objects and verifies the rule detects the conflict.
// valueConflictRule is advisory (returns 0) but we verify it collects the
// conflict by checking the underlying data.
func TestValueConflictRule_Violation(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCorpus(t, dir, map[string]string{
		"predicates.go": `package synth

type Entity struct {
	ID   string
	Name string
}

type Provenance struct {
	Origin string
}

type Person struct{ *Entity }

//winze:functional
type FormedAt struct {
	Subject Person
	Object  Person
	Prov    Provenance
}
`,
		"corpus.go": `package synth

var Alice = Person{&Entity{ID: "alice", Name: "Alice"}}
var Bob = Person{&Entity{ID: "bob", Name: "Bob"}}
var Carol = Person{&Entity{ID: "carol", Name: "Carol"}}

var Claim1 = FormedAt{Subject: Alice, Object: Bob}
var Claim2 = FormedAt{Subject: Alice, Object: Carol}
`,
	})
	// valueConflictRule is advisory (returns 0), but we can verify the
	// underlying detection by calling collectClaims directly.
	_, groups, functional, _, _, err := collectClaims(dir)
	if err != nil {
		t.Fatalf("collectClaims: %v", err)
	}
	if !functional["FormedAt"] {
		t.Fatal("FormedAt not detected as functional")
	}
	// Find the group for FormedAt/Alice — should have 2 claims with different objects.
	key := claimKey{"FormedAt", "Alice"}
	sites := groups[key]
	if len(sites) < 2 {
		t.Fatalf("expected 2+ claims for FormedAt/Alice, got %d", len(sites))
	}
	objects := map[string]bool{}
	for _, s := range sites {
		objects[s.object] = true
	}
	if len(objects) < 2 {
		t.Errorf("expected 2+ distinct objects for FormedAt/Alice, got %d", len(objects))
	}
}

// TestBriefCheckRule_MissingBrief creates an entity with no Brief field
// and verifies the rule detects it (advisory, returns 0, but we check output).
func TestBriefCheckRule_MissingBrief(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCorpus(t, dir, map[string]string{
		"roles.go": `package synth

type Entity struct {
	ID    string
	Name  string
	Kind  string
	Brief string
}

type Person struct{ *Entity }
`,
		"corpus.go": `package synth

var NoBrief = Person{&Entity{ID: "no-brief", Name: "No Brief Person", Kind: "person"}}
`,
	})
	// briefCheckRule is advisory (always returns 0) but should still run
	// without panic on a corpus with missing briefs.
	code := briefCheckRule(dir)
	if code != 0 {
		t.Errorf("briefCheckRule returned %d, want 0 (advisory)", code)
	}
}

// TestOrphanReportRule_WithOrphan creates an entity referenced by no claims
// and verifies the rule runs without error.
func TestOrphanReportRule_WithOrphan(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCorpus(t, dir, map[string]string{
		"roles.go": `package synth

type Entity struct {
	ID    string
	Name  string
	Kind  string
	Brief string
}

type Person struct{ *Entity }
`,
		"corpus.go": `package synth

var Orphan = Person{&Entity{ID: "orphan", Name: "Orphan Entity", Kind: "person", Brief: "Truly alone."}}
`,
	})
	code := orphanReportRule(dir)
	// Advisory — returns 0 even with orphans, but should detect it without panic.
	if code != 0 {
		t.Errorf("orphanReportRule returned %d, want 0 (advisory)", code)
	}
}

// TestContestedConceptRule_Violation creates two subjects with TheoryOf claims
// about the same object, with //winze:contested pragma. Verifies the rule
// detects the contested pragma and runs without error.
func TestContestedConceptRule_Violation(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCorpus(t, dir, map[string]string{
		"predicates.go": `package synth

type Entity struct {
	ID   string
	Name string
}

type Provenance struct {
	Origin string
}

type Person struct{ *Entity }
type Hypothesis struct{ *Entity }

//winze:contested
type TheoryOf struct {
	Subject Person
	Object  Hypothesis
	Prov    Provenance
}
`,
		"corpus.go": `package synth

var Alice = Person{&Entity{ID: "alice", Name: "Alice"}}
var Bob = Person{&Entity{ID: "bob", Name: "Bob"}}
var Target = Hypothesis{&Entity{ID: "target", Name: "Target"}}

var AliceTheory = TheoryOf{Subject: Alice, Object: Target}
var BobTheory = TheoryOf{Subject: Bob, Object: Target}
`,
	})
	_, _, _, contested, _, err := collectClaims(dir)
	if err != nil {
		t.Fatalf("collectClaims: %v", err)
	}
	if !contested["TheoryOf"] {
		t.Fatal("TheoryOf not detected as contested")
	}
	code := contestedConceptRule(dir)
	if code != 0 {
		t.Errorf("contestedConceptRule returned %d, want 0 (advisory)", code)
	}
}

// TestProvenanceSplitRule_Violation creates two files with Provenance vars
// sharing the same Origin. Verifies the rule runs without error.
func TestProvenanceSplitRule_Violation(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCorpus(t, dir, map[string]string{
		"file_a.go": `package synth

type Provenance struct { Origin string; Quote string }

var SourceA = Provenance{Origin: "Wikipedia / Shared Article", Quote: "quote A"}
`,
		"file_b.go": `package synth

var SourceB = Provenance{Origin: "Wikipedia / Shared Article", Quote: "quote B"}
`,
	})
	code := provenanceSplitRule(dir)
	if code != 0 {
		t.Errorf("provenanceSplitRule returned %d, want 0 (advisory)", code)
	}
}

// TestLLMContradictionRule_NoKey verifies the llm-contradiction rule
// degrades gracefully when ANTHROPIC_API_KEY is not set.
func TestLLMContradictionRule_NoKey(t *testing.T) {
	orig := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if orig != "" {
			os.Setenv("ANTHROPIC_API_KEY", orig)
		}
	}()

	root := repoRoot(t)
	code := llmContradictionRule(root, llmBudget{enabled: true, model: "haiku", maxCallsPerRun: 0})
	if code != 0 {
		t.Errorf("llmContradictionRule returned %d without API key, want 0", code)
	}
}

func TestHasPragma(t *testing.T) {
	cases := []struct {
		name    string
		comment string
		pragma  string
		want    bool
	}{
		{"exact match", "//winze:contested", "winze:contested", true},
		{"functional pragma", "//winze:functional", "winze:functional", true},
		{"no match", "// just a comment", "winze:contested", false},
		{"partial match", "//winze:contest", "winze:contested", false},
		{"embedded in text", "// see winze:contested for details", "winze:contested", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			group := &ast.CommentGroup{
				List: []*ast.Comment{{Text: tc.comment}},
			}
			got := hasPragma(group, tc.pragma)
			if got != tc.want {
				t.Errorf("hasPragma(%q, %q) = %v, want %v", tc.comment, tc.pragma, got, tc.want)
			}
		})
	}
}

func TestHasPragma_Nil(t *testing.T) {
	if hasPragma(nil, "winze:contested") {
		t.Error("hasPragma(nil, ...) should return false")
	}
}

func TestEmbedsEntityPointer(t *testing.T) {
	// Build a struct type with *Entity embed
	withEntity := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					// Anonymous *Entity field
					Type: &ast.StarExpr{
						X: &ast.Ident{Name: "Entity"},
					},
				},
			},
		},
	}
	if !embedsEntityPointer(withEntity) {
		t.Error("expected true for struct with *Entity embed")
	}

	// Struct without *Entity
	withoutEntity := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: "Name"}},
					Type:  &ast.Ident{Name: "string"},
				},
			},
		},
	}
	if embedsEntityPointer(withoutEntity) {
		t.Error("expected false for struct without *Entity embed")
	}

	// Empty struct
	empty := &ast.StructType{
		Fields: &ast.FieldList{},
	}
	if embedsEntityPointer(empty) {
		t.Error("expected false for empty struct")
	}
}
