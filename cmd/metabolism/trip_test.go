package main

import (
	"testing"
)

func TestPickCrossClusterPairs(t *testing.T) {
	t.Run("two clusters", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "entity a1"},
			{name: "A2", cluster: 0, brief: "entity a2"},
			{name: "B1", cluster: 1, brief: "entity b1"},
			{name: "B2", cluster: 1, brief: "entity b2"},
		}
		pairs := pickCrossClusterPairs(entities, 2)
		if len(pairs) == 0 {
			t.Fatal("expected at least 1 cross-cluster pair")
		}
		if len(pairs) > 2 {
			t.Errorf("requested 2 pairs, got %d", len(pairs))
		}
		for _, p := range pairs {
			if p.A.cluster == p.B.cluster {
				t.Errorf("pair %s-%s are in same cluster %d", p.A.name, p.B.name, p.A.cluster)
			}
		}
	})

	t.Run("single cluster returns nil", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "a1"},
			{name: "A2", cluster: 0, brief: "a2"},
		}
		pairs := pickCrossClusterPairs(entities, 5)
		if pairs != nil {
			t.Errorf("expected nil for single cluster, got %d pairs", len(pairs))
		}
	})

	t.Run("prefer entities with briefs", func(t *testing.T) {
		entities := []tripEntity{
			{name: "Blank1", cluster: 0},
			{name: "Rich1", cluster: 0, brief: "has a brief"},
			{name: "Blank2", cluster: 1},
			{name: "Rich2", cluster: 1, brief: "also has a brief"},
		}
		pairs := pickCrossClusterPairs(entities, 1)
		if len(pairs) == 0 {
			t.Fatal("expected at least 1 pair")
		}
		// First pair should use the entities with briefs (score 2 > score 0)
		p := pairs[0]
		if p.A.brief == "" || p.B.brief == "" {
			t.Errorf("expected pair with briefs, got %q + %q", p.A.brief, p.B.brief)
		}
	})

	t.Run("negative cluster excluded", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "a1"},
			{name: "Orphan", cluster: -1, brief: "no cluster"},
			{name: "B1", cluster: 1, brief: "b1"},
		}
		pairs := pickCrossClusterPairs(entities, 5)
		for _, p := range pairs {
			if p.A.name == "Orphan" || p.B.name == "Orphan" {
				t.Error("orphan entity (cluster -1) should not appear in pairs")
			}
		}
	})
}

func TestValidatePredicate(t *testing.T) {
	cases := []struct {
		pred     string
		subjRole string
		objRole  string
		want     bool
	}{
		{"TheoryOf", "Hypothesis", "Concept", true},
		{"TheoryOf", "Concept", "Hypothesis", false}, // reversed
		{"Proposes", "Person", "Hypothesis", true},
		{"Proposes", "Hypothesis", "Person", false},
		{"InfluencedBy", "Person", "Person", true},
		{"InfluencedBy", "Person", "Concept", false},
		{"BelongsTo", "Concept", "Concept", true},
		{"BelongsTo", "Hypothesis", "Concept", false},
		{"BogusPredicate", "Person", "Person", false}, // unknown
	}
	for _, tc := range cases {
		t.Run(tc.pred+"/"+tc.subjRole+"->"+tc.objRole, func(t *testing.T) {
			got := validatePredicate(tc.pred, tc.subjRole, tc.objRole)
			if got != tc.want {
				t.Errorf("validatePredicate(%q, %q, %q) = %v, want %v",
					tc.pred, tc.subjRole, tc.objRole, got, tc.want)
			}
		})
	}
}

func TestCompatiblePredicates(t *testing.T) {
	cases := []struct {
		roleA string
		roleB string
		want  []string
	}{
		{"Person", "Person", []string{"InfluencedBy"}},
		{"Person", "Hypothesis", []string{"Accepts", "Disputes", "Proposes"}},
		{"Hypothesis", "Person", []string{"Accepts", "Disputes", "Proposes"}}, // symmetric
		{"Hypothesis", "Concept", []string{"TheoryOf"}},
		{"Concept", "Concept", []string{"BelongsTo", "CommentaryOn", "DerivedFrom"}},
		{"Place", "Person", []string{}}, // no compatible predicate
	}
	for _, tc := range cases {
		t.Run(tc.roleA+"-"+tc.roleB, func(t *testing.T) {
			got := compatiblePredicates(tc.roleA, tc.roleB)
			if len(got) != len(tc.want) {
				t.Errorf("compatiblePredicates(%q, %q) = %v, want %v", tc.roleA, tc.roleB, got, tc.want)
				return
			}
			for i, p := range tc.want {
				if got[i] != p {
					t.Errorf("compatiblePredicates(%q, %q)[%d] = %q, want %q",
						tc.roleA, tc.roleB, i, got[i], p)
				}
			}
		})
	}
}

func TestDrugProfile(t *testing.T) {
	cases := []struct {
		temp       float64
		promptType string
		want       string
	}{
		{1.3, "analogy", "psychedelic/pattern-matching"},
		{1.2, "contradiction", "psychedelic/adversarial"},
		{0.8, "genealogy", "exploratory/causal-tracing"},
		{1.0, "prediction", "exploratory/forecasting"},
		{0.4, "analogy", "microdose/pattern-matching"},
		{0.7, "contradiction", "microdose/adversarial"},
		{0.2, "genealogy", "sober/causal-tracing"},
		{0.0, "prediction", "sober/forecasting"},
	}
	for _, tc := range cases {
		got := drugProfile(tc.temp, tc.promptType)
		if got != tc.want {
			t.Errorf("drugProfile(%.1f, %q) = %q, want %q", tc.temp, tc.promptType, got, tc.want)
		}
	}
}
