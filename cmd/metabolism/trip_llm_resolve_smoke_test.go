package main

import (
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// TestCheckOneContradiction_PinkerProposesLoyTypology pins the regression
// from session 6: the prompt previously accepted
//
//	new:      Proposes(StevenPinker, LoyFiveFlavorsTypology)
//	existing: Proposes(DavidLoy, LoyFiveFlavorsTypology)
//
// as compatible (LLM reasoned "multiple proposers can coexist"). With the
// session-7 prompt that embeds Proposes' "exclusive to one originator"
// semantics, this should now be flagged refuted.
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
	if verdict != "refuted" {
		t.Errorf("expected refuted (Pinker and Loy can't both originate the same typology), got %q (evidence: %s)", verdict, evidence)
	}
}
