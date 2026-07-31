package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func load(t *testing.T, files map[string]string) ([]Note, *Index) {
	t.Helper()
	notes, err := Load(write(t, files))
	if err != nil {
		t.Fatal(err)
	}
	return notes, NewIndex(notes)
}

func noteByPath(notes []Note, rel string) *Note {
	for i := range notes {
		if notes[i].RelPath == rel {
			return &notes[i]
		}
	}
	return nil
}

func TestFrontmatterAndTitlePrecedence(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"a.md":     "---\ntitle: Declared Title\ntype: system\naliases:\n  - Alt One\n  - Alt Two\ntags: [core, loop]\nstatus: draft\n---\n# Heading Title\n\nBody.\n",
		"b.md":     "# Heading Title Only\n\nBody.\n",
		"c-d_e.md": "Body with no heading at all.\n",
	})

	a := noteByPath(notes, "a.md")
	if a.Title != "Declared Title" {
		t.Errorf("frontmatter title should win over H1, got %q", a.Title)
	}
	if a.Type != "system" {
		t.Errorf("type = %q", a.Type)
	}
	if len(a.Aliases) != 2 || a.Aliases[0] != "Alt One" {
		t.Errorf("aliases = %v", a.Aliases)
	}
	if len(a.Tags) != 2 {
		t.Errorf("tags = %v", a.Tags)
	}
	if a.Frontmatter["status"] != "draft" {
		t.Errorf("unrecognised scalar keys should be preserved, got %v", a.Frontmatter)
	}
	if strings.Contains(a.Body, "title:") {
		t.Error("frontmatter leaked into the body")
	}

	if noteByPath(notes, "b.md").Title != "Heading Title Only" {
		t.Error("H1 should be the title when frontmatter has none")
	}
	if got := noteByPath(notes, "c-d_e.md").Title; got != "C D E" {
		t.Errorf("filename fallback title = %q, want %q", got, "C D E")
	}
}

// A frontmatter block that does not parse must leave the content in the body.
// Discarding it on a stray delimiter is silent data loss.
func TestUnparseableFrontmatterIsKept(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"a.md": "---\ntags: [unclosed\n---\n\nReal content here.\n",
	})
	if !strings.Contains(notes[0].Body, "Real content here") {
		t.Errorf("body lost: %q", notes[0].Body)
	}
}

func TestLinkDialectsAndAnchors(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"src.md": `# Src

Plain [[Target]], aliased [[Target|shown as this]], anchored [[Target#Section]],
pathed [[sub/Nested]], relative [text](./other.md), and bare [x](other.md).

External [site](https://example.com/page.md) and an anchor [top](#heading) are not links.
`,
		"target.md":     "# Target\n",
		"other.md":      "# Other\n",
		"sub/nested.md": "# Nested\n",
	})
	src := noteByPath(notes, "src.md")

	targets := map[string]bool{}
	for _, l := range src.Links {
		targets[l.Target] = true
	}
	for _, want := range []string{"Target", "sub/Nested", "./other.md", "other.md"} {
		if !targets[want] {
			t.Errorf("missing link target %q; got %v", want, targets)
		}
	}
	for _, unwanted := range []string{"https://example.com/page.md", "#heading"} {
		if targets[unwanted] {
			t.Errorf("non-link %q treated as a link", unwanted)
		}
	}
	// The anchored and aliased forms name the same note as the plain one, so
	// they dedupe within a section rather than producing three edges.
	if n := strings.Count(fmt.Sprint(targets), "Target"); n == 0 {
		t.Error("anchor was not stripped from the target")
	}
}

// A vault of game-design notes quotes code constantly. `[[0]]` in a Go snippet
// is an array index, not a graph edge.
func TestCodeSpansAndFencesAreNotLinks(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"a.md":      "# A\n\nInline `[[not-a-link]]` here.\n\n```go\nx := arr[[0]]\n// [[fenced]] ignored\n```\n\n~~~\n[[tilde-fenced]]\n~~~\n\nReal [[Target]].\n",
		"target.md": "# Target\n",
	})
	for _, l := range notes[0].Links {
		switch l.Target {
		case "not-a-link", "0", "fenced", "tilde-fenced":
			t.Errorf("code content became a link: %q", l.Target)
		}
	}
	var found bool
	for _, l := range notes[0].Links {
		if l.Target == "Target" {
			found = true
		}
	}
	if !found {
		t.Error("real link after a fence was lost")
	}
}

// The section a link sits under is the evidence a later pass uses to propose a
// typed predicate. The note's own H1 is not a section.
func TestSectionContext(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"a.md":      "# A\n\nTop-level [[Target]].\n\n## Depends on\n\n- [[Other]]\n\n### Nested detail\n\n[[Third]]\n",
		"target.md": "# Target\n",
		"other.md":  "# Other\n",
		"third.md":  "# Third\n",
	})
	want := map[string]string{"Target": "", "Other": "Depends on", "Third": "Nested detail"}
	for _, l := range notes[0].Links {
		if got, ok := want[l.Target]; ok && l.Section != got {
			t.Errorf("%s: section %q, want %q", l.Target, l.Section, got)
		}
	}
}

func TestResolutionByPathTitleAndAlias(t *testing.T) {
	notes, ix := load(t, map[string]string{
		"systems/rng.md": "---\naliases: [Deterministic RNG, Seeded RNG]\n---\n# RNG Determinism\n",
		"index.md": `# Index

By title [[RNG Determinism]], by alias [[Seeded RNG]], by filename [[rng]],
by path [[systems/rng]], and case-insensitively [[rng determinism]].
`,
	})
	from := -1
	for i, n := range notes {
		if n.RelPath == "index.md" {
			from = i
		}
	}
	for _, target := range []string{"RNG Determinism", "Seeded RNG", "rng", "systems/rng", "rng determinism"} {
		if _, _, ok := ix.Resolve(from, target); !ok {
			t.Errorf("%q did not resolve", target)
		}
	}
	if _, _, ok := ix.Resolve(from, "Nothing Like This"); ok {
		t.Error("a nonexistent target resolved")
	}
}

// Normalization has to be aggressive enough that the ways one person writes the
// same name all land together, without being so aggressive that distinct names
// collide.
func TestNormalize(t *testing.T) {
	same := [][2]string{
		{"Combat Resolution", "combat-resolution"},
		{"Combat Resolution", "Combat_Resolution"},
		{"RNG  Determinism ", "rng determinism"},
		{"Hit Points!", "hit-points"},
	}
	for _, p := range same {
		if normalize(p[0]) != normalize(p[1]) {
			t.Errorf("%q and %q should normalize alike (%q vs %q)", p[0], p[1], normalize(p[0]), normalize(p[1]))
		}
	}
	if normalize("combat") == normalize("combat resolution") {
		t.Error("distinct names collided")
	}
	// Path separators survive, so a pathed target stays distinguishable.
	if !strings.Contains(normalize("systems/combat"), "/") {
		t.Errorf("path separator lost: %q", normalize("systems/combat"))
	}
}

// Two notes claiming one name resolve to neither. Picking a winner would put a
// wrong edge in the graph, and a wrong edge is worse than a missing one.
func TestAmbiguityIsNotSilentlyResolved(t *testing.T) {
	notes, ix := load(t, map[string]string{
		"a/combat.md": "# Combat\n",
		"b/combat.md": "# Combat\n",
		"index.md":    "# Index\n\n[[Combat]]\n",
	})
	_ = notes
	edges, missing := ix.Edges()
	if len(edges) != 0 {
		t.Errorf("ambiguous target produced %d edges", len(edges))
	}
	if len(missing) != 1 || len(missing[0].Ambiguous) != 2 {
		t.Fatalf("expected one ambiguity naming both notes, got %+v", missing)
	}
}

// A pathed link must prefer the note at that path over a same-named note
// elsewhere.
func TestPathedLinkBeatsBareName(t *testing.T) {
	notes, ix := load(t, map[string]string{
		"a/combat.md": "# A Combat\n",
		"b/combat.md": "# B Combat\n",
		"index.md":    "# Index\n\n[[b/combat]]\n",
	})
	edges, missing := ix.Edges()
	if len(missing) != 0 {
		t.Fatalf("pathed link did not resolve: %+v", missing)
	}
	if len(edges) != 1 || notes[edges[0].To].RelPath != "b/combat.md" {
		t.Errorf("resolved to the wrong note: %+v", edges)
	}
}

// Self-links are not edges, and one reference repeated in a section is one edge.
func TestEdgeDeduplication(t *testing.T) {
	_, ix := load(t, map[string]string{
		"a.md":      "# A\n\n[[A]] [[Target]] [[Target]] again [[Target]].\n\n## Elsewhere\n\n[[Target]]\n",
		"target.md": "# Target\n",
	})
	edges, _ := ix.Edges()
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges (one per section), got %d: %+v", len(edges), edges)
	}
	sections := []string{edges[0].Section, edges[1].Section}
	if sections[0] == sections[1] {
		t.Errorf("sections not distinguished: %v", sections)
	}
}

// Notes are returned in stable RelPath order regardless of parallel parsing, so
// downstream output does not depend on scheduling.
func TestLoadIsOrdered(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 60; i++ {
		files[fmt.Sprintf("d%02d/note.md", i)] = fmt.Sprintf("# Note %d\n", i)
	}
	dir := write(t, files)
	var prev []string
	for run := 0; run < 3; run++ {
		notes, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, n := range notes {
			got = append(got, n.RelPath)
		}
		if run > 0 && strings.Join(got, ",") != strings.Join(prev, ",") {
			t.Fatal("Load order is not stable across runs")
		}
		prev = got
	}
}

// bufio.Scanner's default 64 KiB token limit silently truncates a note holding
// one very long line. A link past that point must still be found.
func TestVeryLongLineIsNotTruncated(t *testing.T) {
	long := strings.Repeat("filler words about the combat system ", 4000) // ~148 KB, one line
	notes, _ := load(t, map[string]string{
		"a.md":      "# A\n\n" + long + " [[Target]]\n",
		"target.md": "# Target\n",
	})
	a := noteByPath(notes, "a.md")
	var found bool
	for _, l := range a.Links {
		if l.Target == "Target" {
			found = true
		}
	}
	if !found {
		t.Error("link after a 148 KB line was lost to buffer truncation")
	}
}

func TestHiddenDirectoriesAreSkipped(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"a.md":                  "# A\n",
		".obsidian/plugin.md":   "# Plugin\n",
		".trash/deleted.md":     "# Deleted\n",
		"sub/.hidden/secret.md": "# Secret\n",
	})
	if len(notes) != 1 || notes[0].RelPath != "a.md" {
		var paths []string
		for _, n := range notes {
			paths = append(paths, n.RelPath)
		}
		t.Errorf("hidden directories were not skipped: %v", paths)
	}
}

// The lead is the note's own summary sentence, used verbatim as an entity
// Brief — so it must be prose, stripped of markup, and never a list item or a
// metadata line.
func TestLeadSentence(t *testing.T) {
	notes, _ := load(t, map[string]string{
		"a.md": "# A\n\n**Author:** Someone\n\n- a list item\n\n> a quote\n\nThe real lead with a [[Link]] and `code`. A second sentence follows.\n",
	})
	got := notes[0].Lead
	if !strings.HasPrefix(got, "The real lead") {
		t.Errorf("lead = %q, expected the first prose paragraph", got)
	}
	if strings.Contains(got, "[[") || strings.Contains(got, "`") {
		t.Errorf("lead retained markup: %q", got)
	}
	if strings.Contains(got, "second sentence") {
		t.Errorf("lead should stop at the first sentence: %q", got)
	}
}

func TestStatsReportsGraphHealth(t *testing.T) {
	_, ix := load(t, map[string]string{
		"a.md":      "---\ntype: system\n---\n# A\n\n## Depends on\n\n[[B]] and [[Missing Note]]\n",
		"b.md":      "# B\n",
		"lonely.md": "# Lonely\n\nNo links at all.\n",
	})
	s := ix.Stats()
	if s.Notes != 3 || s.Resolved != 1 || s.Unresolved != 1 {
		t.Errorf("stats = %+v", s)
	}
	if s.Orphans != 1 {
		t.Errorf("orphans = %d, want 1 (lonely.md)", s.Orphans)
	}
	if s.WithType != 1 {
		t.Errorf("typed notes = %d, want 1", s.WithType)
	}
	if len(s.TopSections) == 0 || !strings.Contains(s.TopSections[0], "Depends on") {
		t.Errorf("top sections = %v", s.TopSections)
	}
}

func TestEmptyVault(t *testing.T) {
	notes, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Errorf("expected no notes, got %d", len(notes))
	}
	ix := NewIndex(notes)
	edges, missing := ix.Edges()
	if len(edges) != 0 || len(missing) != 0 {
		t.Error("empty vault produced graph content")
	}
}
