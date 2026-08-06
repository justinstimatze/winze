package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestSettleEstimateToActual covers the direction rules. The case that
// caused the outage is "estimate far above actual"; the rest are the
// refusals that keep it from erasing spend it cannot see.
func TestSettleEstimateToActual(t *testing.T) {
	cases := []struct {
		name      string
		in        budgetState
		wantSpent int
		wantMoved bool
	}{
		{
			// The 2026-08-06 state, verbatim. Eighteen hours of refusing
			// every generative phase at 4% of the cap.
			name:      "settles the observed drift",
			in:        budgetState{SpentCents: 300, ActualSpentCents: 11.899},
			wantSpent: 12, wantMoved: true,
		},
		{
			// Rounds up, never down: settling to 11 would under-report a
			// month that really cost 11.899.
			name:      "ceils rather than truncates",
			in:        budgetState{SpentCents: 300, ActualSpentCents: 11.001},
			wantSpent: 12, wantMoved: true,
		},
		{
			// The estimator was optimistic. Keeping the larger number is the
			// safe direction, so this must not settle upward.
			name:      "never settles upward",
			in:        budgetState{SpentCents: 10, ActualSpentCents: 90},
			wantSpent: 10, wantMoved: false,
		},
		{
			// One unpriced model makes ActualSpentCents a floor. Settling to
			// a floor erases whatever that call cost.
			name:      "refuses when a call went unpriced",
			in:        budgetState{SpentCents: 300, ActualSpentCents: 11.899, UnmeasuredCalls: 1},
			wantSpent: 300, wantMoved: false,
		},
		{
			// The hazard the pre-existing charge test caught: charges with no
			// measurement under them read identically whether nothing was
			// billed or recordActualUsage never ran. Settling here would make
			// the cap permanently unreachable.
			name:      "refuses when nothing was measured at all",
			in:        budgetState{SpentCents: 28, ActualSpentCents: 0},
			wantSpent: 28, wantMoved: false,
		},
		{
			name:      "no-op when already equal",
			in:        budgetState{SpentCents: 12, ActualSpentCents: 11.899},
			wantSpent: 12, wantMoved: false,
		},
		{
			name:      "no-op on a clean slate",
			in:        budgetState{},
			wantSpent: 0, wantMoved: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.in
			moved, reason := settleEstimateToActual(&s)
			if moved != tc.wantMoved {
				t.Errorf("moved = %v, want %v (reason %q)", moved, tc.wantMoved, reason)
			}
			if s.SpentCents != tc.wantSpent {
				t.Errorf("SpentCents = %d, want %d", s.SpentCents, tc.wantSpent)
			}
			// A refusal that had something to refuse must explain itself. A
			// silent one is indistinguishable from nothing needing settling,
			// which is exactly the state an operator would want to see named.
			if !moved && tc.in.SpentCents > 0 && tc.in.SpentCents != tc.wantSpent {
				t.Error("refused but changed SpentCents — refusals must leave it alone")
			}
			if !moved && reason == "" && (tc.in.UnmeasuredCalls > 0 ||
				(tc.in.SpentCents > 0 && tc.in.ActualSpentCents <= 0)) {
				t.Error("refusing must say why — silence reads as 'nothing to do'")
			}
		})
	}
}

// TestLoadBudgetGuard_SettlesAndRollsOver checks the two things load has to
// get right together: settle an over-estimated month, and clear every
// per-month tally on rollover so the settle step never sees a stale actual.
//
// The rollover leg is the one worth guarding. Before this change the reset
// cleared SpentCents alone, which was harmless while actuals were read-only
// telemetry — the moment load started settling to them, a rolled-over month
// would have inherited the previous month's measured total as its opening
// balance.
func TestLoadBudgetGuard_SettlesAndRollsOver(t *testing.T) {
	write := func(t *testing.T, dir string, s budgetState) {
		t.Helper()
		b, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".metabolism-budget.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("METABOLISM_BUDGET_CENTS", "1000")

	t.Run("settles the current month", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, budgetState{
			SchemaVersion: budgetSchema, Month: currentMonth(),
			SpentCents: 300, ActualSpentCents: 11.899,
		})
		g := loadBudgetGuard(dir)
		if g.state.SpentCents != 12 {
			t.Errorf("SpentCents = %d, want 12", g.state.SpentCents)
		}
		// The gate must now let a phase through that it was refusing.
		if ok, why := g.allow("trip", 15); !ok {
			t.Errorf("a 15¢ phase should fit after settling: %s", why)
		}
		// And the settled value must be on disk, not just in memory —
		// otherwise the next process re-reads 300 and nothing changed.
		reloaded := loadBudgetGuard(dir)
		if reloaded.state.SpentCents != 12 {
			t.Errorf("after reload SpentCents = %d, want 12 — settle did not persist",
				reloaded.state.SpentCents)
		}
	})

	t.Run("rollover clears actual before settling", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, budgetState{
			SchemaVersion: budgetSchema, Month: "1999-01",
			SpentCents: 300, ActualSpentCents: 950, UnmeasuredCalls: 4,
			CacheReadTokens: 5, FreshInputTokens: 7,
		})
		g := loadBudgetGuard(dir)
		if g.state.Month != currentMonth() {
			t.Fatalf("Month = %q, want %q", g.state.Month, currentMonth())
		}
		for _, f := range []struct {
			name string
			got  float64
		}{
			{"SpentCents", float64(g.state.SpentCents)},
			{"ActualSpentCents", g.state.ActualSpentCents},
			{"UnmeasuredCalls", float64(g.state.UnmeasuredCalls)},
			{"CacheReadTokens", float64(g.state.CacheReadTokens)},
			{"FreshInputTokens", float64(g.state.FreshInputTokens)},
		} {
			if f.got != 0 {
				t.Errorf("%s = %v after rollover, want 0", f.name, f.got)
			}
		}
	})
}
