package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/justinstimatze/winze/internal/cliutil"
)

const rerankMaxBriefBytes = 300

const rerankSystemPrompt = `You rerank search candidates by relevance to a query.

Given a query and a numbered list of candidates (name and brief description), return a JSON array of candidate ids ordered from most to least relevant to the query. Include every id exactly once. Never invent an id that wasn't in the list.

Respond with ONLY a JSON array of integers, no other text.`

// callRerank makes one Haiku call scoring relevance across candidates.
// Mirrors generateSurfaceForms's shape, plus the timeout checkOneContradiction
// uses (generateSurfaceForms itself has none -- fine on a cached write-time
// path, not fine here) and cache-effect visibility so a silently-uncached
// system prompt is caught rather than assumed away.
func callRerank(client anthropic.Client, model anthropic.Model, query string, candidates []rerankCandidate) ([]int, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Query: %s\n\nCandidates:\n", query)
	validIDs := make([]int, len(candidates))
	for i, c := range candidates {
		fmt.Fprintf(&b, "%d. %s: %s\n", c.id, c.name, cliutil.Truncate(c.brief, rerankMaxBriefBytes))
		validIDs[i] = c.id
	}

	ctx, cancel := context.WithTimeout(context.Background(), rerankTimeout())
	defer cancel()
	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{
			{Text: rerankSystemPrompt, CacheControl: anthropic.CacheControlEphemeralParam{Type: "ephemeral"}},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(b.String())),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[rerank cache] written=%d read=%d fresh=%d\n",
		resp.Usage.CacheCreationInputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.InputTokens)
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
	return parseRerankResponse(text, validIDs)
}

// parseRerankResponse pulls the JSON id list out of a model response,
// tolerating a markdown code fence -- mirrors parseSurfaceFormsResponse.
// Unlike that function, a returned id not in validIDs is a data-quality
// issue, not a fatal one: it's silently dropped rather than erroring the
// whole call, since the caller (rerankTop) already has a correct fallback
// for any id the model omits or gets wrong.
func parseRerankResponse(text string, validIDs []int) ([]int, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	var order []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &order); err != nil {
		return nil, fmt.Errorf("unparseable rerank order: %w\nraw: %s", err, text)
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("empty rerank order")
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
	return filtered, nil
}

// rerankEnabled gates the whole feature -- off by default, matching every
// other retrieval knob in this package (WINZE_EMBED_MODEL, WINZE_SURFACE_FORMS).
func rerankEnabled() bool { return os.Getenv("WINZE_RERANK") != "" }

// rerankFused is the thin production wrapper: resolves gating, the API key
// (loadDotEnv, mirroring surfaceFormsFor), and the client, then delegates to
// rerankTop. force bypasses the WINZE_RERANK env check for a single
// invocation -- the scoping fix for cmd/agent's --hybrid callers: handleRecall
// (winze_recall) passes --rerank explicitly, while currentBrief's identity
// lookup (called from the write-path handleUpdate) does not, so a
// process-wide env var is never the only thing standing between "read path
// reranks" and "every write pays an LLM call too" -- both share the same
// winze-query subprocess environment via runQueryRaw, which has no per-call
// env override. Returns fused unchanged whenever the feature is off or
// unusable -- callers need no branching.
func rerankFused(dir string, fused []fusedHit, kb *kbIndex, query string, force bool) []fusedHit {
	if !rerankShouldRun(force) {
		return fused
	}
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		loadDotEnv(dir)
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return fused
	}
	client := anthropic.NewClient(option.WithAPIKey(key))
	model := rerankModel()
	return rerankTop(fused, kb, query, rerankTopK(), func(q string, cands []rerankCandidate) ([]int, error) {
		return callRerank(client, model, q, cands)
	})
}

// rerankModel is an env knob, not hardcoded -- the corpus stays small on
// purpose, so a Sonnet call over a bounded candidate window is still cheap;
// no reason to lock in Haiku if measurement shows it isn't ranking well
// enough.
func rerankModel() anthropic.Model {
	if v := os.Getenv("WINZE_RERANK_MODEL"); v != "" {
		return anthropic.Model(v)
	}
	return anthropic.ModelClaudeHaiku4_5
}

// rerankTimeout mirrors checkOneContradiction/critiqueTripConnection
// (cmd/metabolism) -- a hard timeout so a slow or down API degrades to
// unranked order rather than hanging winze_recall, which now depends
// synchronously on this call for the first time in this codebase's read path.
func rerankTimeout() time.Duration { return 30 * time.Second }

// rerankTop is the injectable core: reorders only the head of fused (the
// topK highest-RRF-ranked candidates) via the injected rerank closure,
// leaving the tail untouched. Any candidate id the closure omits or
// hallucinates is handled without ever dropping or duplicating a hit:
// omitted ids are appended after the ranked ones in their original
// relative order; hallucinated ids (already filtered by parseRerankResponse)
// are dropped as a second guard. On ANY error from the closure -- timeout,
// network failure, malformed response, missing key -- fused is returned
// completely unchanged: never partial, never panics.
func rerankTop(fused []fusedHit, kb *kbIndex, query string, topK int,
	rerank func(query string, candidates []rerankCandidate) ([]int, error)) []fusedHit {
	if topK <= 0 || len(fused) <= 1 {
		return fused
	}
	n := topK
	if n > len(fused) {
		n = len(fused)
	}
	head, tail := fused[:n], fused[n:]

	cands := make([]rerankCandidate, len(head))
	for i, f := range head {
		e := kb.Entities[f.idx]
		cands[i] = rerankCandidate{id: f.idx, name: e.Name, brief: e.Brief}
	}

	order, err := rerank(query, cands)
	if err != nil {
		return fused // fail-open, unchanged
	}

	byID := make(map[int]fusedHit, len(head))
	for _, f := range head {
		byID[f.idx] = f
	}
	reordered := make([]fusedHit, 0, len(head))
	seen := make(map[int]bool, len(head))
	for _, id := range order {
		if f, ok := byID[id]; ok && !seen[id] {
			reordered = append(reordered, f)
			seen[id] = true
		}
	}
	for _, f := range head { // ids the model omitted -- append after, original relative order preserved
		if !seen[f.idx] {
			reordered = append(reordered, f)
		}
	}

	out := make([]fusedHit, 0, len(fused))
	out = append(out, reordered...)
	return append(out, tail...)
}

// rerankTopK bounds how many of the fused candidates get sent to the
// reranker in one call. No 0-means-unlimited escape hatch -- this cost is
// billed per call and must never be accidentally uncapped, same policy as
// surfaceFormsMaxCalls.
func rerankTopK() int {
	if v := os.Getenv("WINZE_RERANK_TOPK"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 40 // comfortably above the measured median rank (25) at N=150
}

type rerankCandidate struct {
	id          int // fusedHit.idx -- index into kb.Entities
	name, brief string
}

// rerankShouldRun is rerankFused's gate, pulled out so it's testable without
// entangling env state with the API-key/client-construction branches below it
// -- "disabled" and "force=true but no key" both return fused unchanged from
// rerankFused, indistinguishable by output alone, so the gate itself needs
// its own direct test.
func rerankShouldRun(force bool) bool {
	return rerankEnabled() || force
}
