package main

import (
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// TestCheckOneContradiction_PinkerProposesLoyTypology pins the resolver's
// behavior on multi-attribution Proposes claims:
//
//	new:      Proposes(StevenPinker, LoyFiveFlavorsTypology)
//	existing: Proposes(DavidLoy, LoyFiveFlavorsTypology)
//
// MUST resolve as confirmed (not refuted). The corpus uses Proposes as a
// multi-attribution predicate — see tunguska.go where four scientists each
// Propose HypothesisStonyAsteroidAirburst — and predicates.go carries no
// //winze:single-originator pragma. An earlier version of this test
// asserted refuted under an "exclusive to one originator" rule that did
// not match corpus practice; the rule was dropped (see predicateGuidance
// in trip_llm_resolve.go) and this test now pins the corrected behavior.
//
// (Whether Pinker actually proposed Loy's typology is a *fabrication*
// question that the resolver cannot adjudicate without external evidence;
// catching fabrications is out of scope here.)
//
// API-gated: requires ANTHROPIC_API_KEY plus WINZE_RUN_LLM_TESTS=1 to opt
// in (the second gate keeps CI from spending tokens). Skipped silently
// without both.
func TestCheckOneContradiction_PinkerProposesLoyTypology(t *testing.T) {
	if os.Getenv("WINZE_RUN_LLM_TESTS") != "1" {
		t.Skip("set WINZE_RUN_LLM_TESTS=1 to run LLM smoke tests")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	pc := promotedClaim{
		VarName:   "TripCycle2StevenPinkerProposesLoyFiveFlavorsTypology",
		Subject:   "StevenPinker",
		Predicate: "Proposes",
		Object:    "LoyFiveFlavorsTypology",
	}
	all := []claimSummary{
		{
			VarName:   "LoyProposesFiveFlavors",
			Predicate: "Proposes",
			Subject:   "DavidLoy",
			Object:    "LoyFiveFlavorsTypology",
		},
		// A few unrelated claims touching either entity, to exercise the
		// neighborhood filter and prove the LLM picks out the conflict.
		{
			VarName:   "LoyAuthoredNonduality",
			Predicate: "Authored",
			Subject:   "DavidLoy",
			Object:    "NondualityBook",
		},
		{
			VarName:   "PinkerAuthoredBlankSlate",
			Predicate: "Authored",
			Subject:   "StevenPinker",
			Object:    "BlankSlateBook",
		},
	}

	verdict, evidence := checkOneContradiction(client, pc, all)
	t.Logf("verdict=%q evidence=%s", verdict, evidence)
	if verdict != "confirmed" {
		t.Errorf("expected confirmed (multi-Proposes is corpus-legitimate per tunguska.go pattern), got %q (evidence: %s)", verdict, evidence)
	}
}
