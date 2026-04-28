package main

import "testing"

func TestComputeBiasGates(t *testing.T) {
	cases := []struct {
		name          string
		auditors      []BiasAuditorResult
		wantSkipZim   bool
		wantSkipSense bool
		wantCount     int
	}{
		{
			name:      "none triggered → no gates",
			auditors:  []BiasAuditorResult{{Bias: "AvailabilityHeuristic", Triggered: false}, {Bias: "SurvivorshipBias", Triggered: false}},
			wantCount: 0,
		},
		{
			name:        "availability triggered → skipZim",
			auditors:    []BiasAuditorResult{{Bias: "AvailabilityHeuristic", Triggered: true, Value: 0.6, Threshold: 0.25}},
			wantSkipZim: true,
			wantCount:   1,
		},
		{
			name: "availability + survivorship → skipZim, both in triggered list",
			auditors: []BiasAuditorResult{
				{Bias: "AvailabilityHeuristic", Triggered: true, Value: 0.6, Threshold: 0.25},
				{Bias: "SurvivorshipBias", Triggered: true, BiasName: "Survivorship bias", Metric: "ratio", Value: 197, Threshold: 5},
			},
			wantSkipZim: true,
			wantCount:   2,
		},
		{
			name:      "survivorship alone → no phase gate, still counted",
			auditors:  []BiasAuditorResult{{Bias: "SurvivorshipBias", Triggered: true, BiasName: "Survivorship bias"}},
			wantCount: 1,
		},
		{
			name:          "confirmation triggered → skipSense",
			auditors:      []BiasAuditorResult{{Bias: "ConfirmationBias", Triggered: true, BiasName: "Confirmation bias", Metric: "corroboration_rate", Value: 0.85, Threshold: 0.75}},
			wantSkipSense: true,
			wantCount:     1,
		},
		{
			name: "availability + confirmation → both gates fire independently",
			auditors: []BiasAuditorResult{
				{Bias: "AvailabilityHeuristic", Triggered: true, Value: 0.6, Threshold: 0.25},
				{Bias: "ConfirmationBias", Triggered: true, BiasName: "Confirmation bias", Metric: "corroboration_rate", Value: 0.85, Threshold: 0.75},
			},
			wantSkipZim:   true,
			wantSkipSense: true,
			wantCount:     2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := computeBiasGates(tc.auditors)
			if g.skipZim != tc.wantSkipZim {
				t.Errorf("skipZim = %v, want %v", g.skipZim, tc.wantSkipZim)
			}
			if g.skipSense != tc.wantSkipSense {
				t.Errorf("skipSense = %v, want %v", g.skipSense, tc.wantSkipSense)
			}
			if len(g.triggered) != tc.wantCount {
				t.Errorf("triggered count = %d, want %d", len(g.triggered), tc.wantCount)
			}
			if len(g.triggered) != len(g.triggerNotes) {
				t.Errorf("triggered/notes mismatch: %d vs %d", len(g.triggered), len(g.triggerNotes))
			}
		})
	}
}
