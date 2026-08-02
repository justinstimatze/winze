package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// answerSystem is the answerer's standing instruction — identical every call,
// so it's cache_control'd. The model answers only from the retrieved facts.
const answerSystem = `You answer a question about a user using ONLY the retrieved memory facts provided. Each fact carries the date it was stated.

Rules:
- Answer concisely and directly — a phrase or short sentence, not an essay.
- For temporal questions, reason over the fact dates (which came first, most recent, etc.).
- If a fact was updated, the most recent stated value wins.
- If the facts do not contain the answer, say exactly: I don't know.
- Do not invent facts beyond what is retrieved.`

// answer asks the reasoner to answer the question from the retrieved facts.
func (r *runner) answer(question, questionDate string, facts []Fact) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Question (asked on %s): %s\n\n", questionDate, question)
	b.WriteString("Retrieved memory facts:\n")
	if len(facts) == 0 {
		b.WriteString("(none)\n")
	}
	for _, f := range facts {
		fmt.Fprintf(&b, "- [%s] %s = %s (%s) — %q\n", f.Date, f.Attribute, f.Value, f.Kind, f.Quote)
	}

	resp, err := r.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{{
			Text:         answerSystem,
			CacheControl: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(b.String())),
		},
	})
	if err != nil {
		return "", fmt.Errorf("answer API error: %w", err)
	}
	r.stats.record(string(anthropic.ModelClaudeSonnet4_6), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)

	for _, block := range resp.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("no text in answer response")
}
