package main

import (
	"testing"
)

// TestPickCrossClusterPairs_SkipsUnpromotableRoles pins that the sampler
// refuses to spend a generation on a pair whose roles admit no promotable
// predicate.
//
// The rule already existed downstream, in tripCompatiblePredicates, which
// returns nil for any Person endpoint because trip has no source-grounding
// step. Downstream is too late to save anything: the pair has already been
// selected and a rationale already written for it. Ten of the eleven rows
// in .metabolism-trip-isolated.jsonl on 2026-08-05 were Person pairs that
// died that way, each one a paid call with nowhere to land.
func TestPickCrossClusterPairs_SkipsUnpromotableRoles(t *testing.T) {
	// Shared predicate vocabulary on every entity, so the affinity floor is
	// not what decides this test. Without it the promotable pair is dropped
	// for having no route and the assertion below passes for the wrong
	// reason — which is exactly what the first draft of this test did.
	shared := map[string]bool{"TheoryOf": true, "StructurallyAnalogousTo": true}
	ents := []tripEntity{
		{name: "DavidLoy", roleType: "Person", cluster: 0, brief: "nondualism", predicates: shared},
		{name: "Kresak", roleType: "Person", cluster: 1, brief: "comet hypothesis", predicates: shared},
		{name: "Nondualism", roleType: "Concept", cluster: 0, brief: "subject-object collapse", predicates: shared},
		{name: "HypothesisCometEnckeFragment", roleType: "Hypothesis", cluster: 1, brief: "Encke fragment", predicates: shared},
	}
	// Guard the guard: if the floor is what removes the promotable pair, this
	// test proves nothing about roles.
	if got := pairStructuralAffinity(ents[2], ents[3]); got < affinityFloor {
		t.Fatalf("fixture's promotable pair sits under the affinity floor (%d < %d) — "+
			"it would be dropped for no route, not for its roles", got, affinityFloor)
	}
	pairs := pickCrossClusterPairs(ents, 10, map[string]bool{})

	for _, p := range pairs {
		if p.A.roleType == "Person" || p.B.roleType == "Person" {
			t.Errorf("selected %s(%s) x %s(%s): a Person endpoint can never be promoted, "+
				"so generating for it is a paid call with nowhere to land",
				p.A.name, p.A.roleType, p.B.name, p.B.roleType)
		}
	}
	// The guard must not empty the sampler out. Concept x Hypothesis is
	// promotable and crosses clusters, so it has to survive — a test that
	// only checked "no Person" would pass on a sampler that returned nothing.
	var kept bool
	for _, p := range pairs {
		if (p.A.name == "Nondualism" && p.B.name == "HypothesisCometEnckeFragment") ||
			(p.B.name == "Nondualism" && p.A.name == "HypothesisCometEnckeFragment") {
			kept = true
		}
	}
	if !kept {
		t.Errorf("the promotable Concept x Hypothesis pair was dropped too; got %d pair(s): %v",
			len(pairs), pairNames(pairs))
	}
}

// pairNames renders a pair slice for failure messages.
func pairNames(pairs []tripPair) []string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.A.name+"x"+p.B.name)
	}
	return out
}
