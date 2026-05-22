package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/justinstimatze/winze/internal/corpusparse"
)

// predicateCandidate is one promotable predicate the LLM proposes after
// clustering trip-isolated entries by shape.
type predicateCandidate struct {
	Name          string   `json:"name"`            // suggested Go type name, e.g. "MutuallyAnticipates"
	SubjectSlot   string   `json:"subject_slot"`    // role type, e.g. "Hypothesis"
	ObjectSlot    string   `json:"object_slot"`     // role type, e.g. "Hypothesis"; empty for unary
	IsUnary       bool     `json:"is_unary"`        // true if UnaryClaim shape
	IsFunctional  bool     `json:"is_functional"`   // suggested //winze:functional pragma
	Rationale     string   `json:"rationale"`       // why this predicate is needed; what gap it closes
	SampleEntries []string `json:"sample_entries"`  // entity-pair descriptions from trip-isolated
	SampleClaims  []string `json:"sample_claims"`   // 2-3 concrete claims it would encode
}

// proposeCandidates sends the filtered trip-isolated entries to the LLM
// along with the existing predicate vocabulary. The model clusters by
// predicate-shape and proposes candidates for clusters meeting the size
// threshold.
func proposeCandidates(entries []tripIsolated, existing []string, minCluster int, model string) ([]predicateCandidate, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set (create .env at the corpus root or export it)")
	}

	modelID := anthropic.ModelClaudeSonnet4_6
	if model == "haiku" {
		modelID = anthropic.ModelClaudeHaiku4_5
	}

	prompt := buildPrompt(entries, existing, minCluster)

	schema := anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"candidates": map[string]any{
				"type":        "array",
				"description": "Predicate candidates, one per shape-cluster of size >= min_cluster. Empty array if no cluster reaches the threshold or if all the gap is genuinely noise-shaped.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "Suggested Go type name (PascalCase), e.g. 'MutuallyAnticipates'. Must not duplicate an existing predicate.",
						},
						"subject_slot": map[string]any{
							"type":        "string",
							"description": "Subject role type, e.g. 'Person', 'Hypothesis', 'Concept'. Use winze's existing role types only.",
						},
						"object_slot": map[string]any{
							"type":        "string",
							"description": "Object role type; empty string if is_unary=true.",
						},
						"is_unary": map[string]any{
							"type":        "boolean",
							"description": "True for UnaryClaim shape (tag-style claim); false for BinaryRelation.",
						},
						"is_functional": map[string]any{
							"type":        "boolean",
							"description": "True if each Subject should have at most one Object — would warrant //winze:functional pragma.",
						},
						"rationale": map[string]any{
							"type":        "string",
							"description": "Why this predicate is needed; what specific gap in the existing vocabulary it closes; why an existing predicate doesn't fit.",
						},
						"sample_entries": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Brief descriptions (one line each) of the trip-isolated entries that cluster under this candidate — e.g. 'SearleBiologicalNaturalism + KahnemanDualProcess: both anti-functionalist constraints'.",
						},
						"sample_claims": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "2-3 concrete Go-style claim sketches the predicate would let the corpus encode, e.g. 'MutuallyAnticipates{Subject: SearleHypothesis, Object: KahnemanHypothesis}'.",
						},
					},
					"required": []string{"name", "subject_slot", "object_slot", "is_unary", "is_functional", "rationale", "sample_entries", "sample_claims"},
				},
			},
		},
		Required: []string{"candidates"},
	}
	tool := anthropic.ToolUnionParamOfTool(schema, "report_candidates")

	client := anthropic.NewClient(option.WithAPIKey(key))
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:      modelID,
		MaxTokens:  8192,
		Tools:      []anthropic.ToolUnionParam{tool},
		ToolChoice: anthropic.ToolChoiceParamOfTool("report_candidates"),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	for _, block := range resp.Content {
		if block.Type != "tool_use" {
			continue
		}
		tu := block.AsToolUse()
		var out struct {
			Candidates []predicateCandidate `json:"candidates"`
		}
		if err := json.Unmarshal(tu.Input, &out); err != nil {
			return nil, fmt.Errorf("decode tool input: %w", err)
		}
		return out.Candidates, nil
	}
	return nil, fmt.Errorf("no tool_use block in response")
}

func buildPrompt(entries []tripIsolated, existing []string, minCluster int) string {
	var b strings.Builder
	b.WriteString(`You are extending the predicate vocabulary of a typed knowledge base about the epistemology of minds. The corpus has a strict discipline: predicates are NOT invented speculatively. A new predicate is earned only when multiple real generations ("forcing function": typically 3+ occurrences) want to encode the same relation shape and no existing predicate fits.

Your input is a log of trip-cycle generations from the metabolism loop. Each entry connects two entities the system thought were structurally related, and the rationale states why no existing predicate could capture the connection.

EXISTING PREDICATES (do NOT duplicate; if a cluster fits one of these, omit it):
`)
	sort.Strings(existing)
	for _, p := range existing {
		fmt.Fprintf(&b, "  - %s\n", p)
	}

	b.WriteString(`
WINZE ROLE TYPES (use these as slot types; do NOT invent new role types):
  Person, Organization, Concept, Hypothesis, Place, Event, Facility, Instrument, Substance

YOUR TASK:
1. Read the entries. Each describes a connection the system wanted to encode but couldn't.
2. Cluster them by predicate-shape: entries whose rationales describe the SAME structural relation should cluster together. Pay attention to:
   - the roles (Hypothesis-to-Hypothesis? Person-to-Concept? etc.)
   - the *kind* of relation the rationale describes (analogy? causation? prediction? constraint? rejection? translation?)
   - whether the relation is symmetric, asymmetric, functional, etc.
3. For each cluster of size >= ` + fmt.Sprintf("%d", minCluster) + `, propose a candidate predicate with name, slots, sample claims, rationale.
4. SKIP clusters smaller than ` + fmt.Sprintf("%d", minCluster) + ` — they have not earned a predicate yet.
5. SKIP clusters that an existing predicate would already absorb — surface that as an empty array rather than a duplicate proposal.
6. If the gap is genuinely noise-shaped (no coherent clusters), return an empty array. Empty is a valid answer; do not invent.

DISCIPLINE REMINDERS:
- Be conservative about widening role types. Hypothesis -> Hypothesis is a real shape; "Anything -> Anything" is not.
- Name predicates by the relation they encode, not by the domain. "StructurallyAnalogousTo" is good; "ConsciousnessRelatedTo" is bad.
- A "functional" predicate (each Subject has at most one Object) is the exception, not the default. Only set is_functional=true when the relation logically forces uniqueness (e.g., FormedAt: a place has one formation date).
- For Unary tag-style predicates (e.g. IsCognitiveBias), set is_unary=true and leave object_slot empty.

EMPTY IS A FIRST-CLASS ANSWER. If every cluster you can identify is already absorbed by an existing predicate, the correct response is candidates=[]. Do NOT emit a placeholder candidate with "SKIP" or "absorbed by" prose to hedge — that defeats the schema. An empty array means "the predicate vocabulary is currently adequate for the observed gap," which is a useful, valid finding.

=== TRIP-ISOLATED ENTRIES (` + fmt.Sprintf("%d total", len(entries)) + `) ===

`)
	for i, e := range entries {
		fmt.Fprintf(&b, "--- entry %d (score=%d, prompt=%s) ---\n", i+1, e.Score, e.PromptType)
		fmt.Fprintf(&b, "EntityA: %s\nEntityB: %s\n", e.EntityA, e.EntityB)
		fmt.Fprintf(&b, "Connection: %s\n", e.Connection)
		fmt.Fprintf(&b, "Rationale (why no existing predicate fits): %s\n\n", e.Rationale)
	}

	b.WriteString("Call report_candidates with the proposals. Empty array if no cluster reaches the threshold or if the gap is noise-shaped.\n")
	return b.String()
}

// loadExistingPredicates wraps internal/corpusparse so the rest of this
// command can stay unchanged. Keeping the wrapper (instead of inlining
// the call site) makes it easier to add cmd-local filtering later if a
// specific predicate family should be excluded from suggestion prompts.
func loadExistingPredicates(dir string) ([]string, error) {
	return corpusparse.LoadPredicates(dir)
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
