package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// logTripLintDurability is the first non-tautological resolution path.
// For each trip-promoted claim var, it runs cmd/lint and records whether
// lint flagged the var, writing one Cycle per var to .metabolism-log.json.
//
// Why this matters: calibration today bottoms out in "sensor found papers",
// which is near-tautological. Lint durability is KB-internal: the substrate
// is asked whether its own consistency rules flag what trip just wrote. If
// lint exits clean, all promoted vars are confirmed; if non-zero, each var
// is scanned for in the output.
//
// Lint's own false-positive surface: contested-concept enumerates competing
// TheoryOf subjects, which is the intended behavior (new competing theory =
// good), but would substring-match a trip-promoted TheoryOf var. Guard:
// when lint exits 0, we skip the substring scan — no need to blame anyone.
// When lint exits non-zero, substring-match is honest because at least one
// rule reported a real problem.
func logTripLintDurability(dir string, claimVars []string) error {
	if len(claimVars) == 0 {
		return nil
	}

	cmd := exec.Command("go", "run", "./cmd/lint", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	lintText := string(out)

	// Exit code: 0 = clean, >0 = findings. exec.Command surfaces non-zero
	// exit as an ExitError; treat missing output as a tool failure.
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else if len(lintText) == 0 {
			return fmt.Errorf("run cmd/lint: %w", err)
		}
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)
	today := time.Now().Format("2006-01-02")
	now := time.Now()
	commit := oracleCommit(dir)
	digest := oracleDigest(dir, "trip_lint_durability")

	confirmed, refuted := 0, 0
	for _, varName := range claimVars {
		c := Cycle{
			Timestamp:      now,
			Hypothesis:     varName,
			Prediction:     "trip-promoted claim will not be flagged by cmd/lint",
			VulnType:       "trip_promotion",
			PredictionType: "trip_lint_durability",
			ResolvedAt:     today,
			OracleCommit:   commit,
			OracleDigest:   digest,
		}
		if exitCode == 0 {
			c.Resolution = "confirmed"
			c.Evidence = "cmd/lint exited 0 — no findings on any rule"
			confirmed++
		} else if line := findLintEvidence(lintText, varName); line != "" {
			c.Resolution = "refuted"
			c.Evidence = line
			refuted++
		} else {
			c.Resolution = "confirmed"
			c.Evidence = fmt.Sprintf("cmd/lint exited %d but %s not mentioned in any finding", exitCode, varName)
			confirmed++
		}
		mlog.Cycles = append(mlog.Cycles, c)
	}

	if err := saveLog(logPath, mlog); err != nil {
		return fmt.Errorf("save log: %w", err)
	}

	fmt.Printf("[trip-promote] lint-durability: %d confirmed, %d refuted (logged as trip_lint_durability)\n", confirmed, refuted)
	return nil
}

// tripPromotionAttempt records one would-be trip promotion and whether it
// made it into the corpus. Skipped attempts carry the reason.
type tripPromotionAttempt struct {
	Name     string
	Accepted bool
	Reason   string // accepted | entity_not_found | type_mismatch
	Evidence string
}

// logTripPromotionAttempts writes one Cycle per promotion attempt with
// prediction_type=trip_promotion_attempt. Accepted attempts resolve as
// "confirmed", skipped attempts as "refuted" with the specific reason in
// Evidence. This makes the promotion funnel legible in --calibrate:
// hit rate = attempts that survived all validation gates, and the Reason
// codes break down where we lose candidates.
func logTripPromotionAttempts(dir string, attempts []tripPromotionAttempt) error {
	if len(attempts) == 0 {
		return nil
	}
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)
	today := time.Now().Format("2006-01-02")
	now := time.Now()
	commit := oracleCommit(dir)
	digest := oracleDigest(dir, "trip_promotion_attempt")

	for _, a := range attempts {
		c := Cycle{
			Timestamp:      now,
			Hypothesis:     a.Name,
			Prediction:     "trip connection passes entity-existence and predicate-slot validation",
			VulnType:       "trip_promotion",
			PredictionType: "trip_promotion_attempt",
			ResolvedAt:     today,
			Evidence:       a.Evidence,
			OracleCommit:   commit,
			OracleDigest:   digest,
		}
		if a.Accepted {
			c.Resolution = "confirmed"
		} else {
			c.Resolution = "refuted"
		}
		mlog.Cycles = append(mlog.Cycles, c)
	}

	if err := saveLog(logPath, mlog); err != nil {
		return fmt.Errorf("save log: %w", err)
	}
	return nil
}

// findLintEvidence returns the first lint output line mentioning varName,
// trimmed. Returns "" if not found. Lint prints var names verbatim inside
// finding lines; an exact substring match suffices because trip claim vars
// are long and specific (e.g. "TripCycle8ConsciousnessBelongsToHumanCognition").
func findLintEvidence(lintText, varName string) string {
	for _, line := range strings.Split(lintText, "\n") {
		if strings.Contains(line, varName) {
			trim := strings.TrimSpace(line)
			if len(trim) > 240 {
				trim = trim[:240] + "..."
			}
			return trim
		}
	}
	return ""
}
