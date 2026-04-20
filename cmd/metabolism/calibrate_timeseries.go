package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// calibrationTimeseriesSchema is bumped whenever CalibrationRow's shape
// changes in a way downstream readers (--calibrate-trend, Gas Town, etc.)
// need to notice. Additive field changes can keep the same schema version;
// removed / renamed fields or semantic changes must bump it.
const calibrationTimeseriesSchema = 1

// CalibrationRow is one point in the .metabolism-calibration.jsonl time
// series. Written append-only by runCalibrate; read by --calibrate-trend
// (and eventually by phase self-gating logic that wants to see the
// trajectory instead of just the current snapshot). Designed so Gas Town
// formulas can decide when a phase is worth firing without re-running
// calibrate.
type CalibrationRow struct {
	Timestamp     time.Time `json:"timestamp"`
	SchemaVersion int       `json:"schema_version"`

	TotalCycles       int     `json:"total_cycles"`
	UsefulSignalPct   float64 `json:"useful_signal_pct"` // (corroborated_novel + challenged_novel) / total_cycles * 100
	ChallengedCount   int     `json:"challenged_count"`  // all challenged resolutions
	CorroboratedCount int     `json:"corroborated_count"`
	ChallengedNovel   int     `json:"challenged_novel"`
	CorroboratedNovel int     `json:"corroborated_novel"`

	HHI                 float64            `json:"hhi"`                    // provenance source concentration (0..1)
	PerBackendSignal    map[string]float64 `json:"per_backend_signal"`     // backend → signal_rate as percentage
	BiasTriggers        []string           `json:"bias_triggers"`          // names of triggered auditors
	SurvivorshipRatio   float64            `json:"survivorship_ratio"`     // irrelevant:challenged ratio; high = default-to-irrelevant overfit
	HoursSinceLastSense float64            `json:"hours_since_last_sense"` // for phase self-gating: "no sense in > N hours" → fire sense
}

// appendCalibrationRow appends one JSONL line to .metabolism-calibration.jsonl.
// Append-only; never rewrites. If the file doesn't exist it's created.
func appendCalibrationRow(dir string, row CalibrationRow) error {
	path := filepath.Join(dir, ".metabolism-calibration.jsonl")
	b, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// computeCalibrationRow builds a CalibrationRow from the metabolism log
// and the bias audit. Kept as a pure function of (mlog, biasReport,
// gapCounts, resolutions) so tests can exercise the computation without
// a real corpus.
//
// corrobNovel / challNovel come from the gap_confirmed scan already
// computed by runCalibrate (mixed_overlap or gap_confirmed). corrobTotal
// and challTotal come from the resolution counts. backendTotals maps
// backend → (withSignal, total).
func computeCalibrationRow(
	mlog MetabolismLog,
	biasAuditors []BiasAuditorResult,
	corrobTotal, corrobNovel, challTotal, challNovel int,
	backendTotals map[string]struct{ WithSignal, Total int },
) CalibrationRow {
	row := CalibrationRow{
		Timestamp:         time.Now(),
		SchemaVersion:     calibrationTimeseriesSchema,
		TotalCycles:       len(mlog.Cycles),
		ChallengedCount:   challTotal,
		CorroboratedCount: corrobTotal,
		ChallengedNovel:   challNovel,
		CorroboratedNovel: corrobNovel,
		PerBackendSignal:  map[string]float64{},
	}
	if row.TotalCycles > 0 {
		row.UsefulSignalPct = float64(corrobNovel+challNovel) / float64(row.TotalCycles) * 100
	}
	for be, t := range backendTotals {
		if t.Total > 0 {
			row.PerBackendSignal[be] = float64(t.WithSignal) / float64(t.Total) * 100
		}
	}
	for _, a := range biasAuditors {
		switch a.Bias {
		case "AvailabilityHeuristic":
			row.HHI = a.Value
		case "SurvivorshipBias":
			row.SurvivorshipRatio = a.Value
		}
		if a.Triggered {
			row.BiasTriggers = append(row.BiasTriggers, a.Bias)
		}
	}
	// hours_since_last_sense: time since the most-recent sensor cycle
	// (any backend). Cycles are appended in chronological order in
	// production but we scan defensively in case of manual edits.
	var latest time.Time
	for _, c := range mlog.Cycles {
		if c.Timestamp.After(latest) {
			latest = c.Timestamp
		}
	}
	if !latest.IsZero() {
		row.HoursSinceLastSense = time.Since(latest).Hours()
	}
	return row
}
