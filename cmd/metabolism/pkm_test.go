package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justinstimatze/winze/internal/vault"
)

// gameVault is the shape this ingest exists for: a game-design reference pile
// whose entire structure is wikilinks, with frontmatter types on some notes and
// none on others.
var gameVault = map[string]string{
	"systems/combat-resolution.md": `---
type: system
tags: [core-loop]
---
# Combat Resolution

The core loop resolves attacks against [[Armor Class]] using a d20 roll.

## Depends on
- [[RNG Determinism]]

## Notes
Balance target is 3-5 rounds. See ` + "`[[not-a-link]]`" + ` for the inline-code case.
`,
	"systems/rng-determinism.md": `---
type: system
aliases: [Deterministic RNG, Seeded RNG]
---
# RNG Determinism

All randomness routes through a seeded PCG source so replays reproduce exactly.

` + "```go" + `
// [[fenced-link]] must not become an edge
seed := rng[[0]]
` + "```" + `

Used by [[Combat Resolution]].
`,
	"entities/goblin-skirmisher.md": `---
type: character
---
# Goblin Skirmisher

Low-tier melee enemy. Uses [[Deterministic RNG]] for dodge rolls.
Lives in [[The Deep Mine]].
`,
	"lore/the-deep-mine.md": `---
type: location
---
# The Deep Mine

The setting's central location. Home to [[Goblin Skirmisher]] packs.
Also see [design notes](../systems/combat-resolution.md).
`,
	"systems/armor-class.md": `# Armor Class

Defensive stat. Referenced by [[Combat Resolution]].
`,
}

func writeVault(t *testing.T, files map[string]string) string {
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

func ingest(t *testing.T, files map[string]string) *vaultResult {
	t.Helper()
	notes, err := vault.Load(writeVault(t, files))
	if err != nil {
		t.Fatal(err)
	}
	return extractVault(notes, vault.NewIndex(notes), map[string]string{}, "2026-07-31")
}

// The defect this rewrite exists to fix: wikilinks were parsed and discarded,
// so a fully interlinked vault produced zero claims. A vault's links ARE its
// graph.
func TestWikilinksBecomeClaims(t *testing.T) {
	res := ingest(t, gameVault)
	if len(res.claims) == 0 {
		t.Fatal("no claims generated from a vault whose entire structure is wikilinks")
	}
	var refs int
	for _, c := range res.claims {
		if c.predicate == "References" {
			refs++
		}
	}
	if refs < 6 {
		t.Errorf("expected at least 6 References claims, got %d (%s)", refs, vaultClaimSummary(res))
	}

	// An alias must resolve to the note that declares it.
	if !hasClaim(res, "GoblinSkirmisher.Entity", "RngDeterminism.Entity") {
		t.Errorf("[[Deterministic RNG]] did not resolve via alias; claims: %s", vaultClaimSummary(res))
	}
	// A relative markdown link is a link too.
	if !hasClaim(res, "TheDeepMine.Entity", "CombatResolution.Entity") {
		t.Errorf("relative markdown link did not become a claim; claims: %s", vaultClaimSummary(res))
	}
}

// A `[[link]]` inside a code fence or an inline code span is not an assertion.
// Treating it as one puts fabricated structure into the corpus.
func TestCodeIsNotAGraphEdge(t *testing.T) {
	res := ingest(t, gameVault)
	for _, u := range res.unresolved {
		if strings.Contains(u, "fenced-link") || strings.Contains(u, "not-a-link") {
			t.Errorf("code-block content treated as a link: %s", u)
		}
	}
	for _, c := range res.claims {
		if strings.Contains(c.object, "Fenced") || strings.Contains(c.object, "NotALink") {
			t.Errorf("code-block content became a claim: %+v", c)
		}
	}
}

// Frontmatter `type` is where a note says what it IS — and it is exactly what
// an OKF bundle carries, so an OKF bundle arrives already typed.
func TestFrontmatterTypesTheEntity(t *testing.T) {
	res := ingest(t, gameVault)
	want := map[string]string{
		"GoblinSkirmisher": "Person",  // character
		"TheDeepMine":      "Place",   // location
		"CombatResolution": "Concept", // system
		"ArmorClass":       "Concept", // no frontmatter at all
	}
	for varName, role := range want {
		e := entityByVar(res, varName)
		if e == nil {
			t.Errorf("entity %s not generated (%s)", varName, entitySummary(res))
			continue
		}
		if e.roleType != role {
			t.Errorf("%s: role %s, want %s", varName, e.roleType, role)
		}
	}
}

// The old ingest materialised Book Notes / Productivity Methods / Coffee
// Brewing categories from a hardcoded map, in every vault, whether or not those
// directories existed.
func TestCategoriesComeFromTheVaultNotAHardcodedList(t *testing.T) {
	res := ingest(t, gameVault)
	var cats []string
	for _, e := range res.entities {
		if strings.HasPrefix(e.id, "vault-category-") {
			cats = append(cats, e.id)
		}
	}
	for _, phantom := range []string{"vault-category-books", "vault-category-productivity", "vault-category-coffee"} {
		for _, c := range cats {
			if c == phantom {
				t.Errorf("phantom category %s materialised from a vault with no such directory", phantom)
			}
		}
	}
	if len(cats) != 3 { // systems, entities, lore
		t.Errorf("expected one category per real directory (3), got %d: %v", len(cats), cats)
	}

	// A flat vault has no directories and must produce no categories.
	flat := ingest(t, map[string]string{"a.md": "# A\n\nlinks [[B]]\n", "b.md": "# B\n"})
	for _, e := range flat.entities {
		if strings.HasPrefix(e.id, "vault-category-") {
			t.Errorf("flat vault produced a category entity: %s", e.id)
		}
	}
}

// A link naming no note is a real signal about the vault — usually a note that
// has not been written. It must be reported, never silently dropped.
func TestUnresolvedLinksAreReported(t *testing.T) {
	res := ingest(t, map[string]string{
		"a.md": "# A\n\nRefers to [[Nonexistent Thing]].\n",
	})
	if len(res.unresolved) != 1 {
		t.Fatalf("expected 1 unresolved link, got %v", res.unresolved)
	}
	if !strings.Contains(res.unresolved[0], "Nonexistent Thing") {
		t.Errorf("unresolved report does not name the target: %s", res.unresolved[0])
	}
}

// Two notes claiming one name is a different problem from a missing note, and
// picking a winner would put a wrong edge in the graph.
func TestAmbiguousLinksResolveToNeither(t *testing.T) {
	res := ingest(t, map[string]string{
		"a/combat.md": "# Combat\n",
		"b/combat.md": "# Combat\n",
		"index.md":    "# Index\n\nSee [[Combat]].\n",
	})
	if len(res.unresolved) != 1 || !strings.Contains(res.unresolved[0], "ambiguous") {
		t.Fatalf("expected an ambiguity report, got %v", res.unresolved)
	}
	for _, c := range res.claims {
		if c.predicate == "References" {
			t.Errorf("ambiguous link produced an edge anyway: %+v", c)
		}
	}
}

// Two notes may legitimately share a title in different directories; emitting
// one var name twice would not compile.
func TestVarNamesAreUnique(t *testing.T) {
	res := ingest(t, map[string]string{
		"a/combat.md": "# Combat\n",
		"b/combat.md": "# Combat\n",
	})
	seen := map[string]bool{}
	for _, e := range res.entities {
		if seen[e.varName] {
			t.Errorf("duplicate var name %s", e.varName)
		}
		seen[e.varName] = true
	}
}

// Regenerating an unchanged vault must produce identical output, or every
// ingest is a diff of nothing but ordering.
func TestExtractionIsDeterministic(t *testing.T) {
	dir := writeVault(t, gameVault)
	var prev string
	for i := 0; i < 3; i++ {
		notes, err := vault.Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		res := extractVault(notes, vault.NewIndex(notes), map[string]string{}, "2026-07-31")
		cur := vaultClaimSummary(res) + "|" + entitySummary(res)
		if i > 0 && cur != prev {
			t.Fatalf("run %d differs:\n%s\n%s", i, prev, cur)
		}
		prev = cur
	}
}

// A note already in the KB must wire new links to the existing entity rather
// than coining a duplicate.
func TestExistingEntitiesAreReused(t *testing.T) {
	notes, err := vault.Load(writeVault(t, gameVault))
	if err != nil {
		t.Fatal(err)
	}
	existing := map[string]string{"MyCombatEntity": "combat-resolution"}
	res := extractVault(notes, vault.NewIndex(notes), existing, "2026-07-31")

	if entityByVar(res, "CombatResolution") != nil {
		t.Error("re-coined an entity that already exists in the KB")
	}
	if res.skipped != 1 {
		t.Errorf("skipped = %d, want 1", res.skipped)
	}
	var wired bool
	for _, c := range res.claims {
		if strings.HasPrefix(c.subject, "MyCombatEntity") || strings.HasPrefix(c.object, "MyCombatEntity") {
			wired = true
		}
	}
	if !wired {
		t.Errorf("links were not wired to the existing entity; claims: %s", vaultClaimSummary(res))
	}
}

// Sections are recorded as promotion candidates but must never become a typed
// predicate mechanically — a link commits to the reference, not to its kind.
func TestSectionsAreRecordedNotInvented(t *testing.T) {
	res := ingest(t, gameVault)
	if res.sections["Depends on"] == 0 {
		t.Errorf("section context not recorded: %v", res.sections)
	}
	for _, c := range res.claims {
		if c.predicate == "DependsOn" {
			t.Errorf("invented a typed predicate from a section heading: %+v", c)
		}
	}
}

func TestClampGoStringKeepsValidUTF8(t *testing.T) {
	s := clampGoString(strings.Repeat("é", 100), 20)
	for _, r := range s {
		if r == '�' {
			t.Fatalf("clamp produced invalid UTF-8: %q", s)
		}
	}
	if len([]rune(s)) == 0 {
		t.Error("clamp produced an empty string")
	}
	if got := clampGoString("short", 100); got != "short" {
		t.Errorf("clamp altered a short string: %q", got)
	}
}

// --- helpers ---

func entityByVar(res *vaultResult, v string) *pkmEntity {
	for i := range res.entities {
		if res.entities[i].varName == v {
			return &res.entities[i]
		}
	}
	return nil
}

func hasClaim(res *vaultResult, subject, object string) bool {
	for _, c := range res.claims {
		if c.subject == subject && c.object == object {
			return true
		}
	}
	return false
}

func vaultClaimSummary(res *vaultResult) string {
	var parts []string
	for _, c := range res.claims {
		parts = append(parts, c.subject+"-"+c.predicate+"->"+c.object)
	}
	return strings.Join(parts, " ")
}

func entitySummary(res *vaultResult) string {
	var parts []string
	for _, e := range res.entities {
		parts = append(parts, e.varName+":"+e.roleType)
	}
	return strings.Join(parts, " ")
}
