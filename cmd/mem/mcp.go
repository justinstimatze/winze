package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// runServe hosts the agentic-first MCP interface to winze-memory over stdio:
//
//	winze_remember(note, role?)  — store a note as a typed memory (build-gated,
//	                               auto-committed to the local-only repo)
//	winze_recall(query, limit?)  — hybrid (BM25+semantic) associative recall
//
// Both are thin wrappers over the built winze-add / winze-query binaries — the
// tested logic — so this server never reimplements the corpus machinery.
func runServe() {
	s := server.NewMCPServer("winze-memory", "0.1.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(mcp.NewTool("winze_remember",
		mcp.WithDescription("Store a working-memory note as a typed entity in winze-memory. Runs the build gate and auto-commits to the local-only store. Use for durable facts, decisions, and preferences worth recalling in future sessions — the same bar as a memory file entry."),
		mcp.WithString("note", mcp.Required(), mcp.Description("The fact to remember, as a self-contained sentence or two. Becomes the entity's Brief.")),
		mcp.WithString("title", mcp.Description("A short 2-4 word title for this memory (becomes the entity Name). Recommended — supply a clean title rather than letting it be auto-derived from the note.")),
		mcp.WithString("role", mcp.Description("Role type for the memory entity (default Concept). Use Person for facts about a person, Hypothesis for a claim under test, etc.")),
		mcp.WithBoolean("force", mcp.Description("Store even if a very similar memory already exists. Default false: a near-duplicate is refused so you can update the existing memory instead of accumulating a second one.")),
	), handleRemember)

	s.AddTool(mcp.NewTool("winze_recall",
		mcp.WithDescription("Associative recall from winze-memory: hybrid BM25+semantic search over memory briefs. Returns the most relevant memories as compact headlines (name, var_name, role, score, truncated brief). Call when starting a task or when a past decision/fact might exist. To read one memory in full, call again with a tighter query and brief_chars=0."),
		mcp.WithString("query", mcp.Required(), mcp.Description("What to recall (natural language or keywords).")),
		mcp.WithNumber("limit", mcp.Description("Max memories to return (default 5).")),
		mcp.WithNumber("brief_chars", mcp.Description("Truncate each brief to this many chars to keep results compact (default 240). Set 0 for full briefs — pair with a small limit so the result stays under the tool-result size cap.")),
	), handleRecall)

	s.AddTool(mcp.NewTool("winze_update",
		mcp.WithDescription("Revise an existing memory's Brief (and optionally its title/Name) in place, through the build gate, then auto-commit. Use when a remembered fact changed or should be refined — this is what to do instead of storing a near-duplicate when winze_remember reports one."),
		mcp.WithString("var", mcp.Required(), mcp.Description("The var name of the memory to update (shown in winze_recall results and in the dedup-block message).")),
		mcp.WithString("note", mcp.Required(), mcp.Description("The new Brief content (replaces the old).")),
		mcp.WithString("title", mcp.Description("Optional: also update the display Name.")),
	), handleUpdate)

	s.AddTool(mcp.NewTool("winze_link",
		mcp.WithDescription("Relate two existing memories with a typed claim (RelatesTo, Supersedes, ...), through the build gate, then auto-commit. Records the connection structurally instead of leaving it in prose. The link is winze's OWN assertion (a Conjecture) — no source is invented. Use RelatesTo for a general see-also, Supersedes when one memory replaces a now-stale one."),
		mcp.WithString("from", mcp.Required(), mcp.Description("Subject memory var name (as shown in winze_recall results).")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Object memory var name.")),
		mcp.WithString("rationale", mcp.Required(), mcp.Description("Why the link holds — winze's own reasoning. Recorded on the Conjecture; a self-asserted link carries no source quote.")),
		mcp.WithString("relation", mcp.Description("Predicate type (default RelatesTo). Any predicate in the store schema (see winze-query --schema) works; the build gate validates the slot types.")),
		mcp.WithString("name", mcp.Description("Optional claim var name; auto-derived from the relation and endpoints if omitted. Re-linking the same pair fails the gate, giving free dedup.")),
	), handleLink)

	fmt.Fprintf(os.Stderr, "winze-memory MCP: serving (store: %s)\n", memRoot())
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "winze-memory MCP error: %v\n", err)
		os.Exit(1)
	}
}

// recall result-shaping defaults. Recall exists for interactive latency, so it
// returns compact headlines: capped to a handful of hits, each brief truncated.
// Without these a wide query over kilobyte-scale project memories inlines every
// full brief and overshoots the per-tool-result size cap, forcing the harness to
// spill to disk and the reader to round-trip — the opposite of associative recall.
const (
	recallDefaultLimit      = 5
	recallDefaultBriefChars = 240
)

func handleRecall(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := req.GetArguments()["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return mcp.NewToolResultError("query: required string argument"), nil
	}
	limit := recallDefaultLimit
	if v, ok := req.GetArguments()["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	briefChars := recallDefaultBriefChars
	if v, ok := req.GetArguments()["brief_chars"].(float64); ok && v >= 0 {
		briefChars = int(v)
	}
	res, ok := runQueryJSON("--hybrid", query)
	if !ok {
		return mcp.NewToolResultError("recall failed (is winze-query built and ollama running for --hybrid?)"), nil
	}
	if len(res.Hits) == 0 {
		return mcp.NewToolResultText("no memories matched — nothing recalled."), nil
	}
	hits := res.Hits
	if len(hits) > limit {
		hits = hits[:limit]
	}
	if briefChars > 0 {
		for i := range hits {
			hits[i].Brief = truncate(hits[i].Brief, briefChars)
		}
	}
	out, _ := json.MarshalIndent(struct {
		Matched int        `json:"matched"` // total hits before the limit cap
		Shown   int        `json:"shown"`
		Hits    []queryHit `json:"hits"`
	}{Matched: res.Count, Shown: len(hits), Hits: hits}, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func handleRemember(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	note, ok := req.GetArguments()["note"].(string)
	if !ok || strings.TrimSpace(note) == "" {
		return mcp.NewToolResultError("note: required string argument"), nil
	}
	role := "Concept"
	if r, ok := req.GetArguments()["role"].(string); ok && strings.TrimSpace(r) != "" {
		role = strings.TrimSpace(r)
	}
	title := ""
	if t, ok := req.GetArguments()["title"].(string); ok {
		title = strings.TrimSpace(t)
	}
	force, _ := req.GetArguments()["force"].(bool)

	// Dedup: refuse a clear duplicate, keep a warning for a related one.
	dd := checkDedup(note, force)
	if dd.block != nil {
		return dd.block, nil
	}

	addOut, err := execAdd(note, role, title)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("remember failed at the build gate (not committed):\n%s", addOut)), nil
	}
	if _, cerr := gitCommitMemory(note); cerr != nil {
		return mcp.NewToolResultText(fmt.Sprintf("remembered (gate passed) but NOT committed: %v\n%s", cerr, strings.TrimSpace(addOut))), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("remembered as %s and committed.\n%s%s", role, strings.TrimSpace(addOut), dd.warning)), nil
}

// dedupDecision is checkDedup's verdict on a candidate note: a non-nil block is
// a refusal to return as-is; a non-empty warning is appended to the success
// message when the note stores but resembles an existing memory.
type dedupDecision struct {
	block   *mcp.CallToolResult
	warning string
}

// checkDedup guards an append-only store against silently accumulating
// duplicates (and, worse, contradictions) as the same fact is re-remembered.
// Cosine can't cleanly separate "reworded duplicate" from "distinct but
// related" with this embedder, so it hard-refuses only clear duplicates
// (>= block), advises on the murky middle (>= warn), and lets force override.
func checkDedup(note string, force bool) dedupDecision {
	nearest, score := nearestMemory(note)
	switch {
	case nearest.Name == "":
		return dedupDecision{}
	case score >= dupBlockScore() && !force:
		return dedupDecision{block: mcp.NewToolResultText(fmt.Sprintf(
			"NOT stored — a very similar memory already exists (cosine %.2f):\n  %s [%s] — %s\n\n"+
				"Revise it: winze_update(var=%q, note=…). If this really is a distinct fact, call winze_remember again with force=true.",
			score, nearest.Name, nearest.VarName, truncate(nearest.Brief, 200), nearest.VarName))}
	case score >= dupWarnScore():
		return dedupDecision{warning: fmt.Sprintf(
			"\n\n⚠ a related memory exists (cosine %.2f): %s — check this isn't a near-duplicate.", score, nearest.Name)}
	default:
		return dedupDecision{}
	}
}

func handleUpdate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	varName, ok := req.GetArguments()["var"].(string)
	if !ok || strings.TrimSpace(varName) == "" {
		return mcp.NewToolResultError("var: required string argument"), nil
	}
	note, ok := req.GetArguments()["note"].(string)
	if !ok || strings.TrimSpace(note) == "" {
		return mcp.NewToolResultError("note: required string argument"), nil
	}
	varName = strings.TrimSpace(varName)
	title := ""
	if t, ok := req.GetArguments()["title"].(string); ok {
		title = strings.TrimSpace(t)
	}

	out, err := execSetBrief(varName, note, title)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update failed at the build gate (not committed):\n%s", out)), nil
	}
	if _, cerr := gitCommitMemory("update " + varName); cerr != nil {
		return mcp.NewToolResultText(fmt.Sprintf("updated %s (gate passed) but NOT committed: %v", varName, cerr)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("updated %s and committed.\n%s", varName, strings.TrimSpace(out))), nil
}

// execSetBrief runs winze-edit set-brief to revise a memory's Brief (and
// optionally Name) through the same gate every mutation uses.
func execSetBrief(varName, brief, title string) (string, error) {
	args := []string{"set-brief", "--var", varName, "--brief", brief, "--root", memRoot()}
	if title != "" {
		args = append(args, "--name", title)
	}
	cmd := exec.Command(editBin(), args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	return buf.String(), cmd.Run()
}

func handleLink(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	from, _ := args["from"].(string)
	to, _ := args["to"].(string)
	rationale, _ := args["rationale"].(string)
	from, to, rationale = strings.TrimSpace(from), strings.TrimSpace(to), strings.TrimSpace(rationale)
	if from == "" || to == "" {
		return mcp.NewToolResultError("from and to: required memory var names"), nil
	}
	if rationale == "" {
		return mcp.NewToolResultError("rationale: required (winze's own reasoning for the link)"), nil
	}
	relation := "RelatesTo"
	if r, ok := args["relation"].(string); ok && strings.TrimSpace(r) != "" {
		relation = strings.TrimSpace(r)
	}
	name := ""
	if n, ok := args["name"].(string); ok {
		name = strings.TrimSpace(n)
	}
	if name == "" {
		name = deriveLinkName(relation, from, to)
	}

	out, err := execLink(from, to, relation, rationale, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("link failed at the build gate (not committed) — a name collision here means the link already exists:\n%s", out)), nil
	}
	if _, cerr := gitCommitMemory("link " + from + " " + relation + " " + to); cerr != nil {
		return mcp.NewToolResultText(fmt.Sprintf("linked %s %s %s (gate passed) but NOT committed: %v", from, relation, to, cerr)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("linked: %s %s %s and committed.\n%s", from, relation, to, strings.TrimSpace(out))), nil
}

// execLink runs winze-add --conjecture to relate two memories with a typed
// claim. Conjecture (not Provenance) because the link is winze's own assertion:
// it has no external source, so it carries a Rationale and no Quote.
func execLink(from, to, relation, rationale, name string) (string, error) {
	args := []string{
		"--to", "memory.go", "--root", memRoot(),
		"--name", name, "--predicate", relation,
		"--subject", from, "--object", to,
		"--conjecture", "--rationale", rationale, "--generated-by", "winze-link",
	}
	cmd := exec.Command(addBin(), args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	return buf.String(), cmd.Run()
}

// deriveLinkName builds a deterministic claim var name from the relation and
// endpoints so re-linking the same pair collides on the var name and the build
// gate rejects it — dedup for free. Endpoints are truncated to keep the name
// readable; a rare truncation collision is caught by the gate too.
func deriveLinkName(relation, from, to string) string {
	short := func(s string) string {
		if len(s) > 24 {
			return s[:24]
		}
		return s
	}
	return relation + short(from) + "To" + short(to)
}

// nearestMemory returns the single most semantically-similar existing memory to
// text, with its cosine score. Empty hit + 0 when the store is empty or the
// embedder is unavailable (dedup then simply doesn't fire — fail-open).
func nearestMemory(text string) (queryHit, float64) {
	res, ok := runQueryJSON("--semantic", text)
	if !ok || len(res.Hits) == 0 {
		return queryHit{}, 0
	}
	return res.Hits[0], res.Hits[0].Score
}

// dedup thresholds (cosine). Calibrated on this store: clear reworded
// duplicates score ~0.62-0.71, distinct-but-related facts ~0.25-0.42, with the
// murky middle in between. block hard-refuses; warn advises. Env-overridable
// as the store and embedder evolve.
func dupBlockScore() float64 { return envFloat("WINZE_DEDUP_BLOCK", 0.62) }
func dupWarnScore() float64  { return envFloat("WINZE_DEDUP_WARN", 0.45) }

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// execAdd runs winze-add --entity to append the note as a typed memory. A
// non-empty title becomes the entity's --name (else winze-add auto-derives).
// Returns combined output (winze-add reports the created var + gate result).
func execAdd(note, role, title string) (string, error) {
	args := []string{"--entity", "--role", role, "--brief", note,
		"--to", "memory.go", "--root", memRoot()}
	if title != "" {
		args = append(args, "--name", title)
	}
	cmd := exec.Command(addBin(), args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// gitCommitMemory stages memory.go and commits. The store is local-only (no
// remote), so this is safe and unattended.
func gitCommitMemory(note string) (string, error) {
	subject := oneLine(note)
	if len(subject) > 60 {
		subject = subject[:60] + "…"
	}
	steps := [][]string{
		{"add", "memory.go"},
		{"commit", "-m", "memory: " + subject},
	}
	var out strings.Builder
	for _, args := range steps {
		full := append([]string{"-C", memRoot()}, args...)
		cmd := exec.Command("git", full...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if err := cmd.Run(); err != nil {
			out.WriteString(buf.String())
			return out.String(), fmt.Errorf("git %s: %w", args[0], err)
		}
		out.WriteString(buf.String())
	}
	return out.String(), nil
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
