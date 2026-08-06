package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestMemRootResolutionOrder pins the three-step lookup in memRoot, and in
// particular that a repo's git config reaches every worktree of that repo.
//
// The worktree leg is the whole point of the git step: a linked worktree has
// its own working directory but shares .git/config through the common dir, so
// it must resolve to the same store as the main checkout without any
// per-worktree setup. That is the property native auto-memory keys on the
// repository for, and the one a cwd-derived store would lose.
func TestMemRootResolutionOrder(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}
	run(repo, "init", "-q", "-b", "main")
	run(repo, "config", "user.email", "t@example.com")
	run(repo, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(repo, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "add", "f")
	run(repo, "commit", "-qm", "init")

	// chdir is what memRoot actually reads, since git config resolves from the
	// process working directory.
	chdir := func(dir string) {
		t.Helper()
		prev, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(prev) })
	}

	t.Setenv("WINZE_MEMORY", "")
	chdir(repo)

	// No key set: falls through to the historical default, not to "".
	if got, want := memRoot(), filepath.Join(home(), "winze-memory"); got != want {
		t.Errorf("unset: memRoot() = %q, want the %q default", got, want)
	}

	run(repo, "config", "winze.memory", "/srv/shared-store")
	if got := memRoot(); got != "/srv/shared-store" {
		t.Errorf("main checkout: memRoot() = %q, want /srv/shared-store", got)
	}

	// A linked worktree: different working directory, same common dir, so the
	// key set above must reach it with no setup of its own.
	wt := filepath.Join(t.TempDir(), "wt")
	run(repo, "worktree", "add", "-q", "-b", "side", wt)
	chdir(wt)
	if got := memRoot(); got != "/srv/shared-store" {
		t.Errorf("linked worktree: memRoot() = %q, want /srv/shared-store — "+
			"the worktree is not seeing the repo's config", got)
	}

	// The env var is the explicit override and outranks git config.
	t.Setenv("WINZE_MEMORY", "/srv/override")
	if got := memRoot(); got != "/srv/override" {
		t.Errorf("with WINZE_MEMORY set: memRoot() = %q, want /srv/override", got)
	}
}
