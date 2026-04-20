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
	if !strings.Contains(g2.summary(), "13¢ / 200¢") {
		t.Errorf("summary should show spent/cap: %q", g2.summary())
	}
}

func TestCurrentMonth_Format(t *testing.T) {
	got := currentMonth()
	if _, err := time.Parse("2006-01", got); err != nil {
		t.Errorf("currentMonth() = %q, not parseable as YYYY-MM: %v", got, err)
	}
}
