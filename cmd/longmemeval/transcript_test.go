package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFlattenContentHandlesBothWireShapes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"bare string", `"just text"`, "just text"},
		{"block array", `[{"type":"text","text":"first"},{"type":"text","text":"second"}]`, "first\n\nsecond"},
		{"tool blocks dropped", `[{"type":"tool_use","name":"Bash"},{"type":"text","text":"kept"}]`, "kept"},
		{"thinking dropped", `[{"type":"thinking","thinking":"scratch"},{"type":"text","text":"said"}]`, "said"},
		{"empty array", `[]`, ""},
		{"unparseable", `12345`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := flattenContent(json.RawMessage(c.raw)); got != c.want {
				t.Errorf("flattenContent(%s) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestReadProjectTranscriptsFiltersByMinTurns(t *testing.T) {
	dir := t.TempDir()
	one := `{"type":"user","timestamp":"2026-06-01T00:00:00Z","message":{"role":"user","content":"a"}}`
	two := one + "\n" + `{"type":"assistant","timestamp":"2026-06-01T00:01:00Z","message":{"role":"assistant","content":"b"}}`
	if err := os.WriteFile(filepath.Join(dir, "short.jsonl"), []byte(one+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "long.jsonl"), []byte(two+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readProjectTranscripts(dir, 2)
	if err != nil {
		t.Fatalf("readProjectTranscripts: %v", err)
	}
	if len(got) != 1 || got[0].ID != "long" {
		t.Errorf("got %d sessions, want just \"long\"", len(got))
	}
	if _, err := readProjectTranscripts(dir, 99); err == nil {
		t.Error("want an error when nothing clears minTurns, got nil")
	}
}

func TestReadProjectTranscriptsOrdersByFirstRecordNotMtime(t *testing.T) {
	dir := t.TempDir()
	// Written newest-first so that mtime order is the reverse of real order --
	// which is what claude-mv's transcript rewrite does to a whole project.
	write := func(name, ts, body string) {
		line := `{"type":"user","timestamp":"` + ts + `","cwd":"/w","message":{"role":"user","content":"` + body + `"}}`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(line+"\n"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	write("c.jsonl", "2026-08-01T00:00:00Z", "newest")
	write("b.jsonl", "2026-07-01T00:00:00Z", "middle")
	write("a.jsonl", "2026-06-01T00:00:00Z", "oldest")

	got, err := readProjectTranscripts(dir, 1)
	if err != nil {
		t.Fatalf("readProjectTranscripts: %v", err)
	}
	var order []string
	for _, s := range got {
		order = append(order, s.Turns[0].Content)
	}
	if want := []string{"oldest", "middle", "newest"}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestReadTranscriptSkipsSidechainsMetaAndMalformed(t *testing.T) {
	path := writeTranscript(t, "mixed.jsonl",
		`{"type":"user","timestamp":"2026-06-01T10:00:00Z","cwd":"/w","message":{"role":"user","content":"kept"}}`,
		`{"type":"user","timestamp":"2026-06-01T10:01:00Z","isSidechain":true,"message":{"role":"user","content":"subagent"}}`,
		`{"type":"user","timestamp":"2026-06-01T10:02:00Z","isMeta":true,"message":{"role":"user","content":"meta"}}`,
		`{"type":"attachment","timestamp":"2026-06-01T10:03:00Z"}`,
		`{"type":"user","timestamp":"2026-06-01T10:0`, // truncated: an in-flight append
		`{"type":"assistant","timestamp":"2026-06-01T10:05:00Z","message":{"role":"assistant","content":"also kept"}}`,
	)
	sess, err := readTranscript(path)
	if err != nil {
		t.Fatalf("readTranscript: %v", err)
	}
	var got []string
	for _, turn := range sess.Turns {
		got = append(got, turn.Content)
	}
	want := []string{"kept", "also kept"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("turns = %v, want %v", got, want)
	}
	if sess.Cwd != "/w" {
		t.Errorf("Cwd = %q, want /w", sess.Cwd)
	}
	// The attachment record carries no prose but still widens the span.
	if want := 5 * time.Minute; sess.Span() != want {
		t.Errorf("Span() = %v, want %v", sess.Span(), want)
	}
}

// TestReadTranscriptSurvivesARecordOverScannersLimit is the reason this reader
// uses bufio.Reader and not bufio.Scanner. Scanner caps a token at 64 KB and
// reports the overrun by returning false from Scan(), which every ordinary
// read loop treats as a clean end of file -- so a Scanner-based reader passes
// every other test in this file and silently drops the rest of any session
// containing one pasted file or one large tool result. That is precisely the
// densest, most recall-worthy session, which makes the bug invisible where it
// matters most.
func TestReadTranscriptSurvivesARecordOverScannersLimit(t *testing.T) {
	huge := strings.Repeat("x", 200*1024) // 200 KB, over Scanner's 64 KB cap
	path := writeTranscript(t, "big.jsonl",
		`{"type":"user","timestamp":"2026-06-01T10:00:00Z","cwd":"/w","message":{"role":"user","content":"`+huge+`"}}`,
		`{"type":"assistant","timestamp":"2026-06-01T10:01:00Z","message":{"role":"assistant","content":"after the big one"}}`,
	)
	sess, err := readTranscript(path)
	if err != nil {
		t.Fatalf("readTranscript: %v", err)
	}
	if len(sess.Turns) != 2 {
		t.Fatalf("got %d turns, want 2 -- the record after the oversized one was dropped", len(sess.Turns))
	}
	if len(sess.Turns[0].Content) != len(huge) {
		t.Errorf("oversized turn truncated to %d bytes, want %d", len(sess.Turns[0].Content), len(huge))
	}
	if sess.Turns[1].Content != "after the big one" {
		t.Errorf("turn after the oversized record = %q", sess.Turns[1].Content)
	}
}

// writeTranscript writes lines as a JSONL transcript and returns its path.
func writeTranscript(t *testing.T, name string, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

// TestCleanAskStripsHostWrapperBlocks pins the thing that would quietly ruin
// the later-ask probe: a user record carries the operator's words wrapped in
// host bookkeeping, and system-reminder text is near-identical across every
// session in the corpus. A probe built from one would match every note equally
// and the measurement would read as a retrieval failure rather than a fixture bug.
func TestCleanAskStripsHostWrapperBlocks(t *testing.T) {
	raw := "<system-reminder>\nCodebase instructions follow.\n</system-reminder>\n" +
		"<command-name>/compact</command-name>\n" +
		"actually  hold on\nthe dedup gate is the part that's wrong"
	got := cleanAsk(raw)
	if strings.Contains(got, "Codebase instructions") || strings.Contains(got, "compact") {
		t.Errorf("wrapper text survived: %q", got)
	}
	if want := "actually hold on the dedup gate is the part that's wrong"; got != want {
		t.Errorf("cleanAsk = %q, want %q", got, want)
	}
}

// TestLaterAskEmptyWhenTheSessionHasNoSecondAsk covers the branch the replay
// counts separately: a one-ask session is not a probe failure, it is a session
// with nothing to probe with, and folding the two together would understate
// recall.
func TestLaterAskEmptyWhenTheSessionHasNoSecondAsk(t *testing.T) {
	s := &transcriptSession{Turns: []Turn{
		{Role: "user", Content: "the only ask in this session, long enough to clear the floor"},
		{Role: "assistant", Content: "a long assistant reply that must not be picked up as a later ask"},
		{Role: "user", Content: "ok"},
	}}
	if got := s.LaterAsk(); got != "" {
		t.Errorf("LaterAsk = %q, want empty", got)
	}
}

// TestLaterAskSkipsTheOpeningAskAndShortTurns covers what makes the probe
// honest. The opening ask is already inside the note, so returning it would
// make the later probe a second title probe wearing different words; and a
// corpus of real sessions is full of one-word user turns ("go", "yes",
// "continue") that name nothing and would match every note or none.
func TestLaterAskSkipsTheOpeningAskAndShortTurns(t *testing.T) {
	s := &transcriptSession{Turns: []Turn{
		{Role: "user", Content: "the opening ask, long enough to clear the length floor easily"},
		{Role: "assistant", Content: "a reply that is also long enough to clear the floor if it counted"},
		{Role: "user", Content: "go"},
		{Role: "user", Content: "the poisoned rows survived the environment repair, which is the real finding"},
		{Role: "user", Content: "yes"},
	}}
	got := s.LaterAsk()
	if !strings.HasPrefix(got, "the poisoned rows") {
		t.Errorf("LaterAsk = %q, want the one substantial non-opening user turn", got)
	}
}
