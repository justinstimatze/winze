// Command winze-agent is the agentic read/write interface to a winze store —
// the door an agent remembers and recalls through, as against winze-mcp's
// analytic read surface over a corpus. Five faces:
//
//	winze-agent init <dir>      # scaffold a new store: schema, go.mod, git,
//	                            # and the build gate run before the first commit
//	winze-agent serve           # MCP server: winze_remember / winze_recall /
//	                            # winze_update / winze_link
//	winze-agent recall-hook     # SessionStart / UserPromptSubmit hook: injects
//	                            # associative recall so memory surfaces reflexively
//	winze-agent capture-guard   # PreToolUse hook: blocks native-memory writes
//	                            # where a winze store is configured
//	winze-agent call <tool>     # the same handlers from a shell, for non-MCP hosts
//
// All of them reuse the built winze binaries (winze-query, winze-add,
// winze-edit) as the tested logic — this binary is a thin orchestrator, not a
// reimplementation.
//
// It serves whatever store storeRoot resolves, which is deliberately not one
// fixed directory: several stores are live at once, one per team or project,
// each shared by every repo that names it. See docs/agent.md.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: winze-agent <init|serve|recall-hook|capture-guard|call>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		runInit(os.Args[2:])
	case "recall-hook":
		runRecallHook()
	case "serve":
		runServe()
	case "capture-guard":
		runCaptureGuard()
	case "call":
		runCall(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "winze-agent: unknown subcommand %q (want init|serve|recall-hook|capture-guard|call)\n", os.Args[1])
		os.Exit(2)
	}
}
