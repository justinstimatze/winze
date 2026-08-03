package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPairKeyOrderIndependent(t *testing.T) {
	if pairKey("Alpha", "Beta") != pairKey("Beta", "Alpha") {
		t.Fatal("pairKey must be order-independent")
	}
	if pairKey("Alpha", "Beta") == pairKey("Alpha", "Gamma") {
		t.Fatal("distinct pairs must not collide")
	}
}

func TestPairFromCycle(t *testing.T) {
	cases := []struct {
		name  string
		c     Cycle
		wantA string
		wantB string
	}{
		{
			name:  "evidence primary",
			c:     Cycle{Evidence: "EmbodiedMindThesis StructurallyAnalogousTo UDHRAsTheoryOfHumanRights: critic-reject (shallow)"},
			wantA: "EmbodiedMindThesis", wantB: "UDHRAsTheoryOfHumanRights",
		},
		{
			name:  "hypothesis fallback",
			c:     Cycle{Hypothesis: "Foo_CommentaryOn_Bar"},
			wantA: "Foo", wantB: "Bar",
		},
		{
			name:  "empty when unparseable",
			c:     Cycle{Hypothesis: "loneword"},
			wantA: "", wantB: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := pairFromCycle(tc.c)
			if a != tc.wantA || b != tc.wantB {
				t.Fatalf("pairFromCycle = (%q,%q), want (%q,%q)", a, b, tc.wantA, tc.wantB)
			}
		})
	}
}

func TestRefutedPairKeysFromLog(t *testing.T) {
	dir := t.TempDir()
	log := MetabolismLog{Cycles: []Cycle{
		// refuted trip — should be excluded
		{Timestamp: time.Unix(1, 0), VulnType: "trip_promotion", PredictionType: "trip_promotion_attempt",
			Resolution: "refuted", Evidence: "EmbodiedMindThesis StructurallyAnalogousTo UDHRAsTheoryOfHumanRights: critic-reject (shallow)"},
		// a *confirmed* trip — must NOT be excluded (still a valid pair to build on)
		{Timestamp: time.Unix(2, 0), VulnType: "trip_promotion", PredictionType: "trip_promotion_attempt",
			Resolution: "confirmed", Evidence: "KahnemanDualProcessFraming StructurallyAnalogousTo ConradApopheniaClinicalFraming: accept"},
		// a non-trip refuted cycle — unrelated, must not leak in
		{Timestamp: time.Unix(3, 0), VulnType: "sensor", PredictionType: "sensor_probe",
			Resolution: "refuted", Evidence: "SomeClaim contradicted by arxiv:1234"},
	}}
	if err := saveLog(filepath.Join(dir, ".metabolism-log.json"), log); err != nil {
		t.Fatal(err)
	}
	got := refutedPairKeys(dir)
	if !got[pairKey("EmbodiedMindThesis", "UDHRAsTheoryOfHumanRights")] {
		t.Error("refuted trip pair should be in the exclusion set")
	}
	if got[pairKey("KahnemanDualProcessFraming", "ConradApopheniaClinicalFraming")] {
		t.Error("confirmed trip pair must not be excluded")
	}
	if len(got) != 1 {
		t.Errorf("expected exactly 1 excluded pair, got %d: %v", len(got), got)
	}
}

func TestPickCrossClusterPairsExcludesRefuted(t *testing.T) {
	entities := []tripEntity{
		{name: "A1", cluster: 0},
		{name: "A2", cluster: 0},
		{name: "B1", cluster: 1},
		{name: "B2", cluster: 1},
	}
	refuted := map[string]bool{pairKey("A1", "B1"): true}

	pairs := pickCrossClusterPairs(entities, 10, refuted)

	// 4 cross-cluster pairs exist; one is refuted, so 3 survive.
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs after excluding 1 refuted, got %d", len(pairs))
	}
	for _, p := range pairs {
		if pairKey(p.A.name, p.B.name) == pairKey("A1", "B1") {
			t.Fatal("refuted pair A1↔B1 was still selected")
		}
	}

	// Sanity: with no exclusions, all 4 come back.
	if all := pickCrossClusterPairs(entities, 10, nil); len(all) != 4 {
		t.Fatalf("expected 4 pairs with no exclusions, got %d", len(all))
	}
}
