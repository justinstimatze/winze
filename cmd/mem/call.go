package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// runCall invokes one memory tool from the command line and prints its text
// result, so a host that cannot speak MCP over stdio can still reach the same
// four tools.
//
// It dispatches to the very same handlers `serve` registers rather than
// reimplementing them. A second code path for recall and remember would be a
// second place for the dedup check and the build gate to drift out of, and
// those two are the whole reason the tools are safe to expose.
//
//	winze-mem call winze_recall '{"query":"the trip critic","limit":3}'
//
// Exits non-zero when the tool reports an error, so a caller can branch on the
// status instead of parsing prose.
func runCall(args []string) {
	handlers := map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){
		"winze_recall":   handleRecall,
		"winze_remember": handleRemember,
		"winze_update":   handleUpdate,
		"winze_link":     handleLink,
	}
	if len(args) < 1 {
		names := make([]string, 0, len(handlers))
		for n := range handlers {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "usage: winze-mem call <tool> ['<json-args>']\n  tools: %s\n", strings.Join(names, ", "))
		os.Exit(2)
	}
	h, ok := handlers[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown tool %q\n", args[0])
		os.Exit(2)
	}

	argMap := map[string]any{}
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		if err := json.Unmarshal([]byte(args[1]), &argMap); err != nil {
			fmt.Fprintf(os.Stderr, "args must be a JSON object: %v\n", err)
			os.Exit(2)
		}
	}

	var req mcp.CallToolRequest
	req.Params.Name = args[0]
	req.Params.Arguments = argMap

	res, err := h(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", args[0], err)
		os.Exit(1)
	}
	for _, c := range res.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			fmt.Println(tc.Text)
		}
	}
	if res.IsError {
		os.Exit(1)
	}
}
