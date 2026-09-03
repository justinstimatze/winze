package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	f()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestDatedMeasurementRuleFlagsUndatedMeasurement is FEEDBACK-2026-09-02.md#4:
// a measurement- or status-shaped Brief with no date is exactly the shape
// internal/defndb's stale "Dolt-backed" claim had.
func TestDatedMeasurementRuleFlagsUndatedMeasurement(t *testing.T) {
	src := `package winze

var staleClaim = Entity{Brief: "The store links in 0.14ms and is backed by SQLite."}
`
	dir := writeLintFixture(t, src)
	var rc int
	out := captureStdout(t, func() { rc = datedMeasurementRule(dir) })
	if rc != 0 {
		t.Errorf("dated-measurement rule must stay advisory (rc=0), got rc=%d", rc)
	}
	if !strings.Contains(out, "staleClaim") {
		t.Errorf("undated measurement not flagged:\n%s", out)
	}
}

// TestDatedMeasurementRuleIgnoresOtherFields pins the field scope to
// Brief/Rationale/Quote — the fields FEEDBACK-2026-09-02.md#4 named, not
// every string field a corpus type happens to have.
func TestDatedMeasurementRuleIgnoresOtherFields(t *testing.T) {
	src := `package winze

var notChecked = Entity{Kind: "concept links in 12ms"}
`
	dir := writeLintFixture(t, src)
	out := captureStdout(t, func() { datedMeasurementRule(dir) })
	if strings.Contains(out, "notChecked") {
		t.Errorf("a field other than Brief/Rationale/Quote was checked:\n%s", out)
	}
}

func TestDatedMeasurementRulePassesWhenDated(t *testing.T) {
	src := `package winze

var datedClaim = Entity{Brief: "The store linked in 0.14ms as of 2026-08-15."}
`
	dir := writeLintFixture(t, src)
	var rc int
	out := captureStdout(t, func() { rc = datedMeasurementRule(dir) })
	if rc != 0 {
		t.Errorf("dated-measurement rule must stay advisory (rc=0), got rc=%d", rc)
	}
	if strings.Contains(out, "datedClaim") {
		t.Errorf("dated measurement wrongly flagged despite carrying a date:\n%s", out)
	}
}

// TestDatedMeasurementRuleOnRealCorpus runs against the actual corpus/ (not
// repoRoot(t) the way TestBriefCheckRule/TestProvenanceSplitRule do above,
// which scans a directory with no top-level .go files of its own and so
// never really exercises the corpus) and pins the current measured shape: a
// handful of hits, still advisory. A run against the real thing found 10 of
// 496 Brief/Rationale/Quote strings measurement-shaped with no date — mostly
// Quote fields quoting historical decades ("in the 1970s") or a verbatim
// user remark containing the word "now", neither of which this rule can fix
// without either inventing a date or rewriting a citation's exact source
// text, which is exactly why this stays advisory rather than gating.
func TestDatedMeasurementRuleOnRealCorpus(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "corpus")
	if rc := datedMeasurementRule(dir); rc != 0 {
		t.Errorf("dated-measurement rule must stay advisory (rc=0), got rc=%d", rc)
	}
}
