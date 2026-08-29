package memtooldemo

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/justinstimatze/winze/integrations/memtool"
)

func main() {
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
		fmt.Fprintln(os.Stderr, "memtooldemo: set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) first — this makes real, billed API calls.")
		os.Exit(1)
	}

	indexPath := os.Getenv("MEMTOOL_INDEX")
	if indexPath == "" {
		indexPath = "memtooldemo-index.json"
	}
	ex, err := memtool.NewExecutor(indexPath, os.Getenv("WINZE_AGENT_BIN"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "memtooldemo: %v\n", err)
		os.Exit(1)
	}

	loop := &memtool.Loop{
		Client:        anthropic.NewClient(),
		Model:         anthropic.ModelClaudeSonnet5,
		MaxTokens:     1024,
		Executor:      ex,
		MaxIterations: 6,
	}
	ctx := context.Background()

	fmt.Println("--- conversation 1: writing a memory ---")
	_, final1, err := loop.Run(ctx, nil, `Remember, using your memory tool, that the winze memtool demo's phrase is "periwinkle lighthouse". Store it at /memories/demo-fact.md, then confirm in one sentence that you saved it.`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memtooldemo: conversation 1: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(textOf(final1))

	fmt.Println("\n--- conversation 2: a fresh session, no shared history, recalling it ---")
	_, final2, err := loop.Run(ctx, nil, "Check your memory tool for a file at /memories/demo-fact.md and tell me exactly what it says.")
	if err != nil {
		fmt.Fprintf(os.Stderr, "memtooldemo: conversation 2: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(textOf(final2))
}

// textOf concatenates every text block in the model's final message —
// there is usually exactly one, but a refusal or multi-part reply can add more.
func textOf(m *anthropic.BetaMessage) string {
	if m == nil {
		return "(no final message)"
	}
	var out string
	for _, block := range m.Content {
		if block.Type == "text" {
			out += block.AsText().Text
		}
	}
	return out
}
