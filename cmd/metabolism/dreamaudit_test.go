package main

import (
	"math"
	"strings"
	"testing"
)

func TestSpearmanRho(t *testing.T) {
	cases := []struct {
		name string
		x, y []float64
		want float64
		tol  float64 // tolerance for floating point
	}{
		{
			name: "perfect positive correlation",
			x:    []float64{1, 2, 3, 4, 5},
			y:    []float64{2, 4, 6, 8, 10},
			want: 1.0,
			tol:  1e-10,
		},
		{
			name: "perfect negative correlation",
			x:    []float64{1, 2, 3, 4, 5},
			y:    []float64{10, 8, 6, 4, 2},
			want: -1.0,
			tol:  1e-10,
		},
		{
			name: "partial correlation",
			// x ranks: [1,2,3,4,5], y ranks: [3,1,2,5,4]
			// d = [-2,1,1,-1,1], d^2 = [4,1,1,1,1] = 8
			// rho = 1 - 6*8/(5*24) = 1 - 48/120 = 0.6
			x:    []float64{10, 20, 30, 40, 50},
			y:    []float64{30, 10, 20, 50, 40},
			want: 0.6,
			tol:  1e-10,
		},
		{
			name: "too few elements returns 0",
			x:    []float64{1, 2},
			y:    []float64{3, 4},
			want: 0,
			tol:  0,
		},
		{
			name: "mismatched lengths returns 0",
			x:    []float64{1, 2, 3},
			y:    []float64{4, 5},
			want: 0,
			tol:  0,
		},
		{
			name: "with ties",
			x:    []float64{1, 2, 2, 3},
			y:    []float64{1, 2, 3, 4},
			// x ranks: [1, 2.5, 2.5, 4], y ranks: [1, 2, 3, 4]
			// d = [0, 0.5, -0.5, 0], d^2 = [0, 0.25, 0.25, 0] = 0.5
			// rho = 1 - 6*0.5/(4*15) = 1 - 3/60 = 1 - 0.05 = 0.95
			want: 0.95,
			tol:  1e-10,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := spearmanRho(tc.x, tc.y)
			if math.Abs(got-tc.want) > tc.tol {
				t.Errorf("spearmanRho(%v, %v) = %f, want %f (±%f)", tc.x, tc.y, got, tc.want, tc.tol)
			}
		})
	}
}

func TestAssignRanks(t *testing.T) {
	cases := []struct {
		name string
		vals []float64
		want []float64
	}{
		{
			name: "no ties",
			vals: []float64{10, 30, 20},
			want: []float64{1, 3, 2},
		},
		{
			name: "with ties",
			vals: []float64{10, 20, 20, 30},
			want: []float64{1, 2.5, 2.5, 4},
		},
		{
			name: "all same",
			vals: []float64{5, 5, 5},
			want: []float64{2, 2, 2}, // average of ranks 1,2,3
		},
		{
			name: "already sorted",
			vals: []float64{1, 2, 3, 4},
			want: []float64{1, 2, 3, 4},
		},
		{
			name: "reverse sorted",
			vals: []float64{4, 3, 2, 1},
			want: []float64{4, 3, 2, 1},
		},
		{
			name: "single element",
			vals: []float64{42},
			want: []float64{1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := assignRanks(tc.vals)
			if len(got) != len(tc.want) {
				t.Fatalf("assignRanks(%v) returned %d ranks, want %d", tc.vals, len(got), len(tc.want))
			}
			for i, g := range got {
				if math.Abs(g-tc.want[i]) > 1e-10 {
					t.Errorf("assignRanks(%v)[%d] = %f, want %f", tc.vals, i, g, tc.want[i])
				}
			}
		})
	}
}

func TestClassifyOrigin(t *testing.T) {
	cases := []struct {
		origin string
		want   string
	}{
		{"Wikipedia (zim 2025-12) / Hard_problem", "wikipedia"},
		{"arXiv:2301.12345", "arxiv"},
		{"doi:10.1234/example", "doi"},
		{"https://doi.org/10.1234/example", "doi"},
		{"ISBN 978-0-14-028329-7 (Chalmers, 1996)", "book"},
		{"Direct observation", "manual"},
		{"Some other source", "other"},
		{"", "other"},
	}
	for _, tc := range cases {
		t.Run(tc.origin, func(t *testing.T) {
			got := classifyOrigin(tc.origin)
			if got != tc.want {
				t.Errorf("classifyOrigin(%q) = %q, want %q", tc.origin, got, tc.want)
			}
		})
	}
}

func TestContainsWord(t *testing.T) {
	excl := map[string][]string{
		"controversial": {"controversial claim that"},
		"simplistic":    {"simplistic model"},
	}
	cases := []struct {
		text string
		term string
		want bool
	}{
		{"a groundbreaking discovery", "groundbreaking", true},
		{"no match here", "groundbreaking", false},
		{"the controversial claim that X", "controversial", false}, // excluded
		{"a controversial hypothesis", "controversial", true},     // not excluded
		{"a simplistic model of mind", "simplistic", false},       // excluded
		{"a simplistic view of reality", "simplistic", true},      // not excluded
		{"", "groundbreaking", false},
	}
	for _, tc := range cases {
		got := containsWord(tc.text, tc.term, excl)
		if got != tc.want {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tc.text, tc.term, got, tc.want)
		}
	}
}

// TestAuditorsAgainstRealCorpus runs all 9 bias auditors against the real
// winze corpus and verifies structural invariants. This is an integration
// test: it doesn't check specific values (which change as the KB evolves)
// but verifies each auditor returns coherent results.
func TestAuditorsAgainstRealCorpus(t *testing.T) {
	root := repoRoot(t)

	// Run all auditors (except DunningKruger which needs topology subprocess)
	auditors := []struct {
		name string
		fn   func(string) BiasAuditorResult
	}{
		{"ConfirmationBias", auditConfirmationBias},
		{"AnchoringBias", auditAnchoringBias},
		{"ClusteringIllusion", auditClusteringIllusion},
		{"AvailabilityHeuristic", auditAvailabilityHeuristic},
		{"SurvivorshipBias", auditSurvivorshipBias},
		{"FramingEffect", auditFramingEffect},
		{"BaseRateNeglect", auditBaseRateNeglect},
		{"PrematureClosure", auditPrematureClosure},
	}

	for _, a := range auditors {
		t.Run(a.name, func(t *testing.T) {
			result := a.fn(root)

			// Structural invariants every auditor must satisfy
			if result.Bias == "" {
				t.Error("Bias field is empty")
			}
			if result.BiasName == "" {
				t.Error("BiasName field is empty")
			}
			if result.Metric == "" {
				t.Error("Metric field is empty")
			}
			if result.Threshold <= 0 {
				t.Errorf("Threshold = %f, want > 0", result.Threshold)
			}
			if result.Detail == "" {
				t.Error("Detail field is empty — auditor returned no explanation")
			}
			if result.Triggered && result.Severity == "" {
				t.Error("Triggered but Severity is empty")
			}
			if result.Triggered && result.Conclusion == "" {
				t.Error("Triggered but Conclusion is empty")
			}

			// Value should be finite
			if math.IsNaN(result.Value) || math.IsInf(result.Value, 0) {
				t.Errorf("Value = %f, want finite", result.Value)
			}

			t.Logf("%s: value=%.3f threshold=%.3f triggered=%v detail=%s",
				result.BiasName, result.Value, result.Threshold, result.Triggered,
				truncate(result.Detail, 120))
		})
	}
}

func TestAuditDunningKruger_WithNilReport(t *testing.T) {
	// DunningKruger with nil topoReport triggers a topology subprocess.
	// Skip in CI — this test verifies the nil-report path doesn't panic.
	if testing.Short() {
		t.Skip("skipping topology subprocess test in short mode")
	}
	root := repoRoot(t)
	result := auditDunningKruger(root, nil)
	if result.Bias != "DunningKrugerEffect" {
		t.Errorf("Bias = %q, want %q", result.Bias, "DunningKrugerEffect")
	}
	if result.Detail == "" {
		t.Error("Detail is empty — DunningKruger returned no analysis")
	}
	t.Logf("DunningKruger: value=%.3f triggered=%v", result.Value, result.Triggered)
}

func TestCollectBiasResults_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full bias suite in short mode")
	}
	root := repoRoot(t)
	report := collectBiasResults(root, nil, false, false)
	if len(report.Auditors) != 9 {
		t.Errorf("expected 9 auditors, got %d", len(report.Auditors))
	}
	if report.Summary == "" {
		t.Error("Summary is empty")
	}
	if !strings.Contains(report.Summary, "of 9") {
		t.Errorf("Summary = %q, want to contain 'of 9'", report.Summary)
	}
	t.Logf("bias report: %s", report.Summary)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
