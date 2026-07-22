// Package usagelog appends one JSON line per winze tool invocation to
// <dir>/.winze-usage.jsonl (gitignored). This is the self-telemetry that
// grounds "which winze operations do I actually perform, and how fast" in real
// behavior instead of guesses — the data the speed-priority work was missing.
//
// Privacy by construction: it records the tool name, the FLAG NAMES only (never
// values, queries, paths, or note text), and wall-clock ms. So the log is safe
// even in a shared checkout — it captures the shape of usage, not its content.
package usagelog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Log appends one record for a completed invocation. Call deferred from a
// tool's main once the corpus dir is known: defer usagelog.Log(dir, "query",
// os.Args[1:], time.Now()). Best-effort — any error is silently ignored, since
// telemetry must never affect the operation it measures.
func Log(dir, tool string, argv []string, start time.Time) {
	flags := make([]string, 0, len(argv))
	for _, a := range argv {
		if !strings.HasPrefix(a, "-") {
			continue // a value, path, or query — never recorded
		}
		f := strings.TrimLeft(a, "-")
		if i := strings.IndexByte(f, '='); i >= 0 {
			f = f[:i]
		}
		if f != "" {
			flags = append(flags, f)
		}
	}
	rec := struct {
		TS     string   `json:"ts"`
		Tool   string   `json:"tool"`
		Flags  []string `json:"flags"`
		WallMS int64    `json:"wall_ms"`
	}{
		TS:     start.UTC().Format(time.RFC3339),
		Tool:   tool,
		Flags:  flags,
		WallMS: time.Since(start).Milliseconds(),
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, ".winze-usage.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}
