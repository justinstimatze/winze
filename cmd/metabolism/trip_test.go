package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sharedRoute is a minimal non-zero structural-affinity signal. pickCrossClusterPairs
// drops pairs with zero affinity — no shared 2-hop context, no predicate rhyme, no
// brief vocabulary in common — so a fixture that only means to exercise *scoring*
// must still give its entities some route, or it ends up testing the floor instead
// of the property it names. Uniform across a fixture, so affinity contributes
// equally to every candidate and the property under test is what decides.
var sharedRoute = map[string]bool{"TheoryOf": true}

func TestPickCrossClusterPairs(t *testing.T) {
	t.Run("two clusters", func(t *testing.T) {
		entities := []tripEntity{
			{name: "A1", cluster: 0, brief: "entity a1", predicates: sharedRoute},
			{name: "A2", cluster: 0, brief: "entity a2", predicates: sharedRoute},
			{name: "B1", cluster: 1, brief: "entity b1", predicates: sharedRoute},
			{name: "B2", cluster: 1, brief: "entity b2", predicates: sharedRoute},
		}
		pairs := pickCrossClusterPairs(entities, 2, nil)
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
		pairs := pickCrossClusterPairs(entities, 5, nil)
		if pairs != nil {
			t.Errorf("expected nil for single cluster, got %d pairs", len(pairs))
		}
	})

	t.Run("prefer entities with briefs", func(t *testing.T) {
		entities := []tripEntity{
			{name: "Blank1", cluster: 0, predicates: sharedRoute},
			{name: "Rich1", cluster: 0, brief: "has a brief", predicates: sharedRoute},
			{name: "Blank2", cluster: 1, predicates: sharedRoute},
			{name: "Rich2", cluster: 1, brief: "also has a brief", predicates: sharedRoute},
		}
		pairs := pickCrossClusterPairs(entities, 1, nil)
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
			{name: "A1", cluster: 0, brief: "a1", predicates: sharedRoute},
			{name: "Orphan", cluster: -1, brief: "no cluster", predicates: sharedRoute},
			{name: "B1", cluster: 1, brief: "b1", predicates: sharedRoute},
		}
		pairs := pickCrossClusterPairs(entities, 5, nil)
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
//
// Affinity is held uniform across the fixture on purpose. Since 2026-08-05 the
// rank blends score with affinity rather than treating affinity as a tie-break,
// so bridge bias only decides when the route strength is equal — which is the
// property this test is about. A fixture with varying affinity would be testing
// the blend, and that is TestPickCrossClusterPairs_AffinityOutranksBridge.
func TestPickCrossClusterPairs_BridgeBias(t *testing.T) {
	entities := []tripEntity{
		{name: "BridgeA", cluster: 0, brief: "anchor", bridge: true, predicates: sharedRoute},
		{name: "PlainA", cluster: 0, brief: "plain", predicates: sharedRoute},
		{name: "BridgeB", cluster: 1, brief: "anchor", bridge: true, predicates: sharedRoute},
		{name: "PlainB", cluster: 1, brief: "plain", predicates: sharedRoute},
	}
	pairs := pickCrossClusterPairs(entities, 1, nil)
	if len(pairs) == 0 {
		t.Fatal("expected at least 1 pair")
	}
	p := pairs[0]
	if !p.A.bridge || !p.B.bridge {
		t.Errorf("expected bridge×bridge to surface first, got %s(bridge=%v) ↔ %s(bridge=%v)",
			p.A.name, p.A.bridge, p.B.name, p.B.bridge)
	}
}

// TestPickCrossClusterPairs_AffinityFloor pins the floor: a pair whose
// structural affinity falls below affinityFloor has too little route for an
// analogy to travel, so whatever the generator produces is invented rather than
// found. Such pairs are dropped even when both endpoints are bridges with
// briefs — the maximum primary score.
//
// The weak case here is deliberately not zero-affinity. One shared predicate out
// of three scores 133, which is a real but thin route, and the measured run
// behind affinityFloor found exactly that band ("generic resemblance, not
// structural isomorphism") is what the critic rejects.
func TestPickCrossClusterPairs_AffinityFloor(t *testing.T) {
	entities := []tripEntity{
		// Max score (bridge+brief both sides), thin route: 1 shared predicate
		// of 3 total → 133, under the floor.
		{name: "Thin0", cluster: 0, brief: "a", bridge: true,
			predicates: map[string]bool{"LocatedIn": true, "TheoryOf": true}},
		{name: "Thin1", cluster: 1, brief: "b", bridge: true,
			predicates: map[string]bool{"Disputes": true, "TheoryOf": true}},
	}
	if got := pairStructuralAffinity(entities[0], entities[1]); got >= affinityFloor {
		t.Fatalf("fixture is meant to sit under the floor; scored %d vs floor %d", got, affinityFloor)
	}
	if pairs := pickCrossClusterPairs(entities, 5, nil); len(pairs) != 0 {
		t.Fatalf("expected under-floor bridge×bridge pair to be dropped, got %d: %s↔%s",
			len(pairs), pairs[0].A.name, pairs[0].B.name)
	}

	// Same fixture with matching predicate sets: a full route, so it survives.
	entities[0].predicates = map[string]bool{"TheoryOf": true}
	entities[1].predicates = map[string]bool{"TheoryOf": true}
	if pairs := pickCrossClusterPairs(entities, 5, nil); len(pairs) != 1 {
		t.Fatalf("expected 1 pair once the route clears the floor, got %d", len(pairs))
	}
}

// TestPickCrossClusterPairs_AffinityOutranksBridge is the rebalance itself. Until
// 2026-08-05 affinity was a tie-break only, so a bridge×bridge pair with a barely
// existent route beat a non-bridge pair with a strong one — no amount of semantic
// connection could overcome a 3-point bridge bonus. That ordering is what produced
// shallow promotions like cycle 27's ErrorManagementTheoryOfApophenia ↔
// UDHRAsTheoryOfHumanRights. The rank now blends the two, so a full-strength route
// outranks the bridge bonus.
func TestPickCrossClusterPairs_AffinityOutranksBridge(t *testing.T) {
	entities := []tripEntity{
		// Bridge×bridge, max primary score (8), thin-but-above-floor route:
		// 2 shared predicates of 4 → 200 on that signal, plus a shared 2-hop
		// node, which clears affinityFloor without approaching a full route.
		// It must clear the floor or this test would pass via the floor rather
		// than via the blend it is meant to exercise.
		{name: "BridgeA", cluster: 0, brief: "x", bridge: true,
			predicates: map[string]bool{"TheoryOf": true, "Shared2": true, "P1": true},
			twoHop:     map[string]bool{"N": true, "PA": true}},
		{name: "BridgeB", cluster: 1, brief: "y", bridge: true,
			predicates: map[string]bool{"TheoryOf": true, "Shared2": true, "Q1": true},
			twoHop:     map[string]bool{"N": true, "QB": true}},
		// No bridges, minimal primary score (2), maximal route: identical
		// predicates, identical 2-hop neighborhood, identical brief vocabulary.
		{name: "RoutedA", cluster: 0, brief: "shared vocabulary here",
			predicates:  map[string]bool{"TheoryOf": true},
			twoHop:      map[string]bool{"N": true},
			briefTokens: map[string]bool{"shared": true, "vocabulary": true}},
		{name: "RoutedB", cluster: 1, brief: "shared vocabulary here",
			predicates:  map[string]bool{"TheoryOf": true},
			twoHop:      map[string]bool{"N": true},
			briefTokens: map[string]bool{"shared": true, "vocabulary": true}},
	}

	if got := pairStructuralAffinity(entities[0], entities[1]); got < affinityFloor {
		t.Fatalf("bridge fixture must clear the floor or this tests the floor, not the blend: %d < %d", got, affinityFloor)
	}

	// Run repeatedly: the shuffle must not be what decides this.
	for i := 0; i < 20; i++ {
		pairs := pickCrossClusterPairs(entities, 1, nil)
		if len(pairs) == 0 {
			t.Fatal("expected at least 1 pair")
		}
		if pairs[0].A.bridge || pairs[0].B.bridge {
			t.Fatalf("iter %d: fully-routed non-bridge pair should outrank the thin bridge×bridge pair, got %s↔%s",
				i, pairs[0].A.name, pairs[0].B.name)
		}
	}
}

// TestPickCrossClusterPairs_ClusterDiversity pins the round-robin. Ranking alone
// produced a monoculture on the live corpus: all 25 selected pairs were cluster 0
// × cluster 6, because that one cluster pair saturated the top score class and
// every trip became "some consciousness thesis ↔ some Tunguska hypothesis." The
// budget should spend across the topology instead of exhausting one seam.
func TestPickCrossClusterPairs_ClusterDiversity(t *testing.T) {
	var entities []tripEntity
	for c := 0; c < 4; c++ {
		for i := 0; i < 5; i++ {
			e := tripEntity{
				name: fmt.Sprintf("C%dE%d", c, i), cluster: c,
				brief: "b", predicates: sharedRoute,
			}
			// Cluster 0 and 1 are all bridges, so under pure ranking the 0×1
			// cross-product would take every slot.
			e.bridge = c < 2
			entities = append(entities, e)
		}
	}

	pairs := pickCrossClusterPairs(entities, 6, nil)
	if len(pairs) != 6 {
		t.Fatalf("expected 6 pairs, got %d", len(pairs))
	}
	seen := map[[2]int]bool{}
	for _, p := range pairs {
		a, b := p.A.cluster, p.B.cluster
		if a > b {
			a, b = b, a
		}
		seen[[2]int{a, b}] = true
	}
	if len(seen) < 3 {
		t.Errorf("expected the 6 pairs to span at least 3 cluster pairs, got %d: %v", len(seen), seen)
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

	t.Run("2-hop Jaccard partial overlap", func(t *testing.T) {
		// A reaches {X, Y}, B reaches {X}. Jaccard = 1/2 = 0.5 → 150.
		// Other signals zero (predicates and tokens nil).
		a := tripEntity{name: "A", twoHop: map[string]bool{"X": true, "Y": true}}
		b := tripEntity{name: "B", twoHop: map[string]bool{"X": true}}
		if got := pairStructuralAffinity(a, b); got != 150 {
			t.Errorf("expected 150 (Jaccard 0.5 × 300), got %d", got)
		}
	})

	t.Run("2-hop Jaccard full overlap caps at 3", func(t *testing.T) {
		shared := map[string]bool{"X1": true, "X2": true, "X3": true, "X4": true, "X5": true}
		a := tripEntity{name: "A", twoHop: shared}
		b := tripEntity{name: "B", twoHop: shared}
		// Identical sets → Jaccard 1.0 → cap at 300.
		if got := pairStructuralAffinity(a, b); got != 300 {
			t.Errorf("expected cap at 300, got %d", got)
		}
	})

	t.Run("2-hop Jaccard rejects hub-bias", func(t *testing.T) {
		// Hub entity A has a huge 2-hop neighborhood; partner B has a
		// tiny one with one shared node. Raw count would score 1; with
		// Jaccard normalization the small intersection over a large
		// union scores zero. This is the post-wi-085 fix in action: the
		// hub doesn't auto-win on neighborhood size.
		hub := map[string]bool{}
		for i := 0; i < 50; i++ {
			hub[fmt.Sprintf("hub-neighbor-%d", i)] = true
		}
		hub["shared"] = true
		a := tripEntity{name: "Hub", twoHop: hub}
		b := tripEntity{name: "Tiny", twoHop: map[string]bool{"shared": true}}
		// Jaccard = 1/51 ≈ 0.02 → 5 at milli-scale. The assertion is relative,
		// not exact-zero: what matters is that the hub's diluted overlap ranks
		// far below a genuine partial overlap, not that it rounds away. Before
		// the milli-scale rescale this truncated to 0, which read as a cleaner
		// result than it was — integer division was discarding every weak
		// signal, not just the hub's.
		hubScore := pairStructuralAffinity(a, b)
		genuine := pairStructuralAffinity(
			tripEntity{twoHop: map[string]bool{"X": true, "Y": true}},
			tripEntity{twoHop: map[string]bool{"X": true}},
		)
		if hubScore*10 > genuine {
			t.Errorf("hub overlap (%d) should rank an order of magnitude below a genuine partial overlap (%d)", hubScore, genuine)
		}
	})

	t.Run("predicate complementarity (full overlap)", func(t *testing.T) {
		// Identical predicate sets → Jaccard 1.0 → score 400.
		preds := map[string]bool{"TheoryOf": true, "CommentaryOn": true}
		a := tripEntity{name: "A", predicates: preds}
		b := tripEntity{name: "B", predicates: preds}
		if got := pairStructuralAffinity(a, b); got != 400 {
			t.Errorf("expected 400 (full predicate overlap), got %d", got)
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

	t.Run("brief vocab Jaccard partial", func(t *testing.T) {
		// A: {consciousness, perception}, B: {consciousness}.
		// Jaccard = 1/2 = 0.5 → 150.
		a := tripEntity{name: "A", briefTokens: map[string]bool{"consciousness": true, "perception": true}}
		b := tripEntity{name: "B", briefTokens: map[string]bool{"consciousness": true}}
		if got := pairStructuralAffinity(a, b); got != 150 {
			t.Errorf("expected 150 (Jaccard 0.5 × 300), got %d", got)
		}
	})

	t.Run("brief vocab Jaccard full overlap caps at 3", func(t *testing.T) {
		shared := map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true}
		a := tripEntity{name: "A", briefTokens: shared}
		b := tripEntity{name: "B", briefTokens: shared}
		if got := pairStructuralAffinity(a, b); got != 300 {
			t.Errorf("expected cap at 300, got %d", got)
		}
	})

	t.Run("composite (all three signals, Jaccard)", func(t *testing.T) {
		// All three signals at Jaccard = 1.0: 2-hop +300, predicates +400,
		// tokens +300. Total: affinityMax.
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
		if got := pairStructuralAffinity(a, b); got != affinityMax {
			t.Errorf("expected %d (300+400+300 at Jaccard 1.0), got %d", affinityMax, got)
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
		pairs := pickCrossClusterPairs(entities, 1, nil)
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
		got := sampleAntiExemplars("../../corpus", 10)
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
		got := sampleAntiExemplars("../../corpus", 2)
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

// TestAppendIsolatedConnections pins the NONE-predicate JSONL sink:
// connections with no canonical predicate are persisted so a future
// review can cluster the recurring shapes; named-predicate connections
// are skipped (they go through the normal promotion funnel).
func TestAppendIsolatedConnections(t *testing.T) {
	dir := t.TempDir()
	conns := []TripConnection{
		{EntityA: "ConceptA", EntityB: "ConceptB", Connection: "isomorphism w/o predicate", Score: 4, Predicate: "NONE"},
		{EntityA: "PersonX", EntityB: "HypoY", Connection: "X proposes Y", Score: 4, Predicate: "Proposes"},
		{EntityA: "Hypo1", EntityB: "Hypo2", Connection: "another structural shape", Score: 3, Predicate: ""},
	}
	n := appendIsolatedConnections(dir, conns, "psychedelic/pattern-matching", 0.9, "analogy")
	if n != 2 {
		t.Errorf("expected 2 NONE connections persisted, got %d", n)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".metabolism-trip-isolated.jsonl"))
	if err != nil {
		t.Fatalf("expected JSONL file written: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(string(data), "isomorphism w/o predicate") {
		t.Error("expected first NONE connection in JSONL")
	}
	if !strings.Contains(string(data), "another structural shape") {
		t.Error("expected empty-predicate connection treated as NONE")
	}
	if strings.Contains(string(data), "X proposes Y") {
		t.Error("named-predicate connection leaked into NONE log")
	}

	// Append idempotency: a second call should add 2 more rows, not
	// rewrite the file.
	if n2 := appendIsolatedConnections(dir, conns, "psychedelic/pattern-matching", 0.9, "analogy"); n2 != 2 {
		t.Errorf("second call: expected 2, got %d", n2)
	}
	data2, _ := os.ReadFile(filepath.Join(dir, ".metabolism-trip-isolated.jsonl"))
	lines2 := strings.Split(strings.TrimSpace(string(data2)), "\n")
	if len(lines2) != 4 {
		t.Errorf("expected 4 lines after second append, got %d", len(lines2))
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
		// StructurallyAnalogousTo and Supersedes are both declared over
		// *Entity, so both are compatible with every role pair — including
		// Place↔Person, which has no other predicate. That is the point of the
		// widening: neither predicate says anything about what its endpoints
		// are. Whether trip may *use* either on a given pair is a separate
		// question, answered by tripCompatiblePredicates (Supersedes is banned
		// there regardless of role pair).
		{"Person", "Person", []string{"InfluencedBy", "StructurallyAnalogousTo", "Supersedes"}},
		{"Person", "Hypothesis", []string{"Accepts", "Disputes", "Proposes", "StructurallyAnalogousTo", "Supersedes"}},
		{"Hypothesis", "Person", []string{"Accepts", "Disputes", "Proposes", "StructurallyAnalogousTo", "Supersedes"}}, // symmetric
		{"Hypothesis", "Concept", []string{"StructurallyAnalogousTo", "Supersedes", "TheoryOf"}},
		{"Concept", "Concept", []string{"BelongsTo", "CommentaryOn", "DerivedFrom", "StructurallyAnalogousTo", "Supersedes"}},
		{"Place", "Person", []string{"StructurallyAnalogousTo", "Supersedes"}},
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
		// Place ↔ Person: StructurallyAnalogousTo is role-compatible since the
		// *Entity widening, so the per-predicate ban list would let it through.
		// The role-level Person guard is what stops it, and this case is here to
		// keep that guard honest — it is the one a future wildcard predicate
		// would slip past.
		{"Place", "Person", []string{}},
		// Concept-relational cases now also offer the analogy predicate, which
		// is the whole point: Hypothesis↔Concept pairs were being forced into
		// TheoryOf and refused as predicate misuse.
		{"Hypothesis", "Concept", []string{"StructurallyAnalogousTo", "TheoryOf"}},
		{"Concept", "Concept", []string{"BelongsTo", "CommentaryOn", "DerivedFrom", "StructurallyAnalogousTo"}},
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

// TestPredicateSlotsMatchCorpus pins every hardcoded trip predicateSlots entry
// to what predicates.go actually declares. Catches the slow drift where the
// corpus retypes or renames a predicate and the trip map silently disagrees —
// a disagreement that surfaces only as wrong role-validation, never an error.
func TestPredicateSlotsMatchCorpus(t *testing.T) {
	corpus := collectPredicateSlots(corpusDir(t))
	for name, slots := range predicateSlots {
		got, ok := corpus[name]
		if !ok {
			t.Errorf("predicateSlots has %q but predicates.go does not declare it — stale entry", name)
			continue
		}
		if got[0] != slots.Subject || got[1] != slots.Object {
			t.Errorf("predicateSlots[%q] = {%s→%s}, corpus declares {%s→%s} — drift",
				name, slots.Subject, slots.Object, got[0], got[1])
		}
	}
}

// TestAnalogyPredicatesReachTrip is the regression guard for the bug that
// stranded 48 cross-cluster analogies at NONE: StructurallyAnalogousTo, whose
// entire reason for existing is analogy-mode trip cycles, was absent from the
// trip emit map, so compatiblePredicates("Hypothesis","Hypothesis") returned
// nothing and the predicate enum offered the model only NONE. Every
// [Hypothesis,Hypothesis] predicate in the corpus is inherently a structural
// relation between ideas — trip's home turf — and must reach the emit menu.
func TestAnalogyPredicatesReachTrip(t *testing.T) {
	corpus := collectPredicateSlots(corpusDir(t))
	offered := map[string]bool{}
	for _, p := range tripCompatiblePredicates("Hypothesis", "Hypothesis") {
		offered[p] = true
	}
	for name, slots := range corpus {
		if slots[0] == "Hypothesis" && slots[1] == "Hypothesis" {
			if !offered[name] {
				t.Errorf("corpus predicate %q is [Hypothesis,Hypothesis] but the trip cycle never offers it "+
					"for a Hypothesis pair — every such analogy is forced to NONE. Add it to predicateSlots.", name)
			}
		}
	}
	// Pin the specific predicate this test was written for.
	if !offered["StructurallyAnalogousTo"] {
		t.Error("StructurallyAnalogousTo not offered for a Hypothesis pair — the analogy predicate is dark")
	}
}

// TestSupersedesIsTripBanned locks in Phase 2's explicit trip-eligibility
// decision: the trip cycle must never generate a Supersedes claim, because
// cmd/query/hybrid.go's ranking downranks whatever a Supersedes claim names
// as superseded, so a fabricated one would bury a real, current memory
// behind a hallucinated replacement. Checked directly (membership in
// tripBannedPredicates) and behaviorally (absent from the emit menu for a
// role pair it would otherwise qualify for, since its slots are *Entity).
func TestSupersedesIsTripBanned(t *testing.T) {
	if !tripBannedPredicates["Supersedes"] {
		t.Fatal("Supersedes must be in tripBannedPredicates — a trip-fabricated supersession is retrieval-consequential, not just a provenance risk")
	}
	for _, p := range tripCompatiblePredicates("Concept", "Concept") {
		if p == "Supersedes" {
			t.Error("tripCompatiblePredicates(\"Concept\", \"Concept\") offers Supersedes — the ban list is not filtering it")
		}
	}
}
