package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestComputeVisionMetrics pins the counting definitions against a fixture
// with one of each shape the metric has to tell apart: well-covered (3
// theories), thin-contested (exactly 2), uncontested (1), disputes, a claim
// touching the thin neighborhood, and a claim that doesn't. Fixture parsed
// by AST — no type-checking, so the literals need no supporting type defs
// (see TestGoalCoverageStops for the established pattern).
func TestComputeVisionMetrics(t *testing.T) {
	dir := t.TempDir()
	src := `package winze

// Well-covered: 3 distinct theories — contested, but not thin.
var WellCoveredA = TheoryOf{Subject: TheoristOne, Object: WellCoveredConcept}
var WellCoveredB = TheoryOf{Subject: TheoristTwo, Object: WellCoveredConcept}
var WellCoveredC = TheoryOf{Subject: TheoristThree, Object: WellCoveredConcept}

// Thin: exactly 2 distinct theories — contested AND thin, the depth-first frontier.
var ThinA = TheoryOf{Subject: ThinTheoryOne, Object: ThinConcept}
var ThinB = TheoryOf{Subject: ThinTheoryTwo, Object: ThinConcept}

// Uncontested: 1 theory — not counted as contested at all.
var Solo = TheoryOf{Subject: SoloTheorist, Object: UncontestedConcept}

var D1 = Disputes{Subject: ThinTheoryOne, Object: ThinTheoryTwo}
var D2 = DisputesOrg{Subject: SomeOrg, Object: OtherOrg}

// Touches the thin neighborhood via one of its two theories.
var Commentary = CommentaryOn{Subject: SomeCommentator, Object: ThinTheoryOne}

// Does not touch the thin neighborhood at all.
var Unrelated = InfluencedBy{Subject: SoloTheorist, Object: WellCoveredConcept}
`
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := computeVisionMetrics(dir)
	if err != nil {
		t.Fatalf("computeVisionMetrics: %v", err)
	}
	want := visionMetrics{
		TotalClaims:           10,
		ContestedConcepts:     2, // WellCoveredConcept (3) + ThinConcept (2); UncontestedConcept excluded (1)
		Disputes:              2, // D1 + D2
		ThinContestedConcepts: 1, // ThinConcept only — WellCoveredConcept has 3, not exactly 2
		ThinContestedClaims:   4, // ThinA, ThinB, D1, Commentary — Unrelated and the well-covered/solo claims don't touch the thin neighborhood
	}
	if got != want {
		t.Errorf("computeVisionMetrics(fixture) = %+v, want %+v", got, want)
	}
}

// TestComputeVisionMetricsExcludesReifyBookkeeping is the test for the
// specific reason this scan does not just reuse a generic claim-counter:
// predictions.go is reify's own record of the loop's search activity, not
// knowledge (see isSubstantiveResolution in reify.go), and counting it here
// would silently reintroduce the activity-vs-findings conflation item 3 of
// the metabolism plan removed from the corpus. The fixture below plants an
// extra TheoryOf claim in a file literally named predictions.go that would,
// if counted, push ThinConcept's theory count from 2 to 3 — flipping it out
// of the thin-contested bucket entirely and changing every metric this test
// checks. If the exclusion breaks, this test fails on all three, not just
// TotalClaims — the same result a byte-count check would miss.
func TestComputeVisionMetricsExcludesReifyBookkeeping(t *testing.T) {
	dir := t.TempDir()
	corpus := `package winze

var ThinA = TheoryOf{Subject: ThinTheoryOne, Object: ThinConcept}
var ThinB = TheoryOf{Subject: ThinTheoryTwo, Object: ThinConcept}
`
	reifyBookkeeping := `package winze

var ReifyBogus = TheoryOf{Subject: ReifyGhost, Object: ThinConcept}
var ReifyDispute = Disputes{Subject: ReifyGhost, Object: ThinTheoryOne}
`
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(corpus), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "predictions.go"), []byte(reifyBookkeeping), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := computeVisionMetrics(dir)
	if err != nil {
		t.Fatalf("computeVisionMetrics: %v", err)
	}
	if got.TotalClaims != 2 {
		t.Errorf("TotalClaims = %d, want 2 — predictions.go's 2 claims leaked in", got.TotalClaims)
	}
	if got.ThinContestedConcepts != 1 {
		t.Errorf("ThinContestedConcepts = %d, want 1 — predictions.go's third theory pushed ThinConcept out of the thin bucket", got.ThinContestedConcepts)
	}
	if got.Disputes != 0 {
		t.Errorf("Disputes = %d, want 0 — predictions.go's Disputes claim leaked in", got.Disputes)
	}
}
