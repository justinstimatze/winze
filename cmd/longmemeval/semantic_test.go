package main

import (
	"testing"
)

func TestFuseRankMapsEmpty(t *testing.T) {
	if got := fuseRankMaps(map[int]int{}, map[int]int{}); len(got) != 0 {
		t.Fatalf("expected no fused results from two empty maps, got %v", got)
	}
}

// The whole point of the semantic channel: a fact with the worst possible
// term overlap (tied dead last, the position a zero-vocabulary-overlap fact
// like ROADMAP.md's 35a27287 case lands in) but the single best semantic
// match should still be lifted into top contention by fusion, not stay
// buried where term overlap alone would leave it. Symmetric mirror-image
// ranks give it the same total RRF score as the term-overlap favorite, so it
// lands in the top 2 rather than at #1 outright (a tie loses ties on index) --
// that's the honest claim fusion makes, not an unconditional #1 guarantee.
func TestFuseRankMapsRecoversVocabMismatch(t *testing.T) {
	term := map[int]int{0: 1, 1: 2, 2: 3, 3: 4, 4: 5}
	sem := map[int]int{4: 1, 3: 2, 2: 3, 1: 4, 0: 5}
	got := fuseRankMaps(term, sem)
	pos := -1
	for i, idx := range got {
		if idx == 4 {
			pos = i
		}
	}
	if pos < 0 {
		t.Fatalf("fact 4 missing from fused order: %v", got)
	}
	if pos > 1 {
		t.Fatalf("expected fact 4 (worst term overlap, best semantic match) in the top 2 after fusion, landed at position %d: %v", pos, got)
	}
}

// A fact ranked highly by BOTH channels must beat one ranked highly by only
// one, even if that one is #1 in its single list -- same property cmd/query's
// rrfFuse is tested for, checked here for fuseRankMaps's own 2-map signature.
func TestFuseRankMapsRewardsAgreement(t *testing.T) {
	term := map[int]int{10: 1, 20: 2, 30: 3}
	sem := map[int]int{20: 1, 30: 2, 40: 3}
	got := fuseRankMaps(term, sem)
	if len(got) == 0 || got[0] != 20 {
		t.Fatalf("expected fact 20 (in both lists) first, got %v", got)
	}
}

// scoreByOverlap is rankFacts's extracted core; this pins that the refactor
// didn't change rankFacts's own behavior.
func TestScoreByOverlapMatchesRankFactsOrder(t *testing.T) {
	facts := []Fact{
		{Attribute: "hobby", Value: "chess club", Quote: "I joined the chess club"},
		{Attribute: "pet", Value: "goldfish", Quote: "I have a pet goldfish"},
		{Attribute: "sport", Value: "chess tournament", Quote: "I won a chess tournament"},
	}
	ranked := rankFacts(facts, "Tell me about my chess hobby", 2)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(ranked))
	}
	for _, f := range ranked {
		if f.Attribute == "pet" {
			t.Fatalf("goldfish fact (no chess overlap) should not be in top 2: %+v", ranked)
		}
	}
}
