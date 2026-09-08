package main

import (
	"fmt"
	"os"
	"testing"
)

func TestParseRerankResponseFiltersInvalidIDs(t *testing.T) {
	out, err := parseRerankResponse("[5, 1, 0]", []int{0, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 || out[0] != 1 || out[1] != 0 {
		t.Fatalf("expected hallucinated id 5 filtered, got %v", out)
	}
}

func TestParseRerankResponseRejectsEmptyArray(t *testing.T) {
	if _, err := parseRerankResponse("[]", []int{0, 1}); err == nil {
		t.Fatalf("expected error on empty array")
	}
}

func TestParseRerankResponseRejectsNonJSON(t *testing.T) {
	if _, err := parseRerankResponse("not json", []int{0, 1}); err == nil {
		t.Fatalf("expected error on non-JSON input")
	}
}

func TestParseRerankResponseStripsMarkdownFence(t *testing.T) {
	out, err := parseRerankResponse("```json\n[2,1,0]\n```", []int{0, 1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 || out[0] != 2 {
		t.Fatalf("expected [2,1,0], got %v", out)
	}
}

func TestRerankEnabled(t *testing.T) {
	os.Unsetenv("WINZE_RERANK")
	if rerankEnabled() {
		t.Fatalf("expected disabled when unset")
	}
	os.Setenv("WINZE_RERANK", "1")
	defer os.Unsetenv("WINZE_RERANK")
	if !rerankEnabled() {
		t.Fatalf("expected enabled when set")
	}
}

func TestRerankFusedDisabledReturnsUnchanged(t *testing.T) {
	os.Unsetenv("WINZE_RERANK")
	kb := &kbIndex{Entities: []entityRecord{{Name: "A", Brief: "b"}}}
	fused := []fusedHit{{idx: 0}}
	out := rerankFused(".", fused, kb, "q")
	if len(out) != 1 || out[0].idx != 0 {
		t.Fatalf("expected unchanged output when disabled, got %v", out)
	}
}

func TestRerankFusedNoAPIKeyReturnsUnchanged(t *testing.T) {
	os.Setenv("WINZE_RERANK", "1")
	defer os.Unsetenv("WINZE_RERANK")
	oldKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", oldKey)
		}
	}()
	kb := &kbIndex{Entities: []entityRecord{{Name: "A", Brief: "b"}}}
	fused := []fusedHit{{idx: 0}}
	// t.TempDir() is outside any git worktree, so loadDotEnv's git-worktree
	// fallback finds nothing and this stays genuinely network-free.
	out := rerankFused(t.TempDir(), fused, kb, "q")
	if len(out) != 1 || out[0].idx != 0 {
		t.Fatalf("expected unchanged output with no API key, got %v", out)
	}
}

func TestRerankTopAppendsMissingIDs(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{Name: "A", Brief: "brief a"},
		{Name: "B", Brief: "brief b"},
		{Name: "C", Brief: "brief c"},
	}}
	fused := []fusedHit{{idx: 0}, {idx: 1}, {idx: 2}}
	rerank := func(q string, cands []rerankCandidate) ([]int, error) {
		return []int{2}, nil // omits 0 and 1
	}
	out := rerankTop(fused, kb, "q", 10, rerank)
	if len(out) != 3 {
		t.Fatalf("expected all 3 present, got %d", len(out))
	}
	if out[0].idx != 2 {
		t.Fatalf("expected ranked id 2 first, got %v", out)
	}
	if out[1].idx != 0 || out[2].idx != 1 {
		t.Fatalf("expected omitted ids 0,1 appended in original relative order, got %v", out)
	}
}

func TestRerankTopDropsHallucinatedIDs(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{Name: "A", Brief: "brief a"},
		{Name: "B", Brief: "brief b"},
	}}
	fused := []fusedHit{{idx: 0}, {idx: 1}}
	rerank := func(q string, cands []rerankCandidate) ([]int, error) {
		return []int{99, 1, 0}, nil // 99 doesn't exist
	}
	out := rerankTop(fused, kb, "q", 10, rerank)
	if len(out) != 2 {
		t.Fatalf("expected hallucinated id dropped, output length unchanged at 2, got %d", len(out))
	}
	for _, f := range out {
		if f.idx == 99 {
			t.Fatalf("hallucinated id 99 leaked into output: %v", out)
		}
	}
}

func TestRerankTopEnforcesCap(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{Name: "A", Brief: "brief a"},
		{Name: "B", Brief: "brief b"},
		{Name: "C", Brief: "brief c"},
	}}
	fused := []fusedHit{{idx: 0}, {idx: 1}, {idx: 2}}
	var gotN int
	rerank := func(q string, cands []rerankCandidate) ([]int, error) {
		gotN = len(cands)
		return []int{2, 1, 0}, nil
	}
	rerankTop(fused, kb, "q", 2, rerank)
	if gotN != 2 {
		t.Fatalf("expected exactly 2 candidates sent for topK=2, got %d", gotN)
	}
}

func TestRerankTopFailsOpenOnError(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{Name: "A", Brief: "brief a"},
		{Name: "B", Brief: "brief b"},
	}}
	fused := []fusedHit{{idx: 0}, {idx: 1}}
	rerank := func(q string, cands []rerankCandidate) ([]int, error) {
		return nil, fmt.Errorf("boom")
	}
	out := rerankTop(fused, kb, "q", 10, rerank)
	if len(out) != len(fused) {
		t.Fatalf("expected unchanged length on error, got %d", len(out))
	}
	for i := range fused {
		if out[i].idx != fused[i].idx {
			t.Fatalf("expected order-for-order unchanged on error, got %v want %v", out, fused)
		}
	}
}

func TestRerankTopKDefaultAndOverride(t *testing.T) {
	defer os.Unsetenv("WINZE_RERANK_TOPK")

	os.Unsetenv("WINZE_RERANK_TOPK")
	if got := rerankTopK(); got != 40 {
		t.Fatalf("expected default 40, got %d", got)
	}

	os.Setenv("WINZE_RERANK_TOPK", "10")
	if got := rerankTopK(); got != 10 {
		t.Fatalf("expected override 10, got %d", got)
	}

	os.Setenv("WINZE_RERANK_TOPK", "0")
	if got := rerankTopK(); got != 40 {
		t.Fatalf("expected fallback to 40 on non-positive override, got %d", got)
	}

	os.Setenv("WINZE_RERANK_TOPK", "not-a-number")
	if got := rerankTopK(); got != 40 {
		t.Fatalf("expected fallback to 40 on invalid override, got %d", got)
	}
}
