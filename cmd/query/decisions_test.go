package main

import "testing"

func kbWith(claims []claimRecord, names map[string]string) *kbIndex {
	kb := &kbIndex{Claims: claims}
	for v, n := range names {
		kb.Entities = append(kb.Entities, entityRecord{VarName: v, Name: n})
	}
	return kb
}

func TestDecisionChains(t *testing.T) {
	// C supersedes B supersedes A — one chain, head C, two superseded.
	kb := kbWith([]claimRecord{
		{Predicate: "Supersedes", Subject: "C", Object: "B"},
		{Predicate: "Supersedes", Subject: "B", Object: "A"},
		{Predicate: "RelatesTo", Subject: "X", Object: "Y"}, // ignored
	}, map[string]string{"A": "Alpha", "B": "Beta", "C": "Gamma"})

	chains := decisionChains(kb)
	if len(chains) != 1 {
		t.Fatalf("want 1 chain, got %d: %+v", len(chains), chains)
	}
	if chains[0].CurrentName != "Gamma" {
		t.Errorf("head should be the un-superseded Gamma, got %q", chains[0].CurrentName)
	}
	want := []string{"Beta", "Alpha"}
	if len(chains[0].Superseded) != 2 || chains[0].Superseded[0] != want[0] || chains[0].Superseded[1] != want[1] {
		t.Errorf("chain order wrong: got %v want %v", chains[0].Superseded, want)
	}
}

func TestDecisionChainsMultipleHeadsOrderedByDepth(t *testing.T) {
	// Two independent chains: B>A (depth 1) and E>D>C (depth 2). Deeper first.
	kb := kbWith([]claimRecord{
		{Predicate: "Supersedes", Subject: "B", Object: "A"},
		{Predicate: "Supersedes", Subject: "E", Object: "D"},
		{Predicate: "Supersedes", Subject: "D", Object: "C"},
	}, nil)
	chains := decisionChains(kb)
	if len(chains) != 2 {
		t.Fatalf("want 2 chains, got %d", len(chains))
	}
	if chains[0].Current != "E" || len(chains[0].Superseded) != 2 {
		t.Errorf("deepest chain (E) should sort first, got %+v", chains[0])
	}
	if chains[1].Current != "B" {
		t.Errorf("second chain should be B, got %q", chains[1].Current)
	}
}

func TestDecisionChainsEmpty(t *testing.T) {
	kb := kbWith([]claimRecord{{Predicate: "RelatesTo", Subject: "X", Object: "Y"}}, nil)
	if got := decisionChains(kb); got != nil {
		t.Errorf("no Supersedes claims should yield no chains, got %+v", got)
	}
}

func TestDecisionChainsCycleTerminates(t *testing.T) {
	// A pathological Supersedes loop (a data error) must terminate, not hang.
	// A pure 2-cycle has no un-superseded head, so it correctly yields no
	// chains — the point of the test is that the walk returns at all.
	kb := kbWith([]claimRecord{
		{Predicate: "Supersedes", Subject: "A", Object: "B"},
		{Predicate: "Supersedes", Subject: "B", Object: "A"},
	}, nil)
	if got := decisionChains(kb); len(got) != 0 {
		t.Errorf("pure cycle has no head, want 0 chains, got %+v", got)
	}
}
