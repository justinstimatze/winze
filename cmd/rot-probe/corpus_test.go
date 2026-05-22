package main

import (
	"testing"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// Most parse / trip-detection coverage lives in internal/corpusparse's
// own tests. cmd/rot-probe owns sampling, filterConnected, and the
// reify-machinery exclusion — those are tested here.

func TestExcludeReifyMachinery(t *testing.T) {
	in := []entity{
		{VarName: "Apophenia"},
		{VarName: "TripLintTripCycle25Check"},
		{VarName: "TripBuildXCheck"},
		{VarName: "TripLLMYCheck"},
		{VarName: "TripFunctionalZCheck"},
		{VarName: "EvidenceSearchUDHRArticle3"},
		{VarName: "TripCycle25Real"}, // trip promotion is NOT machinery
		{VarName: "KlausConrad"},
	}
	got := excludeReifyMachinery(in)

	keptNames := map[string]bool{}
	for _, e := range got {
		keptNames[e.VarName] = true
	}
	wantKept := []string{"Apophenia", "TripCycle25Real", "KlausConrad"}
	wantDropped := []string{
		"TripLintTripCycle25Check",
		"TripBuildXCheck",
		"TripLLMYCheck",
		"TripFunctionalZCheck",
		"EvidenceSearchUDHRArticle3",
	}
	for _, w := range wantKept {
		if !keptNames[w] {
			t.Errorf("want %s kept, got dropped", w)
		}
	}
	for _, w := range wantDropped {
		if keptNames[w] {
			t.Errorf("want %s dropped, got kept", w)
		}
	}
	if len(got) != len(wantKept) {
		t.Errorf("kept %d, want %d", len(got), len(wantKept))
	}
}

func TestFilterConnected(t *testing.T) {
	hoods := []neighborhood{
		{ent: corpusparse.Entity{VarName: "lonely"}},
		{ent: corpusparse.Entity{VarName: "connected"}, asSubj: []claim{{VarName: "c1"}}},
	}
	got := filterConnected(hoods)
	if len(got) != 1 || got[0].ent.VarName != "connected" {
		t.Errorf("filterConnected: %+v", got)
	}
}

func TestSampleDeterministic(t *testing.T) {
	hoods := make([]neighborhood, 100)
	for i := range hoods {
		hoods[i] = neighborhood{
			ent:    corpusparse.Entity{VarName: stringN(i)},
			asSubj: []claim{{VarName: "c"}},
		}
	}
	a := sample(hoods, 5, 42)
	b := sample(hoods, 5, 42)
	if len(a) != 5 || len(b) != 5 {
		t.Fatalf("sample size: %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ent.VarName != b[i].ent.VarName {
			t.Errorf("same seed gave different samples at %d: %s vs %s",
				i, a[i].ent.VarName, b[i].ent.VarName)
		}
	}
}

func TestSampleAllWhenNGEN(t *testing.T) {
	hoods := []neighborhood{
		{ent: corpusparse.Entity{VarName: "a"}, asSubj: []claim{{VarName: "c"}}},
		{ent: corpusparse.Entity{VarName: "b"}, asSubj: []claim{{VarName: "c"}}},
	}
	got := sample(hoods, 100, 1)
	if len(got) != 2 {
		t.Errorf("want all 2 returned, got %d", len(got))
	}
}

func stringN(i int) string {
	return string(rune('a'+(i%26))) + string(rune('0'+(i/26)))
}
