package main

// Semantic recall over the project's own prose docs (docs/*.md), for the
// per-prompt hook that replaces a giant always-loaded CLAUDE.md. The core
// stays resident; everything else lives in docs/ and is surfaced only when a
// prompt implicates it. Same embedder, cache, and cosine as entity --semantic
// (semantic.go) — a doc chunk is just another string to embed, so the vector
// cache is shared and content-addressed: an unchanged section never re-embeds.
//
// Chunk granularity is the H2 section. A markdown file's H1 titles it; each ##
// section under that H1 is one recall unit, prefixed with the H1 for context
// (a section named "Batch mode" means nothing without "Authoring helper"
// above it). Prose before the first H2 — or a whole file with no H2 — is one
// chunk. Recall points at file#anchor so the reader opens the section, not the
// 600-line file.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/justinstimatze/winze/internal/cliutil"
)

// runDocsHook is the UserPromptSubmit-hook entry: read the Claude Code hook
// payload from stdin, pull the prompt, and emit doc pointers for the sections
// it implicates. It NEVER fails the hook — any error path returns with no
// output, so a broken recall (ollama down, malformed payload) can't block a
// prompt. dir is the winze repo root whose docs/ to search.
func runDocsHook(dir string) {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return // no piped payload (interactive) — nothing to do
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(os.Stdin); err != nil {
		return
	}
	var in struct {
		HookEventName string `json:"hook_event_name"`
		Prompt        string `json:"prompt"`
	}
	if err := json.Unmarshal(buf.Bytes(), &in); err != nil {
		return
	}
	// Only fire on a real prompt. SessionStart carries no prompt to match on;
	// the resident CLAUDE.md core already orients a new session.
	if in.HookEventName != "UserPromptSubmit" || strings.TrimSpace(in.Prompt) == "" {
		return
	}
	runDocsRecall(dir, in.Prompt, 0, -1, false)
}

type docChunk struct {
	File    string // repo-relative, e.g. "docs/metabolism.md"
	Anchor  string // GitHub-style slug of the heading
	Heading string // "Authoring helper › Batch mode" (H1 › H2)
	Text    string // heading + body, what gets embedded
}

// chunkDocs walks dir for *.md files and splits each into H2 sections. Files
// listed in skip (basename) are left out — CLAUDE.md is the resident core, not
// recall material, and would only ever rank first against itself.
func chunkDocs(dir string, skip map[string]bool) ([]docChunk, error) {
	var chunks []docChunk
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Don't descend into vendored / tooling dirs that happen to carry
			// markdown; docs live at the top level and under docs/.
			base := d.Name()
			if base == "node_modules" || base == ".git" || base == ".winze-embed" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") || skip[d.Name()] {
			return nil
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			rel = path
		}
		// Recall corpus is the project's own docs only: top-level *.md (README,
		// CONTRIBUTING, …) and the docs/ tree. Not markdown that happens to live
		// in nested tooling dirs (.claude/commands, Gas Town rig scaffolding),
		// which is instructions-to-other-tools, not documentation of winze.
		if d := filepath.Dir(rel); d != "." && !strings.HasPrefix(rel, "docs/") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil // unreadable file drops out of recall, never fails the walk
		}
		chunks = append(chunks, splitSections(rel, string(data))...)
		return nil
	})
	return chunks, err
}

var headingRe = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*$`)

// splitSections turns one markdown file into H2-granular chunks. The running
// H1 titles each chunk; body up to the next ## (or EOF) is the chunk text.
func splitSections(rel, content string) []docChunk {
	lines := strings.Split(content, "\n")
	var (
		chunks      []docChunk
		h1          string
		curHeading  string
		curAnchor   string
		body        []string
		haveSection bool
	)
	flush := func() {
		text := strings.TrimSpace(strings.Join(body, "\n"))
		body = body[:0]
		if !haveSection && text == "" {
			return
		}
		heading := curHeading
		if h1 != "" && curHeading != "" && curHeading != h1 {
			heading = h1 + " › " + curHeading
		} else if heading == "" {
			heading = h1
		}
		full := strings.TrimSpace(heading + "\n" + text)
		if full == "" {
			return
		}
		chunks = append(chunks, docChunk{
			File: rel, Anchor: curAnchor, Heading: heading, Text: full,
		})
	}
	inFence := false
	for _, line := range lines {
		// A ``` or ~~~ fence toggles code-block state. Lines inside a fence are
		// body, never headings — otherwise a bash comment (`# Inline-source
		// mode`) parses as an H1 and spawns a bogus chunk.
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inFence = !inFence
			body = append(body, line)
			continue
		}
		if inFence {
			body = append(body, line)
			continue
		}
		m := headingRe.FindStringSubmatch(line)
		if m == nil {
			body = append(body, line)
			continue
		}
		level, title := len(m[1]), m[2]
		switch {
		case level == 1 && h1 == "":
			// First H1 titles the file; its own preamble becomes a chunk only
			// if an H2 never follows (handled by the EOF flush).
			flush()
			h1 = stripInlineMarkdown(title)
			curHeading, curAnchor = h1, slugAnchor(title)
			haveSection = false
		case level <= 2:
			// A new top-or-section heading starts a new chunk.
			flush()
			curHeading = stripInlineMarkdown(title)
			curAnchor = slugAnchor(title)
			haveSection = true
		default:
			// H3+ stays inside the current H2 chunk as body.
			body = append(body, line)
		}
	}
	flush()
	return chunks
}

var (
	anchorStrip = regexp.MustCompile(`[^\w\- ]+`)
	inlineMd    = regexp.MustCompile("[`*_]+")
)

// slugAnchor mimics GitHub's heading-anchor rules closely enough that
// file#anchor links resolve: lowercase, strip punctuation, spaces to hyphens.
func slugAnchor(h string) string {
	h = inlineMd.ReplaceAllString(h, "")
	h = anchorStrip.ReplaceAllString(strings.ToLower(h), "")
	return strings.ReplaceAll(strings.TrimSpace(h), " ", "-")
}

func stripInlineMarkdown(s string) string {
	return strings.TrimSpace(inlineMd.ReplaceAllString(s, ""))
}

// runDocsRecall embeds each doc chunk (cached) and the query, then prints the
// top matches above a cosine floor as file#anchor pointers. Reflexive recall
// is precision-first — a marginal doc shown every prompt trains the reader to
// ignore the banner — so it shares --semantic's floor rationale. On any embed
// error (ollama down) it prints nothing and exits 0: recall degrades, it never
// blocks a prompt. topN<=0 and floor<0 fall back to the recall defaults.
func runDocsRecall(dir, query string, topN int, floor float64, jsonOut bool) {
	if topN <= 0 {
		topN = docsRecallTopN
	}
	if floor < 0 {
		floor = docsRecallFloor
		if v := os.Getenv("WINZE_DOCS_RECALL_MIN"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				floor = f
			}
		}
	}
	chunks, err := chunkDocs(dir, map[string]bool{"CLAUDE.md": true})
	if err != nil || len(chunks) == 0 {
		return
	}

	cache := loadVecCache(dir)
	type cv struct {
		idx int
		vec []float32
	}
	var vecs []cv
	for i, c := range chunks {
		// Embed the prose, not the code. A chunk whose body is a bash block
		// (authoring, query) would otherwise embed 512 chars of shell syntax and
		// never reach the paragraph that explains it. The displayed chunk keeps
		// its code; only the vector is prose-only.
		et := embedTextFor(c.Text)
		if v, ok := cache.m[embedKey(et)]; ok {
			vecs = append(vecs, cv{i, v})
			continue
		}
		v, err := embed(et)
		if err != nil {
			return // embedder unavailable — stay silent, never block the prompt
		}
		cache.m[embedKey(et)] = v
		cache.dirty = true
		vecs = append(vecs, cv{i, v})
	}
	cache.save()

	qv, err := embed(query)
	if err != nil {
		return
	}
	type scored struct {
		idx   int
		score float64
	}
	ranked := make([]scored, 0, len(vecs))
	for _, e := range vecs {
		ranked = append(ranked, scored{e.idx, dot(qv, e.vec)})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

	var top []scored
	for _, r := range ranked {
		if r.score < floor {
			break
		}
		if top = append(top, r); len(top) >= topN {
			break
		}
	}
	if len(top) == 0 {
		return
	}

	if jsonOut {
		out := make([]map[string]any, 0, len(top))
		for _, r := range top {
			c := chunks[r.idx]
			out = append(out, map[string]any{
				"file": c.File, "anchor": c.Anchor, "heading": c.Heading, "score": r.score,
			})
		}
		printJSON(map[string]any{"query": query, "model": embedModel, "count": len(top), "hits": out})
		return
	}

	fmt.Println("winze docs — sections this prompt implicates (read the file#anchor for detail):")
	for _, r := range top {
		c := chunks[r.idx]
		fmt.Printf("  • %s#%s — %s\n", c.File, c.Anchor, c.Heading)
		fmt.Printf("        %s\n", cliutil.Truncate(firstSentence(bodyOf(c.Text), 160), 160))
	}
}

var fenceLineRe = regexp.MustCompile("(?m)^\\s*(```|~~~).*$")

// embedTextFor returns the chunk text with fenced code blocks removed, so the
// vector reflects the prose that explains a command rather than the command's
// syntax. The heading line survives (it's prose and high-signal). If stripping
// leaves nothing (a chunk that is only a code block), fall back to the full
// text so the chunk is still embeddable.
func embedTextFor(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		if fenceLineRe.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		out = append(out, line)
	}
	stripped := strings.TrimSpace(strings.Join(out, "\n"))
	if stripped == "" {
		return text
	}
	return stripped
}

// bodyOf drops the leading heading line from a chunk so the snippet shows prose.
func bodyOf(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return strings.TrimSpace(text[i+1:])
	}
	return text
}

const (
	// docsRecallTopN caps injected pointers. Small: recall nudges, not floods.
	docsRecallTopN = 3
	// docsRecallFloor is the cosine floor for injecting a doc pointer. Measured
	// on the split docs corpus (2026-07-23, all-minilm): across 10 realistic
	// prompts the correct top-1 doc scored 0.33-0.65 (9/10 correct top-1), so
	// 0.30 keeps every real hit — 0.35 would drop the merge and trip-cycle
	// queries. all-minilm compresses scores, so noise lives near the floor too;
	// top-3 covers the case where the right doc is rank 2. Override with
	// WINZE_DOCS_RECALL_MIN. Precision-first: a section shown every prompt
	// trains the reader to ignore the banner.
	docsRecallFloor = 0.30
)
