package main

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/justinstimatze/winze/internal/vault"
)

// Vault ingest: markdown notes -> typed corpus.
//
// The previous extraction was written against one demo vault and encoded it
// literally: author lines only counted inside a `books/` directory, categories
// came from a hardcoded {books, productivity, coffee} map that materialised
// three entities whether or not those directories existed, every note became a
// Concept regardless of content, and — the load-bearing defect — wikilinks were
// parsed into a field that nothing ever read. On a six-note game-design vault
// with thirteen links between the notes, it produced nine entities and ZERO
// claims: a disconnected bag with three phantom categories from someone else's
// coffee notes.
//
// That is the difference between winze and a folder of markdown, and it was on
// the wrong side of it. A vault's links ARE its graph; discarding them left
// winze with nothing to offer that the markdown did not already have.
//
// What replaces it maps only what a note actually commits to:
//
//	note                 -> entity, role from frontmatter `type` when it names
//	                        a real winze role, else Concept
//	[[link]]             -> References claim, provenance quoting the linking line
//	frontmatter aliases  -> entity Aliases
//	directory            -> FiledUnder a category entity, one per directory that
//	                        actually exists
//
// Nothing is inferred beyond that. A link under `## Depends on` is *evidence*
// for a typed DependsOn, and the section is recorded so a later pass can
// propose one — but the mechanical path never invents the predicate, because a
// link commits to the reference and deliberately not to its kind.

// roleForType maps a frontmatter `type` onto a winze role. Vault authors write
// whatever they like, so matching is normalized and unknown values fall back to
// Concept rather than coining a role — schema accretion waits for a forcing
// function, and one vault's frontmatter is not one.
var roleForType = map[string]string{
	"person": "Person", "people": "Person", "character": "Person", "npc": "Person",
	"author": "Person", "designer": "Person",
	"org": "Organization", "organization": "Organization", "faction": "Organization",
	"guild": "Organization", "company": "Organization",
	"place": "Place", "location": "Place", "region": "Place", "zone": "Place",
	"area": "Place", "level": "Place", "map": "Place",
	"event": "Event", "encounter": "Event", "quest": "Event", "session": "Event",
	"hypothesis": "Hypothesis", "theory": "Hypothesis", "proposal": "Hypothesis",
	"rfc": "Hypothesis", "design-decision": "Hypothesis",
	"item": "Instrument", "weapon": "Instrument", "tool": "Instrument",
	"instrument": "Instrument", "artifact": "Instrument",
	"facility": "Facility", "building": "Facility", "structure": "Facility",
	"substance": "Substance", "material": "Substance", "resource": "Substance",
	"concept": "Concept", "system": "Concept", "mechanic": "Concept",
	"note": "Concept", "topic": "Concept", "lore": "Concept",
}

// vaultResult is everything one ingest run produces, ready to render.
type vaultResult struct {
	entities []pkmEntity
	claims   []pkmClaim
	provs    map[string]pkmProvenance

	notes      []vault.Note
	stats      vault.Stats
	skipped    int      // notes whose entity already existed in the KB
	unresolved []string // human-readable unresolved/ambiguous link report
	sections   map[string]int
}

// extractVault turns a parsed vault into entities and claims. `existing` maps
// an existing KB entity var name to its ID, so a re-run wires new links to the
// entities already committed instead of coining duplicates.
func extractVault(notes []vault.Note, ix *vault.Index, existing map[string]string, ingestedAt string) *vaultResult {
	res := &vaultResult{
		provs:    map[string]pkmProvenance{},
		notes:    notes,
		stats:    ix.Stats(),
		sections: map[string]int{},
	}

	existingByID := make(map[string]string, len(existing))
	for varName, id := range existing {
		existingByID[id] = varName
	}

	// Pass 1: one entity per note. varForNote maps a note index to the var name
	// claims must use — an existing KB var when the note is already ingested, a
	// freshly coined one otherwise.
	varForNote := make([]string, len(notes))
	usedVar := map[string]bool{}
	for varName := range existing {
		usedVar[varName] = true
	}

	for i, n := range notes {
		id := slugify(n.Title)
		if id == "" {
			id = slugify(strings.TrimSuffix(n.RelPath, path.Ext(n.RelPath)))
		}
		if existingVar, ok := existingByID[id]; ok {
			varForNote[i] = existingVar
			res.skipped++
			continue
		}
		varName := uniqueVar(goIdentifier(n.Title), usedVar)
		varForNote[i] = varName

		res.entities = append(res.entities, pkmEntity{
			varName:    varName,
			roleType:   roleFor(n),
			id:         id,
			name:       n.Title,
			kind:       strings.ToLower(roleFor(n)),
			brief:      briefFor(n),
			sourceNote: n.RelPath,
		})
	}

	// Pass 2: links become References claims. Provenance is per source note, so
	// every claim from one note shares one provenance var rather than
	// fragmenting — provenance-split lint exists to catch the alternative.
	edges, missing := ix.Edges()
	usedClaim := map[string]bool{}
	for _, e := range edges {
		from, to := notes[e.From], notes[e.To]
		subj, obj := varForNote[e.From], varForNote[e.To]
		if subj == "" || obj == "" || subj == obj {
			continue
		}
		provVar := provVarFor(from)
		if _, ok := res.provs[provVar]; !ok {
			res.provs[provVar] = pkmProvenance{
				origin: fmt.Sprintf("vault / %s", from.RelPath),
				quote:  linkQuote(from, to, e.Section),
			}
		}
		if e.Section != "" {
			res.sections[e.Section]++
		}
		res.claims = append(res.claims, pkmClaim{
			varName:    uniqueVar(subj+"References"+obj, usedClaim),
			predicate:  "References",
			subject:    subj + ".Entity",
			object:     obj + ".Entity",
			provVar:    provVar,
			sourceNote: from.RelPath,
		})
	}

	// Pass 3: directories become categories — the ones that exist, not a fixed
	// list. A flat vault produces none, which is correct; the old hardcoded map
	// produced three regardless.
	res.addCategories(notes, varForNote, existingByID, usedVar, usedClaim)

	for _, m := range missing {
		from := notes[m.From].RelPath
		if len(m.Ambiguous) > 0 {
			res.unresolved = append(res.unresolved,
				fmt.Sprintf("%s:%d [[%s]] is ambiguous: %s", from, m.Line, m.Target, strings.Join(m.Ambiguous, ", ")))
			continue
		}
		res.unresolved = append(res.unresolved,
			fmt.Sprintf("%s:%d [[%s]] names no note in the vault", from, m.Line, m.Target))
	}
	return res
}

func (res *vaultResult) addCategories(notes []vault.Note, varForNote []string, existingByID map[string]string, usedVar, usedClaim map[string]bool) {
	dirs := map[string]bool{}
	for _, n := range notes {
		if n.Dir != "" {
			dirs[n.Dir] = true
		}
	}
	if len(dirs) == 0 {
		return
	}
	names := make([]string, 0, len(dirs))
	for d := range dirs {
		names = append(names, d)
	}
	sort.Strings(names)

	catVar := map[string]string{}
	for _, dir := range names {
		id := "vault-category-" + slugify(dir)
		if existing, ok := existingByID[id]; ok {
			catVar[dir] = existing
			continue
		}
		v := uniqueVar("Category"+goIdentifier(dir), usedVar)
		catVar[dir] = v
		res.entities = append(res.entities, pkmEntity{
			varName:    v,
			roleType:   "Concept",
			id:         id,
			name:       categoryLabel(dir),
			kind:       "concept",
			brief:      fmt.Sprintf("Vault category: notes under %s/.", dir),
			sourceNote: dir + "/",
		})
	}

	for i, n := range notes {
		v, ok := catVar[n.Dir]
		if !ok || varForNote[i] == "" {
			continue
		}
		provVar := provVarFor(n)
		if _, ok := res.provs[provVar]; !ok {
			res.provs[provVar] = pkmProvenance{
				origin: fmt.Sprintf("vault / %s", n.RelPath),
				quote:  n.Lead,
			}
		}
		res.claims = append(res.claims, pkmClaim{
			varName:    uniqueVar(varForNote[i]+"FiledUnder"+v, usedClaim),
			predicate:  "FiledUnder",
			subject:    varForNote[i] + ".Entity",
			object:     v,
			provVar:    provVar,
			sourceNote: n.RelPath,
		})
	}
}

// roleFor reads the note's own declared type. Frontmatter is the only place a
// vault states what a note IS, and it is exactly what an OKF bundle carries —
// so an OKF bundle dropped into this path arrives already typed.
func roleFor(n vault.Note) string {
	if n.Type == "" {
		return "Concept"
	}
	if role, ok := roleForType[normalizeType(n.Type)]; ok {
		return role
	}
	return "Concept"
}

func normalizeType(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// briefFor prefers the note's own lead sentence and falls back to its title.
// It never synthesizes prose — a Brief that the note does not support is an
// invented claim wearing an entity's name.
func briefFor(n vault.Note) string {
	if n.Lead != "" {
		return n.Lead
	}
	return n.Title + "."
}

// linkQuote is the source text backing a References claim: the section the link
// appeared under, plus the linking note's lead. Mirror-source-commitments —
// the quote has to be text the note actually contains.
func linkQuote(from, to vault.Note, section string) string {
	if section != "" {
		return fmt.Sprintf("%s — under %q — links to %s", from.Title, section, to.Title)
	}
	return fmt.Sprintf("%s links to %s", from.Title, to.Title)
}

func provVarFor(n vault.Note) string {
	return "vault" + goIdentifier(strings.TrimSuffix(n.RelPath, path.Ext(n.RelPath))) + "Source"
}

func categoryLabel(dir string) string {
	parts := strings.FieldsFunc(dir, func(r rune) bool { return r == '/' || r == '-' || r == '_' })
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// uniqueVar keeps generated identifiers collision-free. Two notes titled the
// same in different directories are a real occurrence in a vault, and emitting
// one var name twice would not compile — the build gate would catch it, but
// failing the whole ingest over a name clash is not a useful outcome.
func uniqueVar(base string, used map[string]bool) string {
	if base == "" {
		base = "Note"
	}
	name := base
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	used[name] = true
	return name
}
