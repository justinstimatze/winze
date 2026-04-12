package main

import (
	"strings"
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

func TestParseTripResponse(t *testing.T) {
	cases := []struct {
		name     string
		response string
		wantConn string
		wantPred string
		wantSc   int
		wantRat  string
	}{
		{
			name: "well formed",
			response: `CONNECTION: Both deal with limits of knowledge
PREDICATE: TheoryOf
SCORE: 4
RATIONALE: Structural parallel between incompleteness and hard problem`,
			wantConn: "Both deal with limits of knowledge",
			wantPred: "TheoryOf",
			wantSc:   4,
			wantRat:  "Structural parallel",
		},
		{
			name: "missing score defaults to 1",
			response: `CONNECTION: Weak link
PREDICATE: NONE
RATIONALE: Not convincing`,
			wantConn: "Weak link",
			wantPred: "",
			wantSc:   1,
			wantRat:  "Not convincing",
		},
		{
			name:     "empty response",
			response: "",
			wantConn: "",
			wantSc:   1,
		},
		{
			name: "NONE predicate treated as empty",
			response: `CONNECTION: Something
PREDICATE: NONE
SCORE: 3`,
			wantPred: "",
			wantSc:   3,
		},
		{
			name: "score out of range clamped",
			response: `SCORE: 9`,
			wantSc: 5, // 9 > 5, clamped to max 5
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTripResponse(tc.response)
			if tc.wantConn != "" && got.Connection != tc.wantConn {
				t.Errorf("Connection = %q, want %q", got.Connection, tc.wantConn)
			}
			if got.Predicate != tc.wantPred {
				t.Errorf("Predicate = %q, want %q", got.Predicate, tc.wantPred)
			}
			if got.Score != tc.wantSc {
				t.Errorf("Score = %d, want %d", got.Score, tc.wantSc)
			}
			if tc.wantRat != "" && !strings.Contains(got.Rationale, tc.wantRat) {
				t.Errorf("Rationale = %q, want to contain %q", got.Rationale, tc.wantRat)
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
