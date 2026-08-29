package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// askFixture is a small CLAUDE.md carrying one onsetter ask that fires on the
// word "always" anywhere in the note — patterned on onsetter's own
// ask_external_test.go.
const askFixture = "```ask\n" +
	"when: \\balways\\b\n" +
	"\n" +
	"A rule-shaped memory should probably be a hook instead of prose that is\n" +
	"only read when it happens to be in context.\n" +
	"```\n"

func writeAskFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte(askFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WINZE_AGENT_CLAUDE_MD", path)
	return path
}

func TestCheckOnsetterGateFiresOnRuleShapedNote(t *testing.T) {
	writeAskFixture(t)

	hits, err := checkOnsetterGate("always run gofmt before committing")
	if err != nil {
		t.Fatalf("checkOnsetterGate: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits, want 1: %+v", len(hits), hits)
	}
	if !strings.Contains(hits[0].Body, "hook") {
		t.Errorf("hit body = %q, want it to mention a hook", hits[0].Body)
	}
	if hits[0].Matched != "always" {
		t.Errorf("hit matched = %q, want %q", hits[0].Matched, "always")
	}
}

func TestCheckOnsetterGatePassesOrdinaryNote(t *testing.T) {
	writeAskFixture(t)

	hits, err := checkOnsetterGate("the sky looked orange at sunset over the bay")
	if err != nil {
		t.Fatalf("checkOnsetterGate: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("got %d hits for an ordinary note, want 0: %+v", len(hits), hits)
	}
}

func TestCheckOnsetterGateNoClaudeMDNoops(t *testing.T) {
	clearStoreEnv(t)
	t.Setenv("WINZE_AGENT_CLAUDE_MD", "")
	prevOverride := onsetterCheckOverride
	t.Cleanup(func() { onsetterCheckOverride = prevOverride })
	onsetterCheckOverride = ""
	t.Setenv("WINZE_STORE", filepath.Join(t.TempDir(), "no-such-store"))

	hits, err := checkOnsetterGate("always run gofmt before committing")
	if err != nil {
		t.Fatalf("checkOnsetterGate: %v, want nil error when no CLAUDE.md exists", err)
	}
	if hits != nil {
		t.Fatalf("got %+v, want nil hits when no CLAUDE.md exists", hits)
	}
}

func TestOnsetterAdvisoryEmptyWhenNoHits(t *testing.T) {
	if got := onsetterAdvisory(nil); got != "" {
		t.Errorf("onsetterAdvisory(nil) = %q, want \"\"", got)
	}
}

func TestOnsetterAdvisoryQuotesTheMatch(t *testing.T) {
	hits := []onsetterHit{{Where: "CLAUDE.md:1", Matched: "always", Body: "want a hook instead"}}
	got := onsetterAdvisory(hits)
	for _, want := range []string{"CLAUDE.md:1", "always", "want a hook instead"} {
		if !strings.Contains(got, want) {
			t.Errorf("onsetterAdvisory(...) = %q, want it to contain %q", got, want)
		}
	}
}
