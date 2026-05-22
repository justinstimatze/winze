package main

// This file is now a thin convenience layer over internal/corpusparse.
// The full AST walker, IsTripGenerated, and IsReifyMachinery live in
// the shared internal package since they have three consumers now
// (cmd/rot-probe, cmd/predicates-suggest, future cmd/lint migration).

import (
	"github.com/justinstimatze/winze/internal/corpusparse"
)

// Local aliases keep the rest of cmd/rot-probe terse and let us flip
// the underlying type without touching the LLM / sampling code.
type entity = corpusparse.Entity
type claim = corpusparse.Claim

// parseCorpus delegates to the shared parser. Kept as a function (not a
// re-export) so future cmd/rot-probe-specific filtering can wrap it.
func parseCorpus(dir string) ([]entity, []claim, error) {
	return corpusparse.ParseCorpus(dir)
}

// excludeReifyMachinery drops entities whose var names match
// metabolism-reify auto-generated families (TripLint*, TripBuild*,
// TripLLM*, TripFunctional*, EvidenceSearch*). These have templated
// Briefs and auto-promotion claim chains — there is no human-actionable
// rot signal to surface, so spending LLM budget on them is waste.
// Verified (wi-1lnt): three earlier probe runs sampled ~57% reify
// machinery before this filter existed.
func excludeReifyMachinery(ents []entity) []entity {
	out := ents[:0]
	for _, e := range ents {
		if corpusparse.IsReifyMachinery(e.VarName) {
			continue
		}
		out = append(out, e)
	}
	return out
}
