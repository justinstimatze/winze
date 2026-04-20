package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunCalibrateTrend_NoFile(t *testing.T) {
	// No file → friendly message, no crash.
	dir := t.TempDir()
	// Just ensure it runs without panicking; output goes to stdout.
	runCalibrateTrend(dir, false)
}

func TestRunCalibrateTrend_HandlesMalformedLine(t *testing.T) {
	dir := t.TempDir()
	// Mix one valid row with one corrupt line; expect no crash, valid row read.
	content := `{"timestamp":"2026-04-19T20:44:37Z","schema_version":1,"total_cycles":314,"useful_signal_pct":9.2,"hhi":0.58,"challenged_count":2,"corroborated_count":28,"per_backend_signal":{"kagi":100},"bias_triggers":["AvailabilityHeuristic"],"survivorship_ratio":94.5,"hours_since_last_sense":6.3}
this line is not valid json {{{
{"timestamp":"2026-04-19T21:00:00Z","schema_version":1,"total_cycles":315,"useful_signal_pct":9.5,"hhi":0.59,"challenged_count":2,"corroborated_count":29,"per_backend_signal":{"kagi":100},"bias_triggers":["AvailabilityHeuristic"],"survivorship_ratio":94.5,"hours_since_last_sense":7.0}
`
	if err := os.WriteFile(filepath.Join(dir, ".metabolism-calibration.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	runCalibrateTrend(dir, false) // smoke test: should not crash
}

func TestRunCalibrateTrend_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	row := CalibrationRow{
		Timestamp:         time.Now(),
		SchemaVersion:     1,
		TotalCycles:       10,
		UsefulSignalPct:   12.5,
		HHI:               0.45,
		ChallengedCount:   1,
		CorroboratedCount: 5,
	}
	if err := appendCalibrationRow(dir, row); err != nil {
		t.Fatal(err)
	}
	// Smoke test: --json doesn't crash.
	runCalibrateTrend(dir, true)
}

func TestShortenBiasNames(t *testing.T) {
	in := []string{"AvailabilityHeuristic", "SurvivorshipBias", "UnknownBiasZZZ"}
	got := shortenBiasNames(in)
	want := []string{"AvailHeur", "SurvBias", "UnknownBiasZZZ"} // unknowns pass through
	if len(got) != len(want) {
		t.Fatalf("len mismatch: %v vs %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}
