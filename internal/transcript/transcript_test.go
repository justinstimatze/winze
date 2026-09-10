package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllSubstantialReturnsEveryQualifyingTurnInOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi-turn-session.jsonl")
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":"first substantial reply clears the forty character floor here"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"too short"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"second substantial reply also clears the forty character floor"}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	turns, err := AllSubstantial(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("got %d turns, want 2: %+v", len(turns), turns)
	}
	if turns[0].Text != "first substantial reply clears the forty character floor here" {
		t.Errorf("turns[0].Text = %q", turns[0].Text)
	}
	if turns[1].Text != "second substantial reply also clears the forty character floor" {
		t.Errorf("turns[1].Text = %q", turns[1].Text)
	}
	if turns[0].Index != 0 || turns[1].Index != 2 {
		t.Errorf("indices = %d, %d, want 0, 2", turns[0].Index, turns[1].Index)
	}
}

func TestFlattenContentHandlesBothWireShapes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"bare_string", `"hello there"`, "hello there"},
		{"block_array", `[{"type":"text","text":"first"},{"type":"text","text":"second"}]`, "first\n\nsecond"},
		{"tool_blocks_dropped", `[{"type":"tool_use","text":"ignored"},{"type":"text","text":"kept"}]`, "kept"},
		{"thinking_dropped", `[{"type":"thinking","text":"scratch work"},{"type":"text","text":"kept"}]`, "kept"},
		{"empty_array", `[]`, ""},
		{"unparseable", `not json`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FlattenContent(json.RawMessage(c.raw))
			if got != c.want {
				t.Errorf("FlattenContent(%s) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestLastSubstantialEmptyWhenNoneQualify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "thin-session.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hi"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"ok"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"sure"}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LastSubstantial(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("LastSubstantial = %q, want empty", got)
	}
}

func TestLastSubstantialReturnsTheLastQualifyingBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-session.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"hi"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"too short"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"this first substantial reply clears the forty character floor easily"}}`,
		`{"type":"user","message":{"role":"user","content":"and then?"}}`,
		`{"type":"assistant","isSidechain":true,"message":{"role":"assistant","content":"this sidechain reply also clears the floor but must be skipped"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":"this final substantial reply is the one that should come back"}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LastSubstantial(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "this final substantial reply is the one that should come back"
	if got != want {
		t.Errorf("LastSubstantial = %q, want %q", got, want)
	}
}
