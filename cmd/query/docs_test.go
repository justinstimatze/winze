package main

import "testing"

func TestSplitSections(t *testing.T) {
	md := "# Editing helper\n\nIntro prose under the title.\n\n" +
		"## Rename\n\nRename works on byte offsets.\n\n" +
		"### Detail\n\nNested H3 stays in the Rename chunk.\n\n" +
		"## Merge\n\nMerge folds A into B.\n"
	got := splitSections("docs/editing.md", md)
	if len(got) != 3 {
		t.Fatalf("want 3 chunks (title-preamble, Rename, Merge), got %d: %+v", len(got), got)
	}

	// H1 preamble is its own chunk, titled by the H1.
	if got[0].Anchor != "editing-helper" || got[0].Heading != "Editing helper" {
		t.Errorf("chunk 0 heading/anchor wrong: %+v", got[0])
	}

	// H2 chunks are prefixed with the H1 for standalone context.
	if got[1].Heading != "Editing helper › Rename" || got[1].Anchor != "rename" {
		t.Errorf("chunk 1 heading/anchor wrong: %+v", got[1])
	}
	// H3 body folds into its parent H2 chunk, not a chunk of its own.
	if !contains(got[1].Text, "Nested H3 stays") {
		t.Errorf("H3 body should fold into the Rename chunk: %q", got[1].Text)
	}
	if got[2].Heading != "Editing helper › Merge" {
		t.Errorf("chunk 2 heading wrong: %+v", got[2])
	}
}

func TestSplitSectionsIgnoresFencedHashes(t *testing.T) {
	// A `#` comment inside a code fence must not be parsed as a heading.
	md := "# Authoring helper\n\n```bash\n# Inline-source mode (one-off):\ngo run ./cmd/add --name X\n```\n\nProse after the block.\n"
	got := splitSections("docs/authoring.md", md)
	if len(got) != 1 {
		t.Fatalf("fenced # comment spawned a bogus chunk: got %d chunks: %+v", len(got), got)
	}
	if got[0].Anchor != "authoring-helper" {
		t.Errorf("anchor should come from the real H1, got %q", got[0].Anchor)
	}
	if !contains(got[0].Text, "Inline-source mode") {
		t.Errorf("fenced content should stay in the chunk body: %q", got[0].Text)
	}
}

func TestSplitSectionsNoH2(t *testing.T) {
	// A file that is all one H1 section becomes a single chunk.
	got := splitSections("docs/x.md", "# Just A Title\n\nAll the body, no subsections.\n")
	if len(got) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(got))
	}
	if got[0].Anchor != "just-a-title" {
		t.Errorf("anchor wrong: %q", got[0].Anchor)
	}
}

func TestSlugAnchor(t *testing.T) {
	cases := map[string]string{
		"Editing helper":                    "editing-helper",
		"Rot probe":                         "rot-probe",
		"`cmd/add` Batch mode":              "cmdadd-batch-mode",
		"Skeptical ingest (sensor defense)": "skeptical-ingest-sensor-defense",
		"MCP tools available":               "mcp-tools-available",
	}
	for in, want := range cases {
		if got := slugAnchor(in); got != want {
			t.Errorf("slugAnchor(%q) = %q, want %q", in, got, want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
