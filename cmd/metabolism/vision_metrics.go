package main

import (
	"path/filepath"

	"github.com/justinstimatze/winze/internal/astutil"
)

// computeVisionMetrics scans the corpus directly rather than through
// topology's SensorTargets — that list is already ranked and cut to the top
// 5 candidates for querying, not a census of every contested concept in the
// KB, so it cannot answer "how many are there" or "did the ones we have get
// deeper."
//
// AST-only, no go/types, matching collectClaims's own reasoning (cmd/lint):
// a claim literal is identified by having both Subject and Object keyed
// fields, which is enough to find every predicate instantiation without the
// cost of type-checking the whole corpus.
func computeVisionMetrics(dir string) (visionMetrics, error) {
	files, _, err := astutil.ParseCorpus(dir)
	if err != nil {
		return visionMetrics{}, err
	}

	type claim struct{ subj, obj string }
	var claims []claim
	theoryOf := map[string]map[string]bool{} // concept (Object) -> set of theory Subjects
	disputes := 0

	astutil.WalkVarDecls(files, func(v astutil.VarDecl) {
		// astutil.ParseCorpus keys files by the path it was given joined
		// with dir, not a bare filename — a raw v.File == "predictions.go"
		// check only matches when dir is exactly "." (the documented
		// invocation, run from inside corpus/). Any other dir shape, an
		// absolute path or a relative one like "corpus" from the repo root,
		// makes it silently stop excluding reify's bookkeeping. Comparing
		// the basename is correct under all of them.
		if filepath.Base(v.File) == "predictions.go" {
			return
		}
		subj, obj := astutil.ExtractSubjectObject(v.Lit)
		if subj == "" && obj == "" {
			return // not a claim literal — an entity, provenance, or outcome
		}
		claims = append(claims, claim{subj, obj})
		switch v.TypeName {
		case "TheoryOf":
			if theoryOf[obj] == nil {
				theoryOf[obj] = map[string]bool{}
			}
			theoryOf[obj][subj] = true
		case "Disputes", "DisputesOrg":
			disputes++
		}
	})

	vm := visionMetrics{TotalClaims: len(claims), Disputes: disputes}

	// thinNeighborhood collects every entity var name belonging to a
	// thin-contested concept's neighborhood: the concept itself and its two
	// theories. A claim mentioning any of them, as either Subject or Object,
	// counts toward ThinContestedClaims — that is what "did this neighborhood
	// get deeper" means operationally: is anything (a commentary, a dispute,
	// an influence claim, a third source) accreting around it.
	thinNeighborhood := map[string]bool{}
	for concept, subjects := range theoryOf {
		if len(subjects) >= 2 {
			vm.ContestedConcepts++
		}
		if len(subjects) == 2 {
			vm.ThinContestedConcepts++
			thinNeighborhood[concept] = true
			for s := range subjects {
				thinNeighborhood[s] = true
			}
		}
	}
	if len(thinNeighborhood) > 0 {
		for _, c := range claims {
			if thinNeighborhood[c.subj] || thinNeighborhood[c.obj] {
				vm.ThinContestedClaims++
			}
		}
	}

	return vm, nil
}

// visionMetrics counts the things the README's own criterion for a
// productive metabolism is actually about — "better answers the next time
// someone asks" — rather than the loop's activity. survivorship_ratio and
// useful_signal_pct (see CalibrationRow) measure the sensor's retrieval
// behavior; neither says whether the corpus itself is getting deeper.
//
// Counting definitions match coreStats (cmd/mcp) exactly: contested concepts
// are TheoryOf claims grouped by Object with 2+ distinct Subjects; disputes
// are Disputes/DisputesOrg claims. So this number and `winze-query --stats`'s
// "N contested concepts" / "N disputes" are the same number, not two
// competing ones.
//
// Deliberately excludes predictions.go — reify's own bookkeeping is not
// knowledge (see isSubstantiveResolution in reify.go), and counting it here
// would silently reintroduce the activity-vs-findings conflation item 3 of
// the metabolism plan just removed from the corpus itself.
type visionMetrics struct {
	TotalClaims           int
	ContestedConcepts     int
	Disputes              int
	ThinContestedConcepts int // exactly 2 theories — the depth-first frontier
	ThinContestedClaims   int // claims touching a thin-contested concept or either of its two theories
}
