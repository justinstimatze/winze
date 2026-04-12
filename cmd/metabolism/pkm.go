package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// pkmNote holds parsed content from a single markdown note.
type pkmNote struct {
	relPath     string   // relative path in vault (e.g., "books/deep-work.md")
	dir         string   // subdirectory (e.g., "books")
	title       string   // first H1 heading or derived from filename
	author      string   // from **Author:** line (book notes)
	rating      string   // from **Rating:** line
	wikilinks   []string // all [[target]] references (deduplicated, ordered)
	predictions []string // **Prediction**: lines (full text)
	content     string   // full note text
}

// pkmEntity is a generated entity for code output.
type pkmEntity struct {
	varName  string // Go variable name (PascalCase)
	roleType string // Person, Concept, Hypothesis
	id       string // kebab-case ID
	name     string // display name
	kind     string // person, concept, hypothesis
	brief    string // one-line description
	sourceNote string // which note introduced this entity
}

// pkmClaim is a generated claim for code output.
type pkmClaim struct {
	varName    string // Go variable name
	predicate  string // Authored, BelongsTo, Proposes, etc.
	subject    string // subject entity varName
	object     string // object entity varName
	provVar    string // provenance variable name
	sourceNote string // which note generated this claim
}

// runPKMIngest reads a PKM vault directory and generates typed Go corpus files.
func runPKMIngest(kbDir, vaultDir string, entityCap *int, dryRun, jsonOut bool) {
	// 1. Parse all markdown notes
	notes := parseVaultNotes(vaultDir)
	if len(notes) == 0 {
		fmt.Fprintf(os.Stderr, "pkm-ingest: no .md files found in %s\n", vaultDir)
		os.Exit(1)
	}

	fmt.Printf("[pkm-ingest] parsed %d notes from %s\n", len(notes), vaultDir)

	// 2. Collect existing KB entity names to avoid duplicates
	existing := collectExistingEntityNames(kbDir)
	fmt.Printf("[pkm-ingest] %d existing entities in KB\n", len(existing))

	// 3. Extract entities and claims mechanically
	entities, claims, provs := extractPKMContent(notes, existing)

	fmt.Printf("[pkm-ingest] extracted %d entities, %d claims\n", len(entities), len(claims))

	// Group by source directory
	dirs := map[string]bool{}
	for _, n := range notes {
		dirs[n.dir] = true
	}

	if dryRun {
		fmt.Println("\n[pkm-ingest] dry run — would generate:")
		for dir := range dirs {
			dirEnts := 0
			dirClaims := 0
			for _, e := range entities {
				if noteDir(e.sourceNote) == dir {
					dirEnts++
				}
			}
			for _, c := range claims {
				if noteDir(c.sourceNote) == dir {
					dirClaims++
				}
			}
			fmt.Printf("  pkm_%s.go: %d entities, %d claims\n", sanitizeDir(dir), dirEnts, dirClaims)
		}
		fmt.Printf("\n  Total: %d entities, %d claims across %d files\n", len(entities), len(claims), len(dirs))
		return
	}

	// 4. Check entity cap
	totalEntities := len(existing) + len(entities)
	if entityCap != nil && *entityCap > 0 && totalEntities > *entityCap {
		fmt.Fprintf(os.Stderr, "[pkm-ingest] WARNING: ingest would bring entity count to %d (cap: %d)\n", totalEntities, *entityCap)
		fmt.Fprintf(os.Stderr, "  use --entity-cap to increase the limit\n")
		os.Exit(1)
	}

	// 5. Generate Go files grouped by directory
	var generated []string
	for dir := range dirs {
		outPath := generatePKMFile(kbDir, dir, notes, entities, claims, provs)
		if outPath != "" {
			generated = append(generated, outPath)
		}
	}

	if len(generated) == 0 {
		fmt.Println("[pkm-ingest] no code generated")
		return
	}

	// 6. Validate with go build
	fmt.Printf("\n[pkm-ingest] validating %d generated files...\n", len(generated))
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = kbDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n[pkm-ingest] go build FAILED — generated code needs fixes\n")
		fmt.Fprintf(os.Stderr, "  files: %s\n", strings.Join(generated, ", "))
		os.Exit(1)
	}

	// Also run vet
	cmd = exec.Command("go", "vet", "./...")
	cmd.Dir = kbDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[pkm-ingest] go vet WARNING: %v\n", err)
	}

	fmt.Printf("[pkm-ingest] SUCCESS: %d entities, %d claims across %d files\n",
		len(entities), len(claims), len(generated))
	for _, f := range generated {
		fmt.Printf("  %s\n", filepath.Base(f))
	}
}

// parseVaultNotes walks the vault directory and parses all .md files.
func parseVaultNotes(vaultDir string) []pkmNote {
	var notes []pkmNote

	err := filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden dirs and .git
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		// Skip README and LICENSE
		if info.Name() == "README.md" || info.Name() == "LICENSE" || info.Name() == "LICENSE.md" {
			return nil
		}

		rel, _ := filepath.Rel(vaultDir, path)
		note := parseSingleNote(path, rel)
		if note != nil {
			notes = append(notes, *note)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pkm-ingest: walk %s: %v\n", vaultDir, err)
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].relPath < notes[j].relPath
	})
	return notes
}

var (
	wikilinkRe   = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
	predictionRe = regexp.MustCompile(`(?i)\*\*prediction\*\*:\s*(.+)`)
	authorRe     = regexp.MustCompile(`(?i)^\*\*author:?\*\*\s*(.+)`)
	ratingRe     = regexp.MustCompile(`(?i)^\*\*rating:?\*\*\s*(.+)`)
	h1Re         = regexp.MustCompile(`^#\s+(.+)`)
)

func parseSingleNote(path, relPath string) *pkmNote {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	note := &pkmNote{
		relPath: relPath,
		dir:     filepath.Dir(relPath),
	}
	if note.dir == "." {
		note.dir = "root"
	}

	var content strings.Builder
	linkSeen := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		content.WriteString(line)
		content.WriteString("\n")

		// Extract title from first H1
		if note.title == "" {
			if m := h1Re.FindStringSubmatch(line); m != nil {
				note.title = strings.TrimSpace(m[1])
			}
		}

		// Extract author
		if note.author == "" {
			if m := authorRe.FindStringSubmatch(line); m != nil {
				note.author = strings.TrimSpace(m[1])
			}
		}

		// Extract rating
		if note.rating == "" {
			if m := ratingRe.FindStringSubmatch(line); m != nil {
				note.rating = strings.TrimSpace(m[1])
			}
		}

		// Extract predictions
		if m := predictionRe.FindStringSubmatch(line); m != nil {
			note.predictions = append(note.predictions, strings.TrimSpace(m[1]))
		}

		// Extract wikilinks
		for _, m := range wikilinkRe.FindAllStringSubmatch(line, -1) {
			target := strings.TrimSpace(m[1])
			if !linkSeen[target] {
				linkSeen[target] = true
				note.wikilinks = append(note.wikilinks, target)
			}
		}
	}

	note.content = content.String()

	// Fallback title from filename
	if note.title == "" {
		base := filepath.Base(relPath)
		note.title = strings.TrimSuffix(base, ".md")
		note.title = strings.ReplaceAll(note.title, "-", " ")
		note.title = strings.Title(note.title)
	}

	return note
}

// collectExistingEntityNames scans the KB for existing entity variable names and IDs.
func collectExistingEntityNames(dir string) map[string]string {
	existing := map[string]string{} // varName -> id
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return existing
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		// Skip pkm_ files (our own output)
		if strings.HasPrefix(e.Name(), "pkm_") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}

		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, nameIdent := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					// Check if this is a role-typed entity (Person{...}, Concept{...}, etc.)
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					typeName := ""
					switch t := cl.Type.(type) {
					case *ast.Ident:
						typeName = t.Name
					}
					roleTypes := map[string]bool{
						"Person": true, "Organization": true, "Place": true,
						"Event": true, "Facility": true, "Substance": true,
						"Instrument": true, "Hypothesis": true, "Concept": true,
					}
					if !roleTypes[typeName] {
						continue
					}
					// Extract ID
					id := extractEntityID(cl)
					existing[nameIdent.Name] = id
				}
			}
		}
	}
	return existing
}

// extractEntityID pulls the ID field from a role-typed entity literal.
func extractEntityID(cl *ast.CompositeLit) string {
	// Role types wrap &Entity{...} — the first element is a unary expression
	for _, elt := range cl.Elts {
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, field := range inner.Elts {
			kv, ok := field.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "ID" {
				return resolveStringExpr(kv.Value)
			}
		}
	}
	return ""
}

// extractPKMContent generates entities and claims from parsed notes.
func extractPKMContent(notes []pkmNote, existing map[string]string) ([]pkmEntity, []pkmClaim, map[string]pkmProvenance) {
	var entities []pkmEntity
	var claims []pkmClaim
	provs := map[string]pkmProvenance{} // provVar -> provenance

	entityByID := map[string]*pkmEntity{}    // id -> entity
	existingByID := map[string]string{}       // id -> varName (existing KB)
	for varName, id := range existing {
		existingByID[id] = varName
	}

	// Note title -> entity var name (for wikilink resolution)
	noteEntityMap := map[string]string{} // lowercase note slug -> entity var name

	// First pass: create entities for each note (the note itself is a concept)
	for _, note := range notes {
		noteID := slugify(note.title)
		noteVar := "Pkm" + goIdentifier(note.title)

		// Check if this entity already exists in the KB
		if existingVar, ok := existingByID[noteID]; ok {
			noteEntityMap[strings.ToLower(slugFromPath(note.relPath))] = existingVar
			continue
		}

		// Determine role type based on directory and content
		roleType := "Concept"
		kind := "concept"
		brief := truncateBrief(note.title, note.content)

		ent := pkmEntity{
			varName:    noteVar,
			roleType:   roleType,
			id:         noteID,
			name:       note.title,
			kind:       kind,
			brief:      brief,
			sourceNote: note.relPath,
		}

		if _, exists := entityByID[noteID]; !exists {
			entityByID[noteID] = &ent
			entities = append(entities, ent)
		}
		noteEntityMap[strings.ToLower(slugFromPath(note.relPath))] = ent.varName
	}

	// Second pass: create author entities and Authored claims for book notes
	for _, note := range notes {
		if note.author == "" || note.dir != "books" {
			continue
		}

		authorID := slugify(note.author)
		authorVar := "Pkm" + goIdentifier(note.author)

		// Check existing KB
		if existingVar, ok := existingByID[authorID]; ok {
			authorVar = existingVar
		} else if _, exists := entityByID[authorID]; !exists {
			ent := pkmEntity{
				varName:    authorVar,
				roleType:   "Person",
				id:         authorID,
				name:       note.author,
				kind:       "person",
				brief:      fmt.Sprintf("Author of %s.", note.title),
				sourceNote: note.relPath,
			}
			entityByID[authorID] = &ent
			entities = append(entities, ent)
		}

		// Find the note's entity var
		noteSlug := strings.ToLower(slugFromPath(note.relPath))
		noteVar := noteEntityMap[noteSlug]
		if noteVar == "" {
			continue
		}

		// Create provenance
		provVar := "pkm" + goIdentifier(note.title) + "Source"
		provs[provVar] = pkmProvenance{
			origin: fmt.Sprintf("PKM vault / %s", note.relPath),
			quote:  extractLeadSentence(note.content),
		}

		// Authored claim
		claimVar := authorVar + "Authored" + stripPrefix(noteVar, "Pkm")
		claims = append(claims, pkmClaim{
			varName:    claimVar,
			predicate:  "Authored",
			subject:    authorVar,
			object:     noteVar,
			provVar:    provVar,
			sourceNote: note.relPath,
		})
	}

	// Third pass: create prediction hypotheses
	predCount := 0
	for _, note := range notes {
		for _, pred := range note.predictions {
			predCount++
			predID := fmt.Sprintf("pkm-prediction-%d", predCount)
			predVar := fmt.Sprintf("PkmPrediction%d", predCount)

			// Truncate prediction text for brief
			predBrief := pred
			if len(predBrief) > 200 {
				predBrief = predBrief[:197] + "..."
			}

			ent := pkmEntity{
				varName:    predVar,
				roleType:   "Hypothesis",
				id:         predID,
				name:       fmt.Sprintf("PKM Prediction %d", predCount),
				kind:       "hypothesis",
				brief:      predBrief,
				sourceNote: note.relPath,
			}
			entityByID[predID] = &ent
			entities = append(entities, ent)

			// Find the note entity to create a relationship
			noteSlug := strings.ToLower(slugFromPath(note.relPath))
			noteVar := noteEntityMap[noteSlug]

			// Create provenance for the prediction
			provVar := fmt.Sprintf("pkmPred%dSource", predCount)
			provs[provVar] = pkmProvenance{
				origin: fmt.Sprintf("PKM vault / %s", note.relPath),
				quote:  pred,
			}

			// If the note has an author, the author proposes the prediction
			if note.author != "" {
				authorID := slugify(note.author)
				authorVar := "Pkm" + goIdentifier(note.author)
				if existingVar, ok := existingByID[authorID]; ok {
					authorVar = existingVar
				}
				claimVar := authorVar + "Proposes" + predVar
				claims = append(claims, pkmClaim{
					varName:    claimVar,
					predicate:  "Proposes",
					subject:    authorVar,
					object:     predVar,
					provVar:    provVar,
					sourceNote: note.relPath,
				})
			}

			// TheoryOf: prediction is about the note's concept
			if noteVar != "" {
				claimVar := predVar + "TheoryOf" + stripPrefix(noteVar, "Pkm")
				claims = append(claims, pkmClaim{
					varName:    claimVar,
					predicate:  "TheoryOf",
					subject:    predVar,
					object:     noteVar,
					provVar:    provVar,
					sourceNote: note.relPath,
				})
			}
		}
	}

	// Fourth pass: create BelongsTo claims from directory structure
	// (all notes in "books" belong to a "PKM Books" category, etc.)
	categoryEntities := map[string]string{} // dir -> entity var name
	dirLabels := map[string]string{
		"books":        "Book Notes",
		"productivity": "Productivity Methods",
		"coffee":       "Coffee Brewing",
	}

	for dir, label := range dirLabels {
		catID := "pkm-category-" + dir
		catVar := "PkmCategory" + goIdentifier(dir)

		if _, exists := entityByID[catID]; !exists {
			ent := pkmEntity{
				varName:    catVar,
				roleType:   "Concept",
				id:         catID,
				name:       label,
				kind:       "concept",
				brief:      fmt.Sprintf("PKM category: %s.", label),
				sourceNote: dir + "/",
			}
			entityByID[catID] = &ent
			entities = append(entities, ent)
		}
		categoryEntities[dir] = catVar
	}

	for _, note := range notes {
		catVar, ok := categoryEntities[note.dir]
		if !ok {
			continue
		}
		noteSlug := strings.ToLower(slugFromPath(note.relPath))
		noteVar := noteEntityMap[noteSlug]
		if noteVar == "" {
			continue
		}

		provVar := "pkm" + goIdentifier(note.title) + "Source"
		if _, ok := provs[provVar]; !ok {
			provs[provVar] = pkmProvenance{
				origin: fmt.Sprintf("PKM vault / %s", note.relPath),
				quote:  extractLeadSentence(note.content),
			}
		}

		claimVar := stripPrefix(noteVar, "Pkm") + "BelongsTo" + stripPrefix(catVar, "Pkm")
		claims = append(claims, pkmClaim{
			varName:    claimVar,
			predicate:  "BelongsTo",
			subject:    noteVar,
			object:     catVar,
			provVar:    provVar,
			sourceNote: note.relPath,
		})
	}

	return entities, claims, provs
}

type pkmProvenance struct {
	origin string
	quote  string
}

// generatePKMFile writes a .go file for one vault subdirectory.
func generatePKMFile(kbDir, dir string, notes []pkmNote, entities []pkmEntity, claims []pkmClaim, provs map[string]pkmProvenance) string {
	safeName := sanitizeDir(dir)
	outPath := filepath.Join(kbDir, fmt.Sprintf("pkm_%s.go", safeName))

	var b strings.Builder
	b.WriteString("package winze\n\n")
	b.WriteString(fmt.Sprintf("// PKM ingest: %s/\n", dir))
	b.WriteString(fmt.Sprintf("// Generated by: go run ./cmd/metabolism --pkm <vault> .\n"))
	b.WriteString(fmt.Sprintf("// Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("//\n")
	b.WriteString("// Review before committing: verify entity briefs and claims\n")
	b.WriteString("// accurately reflect the source notes.\n\n")

	// Collect provenance vars needed for this directory
	dirProvs := map[string]bool{}
	dirEntities := []pkmEntity{}
	dirClaims := []pkmClaim{}

	for _, e := range entities {
		if noteDir(e.sourceNote) == dir {
			dirEntities = append(dirEntities, e)
		}
	}
	for _, c := range claims {
		if noteDir(c.sourceNote) == dir {
			dirClaims = append(dirClaims, c)
			dirProvs[c.provVar] = true
		}
	}

	if len(dirEntities) == 0 && len(dirClaims) == 0 {
		return ""
	}

	// Write provenance vars
	var provKeys []string
	for k := range dirProvs {
		provKeys = append(provKeys, k)
	}
	sort.Strings(provKeys)
	for _, pv := range provKeys {
		p := provs[pv]
		quote := cleanGoString(p.quote)
		if len(quote) > 200 {
			quote = quote[:197] + "..."
		}
		b.WriteString(fmt.Sprintf("var %s = Provenance{\n", pv))
		b.WriteString(fmt.Sprintf("\tOrigin:     %q,\n", p.origin))
		b.WriteString(fmt.Sprintf("\tIngestedAt: %q,\n", time.Now().Format("2006-01-02")))
		b.WriteString(fmt.Sprintf("\tIngestedBy: \"winze pkm-ingest\",\n"))
		b.WriteString(fmt.Sprintf("\tQuote:      %q,\n", quote))
		b.WriteString("}\n\n")
	}

	// Write entities
	for _, e := range dirEntities {
		brief := cleanGoString(e.brief)
		if len(brief) > 250 {
			brief = brief[:247] + "..."
		}
		b.WriteString(fmt.Sprintf("var %s = %s{&Entity{\n", e.varName, e.roleType))
		b.WriteString(fmt.Sprintf("\tID:    %q,\n", e.id))
		b.WriteString(fmt.Sprintf("\tName:  %q,\n", e.name))
		b.WriteString(fmt.Sprintf("\tKind:  %q,\n", e.kind))
		b.WriteString(fmt.Sprintf("\tBrief: %q,\n", brief))
		b.WriteString("}}\n\n")
	}

	// Write claims
	for _, c := range dirClaims {
		b.WriteString(fmt.Sprintf("var %s = %s{\n", c.varName, c.predicate))
		b.WriteString(fmt.Sprintf("\tSubject: %s,\n", c.subject))
		b.WriteString(fmt.Sprintf("\tObject:  %s,\n", c.object))
		b.WriteString(fmt.Sprintf("\tProv:    %s,\n", c.provVar))
		b.WriteString("}\n\n")
	}

	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "pkm-ingest: write %s: %v\n", outPath, err)
		return ""
	}

	fmt.Printf("  wrote %s: %d entities, %d claims\n", filepath.Base(outPath), len(dirEntities), len(dirClaims))
	return outPath
}

// --- helpers ---

func slugify(s string) string {
	var result []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, r)
		} else if len(result) > 0 && result[len(result)-1] != '-' {
			result = append(result, '-')
		}
	}
	return strings.Trim(string(result), "-")
}

func slugFromPath(relPath string) string {
	base := filepath.Base(relPath)
	return strings.TrimSuffix(base, ".md")
}

func noteDir(sourceNote string) string {
	d := filepath.Dir(sourceNote)
	if d == "." {
		return "root"
	}
	return d
}

func sanitizeDir(dir string) string {
	return strings.ReplaceAll(strings.ReplaceAll(dir, "/", "_"), ".", "root")
}

func stripPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

func truncateBrief(title, content string) string {
	// Extract first meaningful paragraph after the title
	lines := strings.Split(content, "\n")
	var paragraphs []string
	current := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != "" {
				paragraphs = append(paragraphs, current)
				current = ""
			}
			continue
		}
		// Skip metadata lines and headings
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "**Author") ||
			strings.HasPrefix(line, "**Rating") || strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "*Notes adapted") || strings.HasPrefix(line, "*Original summary") {
			continue
		}
		if current != "" {
			current += " "
		}
		current += line
	}
	if current != "" {
		paragraphs = append(paragraphs, current)
	}

	if len(paragraphs) == 0 {
		return title
	}

	brief := paragraphs[0]
	// Strip markdown formatting
	brief = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`).ReplaceAllString(brief, "$1")
	brief = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(brief, "$1")
	brief = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(brief, "$1")
	brief = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(brief, "$1")
	brief = regexp.MustCompile(`> `).ReplaceAllString(brief, "")

	if len(brief) > 250 {
		brief = brief[:247] + "..."
	}
	return brief
}

func extractLeadSentence(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "**") ||
			strings.HasPrefix(line, "---") || strings.HasPrefix(line, "*") ||
			strings.HasPrefix(line, "|") {
			continue
		}
		// Strip markdown
		line = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`).ReplaceAllString(line, "$1")
		line = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(line, "$1")
		if len(line) > 200 {
			line = line[:197] + "..."
		}
		return line
	}
	return ""
}

// goIdentifier converts a string to a valid Go identifier (PascalCase, alphanumeric only).
func goIdentifier(s string) string {
	// Split on any non-alphanumeric character
	var words []string
	current := ""
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current += string(r)
		} else {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		}
	}
	if current != "" {
		words = append(words, current)
	}

	var parts []string
	for _, w := range words {
		if len(w) > 0 {
			// Capitalize first letter, lowercase rest
			first := strings.ToUpper(w[:1])
			rest := ""
			if len(w) > 1 {
				rest = strings.ToLower(w[1:])
			}
			parts = append(parts, first+rest)
		}
	}

	result := strings.Join(parts, "")
	// Ensure it doesn't start with a digit
	if len(result) > 0 && unicode.IsDigit(rune(result[0])) {
		result = "N" + result
	}
	return result
}

func cleanGoString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
