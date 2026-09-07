package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/cliutil"
)

// runRawHybrid answers --raw: BM25 + semantic + temporal search over a
// store's raw.jsonl evidence log, fused with rrfFuseRaw (cmd/query/rawhybrid.go's
// own 3-channel fuser -- see its doc comment for why this doesn't reuse
// --hybrid's rrfFuse directly). Like runRawFulltext before it, this does not
// need the typed corpus index at all -- main dispatches --raw before
// buildIndex.
//
// Deliberately does not port --hybrid's --type filter, --expand
// neighborhood, or supersession downrank: none of those concepts exist for
// raw evidence -- no role type, no claim graph, no Supersedes claims on a
// log line.
func runRawHybrid(dir, query string, jsonOut bool) {
	docs, err := loadRawDocs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query --raw: %v\n", err)
		os.Exit(1)
	}

	lexRank := map[int]int{}
	r := 0
	for _, h := range buildRawFTIndex(docs).search(query, 0) {
		if _, seen := lexRank[h.ref]; !seen {
			r++
			lexRank[h.ref] = r
		}
	}

	semHits, err := semanticRankRaw(docs, query, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query --raw: %v\n", err)
		os.Exit(1)
	}
	semRank := make(map[int]int, len(semHits))
	for i, h := range semHits {
		semRank[h.idx] = i + 1
	}

	tmpRank := map[int]int{}
	if start, end, ok := parseTemporalRange(query, time.Now()); ok {
		tmpRank = temporalRank(docs, start, end)
	}

	fused := rrfFuseRaw(lexRank, semRank, tmpRank)
	if len(fused) > 15 {
		fused = fused[:15]
	}

	if jsonOut {
		out := make([]map[string]any, 0, len(fused))
		for _, f := range fused {
			d := docs[f.idx]
			out = append(out, map[string]any{
				"score":    f.rrf,
				"time":     d.Time,
				"tool":     d.Tool,
				"var":      d.Var,
				"note":     d.Note,
				"tmp_rank": f.tmp,
			})
		}
		printJSON(map[string]any{"query": query, "count": len(fused), "hits": out})
		return
	}

	if len(fused) == 0 {
		fmt.Printf("No raw-evidence matches for %q\n", query)
		return
	}
	fmt.Printf("Raw-evidence matches (BM25 + %s + temporal, RRF) for %q (%d):\n\n", embedModel, query, len(fused))
	for _, f := range fused {
		d := docs[f.idx]
		fmt.Printf("  [%.4f] %s  %s (%s)  [lex %s · sem %s · tmp %s]\n", f.rrf, d.Time, d.Tool, d.Var, rankStr(f.lex), rankStr(f.sem), rankStr(f.tmp))
		fmt.Printf("        %s\n", cliutil.Truncate(d.Note, 200))
	}
}

// semanticRankRaw embeds every raw-tier doc's Note (cached, incremental --
// the same content-addressed vecCache the typed corpus's semanticRank uses)
// plus the query, then returns docs ranked by cosine similarity, highest
// first. Mirrors semanticRank exactly; the only difference is the source of
// the text to embed.
func semanticRankRaw(docs []rawDoc, query, dir string) ([]semHit, error) {
	cache := loadVecCache(dir)

	type ev struct {
		idx  int
		segs [][]float32
	}
	var vecs []ev
	built, hit := 0, 0
	for i, d := range docs {
		text := strings.TrimSpace(d.Note)
		if text == "" {
			continue
		}
		segs, n, err := embedSegments(cache, text)
		if err != nil {
			return nil, err
		}
		built += n
		hit += len(segs) - n
		vecs = append(vecs, ev{i, segs})
	}
	cache.save()
	if built > 0 {
		fmt.Fprintf(os.Stderr, "embedded %d new segments, %d from cache\n", built, hit)
	}

	qv, err := embed(query)
	if err != nil {
		return nil, err
	}

	hits := make([]semHit, 0, len(vecs))
	for _, e := range vecs {
		hits = append(hits, semHit{e.idx, bestCosine(qv, e.segs)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	return hits, nil
}

// rawFusedHit is rrfFuse's fusedHit (cmd/query/hybrid.go) extended to a
// third channel. Kept separate rather than generalizing rrfFuse/fusedHit:
// those are already shipped and tested, and --hybrid's display/JSON code
// reads their named lex/sem fields directly -- making rrfFuse variadic
// would force touching that already-working path for zero behavior change
// on it.
type rawFusedHit struct {
	idx           int
	rrf           float64
	lex, sem, tmp int
}

// rrfFuseRaw combines three rankings (map: doc index -> 1-based rank) into
// a fused list sorted by descending RRF score. Same accumulate-then-sort
// shape as rrfFuse, one more channel.
func rrfFuseRaw(lexRank, semRank, tmpRank map[int]int) []rawFusedHit {
	acc := map[int]*rawFusedHit{}
	get := func(idx int) *rawFusedHit {
		f, ok := acc[idx]
		if !ok {
			f = &rawFusedHit{idx: idx}
			acc[idx] = f
		}
		return f
	}
	for idx, rank := range lexRank {
		f := get(idx)
		f.rrf += 1 / float64(rrfK+rank)
		f.lex = rank
	}
	for idx, rank := range semRank {
		f := get(idx)
		f.rrf += 1 / float64(rrfK+rank)
		f.sem = rank
	}
	for idx, rank := range tmpRank {
		f := get(idx)
		f.rrf += 1 / float64(rrfK+rank)
		f.tmp = rank
	}
	out := make([]rawFusedHit, 0, len(acc))
	for _, f := range acc {
		out = append(out, *f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].rrf != out[j].rrf {
			return out[i].rrf > out[j].rrf
		}
		return out[i].idx < out[j].idx
	})
	return out
}
