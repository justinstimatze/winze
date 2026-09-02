package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadProjectTranscriptsAgainstTheRealCorpus runs the reader over an
// actual Claude Code project directory and reports the shape Phase 3b would
// draw from.
//
// It is an instrument, not a gate: its input lives in ~/.claude, never in
// this repo, so it skips in CI and on any machine but the author's -- the
// same honest status TestAskOnceReplayAgainstTheRealLog carries for the
// metabolism log. The fixtures in transcript_test.go prove the parsing rules;
// only this proves the rules survive contact with files a live process wrote.
//
// Point it elsewhere with WINZE_TRANSCRIPT_DIR. The default is the ~/Documents
// project, chosen over the far larger stope project (422 sessions) because
// stope's contents are private and this test prints session prose to the log.
func TestReadProjectTranscriptsAgainstTheRealCorpus(t *testing.T) {
	dir := os.Getenv("WINZE_TRANSCRIPT_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("no home dir")
		}
		dir = filepath.Join(home, ".claude", "projects", "-home-gas6amus-Documents")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no transcript corpus at %s", dir)
	}

	start := time.Now()
	sessions, err := readProjectTranscripts(dir, 4)
	if err != nil {
		t.Fatalf("readProjectTranscripts(%s): %v", dir, err)
	}
	elapsed := time.Since(start)

	var turns, bytes int
	for _, s := range sessions {
		turns += len(s.Turns)
		for _, turn := range s.Turns {
			bytes += len(turn.Content)
		}
	}
	first, last := sessions[0], sessions[len(sessions)-1]
	t.Logf("%d sessions, %s .. %s (%d days), %d turns, %.1f MB of prose, read in %s",
		len(sessions),
		first.Start.Format("2006-01-02"), last.Start.Format("2006-01-02"),
		int(last.Start.Sub(first.Start).Hours()/24),
		turns, float64(bytes)/(1<<20), elapsed.Round(time.Millisecond))

	// The property Phase 3b actually needs: writes spread across real calendar
	// time, so a session's recall can be scored against a store that grew
	// after it. A corpus that is dense but same-day cannot answer the question
	// docs/agent-identity-integration.md left open, however many sessions it has.
	span := last.Start.Sub(first.Start)
	if span < 14*24*time.Hour {
		t.Errorf("corpus spans %v; Phase 3b needs weeks of separation, not hours", span)
	}
	for i := 1; i < len(sessions); i++ {
		if sessions[i].Start.Before(sessions[i-1].Start) {
			t.Fatalf("session %d starts before %d -- not chronological", i, i-1)
		}
	}
}
