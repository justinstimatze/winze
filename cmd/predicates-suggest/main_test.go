package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterByScore(t *testing.T) {
	entries := []tripIsolated{
		{Score: 1}, {Score: 2}, {Score: 3}, {Score: 4}, {Score: 5},
	}
	got := filterByScore(entries, 3)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	for _, e := range got {
		if e.Score < 3 {
			t.Errorf("kept low score: %d", e.Score)
		}
	}
}

func TestReadTripIsolated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	body := `{"entity_a":"A","entity_b":"B","connection":"c","rationale":"r","score":3,"prompt_type":"analogy"}
{"entity_a":"X","entity_b":"Y","connection":"c2","rationale":"r2","score":4,"prompt_type":"analogy"}
malformed line skipped silently
{"entity_a":"P","entity_b":"Q","connection":"c3","rationale":"r3","score":2,"prompt_type":"analogy"}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readTripIsolated(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries (malformed skipped), got %d", len(got))
	}
	if got[0].EntityA != "A" || got[1].Score != 4 || got[2].EntityB != "Q" {
		t.Errorf("decode shape wrong: %+v", got)
	}
}

func TestLoadExistingPredicates(t *testing.T) {
	const fixture = `package winze

type Foo BinaryRelation[Person, Hypothesis]
type Bar UnaryClaim[Concept]
type NotAPredicate struct{ X int }
type Baz BinaryRelation[Place, Place]
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "predicates.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	names, err := loadExistingPredicates(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	for _, want := range []string{"Foo", "Bar", "Baz"} {
		if !got[want] {
			t.Errorf("want predicate %s in list, got %v", want, names)
		}
	}
	if got["NotAPredicate"] {
		t.Errorf("NotAPredicate should not be in list, got %v", names)
	}
	if len(names) != 3 {
		t.Errorf("want 3 predicates exactly, got %d: %v", len(names), names)
	}
}

func TestBuildPromptMentionsExistingPredicates(t *testing.T) {
	entries := []tripIsolated{
		{EntityA: "A", EntityB: "B", Connection: "conn", Rationale: "rat", Score: 3, PromptType: "analogy"},
	}
	existing := []string{"Proposes", "TheoryOf", "StructurallyAnalogousTo"}
	p := buildPrompt(entries, existing, 3)

	for _, ex := range existing {
		if !contains(p, ex) {
			t.Errorf("prompt missing existing predicate %q", ex)
		}
	}
	if !contains(p, "EntityA: A") {
		t.Error("prompt missing entity A")
	}
	if !contains(p, "rat") {
		t.Error("prompt missing rationale")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestFilterValidCandidates(t *testing.T) {
	cands := []predicateCandidate{
		{
			Name:          "RealCandidate",
			SubjectSlot:   "Hypothesis",
			ObjectSlot:    "Hypothesis",
			Rationale:     "encodes a genuine relation",
			SampleEntries: []string{"e1", "e2", "e3"},
			SampleClaims:  []string{"RealCandidate{Subject: A, Object: B}"},
		},
		{
			Name:          "TooSmall",
			SubjectSlot:   "Person",
			ObjectSlot:    "Concept",
			Rationale:     "only one example",
			SampleEntries: []string{"e1"},
			SampleClaims:  []string{"TooSmall{Subject: A, Object: B}"},
		},
		{
			Name:          "SkipMarker",
			SubjectSlot:   "Hypothesis",
			ObjectSlot:    "Hypothesis",
			Rationale:     "SKIP — absorbed by StructurallyAnalogousTo",
			SampleEntries: []string{"e1", "e2", "e3"},
			SampleClaims:  []string{"SKIP"},
		},
		{
			Name:          "SKIP",
			SubjectSlot:   "Person",
			ObjectSlot:    "Hypothesis",
			Rationale:     "everything is fine",
			SampleEntries: []string{"e1", "e2", "e3"},
			SampleClaims:  []string{"foo"},
		},
	}

	got := filterValidCandidates(cands, 3)
	if len(got) != 1 {
		t.Fatalf("want 1 valid candidate, got %d: %+v", len(got), got)
	}
	if got[0].Name != "RealCandidate" {
		t.Errorf("kept wrong candidate: %s", got[0].Name)
	}
}
