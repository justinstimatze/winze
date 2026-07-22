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
	"strings"

	"github.com/justinstimatze/winze/internal/cliutil"
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

// runHybrid fuses BM25 + semantic rankings, then applies the VERIFIED type
// signal: --type filters results to a role with zero classification error
// (the role compiled), and --expand returns each hit's typed claim
// neighborhood so downstream reasoning gets the relationships, not just a prose
// snippet. This is the "typed for retrieval, from the same source of truth as
// typed for compilation" half — the type is a retrieval signal for free.
func runHybrid(kb *kbIndex, query, dir, typeFilter string, expand, jsonOut bool) {
	canonRole := ""
	if typeFilter != "" {
		var ok bool
		canonRole, ok = canonicalRole(kb, typeFilter)
		if !ok {
			fmt.Fprintf(os.Stderr, "hybrid: unknown --type %q; known roles: %s\n", typeFilter, strings.Join(sortedRoles(kb), ", "))
			os.Exit(2)
		}
	}

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

	// Verified-type filter: applied to the RRF-sorted list before the top-N cut,
	// so we keep the 15 best-ranked entities OF THE REQUESTED ROLE. The role was
	// type-checked at build time, so this is an exact filter — no misclassified
	// tag can leak the wrong kind in or a right one out.
	if canonRole != "" {
		kept := fused[:0]
		for _, f := range fused {
			if kb.Entities[f.idx].RoleType == canonRole {
				kept = append(kept, f)
			}
		}
		fused = kept
	}
	if len(fused) > 15 {
		fused = fused[:15]
	}

	if jsonOut {
		recs := make([]map[string]any, 0, len(fused))
		for _, f := range fused {
			e := kb.Entities[f.idx]
			rec := map[string]any{
				"var_name": e.VarName, "name": e.Name, "role_type": e.RoleType, "rrf": f.rrf,
				"lex_rank": f.lex, "sem_rank": f.sem, "brief": e.Brief, "file": e.File,
			}
			if expand {
				rec["neighborhood"] = neighborhood(kb, e.VarName)
			}
			recs = append(recs, rec)
		}
		out := map[string]any{"query": query, "count": len(fused), "hits": recs}
		if canonRole != "" {
			out["type"] = canonRole
		}
		printJSON(out)
		return
	}

	if len(fused) == 0 {
		if canonRole != "" {
			fmt.Printf("No hybrid matches for %q of type %s\n", query, canonRole)
		} else {
			fmt.Printf("No hybrid matches for %q\n", query)
		}
		return
	}
	scope := ""
	if canonRole != "" {
		scope = ", type=" + canonRole
	}
	fmt.Printf("Hybrid (BM25 + %s, RRF%s) matches for %q:\n\n", embedModel, scope, query)
	for _, f := range fused {
		e := kb.Entities[f.idx]
		fmt.Printf("  [%.4f] %s (%s · %s)  [lex %s · sem %s]\n", f.rrf, e.Name, e.VarName, e.RoleType, rankStr(f.lex), rankStr(f.sem))
		if e.Brief != "" {
			fmt.Printf("        %s\n", cliutil.Truncate(e.Brief, 200))
		}
		if expand {
			for _, edge := range neighborhood(kb, e.VarName) {
				fmt.Printf("        ↳ %s\n", edge["label"])
			}
		}
	}
}

// neighborhood returns an entity's typed claim edges: for each claim touching
// varName, the predicate, the neighbor at the other end, and the neighbor's
// (verified) role. Unary claims report the predicate on the entity itself. This
// is the reasoning-ready context — relationships, not prose.
func neighborhood(kb *kbIndex, varName string) []map[string]any {
	roleOf := func(v string) string {
		for i := range kb.Entities {
			if kb.Entities[i].VarName == v {
				return kb.Entities[i].RoleType
			}
		}
		return ""
	}
	var out []map[string]any
	for _, c := range claimsInvolving(kb, varName) {
		if c.Object == "" { // unary
			out = append(out, map[string]any{
				"predicate": c.Predicate, "label": fmt.Sprintf("%s (unary)", c.Predicate),
			})
			continue
		}
		neighbor := c.Object
		dir := "→"
		if c.Object == varName { // varName is the object; the other end is the subject
			neighbor, dir = c.Subject, "←"
		}
		out = append(out, map[string]any{
			"predicate": c.Predicate, "neighbor": neighbor, "neighbor_role": roleOf(neighbor),
			"label": fmt.Sprintf("%s %s %s (%s)", c.Predicate, dir, neighbor, roleOf(neighbor)),
		})
	}
	return out
}

// canonicalRole resolves a user-supplied --type (case-insensitively) to the
// exact RoleType string used in the index, so `--type hypothesis` matches
// `Hypothesis`.
func canonicalRole(kb *kbIndex, want string) (string, bool) {
	for role := range kb.RoleTypes {
		if strings.EqualFold(role, want) {
			return role, true
		}
	}
	return "", false
}

func sortedRoles(kb *kbIndex) []string {
	roles := make([]string, 0, len(kb.RoleTypes))
	for role := range kb.RoleTypes {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func rankStr(r int) string {
	if r == 0 {
		return "—"
	}
	return fmt.Sprintf("#%d", r)
}
