// Package corpuslock provides a corpus-wide advisory write lock so that
// concurrent winze mutators serialize their read-modify-gate-commit
// critical sections.
//
// winze's write path (cmd/add, cmd/add --batch, cmd/edit rename/merge) is an
// unguarded read-modify-write: back up the target, mutate it, run the
// `go build .` gate over the whole corpus dir, revert on failure. That is
// correct for a single writer. Under the multi-session shared-KB shape
// (docs/multi-session-write-shape.md) — N parallel Claude Code sessions all
// pointed at one corpus — it races three ways:
//
//  1. Lost update: A and B both read foo.go; A appends+writes; B appends+writes
//     from its now-stale backup, silently dropping A's claim.
//  2. Revert clobber: A's gate fails and it reverts to its own backup, wiping
//     B's already-committed good write. A failing write destroys a succeeding one.
//  3. Cross-file false revert: the gate is `go build .` over the whole dir, so
//     B running its gate while A is mid-write (a syntactically broken
//     intermediate file) sees a build failure on A's partial content and
//     reverts B's own valid change — even for writes to different files.
//
// The window is the entire gate (~100-300ms), so with parallel writers these
// are not hypothetical. A single corpus-wide exclusive lock held across the
// whole critical section serializes writers and closes all three. It must be
// corpus-wide rather than per-file precisely because of race 3: the shared
// build couples writes to unrelated files.
//
// The lock is a flock(2) on a fd in the corpus root. flock associates the lock
// with the open file description, so it serializes both across processes and
// across two Acquire calls within one process, and the kernel releases it when
// the fd closes — including on process crash. There is no stale-lock file to
// reap and no PID bookkeeping.
package corpuslock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// LockName is the advisory lock file created in the corpus root. It carries no
// content — the flock on its fd is the lock. Add it to .gitignore.
const LockName = ".winze.lock"

// Acquire takes an exclusive advisory lock on the corpus directory, blocking
// until it is available. Call the returned release func (via defer) after the
// mutation commits or reverts. A crashed holder releases the lock
// automatically when the OS closes its fd, so callers need no recovery path.
func Acquire(dir string) (release func(), err error) {
	path := filepath.Join(dir, LockName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open corpus lock %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("flock corpus lock %s: %w", path, err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
