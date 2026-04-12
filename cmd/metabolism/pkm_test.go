package main

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"  spaces  ", "spaces"},
		{"Special!@#chars", "special-chars"},
		{"already-slug", "already-slug"},
		{"123 Numbers", "123-numbers"},
		{"", ""},
		{"UPPER CASE", "upper-case"},
		{"multi---hyphens", "multi-hyphens"},
		{"trailing-", "trailing"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := slugify(tc.input)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGoIdentifier(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello world", "HelloWorld"},
		{"some-thing", "SomeThing"},
		{"UPPER", "Upper"},
		{"123start", "N123start"},
		{"", ""},
		{"a", "A"},
		{"multi   space", "MultiSpace"},
		{"with.dots.here", "WithDotsHere"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := goIdentifier(tc.input)
			if got != tc.want {
				t.Errorf("goIdentifier(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestTruncateBrief(t *testing.T) {
	cases := []struct {
		name    string
		title   string
		content string
		wantSub string // substring that should appear in result
	}{
		{
			"simple paragraph",
			"Test Note",
			"This is the first paragraph.\n\nSecond paragraph here.",
			"This is the first paragraph.",
		},
		{
			"strips markdown bold",
			"Note",
			"This is **bold** text.",
			"This is bold text.",
		},
		{
			"strips wikilinks",
			"Note",
			"See [[SomeConcept]] for details.",
			"See SomeConcept for details.",
		},
		{
			"skips headers",
			"Note",
			"# Header\n\nActual content here.",
			"Actual content here.",
		},
		{
			"returns title if no content",
			"Fallback Title",
			"# Header\n---\n**Author**: Someone",
			"Fallback Title",
		},
		{
			"truncates long content",
			"Note",
			strings.Repeat("A very long sentence that goes on and on. ", 20),
			"...",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateBrief(tc.title, tc.content)
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("truncateBrief(%q, ...) = %q, want to contain %q", tc.title, got, tc.wantSub)
			}
			if len(got) > 250 {
				t.Errorf("truncateBrief returned %d chars, max 250", len(got))
			}
		})
	}
}

func TestExtractLeadSentence(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantSub string
	}{
		{
			"simple content",
			"First line of text.\nSecond line.",
			"First line of text.",
		},
		{
			"skips headers and metadata",
			"# Title\n**Author**: Someone\n---\nActual content.",
			"Actual content.",
		},
		{
			"empty content",
			"",
			"",
		},
		{
			"all metadata",
			"# Title\n**Bold**\n---\n| table |",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractLeadSentence(tc.content)
			if tc.wantSub == "" {
				if got != "" {
					t.Errorf("extractLeadSentence(...) = %q, want empty", got)
				}
			} else if !strings.Contains(got, tc.wantSub) {
				t.Errorf("extractLeadSentence(...) = %q, want to contain %q", got, tc.wantSub)
			}
		})
	}
}

func TestCleanGoString(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello\nworld", "hello world"},
		{"tabs\there", "tabs here"},
		{"double  spaces", "double spaces"},
		{"  trimmed  ", "trimmed"},
		{"normal text", "normal text"},
		{"multi\n\nnewlines", "multi newlines"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := cleanGoString(tc.input)
			if got != tc.want {
				t.Errorf("cleanGoString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExtractPKMContent(t *testing.T) {
	notes := []pkmNote{
		{
			relPath: "consciousness.md",
			title:   "Consciousness",
			content: "Consciousness is the quality of awareness.\n\n**Prediction**: consciousness will be explained by IIT.",
			wikilinks: []string{"IIT", "Qualia"},
		},
		{
			relPath: "qualia.md",
			title:   "Qualia",
			content: "Qualia are individual instances of subjective experience.",
		},
	}

	existing := map[string]string{} // no existing entities

	entities, claims, provs := extractPKMContent(notes, existing)

	// Should create entities for each note
	if len(entities) < 2 {
		t.Fatalf("expected at least 2 entities, got %d", len(entities))
	}

	// Check entity structure
	found := false
	for _, e := range entities {
		if strings.Contains(strings.ToLower(e.name), "consciousness") {
			found = true
			if e.varName == "" {
				t.Error("consciousness entity has empty varName")
			}
			if e.id == "" {
				t.Error("consciousness entity has empty id")
			}
			if e.roleType == "" {
				t.Error("consciousness entity has empty roleType")
			}
		}
	}
	if !found {
		t.Error("consciousness entity not found in output")
	}

	// Log what was produced. Claims and provenance depend on wikilink
	// resolution against existing entities; with empty existing map,
	// wikilinks create new entities but may not generate claims.
	t.Logf("extractPKMContent: %d entities, %d claims, %d provenance records",
		len(entities), len(claims), len(provs))
}
