package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleRecallRaw answers winze_recall_raw: BM25 search over the store's
// raw.jsonl evidence log (cmd/agent/rawlog.go's appendRawLog writes it).
// Unlike handleRecall, this returns verbatim note text, never a typed claim
// -- nothing here is encoded, so it carries none of winze_remember's gates:
// no dedup check, no onsetter check, no build gate, no commit.
func handleRecallRaw(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := req.GetArguments()["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return mcp.NewToolResultError("query: required string argument"), nil
	}
	limit := recallDefaultLimit
	if v, ok := req.GetArguments()["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	out, err := runQueryRaw("--raw", query, "--json")
	if err != nil {
		return mcp.NewToolResultError(recallFailureMessage()), nil
	}
	var res rawQueryResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return mcp.NewToolResultError(recallFailureMessage()), nil
	}
	if len(res.Hits) == 0 {
		return mcp.NewToolResultText("no raw evidence matched — nothing recalled."), nil
	}
	hits := res.Hits
	if len(hits) > limit {
		hits = hits[:limit]
	}

	text, _ := json.MarshalIndent(struct {
		Matched int            `json:"matched"`
		Shown   int            `json:"shown"`
		Hits    []rawRecallHit `json:"hits"`
	}{Matched: res.Count, Shown: len(hits), Hits: hits}, "", "  ")
	return mcp.NewToolResultText(string(text)), nil
}

type rawQueryResult struct {
	Count int            `json:"count"`
	Hits  []rawRecallHit `json:"hits"`
}

// rawRecallHit is winze_recall_raw's result shape -- verbatim raw-log text
// plus its timestamp and writing tool, not queryHit's curated entity
// headline. The two tools return different object classes on purpose (see
// docs/raw-evidence-retrieval.md): a typed memory is a claim, a raw hit is
// source text with no claim made about it.
type rawRecallHit struct {
	Time  string  `json:"time"`
	Tool  string  `json:"tool"`
	Var   string  `json:"var"`
	Note  string  `json:"note"`
	Score float64 `json:"score"`
}
