package corpuslock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAcquire_MutualExclusion proves a second Acquire blocks while the first
// lock is held and proceeds once it is released. flock locks the open file
// description, not the process, so two Acquire calls in one process contend
// through two separate fds — the same mechanism that serializes two separate
// winze processes. This is the property the concurrent-write races depend on.
func TestAcquire_MutualExclusion(t *testing.T) {
	dir := t.TempDir()

	rel1, err := Acquire(dir)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}

	got := make(chan struct{})
	go func() {
		rel2, err := Acquire(dir)
		if err != nil {
			t.Errorf("second Acquire: %v", err)
			return
		}
		close(got)
		rel2()
	}()

	select {
	case <-got:
		t.Fatal("second Acquire returned while first lock held — not mutually exclusive")
	case <-time.After(100 * time.Millisecond):
		// expected: the second Acquire is blocked.
	}

	rel1()

	select {
	case <-got:
		// expected: released, second Acquire proceeded.
	case <-time.After(2 * time.Second):
		t.Fatal("second Acquire did not proceed after the first lock was released")
	}
}

// TestAcquire_CreatesLockFile confirms Acquire materializes the lock file so a
// fresh corpus (no prior write) locks cleanly rather than erroring.
func TestAcquire_CreatesLockFile(t *testing.T) {
	dir := t.TempDir()
	rel, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer rel()
	if _, err := os.Stat(filepath.Join(dir, LockName)); err != nil {
		t.Fatalf("lock file not created: %v", err)
	}
}
