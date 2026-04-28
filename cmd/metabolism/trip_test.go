package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPickCrossClusterPairs(t *testing.T) {
	t.Run("two clusters", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "entity a1"},
			{name: "A2", cluster: 0, brief: "entity a2"},
			{name: "B1", cluster: 1, brief: "entity b1"},
			{name: "B2", cluster: 1, brief: "entity b2"},
		}
		pairs := pickCrossClusterPairs(entities, 2)
		if len(pairs) == 0 {
			t.Fatal("expected at least 1 cross-cluster pair")
		}
		if len(pairs) > 2 {
			t.Errorf("requested 2 pairs, got %d", len(pairs))
		}
		for _, p := range pairs {
			if p.A.cluster == p.B.cluster {
				t.Errorf("pair %s-%s are in same cluster %d", p.A.name, p.B.name, p.A.cluster)
			}
		}
	})

	t.Run("single cluster returns nil", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "a1"},
			{name: "A2", cluster: 0, brief: "a2"},
		}
		pairs := pickCrossClusterPairs(entities, 5)
		if pairs != nil {
			t.Errorf("expected nil for single cluster, got %d pairs", len(pairs))
		}
	})

	t.Run("prefer entities with briefs", func(t *testing.T) {
		entities := []tripEntity{
			{name: "Blank1", cluster: 0},
			{name: "Rich1", cluster: 0, brief: "has a brief"},
			{name: "Blank2", cluster: 1},
			{name: "Rich2", cluster: 1, brief: "also has a brief"},
		}
		pairs := pickCrossClusterPairs(entities, 1)
		if len(pairs) == 0 {
			t.Fatal("expected at least 1 pair")
		}
		// First pair should use the entities with briefs (score 2 > score 0)
		p := pairs[0]
		if p.A.brief == "" || p.B.brief == "" {
			t.Errorf("expected pair with briefs, got %q + %q", p.A.brief, p.B.brief)
		}
	})

	t.Run("negative cluster excluded", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "a1"},
			{name: "Orphan", cluster: -1, brief: "no cluster"},
			{name: "B1", cluster: 1, brief: "b1"},
		}
		pairs := pickCrossClusterPairs(entities, 5)
		for _, p := range pairs {
			if p.A.name == "Orphan" || p.B.name == "Orphan" {
				t.Error("orphan entity (cluster -1) should not appear in pairs")
			}
		}
	})
}

// TestPairCandidateScore pins the bridge-bias scoring: bridge endpoints
// add 2 points each, brief presence adds 1. Weights chosen so any
// bridge-anchored pair outranks any non-bridge pair, even one with both
// briefs filled.
func TestPairCandidateScore(t *testing.T) {
	cases := []struct {
		name string
		a, b tripEntity
		want int
	}{
		{"both bridges, both briefs", tripEntity{bridge: true, brief: "x"}, tripEntity{bridge: true, brief: "y"}, 8},
		{"both bridges, no briefs", tripEntity{bridge: true}, tripEntity{bridge: true}, 6},
		{"one bridge, both briefs", tripEntity{bridge: true, brief: "x"}, tripEntity{brief: "y"}, 5},
		{"one bridge alone", tripEntity{bridge: true}, tripEntity{}, 3},
		{"both briefs, no bridges", tripEntity{brief: "x"}, tripEntity{brief: "y"}, 2},
		{"one brief", tripEntity{brief: "x"}, tripEntity{}, 1},
		{"empty", tripEntity{}, tripEntity{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pairCandidateScore(tc.a, tc.b); got != tc.want {
				t.Errorf("pairCandidateScore = %d, want %d", got, tc.want)
			}
		})
	}

	// Invariant: any pair with at least one bridge endpoint outscores any
	// pair with no bridge endpoint, regardless of brief completeness.
	worstBridge := pairCandidateScore(tripEntity{bridge: true}, tripEntity{})
	bestNonBridge := pairCandidateScore(tripEntity{brief: "x"}, tripEntity{brief: "y"})
	if worstBridge <= bestNonBridge {
		t.Errorf("invariant broken: worst-bridge (%d) must outrank best-non-bridge (%d)", worstBridge, bestNonBridge)
	}
}

// TestPickCrossClusterPairs_BridgeBias verifies the sampler surfaces
// bridge-anchored pairs first when bridges exist. Without this bias the
// sampler picks uniformly across cross-cluster pairs and most candidates
// are weak-analogy random concept pairs (the 2026-04-27 demo failure
// mode).
func TestPickCrossClusterPairs_BridgeBias(t *testing.T) {
	entities := []tripEntity{
		{name: "BridgeA", cluster: 0, brief: "anchor", bridge: true},
		{name: "PlainA", cluster: 0, brief: "plain"},
		{name: "BridgeB", cluster: 1, brief: "anchor", bridge: true},
		{name: "PlainB", cluster: 1, brief: "plain"},
	}
	pairs := pickCrossClusterPairs(entities, 1)
	if len(pairs) == 0 {
		t.Fatal("expected at least 1 pair")
	}
	p := pairs[0]
	if !p.A.bridge || !p.B.bridge {
		t.Errorf("expected bridge×bridge to surface first, got %s(bridge=%v) ↔ %s(bridge=%v)",
			p.A.name, p.A.bridge, p.B.name, p.B.bridge)
	}
}

// TestFindBridgesFromAdj covers the articulation-point detector inlined
// from cmd/topology. A path graph A-B-C-D has B and C as bridges;
// removing either splits the graph. Endpoints (A, D) are not bridges
// (they have <2 neighbors).
func TestFindBridgesFromAdj(t *testing.T) {
	t.Run("path graph", func(t *testing.T) {
		adj := map[string]map[string]bool{
			"A": {"B": true},
			"B": {"A": true, "C": true},
			"C": {"B": true, "D": true},
			"D": {"C": true},
		}
		got := findBridgesFromAdj(adj)
		if !got["B"] || !got["C"] {
			t.Errorf("expected B and C to be bridges, got %v", got)
		}
		if got["A"] || got["D"] {
			t.Errorf("expected endpoints A, D to NOT be bridges, got %v", got)
		}
	})

	t.Run("triangle has no bridges", func(t *testing.T) {
		// In a 3-cycle, no node's removal disconnects the rest.
		// Algorithm requires len(adj) >= 4, so add a pendant.
		adj := map[string]map[string]bool{
			"A": {"B": true, "C": true},
			"B": {"A": true, "C": true},
			"C": {"A": true, "B": true, "D": true},
			"D": {"C": true},
		}
		got := findBridgesFromAdj(adj)
		// C is a bridge (removing it isolates D); A, B, D are not.
		if !got["C"] {
			t.Error("expected C to be bridge (cuts pendant D)")
		}
		if got["A"] || got["B"] {
			t.Errorf("expected triangle nodes A, B to NOT be bridges, got %v", got)
		}
	})

	t.Run("tiny graph returns nil", func(t *testing.T) {
		// Algorithm short-circuits below threshold.
		adj := map[string]map[string]bool{
			"A": {"B": true},
			"B": {"A": true},
		}
		if got := findBridgesFromAdj(adj); got != nil {
			t.Errorf("expected nil for tiny graph, got %v", got)
		}
	})
}

// TestPairStructuralAffinity pins each of the three signals
// (2-hop overlap, predicate complementarity, brief-vocab overlap)
// independently, so a regression in one component is localized.
func TestPairStructuralAffinity(t *testing.T) {
	t.Run("zero across the board", func(t *testing.T) {
		a := tripEntity{name: "A"}
		b := tripEntity{name: "B"}
		if got := pairStructuralAffinity(a, b); got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("2-hop overlap only", func(t *testing.T) {
		// Both reach the same intermediate node but share no predicates
		// and no brief vocabulary.
		a := tripEntity{name: "A", twoHop: map[string]bool{"X": true, "Y": true}}
		b := tripEntity{name: "B", twoHop: map[string]bool{"X": true}}
		if got := pairStructuralAffinity(a, b); got != 1 {
			t.Errorf("expected 1 (single 2-hop overlap), got %d", got)
		}
	})

	t.Run("2-hop overlap caps at 3", func(t *testing.T) {
		shared := map[string]bool{"X1": true, "X2": true, "X3": true, "X4": true, "X5": true}
		a := tripEntity{name: "A", twoHop: shared}
		b := tripEntity{name: "B", twoHop: shared}
		if got := pairStructuralAffinity(a, b); got != 3 {
			t.Errorf("expected cap at 3, got %d", got)
		}
	})

	t.Run("predicate complementarity (full overlap)", func(t *testing.T) {
		// Identical predicate sets → Jaccard 1.0 → score 4.
		preds := map[string]bool{"TheoryOf": true, "CommentaryOn": true}
		a := tripEntity{name: "A", predicates: preds}
		b := tripEntity{name: "B", predicates: preds}
		if got := pairStructuralAffinity(a, b); got != 4 {
			t.Errorf("expected 4 (full predicate overlap), got %d", got)
		}
	})

	t.Run("predicate complementarity (no overlap)", func(t *testing.T) {
		// Disjoint sets → Jaccard 0 → score 0.
		a := tripEntity{name: "A", predicates: map[string]bool{"TheoryOf": true}}
		b := tripEntity{name: "B", predicates: map[string]bool{"LocatedIn": true}}
		if got := pairStructuralAffinity(a, b); got != 0 {
			t.Errorf("expected 0 (disjoint predicates), got %d", got)
		}
	})

	t.Run("brief vocab overlap only", func(t *testing.T) {
		a := tripEntity{name: "A", briefTokens: map[string]bool{"consciousness": true, "perception": true}}
		b := tripEntity{name: "B", briefTokens: map[string]bool{"consciousness": true}}
		if got := pairStructuralAffinity(a, b); got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})

	t.Run("brief vocab caps at 3", func(t *testing.T) {
		shared := map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true}
		a := tripEntity{name: "A", briefTokens: shared}
		b := tripEntity{name: "B", briefTokens: shared}
		// 5 tokens overlap, but the brief-vocab signal caps at 3.
		// (Other signals are zero because predicates/twoHop are nil.)
		if got := pairStructuralAffinity(a, b); got != 3 {
			t.Errorf("expected cap at 3, got %d", got)
		}
	})

	t.Run("composite (all three signals)", func(t *testing.T) {
		// 2-hop: 1 overlap (+1), predicates: identical (+4), tokens: 2 overlap (+2). Total: 7.
		a := tripEntity{
			name:        "A",
			twoHop:      map[string]bool{"X": true},
			predicates:  map[string]bool{"TheoryOf": true},
			briefTokens: map[string]bool{"alpha": true, "beta": true},
		}
		b := tripEntity{
			name:        "B",
			twoHop:      map[string]bool{"X": true},
			predicates:  map[string]bool{"TheoryOf": true},
			briefTokens: map[string]bool{"alpha": true, "beta": true},
		}
		if got := pairStructuralAffinity(a, b); got != 7 {
			t.Errorf("expected 7 (1+4+2), got %d", got)
		}
	})
}

// TestPickCrossClusterPairs_AffinityTieBreaker verifies that among
// candidates with identical primary score (bridge/brief), the candidate
// with higher structural affinity surfaces first. Without the tie-breaker
// the order within a score class is purely random shuffle, so the same
// fixture would sometimes pick the lower-affinity pair.
func TestPickCrossClusterPairs_AffinityTieBreaker(t *testing.T) {
	// Two cross-cluster candidate pairs, both bridge×bridge with both
	// briefs (primary score = 8 each). The "Aligned" pair shares a 2-hop
	// neighbor and a predicate; the "Mismatched" pair shares neither.
	// The aligned pair must sort first deterministically across many runs.
	entities := []tripEntity{
		{
			name: "AlignedA", cluster: 0, brief: "x", bridge: true,
			twoHop:     map[string]bool{"Shared": true},
			predicates: map[string]bool{"TheoryOf": true},
		},
		{
			name: "MismatchedA", cluster: 0, brief: "x", bridge: true,
		},
		{
			name: "AlignedB", cluster: 1, brief: "y", bridge: true,
			twoHop:     map[string]bool{"Shared": true},
			predicates: map[string]bool{"TheoryOf": true},
		},
		{
			name: "MismatchedB", cluster: 1, brief: "y", bridge: true,
		},
	}

	// Run multiple times — random shuffle within score class would
	// occasionally pick MismatchedA-MismatchedB first if there were no
	// affinity tie-breaker. With the tie-breaker the aligned pair wins
	// every time.
	for i := 0; i < 20; i++ {
		pairs := pickCrossClusterPairs(entities, 1)
		if len(pairs) == 0 {
			t.Fatal("expected at least 1 pair")
		}
		p := pairs[0]
		alignedFirst := (p.A.name == "AlignedA" && p.B.name == "AlignedB") ||
			(p.A.name == "AlignedB" && p.B.name == "AlignedA")
		if !alignedFirst {
			t.Errorf("iter %d: expected AlignedA↔AlignedB to surface first via affinity tie-break, got %s↔%s",
				i, p.A.name, p.B.name)
			return
		}
	}
}

// TestSampleAntiExemplars verifies the corpus-mining helper that
// reconstructs deleted-claim shapes from `// YYYY-MM-DD audit:` comment
// blocks. The sampler is the data source for idea #3 (negative-shape
// guidance in the trip prompt), so a regression here silently empties
// the prompt's <anti_exemplars> section.
func TestSampleAntiExemplars(t *testing.T) {
	t.Run("extracts shape and reason from a synthetic block", func(t *testing.T) {
		dir := t.TempDir()
		// Self-contained Go file with a single audit comment block.
		// Mirrors the real comments in apophenia.go etc.
		content := `package fixture

var Foo = 1

// 2026-04-27 audit: A polecat ingest claim — ` + "`SubjectX Predicate ObjectY`" + `
// — was deleted here. The predicate was misused: SubjectX is a Place,
// and Predicate requires a Person Subject.

var Bar = 2
`
		if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got := sampleAntiExemplars(dir, 5)
		if len(got) != 1 {
			t.Fatalf("expected 1 anti-exemplar, got %d (%v)", len(got), got)
		}
		if got[0].Shape != "SubjectX Predicate ObjectY" {
			t.Errorf("shape: got %q, want %q", got[0].Shape, "SubjectX Predicate ObjectY")
		}
		if !strings.Contains(got[0].Reason, "predicate was misused") {
			t.Errorf("reason: missing key phrase, got %q", got[0].Reason)
		}
	})

	t.Run("ignores non-audit comments", func(t *testing.T) {
		dir := t.TempDir()
		content := `package fixture

// Plain comment — should be ignored.
// Another plain line.

var Foo = 1
`
		if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		got := sampleAntiExemplars(dir, 5)
		if got != nil {
			t.Errorf("expected nil for no audit blocks, got %v", got)
		}
	})

	t.Run("matches live corpus audit blocks", func(t *testing.T) {
		// Test runs from cmd/metabolism; the corpus is two levels up.
		got := sampleAntiExemplars("../..", 10)
		// Four audit blocks were added 2026-04-27 (see apophenia.go,
		// theory_seeds.go, tunguska.go, predictive_processing.go). At
		// least four anti-exemplars should be mineable.
		if len(got) < 4 {
			t.Errorf("expected >= 4 anti-exemplars from live corpus, got %d (%v)", len(got), got)
		}
		for _, a := range got {
			if a.Shape == "" {
				t.Errorf("anti-exemplar with empty Shape: %v", a)
			}
			if a.Reason == "" {
				t.Errorf("anti-exemplar with empty Reason: %v", a)
			}
		}
	})

	t.Run("respects sample size cap", func(t *testing.T) {
		// Live corpus has 4 audit blocks; n=2 should return exactly 2.
		got := sampleAntiExemplars("../..", 2)
		if len(got) != 2 {
			t.Errorf("expected exactly 2 anti-exemplars (n=2), got %d", len(got))
		}
	})
}

// TestBuildTripPrompt_AntiExemplarsRendered verifies that anti-exemplars
// passed to buildTripPrompt actually surface in the rendered prompt.
// Without this, sampleAntiExemplars could go silently nil-routed (e.g.,
// if the format string drifts).
func TestBuildTripPrompt_AntiExemplarsRendered(t *testing.T) {
	pair := tripPair{
		A: tripEntity{name: "ConceptA", roleType: "Concept", brief: "alpha brief"},
		B: tripEntity{name: "ConceptB", roleType: "Concept", brief: "beta brief"},
	}
	exemplars := []antiExemplar{
		{Shape: "BadSubject BadPredicate BadObject", Reason: "synthetic test reason about predicate misuse"},
	}
	prompt := buildTripPrompt(pair, "analogy", exemplars)
	if !strings.Contains(prompt, "FAILURE-MODE ANTI-EXEMPLARS") {
		t.Error("expected anti-exemplar section header in prompt")
	}
	if !strings.Contains(prompt, "BadSubject BadPredicate BadObject") {
		t.Error("expected anti-exemplar shape to surface in prompt")
	}
	if !strings.Contains(prompt, "synthetic test reason about predicate misuse") {
		t.Error("expected anti-exemplar reason to surface in prompt")
	}
}

// TestBuildTripPrompt_NoAntiExemplars verifies that an empty exemplars
// slice produces no anti-exemplar section (no leftover header text or
// stray formatting).
func TestBuildTripPrompt_NoAntiExemplars(t *testing.T) {
	pair := tripPair{
		A: tripEntity{name: "ConceptA", roleType: "Concept", brief: "alpha brief"},
		B: tripEntity{name: "ConceptB", roleType: "Concept", brief: "beta brief"},
	}
	prompt := buildTripPrompt(pair, "analogy", nil)
	if strings.Contains(prompt, "FAILURE-MODE ANTI-EXEMPLARS") {
		t.Error("expected NO anti-exemplar section for nil input")
	}
}

// TestTokenizeBrief pins the brief-vocabulary tokenizer behavior. The
// signal is sensitive to which tokens survive — an over-aggressive
// stopword filter would zero out the brief-vocab affinity component.
func TestTokenizeBrief(t *testing.T) {
	t.Run("empty brief returns nil", func(t *testing.T) {
		if got := tokenizeBrief(""); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("drops short tokens", func(t *testing.T) {
		// "a", "of", "is" are <4 chars and should be dropped.
		got := tokenizeBrief("a brief is of the consciousness")
		// Survivors: "brief" (5), "consciousness" (13). "the" is 3 → dropped.
		if !got["brief"] || !got["consciousness"] {
			t.Errorf("expected brief+consciousness, got %v", got)
		}
		if got["a"] || got["is"] || got["of"] || got["the"] {
			t.Errorf("expected short stopwords dropped, got %v", got)
		}
	})

	t.Run("lowercases and splits on punctuation", func(t *testing.T) {
		got := tokenizeBrief("Predictive-Processing, the brain's framework!")
		if !got["predictive"] || !got["processing"] || !got["brain"] || !got["framework"] {
			t.Errorf("expected normalized tokens, got %v", got)
		}
	})
}

func TestValidatePredicate(t *testing.T) {
	cases := []struct {
		pred     string
		subjRole string
		objRole  string
		want     bool
	}{
		{"TheoryOf", "Hypothesis", "Concept", true},
		{"TheoryOf", "Concept", "Hypothesis", false}, // reversed
		{"Proposes", "Person", "Hypothesis", true},
		{"Proposes", "Hypothesis", "Person", false},
		{"InfluencedBy", "Person", "Person", true},
		{"InfluencedBy", "Person", "Concept", false},
		{"BelongsTo", "Concept", "Concept", true},
		{"BelongsTo", "Hypothesis", "Concept", false},
		{"BogusPredicate", "Person", "Person", false}, // unknown
	}
	for _, tc := range cases {
		t.Run(tc.pred+"/"+tc.subjRole+"->"+tc.objRole, func(t *testing.T) {
			got := validatePredicate(tc.pred, tc.subjRole, tc.objRole)
			if got != tc.want {
				t.Errorf("validatePredicate(%q, %q, %q) = %v, want %v",
					tc.pred, tc.subjRole, tc.objRole, got, tc.want)
			}
		})
	}
}

func TestCompatiblePredicates(t *testing.T) {
	cases := []struct {
		roleA string
		roleB string
		want  []string
	}{
		{"Person", "Person", []string{"InfluencedBy"}},
		{"Person", "Hypothesis", []string{"Accepts", "Disputes", "Proposes"}},
		{"Hypothesis", "Person", []string{"Accepts", "Disputes", "Proposes"}}, // symmetric
		{"Hypothesis", "Concept", []string{"TheoryOf"}},
		{"Concept", "Concept", []string{"BelongsTo", "CommentaryOn", "DerivedFrom"}},
		{"Place", "Person", []string{}}, // no compatible predicate
	}
	for _, tc := range cases {
		t.Run(tc.roleA+"-"+tc.roleB, func(t *testing.T) {
			got := compatiblePredicates(tc.roleA, tc.roleB)
			if len(got) != len(tc.want) {
				t.Errorf("compatiblePredicates(%q, %q) = %v, want %v", tc.roleA, tc.roleB, got, tc.want)
				return
			}
			for i, p := range tc.want {
				if got[i] != p {
					t.Errorf("compatiblePredicates(%q, %q)[%d] = %q, want %q",
						tc.roleA, tc.roleB, i, got[i], p)
				}
			}
		})
	}
}

// TestTripCompatiblePredicates pins the contract between the trip cycle and
// tripBannedPredicates: the LLM prompt must never offer Person-attribution
// predicates (Proposes, Disputes, Accepts, InfluencedBy). If a future
// predicate is added that would let the trip cycle fabricate attribution,
// this test catches the omission when the predicate gets banned.
func TestTripCompatiblePredicates(t *testing.T) {
	cases := []struct {
		roleA string
		roleB string
		want  []string
	}{
		// Person ↔ Hypothesis used to surface Proposes/Disputes/Accepts.
		// All three are Person-attribution and now filtered.
		{"Person", "Hypothesis", []string{}},
		{"Hypothesis", "Person", []string{}},
		// Person ↔ Person used to surface InfluencedBy.
		// InfluencedBy is biographical attribution; banned.
		{"Person", "Person", []string{}},
		// Concept-relational cases unaffected.
		{"Hypothesis", "Concept", []string{"TheoryOf"}},
		{"Concept", "Concept", []string{"BelongsTo", "CommentaryOn", "DerivedFrom"}},
	}
	for _, tc := range cases {
		t.Run(tc.roleA+"-"+tc.roleB, func(t *testing.T) {
			got := tripCompatiblePredicates(tc.roleA, tc.roleB)
			if len(got) != len(tc.want) {
				t.Errorf("tripCompatiblePredicates(%q, %q) = %v, want %v", tc.roleA, tc.roleB, got, tc.want)
				return
			}
			for i, p := range tc.want {
				if got[i] != p {
					t.Errorf("tripCompatiblePredicates(%q, %q)[%d] = %q, want %q",
						tc.roleA, tc.roleB, i, got[i], p)
				}
			}
		})
	}

	// Independent check: every entry in tripBannedPredicates must be a
	// real predicate (defined in predicateSlots). Catches typos in the
	// banned list — silently dead entries would let attribution slip
	// through.
	for p := range tripBannedPredicates {
		if _, ok := predicateSlots[p]; !ok {
			t.Errorf("tripBannedPredicates entry %q has no predicateSlots entry — dead ban", p)
		}
	}
}

// TestIsReifiedEntityFile pins which corpus files are treated as reify
// output and excluded from trip pair selection. Currently just
// predictions.go (the reify command's only output target). If reify ever
// emits to a sibling file, that file must be added here too — otherwise
// recursive amplification (trip picks a meta-hypothesis as a candidate,
// promotes a claim about it, reify emits a meta-claim about that claim)
// silently re-opens.
func TestIsReifiedEntityFile(t *testing.T) {
	cases := []struct {
		file string
		want bool
	}{
		{"predictions.go", true},
		{"hard_problem.go", false},
		{"metabolism_cycle3.go", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isReifiedEntityFile(tc.file); got != tc.want {
			t.Errorf("isReifiedEntityFile(%q) = %v, want %v", tc.file, got, tc.want)
		}
	}
}

func TestDrugProfile(t *testing.T) {
	cases := []struct {
		temp       float64
		promptType string
		want       string
	}{
		{1.3, "analogy", "psychedelic/pattern-matching"},
		{1.2, "contradiction", "psychedelic/adversarial"},
		{0.8, "genealogy", "exploratory/causal-tracing"},
		{1.0, "prediction", "psychedelic/forecasting"}, // 1.0 >= 0.9 => psychedelic
		{0.4, "analogy", "microdose/pattern-matching"},
		{0.7, "contradiction", "exploratory/adversarial"}, // 0.7 >= 0.6 => exploratory
		{0.2, "genealogy", "sober/causal-tracing"},
		{0.0, "prediction", "sober/forecasting"},
	}
	for _, tc := range cases {
		got := drugProfile(tc.temp, tc.promptType)
		if got != tc.want {
			t.Errorf("drugProfile(%.1f, %q) = %q, want %q", tc.temp, tc.promptType, got, tc.want)
		}
	}
}
