package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/justinstimatze/winze/internal/astutil"
)

type llmBudget struct {
	enabled        bool
	model          string
	maxCallsPerRun int
	maxTokens      int
}

type llmFinding struct {
	claimA      string
	claimB      string
	explanation string
}

// collectBriefs walks .go files and extracts Brief fields from entity
// composite literals. Entities have the shape RoleType{&Entity{Brief: "..."}}.
func collectBriefs(dir string) (map[string]string, error) {
	briefs := map[string]string{}
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, nameIdent := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					brief := extractBrief(cl)
					if brief != "" {
						briefs[nameIdent.Name] = brief
					}
				}
			}
		}
	}
	return briefs, nil
}

// extractBrief finds the Brief field from an entity composite literal.
// Handles: RoleType{&Entity{Brief: "..."}} where the first positional
// element is a &Entity{...} unary expression.
func extractBrief(cl *ast.CompositeLit) string {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if ok {
			key, ok := kv.Key.(*ast.Ident)
			if ok && key.Name == "Brief" {
				return basicLitString(kv.Value)
			}
			continue
		}
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, innerElt := range inner.Elts {
			kv, ok := innerElt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if key.Name == "Brief" {
				return basicLitString(kv.Value)
			}
		}
	}
	return ""
}

func resolveStringExpr(e ast.Expr) string { return astutil.ResolveStringExpr(e) }
func basicLitString(e ast.Expr) string     { return astutil.ResolveStringExpr(e) }

type neighborhood struct {
	entity entitySite
	brief  string
	asSubj []claimSite
	asObj  []claimSite
	hash   string
}

func buildNeighborhoods(entities []entitySite, claims []claimSite, briefs map[string]string) []neighborhood {
	subjMap := map[string][]claimSite{}
	objMap := map[string][]claimSite{}
	for _, c := range claims {
		subjMap[c.subject] = append(subjMap[c.subject], c)
		objMap[c.object] = append(objMap[c.object], c)
	}

	seen := map[string]bool{}
	var out []neighborhood
	for _, e := range entities {
		asSubj := subjMap[e.name]
		asObj := objMap[e.name]
		if len(asSubj) == 0 && len(asObj) == 0 {
			continue
		}
		h := neighborhoodHash(e.name, asSubj, asObj)
		if seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, neighborhood{
			entity: e,
			brief:  briefs[e.name],
			asSubj: asSubj,
			asObj:  asObj,
			hash:   h,
		})
	}
	return out
}

func neighborhoodHash(name string, asSubj, asObj []claimSite) string {
	h := sha256.New()
	h.Write([]byte(name))
	for _, c := range asSubj {
		h.Write([]byte(c.name))
	}
	for _, c := range asObj {
		h.Write([]byte(c.name))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func serializeNeighborhood(n neighborhood, briefs map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== Entity: %s (%s) ===\n", n.entity.name, n.entity.roleType)
	if n.brief != "" {
		fmt.Fprintf(&b, "Brief: %s\n", n.brief)
	}
	b.WriteString("\n")

	if len(n.asSubj) > 0 {
		b.WriteString("Claims where this entity is the subject:\n")
		for _, c := range n.asSubj {
			fmt.Fprintf(&b, "  [%s] %s --%s--> %s\n", c.name, c.subject, c.predicateType, c.object)
			if ob, ok := briefs[c.object]; ok {
				fmt.Fprintf(&b, "    (%s brief: %s)\n", c.object, truncate(ob, 120))
			}
		}
		b.WriteString("\n")
	}

	if len(n.asObj) > 0 {
		b.WriteString("Claims where this entity is the object:\n")
		for _, c := range n.asObj {
			fmt.Fprintf(&b, "  [%s] %s --%s--> %s\n", c.name, c.subject, c.predicateType, c.object)
			if sb, ok := briefs[c.subject]; ok {
				fmt.Fprintf(&b, "    (%s brief: %s)\n", c.subject, truncate(sb, 120))
			}
		}
	}

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func buildPrompt(serialized string, suppressed map[claimKey]string) string {
	var b strings.Builder
	b.WriteString(`You are a consistency checker for a knowledge base. You will receive a set of claims about an entity. Your job is to identify claims that CANNOT all be simultaneously true.

DO NOT flag these as contradictions:
- Academic disagreement: Multiple people proposing different theories about the same topic is normal scholarship. "A proposes X" and "B proposes Y" about the same concept are NOT contradictory.
- Multiple valid relations: An entity can have multiple instances of the same predicate (authored multiple books, belongs to multiple categories).
- Influence and disagreement: A person can be influenced by someone AND dispute their theories. Students regularly disagree with mentors. InfluencedBy and Disputes/Refutes on related entities is NOT a contradiction.
- Claims tagged as fictional: these exist within a story's frame and do not contradict real-world claims.
`)

	if len(suppressed) > 0 {
		b.WriteString("- Known disputes (intentionally recorded, do NOT flag):\n")
		keys := make([]claimKey, 0, len(suppressed))
		for k := range suppressed {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].predicateType != keys[j].predicateType {
				return keys[i].predicateType < keys[j].predicateType
			}
			return keys[i].subject < keys[j].subject
		})
		for _, k := range keys {
			fmt.Fprintf(&b, "    %s on %s: %s\n", k.predicateType, k.subject, suppressed[k])
		}
	}

	b.WriteString(`
WHAT TO FLAG as contradictions:
- Claims using DIFFERENT predicate types that semantically conflict (e.g., "X trusts Y" alongside "X distrusts Y")
- An entity's Brief text contradicting specific claims about it
- Temporal impossibilities (e.g., a person born after an event they supposedly led)
- Logical impossibilities across claims

Call report_contradictions with the list of contradictions found. If none, pass an empty array.

=== CLAIMS TO CHECK ===
`)
	b.WriteString(serialized)
	return b.String()
}

// callLLMForContradictions invokes the API with a forced report_contradictions
// tool and returns the structured findings. Schema enforces var-name pairs +
// explanation per finding — no fragile text parsing.
func callLLMForContradictions(prompt string, budget llmBudget) ([]llmFinding, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set (create .env or export it)")
	}

	model := anthropic.ModelClaudeHaiku4_5
	if budget.model == "sonnet" {
		model = anthropic.ModelClaudeSonnet4_6
	}

	schema := anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"contradictions": map[string]any{
				"type":        "array",
				"description": "List of contradictions found. Empty if none.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"claim_a": map[string]any{
							"type":        "string",
							"description": "Variable name of the first claim (e.g. SmithDisputesJonesHypothesis).",
						},
						"claim_b": map[string]any{
							"type":        "string",
							"description": "Variable name of the second claim.",
						},
						"explanation": map[string]any{
							"type":        "string",
							"description": "One sentence explaining the contradiction.",
						},
					},
					"required": []string{"claim_a", "claim_b", "explanation"},
				},
			},
		},
		Required: []string{"contradictions"},
	}
	tool := anthropic.ToolUnionParamOfTool(schema, "report_contradictions")

	client := anthropic.NewClient(option.WithAPIKey(key))
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:      model,
		MaxTokens:  int64(budget.maxTokens),
		Tools:      []anthropic.ToolUnionParam{tool},
		ToolChoice: anthropic.ToolChoiceParamOfTool("report_contradictions"),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}

	for _, block := range resp.Content {
		if block.Type != "tool_use" {
			continue
		}
		tu := block.AsToolUse()
		var out struct {
			Contradictions []struct {
				ClaimA      string `json:"claim_a"`
				ClaimB      string `json:"claim_b"`
				Explanation string `json:"explanation"`
			} `json:"contradictions"`
		}
		if err := json.Unmarshal(tu.Input, &out); err != nil {
			return nil, fmt.Errorf("decode tool input: %w", err)
		}
		findings := make([]llmFinding, 0, len(out.Contradictions))
		for _, c := range out.Contradictions {
			findings = append(findings, llmFinding{
				claimA:      c.ClaimA,
				claimB:      c.ClaimB,
				explanation: c.Explanation,
			})
		}
		return findings, nil
	}
	return nil, fmt.Errorf("no tool_use block in response")
}

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
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func llmContradictionRule(dir string, budget llmBudget) int {
	if !budget.enabled {
		fmt.Println("[llm-contradiction] skipped (use --llm to enable)")
		return 0
	}

	loadDotEnv(dir)
	loadDotEnv(".")

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println("[llm-contradiction] error: ANTHROPIC_API_KEY not set")
		return 0
	}

	roles, err := collectRoleTypes(dir)
	if err != nil {
		fmt.Printf("[llm-contradiction] error collecting role types: %v\n", err)
		return 0
	}
	roleSet := map[string]bool{}
	for _, r := range roles {
		roleSet[r.name] = true
	}

	entities, err := collectEntityVars(dir, roleSet)
	if err != nil {
		fmt.Printf("[llm-contradiction] error collecting entities: %v\n", err)
		return 0
	}

	allClaims, _, _, _, suppressed, err := collectClaims(dir)
	if err != nil {
		fmt.Printf("[llm-contradiction] error collecting claims: %v\n", err)
		return 0
	}

	briefs, err := collectBriefs(dir)
	if err != nil {
		fmt.Printf("[llm-contradiction] error collecting briefs: %v\n", err)
		return 0
	}

	neighborhoods := buildNeighborhoods(entities, allClaims, briefs)

	var allFindings []struct {
		entity   entitySite
		findings []llmFinding
	}
	callCount := 0
	errCount := 0

	for _, n := range neighborhoods {
		if budget.maxCallsPerRun > 0 && callCount >= budget.maxCallsPerRun {
			break
		}

		serialized := serializeNeighborhood(n, briefs)
		prompt := buildPrompt(serialized, suppressed)

		findings, err := callLLMForContradictions(prompt, budget)
		callCount++
		if err != nil {
			errCount++
			continue
		}

		if len(findings) > 0 {
			allFindings = append(allFindings, struct {
				entity   entitySite
				findings []llmFinding
			}{entity: n.entity, findings: findings})
		}
	}

	totalFindings := 0
	for _, f := range allFindings {
		totalFindings += len(f.findings)
	}

	fmt.Printf("[llm-contradiction] %d entities, %d neighborhoods, %d findings, %d errors (model=%s, calls=%d)\n",
		len(entities), len(neighborhoods), totalFindings, errCount, budget.model, callCount)

	if errCount > 0 && errCount == callCount {
		fmt.Println("[llm-contradiction] WARNING: all LLM calls failed — contradiction check inconclusive")
	}

	if len(allFindings) > 0 {
		fmt.Println("  findings:")
		for _, f := range allFindings {
			fmt.Printf("    %s (%s):\n", f.entity.name, f.entity.roleType)
			for _, finding := range f.findings {
				fmt.Printf("      %s vs %s: %s\n", finding.claimA, finding.claimB, finding.explanation)
			}
		}
	}

	return 0
}
