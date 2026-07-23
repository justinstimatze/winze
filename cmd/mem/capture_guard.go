package main

// capture-guard — a PreToolUse hook that makes winze-memory the memory store
// by deterministic boundary, not by hope. Matched on Write|Edit|MultiEdit: if
// the target is a native Claude Code memory file, it exits 2 (hard block) and
// tells the model to capture through winze_remember / winze_update instead.
//
// This is defn's lesson applied to memory: a soft instruction in CLAUDE.md is
// not sticky across sessions — the model drifts back to the path of least
// resistance (native memory.md). Only the exit-2 boundary is enforceable. The
// recall half is already covered by the injection hook ("can't forget what's
// in context"); this is the capture half ("can't write to the wrong store").
//
// Bypass, when a native-memory write is genuinely intended:
//   - touch ~/.claude-allow-memory-edit   (the load-bearing path: env vars set
//     inside a session don't propagate to later hook invocations, so a
//     filesystem sentinel is the only reliable per-session unblock), OR
//   - export CLAUDE_ALLOW_MEMORY_EDIT=1   (works when set in the launching env)
//
// Fail-open: any parse/read failure exits 0. A broken guard must never wedge
// the session by blocking every edit.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type preToolInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

func runCaptureGuard() {
	// Bypass first — cheapest, and the intended escape hatch.
	if os.Getenv("CLAUDE_ALLOW_MEMORY_EDIT") != "" || sentinelExists() {
		os.Exit(0)
	}

	data, err := readAllStdin()
	if err != nil || len(data) == 0 {
		os.Exit(0) // fail-open
	}
	var in preToolInput
	if err := json.Unmarshal(data, &in); err != nil {
		os.Exit(0) // fail-open
	}

	if !isMutateTool(in.ToolName) || !isNativeMemoryPath(in.ToolInput.FilePath) {
		os.Exit(0)
	}

	fmt.Fprint(os.Stderr, blockMessage)
	os.Exit(2)
}

func isMutateTool(name string) bool {
	switch name {
	case "Write", "Edit", "MultiEdit":
		return true
	}
	return false
}

// isNativeMemoryPath reports whether p is a Claude Code native memory file
// (…/.claude/projects/<slug>/memory/…). That directory is the store this guard
// redirects away from.
func isNativeMemoryPath(p string) bool {
	if p == "" {
		return false
	}
	p = filepath.ToSlash(p)
	return strings.Contains(p, "/.claude/projects/") && strings.Contains(p, "/memory/")
}

func sentinelExists() bool {
	_, err := os.Stat(filepath.Join(home(), ".claude-allow-memory-edit"))
	return err == nil
}

const blockMessage = `BLOCKED: native memory-file write.

winze-memory is the memory store now — capture through the tools, not native memory.md:
  • winze_remember(note, title)  — store a new durable fact/decision
  • winze_update(var, note)      — revise an existing memory (see winze_recall for the var)

If this native-memory write is genuinely intended, bypass for the session:
  touch ~/.claude-allow-memory-edit
(then retry the edit).
`
