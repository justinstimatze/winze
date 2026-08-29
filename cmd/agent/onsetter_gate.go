package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/justinstimatze/onsetter/ask"
)

// checkOnsetterGate runs note past every ask in the CLAUDE.md onsetterClaudeMD
// resolves, and returns the ones that fired. It no-ops (nil, nil) when no
// CLAUDE.md is configured or found — advisory only, never a reason to block a
// memory write — and it is a path-less Match call, so an ask using in:,
// not-in:, or untouched: cannot fire here by construction (ask.Ask.Match).
func checkOnsetterGate(note string) ([]onsetterHit, error) {
	path := onsetterClaudeMD()
	if path == "" {
		return nil, nil
	}
	asks, err := ask.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("onsetter: parsing %s: %w", path, err)
	}
	base := filepath.Dir(path)
	var fired []onsetterHit
	for _, a := range asks {
		res := a.Match(ask.Edit{New: note})
		if !res.OK {
			continue
		}
		fired = append(fired, onsetterHit{Where: a.Where(base), Matched: res.Matched, Body: a.Body})
	}
	return fired, nil
}

// onsetterAdvisory renders fired asks as the text appended to a
// winze_remember/winze_update result — a question the calling agent can
// dismiss in one sentence if it's not applicable, per onsetter's own posture.
func onsetterAdvisory(fired []onsetterHit) string {
	if len(fired) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nonsetter: this note reads as rule-shaped — did you mean a hook instead of a memory?")
	for _, h := range fired {
		fmt.Fprintf(&b, "\n- %s (matched %q): %s", h.Where, h.Matched, h.Body)
	}
	return b.String()
}

// onsetterHit is one ask that fired against a pending winze_remember or
// winze_update note — the same "want a hook instead?" question onsetter
// already asks of a rule-shaped CLAUDE.md edit, aimed at winze-agent's write
// path instead (docs/onsetter-integration.md).
type onsetterHit struct {
	Where   string // ask.Where(base) — e.g. "CLAUDE.md:12"
	Matched string // the text the ask's content gate matched
	Body    string // the ask's prose
}

// onsetterCheckOverride, when non-empty, forces onsetterClaudeMD to use this
// CLAUDE.md instead of its normal env-then-store-root resolution. Set by
// winze-agent serve --onsetter-check=<path>.
var onsetterCheckOverride string

// serveArgs is what `serve` was asked to do, parsed from its flags.
type serveArgs struct {
	onsetterCheck string
}

// parseServeArgs parses winze-agent serve's flags. Unknown flags are fatal,
// mirroring parseInitArgs's posture.
func parseServeArgs(args []string) serveArgs {
	var a serveArgs
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--onsetter-check="):
			a.onsetterCheck = strings.TrimPrefix(arg, "--onsetter-check=")
		default:
			fatalf("serve: unknown flag %q", arg)
		}
	}
	return a
}
