package main

import (
	"fmt"
	"sort"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// priorClaimsOfPredicate returns the pairs already linked by this predicate,
// as "A ~ B" lines, for the critic to check a candidate against.
//
// The critic judges one connection at a time, which is why it cannot tell that
// a candidate is the twelfth restatement of a claim the corpus already holds.
// Reading 48 stranded trip generations by hand found four shapes, not 48
// insights: eighteen said "a system claiming completeness meets what it cannot
// reach" with Godel or Hilbert on one side of every one. Each was defensible
// alone, which is the only way the critic ever saw them.
//
// The corpus has a structural-dedup lint rule, but it runs after promotion and
// compares entity neighbourhoods. This runs before, and compares a candidate
// against the claims of its own predicate — the axis restatement travels on.
func priorClaimsOfPredicate(dir, predicate string) []string {
	_, claims, err := corpusparse.ParseCorpus(dir)
	if err != nil {
		// The critic is best-effort everywhere else too; a parse failure
		// should cost novelty checking, not the whole gate.
		return nil
	}
	var out []string
	for _, c := range claims {
		if c.PredicateType != predicate || c.SubjectVar == "" || c.ObjectVar == "" {
			continue
		}
		out = append(out, fmt.Sprintf("%s ~ %s", c.SubjectVar, c.ObjectVar))
	}
	sort.Strings(out)
	return out
}
