package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// irrelevance_audit — diagnostic subcommand that re-classifies a sample
// of "irrelevant" cycles under a neutral prompt (no default-irrelevant
// framing) and reports the flip rate.
//
// Motivation: Path A (sensor) sits at ~70% irrelevant-rate. The current
// llmResolve prompt has four separate reinforcements of the "default to
// irrelevant" prior, plus an asymmetric criteria set (corroboration:
// 1 condition, challenge: 4). If that prior is over-calibrated, a
// meaningful fraction of "irrelevant" cycles might actually be
// evidential, and the hit rate would rise without new ingest.
//
// This tool does not mutate .metabolism-log.json. It produces numbers +
// per-cycle snippets so the user can spot-check the flips and decide
// whether to tune the production prompt.

type irrelevanceAuditEntry struct {
	CycleIndex       int      `json:"cycle_index"`
	Hypothesis       string   `json:"hypothesis"`
	Backend          string   `json:"backend"`
	OriginalVerdict  string   `json:"original_verdict"`
	AuditVerdict     string   `json:"audit_verdict"`
	Flipped          bool     `json:"flipped"`
	PaperTitles      []string `json:"paper_titles"`
	SnippetPreview   string   `json:"snippet_preview,omitempty"`
}

type irrelevanceAuditReport struct {
	Sampled          int                     `json:"sampled"`
	TotalIrrelevant  int                     `json:"total_irrelevant"`
	Flipped          int                     `json:"flipped"`
	FlipRate         float64                 `json:"flip_rate"`
	FlipBreakdown    map[string]int          `json:"flip_breakdown"`
	ModelUsed        string                  `json:"model"`
	Entries          []irrelevanceAuditEntry `json:"entries"`
}

// runIrrelevanceAudit samples n "irrelevant" cycles from the log and
// reclassifies them, comparing verdicts.
//
// mode selects which prompt to reclassify with:
//
//	"neutral"    — audit's own neutral prompt (default-irrelevant framing
//	               removed, criteria symmetrized). Diagnostic against any
//	               current production prompt.
//	"production" — the actual llmResolve production prompt. Useful after
//	               a prompt tune to measure whether the change recovers
//	               the flip rate that motivated the tune.
//
// requireSnippet=true restricts sampling to cycles whose papers carry
// at least one non-empty snippet — the prompt is only meaningful when
// the LLM has paper content beyond titles.
func runIrrelevanceAudit(dir string, n int, jsonOut bool, useHaiku, requireSnippet bool, mode string) {
	if n <= 0 {
		n = 10
	}
	if mode != "neutral" && mode != "production" {
		fmt.Fprintf(os.Stderr, "metabolism: --audit-mode must be 'neutral' or 'production' (got %q)\n", mode)
		os.Exit(1)
	}
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	var irrelevantIdx []int
	for i, c := range mlog.Cycles {
		if c.Resolution != "irrelevant" || len(c.Papers) == 0 {
			continue
		}
		if requireSnippet && !anyPaperHasSnippet(c.Papers) {
			continue
		}
		irrelevantIdx = append(irrelevantIdx, i)
	}
	if len(irrelevantIdx) == 0 {
		fmt.Println("[irrelevance-audit] no irrelevant cycles with papers in log")
		return
	}

	// Deterministic sample: evenly spaced indices across the irrelevant
	// population. Reproducible across runs; no RNG seed to remember.
	sample := pickSpacedIndices(irrelevantIdx, n)

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "[irrelevance-audit] ANTHROPIC_API_KEY not set — cannot reclassify")
		os.Exit(1)
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	model := anthropic.ModelClaudeSonnet4_5
	modelName := "sonnet-4-5"
	if useHaiku {
		model = anthropic.ModelClaudeHaiku4_5
		modelName = "haiku-4-5"
	}

	report := irrelevanceAuditReport{
		Sampled:         len(sample),
		TotalIrrelevant: len(irrelevantIdx),
		FlipBreakdown:   map[string]int{},
		ModelUsed:       modelName + "/" + mode,
	}

	for _, ci := range sample {
		c := mlog.Cycles[ci]
		verdict, err := reclassifyUnderMode(client, model, c, mode)
		entry := irrelevanceAuditEntry{
			CycleIndex:      ci,
			Hypothesis:      c.Hypothesis,
			Backend:         c.Backend,
			OriginalVerdict: c.Resolution,
			AuditVerdict:    verdict,
		}
		for _, p := range c.Papers {
			entry.PaperTitles = append(entry.PaperTitles, p.Title)
		}
		if len(c.Papers) > 0 && c.Papers[0].Snippet != "" {
			snip := c.Papers[0].Snippet
			if len(snip) > 240 {
				snip = snip[:240] + "…"
			}
			entry.SnippetPreview = snip
		}
		if err != nil {
			entry.AuditVerdict = "error:" + err.Error()
		} else if verdict != "" && verdict != "irrelevant" {
			entry.Flipped = true
			report.Flipped++
			report.FlipBreakdown[verdict]++
		}
		report.Entries = append(report.Entries, entry)
	}
	if report.Sampled > 0 {
		report.FlipRate = float64(report.Flipped) / float64(report.Sampled) * 100
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}
	emitIrrelevanceAuditText(report)
}

// reclassifyUnderMode dispatches to the neutral reclassifier or the
// production llmResolve call based on mode.
func reclassifyUnderMode(client anthropic.Client, model anthropic.Model, c Cycle, mode string) (string, error) {
	if mode == "production" {
		// llmResolve is hard-coded to Sonnet internally. For cheap
		// auditing we ideally reuse the same prompt at the audit's
		// chosen model (Haiku by default); but respecting llmResolve's
		// exact config — including model — gives the most faithful
		// readout of "what production will actually do." Accept the
		// higher cost as a tradeoff.
		_ = model
		return llmResolve(client, c.Hypothesis, lookupBrief(c.Hypothesis), c.Papers)
	}
	return reclassifyNeutral(client, model, c)
}

// reclassifyNeutral runs llmResolve-equivalent classification but with
// the default-irrelevant framing removed. Same three labels, same
// extraction logic — just a neutral prior so we can measure whether
// the production prompt's default is over-calibrated.
func reclassifyNeutral(client anthropic.Client, model anthropic.Model, c Cycle) (string, error) {
	var sources []string
	for _, p := range c.Papers {
		desc := sanitizeText(p.Title, 200)
		if p.Snippet != "" {
			desc += "\n  Content: " + sanitizeText(p.Snippet, 500)
		}
		sources = append(sources, desc)
	}

	// Neutral prompt: no "DEFAULT to irrelevant" framing, no closing
	// "If neither — classify as irrelevant" instruction. Criteria are
	// retained but symmetrized so corroboration and challenge get
	// equivalent treatment. The final label set is identical so flip
	// rates are directly comparable to the production prompt.
	prompt := fmt.Sprintf(`Classify whether the sources below provide evidence about the hypothesis. Weigh the evidence on its merits.

Hypothesis: %s
Brief: %s

Sources:
- %s

Labels:
- "corroborated" — sources contain specific evidence, data, or arguments that support the hypothesis's central claim (beyond merely discussing the same topic).
- "challenged" — sources contain specific evidence, data, or arguments that contradict the hypothesis, present a competing account, or undermine its evidentiary basis.
- "irrelevant" — sources discuss the same topic area but do not provide evidence bearing on whether this hypothesis is true or false.

Consider specific facts: dates, numbers, attributions, experimental results. Weigh evidence for and against with equal rigor.

Think step by step:
1. What specific claim does the hypothesis make?
2. Do the sources contain evidence that supports, contradicts, or is neutral on that claim?
3. If the evidence is substantive (not just keyword overlap), classify as corroborated or challenged. If it is keyword overlap only, classify as irrelevant.

State your final classification: irrelevant, corroborated, or challenged.`,
		c.Hypothesis, lookupBrief(c.Hypothesis), strings.Join(sources, "\n- "))

	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", err
	}
	recordActualUsage(string(model), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)
	raw := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			raw = strings.TrimSpace(strings.ToLower(block.Text))
		}
	}
	return extractClassification(raw)
}

// lookupBrief reads the corpus to find the Brief for a given Hypothesis
// var name. Returns empty string if not found — the prompt tolerates a
// missing Brief (the Hypothesis name alone is usually enough).
func lookupBrief(hypName string) string {
	// Simple grep-style scan. Production llmResolve gets Brief from the
	// caller; this audit runs post-hoc from the log, which doesn't
	// preserve the Brief. Reading it here keeps the prompt comparable.
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			continue
		}
		s := string(data)
		needle := "var " + hypName + " "
		idx := strings.Index(s, needle)
		if idx < 0 {
			continue
		}
		// Find Brief: in the composite literal
		briefIdx := strings.Index(s[idx:], "Brief:")
		if briefIdx < 0 {
			continue
		}
		rest := s[idx+briefIdx+len("Brief:"):]
		// Skip to opening quote
		qs := strings.Index(rest, "\"")
		if qs < 0 {
			continue
		}
		rest = rest[qs+1:]
		qe := strings.Index(rest, "\"")
		if qe < 0 {
			continue
		}
		return rest[:qe]
	}
	return ""
}

func sanitizeText(s string, maxLen int) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, s)
	if len(cleaned) > maxLen {
		cleaned = cleaned[:maxLen]
	}
	return cleaned
}

// anyPaperHasSnippet returns true if at least one paper has a non-empty
// Snippet field. Used to filter cycles that can actually be reclassified
// on substance rather than title alone.
func anyPaperHasSnippet(papers []PaperSummary) bool {
	for _, p := range papers {
		if strings.TrimSpace(p.Snippet) != "" {
			return true
		}
	}
	return false
}

// pickSpacedIndices returns up to n evenly-spaced elements from idx.
// Deterministic — no RNG. If n >= len(idx), returns all.
func pickSpacedIndices(idx []int, n int) []int {
	if n >= len(idx) {
		out := make([]int, len(idx))
		copy(out, idx)
		return out
	}
	out := make([]int, 0, n)
	step := float64(len(idx)) / float64(n)
	for i := 0; i < n; i++ {
		pos := int(float64(i) * step)
		if pos >= len(idx) {
			pos = len(idx) - 1
		}
		out = append(out, idx[pos])
	}
	return out
}

func emitIrrelevanceAuditText(r irrelevanceAuditReport) {
	fmt.Printf("[irrelevance-audit] model=%s, sampled %d of %d irrelevant cycles\n\n",
		r.ModelUsed, r.Sampled, r.TotalIrrelevant)
	for _, e := range r.Entries {
		marker := "  "
		if e.Flipped {
			marker = "→ "
		}
		fmt.Printf("%scycle %d [%s] %s\n", marker, e.CycleIndex, e.Backend, e.Hypothesis)
		fmt.Printf("    original: %s    audit: %s", e.OriginalVerdict, e.AuditVerdict)
		if e.Flipped {
			fmt.Printf("    *FLIP*")
		}
		fmt.Println()
		if len(e.PaperTitles) > 0 {
			titles := e.PaperTitles
			if len(titles) > 3 {
				titles = append(titles[:3], fmt.Sprintf("(+%d more)", len(e.PaperTitles)-3))
			}
			fmt.Printf("    papers: %s\n", strings.Join(titles, "; "))
		}
		if e.SnippetPreview != "" {
			fmt.Printf("    snippet: %s\n", e.SnippetPreview)
		}
	}
	fmt.Println()
	fmt.Printf("[irrelevance-audit] flip rate: %.0f%% (%d/%d)\n", r.FlipRate, r.Flipped, r.Sampled)
	if len(r.FlipBreakdown) > 0 {
		keys := make([]string, 0, len(r.FlipBreakdown))
		for k := range r.FlipBreakdown {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Print("[irrelevance-audit] flipped verdicts: ")
		for i, k := range keys {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s=%d", k, r.FlipBreakdown[k])
		}
		fmt.Println()
	}
	if r.Flipped > 0 {
		fmt.Println("[irrelevance-audit] meaningful flip rate — production prompt may be over-strict")
	} else {
		fmt.Println("[irrelevance-audit] zero flips — production prompt appears well-calibrated")
	}
}
