package main

import (
	"strings"
	"testing"
)

// A long assistant monologue early in a session must not push a later user turn
// out of the rendered body. The old positional s[:sessionCap] cut did exactly
// that — measured at 8/500 questions, all user-answer types. renderSession now
// sheds assistant-turn length first, so user turns survive.
func TestRenderSessionKeepsUserTurnsOverBudget(t *testing.T) {
	// Five ~turnCap assistant turns (~20k capped) overflow sessionCap before the
	// final user turn is reached; a blind head-cut would drop it.
	filler := strings.Repeat("x", turnCap)
	var turns []Turn
	for i := 0; i < 5; i++ {
		turns = append(turns, Turn{Role: "assistant", Content: filler})
	}
	const marker = "USER_ANSWER_SENTINEL_9f3a1c"
	turns = append(turns, Turn{Role: "user", Content: marker})

	out := renderSession(turns)
	if !strings.Contains(out, marker) {
		t.Fatalf("user answer turn dropped from over-budget session (rendered %d bytes)", len(out))
	}
}

// An in-budget session renders exactly as the naive per-turn-capped flatten —
// the role-aware path is a no-op below sessionCap, so existing behavior (and the
// lens disk-cache keyed on the rendered body) is unchanged.
func TestRenderSessionInBudgetUnchanged(t *testing.T) {
	turns := []Turn{
		{Role: "user", Content: "hi, I moved to Seattle in 2021"},
		{Role: "assistant", Content: "Congrats on the move!"},
		{Role: "user", Content: "I drive a 2023 Honda Civic"},
	}
	var want strings.Builder
	for _, tn := range turns {
		want.WriteString(tn.Role + ": " + tn.Content + "\n")
	}
	if got := renderSession(turns); got != want.String() {
		t.Fatalf("in-budget render changed:\n got: %q\nwant: %q", got, want.String())
	}
}

// Even when squeezing assistant turns is not enough (a session made entirely of
// long user turns), the fallback positional cut still bounds the output so the
// call stays cheap.
func TestRenderSessionStaysBounded(t *testing.T) {
	filler := strings.Repeat("y", turnCap)
	var turns []Turn
	for i := 0; i < 10; i++ {
		turns = append(turns, Turn{Role: "user", Content: filler})
	}
	out := renderSession(turns)
	// sessionCap plus the truncation sentinel line is the ceiling.
	if len(out) > sessionCap+len("\n[...session truncated...]") {
		t.Fatalf("rendered session not bounded: %d bytes", len(out))
	}
	if !strings.Contains(out, "[...session truncated...]") {
		t.Fatalf("over-budget all-user session should carry the truncation sentinel")
	}
}
