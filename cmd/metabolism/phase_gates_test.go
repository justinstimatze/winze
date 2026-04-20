package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShouldFireSense_Disabled(t *testing.T) {
	cfg := phaseGateConfig{SenseMinHours: 0}
	d := shouldFireSense(cfg, MetabolismLog{}, time.Now())
	if !d.Fire {
		t.Errorf("gate disabled should always fire, got %v", d)
	}
	if !strings.Contains(d.Reason, "disabled") {
		t.Errorf("reason should mention 'disabled': %q", d.Reason)
	}
}

func TestShouldFireSense_NoPriorCycles(t *testing.T) {
	cfg := phaseGateConfig{SenseMinHours: 4}
	d := shouldFireSense(cfg, MetabolismLog{}, time.Now())
	if !d.Fire {
		t.Errorf("empty log should fire (no prior sense), got %v", d)
	}
}

func TestShouldFireSense_RecentCycle_Skips(t *testing.T) {
	cfg := phaseGateConfig{SenseMinHours: 4}
	now := time.Now()
	mlog := MetabolismLog{Cycles: []Cycle{{Timestamp: now.Add(-1 * time.Hour)}}}
	d := shouldFireSense(cfg, mlog, now)
	if d.Fire {
		t.Errorf("1h ago < 4h threshold — should skip, got fire: %v", d)
	}
}

func TestShouldFireSense_OldCycle_Fires(t *testing.T) {
	cfg := phaseGateConfig{SenseMinHours: 4}
	now := time.Now()
	mlog := MetabolismLog{Cycles: []Cycle{{Timestamp: now.Add(-5 * time.Hour)}}}
	d := shouldFireSense(cfg, mlog, now)
	if !d.Fire {
		t.Errorf("5h ago ≥ 4h threshold — should fire, got skip: %v", d)
	}
}

func TestShouldFireResolve_BelowThreshold_Skips(t *testing.T) {
	cfg := phaseGateConfig{ResolveMinUnresolved: 2}
	// Only one hypothesis meets the 3+ unresolved signal cycle rule.
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "", PapersFound: 1},
		{Hypothesis: "A", Resolution: "", PapersFound: 1},
		{Hypothesis: "A", Resolution: "", PapersFound: 1},
		{Hypothesis: "B", Resolution: "", PapersFound: 1}, // only 1 cycle — below 3+ floor
	}}
	d := shouldFireResolve(cfg, mlog)
	if d.Fire {
		t.Errorf("1 resolvable hyp < 2 threshold — should skip, got fire: %v", d)
	}
}

func TestShouldFireResolve_MeetsThreshold_Fires(t *testing.T) {
	cfg := phaseGateConfig{ResolveMinUnresolved: 1}
	// One hypothesis with 3 unresolved signal cycles.
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "", PapersFound: 1},
		{Hypothesis: "A", Resolution: "", PapersFound: 1},
		{Hypothesis: "A", Resolution: "", PapersFound: 1},
	}}
	d := shouldFireResolve(cfg, mlog)
	if !d.Fire {
		t.Errorf("1 resolvable hyp ≥ 1 threshold — should fire, got skip: %v", d)
	}
}

func TestShouldFireResolve_IgnoresResolved(t *testing.T) {
	cfg := phaseGateConfig{ResolveMinUnresolved: 1}
	// 4 cycles but all already resolved — nothing for the resolver to do.
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "corroborated", PapersFound: 1},
		{Hypothesis: "A", Resolution: "corroborated", PapersFound: 1},
		{Hypothesis: "A", Resolution: "corroborated", PapersFound: 1},
		{Hypothesis: "A", Resolution: "corroborated", PapersFound: 1},
	}}
	d := shouldFireResolve(cfg, mlog)
	if d.Fire {
		t.Errorf("all resolved — should skip, got fire: %v", d)
	}
}

func TestShouldFireResolve_IgnoresZeroSignal(t *testing.T) {
	cfg := phaseGateConfig{ResolveMinUnresolved: 1}
	// 3 unresolved cycles but no papers — resolver has nothing to evaluate.
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "", PapersFound: 0},
		{Hypothesis: "A", Resolution: "", PapersFound: 0},
		{Hypothesis: "A", Resolution: "", PapersFound: 0},
	}}
	d := shouldFireResolve(cfg, mlog)
	if d.Fire {
		t.Errorf("zero-signal cycles — should skip, got fire: %v", d)
	}
}

func TestShouldFireTrip_NoTripFiles(t *testing.T) {
	cfg := phaseGateConfig{TripMinHours: 24}
	dir := t.TempDir()
	d := shouldFireTrip(cfg, dir, time.Now())
	if !d.Fire {
		t.Errorf("no prior trip files — should fire, got skip: %v", d)
	}
}

func TestShouldFireTrip_RecentFile_Skips(t *testing.T) {
	cfg := phaseGateConfig{TripMinHours: 24}
	dir := t.TempDir()
	// Create a metabolism_cycleN.go file with recent mtime.
	path := filepath.Join(dir, "metabolism_cycle7.go")
	if err := os.WriteFile(path, []byte("package winze\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Touch mtime to 1 hour ago.
	onehago := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(path, onehago, onehago); err != nil {
		t.Fatal(err)
	}
	d := shouldFireTrip(cfg, dir, time.Now())
	if d.Fire {
		t.Errorf("1h ago < 24h threshold — should skip, got fire: %v", d)
	}
}

func TestShouldFireTrip_OldFile_Fires(t *testing.T) {
	cfg := phaseGateConfig{TripMinHours: 24}
	dir := t.TempDir()
	path := filepath.Join(dir, "metabolism_cycle7.go")
	if err := os.WriteFile(path, []byte("package winze\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Touch mtime to 2 days ago.
	ago := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, ago, ago); err != nil {
		t.Fatal(err)
	}
	d := shouldFireTrip(cfg, dir, time.Now())
	if !d.Fire {
		t.Errorf("48h ago ≥ 24h threshold — should fire, got skip: %v", d)
	}
}

func TestShouldFireIngest_NoCorroborated_Skips(t *testing.T) {
	cfg := phaseGateConfig{IngestMinCorroborated: 1}
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "irrelevant", PapersFound: 1},
		{Hypothesis: "A", Resolution: "challenged", PapersFound: 1},
	}}
	d := shouldFireIngest(cfg, mlog)
	if d.Fire {
		t.Errorf("no corroborated-uningested — should skip, got fire: %v", d)
	}
}

func TestShouldFireIngest_HasCorroborated_Fires(t *testing.T) {
	cfg := phaseGateConfig{IngestMinCorroborated: 1}
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "corroborated", PapersFound: 1, Ingested: false},
	}}
	d := shouldFireIngest(cfg, mlog)
	if !d.Fire {
		t.Errorf("1 corroborated-uningested ≥ 1 threshold — should fire, got skip: %v", d)
	}
}

func TestShouldFireIngest_AlreadyIngested_Skips(t *testing.T) {
	cfg := phaseGateConfig{IngestMinCorroborated: 1}
	mlog := MetabolismLog{Cycles: []Cycle{
		{Hypothesis: "A", Resolution: "corroborated", PapersFound: 1, Ingested: true},
	}}
	d := shouldFireIngest(cfg, mlog)
	if d.Fire {
		t.Errorf("already ingested — should skip, got fire: %v", d)
	}
}
