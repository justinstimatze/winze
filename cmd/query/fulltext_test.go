package main

import "testing"

func TestTokenize(t *testing.T) {
	got := tokenize("Pattern-Detection FAILURE, a 3rd x")
	want := []string{"pattern", "detection", "failure", "3rd"} // 1-char "a"/"x" dropped
	if len(got) != len(want) {
		t.Fatalf("tokenize = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tokenize = %v, want %v", got, want)
		}
	}
}

// TestFulltextRanking: a document carrying the query terms outranks the field,
// and a document with none of them does not appear at all.
func TestFulltextRanking(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{VarName: "A", Name: "Apophenia", Brief: "finding meaningful patterns in random noise, a pattern detection failure"},
		{VarName: "B", Name: "Consciousness", Brief: "the hard problem of subjective experience"},
		{VarName: "C", Name: "Noise", Brief: "random unstructured signal"},
	}}
	hits := buildFTIndex(kb).search("pattern detection", 0)
	if len(hits) == 0 {
		t.Fatal("expected hits for 'pattern detection'")
	}
	if hits[0].kind != "entity" || kb.Entities[hits[0].ref].VarName != "A" {
		t.Fatalf("expected entity A ranked first, got %+v", hits[0])
	}
	for _, h := range hits {
		if kb.Entities[h.ref].VarName == "B" {
			t.Fatal("B shares no query terms; must not match")
		}
	}
}

// TestFulltextIncludesProvenance confirms Quotes are searchable, not just Briefs.
func TestFulltextIncludesProvenance(t *testing.T) {
	kb := &kbIndex{Provenance: []provRecord{
		{VarName: "P1", Origin: "Sagan 1995", Quote: "extraordinary claims require extraordinary evidence"},
	}}
	hits := buildFTIndex(kb).search("extraordinary evidence", 0)
	if len(hits) != 1 || hits[0].kind != "provenance" {
		t.Fatalf("expected one provenance hit, got %+v", hits)
	}
}

func TestFulltextEmptyQuery(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{{VarName: "A", Brief: "x y z"}}}
	if hits := buildFTIndex(kb).search("   ,  ", 0); hits != nil {
		t.Fatalf("empty/punctuation-only query should return no hits, got %+v", hits)
	}
}
