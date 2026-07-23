// Package events is winze's fleet event stream: a single append-only JSONL log
// that winze tools write to as they work (metabolism phase decisions, meld and
// unmeld), so a dashboard (winze-observatory) or an external watcher can render
// what the fleet is actually doing from one tail target instead of six.
//
// The contract is one JSON object per line:
//
//	{"ts":1784792000,"instance":"/abs/corpus","name":"winze","kind":"phase",
//	 "payload":{"phase":"trip","decision":"allow","reason":"..."}}
//
// Kinds in use: cycle_start, phase, cycle_end (metabolism); meld, unmeld
// (winze-meld). Writing is best-effort — a logging failure never fails the
// caller, because event telemetry must never break metabolism or a meld.
package events

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Event is one line of the stream.
type Event struct {
	TS       int64          `json:"ts"`       // unix seconds
	Instance string         `json:"instance"` // absolute corpus dir the event is about
	Name     string         `json:"name"`     // dir basename, for display
	Kind     string         `json:"kind"`     // phase | cycle_start | cycle_end | meld | unmeld
	Payload  map[string]any `json:"payload,omitempty"`
}

// LogPath resolves the stream location: $WINZE_EVENTS, else
// $XDG_STATE_HOME/winze/events.jsonl, else ~/.local/state/winze/events.jsonl.
func LogPath() string {
	if v := os.Getenv("WINZE_EVENTS"); v != "" {
		return v
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "winze", "events.jsonl")
}

// Emit appends one event for the given corpus dir. Best-effort and safe to call
// from any winze tool; concurrent writers from separate processes serialize via
// an advisory flock on the log fd.
func Emit(instanceDir, kind string, payload map[string]any) {
	abs := instanceDir
	if a, err := filepath.Abs(instanceDir); err == nil {
		abs = a
	}
	write(Event{
		TS:       time.Now().Unix(),
		Instance: abs,
		Name:     filepath.Base(abs),
		Kind:     kind,
		Payload:  payload,
	})
}

func write(ev Event) {
	path := LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fd := int(f.Fd())
	if syscall.Flock(fd, syscall.LOCK_EX) != nil {
		return
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if data, err := json.Marshal(ev); err == nil {
		f.Write(append(data, '\n'))
	}
}

// ReadRecent returns up to the last n events from the stream, oldest first.
// Missing/unreadable log is not an error — it returns nil. Used by consumers
// (the observatory) to seed and poll the fleet's recent activity.
func ReadRecent(n int) []Event {
	f, err := os.Open(LogPath())
	if err != nil {
		return nil
	}
	defer f.Close()
	// ring buffer of the last n raw lines
	buf := make([]string, 0, n)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if len(buf) < n {
			buf = append(buf, line)
		} else {
			copy(buf, buf[1:])
			buf[n-1] = line
		}
	}
	out := make([]Event, 0, len(buf))
	for _, line := range buf {
		var ev Event
		if json.Unmarshal([]byte(line), &ev) == nil {
			out = append(out, ev)
		}
	}
	return out
}
