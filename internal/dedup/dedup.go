// Package dedup finds probable duplicate entities by STRUCTURE rather than by
// prose. An entity's meaning in winze lives in its claim edges, not its Brief
// (two entities that are the same concept, coined in different sessions, may
// have Briefs written nothing alike). So the duplicate signal is a shared
// neighborhood: two entities connected to the same neighbors by the same
// predicates occupy the same position in the graph and are probably the same
// thing. This is the calque-faithful check — index the contract (the edges),
// not the representation (the prose).
//
// The build gate cannot catch this: two differently-named entities of the same
// type both type-check. Structural dedup is the surfacer for the defect class
// the load-bearing gate is blind to.
package dedup

import (
	"math"
	"sort"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// Edge is one concrete relation on an entity's neighborhood: a predicate to or
// from a specific named neighbor. Concrete neighbors (not just neighbor roles)
// are the point — two Persons who each "Proposes → Hypothesis" are not
// duplicates, but two who "Proposes → the SAME hypothesis" very likely are.
type Edge struct {
	Predicate string
	Neighbor  string // the other entity's var name; "" for a unary claim
	Dir       string // "out" (this entity is Subject), "in" (Object), "unary"
}

// Signatures maps each entity var name to its neighborhood edge set.
func Signatures(claims []corpusparse.Claim) map[string]map[Edge]int {
	sig := map[string]map[Edge]int{}
	add := func(v string, e Edge) {
		if sig[v] == nil {
			sig[v] = map[Edge]int{}
		}
		sig[v][e]++
	}
	for _, c := range claims {
		if c.SubjectVar == "" {
			continue
		}
		if c.ObjectVar == "" {
			add(c.SubjectVar, Edge{c.PredicateType, "", "unary"})
			continue
		}
		add(c.SubjectVar, Edge{c.PredicateType, c.ObjectVar, "out"})
		add(c.ObjectVar, Edge{c.PredicateType, c.SubjectVar, "in"})
	}
	return sig
}

// idf weights each edge by rarity: an edge shared by many entities (a category
// tag like IsCognitiveBias, or a taxonomic parent every sibling has) is weak
// evidence of duplication; an edge to a specific rare neighbor is strong. This
// is what separates duplicates from siblings — without it, everything in a
// taxonomy looks alike. df is the number of entities carrying the edge; idf =
// ln(1 + N/df).
func idf(sig map[string]map[Edge]int) map[Edge]float64 {
	df := map[Edge]int{}
	for _, s := range sig {
		for e := range s {
			df[e]++
		}
	}
	n := float64(len(sig))
	w := make(map[Edge]float64, len(df))
	for e, d := range df {
		w[e] = math.Log(1 + n/float64(d))
	}
	return w
}

// Overlap returns the shared-edge count and two coefficients:
//   - contain = shared / size of the SMALLER neighborhood. Rewards containment:
//     a thin, just-coined entity whose few edges all reappear on an established
//     one scores ~1.0. This is the coin-time signal (MatchesFor).
//   - sym = shared / size of the LARGER neighborhood. High only when BOTH
//     neighborhoods are near-identical — the genuine "same thing" signal that
//     distinguishes a duplicate from a sibling that merely shares a category.
//     This is the all-pairs audit signal (Candidates).
//
// A bias sharing 2 of {4, 5} edges with a sibling has contain=0.5 but sym=0.4;
// two entities with truly identical neighborhoods have both ~1.0.
func Overlap(a, b map[Edge]int) (shared int, contain, sym float64) {
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	for e := range small {
		if _, ok := large[e]; ok {
			shared++
		}
	}
	if len(small) == 0 || len(large) == 0 {
		return 0, 0, 0
	}
	return shared, float64(shared) / float64(len(small)), float64(shared) / float64(len(large))
}

// weightedShared sums the idf of the edges a and b share — the mass of rare
// structure they hold in common. Two siblings sharing only common category
// edges score low; two duplicates sharing specific neighbors score high.
func weightedShared(a, b map[Edge]int, w map[Edge]float64) float64 {
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	var s float64
	for e := range small {
		if _, ok := large[e]; ok {
			s += w[e]
		}
	}
	return s
}

// Candidate is a probable-duplicate pair, most-suspicious first when sorted.
type Candidate struct {
	A, B   string
	Role   string
	Shared int
	Coeff  float64 // the coefficient the caller filtered on (sym for audits, contain for coin-time)
	Weight float64 // idf mass of shared edges — the rarity-aware strength
}

// Candidates scans all same-role entity pairs and returns those sharing at
// least minShared edges, an overlap coefficient of at least minCoeff, AND an
// idf mass of at least minWeight (the rarity filter that separates duplicates
// from taxonomic siblings). Same-role only: the type is verified, so entities
// of different types are never duplicates. Sorted most-suspicious first.
func Candidates(entities []corpusparse.Entity, claims []corpusparse.Claim, minShared int, minCoeff, minWeight float64) []Candidate {
	sig := Signatures(claims)
	w := idf(sig)
	role := map[string]string{}
	for _, e := range entities {
		role[e.VarName] = e.RoleType
	}
	var vars []string
	for _, e := range entities {
		if len(sig[e.VarName]) >= minShared {
			vars = append(vars, e.VarName)
		}
	}
	sort.Strings(vars)

	var out []Candidate
	for i := 0; i < len(vars); i++ {
		for j := i + 1; j < len(vars); j++ {
			a, b := vars[i], vars[j]
			if role[a] != role[b] {
				continue
			}
			shared, _, sym := Overlap(sig[a], sig[b])
			ws := weightedShared(sig[a], sig[b], w)
			// Symmetric coefficient: both neighborhoods must be near-identical,
			// so sibling clusters that only share a category tag fall out.
			if shared >= minShared && sym >= minCoeff && ws >= minWeight {
				out = append(out, Candidate{A: a, B: b, Role: role[a], Shared: shared, Coeff: sym, Weight: ws})
			}
		}
	}
	sortCandidates(out)
	return out
}

// MatchesFor returns the entities that structurally overlap a single target —
// the coin-time query. After a claim is added that touches `target`, this
// reports whether `target` now duplicates an existing same-role entity, so the
// author reuses the canonical one instead of growing a duplicate. minShared may
// be 1 here (coin-time neighborhoods are thin); the caller decides how loud to
// be.
func MatchesFor(target string, entities []corpusparse.Entity, claims []corpusparse.Claim, minShared int, minCoeff, minWeight float64) []Candidate {
	sig := Signatures(claims)
	w := idf(sig)
	ts, ok := sig[target]
	if !ok || len(ts) < minShared {
		return nil
	}
	role := map[string]string{}
	for _, e := range entities {
		role[e.VarName] = e.RoleType
	}
	var out []Candidate
	for _, e := range entities {
		if e.VarName == target || role[e.VarName] != role[target] {
			continue
		}
		shared, contain, _ := Overlap(ts, sig[e.VarName])
		ws := weightedShared(ts, sig[e.VarName], w)
		// Containment: the thin new target contained in an established entity is
		// the coin-time duplicate signal.
		if shared >= minShared && contain >= minCoeff && ws >= minWeight {
			out = append(out, Candidate{A: target, B: e.VarName, Role: role[target], Shared: shared, Coeff: contain, Weight: ws})
		}
	}
	sortCandidates(out)
	return out
}

func sortCandidates(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool {
		if c[i].Weight != c[j].Weight {
			return c[i].Weight > c[j].Weight // rarity mass is the real strength
		}
		if c[i].Coeff != c[j].Coeff {
			return c[i].Coeff > c[j].Coeff
		}
		if c[i].Shared != c[j].Shared {
			return c[i].Shared > c[j].Shared
		}
		return c[i].B < c[j].B
	})
}
