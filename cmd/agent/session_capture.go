package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// flattenSessionContent renders a message's content to plain text. Claude
// Code writes content either as a bare string or as an array of typed
// blocks; only text blocks carry prose. thinking blocks are dropped
// deliberately -- they are the model's scratch work, never something the
// operator saw, so capturing them would be both the wrong content and a
// privacy overreach past what a session-end summary should hold.
func flattenSessionContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var b strings.Builder
	for _, blk := range blocks {
		text := strings.TrimSpace(blk.Text)
		if blk.Type != "text" || text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(text)
	}
	return b.String()
}

// lastSubstantialAssistantTurn returns the session's final assistant text
// block with len >= 40 (the same substantiveness floor cmd/longmemeval's
// midpointOutcome uses -- the best-measured shape of everything tried, see
// ROADMAP.md), or "" if none clears it. Production has no held-out probe to
// walk toward the way the benchmark does, so there is nothing to exclude:
// this is simply the last qualifying turn in the whole transcript.
func lastSubstantialAssistantTurn(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var best string
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		raw, readErr := r.ReadString('\n')
		if len(raw) > 0 {
			var line sessionEndLine
			if json.Unmarshal([]byte(raw), &line) == nil &&
				!line.IsSidechain && !line.IsMeta &&
				line.Type == "assistant" && line.Message != nil {
				if text := flattenSessionContent(line.Message.Content); len(text) >= 40 {
					best = text
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return "", readErr
		}
	}
	return best, nil
}

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
// commit) rather than calling handleRemember itself, so the Origin string
// can mark this as auto-captured ("session-end-capture <id> <time>",
// distinct from an explicit call's "winze_remember <time>") without
// changing handleRemember's public MCP argument contract. role stays at
// execAdd's own "Concept" default -- deliberately not a new role type in
// winze-memory's schema, since Origin is what marks this as auto-captured,
// not the role. Link-suggestion and the onsetter advisory are both skipped:
// their only output is text meant for whoever reads winze_remember's
// return value, and nobody reads a hook's stdout that way.
func runSessionCapture() {
	data, err := readAllStdin()
	if err != nil || len(data) == 0 {
		return
	}
	var in hookInput
	if json.Unmarshal(data, &in) != nil || in.HookEventName != "SessionEnd" || in.TranscriptPath == "" {
		return
	}

	// Only capture where a store actually exists to write into -- the same
	// gate capture-guard uses. Verified today this activates in exactly the
	// projects that deliberately opted in (no global/system git config sets
	// winze.store broadly, and the bare ~/winze-memory fallback doesn't
	// exist on this machine) -- if that directory is ever created, this
	// gate widens along with capture-guard's, worth knowing, not a bug.
	if !storeRootConfigured() {
		return
	}
	if !storeHasNoRemote(storeRoot()) {
		return
	}

	note, err := lastSubstantialAssistantTurn(in.TranscriptPath)
	if err != nil || note == "" {
		return
	}

	sessionID := strings.TrimSuffix(filepath.Base(in.TranscriptPath), ".jsonl")
	shortID := sessionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	now := time.Now().UTC()
	title := fmt.Sprintf("session-end capture %s %s", now.Format("2006-01-02"), shortID)
	origin := fmt.Sprintf("session-end-capture %s %s", sessionID, now.Format(time.RFC3339))

	dd := checkDedup(note, false)
	if dd.block != nil {
		if dd.blockedAgainst != "" {
			if _, derr := execDocument(dd.blockedAgainst, note, origin); derr == nil {
				_, _ = gitCommitMemory(fmt.Sprintf("document session-end recurrence against %s", dd.blockedAgainst))
			}
		}
		return
	}

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

// sessionEndLine is the subset of a Claude Code transcript record
// session-capture needs. Deliberately not shared with cmd/longmemeval's
// near-identical transcriptLine: that package is a benchmark harness, this
// is production code, and importing one into the other would couple a
// shipped binary to a dogfood test tool for a handful of struct fields.
type sessionEndLine struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	IsMeta      bool   `json:"isMeta"`
	Message     *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}
