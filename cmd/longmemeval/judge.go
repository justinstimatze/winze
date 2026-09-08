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
//
// Fixed 2026-09-08: the original rubric said "says it doesn't know" is always
// INCORRECT, with no carve-out for an abstention question whose own gold
// answer IS "the information provided is not enough" — a real, if
// inconsistent, source of false negatives (the judge is itself an LLM
// applying the rubric with some independent judgment, not a literal string
// match: measured 8/12 abstention questions passing anyway, but 2 of the 4
// that failed — 30-gallon-tank fish count, Rachel's-wedding-age — were model
// answers that correctly abstained and got marked wrong for it). Also added:
// a gold answer offering more than one acceptable value ("Pilsner or Lager")
// should accept a model naming either one; a real failure (`16c90bf4`) named
// "Pilsner" alone and was marked incorrect.
const judgeSystem = `You are grading whether a model's answer to a question about a user is correct, given the gold answer.

The model answer is CORRECT if it conveys the same information as the gold answer, even if phrased differently or with extra detail.

If the gold answer itself states that the information is not available or not enough was provided, a model answer that reaches the same conclusion — declines to answer, says it doesn't know, or says the information isn't there — is CORRECT. Do not penalize an answer for declining when declining is what the gold answer does too.

If the gold answer offers more than one acceptable value ("X or Y"), a model answer naming any one of them is CORRECT.

Otherwise, it is INCORRECT if it contradicts the gold answer, omits the key information, or says it doesn't know when the gold answer contains real, retrievable information.

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
