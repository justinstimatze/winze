package main

import (
	"testing"
	"time"
)

func TestClassifyDrift(t *testing.T) {
	cases := []struct {
		name       string
		oldVerdict string
		newVerdict string
		oldDigest  string
		newDigest  string
		want       driftCategory
	}{
		{"stable confirmed, digest match", "confirmed", "confirmed", "abc", "abc", driftStable},
		{"stable refuted, digest match", "refuted", "refuted", "abc", "abc", driftStable},
		{"stable confirmed, empty digests", "confirmed", "confirmed", "", "", driftStable},
		{"stable confirmed, one empty digest (pre-versioning)", "confirmed", "confirmed", "", "abc", driftStable},

		{"flipped to refuted", "confirmed", "refuted", "abc", "abc", driftFlippedRefuted},
		{"flipped to confirmed", "refuted", "confirmed", "abc", "abc", driftFlippedConfirmed},

		{"resolver changed, verdict stable", "confirmed", "confirmed", "abc", "def", driftResolverChanged},
		{"resolver changed, refuted stable", "refuted", "refuted", "abc", "def", driftResolverChanged},

		{"unresolvable takes precedence over digest change", "confirmed", "unresolvable", "abc", "def", driftUnresolvable},
		{"unresolvable takes precedence over flip", "refuted", "unresolvable", "abc", "abc", driftUnresolvable},

		{"empty new verdict treated as ambiguous", "confirmed", "", "abc", "abc", driftNowAmbiguous},

		{"old unset, new confirmed (stable)", "", "confirmed", "", "abc", driftFlippedConfirmed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyDrift(tc.oldVerdict, tc.newVerdict, tc.oldDigest, tc.newDigest)
			if got != tc.want {
				t.Errorf("classifyDrift(%q, %q, %q, %q) = %q, want %q",
					tc.oldVerdict, tc.newVerdict, tc.oldDigest, tc.newDigest, got, tc.want)
			}
		})
	}
}

func TestLatestVerdicts_PicksMostRecent(t *testing.T) {
	early := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)

	cycles := []Cycle{
		{Timestamp: early, Hypothesis: "Foo", PredictionType: "trip_lint_durability", Resolution: "confirmed", Evidence: "old"},
		{Timestamp: late, Hypothesis: "Foo", PredictionType: "trip_lint_durability", Resolution: "refuted", Evidence: "new"},
		{Timestamp: late, Hypothesis: "Bar", PredictionType: "trip_functional_durability", Resolution: "confirmed"},
		// Sensor entries should be ignored
		{Timestamp: late, Hypothesis: "Baz", PredictionType: "", Resolution: "corroborated"},
		{Timestamp: late, Hypothesis: "Baz", PredictionType: "structural_fragility", Resolution: "corroborated"},
		// Recheck entries should be ignored (we only re-run originals)
		{Timestamp: late, Hypothesis: "Foo", PredictionType: "trip_lint_durability_recheck", Resolution: "confirmed"},
	}

	latest := latestVerdicts(cycles)

	fooKey := verdictKey{hypothesis: "Foo", predictionType: "trip_lint_durability"}
	if v, ok := latest[fooKey]; !ok {
		t.Fatalf("expected Foo/lint entry")
	} else if v.resolution != "refuted" || v.evidence != "new" {
		t.Errorf("Foo/lint got %+v, want most-recent refuted/new", v)
	}

	barKey := verdictKey{hypothesis: "Bar", predictionType: "trip_functional_durability"}
	if _, ok := latest[barKey]; !ok {
		t.Errorf("expected Bar/functional entry")
	}

	// Nothing for Baz (sensor type) or Foo/_recheck
	for k := range latest {
		if k.hypothesis == "Baz" {
			t.Errorf("sensor entry should be ignored: %+v", k)
		}
		if k.predictionType == "trip_lint_durability_recheck" {
			t.Errorf("recheck entry should be ignored: %+v", k)
		}
	}
}

func TestIsRecheckable(t *testing.T) {
	want := map[string]bool{
		"trip_lint_durability":         true,
		"trip_functional_durability":   true,
		"trip_promotion_attempt":       true,
		"trip_llm_durability":          false, // LLM rechecks deferred
		"structural_fragility":         false, // sensor type
		"":                             false, // legacy sensor
		"trip_lint_durability_recheck": false, // don't recurse on rechecks
	}
	for pt, expect := range want {
		if got := isRecheckable(pt); got != expect {
			t.Errorf("isRecheckable(%q) = %v, want %v", pt, got, expect)
		}
	}
}

func TestBuildDurabilityReport(t *testing.T) {
	results := []durabilityResult{
		{oldPredictionType: "trip_lint_durability", drift: driftStable},
		{oldPredictionType: "trip_lint_durability", drift: driftFlippedRefuted},
		{oldPredictionType: "trip_functional_durability", drift: driftResolverChanged},
		{oldPredictionType: "trip_functional_durability", drift: driftUnresolvable},
		{oldPredictionType: "trip_promotion_attempt", drift: driftStable},
	}
	r := buildDurabilityReport(results)
	if r.Total != 5 {
		t.Errorf("Total = %d, want 5", r.Total)
	}
	if r.Stable != 2 {
		t.Errorf("Stable = %d, want 2", r.Stable)
	}
	if r.FlippedToRefuted != 1 {
		t.Errorf("FlippedToRefuted = %d, want 1", r.FlippedToRefuted)
	}
	if r.ResolverChanged != 1 {
		t.Errorf("ResolverChanged = %d, want 1", r.ResolverChanged)
	}
	if r.Unresolvable != 1 {
		t.Errorf("Unresolvable = %d, want 1", r.Unresolvable)
	}
	if r.ByResolver["trip_lint_durability"] != 2 {
		t.Errorf("ByResolver[trip_lint_durability] = %d, want 2", r.ByResolver["trip_lint_durability"])
	}
}
