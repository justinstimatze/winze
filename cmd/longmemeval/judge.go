package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// judgeSystem mirrors LongMemEval's GPT-4o judging rubric: correctness is
// whether the hypothesis answer conveys the gold answer, not string equality.
// Identical every call → cache_control'd.
const judgeSystem = `You are grading whether a model's answer to a question about a user is correct, given the gold answer.

The model answer is CORRECT if it conveys the same information as the gold answer, even if phrased differently or with extra detail. It is INCORRECT if it contradicts the gold answer, omits the key information, or says it doesn't know.

Respond with exactly one word on the first line: CORRECT or INCORRECT.`

// judge grades one answer against gold. Returns true if correct.
func (r *runner) judge(question, gold, hypothesis string) (bool, error) {
	prompt := fmt.Sprintf("Question: %s\n\nGold answer: %s\n\nModel answer: %s", question, gold, hypothesis)
	resp, err := r.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:       anthropic.ModelClaudeSonnet4_6,
		MaxTokens:   16,
		Temperature: anthropic.Float(0),
		System: []anthropic.TextBlockParam{{
			Text:         judgeSystem,
			CacheControl: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return false, fmt.Errorf("judge API error: %w", err)
	}
	r.stats.record(string(anthropic.ModelClaudeSonnet4_6), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)

	for _, block := range resp.Content {
		if block.Type == "text" {
			return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(block.Text)), "CORRECT"), nil
		}
	}
	return false, fmt.Errorf("no text in judge response")
}
