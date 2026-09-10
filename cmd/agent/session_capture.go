package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/transcript"
)

// runSessionCapture reads a SessionEnd hook payload and auto-indexes the
// session's last substantial assistant turn into the winze store --
// mirrors cmd/longmemeval's outcome shape (the best-measured of everything
// tried: outcome 51-53%, claims 50%, topk=3 45%, topk=1 26%; see
// ROADMAP.md) rather than the longest turn or every turn, both measured
// worse. Never fails the hook: any error, unmet gate, or empty result
// exits silently, matching runRecallHook's posture, so a broken capture
// can't block a session from closing.
//
// Mirrors handleRemember's own orchestration (dedup -> add -> document ->
// commit, split across writeSessionCapture/documentSessionRecurrence)
// rather than calling handleRemember itself, so the Origin string can mark
// this as auto-captured ("session-end-capture <id> <time>", distinct from
// an explicit call's "winze_remember <time>") without changing
// handleRemember's public MCP argument contract. role stays at execAdd's
// own "Concept" default -- deliberately not a new role type in
// winze-memory's schema, since Origin is what marks this as auto-captured,
// not the role. Link-suggestion and the onsetter advisory are both
// skipped: their only output is text meant for whoever reads
// winze_remember's return value, and nobody reads a hook's stdout that way.
//
// Transcript parsing lives in internal/transcript, shared with cmd/query's
// tier-2 search -- this function is a thin caller.
func runSessionCapture() {
	transcriptPath := parseSessionEndInput()
	if transcriptPath == "" {
		return
	}

	// Only capture where a store actually exists to write into -- the same
	// gate capture-guard uses. Verified today this activates in exactly the
	// projects that deliberately opted in (no global/system git config sets
	// winze.store broadly, and the bare ~/winze-memory fallback doesn't
	// exist on this machine) -- if that directory is ever created, this
	// gate widens along with capture-guard's, worth knowing, not a bug.
	if !storeRootConfigured() || !storeHasNoRemote(storeRoot()) {
		return
	}

	note, err := transcript.LastSubstantial(transcriptPath)
	if err != nil || note == "" {
		return
	}

	sessionID := strings.TrimSuffix(filepath.Base(transcriptPath), ".jsonl")
	shortID := sessionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	now := time.Now().UTC()
	title := fmt.Sprintf("session-end capture %s %s", now.Format("2006-01-02"), shortID)
	origin := fmt.Sprintf("session-end-capture %s %s", sessionID, now.Format(time.RFC3339))

	dd := checkDedup(note, false)
	if dd.block != nil {
		documentSessionRecurrence(dd.blockedAgainst, note, origin)
		return
	}
	writeSessionCapture(note, title, origin)
}

// storeHasNoRemote reports whether root's git repo has zero configured
// remotes. Every real winze store is local-only by design (verified
// directly for winze-memory and germline-memory: both zero remotes, both
// carry a pre-push hook that hard-blocks regardless) -- this is a
// defensive belt-and-suspenders check on top of that, not the only guard.
// A store this can't verify, or that has any remote at all, is treated as
// unsafe for auto-capture and skipped rather than risk turning session
// content into something that could ever be pushed.
func storeHasNoRemote(root string) bool {
	out, err := exec.Command("git", "-C", root, "remote").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}

// documentSessionRecurrence attaches note as a Documented claim on the
// entity checkDedup blocked against, best-effort -- mirrors
// handleRemember's own dedup-block attach, tagged with origin instead of
// an explicit call's.
func documentSessionRecurrence(against, note, origin string) {
	if against == "" {
		return
	}
	if _, err := execDocument(against, note, origin); err == nil {
		_, _ = gitCommitMemory(fmt.Sprintf("document session-end recurrence against %s", against))
	}
}

// parseSessionEndInput reads and validates the hook payload from stdin,
// returning "" when this call isn't a SessionEnd event worth acting on --
// wrong event, missing transcript path, or unparseable input.
func parseSessionEndInput() (transcriptPath string) {
	data, err := readAllStdin()
	if err != nil || len(data) == 0 {
		return ""
	}
	var in hookInput
	if json.Unmarshal(data, &in) != nil || in.HookEventName != "SessionEnd" {
		return ""
	}
	return in.TranscriptPath
}

// writeSessionCapture creates a new entity for note and documents it with
// origin, mirroring handleRemember's add-then-document-then-commit shape.
func writeSessionCapture(note, title, origin string) {
	addOut, err := execAdd(note, "Concept", title)
	if err != nil {
		return
	}
	newVar := createdVar(addOut)
	if newVar == "" {
		return
	}
	_, _ = execDocument(newVar, note, origin)
	_, _ = gitCommitMemory(note)
}
