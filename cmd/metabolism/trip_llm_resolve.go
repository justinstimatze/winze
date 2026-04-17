package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/justinstimatze/winze/internal/defndb"
)

// promotedClaim carries enough context about a trip-promoted claim to
// reconstruct its (subject, predicate, object) triple at resolve time,
// without re-parsing var names.
type promotedClaim struct {
	VarName   string
	Subject   string
	Predicate string
	Object    string
}

// logTripLLMDurability runs a targeted LLM contradiction check per promoted
// claim. For each claim, we collect the neighborhood of existing claims
// mentioning either entity, send the neighborhood + new claim to an LLM,
// and ask whether the new claim contradicts anything. Stricter than
// lint-durability because deterministic rules can't catch semantic
// conflict — e.g. two non-functional predicates making opposing
// commitments about the same entities.
//
// Cost: one LLM call per promoted claim. Gated on ANTHROPIC_API_KEY;
// silently skipped when the key is absent or the SDK call fails (logged
// with resolution="" so calibrate shows pending instead of a false
// confirmation).
func logTripLLMDurability(dir string, claims []promotedClaim) error {
	if len(claims) == 0 {
		return nil
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("[trip-promote] llm-durability: skipped (ANTHROPIC_API_KEY not set)")
		return nil
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	dbClient, err := defndb.New(dir)
	if err != nil {
		return fmt.Errorf("defndb: %w", err)
	}
	defer dbClient.Close()
	neighborhood, err := neighborhoodClaims(dbClient)
	if err != nil {
		return fmt.Errorf("collect neighborhood: %w", err)
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)
	today := time.Now().Format("2006-01-02")

	confirmed, refuted, errored := 0, 0, 0
	for _, pc := range claims {
		result, evidence := checkOneContradiction(client, pc, neighborhood)
		c := Cycle{
			Timestamp:      time.Now(),
			Hypothesis:     pc.VarName,
			Prediction:     "trip-promoted claim does not contradict existing neighborhood claims",
			VulnType:       "trip_promotion",
			PredictionType: "trip_llm_durability",
			ResolvedAt:     today,
			Evidence:       evidence,
		}
		switch result {
		case "confirmed":
			c.Resolution = "confirmed"
			confirmed++
		case "refuted":
			c.Resolution = "refuted"
			refuted++
		default:
			errored++
		}
		mlog.Cycles = append(mlog.Cycles, c)
	}

	if err := saveLog(logPath, mlog); err != nil {
		return fmt.Errorf("save log: %w", err)
	}

	fmt.Printf("[trip-promote] llm-durability: %d confirmed, %d refuted, %d errored (logged as trip_llm_durability)\n",
		confirmed, refuted, errored)
	return nil
}

// claimSummary is the slim per-claim view the contradiction prompt needs.
type claimSummary struct {
	VarName   string
	Predicate string
	Subject   string
	Object    string
}

// neighborhoodClaims pulls every existing claim (defName + Subject + Object +
// TypeName) from defndb. The contradiction check filters by entity at prompt
// time, so we pull once and filter in-memory per promoted claim.
func neighborhoodClaims(c *defndb.Client) ([]claimSummary, error) {
	fields, err := c.ClaimFields()
	if err != nil {
		return nil, err
	}
	type partial struct {
		typeName, subject, object string
		hasS, hasO                bool
	}
	byName := map[string]*partial{}
	for _, f := range fields {
		parts := strings.Split(f.TypeName, ".")
		tn := parts[len(parts)-1]
		p, ok := byName[f.DefName]
		if !ok {
			p = &partial{typeName: tn}
			byName[f.DefName] = p
		}
		v := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Subject":
			p.subject = v
			p.hasS = true
		case "Object":
			p.object = v
			p.hasO = true
		}
	}
	var out []claimSummary
	for name, p := range byName {
		if !p.hasS || !p.hasO {
			continue
		}
		out = append(out, claimSummary{
			VarName:   name,
			Predicate: p.typeName,
			Subject:   p.subject,
			Object:    p.object,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VarName < out[j].VarName })
	return out, nil
}

// checkOneContradiction asks the LLM whether the new claim contradicts
// any claim in the neighborhood touching either entity. Returns
// ("confirmed"|"refuted"|"", evidence). On any SDK or decode error the
// first return is "" (treated as pending by calibrate, not a hit or miss).
func checkOneContradiction(client anthropic.Client, pc promotedClaim, all []claimSummary) (string, string) {
	// Filter neighborhood: any claim where Subject or Object ∈ {pc.Subject, pc.Object},
	// minus the new claim itself.
	var neighborhood []claimSummary
	for _, cs := range all {
		if cs.VarName == pc.VarName {
			continue
		}
		if cs.Subject == pc.Subject || cs.Object == pc.Subject ||
			cs.Subject == pc.Object || cs.Object == pc.Object {
			neighborhood = append(neighborhood, cs)
		}
	}

	var nbLines []string
	for _, cs := range neighborhood {
		nbLines = append(nbLines, fmt.Sprintf("  %s(%s, %s)", cs.Predicate, cs.Subject, cs.Object))
	}
	if len(nbLines) == 0 {
		nbLines = []string{"  (no existing claims touching either entity)"}
	}

	// Collect distinct predicates appearing in the new claim + neighborhood
	// and emit guidance for each. Tightening the oracle: the prior prompt
	// was too permissive — it accepted "two distinct people Propose the same
	// hypothesis" because it didn't know Proposes implies originating
	// authorship. Predicate guidance closes that hole.
	predSet := map[string]bool{pc.Predicate: true}
	for _, cs := range neighborhood {
		predSet[cs.Predicate] = true
	}
	var predNames []string
	for p := range predSet {
		predNames = append(predNames, p)
	}
	sort.Strings(predNames)
	var guidanceLines []string
	for _, p := range predNames {
		if g := predicateGuidance(p); g != "" {
			guidanceLines = append(guidanceLines, fmt.Sprintf("  %s: %s", p, g))
		}
	}
	guidanceBlock := "  (no specific guidance)"
	if len(guidanceLines) > 0 {
		guidanceBlock = strings.Join(guidanceLines, "\n")
	}

	prompt := fmt.Sprintf(`You audit a newly-added claim against a knowledge base neighborhood for semantic contradiction. The KB is small and curated; let predicate semantics drive the verdict, not common-sense leniency.

NEW CLAIM:
  %s(%s, %s)

EXISTING CLAIMS IN NEIGHBORHOOD (%d):
%s

PREDICATE SEMANTICS (apply these strictly):
%s

FLAG (contradicts=true) when:
  - A predicate marked "exclusive to one originator" already has a different Subject for the same Object (e.g. existing Proposes(A, H) plus new Proposes(B, H), where A and B are not stated co-originators).
  - A predicate marked "functional" already has a different Object for the same Subject (e.g. existing FormedAt(P, T1) plus new FormedAt(P, T2)).
  - The new claim implies a cycle that the predicate disallows (e.g. LocatedIn(A, B) when LocatedIn(B, A) already holds, transitively or directly).
  - The new claim asserts a relationship the existing neighborhood explicitly contradicts (e.g. existing Disputes(P, H) plus new Accepts(P, H)).

DO NOT FLAG:
  - Multiple TheoryOf claims for the same Concept (TheoryOf is //winze:contested — competing theories are normal).
  - Two non-functional, non-exclusive predicates touching the same entities with no semantic conflict.
  - Mere topical overlap.

If the new claim is ambiguous and the predicate is not exclusive/functional, contradicts=false. If exclusive/functional and the conflict is plausible, contradicts=true; do not extend the benefit of the doubt.

Call check_contradiction with your verdict, the conflicting existing-claim var name (or "none"), and a one-sentence reason citing the rule above.`,
		pc.Predicate, pc.Subject, pc.Object,
		len(neighborhood),
		strings.Join(nbLines, "\n"),
		guidanceBlock)

	schema := anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"contradicts": map[string]any{
				"type":        "boolean",
				"description": "true if the new claim contradicts at least one existing claim per the predicate-semantics rules in the prompt; false otherwise",
			},
			"conflicting_var": map[string]any{
				"type":        "string",
				"description": "the var name of the conflicting existing claim, or 'none' if contradicts=false",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "one-sentence explanation citing the rule from the prompt that fired (or that absented)",
			},
		},
		Required: []string{"contradicts", "conflicting_var", "reason"},
	}
	tool := anthropic.ToolUnionParamOfTool(schema, "check_contradiction")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.ModelClaudeHaiku4_5,
		MaxTokens:   256,
		Temperature: anthropic.Float(0.0),
		Tools:       []anthropic.ToolUnionParam{tool},
		ToolChoice:  anthropic.ToolChoiceParamOfTool("check_contradiction"),
		Messages:    []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return "", fmt.Sprintf("llm error: %v", err)
	}
	for _, block := range resp.Content {
		if block.Type == "tool_use" && block.Name == "check_contradiction" {
			var out struct {
				Contradicts    bool   `json:"contradicts"`
				ConflictingVar string `json:"conflicting_var"`
				Reason         string `json:"reason"`
			}
			if err := json.Unmarshal([]byte(block.Input), &out); err != nil {
				return "", fmt.Sprintf("llm decode: %v", err)
			}
			if out.Contradicts {
				return "refuted", fmt.Sprintf("LLM flagged contradiction with %s: %s", out.ConflictingVar, out.Reason)
			}
			return "confirmed", fmt.Sprintf("LLM found no contradiction across %d neighborhood claim(s): %s", len(neighborhood), out.Reason)
		}
	}
	return "", "llm returned no tool_use block"
}

// predicateGuidance returns the contradiction-relevant semantics of a
// predicate as a one-line string the LLM can apply mechanically. Empty
// means the prompt's generic rules suffice — keep additions minimal,
// only encode what the LLM has been seen to get wrong.
//
// Source of truth is predicates.go (//winze:functional pragma is the
// canonical functional list). The "exclusive to one originator" cluster
// is a curated judgment call, not a pragma — a future predicate-metadata
// pragma (//winze:exclusive) could replace this lookup.
func predicateGuidance(predicate string) string {
	switch predicate {
	case "Proposes", "ProposesOrg":
		return "exclusive to one originator. Two distinct Subjects Proposing the same Object is a conflict unless they are stated co-originators."
	case "FormedAt", "EnergyEstimate", "ResolvedAs", "EnglishTranslationOf":
		return "//winze:functional. Each Subject has at most one Object via this predicate; a second Object with the same Subject is a conflict."
	case "TheoryOf":
		return "//winze:contested. Multiple distinct Subjects (theories) for the same Object (concept) is the expected, not contradictory, shape."
	case "LocatedIn":
		return "spatial containment. A LocatedIn cycle (A in B and B in A, transitively or directly) is a conflict."
	case "Accepts", "AcceptsOrg":
		return "an Accepts(P, H) directly contradicts an existing Disputes(P, H) (and vice versa) for the same Subject and Object."
	case "Disputes", "DisputesOrg":
		return "a Disputes(P, H) directly contradicts an existing Accepts(P, H) (and vice versa) for the same Subject and Object."
	}
	return ""
}
