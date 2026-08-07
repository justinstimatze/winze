package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// TestCriticStability measures how often the trip critic returns the same
// verdict for the same candidate. It is skipped by default: it makes real
// API calls and costs money.
//
//	WINZE_CRITIC_STABILITY=1 go test ./cmd/metabolism -run CriticStability -v
//
// Why this exists: on 2026-08-05 the critic accepted
// SelfAuditEpiphenomenalism ↔ Schizophrenia and promoted it to
// metabolism_cycle27.go, then rejected the same pair as
// shallow_pattern_matching one run later on a materially equivalent
// rationale. A threshold cannot be calibrated against a judge that
// disagrees with itself, so the flip rate has to be a measured number
// before any bar is moved. critiqueTripConnection sets no Temperature, so
// it samples at the API default of 1.0 with a 256-token budget and no
// reasoning step — this test says what that costs in consistency.
func TestCriticStability(t *testing.T) {
	if os.Getenv("WINZE_CRITIC_STABILITY") == "" {
		t.Skip("set WINZE_CRITIC_STABILITY=1 to run (makes billed API calls)")
	}
	dir := os.Getenv("WINZE_CORPUS_DIR")
	if dir == "" {
		dir = "../../corpus"
	}
	loadDotEnv(dir)
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	exemplars := sampleHighQualityClaims(dir, 5, 200)

	// Both phrasings of the pair that flipped, as the generator actually
	// produced them. The promoted one is verbatim from cycle27.
	cases := []struct {
		name string
		conn TripConnection
	}{
		{
			name: "promoted-phrasing",
			conn: TripConnection{
				EntityA: "SelfAuditEpiphenomenalism", EntityB: "Schizophrenia",
				Predicate: "StructurallyAnalogousTo", Score: 4,
				PromptType: "analogy", Temperature: 1.0,
				Connection: "SelfAuditEpiphenomenalism and Schizophrenia both feature a dissociation between internal modeling activity and actual behavioral/decisional output.",
				Rationale:  "Both exhibit a structurally isomorphic failure mode: the system generates internal signals (audit reports, self-awareness of delusions) that should inform behavior but demonstrably do not—in SelfAuditEpiphenomenalism by definitional design, in schizophrenia via the classic insight-avolition gap where awareness of false beliefs coexists with inability to revise action. This is a specific epistemic pathology, not generic similarity; it targets the causal isolation of self-model from behavioral control.",
			},
		},
		{
			name: "rejected-phrasing",
			conn: TripConnection{
				EntityA: "Schizophrenia", EntityB: "SelfAuditEpiphenomenalism",
				Predicate: "StructurallyAnalogousTo", Score: 4,
				PromptType: "analogy", Temperature: 1.0,
				Connection: "Schizophrenia manifests a failure of reality-model validation that is structurally isomorphic to SelfAuditEpiphenomenalism's core problem: detection of internal error that produces no corrective behavioral output.",
				Rationale:  "Both entities instantiate the same epistemic failure: a gap between model-assessment (percept-generation in schizophrenia; self-audit report-generation in the KB) and reality-congruent behavior. In schizophrenia, the mind generates reports (hallucinations) it cannot invalidate; in SelfAuditEpiphenomenalism, the KB generates audit reports (detected biases) that do not drive corrective action—a shared structural decoupling between signal-detection and behavioral correction.",
			},
		},
	}

	// Ablations. The two phrasings above differ in two ways: the accepted
	// one names a clinical term ("the classic insight-avolition gap"), and
	// it ends with a sentence asserting its own non-genericity ("This is a
	// specific epistemic pathology, not generic similarity"). If that
	// second sentence is what moves the verdict, the critic can be talked
	// out of a rejection by the candidate's self-assessment — a gate the
	// generator can game by appending a disclaimer.
	const selfCert = " This is a specific epistemic pathology, not generic similarity; it targets the causal isolation of self-model from behavioral control."
	rejectedPlus := cases[1].conn
	rejectedPlus.Rationale += selfCert
	promotedMinus := cases[0].conn
	promotedMinus.Rationale = "Both exhibit a structurally isomorphic failure mode: the system generates internal signals (audit reports, self-awareness of delusions) that should inform behavior but demonstrably do not—in SelfAuditEpiphenomenalism by definitional design, in schizophrenia via the classic insight-avolition gap where awareness of false beliefs coexists with inability to revise action."
	cases = append(cases,
		struct {
			name string
			conn TripConnection
		}{"rejected+selfcert", rejectedPlus},
		struct {
			name string
			conn TripConnection
		}{"promoted-selfcert", promotedMinus},
	)

	// Two variables survive the self-certification ablation. Isolate each.
	//
	// Direction: StructurallyAnalogousTo is symmetric and documented as
	// such, so swapping Subject and Object while holding the rationale
	// fixed must not move the verdict. If it does, the critic is reading a
	// symmetric predicate as directional.
	swapped := cases[0].conn
	swapped.EntityA, swapped.EntityB = cases[0].conn.EntityB, cases[0].conn.EntityA
	// Named term: the accepted rationale anchors on "the classic
	// insight-avolition gap"; the rejected one describes the same
	// decoupling without naming it. If inserting the term flips the
	// rejection, the critic is scoring vocabulary rather than structure.
	termed := cases[1].conn
	termed.Rationale = "Both entities instantiate the same epistemic failure: a gap between model-assessment (percept-generation in schizophrenia; self-audit report-generation in the KB) and reality-congruent behavior. In schizophrenia this is the classic insight-avolition gap, where awareness of false beliefs coexists with inability to revise action; in SelfAuditEpiphenomenalism, the KB generates audit reports (detected biases) that do not drive corrective action—a shared structural decoupling between signal-detection and behavioral correction."
	cases = append(cases,
		struct {
			name string
			conn TripConnection
		}{"promoted-swapped-dir", swapped},
		struct {
			name string
			conn TripConnection
		}{"rejected+namedterm", termed},
	)

	const trials = 10
	for _, tc := range cases {
		accepts, reasons := 0, map[string]int{}
		for i := 0; i < trials; i++ {
			v := critiqueTripConnection(client, tc.conn, exemplars, nil)
			if v.Accept {
				accepts++
			} else {
				reasons[v.Reason]++
			}
		}
		fmt.Printf("[critic-stability] %-18s %d/%d accept", tc.name, accepts, trials)
		for r, n := range reasons {
			fmt.Printf("  %s×%d", r, n)
		}
		fmt.Println()
		// Not an assertion about which verdict is right — only that the
		// judge should reach the same one twice. A split decision means
		// the bar is noise and no threshold change can be evaluated.
		if accepts != 0 && accepts != trials {
			t.Errorf("%s: unstable verdict — %d/%d accept; a threshold cannot be calibrated against this",
				tc.name, accepts, trials)
		}
	}
}
