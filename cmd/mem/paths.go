package main

import (
	"os"
	"path/filepath"
)

// The memory store and the winze tool binaries are located by env var, so one
// winze-mem binary serves any number of stores (point WINZE_MEMORY at each) and
// runs against tooling wherever it is installed.

// memRoot is the winze-memory store (the dir holding memory.go + schema files).
// Set WINZE_MEMORY per store; defaults to ~/winze-memory.
func memRoot() string {
	if v := os.Getenv("WINZE_MEMORY"); v != "" {
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
