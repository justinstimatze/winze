package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// budgetSchema is bumped when budgetState's shape changes incompatibly.
const budgetSchema = 1

// Per-phase declared cost estimates in cents. These are the brief's
// reference numbers — sense (Kagi ×5 targets ≈ $0.13), resolve (Sonnet,
// variable, midpoint estimate), pipeline (Sonnet + extraction), trip
// (Sonnet generation + Haiku scoring), dream-fix (Haiku tighten).
//
// These are estimates, not actual spend. A future enhancement can plug
// in measured token counts; for now the file-backed running total is
// "estimated cumulative cost" and the gate is conservative.
const (
	costSenseCents     = 13 // 5 Kagi targets × $0.025 = $0.125
	costResolveCents   = 10 // Sonnet, midpoint of $0.05-0.15 range
	costIngestCents    = 20 // Sonnet pipeline, midpoint of $0.10-0.30
	costTripCents      = 15 // Sonnet narrative + Haiku scoring
	costDreamFixCents  = 1  // Haiku Brief tightening
)

// budgetState is what gets persisted to .metabolism-budget.json. The
// month rollover is checked on every load so resets happen lazily —
// no cron required.
type budgetState struct {
	SchemaVersion int    `json:"schema_version"`
	Month         string `json:"month"`        // "2026-04"
	SpentCents    int    `json:"spent_cents"` // accumulated estimated spend this month
	UpdatedAt     string `json:"updated_at"`  // RFC3339, last write
}

// budgetGuard wraps the persisted state plus the cap loaded from
// METABOLISM_BUDGET_CENTS. Cap of 0 means unlimited (no gating).
type budgetGuard struct {
	dir       string
	capCents  int
	state     budgetState
	loaded    bool
}

// loadBudgetGuard reads .metabolism-budget.json (creating a fresh state
// if absent) and resets the spent counter on month rollover. Reads the
// cap from METABOLISM_BUDGET_CENTS — empty/invalid → 0 (unlimited).
func loadBudgetGuard(dir string) *budgetGuard {
	g := &budgetGuard{dir: dir}
	if v := os.Getenv("METABOLISM_BUDGET_CENTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			g.capCents = n
		}
	}
	path := filepath.Join(dir, ".metabolism-budget.json")
	b, err := os.ReadFile(path)
	if err != nil {
		// Fresh state — first ever run, or file deleted.
		g.state = budgetState{SchemaVersion: budgetSchema, Month: currentMonth(), SpentCents: 0}
		g.loaded = true
		return g
	}
	if err := json.Unmarshal(b, &g.state); err != nil {
		// Corrupt file — start fresh. Don't error out — budget is best-effort telemetry.
		g.state = budgetState{SchemaVersion: budgetSchema, Month: currentMonth(), SpentCents: 0}
		g.loaded = true
		return g
	}
	// Month rollover reset.
	if g.state.Month != currentMonth() {
		g.state.Month = currentMonth()
		g.state.SpentCents = 0
		g.state.SchemaVersion = budgetSchema
	}
	g.loaded = true
	return g
}

// allow returns (true, reason) if the phase fits in the remaining
// budget, else (false, reason). The reason is meant for logGate output.
// Cap of 0 = unlimited (always allows).
func (g *budgetGuard) allow(phase string, estCents int) (bool, string) {
	if g.capCents <= 0 {
		return true, "no budget cap (METABOLISM_BUDGET_CENTS unset)"
	}
	remaining := g.capCents - g.state.SpentCents
	if remaining < 0 {
		remaining = 0
	}
	if estCents > remaining {
		return false, fmt.Sprintf("would cost ~%d¢ but only %d¢ remaining of %d¢ monthly cap (set METABOLISM_BUDGET_CENTS=0 to disable)", estCents, remaining, g.capCents)
	}
	return true, fmt.Sprintf("estimated %d¢ fits in %d¢ remaining (cap %d¢)", estCents, remaining, g.capCents)
}

// charge adds the phase's estimated cost to the persisted total and
// writes the file. Called after a phase actually runs (not when gated
// off). Errors during persistence are logged but don't fail the phase
// — budget is bookkeeping, not a hard transaction.
func (g *budgetGuard) charge(phase string, estCents int) {
	if g.capCents <= 0 {
		return // not tracking
	}
	g.state.SpentCents += estCents
	g.state.UpdatedAt = time.Now().Format(time.RFC3339)
	g.state.SchemaVersion = budgetSchema
	path := filepath.Join(g.dir, ".metabolism-budget.json")
	b, err := json.MarshalIndent(g.state, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[budget] marshal: %v\n", err)
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[budget] write %s: %v\n", path, err)
	}
}

// summary returns a one-line status for the cycle header.
func (g *budgetGuard) summary() string {
	if g.capCents <= 0 {
		return "budget: unlimited (METABOLISM_BUDGET_CENTS unset)"
	}
	return fmt.Sprintf("budget: %d¢ / %d¢ spent this month (%s)", g.state.SpentCents, g.capCents, g.state.Month)
}

func currentMonth() string {
	return time.Now().UTC().Format("2006-01")
}
