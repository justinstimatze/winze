package main

import "testing"

func TestComputeBiasGates(t *testing.T) {
	cases := []struct {
		name      string
		auditors  []BiasAuditorResult
		wantSkip  bool
		wantCount int
	}{
		{
			name:      "none triggered → no gates",
			auditors:  []BiasAuditorResult{{Bias: "AvailabilityHeuristic", Triggered: false}, {Bias: "SurvivorshipBias", Triggered: false}},
			wantSkip:  false,
			wantCount: 0,
		},
		{
			name:      "availability triggered → skipZim",
			auditors:  []BiasAuditorResult{{Bias: "AvailabilityHeuristic", Triggered: true, Value: 0.6, Threshold: 0.25}},
			wantSkip:  true,
			wantCount: 1,
		},
		{
			name: "availability + survivorship → skipZim, both in triggered list",
			auditors: []BiasAuditorResult{
				{Bias: "AvailabilityHeuristic", Triggered: true, Value: 0.6, Threshold: 0.25},
				{Bias: "SurvivorshipBias", Triggered: true, BiasName: "Survivorship bias", Metric: "ratio", Value: 197, Threshold: 5},
			},
			wantSkip:  true,
			wantCount: 2,
		},
		{
			name:      "survivorship alone → no phase gate, still counted",
			auditors:  []BiasAuditorResult{{Bias: "SurvivorshipBias", Triggered: true, BiasName: "Survivorship bias"}},
			wantSkip:  false,
			wantCount: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := computeBiasGates(tc.auditors)
			if g.skipZim != tc.wantSkip {
				t.Errorf("skipZim = %v, want %v", g.skipZim, tc.wantSkip)
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
