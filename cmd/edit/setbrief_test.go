package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const setBriefFixture = `package winze

var RecallGate = Concept{&Entity{
	ID:    "concept-recall-gate",
	Name:  "Recall gate",
	Brief: ` + "`old brief text`" + `,
}}

var Other = Concept{&Entity{
	Name:  "Other",
	Brief: ` + "`untouched`" + `,
}}
`

func TestPlanSetBrief(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corpus.go")
	if err := os.WriteFile(path, []byte(setBriefFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	// Brief only.
	file, edits, err := planSetBrief(dir, "RecallGate", "new brief text", "")
	if err != nil {
		t.Fatalf("planSetBrief: %v", err)
	}
	if file != path {
		t.Errorf("file = %q, want %q", file, path)
	}
	src, _ := os.ReadFile(path)
	got := string(applyEdits(src, edits))
	if !strings.Contains(got, "`new brief text`") {
		t.Errorf("Brief not replaced:\n%s", got)
	}
	if !strings.Contains(got, "`untouched`") {
		t.Error("sibling entity's Brief was disturbed")
	}
	if strings.Contains(got, "old brief text") {
		t.Error("old Brief still present")
	}

	// Brief + Name together.
	_, edits2, err := planSetBrief(dir, "RecallGate", "b2", "Recall relevance gate")
	if err != nil {
		t.Fatalf("planSetBrief with name: %v", err)
	}
	got2 := string(applyEdits(src, edits2))
	if !strings.Contains(got2, `"Recall relevance gate"`) || !strings.Contains(got2, "`b2`") {
		t.Errorf("Name+Brief not both replaced:\n%s", got2)
	}

	// Missing var is an error, not a silent no-op.
	if _, _, err := planSetBrief(dir, "Nope", "x", ""); err == nil {
		t.Error("expected error for missing var")
	}
}

func TestBriefLiteral(t *testing.T) {
	if got := briefLiteral("plain text"); got != "`plain text`" {
		t.Errorf("briefLiteral(plain) = %q", got)
	}
	// A backtick can't live in a raw string → must fall back to a quoted string.
	got := briefLiteral("has ` backtick")
	if strings.HasPrefix(got, "`") {
		t.Errorf("briefLiteral with backtick used a raw string: %q", got)
	}
	if !strings.Contains(got, "backtick") {
		t.Errorf("briefLiteral dropped content: %q", got)
	}
}
