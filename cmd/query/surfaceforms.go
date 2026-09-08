package main

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const surfaceFormSystemPrompt = `You generate short natural-language questions that a person might ask later to retrieve a specific note.

Given a note's name and brief, write 3 to 8 short questions someone would plausibly type to find this note again, worded the way a real question is worded -- not a restatement of the note's own phrasing. Vary the angle: what happened, why, what was decided, what it's called.

Respond with ONLY a JSON array of strings, no other text.`

const surfaceFormsCacheFile = "surfaceforms.v1.gob"

// buildSurfaceForms is the injectable core: cache hits always run and never
// touch the call cap; new generation is gated by maxCalls and by the
// injected generate func, so this is unit-testable with a fake closure and
// no network.
func buildSurfaceForms(kb *kbIndex, cache *surfaceFormCache, maxCalls int,
	generate func(e entityRecord) ([]string, error)) map[int][]string {
	out := make(map[int][]string, len(kb.Entities))
	calls := 0
	for i, e := range kb.Entities {
		text := embedText(e)
		if text == "" || text == "." {
			continue
		}
		key := surfaceFormKey(e.Name, e.Brief)
		if forms, ok := cache.m[key]; ok {
			out[i] = forms
			continue
		}
		if calls >= maxCalls {
			continue
		}
		calls++
		forms, err := generate(e)
		if err != nil {
			continue
		}
		cache.m[key] = forms
		cache.dirty = true
		out[i] = forms
	}
	return out
}

// generateSurfaceForms calls Haiku once per entity. A truncated response is
// an error, not a partial success -- same posture as extractFacts
// (cmd/longmemeval/claimnote.go) and callIngestLLM (cmd/metabolism/ingest.go).
func generateSurfaceForms(client anthropic.Client, name, brief string) ([]string, error) {
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{
			{Text: surfaceFormSystemPrompt, CacheControl: anthropic.CacheControlEphemeralParam{Type: "ephemeral"}},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(name + ". " + brief)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}
	if resp.StopReason == "max_tokens" {
		return nil, fmt.Errorf("surface-form generation truncated at %d output tokens", resp.Usage.OutputTokens)
	}
	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	return parseSurfaceFormsResponse(text)
}

func loadSurfaceFormCache(dir string) *surfaceFormCache {
	c := &surfaceFormCache{path: filepath.Join(dir, embedCacheDir, surfaceFormsCacheFile), m: map[string][]string{}}
	if f, err := os.Open(c.path); err == nil {
		defer f.Close()
		_ = gob.NewDecoder(f).Decode(&c.m)
	}
	return c
}

// parseSurfaceFormsResponse pulls the JSON string array out of a model
// response, tolerating a markdown code fence -- pulled out of
// generateSurfaceForms so parsing is testable without a live call, the same
// reasoning parseFactsResponse (cmd/longmemeval) is kept separate.
func parseSurfaceFormsResponse(text string) ([]string, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	var forms []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &forms); err != nil {
		return nil, fmt.Errorf("unparseable surface-form list: %w\nraw: %s", err, text)
	}
	if len(forms) == 0 {
		return nil, fmt.Errorf("zero surface forms generated")
	}
	return forms, nil
}

func surfaceFormKey(name, brief string) string {
	h := sha256.Sum256([]byte(name + "\x00" + brief))
	return fmt.Sprintf("%x", h[:16])
}

// surfaceFormsEnabled gates the whole feature off by default -- generating
// costs a real, billed Haiku call per new entity, unlike the always-on local
// Ollama embedding path, so it stays opt-in.
func surfaceFormsEnabled() bool {
	return os.Getenv("WINZE_SURFACE_FORMS") != ""
}

// surfaceFormsFor is the thin production wrapper: resolves gating, the API
// key (mirroring runAsk's own lookup), the cache, and the client, then
// delegates to buildSurfaceForms. Returns nil whenever the feature is off or
// unusable -- callers treat nil identically to today's behavior.
func surfaceFormsFor(dir string, kb *kbIndex) map[int][]string {
	if !surfaceFormsEnabled() {
		return nil
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		loadDotEnv(dir)
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil
	}
	cache := loadSurfaceFormCache(dir)
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	forms := buildSurfaceForms(kb, cache, surfaceFormsMaxCalls(), func(e entityRecord) ([]string, error) {
		return generateSurfaceForms(client, e.Name, e.Brief)
	})
	cache.save()
	return forms
}

// surfaceFormsMaxCalls bounds how many NEW entities can trigger generation in
// one run -- cache hits never count against this, so cost amortizes to zero
// after each entity's first index build. Unlike cmd/lint's --llm-max-calls,
// there is no "0 = unlimited" escape hatch: an invalid or non-positive
// override falls back to the default rather than disabling the cap, since
// this cost is billed per call and should never be accidentally uncapped.
func surfaceFormsMaxCalls() int {
	if v := os.Getenv("WINZE_SURFACE_FORMS_MAX_CALLS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 20
}

func (c *surfaceFormCache) save() {
	if !c.dirty {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return
	}
	if f, err := os.Create(c.path); err == nil {
		defer f.Close()
		_ = gob.NewEncoder(f).Encode(c.m)
	}
}

// surfaceFormCache persists LLM-generated question surface forms per entity,
// keyed by content hash of Name+Brief -- mirrors vecCache's shape exactly,
// including its lack of atomic-write protection: an accepted, bounded risk
// for an opt-in diagnostic flag (a lost update here wastes a paid API call,
// not just CPU, but is otherwise self-healing on next regeneration).
type surfaceFormCache struct {
	path  string
	m     map[string][]string
	dirty bool
}
