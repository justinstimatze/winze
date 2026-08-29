package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppendRawLogIsBestEffort pins the design intent directly: a write that
// cannot land (here, a store directory that does not exist) must not panic
// and must not surface an error to the caller, because the raw tier is a
// recovery path, not a gate -- the typed write it sits beside must never be
// blocked by this failing.
func TestAppendRawLogIsBestEffort(t *testing.T) {
	clearStoreEnv(t)
	t.Setenv("WINZE_STORE", filepath.Join(t.TempDir(), "does-not-exist"))

	appendRawLog("remember", "", "a note nobody will ever read")
	// Reaching here without panicking is the assertion.
}

// TestAppendRawLogWritesOneJSONLinePerCall pins the recovery-path contract:
// every call is its own line, independent of any other call, so a partial
// write from a crashed process never corrupts an entry that already landed.
func TestAppendRawLogWritesOneJSONLinePerCall(t *testing.T) {
	clearStoreEnv(t)
	store := t.TempDir()
	t.Setenv("WINZE_STORE", store)

	appendRawLog("remember", "", "first note")
	appendRawLog("update", "SomeVar", "second note")

	data, err := os.ReadFile(rawLogPath())
	if err != nil {
		t.Fatalf("reading raw log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), data)
	}

	var first, second rawLogEntry
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 1 not valid JSON: %v", err)
	}
	if first.Tool != "remember" || first.Note != "first note" || first.Var != "" {
		t.Errorf("line 1 = %+v, want tool=remember note=%q var=\"\"", first, "first note")
	}
	if first.Time == "" {
		t.Error("line 1 Time is empty, want an RFC3339 timestamp")
	}

	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("line 2 not valid JSON: %v", err)
	}
	if second.Tool != "update" || second.Note != "second note" || second.Var != "SomeVar" {
		t.Errorf("line 2 = %+v, want tool=update note=%q var=SomeVar", second, "second note")
	}
}

func TestRawLogPathFollowsStoreRoot(t *testing.T) {
	clearStoreEnv(t)
	store := t.TempDir()
	t.Setenv("WINZE_STORE", store)
	if got, want := rawLogPath(), filepath.Join(store, "raw.jsonl"); got != want {
		t.Errorf("rawLogPath() = %q, want %q", got, want)
	}
}
