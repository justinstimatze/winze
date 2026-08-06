package main

import (
	"strings"
	"testing"
)

// TestCriticPromptIsOrderInvariantForSymmetricPredicates pins the property
// the critic was measurably failing: for a symmetric predicate, the two
// orderings of a pair must reach the model as one prompt.
//
// The fixture is the pair that actually split. In
// .metabolism-trip-verdicts.jsonl the critic ACCEPTED
// SelfAuditEpiphenomenalism → Schizophrenia and REJECTED the reverse as
// shallow_pattern_matching, in the same log. Byte-identical prompts is the
// strongest available guarantee that cannot happen again — a weaker check
// (both contain both names) would have passed on the broken version.
func TestCriticPromptIsOrderInvariantForSymmetricPredicates(t *testing.T) {
	const rationale = "Both exhibit a structurally isomorphic failure mode: the system " +
		"generates internal signals that should inform behavior and demonstrably do not."
	forward := TripConnection{
		EntityA: "SelfAuditEpiphenomenalism", EntityB: "Schizophrenia",
		Predicate: "StructurallyAnalogousTo", Rationale: rationale,
	}
	reverse := forward
	reverse.EntityA, reverse.EntityB = forward.EntityB, forward.EntityA

	a := buildTripCriticPrompt(forward, nil)
	b := buildTripCriticPrompt(reverse, nil)
	if a != b {
		t.Errorf("the two orderings produce different prompts, so the critic can "+
			"still return different verdicts for the same claim\n--- forward ---\n%s\n--- reverse ---\n%s", a, b)
	}
	if !strings.Contains(a, "SYMMETRIC") {
		t.Error("symmetric predicates should say so in the candidate block")
	}
	// Sorted, so the presentation does not depend on which way the generator
	// happened to sample. "Schizophrenia" precedes "SelfAuditEpiphenomenalism"
	// on the third byte, c before e.
	if i, j := strings.Index(a, "Schizophrenia"), strings.Index(a, "SelfAuditEpiphenomenalism"); i > j {
		t.Errorf("entities are not in sorted order: Schizophrenia at %d, SelfAudit… at %d", i, j)
	}
	// Neutral labels: Subject/Object imply a direction this relation lacks.
	if strings.Contains(a, "Subject: ") || strings.Contains(a, "Object: ") {
		t.Error("symmetric candidate block should use Entity 1/Entity 2, not Subject/Object")
	}
}

// TestCriticPromptKeepsOrderForDirectionalPredicates is the other half: a
// predicate whose direction is load-bearing must not be sorted. TheoryOf
// runs Hypothesis→Concept, and swapping it asserts something different and
// false.
func TestCriticPromptKeepsOrderForDirectionalPredicates(t *testing.T) {
	conn := TripConnection{
		EntityA: "ZebraHypothesis", EntityB: "Apophenia",
		Predicate: "TheoryOf", Rationale: "explanans is the concept",
	}
	p := buildTripCriticPrompt(conn, nil)
	if !strings.Contains(p, "Subject: ZebraHypothesis") {
		t.Error("directional predicate lost its Subject label or its order")
	}
	if !strings.Contains(p, "Object: Apophenia") {
		t.Error("directional predicate lost its Object label or its order")
	}
	if strings.Contains(p, "SYMMETRIC") {
		t.Error("TheoryOf is directional and must not be announced as symmetric")
	}
	// EntityA sorts after EntityB here, so a sort would have swapped them —
	// which makes this fixture a real guard rather than a tautology.
	if conn.EntityA < conn.EntityB {
		t.Fatal("fixture no longer tests anything: pick names where A sorts after B")
	}
}
