package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// phaseGateConfig holds the thresholds that decide whether each
// expensive phase of --evolve fires on a given Gas Town clock tick.
// Zero values disable the gate (phase always fires, legacy behavior).
// Defaults come from defaultPhaseGates; CLI flags override.
type phaseGateConfig struct {
	SenseMinHours         float64 // no sensor cycle in > N hours → allow sense
	ResolveMinUnresolved  int     // ≥N hypotheses have 3+ unresolved signal cycles → allow resolve
	TripMinHours          float64 // no trip in > N hours → allow trip
	IngestMinCorroborated int     // ≥N corroborated cycles → allow ingest
}

// defaultPhaseGates picks thresholds that match the current KB cadence
// and Gas Town's expected hourly formula. Tuned for a ~300-cycle corpus
// with ~$10/mo budget — expensive phases fire every few ticks, not
// every tick, keeping average cost well below ceiling.
func defaultPhaseGates() phaseGateConfig {
	return phaseGateConfig{
		SenseMinHours:         4.0,
		ResolveMinUnresolved:  3,
		TripMinHours:          24.0,
		IngestMinCorroborated: 1,
	}
}

// gateDecision packages a fire/skip decision plus a human-readable reason
// so the cycle log records why a phase did or didn't run.
type gateDecision struct {
	Fire   bool
	Reason string
}

func (d gateDecision) String() string {
	if d.Fire {
		return "ALLOW: " + d.Reason
	}
	return "SKIP: " + d.Reason
}

// shouldFireSense decides whether the sense phase should fire this tick.
// Sense is expensive because it hits Kagi (~$0.025/query × N targets)
// and rate-limits arXiv. Fires when the last cycle is older than the
// configured freshness window. Ignores zero-signal cycles? No — a
// zero-paper sensor still costs API credits, so any recent cycle
// (signal or not) resets the clock.
func shouldFireSense(cfg phaseGateConfig, mlog MetabolismLog, now time.Time) gateDecision {
	if cfg.SenseMinHours <= 0 {
		return gateDecision{Fire: true, Reason: "gate disabled (sense-min-hours<=0)"}
	}
	var latest time.Time
	for _, c := range mlog.Cycles {
		if c.Timestamp.After(latest) {
			latest = c.Timestamp
		}
	}
	if latest.IsZero() {
		return gateDecision{Fire: true, Reason: "no prior sensor cycles"}
	}
	hours := now.Sub(latest).Hours()
	if hours >= cfg.SenseMinHours {
		return gateDecision{Fire: true, Reason: fmt.Sprintf("last sense %.1fh ago ≥ %.1fh", hours, cfg.SenseMinHours)}
	}
	return gateDecision{Fire: false, Reason: fmt.Sprintf("last sense %.1fh ago < %.1fh (set --sense-min-hours=0 to force)", hours, cfg.SenseMinHours)}
}

// shouldFireResolve decides whether the resolve phase should fire.
// Resolve calls Sonnet per hypothesis with 3+ signal cycles. Gating on
// "at least N hypotheses with 3+ unresolved signal cycles" avoids
// spinning up Sonnet for a single resolvable item when the batch is
// tiny. A hypothesis is resolvable when PapersFound>0 and Resolution=""
// and the count of such cycles is ≥3.
func shouldFireResolve(cfg phaseGateConfig, mlog MetabolismLog) gateDecision {
	if cfg.ResolveMinUnresolved <= 0 {
		return gateDecision{Fire: true, Reason: "gate disabled (resolve-min-unresolved<=0)"}
	}
	// Count unresolved signal cycles per hypothesis.
	byHyp := map[string]int{}
	for _, c := range mlog.Cycles {
		if c.Resolution == "" && c.PapersFound > 0 {
			byHyp[c.Hypothesis]++
		}
	}
	// Count hypotheses crossing the resolvable threshold (3 signal cycles).
	resolvable := 0
	for _, n := range byHyp {
		if n >= 3 {
			resolvable++
		}
	}
	if resolvable >= cfg.ResolveMinUnresolved {
		return gateDecision{Fire: true, Reason: fmt.Sprintf("%d hypotheses have ≥3 unresolved signal cycles (threshold: %d)", resolvable, cfg.ResolveMinUnresolved)}
	}
	return gateDecision{Fire: false, Reason: fmt.Sprintf("%d resolvable hypotheses < %d threshold (set --resolve-min-unresolved=0 to force)", resolvable, cfg.ResolveMinUnresolved)}
}

// shouldFireTrip decides whether the trip (REM speculation) phase fires.
// Trip is cheap per-call but has diminishing returns on rapid cadence
// (same clusters re-sampled). Gates on time since the most recent
// metabolism_cycle*.go file (the artifact trip produces when it promotes).
// When no trip has ever run, fires unconditionally.
func shouldFireTrip(cfg phaseGateConfig, dir string, now time.Time) gateDecision {
	if cfg.TripMinHours <= 0 {
		return gateDecision{Fire: true, Reason: "gate disabled (trip-min-hours<=0)"}
	}
	last, err := lastTripFileTime(dir)
	if err != nil {
		return gateDecision{Fire: true, Reason: fmt.Sprintf("could not read corpus mtimes (%v) — erring on fire", err)}
	}
	if last.IsZero() {
		return gateDecision{Fire: true, Reason: "no metabolism_cycle*.go files yet (first trip)"}
	}
	hours := now.Sub(last).Hours()
	if hours >= cfg.TripMinHours {
		return gateDecision{Fire: true, Reason: fmt.Sprintf("last trip artifact %.1fh ago ≥ %.1fh", hours, cfg.TripMinHours)}
	}
	return gateDecision{Fire: false, Reason: fmt.Sprintf("last trip artifact %.1fh ago < %.1fh (set --trip-min-hours=0 to force)", hours, cfg.TripMinHours)}
}

// shouldFireIngest decides whether the pipeline ingest phase fires.
// Pipeline is bounded by --llm-budget already, so this gate is
// intentionally permissive — it mostly skips ingest when there's
// nothing corroborated to ingest.
//
// Cycles from any pipeline-supported backend are counted: ZIM (full
// article text), Kagi (search snippet), and arXiv (abstract). RSS and
// legacy empty-backend cycles are excluded because runIngest won't
// process them. This keeps the gate prediction in sync with the
// phase's actual behavior — an earlier unconditional count let the
// gate ALLOW cycles runIngest would silently drop.
func shouldFireIngest(cfg phaseGateConfig, mlog MetabolismLog) gateDecision {
	if cfg.IngestMinCorroborated <= 0 {
		return gateDecision{Fire: true, Reason: "gate disabled (ingest-min-corroborated<=0)"}
	}
	supported := map[string]bool{"zim": true, "kagi": true, "arxiv": true}
	n := 0
	for _, c := range mlog.Cycles {
		if c.Resolution != "corroborated" || c.Ingested || c.PapersFound == 0 {
			continue
		}
		if !supported[c.Backend] {
			continue
		}
		n++
	}
	if n >= cfg.IngestMinCorroborated {
		return gateDecision{Fire: true, Reason: fmt.Sprintf("%d corroborated-but-uningested ingestable cycles (threshold: %d)", n, cfg.IngestMinCorroborated)}
	}
	return gateDecision{Fire: false, Reason: fmt.Sprintf("%d corroborated-but-uningested ingestable cycles < %d (set --ingest-min-corroborated=0 to force)", n, cfg.IngestMinCorroborated)}
}

// lastTripFileTime returns the mtime of the most-recent metabolism_cycle*.go
// file in dir, or zero time if none exist. Trip is the only producer of
// those files, so their mtime is a reliable "last trip" signal.
func lastTripFileTime(dir string) (time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, err
	}
	var latest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "metabolism_cycle") || !strings.HasSuffix(name, ".go") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest, nil
}

// logGate prints a phase gate decision in a consistent format so Gas
// Town logs can grep for which phases fired and why.
func logGate(phase string, d gateDecision) {
	fmt.Printf("[gate] %s: %s\n", phase, d)
}

// phaseGatesFromFlags builds a phaseGateConfig from parsed flag pointers.
// Separate helper so main() stays readable even with the flag count growing.
func phaseGatesFromFlags(senseHrs *float64, resolveN *int, tripHrs *float64, ingestN *int) phaseGateConfig {
	return phaseGateConfig{
		SenseMinHours:         *senseHrs,
		ResolveMinUnresolved:  *resolveN,
		TripMinHours:          *tripHrs,
		IngestMinCorroborated: *ingestN,
	}
}

