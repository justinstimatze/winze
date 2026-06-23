package main

import (
	"os"
	"path/filepath"
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

func TestCorpusDigest(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("alpha.go", "package corpus\n\nvar A = 1\n")
	writeFile("beta.go", "package corpus\n\nvar B = 2\n")
	writeFile("go.mod", "module corpus\n\ngo 1.22\n")

	d1 := corpusDigest(dir)
	if len(d1) != 12 {
		t.Fatalf("digest length = %d, want 12", len(d1))
	}
	// Deterministic across calls.
	if d2 := corpusDigest(dir); d2 != d1 {
		t.Errorf("corpusDigest not deterministic: %q vs %q", d1, d2)
	}

	// A content change must move the digest.
	writeFile("beta.go", "package corpus\n\nvar B = 3\n")
	if d3 := corpusDigest(dir); d3 == d1 {
		t.Error("corpusDigest unchanged after editing beta.go — gate would skip unsoundly")
	}

	// _test.go files must NOT affect the digest (go build doesn't compile
	// them; including them would cause spurious re-runs).
	dBefore := corpusDigest(dir)
	writeFile("alpha_test.go", "package corpus\n\n// noise\n")
	if dAfter := corpusDigest(dir); dAfter != dBefore {
		t.Error("corpusDigest changed when only a _test.go file was added")
	}
}

func TestCorpusUnchanged(t *testing.T) {
	const pt = "trip_lint_durability"
	latest := map[verdictKey]verdictRecord{
		{hypothesis: "X", predictionType: pt}: {corpusDigest: "cccccccccccc", oracleDigest: "oooooooooooo"},
		{hypothesis: "Y", predictionType: pt}: {corpusDigest: "cccccccccccc", oracleDigest: "oooooooooooo"},
	}
	vars := []string{"X", "Y"}

	if !corpusUnchanged(vars, latest, pt, "cccccccccccc", "oooooooooooo") {
		t.Error("want skip when both digests match for all vars")
	}
	if corpusUnchanged(vars, latest, pt, "dddddddddddd", "oooooooooooo") {
		t.Error("must NOT skip when corpus digest moved")
	}
	if corpusUnchanged(vars, latest, pt, "cccccccccccc", "pppppppppppp") {
		t.Error("must NOT skip when oracle digest moved (lint code edited)")
	}
	if corpusUnchanged(vars, latest, pt, "", "oooooooooooo") {
		t.Error("must NOT skip when current corpus digest is empty")
	}

	// A legacy entry with no stored CorpusDigest forces a full run.
	latest[verdictKey{hypothesis: "Z", predictionType: pt}] = verdictRecord{corpusDigest: "", oracleDigest: "oooooooooooo"}
	if corpusUnchanged([]string{"X", "Y", "Z"}, latest, pt, "cccccccccccc", "oooooooooooo") {
		t.Error("must NOT skip when any var lacks a stored corpus digest (legacy entry)")
	}
}

func TestStableCarryForward(t *testing.T) {
	const pt = "trip_lint_durability"
	// latest = original promotion verdicts (drift baseline).
	latest := map[verdictKey]verdictRecord{
		{hypothesis: "Good", predictionType: pt}: {resolution: "confirmed", oracleDigest: "oooooooooooo"},
		{hypothesis: "Flip", predictionType: pt}: {resolution: "confirmed", oracleDigest: "oooooooooooo"},
		{hypothesis: "Gone", predictionType: pt}: {resolution: "unresolvable", oracleDigest: "oooooooooooo"},
	}
	// digests = newest run's state. "Flip" already flipped to refuted in a
	// prior recheck; carrying the ORIGINAL "confirmed" would mask that.
	digests := map[verdictKey]verdictRecord{
		{hypothesis: "Good", predictionType: pt}: {resolution: "confirmed", oracleDigest: "oooooooooooo", corpusDigest: "cccccccccccc"},
		{hypothesis: "Flip", predictionType: pt}: {resolution: "refuted", oracleDigest: "oooooooooooo", corpusDigest: "cccccccccccc"},
		{hypothesis: "Gone", predictionType: pt}: {resolution: "unresolvable", oracleDigest: "oooooooooooo", corpusDigest: "cccccccccccc"},
	}
	out := stableCarryForward([]string{"Good", "Flip", "Gone"}, latest, digests, pt, "abc123", "cccccccccccc")

	byHyp := map[string]durabilityResult{}
	for _, r := range out {
		byHyp[r.hypothesis] = r
	}
	if got := byHyp["Good"]; got.drift != driftStable || got.newVerdict != "confirmed" {
		t.Errorf("Good: drift=%q verdict=%q, want stable/confirmed", got.drift, got.newVerdict)
	}
	// The persisted flip must survive the skip, not be masked as stable.
	if got := byHyp["Flip"]; got.newVerdict != "refuted" || got.drift != driftFlippedRefuted {
		t.Errorf("Flip: drift=%q verdict=%q, want flipped_to_refuted/refuted (skip must not mask a prior flip)", got.drift, got.newVerdict)
	}
	if got := byHyp["Gone"]; got.drift != driftUnresolvable {
		t.Errorf("Gone: drift=%q, want unresolvable (must not write a bogus stable entry)", got.drift)
	}
	if byHyp["Good"].newCorpusDigest != "cccccccccccc" {
		t.Errorf("newCorpusDigest=%q, want propagated", byHyp["Good"].newCorpusDigest)
	}
}

// TestLatestRecheckState_ReadsBackDigest is the regression guard for the
// gate-never-fires bug: CorpusDigest is written only on _recheck entries,
// and latestVerdicts filters those out, so the gate must read its digest
// via latestRecheckState (which maps _recheck back to its base type).
func TestLatestRecheckState_ReadsBackDigest(t *testing.T) {
	const base = "trip_lint_durability"
	cycles := []Cycle{
		// Original promotion entry: no corpus digest.
		{Hypothesis: "H", PredictionType: base, Resolution: "confirmed", Timestamp: time.Unix(100, 0)},
		// A later _recheck wrote the corpus digest.
		{Hypothesis: "H", PredictionType: base + "_recheck", Resolution: "confirmed", CorpusDigest: "deadbeef0000", OracleDigest: "oo", Timestamp: time.Unix(200, 0)},
	}

	// latestVerdicts (originals only) must NOT see the digest...
	if got := latestVerdicts(cycles)[verdictKey{"H", base}].corpusDigest; got != "" {
		t.Errorf("latestVerdicts corpusDigest = %q, want empty (it ignores _recheck)", got)
	}
	// ...but latestRecheckState must surface it under the BASE key.
	st := latestRecheckState(cycles)[verdictKey{"H", base}]
	if st.corpusDigest != "deadbeef0000" {
		t.Errorf("latestRecheckState corpusDigest = %q, want deadbeef0000 — gate would be dead", st.corpusDigest)
	}
}
