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

		{"old unset, new confirmed → treated as flip", "", "confirmed", "", "abc", driftFlippedConfirmed},
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
		// Stable, same commit — not counted as held-across-churn.
		{oldPredictionType: "trip_lint_durability", drift: driftStable, oldOracleCommit: "a", newOracleCommit: "a"},
		// Stable, commit changed — counted as held-across-churn.
		{oldPredictionType: "trip_lint_durability", drift: driftStable, oldOracleCommit: "a", newOracleCommit: "b"},
		// Stable but old commit empty (pre-versioning) — not counted.
		{oldPredictionType: "trip_promotion_attempt", drift: driftStable, oldOracleCommit: "", newOracleCommit: "b"},
		{oldPredictionType: "trip_lint_durability", drift: driftFlippedRefuted},
		{oldPredictionType: "trip_functional_durability", drift: driftResolverChanged},
		{oldPredictionType: "trip_functional_durability", drift: driftUnresolvable},
	}
	r := buildDurabilityReport(results)
	if r.Total != 6 {
		t.Errorf("Total = %d, want 6", r.Total)
	}
	if r.Stable != 3 {
		t.Errorf("Stable = %d, want 3", r.Stable)
	}
	if r.StableHeldAcrossCommit != 1 {
		t.Errorf("StableHeldAcrossCommit = %d, want 1", r.StableHeldAcrossCommit)
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
	if r.ByResolver["trip_lint_durability"] != 3 {
		t.Errorf("ByResolver[trip_lint_durability] = %d, want 3", r.ByResolver["trip_lint_durability"])
	}
}

func TestGoVersionDigest(t *testing.T) {
	d := goVersionDigest()
	if d == "" {
		t.Fatal("goVersionDigest returned empty — go binary missing?")
	}
	if len(d) != 12 {
		t.Errorf("digest length = %d, want 12", len(d))
	}
	// Two calls should be identical (Go version doesn't change mid-test)
	if d2 := goVersionDigest(); d2 != d {
		t.Errorf("goVersionDigest not deterministic: %q vs %q", d, d2)
	}
}
