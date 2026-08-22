package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBudgetGuard_NoCap_AlwaysAllows(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "")
	g := loadBudgetGuard(t.TempDir())
	if g.capCents != 0 {
		t.Errorf("expected capCents=0 when env unset, got %d", g.capCents)
	}
	ok, reason := g.allow("sense", 500)
	if !ok {
		t.Errorf("no cap should allow any cost, got skip: %s", reason)
	}
}

func TestBudgetGuard_InvalidEnvIgnored(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "not-a-number")
	g := loadBudgetGuard(t.TempDir())
	if g.capCents != 0 {
		t.Errorf("invalid env should fall back to 0, got %d", g.capCents)
	}
}

func TestBudgetGuard_AllowsWithinCap(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "100")
	g := loadBudgetGuard(t.TempDir())
	ok, _ := g.allow("sense", 13)
	if !ok {
		t.Errorf("13¢ should fit in 100¢ cap")
	}
}

func TestBudgetGuard_BlocksWhenOverCap(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "10")
	dir := t.TempDir()
	g := loadBudgetGuard(dir)
	// 13¢ sense doesn't fit in 10¢ cap.
	ok, reason := g.allow("sense", 13)
	if ok {
		t.Errorf("13¢ shouldn't fit in 10¢ cap")
	}
	if !strings.Contains(reason, "would cost") {
		t.Errorf("reason should explain: %q", reason)
	}
}

func TestBudgetGuard_ChargePersistsAndAccumulates(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "100")
	dir := t.TempDir()
	g := loadBudgetGuard(dir)
	g.charge("sense", 13)
	g.charge("trip", 15)

	// Re-load from disk; verify accumulation persisted.
	g2 := loadBudgetGuard(dir)
	if g2.state.SpentCents != 28 {
		t.Errorf("expected 28¢ spent after two charges, got %d", g2.state.SpentCents)
	}
}

func TestBudgetGuard_ExhaustionBlocks(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "20")
	dir := t.TempDir()
	g := loadBudgetGuard(dir)
	g.charge("sense", 13)
	g.charge("resolve", 10) // total 23 — already over cap
	// Next phase should be blocked even at 1¢.
	ok, reason := g.allow("dream-fix", 1)
	if ok {
		t.Errorf("over-cap budget should block any further phase, got allow: %s", reason)
	}
}

func TestBudgetGuard_MonthRollover_Resets(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "100")
	dir := t.TempDir()
	// Manually write a state file with last month's date.
	old := budgetState{
		SchemaVersion: 1,
		Month:         "2020-01", // ancient — not the current month
		SpentCents:    99,
	}
	b, _ := json.Marshal(old)
	if err := os.WriteFile(filepath.Join(dir, ".metabolism-budget.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	g := loadBudgetGuard(dir)
	if g.state.SpentCents != 0 {
		t.Errorf("month rollover should reset spent to 0, got %d", g.state.SpentCents)
	}
	if g.state.Month != currentMonth() {
		t.Errorf("expected month %s, got %s", currentMonth(), g.state.Month)
	}
}

func TestBudgetGuard_CorruptFileResetsCleanly(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "100")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".metabolism-budget.json"), []byte("not valid json {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	g := loadBudgetGuard(dir)
	// Should not panic; should start fresh.
	if g.state.SpentCents != 0 {
		t.Errorf("corrupt file should reset spent to 0, got %d", g.state.SpentCents)
	}
}

func TestBudgetGuard_Summary(t *testing.T) {
	t.Setenv("METABOLISM_BUDGET_CENTS", "")
	g := loadBudgetGuard(t.TempDir())
	if !strings.Contains(g.summary(), "unlimited") {
		t.Errorf("uncapped summary should say unlimited: %q", g.summary())
	}
	t.Setenv("METABOLISM_BUDGET_CENTS", "200")
	g2 := loadBudgetGuard(t.TempDir())
	g2.charge("sense", 13)
	if !strings.Contains(g2.summary(), "13¢ est") || !strings.Contains(g2.summary(), "200¢ cap") {
		t.Errorf("summary should show est/cap split: %q", g2.summary())
	}
}

func TestCurrentMonth_Format(t *testing.T) {
	got := currentMonth()
	if _, err := time.Parse("2006-01", got); err != nil {
		t.Errorf("currentMonth() = %q, not parseable as YYYY-MM: %v", got, err)
	}
}

func TestCostCents_KnownModel(t *testing.T) {
	// 1M Haiku input + 1M Haiku output = $1 + $5 = $6 = 600¢
	cents, ok := costCents("claude-haiku-4-5", tokenUsage{Input: 1_000_000, Output: 1_000_000})
	if !ok {
		t.Fatal("expected pricing for haiku-4-5")
	}
	if cents < 599.9 || cents > 600.1 {
		t.Errorf("cents = %.4f, want ~600", cents)
	}
}

func TestCostCents_UnknownModel(t *testing.T) {
	cents, ok := costCents("claude-opus-99", tokenUsage{Input: 1000, Output: 1000})
	if ok {
		t.Error("expected unknown model to return ok=false")
	}
	if cents != 0 {
		t.Errorf("cents = %.4f, want 0 for unknown model", cents)
	}
}

func TestCostCents_CacheReadsAreCheaper(t *testing.T) {
	// 1M Sonnet plain input → 300¢; 1M Sonnet cache-read input → 30¢.
	plain, _ := costCents("claude-sonnet-4-5", tokenUsage{Input: 1_000_000})
	cached, _ := costCents("claude-sonnet-4-5", tokenUsage{CacheRead: 1_000_000})
	if cached >= plain {
		t.Errorf("cache-read should be cheaper than plain input: cached=%.2f plain=%.2f", cached, plain)
	}
	if plain < 299.9 || plain > 300.1 {
		t.Errorf("plain sonnet input cents = %.4f, want ~300", plain)
	}
}

func TestChargeActual_AccumulatesAndPersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("METABOLISM_BUDGET_CENTS", "1000")
	g := loadBudgetGuard(dir)
	// Two haiku calls of 100k input + 100k output each:
	// per call: 0.1 * $1 + 0.1 * $5 = $0.60 = 60¢
	g.chargeActual("claude-haiku-4-5", tokenUsage{Input: 100_000, Output: 100_000})
	g.chargeActual("claude-haiku-4-5", tokenUsage{Input: 100_000, Output: 100_000})
	if g.state.ActualSpentCents < 119.9 || g.state.ActualSpentCents > 120.1 {
		t.Errorf("actual = %.4f, want ~120", g.state.ActualSpentCents)
	}
	g2 := loadBudgetGuard(dir)
	if g2.state.ActualSpentCents < 119.9 || g2.state.ActualSpentCents > 120.1 {
		t.Errorf("after reload: actual = %.4f, want ~120", g2.state.ActualSpentCents)
	}
}

func TestChargeActual_UnknownModelDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("METABOLISM_BUDGET_CENTS", "1000")
	g := loadBudgetGuard(dir)
	g.chargeActual("claude-from-the-future", tokenUsage{Input: 1000, Output: 1000})
	if g.state.ActualSpentCents != 0 {
		t.Errorf("unknown model should not be charged: actual=%.4f", g.state.ActualSpentCents)
	}
}

func TestRecordActualUsage_NilGlobalIsNoop(t *testing.T) {
	prev := globalBudget
	t.Cleanup(func() { globalBudget = prev })
	globalBudget = nil
	// Must not panic.
	recordActualUsage("claude-haiku-4-5", tokenUsage{Input: 100, Output: 100})
}

func TestLoadBudgetSnapshot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("METABOLISM_BUDGET_CENTS", "500")
	state := budgetState{
		SchemaVersion:    budgetSchema,
		Month:            currentMonth(),
		SpentCents:       42,
		ActualSpentCents: 17.5,
		UpdatedAt:        time.Now().Format(time.RFC3339),
	}
	b, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, ".metabolism-budget.json"), b, 0644)

	est, cap, actual := loadBudgetSnapshot(dir)
	if est != 42 || cap != 500 || actual < 17.4 || actual > 17.6 {
		t.Errorf("snapshot = (est=%d cap=%d actual=%.2f), want (42, 500, 17.5)", est, cap, actual)
	}
}

func TestLoadBudgetSnapshot_StaleMonthReturnsZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("METABOLISM_BUDGET_CENTS", "500")
	state := budgetState{
		SchemaVersion:    budgetSchema,
		Month:            "1999-12", // stale
		SpentCents:       42,
		ActualSpentCents: 17.5,
	}
	b, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, ".metabolism-budget.json"), b, 0644)

	est, cap, actual := loadBudgetSnapshot(dir)
	if est != 0 || actual != 0 {
		t.Errorf("stale month should drop totals: est=%d actual=%.2f", est, actual)
	}
	if cap != 500 {
		t.Errorf("cap should still come from env: %d", cap)
	}
}

func TestCacheHitPct(t *testing.T) {
	g := &budgetGuard{}
	if _, ok := g.cacheHitPct(); ok {
		t.Error("no usage yet → cacheHitPct must report ok=false, not a meaningless 0%")
	}
	g.state.CacheReadTokens = 900
	g.state.FreshInputTokens = 100
	g.state.CachedPrefixCalls = 3
	pct, ok := g.cacheHitPct()
	if !ok || pct != 90 {
		t.Errorf("900 read / 100 fresh → want 90%% ok, got %.1f%% ok=%v", pct, ok)
	}
	if s := g.cacheSuffix(); !strings.Contains(s, "cache 90% hit") {
		t.Errorf("suffix = %q, want it to contain the hit clause", s)
	}
}

// TestCacheSuffixSeparatesUnexercisedFromBroken is the distinction August
// lacked. Reads at zero against non-zero fresh input has two readings —
// the breakpoint failed, or no call carrying one was ever made — and the
// budget line reported "cache 0% hit" for both. It was the second, for a
// whole month, and the metric could not say so.
func TestCacheSuffixSeparatesUnexercisedFromBroken(t *testing.T) {
	unexercised := &budgetGuard{}
	unexercised.state.FreshInputTokens = 50_000
	got := unexercised.cacheSuffix()
	if strings.Contains(got, "0% hit") {
		t.Errorf("a month with no prefixed calls must not report a hit rate: %q", got)
	}
	if !strings.Contains(got, "no cached-prefix calls") {
		t.Errorf("suffix = %q, want it to say no prefixed calls were made", got)
	}

	// Same token counts, but calls carrying a breakpoint were made and did not
	// cache. That IS a regression, and it must read as one.
	broken := &budgetGuard{}
	broken.state.FreshInputTokens = 50_000
	broken.state.CachedPrefixCalls = 12
	got = broken.cacheSuffix()
	if !strings.Contains(got, "0% hit") {
		t.Errorf("12 prefixed calls with no reads is a real 0%%: %q", got)
	}
}

// TestCostCents_CacheWritesCostDoubleAtOneHourTTL pins the rate that was
// missing entirely until 2026-08-22. Anthropic bills the establishing write by
// TTL — 1.25x base at 5 minutes, 2x at one hour — and every cache_control in
// this repo uses the 1h breakpoint. Pricing it at the 5m rate, or not at all,
// understates: with reads sitting at 0 for all of August, the write premium was
// the entire cost of the caching attempt and none of it was counted.
func TestCostCents_CacheWritesCostDoubleAtOneHourTTL(t *testing.T) {
	plain, _ := costCents("claude-sonnet-4-5", tokenUsage{Input: 1_000_000})
	write, _ := costCents("claude-sonnet-4-5", tokenUsage{CacheWrite: 1_000_000})
	read, _ := costCents("claude-sonnet-4-5", tokenUsage{CacheRead: 1_000_000})

	if write < 599.9 || write > 600.1 {
		t.Errorf("1M sonnet cache-write = %.2f cents, want ~600 (2x the 300 base)", write)
	}
	if write <= plain {
		t.Errorf("a cache write must cost MORE than plain input, not less: write=%.2f plain=%.2f", write, plain)
	}
	// The whole economic argument for caching in one assertion: a write costs
	// 2x and a read 0.1x, so against paying 1x every call the breakpoint is
	// behind by one full prefix after the write and claws back 0.9 per read.
	// Break-even lands just past two reads — three calls in a TTL window is
	// already 27% cheaper, four is 43%.
	if read >= plain {
		t.Errorf("cache read should be far cheaper than plain input: read=%.2f plain=%.2f", read, plain)
	}
}

// TestChargeActualRecordsCacheWrites guards the counter itself. A write that is
// billed but not recorded is the shape of the original defect: spend leaves the
// account and the telemetry says nothing happened.
func TestChargeActualRecordsCacheWrites(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("METABOLISM_BUDGET_CENTS", "10000")
	g := loadBudgetGuard(dir)
	g.chargeActual("claude-sonnet-4-5", tokenUsage{Input: 10, CacheRead: 20, CacheWrite: 30, Output: 40})

	if g.state.CacheWriteTokens != 30 {
		t.Errorf("CacheWriteTokens = %d, want 30", g.state.CacheWriteTokens)
	}
	if g.state.CacheReadTokens != 20 {
		t.Errorf("CacheReadTokens = %d, want 20", g.state.CacheReadTokens)
	}
	if reloaded := loadBudgetGuard(dir); reloaded.state.CacheWriteTokens != 30 {
		t.Errorf("after reload CacheWriteTokens = %d, want 30 — not persisted", reloaded.state.CacheWriteTokens)
	}
}

// TestPacedAllowanceFloorsOnDayOne is the boundary a naive cap×elapsed gets
// wrong. At 00:00 on the 1st the month has earned nothing, so every phase would
// be refused — reproducing the exact silent idling pacing exists to remove, on
// a different date and lasting hours instead of weeks.
func TestPacedAllowanceFloorsOnDayOne(t *testing.T) {
	loc := time.UTC
	midnightFirst := time.Date(2026, 3, 1, 0, 0, 0, 0, loc)
	got := pacedAllowanceCents(3100, midnightFirst)
	if got <= 0 {
		t.Fatalf("allowance at 00:00 on the 1st = %d — the loop is dead on day one", got)
	}
	// March has 31 days, so one day's slice of 3100¢ is 100¢.
	if got != 100 {
		t.Errorf("day-one allowance = %d¢, want one day's slice of 100¢", got)
	}
}

// TestPacedAllowanceGrowsAcrossTheMonth pins the shape: roughly linear, and
// reaching the whole cap by the end rather than stranding part of it.
func TestPacedAllowanceGrowsAcrossTheMonth(t *testing.T) {
	const cap = 3100 // 100¢/day in a 31-day month
	for _, tc := range []struct {
		day, want int
	}{
		{1, 100},   // floored
		{2, 100},   // one day elapsed
		{16, 1500}, // mid-month
		{31, 3000}, // last day, before its own 24h have elapsed
	} {
		got := pacedAllowanceCents(cap, time.Date(2026, 3, tc.day, 0, 0, 0, 0, time.UTC))
		if got != tc.want {
			t.Errorf("day %d: allowance = %d¢, want %d¢", tc.day, got, tc.want)
		}
	}
}

// TestAllowSeparatesPacedFromExhausted keeps the two refusals distinguishable
// in the journal. Reading three weeks of "only 0¢ remaining" as an ordinary
// quiet hour is how the loop stayed dead for eighteen days unnoticed.
func TestAllowSeparatesPacedFromExhausted(t *testing.T) {
	const cap = 3100
	t.Setenv("METABOLISM_BUDGET_CENTS", "3100")
	g := loadBudgetGuard(t.TempDir())

	// Derived from today's allowance rather than hardcoded: a fixed 3000¢ would
	// sit above the earned share for most of a month and below it on the 31st,
	// so the test would pass all month and fail on one day.
	earned := pacedAllowanceCents(cap, time.Now())
	if earned >= cap {
		t.Fatalf("earned allowance %d¢ is the whole cap — no headroom left to test pacing against", earned)
	}

	// Spent past today's share but still under the cap: deferred, not out.
	g.state.SpentCents = earned + 1
	ok, reason := g.allow("trip", 15)
	if ok {
		t.Errorf("3000¢ spent should be ahead of pace early in the month: %s", reason)
	}
	if !strings.Contains(reason, "ahead of pace") {
		t.Errorf("a paced deferral must say so, not read as exhaustion: %q", reason)
	}

	// Genuinely past the cap: a different fact, and it must read differently.
	g.state.SpentCents = cap
	ok, reason = g.allow("trip", 15)
	if ok {
		t.Errorf("at the cap nothing should be allowed: %s", reason)
	}
	if !strings.Contains(reason, "month exhausted") {
		t.Errorf("an exhausted month must say so: %q", reason)
	}
}

// TestPacingWouldHaveKeptAugustAlive is the regression in the terms of the
// failure. August spent 990¢ of a 1000¢ cap by the 13th and then idled. Under
// pacing that spend is above the earned allowance on the 13th, so the phases
// defer rather than draining the month — and on the 25th there is still
// allowance left, which is the property that was missing.
func TestPacingWouldHaveKeptAugustAlive(t *testing.T) {
	const cap = 1000
	on13th := pacedAllowanceCents(cap, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC))
	if on13th >= 990 {
		t.Errorf("allowance on the 13th = %d¢ — pacing would not have stopped the burn that killed August", on13th)
	}
	on25th := pacedAllowanceCents(cap, time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC))
	if on25th <= on13th {
		t.Errorf("allowance must keep growing: 13th=%d¢ 25th=%d¢", on13th, on25th)
	}
	t.Logf("1000¢ cap: %d¢ earned by the 13th, %d¢ by the 25th", on13th, on25th)
}
