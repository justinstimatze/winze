package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// factExtractionSystemPrompt is identical across every call in a replay, so it
// carries the ephemeral cache_control marker: the first call pays full price,
// every later one in the same run pays roughly 10% for this block, per
// CLAUDE.md's standing prompt-caching guidance.
const factExtractionSystemPrompt = `You split a work-session transcript excerpt into a short list of atomic, self-contained facts worth remembering later.

Rules:
- Each fact is one sentence, understandable with no other context.
- Extract concrete specifics: a decision made, a project or tool named, a preference stated, a specific technical fact learned or fixed. Not a vague gist of the topic.
- 3 to 6 facts. Fewer if the excerpt genuinely holds fewer distinct facts -- do not pad.
- Do not include facts about note-taking, memory, or this task itself.
- Respond with ONLY a JSON array of strings, no other text.`

// extractFacts calls Haiku once to split a session excerpt into atomic facts.
// A truncated response is an error, not a partial success -- same posture as
// callIngestLLM in cmd/metabolism: a cut mid-array is refused rather than
// salvaged. Parsing itself is parseFactsResponse, which is unit-tested
// without a live call.
//
// Returns usage alongside the facts (rather than logging it internally) so
// the caller can accumulate a real total across the whole replay instead of
// each call reporting its own number in isolation -- the total is what's
// worth citing, not the per-call figure.
func extractFacts(client anthropic.Client, s *transcriptSession) ([]string, anthropic.Usage, error) {
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: factExtractionSystemPrompt, CacheControl: anthropic.CacheControlEphemeralParam{Type: "ephemeral"}},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(sessionExcerpt(s))),
		},
	})
	if err != nil {
		return nil, anthropic.Usage{}, fmt.Errorf("API error: %w", err)
	}
	if resp.StopReason == "max_tokens" {
		return nil, resp.Usage, fmt.Errorf("fact extraction truncated at %d output tokens", resp.Usage.OutputTokens)
	}
	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	facts, err := parseFactsResponse(text)
	return facts, resp.Usage, err
}

// newAnthropicClientFromEnv is the one place this package reads
// ANTHROPIC_API_KEY, so every caller shares the same skip-if-absent posture
// used throughout cmd/metabolism's live-API tests.
func newAnthropicClientFromEnv() (anthropic.Client, bool) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return anthropic.Client{}, false
	}
	return anthropic.NewClient(option.WithAPIKey(key)), true
}

// sessionExcerpt is the same input noteFor's "arc" shape already builds --
// title, opening ask, later asks minus the held-out probe -- just handed to
// an extractor instead of concatenated into one blob.
func sessionExcerpt(s *transcriptSession) string {
	ask := s.OpeningAsk()
	if len(ask) > 1200 {
		ask = ask[:1200] + "…"
	}
	excerpt := fmt.Sprintf("Session %s: %s\n\nOpened with: %s", s.Start.Format("2006-01-02"), s.Title, ask)
	held := s.LaterAsk()
	var arc []string
	budget := 2000
	for _, a := range s.ArcAsks() {
		if a == held || budget <= 0 {
			continue
		}
		if len(a) > 300 {
			a = a[:300] + "…"
		}
		arc = append(arc, a)
		budget -= len(a)
	}
	if len(arc) > 0 {
		excerpt += "\n\nWent on to: " + strings.Join(arc, " / ")
	}
	return excerpt
}

// parseFactsResponse pulls the JSON string array out of a model response,
// tolerating a markdown code fence around it -- pulled out of extractFacts so
// the parsing itself is testable without a live API call, the same reasoning
// rrfFuse is kept pure and separate from runHybrid.
func parseFactsResponse(text string) ([]string, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	var facts []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &facts); err != nil {
		return nil, fmt.Errorf("unparseable fact list: %w\nraw: %s", err, text)
	}
	if len(facts) == 0 {
		return nil, fmt.Errorf("zero facts extracted")
	}
	return facts, nil
}
