package main

import (
	"strings"
	"testing"
)

func TestMapResolution(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"corroborated", "confirmed"},  // found evidence that strengthens
		{"challenged", "confirmed"},    // found evidence that challenges — prediction still confirmed
		{"irrelevant", "ambiguous"},    // papers found but not relevant — sensor miscalibration
		{"no_signal", "refuted"},       // no papers found — prediction was wrong
		{"", "ambiguous"},             // unknown resolution
		{"something_else", "ambiguous"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := mapResolution(tc.input)
			if got != tc.want {
				t.Errorf("mapResolution(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBetterResolution(t *testing.T) {
	// Priority: corroborated > challenged > irrelevant > no_signal > ""
	cases := []struct {
		a, b string
		want string
	}{
		{"", "no_signal", "no_signal"},
		{"no_signal", "irrelevant", "irrelevant"},
		{"irrelevant", "challenged", "challenged"},
		{"challenged", "corroborated", "corroborated"},
		// Higher stays
		{"corroborated", "no_signal", "corroborated"},
		{"corroborated", "challenged", "corroborated"},
		{"challenged", "irrelevant", "challenged"},
		// Equal stays
		{"irrelevant", "irrelevant", "irrelevant"},
		// Empty vs empty
		{"", "", ""},
	}
	for _, tc := range cases {
		name := tc.a + "_vs_" + tc.b
		if name == "_vs_" {
			name = "empty_vs_empty"
		}
		t.Run(name, func(t *testing.T) {
			got := betterResolution(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("betterResolution(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestBuildEvidenceString(t *testing.T) {
	cases := []struct {
		name    string
		record  *hypothesisRecord
		wantSub string // substring that must appear
	}{
		{
			name: "corroborated with papers",
			record: &hypothesisRecord{
				bestRes:    "corroborated",
				cycles:     10,
				withSignal: 5,
				papers:     []PaperSummary{{Title: "Paper A"}, {Title: "Paper B"}},
			},
			wantSub: "Corroborated: found 2 unique sources",
		},
		{
			name: "corroborated without papers",
			record: &hypothesisRecord{
				bestRes:    "corroborated",
				cycles:     3,
				withSignal: 1,
			},
			wantSub: "Resolution: corroborated",
		},
		{
			name: "challenged",
			record: &hypothesisRecord{
				bestRes:    "challenged",
				cycles:     5,
				withSignal: 2,
			},
			wantSub: "challenged",
		},
		{
			name: "irrelevant",
			record: &hypothesisRecord{
				bestRes:    "irrelevant",
				cycles:     8,
				withSignal: 3,
			},
			wantSub: "irrelevant",
		},
		{
			name: "no_signal",
			record: &hypothesisRecord{
				bestRes:    "no_signal",
				cycles:     4,
				withSignal: 0,
			},
			wantSub: "no signal",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildEvidenceString(tc.record)
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("buildEvidenceString() = %q, want to contain %q", got, tc.wantSub)
			}
		})
	}
}
