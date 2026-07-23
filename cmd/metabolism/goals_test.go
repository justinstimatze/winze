package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoalSensorTargetsRepo pins the live wiring: the corpus's active in-domain
// goal must produce sensor targets (one per seed) marked learning_goal, so the
// exploration drive actually feeds the sense phase.
func TestGoalSensorTargetsRepo(t *testing.T) {
	ts := goalSensorTargets(repoRoot(t))
	if len(ts) == 0 {
		t.Fatal("no goal sensor targets — the active LearningGoal isn't reaching sense")
	}
	for _, target := range ts {
		if target.VulnType != "learning_goal" {
			t.Errorf("goal target VulnType = %q, want learning_goal", target.VulnType)
		}
		if target.Query == "" {
			t.Error("goal target has empty query")
		}
	}
}

// TestGoalCoverageStops is the coverage loop: a goal at/over CoverAt generates
// nothing (satisfied), under it generates one target per seed. Fixture parsed
// by AST — no type-checking, so the literals need no supporting type defs.
func TestGoalCoverageStops(t *testing.T) {
	write := func(dir string, tagged int) {
		src := `package winze
var G = LearningGoal{&Entity{ID: "g", Name: "G"}}
var gSpec = GoalSpec{Goal: G, Seeds: []string{"alpha", "beta"}, InDomain: true, CoverAt: 2}
`
		for i := 0; i < tagged; i++ {
			src += "var tag" + string(rune('A'+i)) + " = AdvancesGoal{Subject: SomeConcept, Object: G}\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "goals.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	under := t.TempDir()
	write(under, 1) // 1 tagged < CoverAt 2 → active, 2 seeds → 2 targets
	if got := len(goalSensorTargets(under)); got != 2 {
		t.Errorf("under cap: want 2 targets (2 seeds), got %d", got)
	}
	if n := countGoalTagged(under, "G"); n != 1 {
		t.Errorf("countGoalTagged = %d, want 1", n)
	}

	met := t.TempDir()
	write(met, 2) // 2 tagged == CoverAt 2 → satisfied → 0 targets
	if got := len(goalSensorTargets(met)); got != 0 {
		t.Errorf("at cap: want 0 targets (goal satisfied), got %d", got)
	}
}

// TestCrossDomainGoalSkipped: an InDomain:false goal seeds a fork, never main,
// so it must produce no sensor targets here.
func TestCrossDomainGoalSkipped(t *testing.T) {
	dir := t.TempDir()
	src := `package winze
var QC = LearningGoal{&Entity{ID: "qc", Name: "QC"}}
var qcSpec = GoalSpec{Goal: QC, Seeds: []string{"surface code"}, InDomain: false, CoverAt: 8}
`
	if err := os.WriteFile(filepath.Join(dir, "goals.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := len(goalSensorTargets(dir)); got != 0 {
		t.Errorf("cross-domain goal must not target main, got %d targets", got)
	}
}

// TestGoalTagSection pins the coverage-tag emission: an AdvancesGoal claim,
// Conjecture-attributed (winze's own bookkeeping), carrying NO Quote — the
// concept was acquired pursuing the goal, not committed to by a source.
func TestGoalTagSection(t *testing.T) {
	got := goalTagSection("AberrantPrecision", "GoalPredictiveHallucination")
	for _, want := range []string{
		"= AdvancesGoal{",
		"Subject: AberrantPrecision,",
		"Object:  GoalPredictiveHallucination,",
		"Prov: Conjecture{",
		`GeneratedBy: "metabolism-goal-ingest"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("goalTagSection missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Quote") {
		t.Error("AdvancesGoal tag must not carry a Quote — it is a Conjecture, not a sourced claim")
	}
	// Var name must be a valid, unique identifier.
	if !strings.HasPrefix(got, "var AberrantPrecisionAdvancesGoalPredictiveHallucination = ") {
		t.Errorf("unexpected var name in: %s", got)
	}
}
