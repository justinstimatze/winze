// Command winze-mem is the agentic-first interface to the private winze-memory
// store. Two faces:
//
//	winze-mem recall-hook     # SessionStart / UserPromptSubmit hook: injects
//	                          # associative recall so memory surfaces reflexively
//	winze-mem serve           # MCP server: winze_remember / winze_recall tools
//
// Both reuse the built winze binaries (winze-query, winze-add) as the tested
// logic — this binary is a thin orchestrator, not a reimplementation.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: winze-mem <recall-hook|serve>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "recall-hook":
		runRecallHook()
	case "serve":
		runServe()
	case "capture-guard":
		runCaptureGuard()
	default:
		fmt.Fprintf(os.Stderr, "winze-mem: unknown subcommand %q (want recall-hook|serve|capture-guard)\n", os.Args[1])
		os.Exit(2)
	}
}
