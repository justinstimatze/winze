package memtool

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// TestExecuteOneCreateThenView pins the JSON round-trip a real API response
// would force: Input arrives as map[string]any, gets re-marshaled to bytes,
// then unmarshaled into BetaMemoryTool20250818CommandUnion. A hand-built map
// is the closest a unit test gets to what the SDK actually hands back.
func TestExecuteOneCreateThenView(t *testing.T) {
	e, _ := newTestExecutor(t)
	l := &Loop{Executor: e}

	create := l.executeOne(anthropic.BetaToolUseBlock{
		ID:   "toolu_1",
		Name: "memory",
		Input: map[string]any{
			"command":   "create",
			"path":      "/memories/a.md",
			"file_text": "hello",
		},
	})
	if create.OfToolResult == nil {
		t.Fatalf("executeOne did not return a tool_result block")
	}
	if create.OfToolResult.IsError.Value {
		t.Fatalf("create returned an error: %s", toolResultText(create))
	}

	view := l.executeOne(anthropic.BetaToolUseBlock{
		ID:   "toolu_2",
		Name: "memory",
		Input: map[string]any{
			"command": "view",
			"path":    "/memories/a.md",
		},
	})
	if view.OfToolResult.IsError.Value {
		t.Fatalf("view returned an error: %s", toolResultText(view))
	}
	if got := toolResultText(view); got != "hello" {
		t.Errorf("view = %q, want %q", got, "hello")
	}
}

func TestExecuteOneRejectsNonMemoryToolName(t *testing.T) {
	e, _ := newTestExecutor(t)
	l := &Loop{Executor: e}

	result := l.executeOne(anthropic.BetaToolUseBlock{ID: "toolu_1", Name: "bash", Input: map[string]any{}})
	if result.OfToolResult == nil || !result.OfToolResult.IsError.Value {
		t.Error("a non-memory tool name should return an error result")
	}
}

func toolResultText(b anthropic.BetaContentBlockParamUnion) string {
	if b.OfToolResult == nil || len(b.OfToolResult.Content) == 0 || b.OfToolResult.Content[0].OfText == nil {
		return ""
	}
	return b.OfToolResult.Content[0].OfText.Text
}
