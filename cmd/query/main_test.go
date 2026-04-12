package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

func TestBuildIndex(t *testing.T) {
	root := repoRoot(t)
	kb, err := buildIndex(root)
	if err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	if len(kb.Entities) < 200 {
		t.Errorf("entities = %d, expected at least 200 (corpus has ~233)", len(kb.Entities))
	}
	if len(kb.Claims) < 200 {
		t.Errorf("claims = %d, expected at least 200 (corpus has ~281)", len(kb.Claims))
	}
	if len(kb.Provenance) < 30 {
		t.Errorf("provenance = %d, expected at least 30 (corpus has ~50)", len(kb.Provenance))
	}
	t.Logf("buildIndex: %d entities, %d claims, %d provenance", len(kb.Entities), len(kb.Claims), len(kb.Provenance))

	// Find Chalmers — a known entity
	found := false
	for _, e := range kb.Entities {
		if e.VarName == "Chalmers" || e.VarName == "DavidChalmers" {
			found = true
			if e.RoleType != "Person" {
				t.Errorf("Chalmers role type = %q, want %q", e.RoleType, "Person")
			}
		}
	}
	if !found {
		t.Error("expected to find Chalmers entity in KB index")
	}

	t.Logf("buildIndex: %d entities, %d claims, %d provenance",
		len(kb.Entities), len(kb.Claims), len(kb.Provenance))
}

func TestMatchEntity(t *testing.T) {
	entity := entityRecord{
		VarName: "ChalmersHardProblemThesis",
		Name:    "Hard problem of consciousness",
		Brief:   "Why and how physical processes give rise to subjective experience.",
		Aliases: []string{"hard problem", "explanatory gap"},
	}

	cases := []struct {
		query string
		want  bool
	}{
		{"chalmers", true},            // VarName match (case-insensitive)
		{"hard problem", true},        // Name match
		{"subjective experience", true}, // Brief match
		{"explanatory gap", true},     // Alias match
		{"tennis", false},             // No match
		{"consciousness", true},       // Name match (query must be pre-lowered)
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := matchEntity(entity, tc.query)
			if got != tc.want {
				t.Errorf("matchEntity(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}
