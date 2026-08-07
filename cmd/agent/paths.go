package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The store and the winze tool binaries are located by env var, so one
// winze-agent binary serves any number of stores (point WINZE_STORE at each)
// and runs against tooling wherever it is installed.
//
// "Store", not "memory": several stores are live at once (one per team or
// project, shared across every repo that names it), so naming the variable
// after any single store's directory would be reading one instance as the
// category.

// storeRoot is the winze store this agent reads and writes (the dir holding
// memory.go + the schema files).
//
// Resolution order, most explicit first:
//
//  1. $WINZE_STORE — a caller that names a store outright always wins.
//  2. `git config --get winze.store` in the working directory. Git keeps config
//     in the common dir, so every worktree of a repo reads the same value, and
//     two different repos may name the same store deliberately. One
//     `git config winze.store <path>` is the whole opt-in.
//  3. ~/winze-memory, the historical default.
//
// The git step exists because the obvious alternative — deriving the store
// from the working directory — gives each worktree its own pile and each
// clone of a related repo another one, which is the fragmentation this is
// meant to avoid.
func storeRoot() string {
	if v := os.Getenv("WINZE_STORE"); v != "" {
		return v
	}
	if v := gitConfigStore(); v != "" {
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

// gitConfigStore reads `winze.store` from git config in the working directory,
// or "" when there is no repo, no key, or git is unavailable.
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
func gitConfigStore() string {
	cmd := exec.Command("git", "config", "--get", "winze.store")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

// storeRootConfigured reports whether this working directory has actually opted
// in to a winze store, as opposed to storeRoot() merely handing back its
// last-resort default.
//
// storeRoot always returns a path, so it cannot distinguish "this project uses
// a winze store" from "nobody said, here is ~/winze-memory". The capture guard
// needs that distinction: it blocks native auto-memory writes and tells the
// caller to use the winze store instead, which is only sound advice where a
// store exists. Fired in a project with no store, it would block the write and
// offer nowhere to put it.
//
// An explicit env var or git config key is opt-in by construction. The bare
// default counts only when the directory is really there, which keeps a
// pre-existing ~/winze-memory user working without requiring them to set
// anything.
func storeRootConfigured() bool {
	if os.Getenv("WINZE_STORE") != "" || gitConfigStore() != "" {
		return true
	}
	fi, err := os.Stat(filepath.Join(home(), "winze-memory"))
	return err == nil && fi.IsDir()
}
