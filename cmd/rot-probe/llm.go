package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// finding is one rot signal surfaced by the probe. Never auto-fixes — the
// human decides whether the finding is real and what to do.
type finding struct {
	Kind       string   `json:"kind"`     // "duplicate" | "contradiction" | "brief_drift" | "trip_attractor"
	Entities   []string `json:"entities"` // var names involved
	Rationale  string   `json:"rationale"`
	Confidence string   `json:"confidence"` // "high" | "medium" | "low"
}

// runProbe builds a prompt over the sampled neighborhoods and asks an LLM
// to flag potential rot. The model is forced to use the report_findings
// tool; output is structured per the schema below.
func runProbe(samples []neighborhood, model string) ([]finding, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set (create .env at the corpus root or export it)")
	}

	modelID := anthropic.ModelClaudeHaiku4_5
	if model == "sonnet" {
		modelID = anthropic.ModelClaudeSonnet4_6
	}

	prompt := buildPrompt(samples)

	schema := anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"findings": map[string]any{
				"type":        "array",
				"description": "Rot signals to surface for human review. Empty if the sample looks clean.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"kind": map[string]any{
							"type":        "string",
							"enum":        []string{"duplicate", "contradiction", "brief_drift", "trip_attractor"},
							"description": "Type of rot. 'duplicate'=two entities likely refer to the same real-world thing; 'contradiction'=claims about an entity that cannot all be true; 'brief_drift'=an entity's Brief no longer matches the substance of its current claims; 'trip_attractor'=an entity has more trip-cycle-generated claims (marked [trip]) than source-grounded claims, and the trip claims attach concepts the Brief does not anticipate — the entity has become a topological magnet rather than an authentically central concept.",
						},
						"entities": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "string"},
							"description": "Var names of entities the finding involves (e.g. ['KlausConrad', 'MichaelShermer']).",
						},
						"rationale": map[string]any{
							"type":        "string",
							"description": "One sentence: what specifically suggests the rot.",
						},
						"confidence": map[string]any{
							"type":        "string",
							"enum":        []string{"high", "medium", "low"},
							"description": "How confident: high=clear, medium=plausible, low=worth a human glance.",
						},
					},
					"required": []string{"kind", "entities", "rationale", "confidence"},
				},
			},
		},
		Required: []string{"findings"},
	}
	tool := anthropic.ToolUnionParamOfTool(schema, "report_findings")

	client := anthropic.NewClient(option.WithAPIKey(key))
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:      modelID,
		MaxTokens:  4096,
		Tools:      []anthropic.ToolUnionParam{tool},
		ToolChoice: anthropic.ToolChoiceParamOfTool("report_findings"),
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
			Findings []finding `json:"findings"`
		}
		if err := json.Unmarshal(tu.Input, &out); err != nil {
			return nil, fmt.Errorf("decode tool input: %w", err)
		}
		return out.Findings, nil
	}
	return nil, fmt.Errorf("no tool_use block in response")
}

func buildPrompt(samples []neighborhood) string {
	var b strings.Builder
	b.WriteString(`You are a rot-probe for a typed knowledge base about the epistemology of minds. You receive a small random sample of entities with their Briefs and the claims that connect them. Your job is to surface signals of rot — places where a human should look — NOT to auto-fix anything.

WHAT TO FLAG:

1. duplicate — two entities that likely refer to the same real-world thing under different names. Look for: same canonical name in different casings, overlapping aliases, identical Briefs, claim graphs that overlap so strongly the two are almost certainly one entity that got entered twice.

2. contradiction — claims about an entity that cannot all be simultaneously true. Note: academic disagreement (different people proposing different theories) is NORMAL and NOT a contradiction. Real contradictions are factual: temporal impossibilities, mutually exclusive states, Briefs that contradict the entity's own claims.

3. brief_drift — the Brief text no longer matches the substance of the entity's current claims. The Brief was written once; subsequent claims may have shifted what's load-bearing about the entity. Drift = "if I had to write the Brief today knowing only these claims, I'd write something materially different."

4. trip_attractor — claims marked [trip] are auto-generated by the metabolism's cross-cluster trip cycle (LLM-speculated structural analogies, NOT primary-source-grounded). Trip claims are first-class but their grounding is the trip-cycle rationale, not external evidence. An entity is a trip_attractor when:
   (a) it has MORE trip-marked claims than non-trip claims, AND
   (b) the trip claims attach concepts the Brief does not anticipate (so the entity's neighborhood is being shaped by structural-analogy pressure rather than evidence ingest).
   This is a topology signal, not a defect — could mean the entity is a real conceptual hub OR a topological accident the trip-cycle prompt over-weights. Either way, worth a human glance.

WHAT NOT TO FLAG:
- Stylistic Brief preferences (it's fine if a Brief could be shorter)
- Missing claims you'd expect to see (rot-probe only sees a sample; absence is uninformative)
- Multiple people holding contrary theories about the same concept (that's contestation, not rot)
- Entities the sample shows once but seems undeveloped (sample is random — that means nothing)

Confidence scale: "high" only when the signal is unambiguous. "medium" for plausible-but-worth-checking. "low" for "I noticed this; a human glance is cheap."

Call report_findings with the list. If the sample looks clean, pass an empty array — empty is a perfectly valid answer.

=== SAMPLE ===
`)
	for _, n := range samples {
		b.WriteString(serializeNeighborhood(n))
		b.WriteString("\n")
	}
	return b.String()
}

// neighborhood is one entity plus the claims it participates in.
type neighborhood struct {
	ent    entity
	asSubj []claim
	asObj  []claim
}

func serializeNeighborhood(n neighborhood) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s (var: %s, role: %s) ---\n", n.ent.name, n.ent.varName, n.ent.roleType)
	if n.ent.brief != "" {
		fmt.Fprintf(&b, "Brief: %s\n", n.ent.brief)
	}
	if len(n.ent.aliases) > 0 {
		fmt.Fprintf(&b, "Aliases: %s\n", strings.Join(n.ent.aliases, ", "))
	}
	if len(n.asSubj) > 0 {
		b.WriteString("As subject:\n")
		for _, c := range n.asSubj {
			tag := claimTag(c)
			if c.objectVar != "" {
				fmt.Fprintf(&b, "  %s --%s--> %s   (claim: %s%s)\n", c.subjectVar, c.predicateType, c.objectVar, c.varName, tag)
			} else {
				fmt.Fprintf(&b, "  %s is %s   (claim: %s%s)\n", c.subjectVar, c.predicateType, c.varName, tag)
			}
		}
	}
	if len(n.asObj) > 0 {
		b.WriteString("As object:\n")
		for _, c := range n.asObj {
			tag := claimTag(c)
			fmt.Fprintf(&b, "  %s --%s--> %s   (claim: %s%s)\n", c.subjectVar, c.predicateType, c.objectVar, c.varName, tag)
		}
	}
	return b.String()
}

// claimTag attaches a [trip] marker to trip-cycle-promoted claims so the
// prompt's WHAT-TO-FLAG description of trip_attractor can be evaluated by
// the LLM against the actual claim provenance.
func claimTag(c claim) string {
	if c.tripGenerated {
		return " [trip]"
	}
	return ""
}

// loadDotEnv mirrors cmd/lint's behaviour: read .env at the corpus root so
// ANTHROPIC_API_KEY can be picked up without exporting it.
func loadDotEnv(dir string) {
	path := filepath.Join(dir, ".env")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
