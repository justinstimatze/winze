package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeCalibrationRow(t *testing.T) {
	now := time.Now()
	mlog := MetabolismLog{
		Cycles: []Cycle{
			{Timestamp: now.Add(-3 * time.Hour), Backend: "arxiv", PapersFound: 0},
			{Timestamp: now.Add(-2 * time.Hour), Backend: "kagi", PapersFound: 5},
			{Timestamp: now.Add(-1 * time.Hour), Backend: "zim", PapersFound: 2},
		},
	}
	auditors := []BiasAuditorResult{
		{Bias: "AvailabilityHeuristic", Value: 0.58, Triggered: true},
		{Bias: "SurvivorshipBias", Value: 94.5, Triggered: true},
		{Bias: "ConfirmationBias", Value: 0.58, Triggered: false}, // not triggered — excluded from bias_triggers
	}
	backendTotals := map[string]struct{ WithSignal, Total int }{
		"arxiv": {WithSignal: 0, Total: 1},
		"kagi":  {WithSignal: 1, Total: 1},
		"zim":   {WithSignal: 1, Total: 1},
	}
	// 3 total cycles; 1 corroborated_novel, 0 challenged_novel → useful = 1/3 = 33.33%
	row := computeCalibrationRow(mlog, auditors,
		/*corrobTotal*/ 2, /*corrobNovel*/ 1,
		/*challTotal*/ 0, /*challNovel*/ 0,
		backendTotals)

	if row.SchemaVersion != calibrationTimeseriesSchema {
		t.Errorf("SchemaVersion = %d, want %d", row.SchemaVersion, calibrationTimeseriesSchema)
	}
	if row.TotalCycles != 3 {
		t.Errorf("TotalCycles = %d, want 3", row.TotalCycles)
	}
	if row.UsefulSignalPct < 33.32 || row.UsefulSignalPct > 33.34 {
		t.Errorf("UsefulSignalPct = %f, want ~33.33", row.UsefulSignalPct)
	}
	if row.HHI != 0.58 {
		t.Errorf("HHI = %f, want 0.58", row.HHI)
	}
	if row.SurvivorshipRatio != 94.5 {
		t.Errorf("SurvivorshipRatio = %f, want 94.5", row.SurvivorshipRatio)
	}
	if len(row.BiasTriggers) != 2 {
		t.Errorf("BiasTriggers len = %d, want 2 (confirmation_bias not triggered, excluded)", len(row.BiasTriggers))
	}
	if row.PerBackendSignal["kagi"] != 100 {
		t.Errorf("kagi signal = %f, want 100", row.PerBackendSignal["kagi"])
	}
	if row.PerBackendSignal["arxiv"] != 0 {
		t.Errorf("arxiv signal = %f, want 0", row.PerBackendSignal["arxiv"])
	}
	// hours_since_last_sense: last cycle was 1h ago
	if row.HoursSinceLastSense < 0.99 || row.HoursSinceLastSense > 1.01 {
		t.Errorf("HoursSinceLastSense = %f, want ~1.0", row.HoursSinceLastSense)
	}
}

func TestComputeCalibrationRow_EmptyLog(t *testing.T) {
	// Empty log → zero cycles, zero signal, no panic.
	row := computeCalibrationRow(
		MetabolismLog{},
		nil,
		0, 0, 0, 0,
		map[string]struct{ WithSignal, Total int }{},
	)
	if row.TotalCycles != 0 {
		t.Errorf("TotalCycles = %d, want 0", row.TotalCycles)
	}
	if row.UsefulSignalPct != 0 {
		t.Errorf("UsefulSignalPct = %f, want 0 (no divide-by-zero)", row.UsefulSignalPct)
	}
	if row.HoursSinceLastSense != 0 {
		t.Errorf("HoursSinceLastSense = %f, want 0 when no cycles", row.HoursSinceLastSense)
	}
}

func TestAppendCalibrationRow_AppendsJSONL(t *testing.T) {
	// Write two rows; verify both survive and parse as separate JSON objects.
	dir := t.TempDir()
	r1 := CalibrationRow{Timestamp: time.Now(), SchemaVersion: 1, TotalCycles: 10, UsefulSignalPct: 15}
	r2 := CalibrationRow{Timestamp: time.Now(), SchemaVersion: 1, TotalCycles: 11, UsefulSignalPct: 18}
	if err := appendCalibrationRow(dir, r1); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := appendCalibrationRow(dir, r2); err != nil {
		t.Fatalf("second append: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, ".metabolism-calibration.jsonl"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := 0
	for _, line := range splitLines(b) {
		if len(line) == 0 {
			continue
		}
		var got CalibrationRow
		if err := json.Unmarshal(line, &got); err != nil {
			t.Errorf("unmarshal line %d: %v (line=%q)", lines, err, line)
		}
		lines++
	}
	if lines != 2 {
		t.Errorf("got %d lines, want 2", lines)
	}
}

func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, b[start:i])
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}
