package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// factRerankSystem asks the model to reorder facts by relevance to a
// question. Mirrors cmd/query's rerankSystemPrompt -- winze_recall's already-
// shipped mechanism, measured there at LATER-PROBE hit@5 14%->51% -- adapted
// from ranking entities to ranking facts. One call per question, and
// rerankFacts fails open to term-overlap ranking on any error, so a rerank
// outage never breaks a run.
const factRerankSystem = `You rerank memory facts by relevance to a question.

Given a question and a numbered list of facts (attribute: value), return a JSON array of fact ids ordered from most to least relevant to answering the question. Include every id exactly once. Never invent an id that wasn't in the list.

Respond with ONLY a JSON array of integers, no other text.`

// rerankCap bounds how many candidates one rerank call scores, keeping the
// prompt small and the cost predictable regardless of how many facts a
// session's extraction produced. Above the cap, term overlap prefilters down
// to this many first -- the rerank step never sees fewer genuinely-relevant
// candidates than plain rankFacts would have kept at k=120.
//
// Sized for the oracle set (45-90 facts/question, never near this cap) and
// never re-checked against the full longmemeval_s haystack until 2026-09-09:
// a 30-question stratified full-haystack sample had 623-1120 facts/question,
// so the 200 cap bound EVERY question -- term overlap made the real
// candidate-inclusion decision before rerank ever ran, and the sample tied
// term overlap exactly, 0 of 30 verdicts differing.
//
// Raised to 1500 same day to test that directly and reverted: on the same
// warm-cache sample, 30/30 -> 17/30 (76.7% -> 56.7%), knowledge-update alone
// going 4/5 -> 0/5. Not a truncation artifact -- callFactRerank's own
// MaxTokens guard returns an explicit error (fail-open to term overlap) on
// a truncated response, and the pre-existing truncated-extraction cohort (8
// of 30 questions, present identically in every run) only accounts for 2 of
// the 6 net regressions. The rest is Haiku's own ranking quality degrading
// as the candidate list grows from 200 to up to 1500 items in one call --
// more candidates is not free just because the context window fits them.
//
// Also tried at 500 (2026-09-09): 23/30, an exact tie with 200 in aggregate
// but a different composition -- fixed one real needle-in-haystack case
// (a preference fact mentioned once in 900+ facts, buried under 200 but
// visible under 500) while breaking one unrelated single-session-user
// question through reordering elsewhere (a real content change, not judge
// noise -- checked the actual answer text on both sides). Net zero on n=30
// is not evidence 500 is better than 200, just evidence the mechanism is
// real in both directions. See ROADMAP.md for the full per-question
// mechanism reading (needle-in-haystack retrieval-volume gaps vs. genuine
// relevance-judgment gaps vs. extraction gaps -- three different causes
// hiding under one "1/5 preference" number, only one of which any cap
// change can touch). Left at 200 -- no cap value tried tonight is a
// confirmed net improvement over it.
const rerankCap = 200

// parseFactRerankResponse pulls the JSON id list out of a model response,
// tolerating a markdown code fence. An id not in validIDs is dropped rather
// than erroring the whole call -- rerankFacts already has a correct fallback
// (original order) for any id the model omits or hallucinates.
func parseFactRerankResponse(text string, validIDs []int) ([]int, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	var order []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &order); err != nil {
		return nil, fmt.Errorf("unparseable rerank order: %w\nraw: %s", err, text)
	}
	valid := make(map[int]bool, len(validIDs))
	for _, id := range validIDs {
		valid[id] = true
	}
	filtered := order[:0]
	for _, id := range order {
		if valid[id] {
			filtered = append(filtered, id)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("empty rerank order")
	}
	return filtered, nil
}

// callFactRerank makes one Haiku call scoring relevance across facts.
// Mirrors cmd/query's callRerank shape and cache-effect visibility.
func (r *runner) callFactRerank(question string, facts []Fact) ([]int, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\nFacts:\n", question)
	validIDs := make([]int, len(facts))
	for i, f := range facts {
		fmt.Fprintf(&b, "%d. %s: %s\n", i, f.Attribute, f.Value)
		validIDs[i] = i
	}

	resp, err := r.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:       anthropic.ModelClaudeHaiku4_5,
		MaxTokens:   2048,
		Temperature: anthropic.Float(0),
		System: []anthropic.TextBlockParam{{
			Text:         factRerankSystem,
			CacheControl: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(b.String())),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("rerank API error: %w", err)
	}
	r.stats.record(string(anthropic.ModelClaudeHaiku4_5), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)
	if resp.StopReason == "max_tokens" {
		return nil, fmt.Errorf("rerank response truncated at %d output tokens", resp.Usage.OutputTokens)
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	return parseFactRerankResponse(text, validIDs)
}

// rerankFacts reorders facts by an LLM's judgment of relevance instead of
// literal term overlap (rankFacts), then returns the top k. It reuses the
// already-extracted, already-cached fact set -- this changes nothing
// upstream of retrieval, so on a warm extraction cache it costs one small
// Haiku call per question and zero re-extraction. term overlap can score a
// genuinely relevant fact at exactly 0 on a plural/singular or synonym
// mismatch (confirmed on bf659f65: "eps" vs "ep"); an LLM judging relevance
// directly doesn't have that specific failure mode.
func (r *runner) rerankFacts(facts []Fact, question string, k int) []Fact {
	pool := facts
	if len(pool) > rerankCap {
		pool = rankFacts(facts, question, rerankCap)
	}
	order, err := r.callFactRerank(question, pool)
	if err != nil {
		return rankFacts(facts, question, k) // fail open, term-overlap fallback
	}
	byID := make(map[int]Fact, len(pool))
	for i, f := range pool {
		byID[i] = f
	}
	seen := make(map[int]bool, len(pool))
	out := make([]Fact, 0, k)
	for _, id := range order {
		if f, ok := byID[id]; ok && !seen[id] {
			out = append(out, f)
			seen[id] = true
			if len(out) == k {
				return out
			}
		}
	}
	for i, f := range pool { // ids the model omitted -- append after, original order preserved
		if !seen[i] {
			out = append(out, f)
			if len(out) == k {
				break
			}
		}
	}
	return out
}
