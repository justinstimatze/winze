package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// briefDriftRule surfaces prose that asserts a relationship the claim graph
// does not encode: an entity whose Brief names another entity it has no claim
// to, in either direction.
//
// The failure mode this guards against was observed in a sibling project,
// where a hand-maintained status field kept saying "partial" next to a note
// that already said "done" — nothing forced the structured field and the
// prose to agree, so a generated report stayed wrong for two sessions. The
// same shape lives here as Entity.Brief versus the entity's claims: editing a
// Brief has no forcing function back onto the claim graph.
//
// Deliberately ADVISORY. Winze's authoring discipline permits Brief-level
// references for connections a source does not explicitly commit to
// (mirror-source-commitments), so a Brief mention with no claim is often
// correct rather than a defect. The rule earns its place by being read two
// ways: as drift detection, and as a worklist of things written about but
// never wired up.
func briefDriftRule(dir string) int {
	entities, claims, err := corpusparse.ParseCorpus(dir)
	if err != nil {
		fmt.Printf("[brief-drift] error: %v\n", err)
		return 2
	}

	// Undirected adjacency over claim endpoints. Direction does not matter:
	// the question is whether the two entities are related at all.
	adj := make(map[string]map[string]bool, len(claims))
	addEdge := func(a, b string) {
		if adj[a] == nil {
			adj[a] = map[string]bool{}
		}
		adj[a][b] = true
	}
	for _, c := range claims {
		if c.SubjectVar == "" || c.ObjectVar == "" {
			continue
		}
		addEdge(c.SubjectVar, c.ObjectVar)
		addEdge(c.ObjectVar, c.SubjectVar)
	}

	// Related within two hops, not one. Winze routinely models a person's
	// relationship to a concept through an intermediate framing entity
	// (KlausConrad -> ConradApopheniaClinicalFraming -> Apophenia), so
	// requiring a direct edge would flag the house pattern as drift.
	related := func(a, b string) bool {
		if adj[a][b] {
			return true
		}
		for mid := range adj[a] {
			if adj[mid][b] {
				return true
			}
		}
		return false
	}

	type mention struct {
		target  string // var name of the mentioned entity
		surface string // the text that matched
	}

	matchers := buildMentionMatchers(entities)

	type finding struct {
		entity   corpusparse.Entity
		mentions []mention
	}
	var findings []finding
	totalMentions := 0

	for _, e := range entities {
		if e.Brief == "" || corpusparse.IsReifyMachinery(e.VarName) {
			continue
		}
		var ms []mention
		seen := map[string]bool{}
		for _, m := range matchers {
			if m.varName == e.VarName || seen[m.varName] {
				continue
			}
			if related(e.VarName, m.varName) {
				continue
			}
			if loc := m.re.FindString(e.Brief); loc != "" {
				seen[m.varName] = true
				ms = append(ms, mention{target: m.varName, surface: loc})
			}
		}
		if len(ms) > 0 {
			sort.Slice(ms, func(i, j int) bool { return ms[i].target < ms[j].target })
			findings = append(findings, finding{entity: e, mentions: ms})
			totalMentions += len(ms)
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if len(findings[i].mentions) != len(findings[j].mentions) {
			return len(findings[i].mentions) > len(findings[j].mentions)
		}
		return findings[i].entity.VarName < findings[j].entity.VarName
	})

	fmt.Printf("[brief-drift] %d entities, %d with Brief mentions lacking a connecting claim (%d mentions total)\n",
		len(entities), len(findings), totalMentions)

	const maxShown = 15
	for i, f := range findings {
		if i >= maxShown {
			fmt.Printf("  ... and %d more\n", len(findings)-maxShown)
			break
		}
		fmt.Printf("  %s (%s) %s\n", f.entity.VarName, f.entity.RoleType, f.entity.File)
		for _, m := range f.mentions {
			fmt.Printf("      Brief names %q -> %s, but no claim links them\n", m.surface, m.target)
		}
	}
	if len(findings) > 0 {
		fmt.Println("  (advisory — a Brief-level reference with no claim is permitted; treat as a candidate-claim worklist)")
	}
	return 0
}

type mentionMatcher struct {
	varName string
	re      *regexp.Regexp
}

// minSurfaceLen is the shortest surface form worth matching. Below this,
// entity names collide with ordinary English ("Mind", "Self") and every Brief
// in the corpus matches everything.
const minSurfaceLen = 6

// buildMentionMatchers compiles one word-boundary matcher per entity over its
// Name and aliases. Very short surfaces and multi-clause Names (hypothesis
// Names are often whole sentences, which never appear verbatim inside another
// entity's Brief) are skipped.
func buildMentionMatchers(entities []corpusparse.Entity) []mentionMatcher {
	var out []mentionMatcher
	for _, e := range entities {
		if corpusparse.IsReifyMachinery(e.VarName) {
			continue
		}
		surfaces := make([]string, 0, len(e.Aliases)+1)
		if e.Name != "" {
			surfaces = append(surfaces, e.Name)
		}
		surfaces = append(surfaces, e.Aliases...)

		var alts []string
		seen := map[string]bool{}
		for _, s := range surfaces {
			s = strings.TrimSpace(s)
			// A Name long enough to be a sentence is prose, not a handle.
			if len(s) < minSurfaceLen || len(s) > 60 || seen[s] {
				continue
			}
			seen[s] = true
			alts = append(alts, regexp.QuoteMeta(s))
		}
		if len(alts) == 0 {
			continue
		}
		re, err := regexp.Compile(`\b(?:` + strings.Join(alts, "|") + `)\b`)
		if err != nil {
			continue
		}
		out = append(out, mentionMatcher{varName: e.VarName, re: re})
	}
	return out
}
