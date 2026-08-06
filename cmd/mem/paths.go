package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The memory store and the winze tool binaries are located by env var, so one
// winze-mem binary serves any number of stores (point WINZE_MEMORY at each) and
// runs against tooling wherever it is installed.

// memRoot is the winze-memory store (the dir holding memory.go + schema files).
//
// Resolution order, most explicit first:
//
//  1. $WINZE_MEMORY — a caller that names a store outright always wins.
//  2. `git config --get winze.memory` in the working directory. Git keeps
//     config in the common dir, so every worktree of a repo reads the same
//     value, and two different repos may name the same store deliberately.
//     One `git config winze.memory <path>` is the whole opt-in.
//  3. ~/winze-memory, the historical default.
//
// The git step exists because the obvious alternative — deriving the store
// from the working directory — gives each worktree its own pile and each
// clone of a related repo another one, which is the fragmentation this is
// meant to avoid.
func memRoot() string {
	if v := os.Getenv("WINZE_MEMORY"); v != "" {
		return v
	}
	if v := gitConfigMemory(); v != "" {
		return v
	}
	return filepath.Join(home(), "winze-memory")
}

// binName resolves a winze tool binary: joined under WINZE_BIN when set, else
// the bare name so it resolves via PATH (e.g. after `make install`).
func binName(name string) string {
	if v := os.Getenv("WINZE_BIN"); v != "" {
		return filepath.Join(v, name)
	}
	return name
}

func queryBin() string { return binName("winze-query") }
func addBin() string   { return binName("winze-add") }
func editBin() string  { return binName("winze-edit") }

func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

// gitConfigMemory reads `winze.memory` from git config in the working
// directory, or "" when there is no repo, no key, or git is unavailable.
//
// The key is read without a scope flag on purpose, so --local, --global and
// --system all resolve normally. Local is the interesting one: it lives in
// .git/config, which git keeps in the common dir, so every worktree of a repo
// sees it and no worktree can hold a different value unless
// extensions.worktreeConfig is deliberately turned on.
//
// Every failure is silent and returns "". A missing key exits non-zero, and
// that is the ordinary case for a repo that never opted in — surfacing it
// would make the common path noisy.
func gitConfigMemory() string {
	cmd := exec.Command("git", "config", "--get", "winze.memory")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}
