package transcript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
)

// substantialFloor is the minimum flattened-text length a turn must clear
// to count as substantial -- the same floor cmd/longmemeval's
// midpointOutcome/ArcAsks use, so production and the benchmark that
// validated this shape agree on what "substantial" means.
const substantialFloor = 40

// AllSubstantial returns every substantial assistant turn in the transcript
// at path, in order. Reads with a bufio.Reader rather than a bufio.Scanner
// on purpose -- a real transcript record is routinely megabytes (a pasted
// file, a large tool result), and Scanner fails a long line with
// bufio.ErrTooLong, which silently truncates exactly the densest turns.
func AllSubstantial(path string) ([]Turn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var turns []Turn
	r := bufio.NewReaderSize(f, 1<<20)
	idx := 0
	for {
		raw, readErr := r.ReadString('\n')
		if text := substantialText(raw); text != "" {
			turns = append(turns, Turn{Text: text, Index: idx})
		}
		idx++
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, readErr
		}
	}
	return turns, nil
}

// FlattenContent renders a message's content to plain text. Claude Code
// writes content either as a bare string or as an array of typed blocks;
// only text blocks carry prose. thinking blocks are dropped deliberately --
// they are the model's scratch work, never something the operator saw, so
// capturing them would be both the wrong content and a privacy overreach
// past what any consumer of this package should hold.
func FlattenContent(raw json.RawMessage) string {
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

// LastSubstantial returns the session's final assistant text block clearing
// substantialFloor, or "" if none clears it. Production has no held-out
// probe to walk toward the way cmd/longmemeval's benchmark does, so there
// is nothing to exclude: this is simply the last qualifying turn in the
// whole transcript -- the best-measured shape of everything tried (see
// ROADMAP.md: outcome-equivalent 51-53% hit@5 vs. richness/multi-note's
// 26%/45%).
func LastSubstantial(path string) (string, error) {
	turns, err := AllSubstantial(path)
	if err != nil || len(turns) == 0 {
		return "", err
	}
	return turns[len(turns)-1].Text, nil
}

// substantialText parses one raw transcript line and returns its flattened
// assistant text if it clears substantialFloor, or "" for a malformed line,
// a non-assistant turn, or a short one.
func substantialText(raw string) string {
	if raw == "" {
		return ""
	}
	var line Line
	if json.Unmarshal([]byte(raw), &line) != nil || !line.IsCapturableAssistantTurn() {
		return ""
	}
	text := FlattenContent(line.Message.Content)
	if len(text) < substantialFloor {
		return ""
	}
	return text
}

// IsCapturableAssistantTurn reports whether l is a real, non-sidechain,
// non-meta assistant message -- the same filter cmd/longmemeval's
// transcriptLine.isProse applies, narrowed to assistant turns only since
// that's the only role any current consumer reads.
func (l Line) IsCapturableAssistantTurn() bool {
	return l.Type == "assistant" && !l.IsSidechain && !l.IsMeta && l.Message != nil
}

// Line is the subset of a Claude Code transcript record any consumer needs.
type Line struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	IsMeta      bool   `json:"isMeta"`
	Message     *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// Turn is one substantial assistant turn extracted from a transcript.
type Turn struct {
	Text  string
	Index int // zero-based position among ALL lines read, not just substantial ones
}
