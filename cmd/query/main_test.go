package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/justinstimatze/winze/internal/defndb"
)

// skipIfNoDefnDB skips the test when defn isn't reachable. CI doesn't run a
// Dolt server and has no .defn/, so these integration tests can't index the
// corpus there. Skip rather than fail — they're real coverage locally.
func skipIfNoDefnDB(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if errors.Is(err, defndb.ErrNotAvailable) || strings.Contains(err.Error(), "defn not available") {
		t.Skipf("defn not available: %v", err)
	}
}

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
	skipIfNoDefnDB(t, err)
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

// TestLoadTripIsolatedConns covers the JSONL loader that surfaces the
// trip cycle's NONE-predicate dream-state into the --ask context.
// Without this, the metabolism's most interesting output (cross-cluster
// isomorphisms that don't fit any canonical predicate) stays inert.
func TestLoadTripIsolatedConns(t *testing.T) {
	t.Run("absent file returns nil silently", func(t *testing.T) {
		dir := t.TempDir()
		got := loadTripIsolatedConns(dir)
		if got != nil {
			t.Errorf("expected nil for missing file, got %d rows", len(got))
		}
	})

	t.Run("parses valid JSONL", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"timestamp":"2026-04-28T05:08:00Z","entity_a":"A","entity_b":"B","cluster_a":0,"cluster_b":1,"connection":"both X","rationale":"r","score":4,"prompt_type":"analogy","temperature":1.0,"drug_profile":"psychedelic/pattern-matching"}
{"timestamp":"2026-04-28T05:09:00Z","entity_a":"C","entity_b":"D","cluster_a":2,"cluster_b":3,"connection":"both Y","rationale":"s","score":3,"prompt_type":"analogy","temperature":0.9,"drug_profile":"exploratory/pattern-matching"}
`
		if err := os.WriteFile(filepath.Join(dir, ".metabolism-trip-isolated.jsonl"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got := loadTripIsolatedConns(dir)
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		if got[0].EntityA != "A" || got[0].Score != 4 {
			t.Errorf("row 0: got %+v", got[0])
		}
		if got[1].EntityA != "C" || got[1].Score != 3 {
			t.Errorf("row 1: got %+v", got[1])
		}
	})

	t.Run("skips malformed lines", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"entity_a":"valid","entity_b":"valid","score":4}
this is not json
{"entity_a":"alsovalid","entity_b":"valid","score":3}
`
		if err := os.WriteFile(filepath.Join(dir, ".metabolism-trip-isolated.jsonl"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got := loadTripIsolatedConns(dir)
		if len(got) != 2 {
			t.Errorf("expected 2 valid rows (skipping malformed), got %d", len(got))
		}
	})
}

// TestBuildKBContext_IncludesTripIsolated verifies the context builder
// surfaces the JSONL section when the file exists, and omits it cleanly
// when absent. This pins the "demand-side wiring" behavior.
func TestBuildKBContext_IncludesTripIsolated(t *testing.T) {
	emptyKB := &kbIndex{}

	t.Run("absent file: no section emitted", func(t *testing.T) {
		dir := t.TempDir()
		out := buildKBContext(emptyKB, dir)
		if strings.Contains(out, "Speculative Cross-Cluster Connections") {
			t.Error("expected no trip-isolated section when file absent")
		}
	})

	t.Run("present file: section surfaces with content", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"entity_a":"GodelFirstIncompletenessTheorem","entity_b":"BaloneyDetectionKitThesis","cluster_a":0,"cluster_b":1,"connection":"both reveal limits of formal validation","score":4,"prompt_type":"analogy","rationale":"r"}
`
		if err := os.WriteFile(filepath.Join(dir, ".metabolism-trip-isolated.jsonl"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		out := buildKBContext(emptyKB, dir)
		if !strings.Contains(out, "Speculative Cross-Cluster Connections") {
			t.Error("expected trip-isolated section header")
		}
		if !strings.Contains(out, "GodelFirstIncompletenessTheorem") || !strings.Contains(out, "both reveal limits of formal validation") {
			t.Error("expected entity names + connection narrative in context")
		}
	})
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
