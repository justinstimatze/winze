package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justinstimatze/winze/internal/transcript"
)

func TestBuildFTIndexForTranscriptEmptyOnNoTurns(t *testing.T) {
	hits := buildFTIndexForTranscript(nil).search("anything", 0)
	if len(hits) != 0 {
		t.Fatalf("expected no hits on an empty index, got %+v", hits)
	}
}

// TestBuildFTIndexForTranscriptRanking mirrors TestFulltextRanking's shape
// against the new population: a turn carrying the query terms outranks the
// field, and a turn with none of them does not appear at all.
func TestBuildFTIndexForTranscriptRanking(t *testing.T) {
	turns := []transcript.Turn{
		{Text: "cleared 16 gigabytes by removing orphaned go-build temp directories", Index: 0},
		{Text: "the dispatch watch died and needed re-arming after the restart", Index: 1},
		{Text: "unrelated turn about something else entirely, no shared terms here", Index: 2},
	}
	hits := buildFTIndexForTranscript(turns).search("go-build temp directories", 0)
	if len(hits) == 0 {
		t.Fatal("expected hits for 'go-build temp directories'")
	}
	if hits[0].ref != 0 {
		t.Fatalf("expected turn 0 ranked first, got %+v", hits[0])
	}
	for _, h := range hits {
		if h.ref == 1 {
			t.Fatal("turn 1 shares no query terms; must not match")
		}
	}
}

func TestResolveSessionTranscriptUnderErrorsWhenNotFound(t *testing.T) {
	root := t.TempDir()
	if _, err := resolveSessionTranscriptUnder(root, "does-not-exist"); err == nil {
		t.Error("expected an error for a session with no matching transcript")
	}
}

func TestResolveSessionTranscriptUnderFindsAcrossProjectDirs(t *testing.T) {
	root := t.TempDir()
	projA := filepath.Join(root, "-home-user-Documents-project-a")
	projB := filepath.Join(root, "-home-user-Documents-project-b")
	if err := os.MkdirAll(projA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projB, 0o755); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(projB, "abc-123.jsonl")
	if err := os.WriteFile(wantPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveSessionTranscriptUnder(root, "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantPath {
		t.Errorf("got %q, want %q", got, wantPath)
	}
}
