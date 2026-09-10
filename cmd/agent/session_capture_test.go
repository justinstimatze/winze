package main

import (
	"os/exec"
	"testing"
)

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
