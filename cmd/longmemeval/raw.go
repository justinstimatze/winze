package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// answerRaw is the control: the same answerer and the same judge, handed the
// chat history verbatim instead of winze's typed facts.
//
// This should have existed before any of the other numbers did. The discipline
// the whole project runs on says a typed substrate has to beat handing the same
// model the raw content, or the typing is ceremony — and cmd/longmemeval has
// been reporting winze-against-nothing since it was written. On the oracle set
// that omission is at its most dangerous, because only the evidence sessions
// are present, a mean of 1.9 per question: there is almost nothing to retrieve
// wrongly, so a control that skips retrieval entirely may well match the
// pipeline. If it does, an 85% is a fact about the dataset and not about winze.
//
// Deliberately kept identical to the real path everywhere it can be:
// renderSession does the truncation, answerSystem is the same instruction,
// Sonnet at temperature 0, and the same judge scores it. The only variable is
// what sits between the sessions and the answerer.
func (r *runner) answerRaw(q Question) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Question (asked on %s): %s\n\n", q.QuestionDate, q.Question)
	b.WriteString("Chat history:\n")
	for i, sess := range q.HaystackSessions {
		date := ""
		if i < len(q.HaystackDates) {
			date = q.HaystackDates[i]
		}
		fmt.Fprintf(&b, "\n--- session %d (%s) ---\n%s\n", i+1, date, renderSession(sess))
	}

	resp, err := r.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 512,
		// The raw history is far larger than a fact list, so unlike the
		// answerer's cached system block this content genuinely dominates the
		// bill. That asymmetry is part of the comparison and not a defect: the
		// control trades input tokens for the whole extraction pipeline.
		Temperature: anthropic.Float(0),
		System: []anthropic.TextBlockParam{{
			Text:         answerSystem,
			CacheControl: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(b.String())),
		},
	})
	if err != nil {
		return "", fmt.Errorf("raw answer API error: %w", err)
	}
	r.stats.record(string(anthropic.ModelClaudeSonnet4_6), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)

	for _, block := range resp.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("no text in raw answer response")
}

// runQuestionRaw runs the control path for one question: no lens, no typed
// store, no build gate, no defn sync, no ranking. Sessions in, answer out,
// same judge.
//
// The timing columns it fills are the honest shape of the comparison — extract,
// build, sync and retrieve are all zero because none of them happen, and the
// answer hop absorbs the whole cost as input tokens. facts is zero for the same
// reason, so the zero-fact report block would flag all sixty; that block reads
// the pipeline's own health and does not apply here.
func (r *runner) runQuestionRaw(q Question) (resultRow, error) {
	row := resultRow{qid: q.QuestionID, qtype: q.QuestionType, sessions: len(q.HaystackSessions)}

	tAns := nowNS()
	ans, err := r.answerRaw(q)
	if err != nil {
		return row, err
	}
	row.answerNS = nowNS() - tAns
	row.answer = ans

	tJudge := nowNS()
	correct, err := r.judge(q.Question, string(q.Answer), ans)
	if err != nil {
		return row, err
	}
	row.judgeNS = nowNS() - tJudge
	row.correct = correct
	return row, nil
}
