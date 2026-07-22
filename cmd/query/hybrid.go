package main

// Hybrid retrieval: fuse the lexical (BM25) and semantic (embedding) rankings
// into one list with Reciprocal Rank Fusion. RRF combines by RANK, not score —
// necessary because BM25 (unbounded, ~0..12 here) and cosine (0..1) live on
// incomparable scales, so summing raw scores lets one signal dominate
// arbitrarily. Each list contributes 1/(k+rank) per document; k=60 is the
// standard damping constant (Cormack et al. 2009). A document that both signals
// rank highly rises to the top; one that only one signal finds still surfaces.

import (
	"fmt"
	"os"
	"sort"
)

const rrfK = 60

type fusedHit struct {
	idx      int
	rrf      float64
	lex, sem int // 1-based rank in each list; 0 = absent from that list
}

// rrfFuse combines two entity rankings (map: entity index -> 1-based rank) into
// a fused list sorted by descending RRF score. Pure and deterministic — the
// testable core of runHybrid, independent of BM25/embeddings.
func rrfFuse(lexRank, semRank map[int]int) []fusedHit {
	acc := map[int]*fusedHit{}
	get := func(idx int) *fusedHit {
		f, ok := acc[idx]
		if !ok {
			f = &fusedHit{idx: idx}
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
	out := make([]fusedHit, 0, len(acc))
	for _, f := range acc {
		out = append(out, *f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].rrf != out[j].rrf {
			return out[i].rrf > out[j].rrf
		}
		return out[i].idx < out[j].idx // stable tiebreak
	})
	return out
}

func runHybrid(kb *kbIndex, query, dir string, jsonOut bool) {
	// Lexical ranking over entities. buildFTIndex also ranks provenance, but the
	// semantic side is entity-only, so fuse over the entity universe: take the
	// entity hits in BM25 order and rank them 1..N.
	lexRank := map[int]int{}
	r := 0
	for _, h := range buildFTIndex(kb).search(query, 0) {
		if h.kind != "entity" {
			continue
		}
		if _, seen := lexRank[h.ref]; !seen {
			r++
			lexRank[h.ref] = r
		}
	}

	semHits, err := semanticRank(kb, query, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hybrid: %v\n", err)
		os.Exit(1)
	}
	semRank := make(map[int]int, len(semHits))
	for i, h := range semHits {
		semRank[h.idx] = i + 1
	}

	fused := rrfFuse(lexRank, semRank)
	if len(fused) > 15 {
		fused = fused[:15]
	}

	if jsonOut {
		recs := make([]map[string]any, 0, len(fused))
		for _, f := range fused {
			e := kb.Entities[f.idx]
			recs = append(recs, map[string]any{
				"var_name": e.VarName, "name": e.Name, "rrf": f.rrf,
				"lex_rank": f.lex, "sem_rank": f.sem, "brief": e.Brief, "file": e.File,
			})
		}
		printJSON(map[string]any{"query": query, "count": len(fused), "hits": recs})
		return
	}

	if len(fused) == 0 {
		fmt.Printf("No hybrid matches for %q\n", query)
		return
	}
	fmt.Printf("Hybrid (BM25 + %s, RRF) matches for %q:\n\n", embedModel, query)
	for _, f := range fused {
		e := kb.Entities[f.idx]
		fmt.Printf("  [%.4f] %s (%s)  [lex %s · sem %s]\n", f.rrf, e.Name, e.VarName, rankStr(f.lex), rankStr(f.sem))
		if e.Brief != "" {
			fmt.Printf("        %s\n", truncate(e.Brief, 200))
		}
	}
}

func rankStr(r int) string {
	if r == 0 {
		return "—"
	}
	return fmt.Sprintf("#%d", r)
}
