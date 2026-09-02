package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// flattenContent renders a message's content to plain text.
//
// Claude Code writes content either as a bare string or as an array of typed
// blocks, and only text blocks carry prose: tool_use and tool_result blocks
// are the machinery of a turn rather than what was said about it. Thinking
// blocks are dropped for a sharper reason -- they are the model's scratch
// work, and scoring a memory write against reasoning the operator never saw
// would measure the wrong thing.
func flattenContent(raw json.RawMessage) string {
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

// readProjectTranscripts reads every session transcript in a Claude Code
// project directory, keeps those with at least minTurns usable turns, and
// returns them oldest-first.
//
// Chronological order is the mechanism, not a convenience. Phase 3b replays
// these writes in sequence so a session's recall is scored against a store
// that has since grown by every later session -- the "real corpus growth in
// between" the design doc asks for, and the thing a same-day trial cannot
// produce no matter how many questions it asks.
//
// Sorted on the first record's timestamp rather than file mtime: mtime moves
// whenever the file is rewritten, and claude-mv rewrites every transcript in
// a project when its directory is renamed, which would silently reorder the
// entire corpus.
func readProjectTranscripts(dir string, minTurns int) ([]*transcriptSession, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var out []*transcriptSession
	for _, p := range paths {
		sess, err := readTranscript(p)
		if err != nil {
			continue
		}
		if len(sess.Turns) < minTurns || sess.Start.IsZero() {
			continue
		}
		out = append(out, sess)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no transcripts with >=%d turns in %s", minTurns, dir)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, nil
}

// readTranscript parses one session transcript into its user and assistant
// turns, in order.
//
// Reads with a bufio.Reader rather than a bufio.Scanner on purpose. A real
// transcript record is routinely megabytes (a pasted file, a large tool
// result -- this project's own longest is 62,646 records over 122 MB), and
// Scanner fails a long line with bufio.ErrTooLong, which a caller that only
// checks Scan()'s bool reads as a clean end of file. That is a silent
// truncation of exactly the densest sessions, which are the ones worth
// reading. ReadString grows its buffer instead and has no such ceiling.
//
// A malformed record skips that line rather than discarding the session:
// these files are appended to by a live process and the last line of an
// in-flight transcript is regularly a partial write.
//
// Sidechain records (subagent transcripts) and meta records are skipped --
// neither is content the operator would later be recalling.
func readTranscript(path string) (*transcriptSession, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sess := &transcriptSession{ID: strings.TrimSuffix(filepath.Base(path), ".jsonl")}
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		raw, readErr := r.ReadString('\n')
		if len(raw) > 0 {
			var line transcriptLine
			if json.Unmarshal([]byte(raw), &line) == nil {
				sess.absorb(line)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, readErr
		}
	}
	if len(sess.Turns) == 0 {
		return nil, fmt.Errorf("no usable turns in %s", path)
	}
	return sess, nil
}

// Span is the wall-clock time the session covers.
func (s *transcriptSession) Span() time.Duration { return s.End.Sub(s.Start) }

// absorb folds one transcript record into the session, keeping the user and
// assistant prose and widening the session's time span.
func (s *transcriptSession) absorb(line transcriptLine) {
	if line.IsSidechain || line.IsMeta {
		return
	}
	if s.Cwd == "" && line.Cwd != "" {
		s.Cwd = line.Cwd
	}
	s.widenSpan(line.Timestamp)
	if !line.isProse() {
		return
	}
	if text := flattenContent(line.Message.Content); text != "" {
		s.Turns = append(s.Turns, Turn{Role: line.Message.Role, Content: text})
	}
}

// transcriptLine is the subset of a Claude Code transcript record this reader
// needs. The real format carries ~40 top-level keys across a dozen record
// types (attachment, queue-operation, permission-mode, file-history-snapshot,
// ai-title, ...). Everything not named here is ignored rather than modelled,
// deliberately: the schema belongs to the host and will drift, and a reader
// that models fields it does not use turns every host upgrade into a
// compile error for no gain.
type transcriptLine struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	Cwd         string `json:"cwd"`
	SessionID   string `json:"sessionId"`
	IsSidechain bool   `json:"isSidechain"`
	IsMeta      bool   `json:"isMeta"`
	Message     *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// transcriptSession is one real Claude Code session, read from a project's
// JSONL transcript under ~/.claude/projects/<encoded-cwd>/.
//
// This is the input end Phase 3b needs and dataset.go structurally cannot
// supply. LongMemEval hands over sessions with gold question/answer pairs
// already annotated; an operator's own sessions arrive with neither. What
// they carry instead is the one property a benchmark file cannot fake --
// real calendar spacing between writes. docs/agent-identity-integration.md
// names precisely that as the gap its N=1 trial left open: "no multi-day gap
// (the 'cold' session was seconds later, not days), and no test under real
// corpus growth."
type transcriptSession struct {
	ID    string
	Start time.Time
	End   time.Time
	Cwd   string
	Turns []Turn
}

// isProse reports whether a record carries something the operator or the model
// actually said. Everything else in the format -- attachments, snapshots,
// permission-mode changes, ai-title records -- is host bookkeeping.
func (l transcriptLine) isProse() bool {
	return (l.Type == "user" || l.Type == "assistant") && l.Message != nil
}

// widenSpan grows the session's [Start, End] to cover one record's timestamp.
//
// Every record type widens the span, not just the prose-bearing ones: an
// attachment or a queue-operation is still time the session was open, and a
// session's real duration is what Phase 3b spaces its writes by.
func (s *transcriptSession) widenSpan(stamp string) {
	ts, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return
	}
	if s.Start.IsZero() || ts.Before(s.Start) {
		s.Start = ts
	}
	if ts.After(s.End) {
		s.End = ts
	}
}
