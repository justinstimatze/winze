package main

import (
	"testing"
)

// TestDownrankSupersededNoSupersedesClaimsIsNoop confirms the common case
// (a corpus with no Supersedes claims at all) leaves ranking untouched.
func TestDownrankSupersededNoSupersedesClaimsIsNoop(t *testing.T) {
	kb := kbWith(nil, map[string]string{"A": "A", "B": "B"})
	fused := []fusedHit{{idx: idxOf(kb, "B")}, {idx: idxOf(kb, "A")}}
	got := downrankSuperseded(kb, fused, false)
	if names := varNames(kb, got); len(names) != 2 || names[0] != "B" || names[1] != "A" {
		t.Errorf("downrankSuperseded with no Supersedes claims = %v, want the untouched order [B A]", names)
	}
}

// TestDownrankSupersededPreservesRelativeOrderWithinEachGroup checks the
// partition is stable, not just "superseded last": ties within the kept
// group and within the stale group keep their original relative order.
func TestDownrankSupersededPreservesRelativeOrderWithinEachGroup(t *testing.T) {
	kb := kbWith([]claimRecord{
		{Predicate: "Supersedes", Subject: "NEW", Object: "OLD1"},
		{Predicate: "Supersedes", Subject: "NEW", Object: "OLD2"},
	}, map[string]string{
		"OLD1": "Old 1", "OLD2": "Old 2", "NEW": "New", "OTHER": "Unrelated",
	})
	fused := []fusedHit{
		{idx: idxOf(kb, "OLD1")},
		{idx: idxOf(kb, "OTHER")},
		{idx: idxOf(kb, "OLD2")},
		{idx: idxOf(kb, "NEW")},
	}
	got := downrankSuperseded(kb, fused, false)
	want := []string{"OTHER", "NEW", "OLD1", "OLD2"}
	names := varNames(kb, got)
	if len(names) != len(want) {
		t.Fatalf("downrankSuperseded(...) = %v, want %v", names, want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("downrankSuperseded(...)[%d] = %q, want %q (got %v)", i, names[i], n, names)
		}
	}
}

func TestDownrankSupersededSinksBelowNonSuperseded(t *testing.T) {
	kb := hybridFixtureKB()
	// OLD is superseded and outranks NEW on raw RRF — kbWith builds Entities
	// from a map, so its order is not guaranteed; resolve indices by name
	// rather than assuming position.
	fused := []fusedHit{
		{idx: idxOf(kb, "OLD"), rrf: 0.9},
		{idx: idxOf(kb, "NEW"), rrf: 0.5},
	}

	got := downrankSuperseded(kb, fused, false)
	if names := varNames(kb, got); len(names) != 2 || names[0] != "NEW" || names[1] != "OLD" {
		t.Fatalf("downrank(include=false) = %v, want NEW first (kept), OLD last (stale) despite OLD's higher raw rrf", names)
	}

	got = downrankSuperseded(kb, fused, true)
	if names := varNames(kb, got); len(names) != 2 || names[0] != "OLD" || names[1] != "NEW" {
		t.Fatalf("downrank(include=true) = %v, want the untouched natural order (OLD, NEW)", names)
	}
}

// hybridFixtureKB builds a two-entity kb where NEW supersedes OLD, for
// downrankSuperseded's tests — pure and deterministic, no BM25/embeddings.
func hybridFixtureKB() *kbIndex {
	return kbWith([]claimRecord{
		{Predicate: "Supersedes", Subject: "NEW", Object: "OLD"},
	}, map[string]string{"OLD": "Old Decision", "NEW": "New Decision"})
}

func idxOf(kb *kbIndex, varName string) int {
	for i, e := range kb.Entities {
		if e.VarName == varName {
			return i
		}
	}
	return -1
}

func varNames(kb *kbIndex, hits []fusedHit) []string {
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = kb.Entities[h.idx].VarName
	}
	return names
}
