package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// recallTopN is how many memories a per-prompt associative recall injects.
// Small on purpose: reflexive recall should nudge, not flood the context.
const recallTopN = 3

// briefMax truncates a memory Brief in the injected block.
const briefMax = 200

// hookInput is the subset of the Claude Code hook stdin payload we read.
type hookInput struct {
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
}

type queryHit struct {
	Name    string  `json:"name"`
	VarName string  `json:"var_name"`
	Role    string  `json:"role_type"`
	Brief   string  `json:"brief"`
	Score   float64 `json:"score"` // cosine similarity (--semantic)
}

// recallMinScore is the cosine-similarity floor a memory must clear to be
// injected by the reflexive hook. Semantic (not BM25) because natural prompts
// paraphrase — "how does caching work" must still reach a "cache_control" memory,
// which lexical match misses and misranks. Calibrated on this corpus: genuinely-
// relevant top hits score ~0.44-0.66, unrelated prompts ~0.05-0.16, a wide clean
// gap. 0.35 sits in it. Cosine is corpus-independent, so this floor holds as the
// store grows (unlike a BM25 score). Reflexive injection is precision-first: a
// marginal memory shown every prompt trains the reader to ignore the banner.
// Deliberate winze_recall has no floor. Override with WINZE_RECALL_MIN_SCORE.
const recallMinScore = 0.35

type queryResult struct {
	Count int        `json:"count"`
	Hits  []queryHit `json:"hits"`
}

// runRecallHook reads the hook payload from stdin and prints a context block
// to stdout (which Claude Code injects). It NEVER fails the hook: any error
// path exits 0 with no output, so a broken recall can't block a prompt.
func runRecallHook() {
	var in hookInput
	if data, err := readAllStdin(); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &in)
	}

	switch in.HookEventName {
	case "UserPromptSubmit":
		if strings.TrimSpace(in.Prompt) == "" {
			return
		}
		emitAssociativeRecall(in.Prompt)
	case "SessionStart", "":
		emitDigest()
	default:
		// Unknown event — stay silent rather than guess.
	}
}

// emitAssociativeRecall runs a semantic query on the prompt and injects only
// matches above the cosine floor, so memory surfaces reflexively when genuinely
// relevant and stays silent otherwise. Semantic (not lexical) so paraphrased
// prompts still reach the right memory; the cosine score gives a clean,
// corpus-independent relevance cutoff that BM25's IDF-dependent score lacks.
// If the embedder (ollama) is down the query fails and the hook stays silent —
// recall degrades, it never blocks a prompt.
func emitAssociativeRecall(prompt string) {
	res, ok := runQueryJSON("--semantic", prompt)
	if !ok || len(res.Hits) == 0 {
		return
	}
	floor := recallMinScore
	if v := os.Getenv("WINZE_RECALL_MIN_SCORE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			floor = f
		}
	}
	var hits []queryHit
	for _, h := range res.Hits {
		if h.Score >= floor {
			hits = append(hits, h)
		}
		if len(hits) >= recallTopN {
			break
		}
	}
	if len(hits) == 0 {
		return
	}
	var b strings.Builder
	b.WriteString("winze-memory — associative recall on this prompt (use winze_recall to dig, winze_remember to add):\n")
	for _, h := range hits {
		fmt.Fprintf(&b, "  • %s — %s\n", h.Name, truncate(h.Brief, briefMax))
	}
	fmt.Print(b.String())
}

// emitDigest prints a one-line orientation at session start so the store's
// existence and size are present without loading every brief.
func emitDigest() {
	out, err := runQueryRaw("--stats")
	if err != nil {
		return
	}
	// --stats prints a human summary; surface just its entity count line if
	// present, else a generic pointer.
	line := firstMatch(out, "entit")
	if line == "" {
		line = "winze-memory available"
	}
	fmt.Printf("winze-memory: %s (associative recall fires per-prompt; winze_recall/winze_remember to query/add)\n", strings.TrimSpace(line))
}

// runQueryJSON execs winze-query with --json and decodes the result. stderr is
// discarded (winze-query prints embed-cache chatter there).
func runQueryJSON(mode, arg string) (queryResult, bool) {
	out, err := runQueryRaw(mode, arg, "--json")
	if err != nil {
		return queryResult{}, false
	}
	// --json may be preceded by nothing on stdout; decode directly.
	var res queryResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return queryResult{}, false
	}
	return res, true
}

// runQueryRaw execs winze-query <mode> [arg] [flags...] <memRoot> and returns
// stdout. The memory root is always the final positional arg.
func runQueryRaw(args ...string) (string, error) {
	full := append([]string{}, args...)
	full = append(full, memRoot())
	cmd := exec.Command(queryBin(), full...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil // discard embed-cache chatter
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func readAllStdin() ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	// No piped input (interactive) — don't block on a read.
	if fi.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func firstMatch(s, needle string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(strings.ToLower(line), needle) {
			return line
		}
	}
	return ""
}

func truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
