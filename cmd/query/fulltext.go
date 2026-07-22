package main

// In-memory BM25 fulltext search over the corpus prose (entity Briefs +
// provenance Quotes). No persistent index, no external engine: at winze's
// scale (hundreds of entities) the whole corpus parses in ~30ms and ranking
// is sub-millisecond, so the index is rebuilt per invocation from the same
// parsed kbIndex every other query mode uses. This is the "lightweight,
// rebuildable, no heavy store" shape the corpus's size makes optimal —
// substring search (runSearch) finds a literal; this ranks by relevance over
// the actual prose.

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// tokenize lowercases and splits on any non-alphanumeric rune, dropping
// single-character tokens. Common words need no stoplist — BM25's IDF term
// down-weights anything that appears in most documents.
func tokenize(s string) []string {
	raw := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := raw[:0]
	for _, t := range raw {
		if len(t) >= 2 {
			out = append(out, t)
		}
	}
	return out
}

type ftDoc struct {
	kind  string // "entity" or "provenance"
	ref   int    // index into kb.Entities or kb.Provenance
	terms map[string]int
	len   int
}

type ftIndex struct {
	docs   []ftDoc
	df     map[string]int // term -> number of docs containing it
	n      int
	avgLen float64
}

// buildFTIndex indexes each entity (Name + Brief + Aliases) and each
// provenance record (Origin + Quote) as a document.
func buildFTIndex(kb *kbIndex) *ftIndex {
	fi := &ftIndex{df: map[string]int{}}
	total := 0

	add := func(kind string, ref int, text string) {
		toks := tokenize(text)
		if len(toks) == 0 {
			return
		}
		tf := make(map[string]int, len(toks))
		for _, t := range toks {
			tf[t]++
		}
		fi.docs = append(fi.docs, ftDoc{kind: kind, ref: ref, terms: tf, len: len(toks)})
		for t := range tf {
			fi.df[t]++
		}
		total += len(toks)
	}

	for i, e := range kb.Entities {
		add("entity", i, e.Name+" "+e.Brief+" "+strings.Join(e.Aliases, " "))
	}
	for i, p := range kb.Provenance {
		add("provenance", i, p.Origin+" "+p.Quote)
	}

	fi.n = len(fi.docs)
	if fi.n > 0 {
		fi.avgLen = float64(total) / float64(fi.n)
	}
	return fi
}

// BM25 parameters (Robertson/Sparck-Jones defaults).
const (
	bm25K1 = 1.5
	bm25B  = 0.75
)

type ftHit struct {
	kind  string
	ref   int
	score float64
}

// search ranks documents against the query with BM25 and returns the top
// `limit` hits (all hits when limit <= 0), highest score first.
func (fi *ftIndex) search(query string, limit int) []ftHit {
	qTerms := tokenize(query)
	if len(qTerms) == 0 || fi.n == 0 {
		return nil
	}
	var hits []ftHit
	for _, d := range fi.docs {
		var score float64
		for _, qt := range qTerms {
			tf, ok := d.terms[qt]
			if !ok {
				continue
			}
			df := float64(fi.df[qt])
			idf := math.Log(1 + (float64(fi.n)-df+0.5)/(df+0.5))
			norm := bm25K1 * (1 - bm25B + bm25B*float64(d.len)/fi.avgLen)
			score += idf * (float64(tf) * (bm25K1 + 1)) / (float64(tf) + norm)
		}
		if score > 0 {
			hits = append(hits, ftHit{kind: d.kind, ref: d.ref, score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}
