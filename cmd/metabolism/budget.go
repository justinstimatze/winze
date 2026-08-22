package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
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
	costSenseCents    = 13 // 5 Kagi targets × $0.025 = $0.125
	costResolveCents  = 10 // Sonnet, midpoint of $0.05-0.15 range
	costIngestCents   = 20 // Sonnet pipeline, midpoint of $0.10-0.30
	costTripCents     = 15 // Sonnet narrative + Haiku scoring
	costDreamFixCents = 1  // Haiku Brief tightening
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
//
// The two used to accumulate independently for the life of a month, which
// made the gate progressively wrong in one direction: estimates are
// deliberately pessimistic, so the tally that decides whether a phase may
// run drifted above what was really spent and stayed there. Measured on
// 2026-08-06 the corpus had SpentCents=300 against 11.90¢ actual and had
// been refusing every generative phase for eighteen hours at 4% of its
// cap. loadBudgetGuard now settles the estimate to the measured total on
// load — see UnmeasuredCalls for when it declines to.
type budgetState struct {
	SchemaVersion    int     `json:"schema_version"`
	Month            string  `json:"month"`              // "2026-04"
	SpentCents       int     `json:"spent_cents"`        // accumulated estimated spend this month
	ActualSpentCents float64 `json:"actual_spent_cents"` // measured spend from anthropic.Usage this month
	UpdatedAt        string  `json:"updated_at"`         // RFC3339, last write

	// UnmeasuredCalls counts LLM responses whose cost could not be measured
	// because the model had no pricing entry (see warnUnknownModel). It is
	// what makes settling safe: ActualSpentCents is a floor, not a total,
	// and a month containing even one unpriced call is a month where
	// settling down to it would erase real spend. Non-zero means the
	// estimate stands and the gate stays conservative, which is the correct
	// direction to be wrong in.
	UnmeasuredCalls int `json:"unmeasured_calls,omitempty"`

	// Cache-effectiveness telemetry (this month). CacheReadTokens is input
	// served from the prompt cache (billed ~10%); FreshInputTokens is
	// uncached input (full price). The hit ratio CacheRead/(CacheRead+Fresh)
	// answers "is the shared cache_control'd prefix actually landing?" — a
	// silent regression (prefix drift below the model's min-cacheable floor,
	// a structure change, a TTL miss) shows up as the ratio collapsing.
	// Note: cache_creation tokens (the one-time establishing write, +25%) are
	// not captured here, so the ratio slightly overstates — it measures reads
	// against fresh input, not against total prompt tokens.
	CacheReadTokens  int64 `json:"cache_read_tokens"`
	FreshInputTokens int64 `json:"fresh_input_tokens"`
	// CacheWriteTokens is the establishing write, billed at 2x base for the 1h
	// TTL this repo uses. Until 2026-08-22 it was neither recorded nor priced,
	// so ActualSpentCents understated by the full cost of every cache write —
	// and with reads sitting at 0 for all of August, every marked call was
	// paying that 2x premium and nothing was reading it back.
	CacheWriteTokens int64 `json:"cache_write_tokens"`
}

// budgetGuard wraps the persisted state plus the cap loaded from
// METABOLISM_BUDGET_CENTS. Cap of 0 means unlimited (no gating).
type budgetGuard struct {
	dir      string
	capCents int
	state    budgetState
	loaded   bool
}

// loadBudgetGuard reads .metabolism-budget.json (creating a fresh state
// if absent) and resets the spent counter on month rollover. Reads the
// cap from METABOLISM_BUDGET_CENTS — empty/invalid → 0 (unlimited).
//
// Load is also where the estimate tally is settled against measured spend
// (see settleEstimateToActual). Doing it here rather than at the end of a
// run means a crashed or killed cycle still gets reconciled on the next
// start, and it keeps the whole correction outside the window where a
// phase's own actuals have not arrived yet.
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
	// Month rollover reset. Every per-month tally goes, not just the
	// estimate: ActualSpentCents and UnmeasuredCalls are scoped to Month
	// exactly as SpentCents is, and leaving them would hand the settle step
	// below last month's measured total as if it were this month's — which
	// would zero out a fresh month's cap on the first run of it.
	if g.state.Month != currentMonth() {
		g.state.Month = currentMonth()
		g.state.SpentCents = 0
		g.state.ActualSpentCents = 0
		g.state.UnmeasuredCalls = 0
		g.state.CacheReadTokens = 0
		g.state.CacheWriteTokens = 0
		g.state.FreshInputTokens = 0
		g.state.SchemaVersion = budgetSchema
	}
	if moved, why := settleEstimateToActual(&g.state); moved {
		fmt.Fprintf(os.Stderr, "[budget] %s\n", why)
		g.persist()
	} else if why != "" {
		fmt.Fprintf(os.Stderr, "[budget] not settling: %s\n", why)
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
//
// The estimate is still what accumulates here, and still what allow()
// reads, so a single run cannot outspend its cap while its own actuals
// are still arriving. The correction happens between runs, in
// settleEstimateToActual.
func (g *budgetGuard) charge(phase string, estCents int) {
	if g.capCents <= 0 {
		return // not tracking
	}
	g.state.SpentCents += estCents
	g.persist()
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
			return fmt.Sprintf("budget: unlimited; actual %.2f¢ this month (%s)%s", g.state.ActualSpentCents, g.state.Month, g.cacheSuffix())
		}
		return "budget: unlimited (METABOLISM_BUDGET_CENTS unset)"
	}
	return fmt.Sprintf("budget: %d¢ est / %.2f¢ actual / %d¢ cap this month (%s)%s", g.state.SpentCents, g.state.ActualSpentCents, g.capCents, g.state.Month, g.cacheSuffix())
}

// cacheHitPct reports the share of input tokens served from the prompt cache
// this month, as read/(read+write+fresh). Returns (0, false) before any LLM
// call has been billed, so the caller can omit the stat rather than print a
// meaningless 0%.
//
// Writes belong in the denominator. Leaving them out asks "of the tokens we
// paid full price for, how many came from cache" — which flatters a breakpoint
// that is being established over and over and never read, because the write
// tokens simply vanish from the accounting. Including them asks the question
// worth asking: of everything this prompt cost, how much did the cache save.
func (g *budgetGuard) cacheHitPct() (float64, bool) {
	total := g.state.CacheReadTokens + g.state.CacheWriteTokens + g.state.FreshInputTokens
	if total == 0 {
		return 0, false
	}
	return 100 * float64(g.state.CacheReadTokens) / float64(total), true
}

// cacheSuffix renders the hit-ratio clause for the budget line, or "" when
// there is no usage data yet.
func (g *budgetGuard) cacheSuffix() string {
	pct, ok := g.cacheHitPct()
	if !ok {
		return ""
	}
	return fmt.Sprintf(" | cache %.0f%% hit (%d read / %d written / %d fresh tok)",
		pct, g.state.CacheReadTokens, g.state.CacheWriteTokens, g.state.FreshInputTokens)
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
	inputPerMTok     float64
	outputPerMTok    float64
	cacheReadPerMTok float64 // 10% of input price for Sonnet/Haiku 4.x
	// cacheWritePerMTok is the one-time establishing write. Anthropic bills it
	// by TTL: 1.25x base for the 5-minute breakpoint, 2x for the 1-hour one.
	// This is the 1h rate, because every cache_control in this repo — all eight
	// of them — uses CacheControlEphemeralTTLTTL1h. A 5m breakpoint added later
	// would need its own rate rather than reusing this one.
	cacheWritePerMTok float64
}

var pricingByModel = map[string]modelPricing{
	"claude-sonnet-4-5":            {inputPerMTok: 3.00, outputPerMTok: 15.00, cacheReadPerMTok: 0.30, cacheWritePerMTok: 6.00},
	"claude-sonnet-4-5-2025-09-29": {inputPerMTok: 3.00, outputPerMTok: 15.00, cacheReadPerMTok: 0.30, cacheWritePerMTok: 6.00},
	"claude-sonnet-4-6":            {inputPerMTok: 3.00, outputPerMTok: 15.00, cacheReadPerMTok: 0.30, cacheWritePerMTok: 6.00},
	"claude-haiku-4-5":             {inputPerMTok: 1.00, outputPerMTok: 5.00, cacheReadPerMTok: 0.10, cacheWritePerMTok: 2.00},
	"claude-haiku-4-5-20251001":    {inputPerMTok: 1.00, outputPerMTok: 5.00, cacheReadPerMTok: 0.10, cacheWritePerMTok: 2.00},
}

// tokenUsage is one response's billable token counts, kept together because
// they are always produced together and always priced together. Passing them as
// four positional int64s made every signature in this file read
// (model, int64, int64, int64, int64) — trivially transposable, and a
// transposed cache-read for a cache-write is a 20x pricing error that nothing
// would catch.
type tokenUsage struct {
	Input      int64 // fresh, uncached input — full price
	CacheRead  int64 // served from the prompt cache — 10% of base
	CacheWrite int64 // establishing write — 2x base at the 1h TTL used here
	Output     int64
}

// usageOf lifts an SDK response's usage into the local shape.
func usageOf(u anthropic.Usage) tokenUsage {
	return tokenUsage{
		Input:      u.InputTokens,
		CacheRead:  u.CacheReadInputTokens,
		CacheWrite: u.CacheCreationInputTokens,
		Output:     u.OutputTokens,
	}
}

// costCents converts measured tokens to a fractional-cent cost.
// Returns (cents, true) on hit, (0, false) on unknown model — caller
// decides whether to log the miss.
func costCents(model string, t tokenUsage) (float64, bool) {
	p, ok := pricingByModel[model]
	if !ok {
		return 0, false
	}
	dollars := (float64(t.Input)/1e6)*p.inputPerMTok +
		(float64(t.CacheRead)/1e6)*p.cacheReadPerMTok +
		(float64(t.CacheWrite)/1e6)*p.cacheWritePerMTok +
		(float64(t.Output)/1e6)*p.outputPerMTok
	return dollars * 100, true
}

// chargeActual records measured spend from one LLM response and persists
// the running total. Unknown models log once to stderr and skip — better
// to under-report than to crash the metabolism on a model rename.
//
// A skip is counted, not just logged. Silent under-reporting was harmless
// while actuals were read-only telemetry; now that loadBudgetGuard settles
// the estimate down to them, an uncounted call would settle away real
// money. UnmeasuredCalls is how the settle step knows to keep its hands
// off.
func (g *budgetGuard) chargeActual(model string, t tokenUsage) {
	cents, ok := costCents(model, t)
	if !ok {
		warnUnknownModel(model)
		g.state.UnmeasuredCalls++
		g.persist()
		return
	}
	g.state.ActualSpentCents += cents
	g.state.CacheReadTokens += t.CacheRead
	g.state.CacheWriteTokens += t.CacheWrite
	g.state.FreshInputTokens += t.Input
	g.persist()
}

// globalBudget is set by runCycle (and ensureBudgetGuard for standalone
// CLI commands) so per-call accounting at LLM call sites doesn't have to
// thread the guard through every signature. Standalone commands that
// make LLM calls (reresolve-irrelevant, irrelevance-audit, autoresolve,
// calibrate-narrative) must call ensureBudgetGuard at entry.
var globalBudget *budgetGuard

// ensureBudgetGuard initializes globalBudget for the given dir if it
// hasn't been set yet. Idempotent — safe to call from any LLM-using
// CLI subcommand. Won't override an existing globalBudget so --evolve's
// guard isn't accidentally replaced from inside a sub-call.
func ensureBudgetGuard(dir string) {
	if globalBudget != nil {
		return
	}
	globalBudget = loadBudgetGuard(dir)
}

// recordActualUsage is the call-site-friendly wrapper. Safe to call
// when globalBudget is nil (no-op).
func recordActualUsage(model string, t tokenUsage) {
	if globalBudget == nil {
		return
	}
	globalBudget.chargeActual(model, t)
}

var unknownModelWarned = map[string]bool{}

func warnUnknownModel(model string) {
	if unknownModelWarned[model] {
		return
	}
	unknownModelWarned[model] = true
	fmt.Fprintf(os.Stderr, "[budget] no pricing entry for model %q — actual spend not counted. Add it to pricingByModel in budget.go.\n", model)
}

// persist stamps the schema version and timestamp and writes the state
// file. Errors are logged, never returned — budget is bookkeeping, not a
// hard transaction, and a failed write must not take a phase down with it.
//
// Extracted because charge and chargeActual had the same eighteen lines
// twice, and the settle logic added a third writer.
func (g *budgetGuard) persist() {
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

// settleEstimateToActual brings the gate's estimate tally back down to
// what was really spent, and reports whether it moved.
//
// This is the between-runs half of a reserve-then-settle pair. Within a
// run the estimate is the right thing to gate on, because a phase is
// charged before its own usage numbers exist and an optimistic tally
// would let a runaway loop through. Between runs that argument is gone:
// every prior call's actual has landed, so carrying the pessimistic
// figure forward just shrinks next month's usable cap for no protection.
// Left alone the two diverge monotonically, and the gap only ever points
// one way — at 4x per cycle the corpus reached 300¢ estimated against
// 11.90¢ actual and stopped generating for eighteen hours.
//
// Three refusals, each closing a way this could erase real spend:
//
// Zero measured. An estimate above zero with nothing measured under it
// has two readings — nothing was billed, or the measurement path is not
// running — and from here they are indistinguishable. The second is the
// dangerous one: recordActualUsage is wired through a package-level
// guard, so a call path that misses it reports zero while spending money,
// and settling to zero every load would leave the cap permanently
// unreachable. Requiring positive evidence that measurement is alive
// costs nothing in the real case, where actuals are tens of cents.
//
// Unpriced calls. ActualSpentCents is then a floor rather than a total,
// and settling to a floor drops whatever those calls cost.
//
// Measured above estimated. The estimator was optimistic; the larger
// number is the safe one to keep.
func settleEstimateToActual(s *budgetState) (moved bool, reason string) {
	if s.UnmeasuredCalls > 0 {
		return false, fmt.Sprintf("%d unpriced call(s) this month — actual is a floor, keeping the estimate", s.UnmeasuredCalls)
	}
	if s.ActualSpentCents <= 0 {
		if s.SpentCents > 0 {
			return false, fmt.Sprintf("%d¢ estimated but nothing measured — cannot tell an unbilled month from a broken meter, keeping the estimate", s.SpentCents)
		}
		return false, ""
	}
	settled := int(math.Ceil(s.ActualSpentCents))
	if settled == s.SpentCents {
		return false, ""
	}
	was := s.SpentCents
	s.SpentCents = settled
	if settled > was {
		// Settling UP. This used to be impossible by construction: the
		// per-phase estimates were assumed to upper-bound measured cost, so the
		// function only ever moved downward and silently returned here. Pricing
		// cache writes at the 1h rate (2x base) broke that assumption — a phase
		// that establishes a breakpoint can now measure above its own estimate.
		// Leaving SpentCents at the low estimate would mean allow() enforces the
		// monthly cap against a number already known to be wrong, so the cap
		// would quietly stop being a cap. Settle up and say so: a persistent
		// message here means the estimate constants need raising.
		return true, fmt.Sprintf("settled %d¢ estimated UP to %d¢ measured — a phase cost more than its estimate; raise the per-phase constants if this repeats", was, settled)
	}
	return true, fmt.Sprintf("settled %d¢ estimated → %d¢ measured", was, settled)
}
