package main

import (
	"strings"
	"testing"
)

func TestFindLintEvidence(t *testing.T) {
	sample := `[value-conflict] 4 functional predicates, 3 KnownDispute annotations, 1 unresolved conflicts
  unresolved:
    FormedAt (LakeCheko):
      TripCycle8LakeChekoFormedAtCrater        metabolism_cycle8.go:12   object=CraterFormation2007

[orphan-report] 256 entities declared, 304 referenced, 0 orphaned
`

	cases := []struct {
		name    string
		varName string
		wantSub string // expect substring in result, "" if none
	}{
		{
			name:    "flagged var found",
			varName: "TripCycle8LakeChekoFormedAtCrater",
			wantSub: "TripCycle8LakeChekoFormedAtCrater",
		},
		{
			name:    "unrelated var not found",
			varName: "TripCycle8SomethingElse",
			wantSub: "",
		},
		{
			name:    "partial match in unrelated line not returned as refutation",
			varName: "NonexistentVar",
			wantSub: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findLintEvidence(sample, tc.varName)
			if tc.wantSub == "" {
				if got != "" {
					t.Errorf("findLintEvidence(%q) = %q, want empty", tc.varName, got)
				}
				return
			}
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("findLintEvidence(%q) = %q, want to contain %q", tc.varName, got, tc.wantSub)
			}
		})
	}
}

func TestFindLintEvidenceTruncates(t *testing.T) {
	longLine := "MyVarName " + strings.Repeat("x", 500)
	got := findLintEvidence(longLine, "MyVarName")
	if len(got) > 243 { // 240 + "..."
		t.Errorf("expected truncation around 240 chars + ellipsis, got %d chars", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected trailing ellipsis on truncated line, got %q", got[len(got)-10:])
	}
}

// TestPredicateGuidance pins the predicate-semantics map that the
// contradiction prompt depends on. If a predicate's exclusivity story
// changes (e.g. a new //winze:functional predicate is added) this test
// fails — forcing the prompt-builder to stay in sync with predicates.go.
//
// Proposes/ProposesOrg deliberately have no guidance: the corpus uses
// them as multi-attribution (see tunguska.go) and predicates.go carries
// no //winze:single-originator pragma. An earlier version of this test
// pinned an "exclusive to one originator" rule; that rule did not match
// corpus practice and was dropped.
func TestPredicateGuidance(t *testing.T) {
	cases := []struct {
		predicate string
		wantSub   string // empty = expect no guidance
	}{
		{"Proposes", ""},
		{"ProposesOrg", ""},
		{"FormedAt", "//winze:functional"},
		{"EnergyEstimate", "//winze:functional"},
		{"ResolvedAs", "//winze:functional"},
		{"EnglishTranslationOf", "//winze:functional"},
		{"TheoryOf", "//winze:contested"},
		{"LocatedIn", "spatial containment"},
		{"Accepts", "directly contradicts"},
		{"Disputes", "directly contradicts"},
		{"InfluencedBy", ""}, // no guidance — generic rules apply
		{"WorksFor", ""},
	}
	for _, tc := range cases {
		got := predicateGuidance(tc.predicate)
		if tc.wantSub == "" {
			if got != "" {
				t.Errorf("predicateGuidance(%q) = %q, want empty", tc.predicate, got)
			}
			continue
		}
		if !strings.Contains(got, tc.wantSub) {
			t.Errorf("predicateGuidance(%q) = %q, want to contain %q", tc.predicate, got, tc.wantSub)
		}
	}
}
