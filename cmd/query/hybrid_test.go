package main

import "testing"

// A document ranked highly by BOTH signals must beat one ranked highly by only
// one, even if that one is #1 in its single list.
func TestRRFRewardsAgreement(t *testing.T) {
	lex := map[int]int{10: 1, 20: 2, 30: 3} // BM25 order
	sem := map[int]int{20: 1, 30: 2, 40: 3} // embedding order
	got := rrfFuse(lex, sem)

	// 20 is #2 lexical + #1 semantic; 10 is #1 lexical only. Agreement wins.
	if got[0].idx != 20 {
		t.Fatalf("expected doc 20 (in both lists) first, got %d (%+v)", got[0].idx, got)
	}
	if got[0].lex != 2 || got[0].sem != 1 {
		t.Fatalf("doc 20 ranks wrong: %+v", got[0])
	}
}

// A document present in only one list still surfaces, scored from that list.
func TestRRFSingleListSurvives(t *testing.T) {
	got := rrfFuse(map[int]int{99: 1}, map[int]int{})
	if len(got) != 1 || got[0].idx != 99 || got[0].sem != 0 {
		t.Fatalf("single-list doc lost: %+v", got)
	}
	want := 1.0 / float64(rrfK+1)
	if got[0].rrf != want {
		t.Fatalf("rrf = %v, want %v", got[0].rrf, want)
	}
}

func TestRRFEmpty(t *testing.T) {
	if got := rrfFuse(map[int]int{}, map[int]int{}); len(got) != 0 {
		t.Fatalf("empty inputs should fuse to nothing, got %+v", got)
	}
}

func TestCanonicalRoleCaseInsensitive(t *testing.T) {
	kb := &kbIndex{RoleTypes: map[string]bool{"Hypothesis": true, "Concept": true}}
	if got, ok := canonicalRole(kb, "hypothesis"); !ok || got != "Hypothesis" {
		t.Errorf("canonicalRole(hypothesis) = %q,%v; want Hypothesis,true", got, ok)
	}
	if _, ok := canonicalRole(kb, "bogus"); ok {
		t.Errorf("canonicalRole(bogus) should not resolve")
	}
}

// neighborhood must read edge DIRECTION off the verified graph: when the query
// entity is the object, the neighbor is the subject (← ), and the neighbor's
// role comes from the index, not a guess. Unary claims report on the entity.
func TestNeighborhoodDirectionAndRole(t *testing.T) {
	kb := &kbIndex{
		Entities: []entityRecord{
			{VarName: "Apophenia", RoleType: "Concept"},
			{VarName: "Shermer", RoleType: "Person"},
		},
		Claims: []claimRecord{
			{Predicate: "Proposes", Subject: "Shermer", Object: "Apophenia"},
			{Predicate: "IsPolyvalentTerm", Subject: "Apophenia", Object: ""}, // unary
		},
	}
	got := neighborhood(kb, "Apophenia")
	if len(got) != 2 {
		t.Fatalf("want 2 edges, got %d: %+v", len(got), got)
	}
	// Apophenia is the OBJECT of Proposes → neighbor is the subject, arrow ←.
	if got[0]["label"] != "Proposes ← Shermer (Person)" {
		t.Errorf("edge 0 label = %q", got[0]["label"])
	}
	if got[1]["label"] != "IsPolyvalentTerm (unary)" {
		t.Errorf("unary edge label = %q", got[1]["label"])
	}
}
