package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runCalibrateTrend reads .metabolism-calibration.jsonl and prints a
// compact time series so the user (or a Gas Town formula) can see how
// useful signal, provenance concentration, and bias triggers are
// trending over time. JSON-lines parsing tolerates a trailing newline
// and skips blank/malformed lines (logging them to stderr) so a single
// corrupt row doesn't stop trend output.
func runCalibrateTrend(dir string, jsonOut bool) {
	path := filepath.Join(dir, ".metabolism-calibration.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("[trend] no calibration history yet — run `go run ./cmd/metabolism --calibrate .` to seed the file")
			return
		}
		fmt.Fprintf(os.Stderr, "[trend] open %s: %v\n", path, err)
		return
	}
	defer f.Close()

	var rows []CalibrationRow
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MiB lines
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row CalibrationRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			fmt.Fprintf(os.Stderr, "[trend] line %d unparseable, skipping: %v\n", lineNo, err)
			continue
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[trend] scan: %v\n", err)
	}
	if len(rows) == 0 {
		fmt.Println("[trend] file exists but contains no usable rows")
		return
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
		return
	}

	fmt.Printf("[trend] %s — %d rows\n\n", path, len(rows))
	// Header. Columns chosen to match what the brief calls out as worth
	// trending: useful-signal trajectory, HHI trajectory, challenge count
	// trajectory, survivorship and time-since-sense, plus month-to-date
	// actual spend so unattended runs are observable.
	fmt.Printf("%-19s  %-7s  %-5s  %-5s  %-7s  %-9s  %-7s  %-15s  %s\n",
		"timestamp", "useful%", "hhi", "chall", "corrob", "actual$", "surv", "spend(est/cap)", "bias_triggers")
	fmt.Println(strings.Repeat("-", 115))
	for _, r := range rows {
		ts := r.Timestamp.Local().Format("2006-01-02 15:04")
		triggers := strings.Join(shortenBiasNames(r.BiasTriggers), ",")
		if triggers == "" {
			triggers = "(none)"
		}
		actualDollars := r.ActualSpentCents / 100.0
		spendEstCap := "—"
		if r.BudgetCapCents > 0 {
			spendEstCap = fmt.Sprintf("%d¢/%d¢", r.EstSpentCents, r.BudgetCapCents)
		} else if r.EstSpentCents > 0 {
			spendEstCap = fmt.Sprintf("%d¢/—", r.EstSpentCents)
		}
		fmt.Printf("%-19s  %6.1f%%  %5.2f  %5d  %6d  $%8.4f  %6.1f  %-15s  %s\n",
			ts, r.UsefulSignalPct, r.HHI, r.ChallengedCount, r.CorroboratedCount,
			actualDollars, r.SurvivorshipRatio, spendEstCap, triggers)
	}

	// Delta line: useful%, HHI, survivorship, and actual-spend change
	// between first and last row. The whole point of the trajectory is
	// "is this improving and what did it cost?" so make both visible
	// without math by hand.
	if len(rows) >= 2 {
		first := rows[0]
		last := rows[len(rows)-1]
		fmt.Println()
		fmt.Printf("[trend] delta first→last: useful %+.1fpp, HHI %+.2f, survivorship %+.1f, challenged %+d, corroborated %+d, actual $%+.4f\n",
			last.UsefulSignalPct-first.UsefulSignalPct,
			last.HHI-first.HHI,
			last.SurvivorshipRatio-first.SurvivorshipRatio,
			last.ChallengedCount-first.ChallengedCount,
			last.CorroboratedCount-first.CorroboratedCount,
			(last.ActualSpentCents-first.ActualSpentCents)/100.0,
		)
	}
}

// shortenBiasNames produces compact labels for the trend table so a row
// with multiple triggers still fits a terminal line.
func shortenBiasNames(names []string) []string {
	short := map[string]string{
		"AvailabilityHeuristic": "AvailHeur",
		"SurvivorshipBias":      "SurvBias",
		"ConfirmationBias":      "Confirm",
		"AnchoringBias":         "Anchor",
		"ClusteringIllusion":    "Cluster",
		"FramingEffect":         "Framing",
		"DunningKruger":         "DK",
		"BaseRateNeglect":       "BaseRate",
		"PrematureClosure":      "PreClose",
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if s, ok := short[n]; ok {
			out = append(out, s)
		} else {
			out = append(out, n)
		}
	}
	return out
}
