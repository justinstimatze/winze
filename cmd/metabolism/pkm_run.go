package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/vault"
)

// runVaultIngest is the --pkm driver: parse a markdown vault, resolve its link
// graph, render typed corpus files, and put the result through the same build
// gate as every other write path.
//
// Reporting is deliberately loud about what did NOT convert. Unresolved and
// ambiguous links are the two facts an operator needs before committing an
// ingest — the first is usually a note that has not been written, the second is
// two notes fighting over a name — and a summary that only counts successes
// would hide both.
func runVaultIngest(kbDir, vaultDir string, entityCap *int, dryRun, jsonOut bool) {
	start := time.Now()

	notes, err := vault.Load(vaultDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault-ingest: %v\n", err)
		os.Exit(1)
	}
	if len(notes) == 0 {
		fmt.Fprintf(os.Stderr, "vault-ingest: no .md files found in %s\n", vaultDir)
		os.Exit(1)
	}
	ix := vault.NewIndex(notes)
	parseMS := time.Since(start).Milliseconds()

	existing := collectExistingEntityNames(kbDir)
	res := extractVault(notes, ix, existing, time.Now().Format("2006-01-02"))

	if jsonOut {
		emitVaultJSON(res, vaultDir, parseMS)
		return
	}

	fmt.Printf("[vault-ingest] %s\n", vaultDir)
	fmt.Printf("  %d notes parsed in %dms, %d existing KB entities\n", len(notes), parseMS, len(existing))
	fmt.Printf("  %d links: %d resolved, %d unresolved, %d ambiguous\n",
		res.stats.Links, res.stats.Resolved, res.stats.Unresolved, res.stats.Ambiguous)
	fmt.Printf("  %d entities, %d claims (%d notes already in KB)\n",
		len(res.entities), len(res.claims), res.skipped)
	if res.stats.WithType > 0 {
		fmt.Printf("  %d notes declared a frontmatter type\n", res.stats.WithType)
	}
	if res.stats.Orphans > 0 {
		fmt.Printf("  %d notes have no resolved link in either direction\n", res.stats.Orphans)
	}
	if len(res.sections) > 0 {
		fmt.Printf("  link sections (promotion candidates): %s\n", topSections(res.sections))
	}
	reportUnresolved(res)

	files := groupByFile(res)
	if dryRun {
		fmt.Println("\n  dry run — would generate:")
		for _, name := range sortedFileNames(files) {
			g := files[name]
			fmt.Printf("    %s: %d entities, %d claims\n", name, len(g.entities), len(g.claims))
		}
		return
	}

	total := len(existing) + len(res.entities)
	if entityCap != nil && *entityCap > 0 && total > *entityCap {
		fmt.Fprintf(os.Stderr, "\n[vault-ingest] refusing: ingest would bring entity count to %d (cap %d)\n", total, *entityCap)
		fmt.Fprintf(os.Stderr, "  raise it with --entity-cap, or narrow the vault directory\n")
		os.Exit(1)
	}

	var generated []string
	for _, name := range sortedFileNames(files) {
		p, err := writeVaultFile(kbDir, name, files[name], res.provs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vault-ingest: %v\n", err)
			os.Exit(1)
		}
		if p != "" {
			generated = append(generated, p)
		}
	}
	if len(generated) == 0 {
		fmt.Println("\n  nothing to generate — every note is already in the KB")
		return
	}

	fmt.Printf("\n  validating %d generated file(s) through the build gate...\n", len(generated))
	if out, err := runIn(kbDir, "go", "build", "./..."); err != nil {
		for _, f := range generated {
			os.Remove(f)
		}
		fmt.Fprintf(os.Stderr, "\n[vault-ingest] go build FAILED — generated files reverted\n%s\n", out)
		os.Exit(1)
	}
	if out, err := runIn(kbDir, "go", "vet", "./..."); err != nil {
		fmt.Fprintf(os.Stderr, "[vault-ingest] go vet warning: %s\n", out)
	}

	fmt.Printf("  OK: %d entities, %d claims across %d file(s) in %dms\n",
		len(res.entities), len(res.claims), len(generated), time.Since(start).Milliseconds())
	for _, f := range generated {
		fmt.Printf("    %s\n", filepath.Base(f))
	}
}

// reportUnresolved prints the link failures, capped so a vault with thousands
// of broken links does not bury the summary. The count is always exact even
// when the listing is truncated — a silent cap would read as "nothing else
// went wrong".
func reportUnresolved(res *vaultResult) {
	if len(res.unresolved) == 0 {
		return
	}
	const cap = 15
	fmt.Printf("\n  %d link(s) did not resolve:\n", len(res.unresolved))
	for i, u := range res.unresolved {
		if i == cap {
			fmt.Printf("    ... and %d more\n", len(res.unresolved)-cap)
			break
		}
		fmt.Printf("    %s\n", u)
	}
}

func topSections(sections map[string]int) string {
	type kv struct {
		k string
		n int
	}
	var list []kv
	for k, n := range sections {
		list = append(list, kv{k, n})
	}
	sort.Slice(list, func(a, b int) bool {
		if list[a].n != list[b].n {
			return list[a].n > list[b].n
		}
		return list[a].k < list[b].k
	})
	var parts []string
	for i := 0; i < len(list) && i < 5; i++ {
		parts = append(parts, fmt.Sprintf("%s×%d", list[i].k, list[i].n))
	}
	return strings.Join(parts, ", ")
}

// fileGroup is one generated corpus file's contents.
type fileGroup struct {
	entities []pkmEntity
	claims   []pkmClaim
}

// groupByFile buckets output by the vault directory a note came from, so a
// vault's own organisation survives into the corpus and a re-ingest of one
// directory touches one file.
func groupByFile(res *vaultResult) map[string]*fileGroup {
	files := map[string]*fileGroup{}
	get := func(sourceNote string) *fileGroup {
		name := "vault_" + sanitizeDir(noteDir(sourceNote)) + ".go"
		if files[name] == nil {
			files[name] = &fileGroup{}
		}
		return files[name]
	}
	for _, e := range res.entities {
		g := get(e.sourceNote)
		g.entities = append(g.entities, e)
	}
	for _, c := range res.claims {
		g := get(c.sourceNote)
		g.claims = append(g.claims, c)
	}
	return files
}

func sortedFileNames(files map[string]*fileGroup) []string {
	out := make([]string, 0, len(files))
	for k := range files {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// writeVaultFile renders one corpus file. Output is deterministic — sorted,
// and carrying no generation timestamp — so re-ingesting an unchanged vault
// leaves a clean tree instead of a diff of nothing but clocks.
func writeVaultFile(kbDir, name string, g *fileGroup, provs map[string]pkmProvenance) (string, error) {
	if len(g.entities) == 0 && len(g.claims) == 0 {
		return "", nil
	}
	sort.Slice(g.entities, func(i, j int) bool { return g.entities[i].varName < g.entities[j].varName })
	sort.Slice(g.claims, func(i, j int) bool { return g.claims[i].varName < g.claims[j].varName })

	needed := map[string]bool{}
	for _, c := range g.claims {
		needed[c.provVar] = true
	}
	provNames := make([]string, 0, len(needed))
	for p := range needed {
		provNames = append(provNames, p)
	}
	sort.Strings(provNames)

	var b strings.Builder
	b.WriteString("package winze\n\n")
	b.WriteString("// Vault ingest — generated by `winze-metabolism --pkm <vault> .`\n")
	b.WriteString("//\n")
	b.WriteString("// Regenerating an unchanged vault produces an identical file. Entities are\n")
	b.WriteString("// typed from each note's frontmatter where it declares one; References claims\n")
	b.WriteString("// record the links the notes actually make, and nothing more. Promoting a\n")
	b.WriteString("// References edge to a typed predicate is a normal corpus edit.\n\n")

	for _, p := range provNames {
		pv := provs[p]
		b.WriteString(fmt.Sprintf("var %s = Provenance{\n", p))
		b.WriteString(fmt.Sprintf("\tOrigin:     %q,\n", pv.origin))
		b.WriteString(fmt.Sprintf("\tIngestedAt: %q,\n", ingestDate))
		b.WriteString("\tIngestedBy: \"winze vault-ingest\",\n")
		b.WriteString(fmt.Sprintf("\tQuote:      %q,\n", clampGoString(pv.quote, 300)))
		b.WriteString("}\n\n")
	}
	for _, e := range g.entities {
		b.WriteString(fmt.Sprintf("var %s = %s{&Entity{\n", e.varName, e.roleType))
		b.WriteString(fmt.Sprintf("\tID:    %q,\n", e.id))
		b.WriteString(fmt.Sprintf("\tName:  %q,\n", e.name))
		b.WriteString(fmt.Sprintf("\tKind:  %q,\n", e.kind))
		b.WriteString(fmt.Sprintf("\tBrief: %q,\n", clampGoString(e.brief, 280)))
		b.WriteString("}}\n\n")
	}
	for _, c := range g.claims {
		b.WriteString(fmt.Sprintf("var %s = %s{\n", c.varName, c.predicate))
		b.WriteString(fmt.Sprintf("\tSubject: %s,\n", c.subject))
		b.WriteString(fmt.Sprintf("\tObject:  %s,\n", c.object))
		b.WriteString(fmt.Sprintf("\tProv:    %s,\n", c.provVar))
		b.WriteString("}\n\n")
	}

	outPath := filepath.Join(kbDir, name)
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	if out, err := runIn(kbDir, "gofmt", "-w", name); err != nil {
		return "", fmt.Errorf("gofmt %s: %s", name, out)
	}
	return outPath, nil
}

// ingestDate is set once per process so every provenance var in one run carries
// the same date — a run that straddles midnight must not emit two.
var ingestDate = time.Now().Format("2006-01-02")

func clampGoString(s string, max int) string {
	s = cleanGoString(s)
	if len(s) <= max {
		return s
	}
	cut := max - 1
	for cut > 0 && !isASCIIBoundary(s, cut) {
		cut--
	}
	return strings.TrimSpace(s[:cut]) + "…"
}

// isASCIIBoundary reports whether cutting at i leaves valid UTF-8 — a naive
// byte slice through a multi-byte rune produces a string literal Go will
// compile but that renders as a replacement character.
func isASCIIBoundary(s string, i int) bool {
	return i <= 0 || i >= len(s) || s[i]&0xC0 != 0x80
}

func runIn(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func emitVaultJSON(res *vaultResult, vaultDir string, parseMS int64) {
	payload := map[string]any{
		"vault":       vaultDir,
		"notes":       res.stats.Notes,
		"parse_ms":    parseMS,
		"links":       res.stats.Links,
		"resolved":    res.stats.Resolved,
		"unresolved":  res.stats.Unresolved,
		"ambiguous":   res.stats.Ambiguous,
		"orphans":     res.stats.Orphans,
		"typed_notes": res.stats.WithType,
		"entities":    len(res.entities),
		"claims":      len(res.claims),
		"skipped":     res.skipped,
		"problems":    res.unresolved,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}
