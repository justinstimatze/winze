package dedup

import (
	"testing"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// Two entities pointing at the SAME neighbors via the SAME predicates are
// structural duplicates; two with the same edge SHAPE but different neighbors
// are not.
func TestCandidatesSharedNeighborsAreDuplicates(t *testing.T) {
	ents := []corpusparse.Entity{
		{VarName: "ConfirmationBias", RoleType: "Concept"},
		{VarName: "SelectiveEvidence", RoleType: "Concept"},
		{VarName: "AnchoringBias", RoleType: "Concept"},
		{VarName: "Kahneman", RoleType: "Person"},
		{VarName: "TverskyPaper", RoleType: "CreativeWork"},
	}
	claims := []corpusparse.Claim{
		// ConfirmationBias and SelectiveEvidence share BOTH neighbors → duplicates.
		{PredicateType: "InfluencedBy", SubjectVar: "ConfirmationBias", ObjectVar: "Kahneman"},
		{PredicateType: "AppearsIn", SubjectVar: "ConfirmationBias", ObjectVar: "TverskyPaper"},
		{PredicateType: "InfluencedBy", SubjectVar: "SelectiveEvidence", ObjectVar: "Kahneman"},
		{PredicateType: "AppearsIn", SubjectVar: "SelectiveEvidence", ObjectVar: "TverskyPaper"},
		// AnchoringBias has the same edge SHAPE but a different neighbor → not a dup.
		{PredicateType: "InfluencedBy", SubjectVar: "AnchoringBias", ObjectVar: "Kahneman"},
	}
	got := Candidates(ents, claims, 2, 0.5, 0)
	if len(got) != 1 {
		t.Fatalf("want exactly 1 candidate pair, got %d: %+v", len(got), got)
	}
	c := got[0]
	if !((c.A == "ConfirmationBias" && c.B == "SelectiveEvidence") ||
		(c.A == "SelectiveEvidence" && c.B == "ConfirmationBias")) {
		t.Errorf("wrong pair: %+v", c)
	}
	if c.Shared != 2 || c.Coeff != 1.0 {
		t.Errorf("want shared=2 coeff=1.0, got shared=%d coeff=%v", c.Shared, c.Coeff)
	}
}

// A cross-role pair is never a candidate even with a shared neighbor: the type
// compiled, so a Person and a Concept are not the same thing.
func TestCandidatesSameRoleOnly(t *testing.T) {
	ents := []corpusparse.Entity{
		{VarName: "A", RoleType: "Concept"},
		{VarName: "B", RoleType: "Person"},
		{VarName: "N", RoleType: "Concept"},
	}
	claims := []corpusparse.Claim{
		{PredicateType: "InfluencedBy", SubjectVar: "A", ObjectVar: "N"},
		{PredicateType: "InfluencedBy", SubjectVar: "B", ObjectVar: "N"},
	}
	if got := Candidates(ents, claims, 1, 0.5, 0); len(got) != 0 {
		t.Errorf("cross-role pair should not be a candidate, got %+v", got)
	}
}

// MatchesFor is the coin-time query: a thin new entity whose single edge
// already exists on an established same-role entity surfaces at coefficient 1.0.
func TestMatchesForCoinTime(t *testing.T) {
	ents := []corpusparse.Entity{
		{VarName: "Established", RoleType: "Concept"},
		{VarName: "JustCoined", RoleType: "Concept"},
		{VarName: "Sartre", RoleType: "Person"},
	}
	claims := []corpusparse.Claim{
		{PredicateType: "InfluencedBy", SubjectVar: "Established", ObjectVar: "Sartre"},
		{PredicateType: "IsPolyvalentTerm", SubjectVar: "Established"},
		{PredicateType: "InfluencedBy", SubjectVar: "JustCoined", ObjectVar: "Sartre"},
	}
	got := MatchesFor("JustCoined", ents, claims, 1, 0.5, 0)
	if len(got) != 1 || got[0].B != "Established" {
		t.Fatalf("coin-time match wrong: %+v", got)
	}
	if got[0].Coeff != 1.0 { // JustCoined's one edge is fully contained in Established
		t.Errorf("want coeff 1.0 (containment), got %v", got[0].Coeff)
	}
}
