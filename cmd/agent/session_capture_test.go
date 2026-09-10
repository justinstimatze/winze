package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlattenSessionContentHandlesBothWireShapes(t *testing.T) {
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
			got := flattenSessionContent(json.RawMessage(c.raw))
			if got != c.want {
				t.Errorf("flattenSessionContent(%s) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestLastSubstantialAssistantTurnEmptyWhenNoneQualify(t *testing.T) {
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
	got, err := lastSubstantialAssistantTurn(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("lastSubstantialAssistantTurn = %q, want empty", got)
	}
}

func TestLastSubstantialAssistantTurnReturnsTheLastQualifyingBlock(t *testing.T) {
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
	got, err := lastSubstantialAssistantTurn(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "this final substantial reply is the one that should come back"
	if got != want {
		t.Errorf("lastSubstantialAssistantTurn = %q, want %q", got, want)
	}
}

func TestStoreHasNoRemoteReflectsActualGitState(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if !storeHasNoRemote(dir) {
		t.Error("freshly initialized repo should report no remote")
	}
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://example.invalid/x.git").CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}
	if storeHasNoRemote(dir) {
		t.Error("repo with a remote configured should not report no remote")
	}
}
