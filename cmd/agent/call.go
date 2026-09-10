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

func runCall(args []string) {
	handlers := map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){
		"winze_recall":            handleRecall,
		"winze_recall_transcript": handleRecallTranscript,
		"winze_remember":          handleRemember,
		"winze_update":            handleUpdate,
		"winze_link":              handleLink,
	}
	if len(args) < 1 {
		names := make([]string, 0, len(handlers))
		for n := range handlers {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "usage: winze-agent call <tool> ['<json-args>']\n  tools: %s\n", strings.Join(names, ", "))
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
