package main

import (
	"strings"
	"testing"
)

func TestBestRankOfPicksTheBetterRank(t *testing.T) {
	hits := recallHits{Hits: []struct {
		VarName string `json:"var_name"`
	}{{VarName: "A"}, {VarName: "B"}, {VarName: "C"}}}
	if got := bestRankOf(hits, []string{"C", "B"}); got != 2 {
		t.Errorf("bestRankOf = %d, want 2 (B's rank, the better of B and C)", got)
	}
	if got := bestRankOf(hits, []string{"nope", ""}); got != 0 {
		t.Errorf("bestRankOf = %d, want 0 when nothing in the set is present", got)
	}
	if got := bestRankOf(hits, nil); got != 0 {
		t.Errorf("bestRankOf = %d, want 0 for an empty set", got)
	}
}

func TestParseFactsResponseRejectsEmptyArray(t *testing.T) {
	if _, err := parseFactsResponse("[]"); err == nil {
		t.Error("want an error for zero facts, got nil")
	}
}

func TestParseFactsResponseRejectsNonJSON(t *testing.T) {
	if _, err := parseFactsResponse("I extracted these facts: one, two"); err == nil {
		t.Error("want an error for unparseable text, got nil")
	}
}

// TestParseFactsResponseStripsMarkdownFence covers the shape Haiku actually
// returns despite the "ONLY a JSON array" instruction -- code fences around
// otherwise-valid JSON are common enough from smaller models to be worth a
// tolerant parse rather than a strict one that discards a good response.
func TestParseFactsResponseStripsMarkdownFence(t *testing.T) {
	raw := "```json\n[\"fact one\", \"fact two\"]\n```"
	got, err := parseFactsResponse(raw)
	if err != nil {
		t.Fatalf("parseFactsResponse: %v", err)
	}
	if len(got) != 2 || got[0] != "fact one" || got[1] != "fact two" {
		t.Errorf("got %v, want [fact one, fact two]", got)
	}
}

// TestSessionExcerptExcludesTheHeldOutLaterAsk mirrors the arc-note holdout
// test: the excerpt fed to extraction must not contain the exact text
// LaterAsk returns, or the extracted facts could trivially answer the probe
// they're meant to be tested against.
func TestSessionExcerptExcludesTheHeldOutLaterAsk(t *testing.T) {
	s := &transcriptSession{Turns: []Turn{
		{Role: "user", Content: "the opening ask, long enough to clear the length floor easily"},
		{Role: "user", Content: "an early later turn, also long enough to clear the floor here"},
		{Role: "user", Content: "the held-out midpoint turn about a very specific unusual topic xyzzy"},
		{Role: "user", Content: "a late later turn, again long enough to clear the floor for this one"},
	}}
	held := s.LaterAsk()
	if held == "" || !strings.Contains(held, "xyzzy") {
		t.Fatalf("test setup: expected LaterAsk to hold the xyzzy turn, got %q", held)
	}
	excerpt := sessionExcerpt(s)
	if strings.Contains(excerpt, "xyzzy") {
		t.Errorf("sessionExcerpt leaked the held-out later ask: %q", excerpt)
	}
	if !strings.Contains(excerpt, "early later turn") || !strings.Contains(excerpt, "late later turn") {
		t.Errorf("sessionExcerpt dropped non-held-out arc content: %q", excerpt)
	}
}
