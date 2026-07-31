package vault

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"
)

// Index resolves link targets to notes. Build it once per vault; every lookup
// is a map hit, so resolving a vault's whole link set is linear in the number
// of links rather than in notes × links.
type Index struct {
	notes []Note
	byKey map[string][]int // normalized key -> note indices claiming it
}

// Edge is a resolved link: one note referencing another, with the section it
// was written under.
type Edge struct {
	From    int // index into Notes()
	To      int
	Section string
	Display string
	Line    int
}

// Unresolved is a link that names no note in the vault.
type Unresolved struct {
	From   int
	Target string
	Line   int
	// Ambiguous is set when the target matched more than one note. That is a
	// different problem from a missing note — two notes are fighting over a
	// name — and it is reported separately so it can be fixed rather than
	// treated as a gap to fill.
	Ambiguous []string
}

// NewIndex builds the resolution table. Keys are registered most-specific
// first; a later note claiming an already-taken key makes that key ambiguous
// rather than overwriting it, because silently picking a winner would put a
// wrong edge in the graph.
func NewIndex(notes []Note) *Index {
	ix := &Index{notes: notes, byKey: make(map[string][]int, len(notes)*3)}
	for i, n := range notes {
		ix.add(n.RelPath, i)
		ix.add(strings.TrimSuffix(n.RelPath, path.Ext(n.RelPath)), i)
		ix.add(path.Base(strings.TrimSuffix(n.RelPath, path.Ext(n.RelPath))), i)
		ix.add(n.Title, i)
		for _, a := range n.Aliases {
			ix.add(a, i)
		}
	}
	return ix
}

func (ix *Index) add(key string, i int) {
	k := normalize(key)
	if k == "" {
		return
	}
	for _, existing := range ix.byKey[k] {
		if existing == i {
			return
		}
	}
	ix.byKey[k] = append(ix.byKey[k], i)
}

// Notes returns the indexed notes in their stable order.
func (ix *Index) Notes() []Note { return ix.notes }

// Resolve maps one link target, as written in note `from`, to a note index.
// A target is tried as a vault-relative path first (so `[[systems/combat]]`
// beats a same-named note elsewhere), then as a path relative to the linking
// note's directory, then by bare name, title, or alias.
func (ix *Index) Resolve(from int, target string) (int, []string, bool) {
	candidates := []string{target, strings.TrimSuffix(target, path.Ext(target))}
	if dir := ix.notes[from].Dir; dir != "" {
		joined := path.Join(dir, target)
		candidates = append(candidates, joined, strings.TrimSuffix(joined, path.Ext(joined)))
	}
	candidates = append(candidates, path.Base(strings.TrimSuffix(target, path.Ext(target))))

	for _, c := range candidates {
		hits := ix.byKey[normalize(c)]
		switch len(hits) {
		case 0:
			continue
		case 1:
			return hits[0], nil, true
		default:
			names := make([]string, 0, len(hits))
			for _, h := range hits {
				names = append(names, ix.notes[h].RelPath)
			}
			sort.Strings(names)
			return -1, names, false
		}
	}
	return -1, nil, false
}

// Edges resolves every link in the vault. Self-links and duplicate
// (from, to, section) triples are dropped: a note referencing itself is not a
// graph edge, and the same reference repeated in one section is one claim.
//
// Both return values are ordered deterministically, so regenerating a corpus
// from an unchanged vault produces an unchanged diff.
func (ix *Index) Edges() ([]Edge, []Unresolved) {
	var edges []Edge
	var missing []Unresolved
	seen := make(map[[2]int]map[string]bool)

	for i, n := range ix.notes {
		for _, l := range n.Links {
			to, ambiguous, ok := ix.Resolve(i, l.Target)
			if !ok {
				missing = append(missing, Unresolved{From: i, Target: l.Target, Line: l.Line, Ambiguous: ambiguous})
				continue
			}
			if to == i {
				continue
			}
			pair := [2]int{i, to}
			if seen[pair] == nil {
				seen[pair] = map[string]bool{}
			}
			if seen[pair][l.Section] {
				continue
			}
			seen[pair][l.Section] = true
			edges = append(edges, Edge{From: i, To: to, Section: l.Section, Display: l.Display, Line: l.Line})
		}
	}

	sort.Slice(edges, func(a, b int) bool {
		if edges[a].From != edges[b].From {
			return edges[a].From < edges[b].From
		}
		if edges[a].To != edges[b].To {
			return edges[a].To < edges[b].To
		}
		return edges[a].Section < edges[b].Section
	})
	sort.Slice(missing, func(a, b int) bool {
		if missing[a].From != missing[b].From {
			return missing[a].From < missing[b].From
		}
		if missing[a].Target != missing[b].Target {
			return missing[a].Target < missing[b].Target
		}
		return missing[a].Line < missing[b].Line
	})
	return edges, missing
}

// Stats summarises a vault's link health — what an operator wants to see before
// committing an ingest.
type Stats struct {
	Notes       int
	Links       int
	Resolved    int
	Unresolved  int
	Ambiguous   int
	Orphans     int // notes with no inbound and no outbound resolved edge
	WithType    int // notes declaring a frontmatter `type`
	Sections    int // distinct heading labels links appeared under
	TopSections []string
}

func (ix *Index) Stats() Stats {
	edges, missing := ix.Edges()
	s := Stats{Notes: len(ix.notes), Resolved: len(edges)}
	for _, m := range missing {
		if len(m.Ambiguous) > 0 {
			s.Ambiguous++
		} else {
			s.Unresolved++
		}
	}
	s.Links = s.Resolved + s.Unresolved + s.Ambiguous

	connected := make([]bool, len(ix.notes))
	sections := map[string]int{}
	for _, e := range edges {
		connected[e.From], connected[e.To] = true, true
		if e.Section != "" {
			sections[e.Section]++
		}
	}
	for i, c := range connected {
		if !c {
			s.Orphans++
		}
		if ix.notes[i].Type != "" {
			s.WithType++
		}
	}
	s.Sections = len(sections)

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
	for i := 0; i < len(list) && i < 10; i++ {
		s.TopSections = append(s.TopSections, fmt.Sprintf("%s (%d)", list[i].k, list[i].n))
	}
	return s
}

// normalize is the identity function for link resolution: case-insensitive,
// punctuation-insensitive, whitespace-insensitive. `[[Combat Resolution]]`,
// `[[combat-resolution]]`, and `[[Combat_Resolution]]` name the same note,
// because in a real vault they are all written by the same person meaning the
// same thing.
func normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSep := true
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevSep = false
		case r == '/':
			b.WriteRune('/')
			prevSep = true
		default:
			if !prevSep {
				b.WriteRune('-')
				prevSep = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
