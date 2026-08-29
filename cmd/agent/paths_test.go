package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// clearStoreEnv unsets what storeRoot reads, so a test's result never depends
// on the ambient environment.
func clearStoreEnv(t *testing.T) {
	t.Helper()
	t.Setenv("WINZE_STORE", "")
}

// TestStoreRootResolutionOrder pins the three-step lookup in storeRoot, and in
// particular that a repo's git config reaches every worktree of that repo.
//
// The worktree leg is the whole point of the git step: a linked worktree has
// its own working directory but shares .git/config through the common dir, so
// it must resolve to the same store as the main checkout without any
// per-worktree setup. That is the property native auto-memory keys on the
// repository for, and the one a cwd-derived store would lose.
func TestStoreRootResolutionOrder(t *testing.T) {
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

	// chdir is what storeRoot actually reads, since git config resolves from the
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

	clearStoreEnv(t)
	chdir(repo)

	// No key set: falls through to the historical default, not to "".
	if got, want := storeRoot(), filepath.Join(home(), "winze-memory"); got != want {
		t.Errorf("unset: storeRoot() = %q, want the %q default", got, want)
	}

	run(repo, "config", "winze.store", "/srv/shared-store")
	if got := storeRoot(); got != "/srv/shared-store" {
		t.Errorf("main checkout: storeRoot() = %q, want /srv/shared-store", got)
	}

	// A linked worktree: different working directory, same common dir, so the
	// key set above must reach it with no setup of its own.
	wt := filepath.Join(t.TempDir(), "wt")
	run(repo, "worktree", "add", "-q", "-b", "side", wt)
	chdir(wt)
	if got := storeRoot(); got != "/srv/shared-store" {
		t.Errorf("linked worktree: storeRoot() = %q, want /srv/shared-store — "+
			"the worktree is not seeing the repo's config", got)
	}

	// The env var is the explicit override and outranks git config.
	t.Setenv("WINZE_STORE", "/srv/override")
	if got := storeRoot(); got != "/srv/override" {
		t.Errorf("with WINZE_STORE set: storeRoot() = %q, want /srv/override", got)
	}
}

// TestStoreRootConfigured covers the distinction storeRoot itself cannot make:
// an opted-in store versus the last-resort default. The capture guard is
// installed user-wide on the strength of this, so the case that matters most
// is the negative one — a directory outside any repo, with no env var and no
// ~/winze-memory, must report false and let the native write through.
func TestStoreRootConfigured(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	// HOME is redirected so the ~/winze-memory leg is testable without
	// depending on whether the real home has one.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	outside := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	clearStoreEnv(t)
	if storeRootConfigured() {
		t.Error("no env, no repo, no ~/winze-memory: want false — the guard would " +
			"block a native write and have nowhere to redirect it")
	}

	// The bare default counts once the directory actually exists.
	if err := os.Mkdir(filepath.Join(fakeHome, "winze-memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !storeRootConfigured() {
		t.Error("~/winze-memory exists: want true")
	}
	if err := os.Remove(filepath.Join(fakeHome, "winze-memory")); err != nil {
		t.Fatal(err)
	}

	// A file at that path is not a store.
	if err := os.WriteFile(filepath.Join(fakeHome, "winze-memory"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if storeRootConfigured() {
		t.Error("~/winze-memory is a regular file: want false")
	}
	if err := os.Remove(filepath.Join(fakeHome, "winze-memory")); err != nil {
		t.Fatal(err)
	}

	// git config is opt-in by construction, and needs no store on disk yet.
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = outside
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "winze.store", "/srv/shared-store")
	if !storeRootConfigured() {
		t.Error("winze.store set in git config: want true")
	}

	// The env var alone is enough, in any directory.
	run("config", "--unset", "winze.store")
	t.Setenv("WINZE_STORE", "/srv/explicit")
	if !storeRootConfigured() {
		t.Error("WINZE_STORE set: want true")
	}
}

// TestOnsetterClaudeMDResolutionOrder pins the override-then-env-then-store-root
// chain onsetterClaudeMD adds on top of storeRoot, and that it fails open (
// returns "") rather than naming a path that does not exist, so
// checkOnsetterGate can skip cleanly instead of erroring on a missing file.
func TestOnsetterClaudeMDResolutionOrder(t *testing.T) {
	clearStoreEnv(t)
	t.Setenv("WINZE_AGENT_CLAUDE_MD", "")
	prevOverride := onsetterCheckOverride
	t.Cleanup(func() { onsetterCheckOverride = prevOverride })
	onsetterCheckOverride = ""

	store := t.TempDir()
	t.Setenv("WINZE_STORE", store)

	// No CLAUDE.md anywhere yet: fails open.
	if got := onsetterClaudeMD(); got != "" {
		t.Errorf("no CLAUDE.md exists: onsetterClaudeMD() = %q, want \"\"", got)
	}

	// The store's own root CLAUDE.md is the default once it exists.
	storeCM := filepath.Join(store, "CLAUDE.md")
	if err := os.WriteFile(storeCM, []byte("store"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := onsetterClaudeMD(); got != storeCM {
		t.Errorf("store CLAUDE.md exists: onsetterClaudeMD() = %q, want %q", got, storeCM)
	}

	// $WINZE_AGENT_CLAUDE_MD outranks the store default, and need not exist on
	// disk itself — the caller resolves it, this function only names it.
	t.Setenv("WINZE_AGENT_CLAUDE_MD", "/somewhere/else/CLAUDE.md")
	if got := onsetterClaudeMD(); got != "/somewhere/else/CLAUDE.md" {
		t.Errorf("with WINZE_AGENT_CLAUDE_MD set: onsetterClaudeMD() = %q, want /somewhere/else/CLAUDE.md", got)
	}

	// --onsetter-check outranks everything, including the env var.
	onsetterCheckOverride = "/forced/CLAUDE.md"
	if got := onsetterClaudeMD(); got != "/forced/CLAUDE.md" {
		t.Errorf("with onsetterCheckOverride set: onsetterClaudeMD() = %q, want /forced/CLAUDE.md", got)
	}
}
