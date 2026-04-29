package main

import (
	"fmt"
	"strings"
)

// biasGates captures the set of phase-behavior overrides produced by the
// bias audit at the start of runCycle. Each field corresponds to one
// decision a downstream phase can branch on. Additional gates should be
// added here, not inlined — runCycle stays readable when bias influence
// is enumerated in one place.
type biasGates struct {
	skipZim      bool     // availability_heuristic triggered: skip Wikipedia ZIM to avoid deepening concentration
	skipSense    bool     // confirmation_bias triggered: skip sense phase to stop accumulating more corroboration-biased signal
	triggered    []string // names of triggered auditors, in audit order
	triggerNotes []string // human-readable per-trigger notes for cycle header
}

// runBiasGates runs the bias audit for the current corpus and returns
// the gates that should influence downstream phases this cycle.
// Closes the README gap "triggered bias auditors don't gate the next
// metabolism phase."
func runBiasGates(dir string) biasGates {
	fmt.Println("=== Phase 0: Bias audit ===")
	report := collectBiasResults(dir, nil, false, false)
	writeBiasState(dir, report)
	g := computeBiasGates(report.Auditors)
	if len(g.triggered) == 0 {
		fmt.Printf("[cycle] bias audit: all clear (0/%d auditors triggered)\n", len(report.Auditors))
		return g
	}
	fmt.Printf("[cycle] bias audit: %d/%d auditors triggered — %s\n",
		len(g.triggered), len(report.Auditors), strings.Join(g.triggered, ", "))
	for _, note := range g.triggerNotes {
		fmt.Println(note)
	}
	return g
}

// computeBiasGates is the pure gate-mapping function: BiasAuditorResult
// slice → biasGates. Separated from runBiasGates so unit tests can cover
// the mapping without needing a real corpus. Any new phase gate should
// be added to the switch here (and a companion field on biasGates).
func computeBiasGates(auditors []BiasAuditorResult) biasGates {
	var g biasGates
	for _, a := range auditors {
		if !a.Triggered {
			continue
		}
		g.triggered = append(g.triggered, a.Bias)
		switch a.Bias {
		case "AvailabilityHeuristic":
			g.skipZim = true
			g.triggerNotes = append(g.triggerNotes,
				fmt.Sprintf("  availability_heuristic (HHI=%.2f > %.2f): ZIM backend skipped to diversify provenance",
					a.Value, a.Threshold))
		case "ConfirmationBias":
			// Corroboration rate is suspiciously high — the resolver or
			// query design is producing too much support and not enough
			// challenge. Skip the sense phase this cycle so we stop adding
			// more corroboration-biased signal. Mitigation, not cure: the
			// rate persists until other phases (trip, ingest) shift the
			// signal mix. The gate re-fires next cycle if the rate is
			// still above threshold; eventually corpus changes break out.
			g.skipSense = true
			g.triggerNotes = append(g.triggerNotes,
				fmt.Sprintf("  confirmation_bias (corroboration_rate=%.2f > %.2f): sense phase skipped to stop accumulating more corroborated signal",
					a.Value, a.Threshold))
		default:
			g.triggerNotes = append(g.triggerNotes,
				fmt.Sprintf("  %s (%s=%.2f vs threshold %.2f) — surfaced only, no phase gate yet",
					a.BiasName, a.Metric, a.Value, a.Threshold))
		}
	}
	return g
}
