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
//
// SpentCents is the conservative *estimate* tally driving the gate
// (charged before each phase from cost*Cents constants).
// ActualSpentCents is the *measured* tally from anthropic.Usage, recorded
// after each LLM response. Estimates protect against runaway loops;
// actuals show what really got spent in the trend reader.
type budgetState struct {
	SchemaVersion    int     `json:"schema_version"`
	Month            string  `json:"month"`              // "2026-04"
	SpentCents       int     `json:"spent_cents"`        // accumulated estimated spend this month
	ActualSpentCents float64 `json:"actual_spent_cents"` // measured spend from anthropic.Usage this month
	UpdatedAt        string  `json:"updated_at"`         // RFC3339, last write
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

// loadBudgetSnapshot reads .metabolism-budget.json without applying month
// rollover or env-var lookup — used by runCalibrate to stamp the
// CalibrationRow with whatever happened to be in the budget file at the
// moment of the calibrate run. Returns zero values on missing/corrupt
// file (no error — calibration is informational, not transactional).
// capCents is read from METABOLISM_BUDGET_CENTS the same way loadBudgetGuard
// does so the snapshot is consistent with what the gate uses.
func loadBudgetSnapshot(dir string) (estCents, capCents int, actualCents float64) {
	if v := os.Getenv("METABOLISM_BUDGET_CENTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			capCents = n
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, ".metabolism-budget.json"))
	if err != nil {
		return 0, capCents, 0
	}
	var s budgetState
	if err := json.Unmarshal(b, &s); err != nil {
		return 0, capCents, 0
	}
	// Cross-month: the file's month-to-date totals are stale. Don't surface
	// last month's spend in this month's row.
	if s.Month != currentMonth() {
		return 0, capCents, 0
	}
	return s.SpentCents, capCents, s.ActualSpentCents
}

// summary returns a one-line status for the cycle header. Shows both the
// conservative estimate (drives the gate) and the measured actual when
// non-zero, so the gap between the two is visible.
func (g *budgetGuard) summary() string {
	if g.capCents <= 0 {
		if g.state.ActualSpentCents > 0 {
			return fmt.Sprintf("budget: unlimited; actual %.2f¢ this month (%s)", g.state.ActualSpentCents, g.state.Month)
		}
		return "budget: unlimited (METABOLISM_BUDGET_CENTS unset)"
	}
	return fmt.Sprintf("budget: %d¢ est / %.2f¢ actual / %d¢ cap this month (%s)", g.state.SpentCents, g.state.ActualSpentCents, g.capCents, g.state.Month)
}

func currentMonth() string {
	return time.Now().UTC().Format("2006-01")
}

// --- Actual-spend accounting -------------------------------------------------
//
// Conservative pre-gate estimates protect against runaway loops; actual
// post-call measurements (from anthropic.Usage) are what the trend reader
// surfaces as $/week. Estimates and actuals are tracked separately so the
// gap between them is observable — if the gap grows the per-phase
// constants are stale and need updating.

// modelPricing in dollars per million tokens. Source: anthropic.com/pricing
// (verified April 2026). Add new model entries here when bumping the SDK.
type modelPricing struct {
	inputPerMTok  float64
	outputPerMTok float64
	cacheReadPerMTok float64 // 10% of input price for Sonnet/Haiku 4.x
}

var pricingByModel = map[string]modelPricing{
	"claude-sonnet-4-5":     {inputPerMTok: 3.00, outputPerMTok: 15.00, cacheReadPerMTok: 0.30},
	"claude-sonnet-4-5-2025-09-29": {inputPerMTok: 3.00, outputPerMTok: 15.00, cacheReadPerMTok: 0.30},
	"claude-sonnet-4-6":     {inputPerMTok: 3.00, outputPerMTok: 15.00, cacheReadPerMTok: 0.30},
	"claude-haiku-4-5":      {inputPerMTok: 1.00, outputPerMTok: 5.00,  cacheReadPerMTok: 0.10},
	"claude-haiku-4-5-20251001": {inputPerMTok: 1.00, outputPerMTok: 5.00,  cacheReadPerMTok: 0.10},
}

// costCents converts measured tokens to a fractional-cent cost.
// Returns (cents, true) on hit, (0, false) on unknown model — caller
// decides whether to log the miss.
func costCents(model string, inputTokens, cachedReadTokens, outputTokens int64) (float64, bool) {
	p, ok := pricingByModel[model]
	if !ok {
		return 0, false
	}
	dollars := (float64(inputTokens)/1e6)*p.inputPerMTok +
		(float64(cachedReadTokens)/1e6)*p.cacheReadPerMTok +
		(float64(outputTokens)/1e6)*p.outputPerMTok
	return dollars * 100, true
}

// chargeActual records measured spend from one LLM response and persists
// the running total. Unknown models log once to stderr and skip — better
// to under-report than to crash the metabolism on a model rename.
func (g *budgetGuard) chargeActual(model string, inputTokens, cachedReadTokens, outputTokens int64) {
	cents, ok := costCents(model, inputTokens, cachedReadTokens, outputTokens)
	if !ok {
		warnUnknownModel(model)
		return
	}
	g.state.ActualSpentCents += cents
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

// globalBudget is set by runCycle so per-call accounting at LLM call
// sites doesn't have to thread the guard through every signature. nil
// when not in an --evolve run (standalone tools, tests).
var globalBudget *budgetGuard

// recordActualUsage is the call-site-friendly wrapper. Safe to call
// when globalBudget is nil (no-op).
func recordActualUsage(model string, inputTokens, cachedReadTokens, outputTokens int64) {
	if globalBudget == nil {
		return
	}
	globalBudget.chargeActual(model, inputTokens, cachedReadTokens, outputTokens)
}

var unknownModelWarned = map[string]bool{}

func warnUnknownModel(model string) {
	if unknownModelWarned[model] {
		return
	}
	unknownModelWarned[model] = true
	fmt.Fprintf(os.Stderr, "[budget] no pricing entry for model %q — actual spend not counted. Add it to pricingByModel in budget.go.\n", model)
}
