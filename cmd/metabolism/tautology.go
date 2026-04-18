package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// tautology — detection for the Wikipedia-over-Wikipedia problem.
// When the sensor hits articles that the KB was itself ingested from,
// a "corroborated" verdict is tautological: the corpus already contains
// the source's claims, so finding the source again provides no new
// epistemic signal. This split is the gap_confirmed vs no_gap
// distinction called out as a known problem in README.
//
// Computed post-hoc at calibrate time from the current corpus state —
// novelty is a moving target as ingest grows, so no schema field is
// persisted on Cycle. Recomputing each run keeps the stat honest.

// corpusProvenanceIndex holds normalized slugs of all Provenance.Origin
// values present in the corpus. A sensor-found paper whose normalized
// title or ID matches one of these slugs is considered "already in
// corpus" and any corroborated verdict referencing it is tautological.
type corpusProvenanceIndex struct {
	slugs map[string]bool
}

func collectCorpusProvenance(dir string) (*corpusProvenanceIndex, error) {
	idx := &corpusProvenanceIndex{slugs: map[string]bool{}}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			ident, ok := cl.Type.(*ast.Ident)
			if !ok || ident.Name != "Provenance" {
				return true
			}
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "Origin" {
					continue
				}
				bl, ok := kv.Value.(*ast.BasicLit)
				if !ok {
					continue
				}
				origin := strings.Trim(bl.Value, "\"")
				for _, slug := range originSlugs(origin) {
					idx.slugs[slug] = true
				}
			}
			return true
		})
	}
	return idx, nil
}

// originSlugs extracts matchable slugs from a Provenance.Origin string.
// Origin formats seen in the corpus:
//
//	"Wikipedia (zim 2025-12) / Chinese_room"
//	"PubMed 23663408 / Clark, Andy (2013). 'Whatever next? ...'"
//	"arXiv:2402.12345 / Paper Title"
//	"Frontiers in Neuroscience / Mattson, Mark P. (2014). '...'"
//
// We emit one or more normalized slugs per origin so a sensor-found
// paper whose title or ID matches any of them counts as overlap.
func originSlugs(origin string) []string {
	var out []string
	// Everything before the first " / " is the source; after is the title/id.
	parts := strings.SplitN(origin, " / ", 2)
	if len(parts) == 2 {
		out = append(out, normalizeSlug(parts[1]))
	}
	// The origin itself may embed identifiers (arXiv ID, DOI, PubMed ID).
	for _, tok := range strings.Fields(origin) {
		tok = strings.Trim(tok, ",.;:()[]{}'\"")
		if looksLikeIdentifier(tok) {
			out = append(out, normalizeSlug(tok))
		}
	}
	return out
}

// looksLikeIdentifier catches strings shaped like arXiv IDs, DOIs, or
// PubMed numeric IDs. Heuristic, not exhaustive — the goal is enough
// coverage that arXiv/PubMed-origin papers match their own slug.
func looksLikeIdentifier(s string) bool {
	if len(s) < 4 {
		return false
	}
	if strings.HasPrefix(strings.ToLower(s), "arxiv:") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(s), "doi:") || strings.HasPrefix(s, "10.") {
		return true
	}
	digits := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	// 7+ consecutive-ish digits suggests PubMed or numeric ID
	return digits >= 7 && digits >= len(s)-2
}

// normalizeSlug lower-cases, collapses separators to '-', strips punctuation.
// Intent: "Chinese_room" / "Chinese room" / "chinese-room" all collide.
func normalizeSlug(s string) string {
	var b strings.Builder
	prevSep := true
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevSep = false
		default:
			if !prevSep {
				b.WriteByte('-')
				prevSep = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// paperIsNovel returns true when none of the paper's identifiers match
// any origin slug in the corpus — i.e., this paper is not already
// ingested. A false return means the paper is already represented in
// the KB's provenance and any corroboration it produces is tautological.
func (idx *corpusProvenanceIndex) paperIsNovel(p PaperSummary) bool {
	if idx == nil {
		return true
	}
	candidates := []string{normalizeSlug(p.Title), normalizeSlug(p.ID)}
	for _, c := range candidates {
		if c != "" && idx.slugs[c] {
			return false
		}
	}
	return true
}

// countNovelPapers returns how many of a cycle's found papers are not
// already in the KB's provenance. A corroborated cycle with zero novel
// papers is tautological (no_gap); a corroborated cycle with at least
// one novel paper is gap_confirmed.
func (idx *corpusProvenanceIndex) countNovelPapers(papers []PaperSummary) int {
	n := 0
	for _, p := range papers {
		if idx.paperIsNovel(p) {
			n++
		}
	}
	return n
}

// classifyGapStatus returns one of:
//
//	"gap_confirmed"  — every found paper is not in the corpus (fully novel)
//	"mixed_overlap"  — some papers novel, some already in the corpus
//	"no_gap"         — every found paper is already in the corpus (tautological)
//	"no_signal"      — sensor found nothing (PapersFound == 0)
//	""               — not applicable (cycle has no papers attached; KB-internal type)
func classifyGapStatus(c Cycle, idx *corpusProvenanceIndex) string {
	if c.PredictionType != "" && c.PredictionType != "structural_fragility" {
		// KB-internal prediction types don't use sensor papers.
		return ""
	}
	if c.PapersFound == 0 {
		return "no_signal"
	}
	novel := idx.countNovelPapers(c.Papers)
	total := len(c.Papers)
	switch {
	case novel == 0:
		return "no_gap"
	case novel == total:
		return "gap_confirmed"
	default:
		return "mixed_overlap"
	}
}
