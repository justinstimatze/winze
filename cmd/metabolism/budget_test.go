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
	cents, ok := costCents("claude-haiku-4-5", 1_000_000, 0, 1_000_000)
	if !ok {
		t.Fatal("expected pricing for haiku-4-5")
	}
	if cents < 599.9 || cents > 600.1 {
		t.Errorf("cents = %.4f, want ~600", cents)
	}
}

func TestCostCents_UnknownModel(t *testing.T) {
	cents, ok := costCents("claude-opus-99", 1000, 0, 1000)
	if ok {
		t.Error("expected unknown model to return ok=false")
	}
	if cents != 0 {
		t.Errorf("cents = %.4f, want 0 for unknown model", cents)
	}
}

func TestCostCents_CacheReadsAreCheaper(t *testing.T) {
	// 1M Sonnet plain input → 300¢; 1M Sonnet cache-read input → 30¢.
	plain, _ := costCents("claude-sonnet-4-5", 1_000_000, 0, 0)
	cached, _ := costCents("claude-sonnet-4-5", 0, 1_000_000, 0)
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
	g.chargeActual("claude-haiku-4-5", 100_000, 0, 100_000)
	g.chargeActual("claude-haiku-4-5", 100_000, 0, 100_000)
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
	g.chargeActual("claude-from-the-future", 1000, 0, 1000)
	if g.state.ActualSpentCents != 0 {
		t.Errorf("unknown model should not be charged: actual=%.4f", g.state.ActualSpentCents)
	}
}

func TestRecordActualUsage_NilGlobalIsNoop(t *testing.T) {
	prev := globalBudget
	t.Cleanup(func() { globalBudget = prev })
	globalBudget = nil
	// Must not panic.
	recordActualUsage("claude-haiku-4-5", 100, 0, 100)
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
