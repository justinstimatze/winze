package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// appendRawLog appends one entry to the raw tier -- the MemPalace-shaped
// recovery path Phase 3a exists for. A fact winze's typed extraction drops
// (dedup-blocked, a build-gate failure, a lossy Brief truncation) is not
// unrecoverable, it is one grep away in this file.
//
// Best-effort by design: a failure here (disk full, permissions, a store
// directory that does not exist yet) must never block the typed write or the
// caller's response, so errors are swallowed rather than surfaced -- the same
// posture checkOnsetterGate takes for its own parse errors.
func appendRawLog(tool, varName, note string) {
	entry := rawLogEntry{Time: time.Now().UTC().Format(time.RFC3339), Tool: tool, Var: varName, Note: note}
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	f, err := os.OpenFile(rawLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	line = append(line, '\n')
	_, _ = f.Write(line)
}

// rawLogPath is where the append-only raw tier lives, alongside memory.go.
// One shared file for the whole store today, not partitioned per session or
// nick, because winze-agent has no per-nick session files yet --
// docs/agent-identity-integration.md's session_<nick>.go shape is still
// gated. Re-partition this once that lands.
func rawLogPath() string {
	return filepath.Join(storeRoot(), "raw.jsonl")
}

// rawLogEntry is one line of the raw tier: the caller's note text, verbatim,
// independent of whether the typed write that follows succeeds.
type rawLogEntry struct {
	Time string `json:"time"`
	Tool string `json:"tool"`
	Var  string `json:"var,omitempty"`
	Note string `json:"note"`
}
