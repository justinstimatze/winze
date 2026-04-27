package main

// Critic replay test — feeds the four ingest claims that the
// 2026-04-27 adversarial review deleted back through the live critic
// and reports the verdict. If the critic rejects all four with
// reasonable reasons, the gates are well-calibrated. If it accepts
// any, the critic prompt needs strengthening.
//
// Skipped without ANTHROPIC_API_KEY. Cost: ~$0.03 per run (5 Haiku
// calls). NOT committed to CI; run manually:
//
//   go test ./cmd/metabolism/ -run TestCriticReplay -v
//
// The bad-input fixtures are reconstructed from the git history of
// commits 1198009, 3b282d8, bfed44b — see corpus/CLAUDE.md audit notes
// in apophenia.go, theory_seeds.go, tunguska.go, predictive_processing.go.

import (
	"fmt"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestCriticReplay_KnownBadIngestClaims(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set — skipping live critic replay")
	}

	// Sample real exemplars from the live corpus so the critic has the
	// same quality bar it would see in production. The test runs from
	// the package dir; the corpus is two levels up.
	corpusDir := "../.."
	exemplars := sampleHighQualityClaims(corpusDir, 5, 200)
	if len(exemplars) == 0 {
		t.Fatalf("no exemplars sampled from %s — corpus state issue?", corpusDir)
	}
	t.Logf("sampled %d exemplars from corpus", len(exemplars))

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	type fixture struct {
		name      string
		candidate *ingestResult
		predicate string
		target    string
	}

	fixtures := []fixture{
		{
			name: "CambridgeCore_AcceptsOrg_VENUE_NOT_ORG",
			candidate: &ingestResult{
				entityName:  "Cambridge Core",
				entityKind:  "organization",
				entityBrief: "Academic publishing platform that published commentary on hierarchical prediction theories in Behavioral and Brain Sciences.",
				quote:       `"Clark makes a convincing case for the merits of conceptualizing brains as hierarchical prediction machines."`,
			},
			predicate: "AcceptsOrg",
			target:    "HierarchicalPredictionMachine",
		},
		{
			name: "AytonFischer_Accepts_CITATION_NOT_ATTRIBUTION",
			candidate: &ingestResult{
				entityName:  "Ayton & Fischer",
				entityKind:  "person",
				entityBrief: "Researchers who documented apophenia as an empirical phenomenon in humans.",
				quote:       `"In humans, this is an empirically well-documented phenomenon (Ayton & Fischer, 2004; Falk & Konold, 1997; Gilovich, Vallone, & Tversky, 1985)."`,
			},
			predicate: "Accepts",
			target:    "ConradApopheniaClinicalFraming",
		},
		{
			name: "Reification_IsCognitiveBias_PREDICATE_MISUSE",
			candidate: &ingestResult{
				entityName:  "reification",
				entityKind:  "concept",
				entityBrief: "The cognitive practice of treating heuristics or abstract constructs as discrete entities, which can result in epistemological violence.",
				quote:       `"Reification is a necessary mechanism to address when countering discursive practices that result in epistemological violence. The misrepresentation of scientific knowledge, which arises when a heuristic such as ADHD is portrayed as a discrete entity, necessitates that authors who report their own research and the work of others do so with..."`,
			},
			predicate: "IsCognitiveBias",
			target:    "ReificationRisk",
		},
		{
			name: "ChybaEtAl_Proposes_DUPLICATE_OF_CANONICAL",
			candidate: &ingestResult{
				entityName:  "Chyba et al.",
				entityKind:  "person",
				entityBrief: "Collaboration of researchers who proposed the asteroidal nature hypothesis for the Tunguska meteoroid in 1993.",
				quote:       `"In relation to the conjecture proposed by Chyba et al. (1993) about the asteroidal nature of the Tunguska meteoroid"`,
			},
			predicate: "Proposes",
			target:    "HypothesisStonyAsteroidAirburst",
		},
	}

	rejectCount := 0
	for _, f := range fixtures {
		v := critiqueIngestClaim(client, f.candidate, f.predicate, f.target, exemplars)
		verdictStr := "ACCEPT (BUG — should have rejected)"
		if !v.Accept {
			verdictStr = fmt.Sprintf("REJECT (%s) ✓", v.Reason)
			rejectCount++
		}
		t.Logf("[%s] → %s", f.name, verdictStr)
	}

	t.Logf("---")
	t.Logf("Critic replay: %d/%d known-bad fixtures rejected", rejectCount, len(fixtures))
	if rejectCount < len(fixtures) {
		t.Errorf("critic accepted %d known-bad inputs that the human audit deleted; prompt needs strengthening", len(fixtures)-rejectCount)
	}
}
