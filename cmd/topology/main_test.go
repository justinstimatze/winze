package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

func TestFindSingleSource(t *testing.T) {
	claims := []claimInfo{
		{name: "AliceProposes", predicateType: "Proposes", subject: "Alice", object: "HypA"},
		{name: "BobProposes", predicateType: "Proposes", subject: "Bob", object: "HypA"},
		{name: "CarolProposes", predicateType: "Proposes", subject: "Carol", object: "HypB"},
	}

	vulns := findSingleSource(claims)

	// HypA has 2 proposers → not flagged
	// HypB has 1 proposer → flagged
	found := false
	for _, v := range vulns {
		if v.Entity == "HypA" {
			t.Error("HypA should not be flagged (has 2 proposers)")
		}
		if v.Entity == "HypB" {
			found = true
			if v.Type != "single_source" {
				t.Errorf("vuln type = %q, want %q", v.Type, "single_source")
			}
		}
	}
	if !found {
		t.Error("HypB should be flagged as single_source")
	}
}

func TestFindUncontested(t *testing.T) {
	claims := []claimInfo{
		{name: "P1", predicateType: "Proposes", subject: "Alice", object: "HypA"},
		{name: "D1", predicateType: "Disputes", subject: "Bob", object: "HypA"},
		{name: "P2", predicateType: "Proposes", subject: "Carol", object: "HypB"},
	}

	vulns := findUncontested(claims)

	// HypA is proposed AND disputed → not flagged
	// HypB is proposed but NOT disputed → flagged
	for _, v := range vulns {
		if v.Entity == "HypA" {
			t.Error("HypA should not be flagged (has a dispute)")
		}
	}
	found := false
	for _, v := range vulns {
		if v.Entity == "HypB" {
			found = true
			if v.Type != "uncontested" {
				t.Errorf("vuln type = %q, want %q", v.Type, "uncontested")
			}
		}
	}
	if !found {
		t.Error("HypB should be flagged as uncontested")
	}
}

func TestBuildAdjacency(t *testing.T) {
	claims := []claimInfo{
		{predicateType: "Proposes", subject: "Alice", object: "HypA"},
		{predicateType: "Disputes", subject: "Bob", object: "HypA"},
		{predicateType: "TheoryOf", subject: "HypA", object: "ConceptX"},
	}

	adj := buildAdjacency(claims)

	// Alice and HypA should be connected
	if !adj["Alice"]["HypA"] {
		t.Error("expected edge Alice → HypA")
	}
	if !adj["HypA"]["Alice"] {
		t.Error("expected edge HypA → Alice")
	}
	// Bob and HypA should be connected
	if !adj["Bob"]["HypA"] {
		t.Error("expected edge Bob → HypA")
	}
	// HypA and ConceptX should be connected
	if !adj["HypA"]["ConceptX"] {
		t.Error("expected edge HypA → ConceptX")
	}
	// Alice and Bob should NOT be directly connected
	if adj["Alice"]["Bob"] {
		t.Error("unexpected direct edge Alice → Bob")
	}
}

func TestFindBridgeEntities(t *testing.T) {
	// Create a graph where removing "Bridge" disconnects A-cluster from B-cluster
	entities := []entityInfo{
		{name: "A1"}, {name: "A2"}, {name: "Bridge"}, {name: "B1"}, {name: "B2"},
	}
	claims := []claimInfo{
		{predicateType: "Proposes", subject: "A1", object: "A2"},
		{predicateType: "Proposes", subject: "A2", object: "Bridge"},
		{predicateType: "Proposes", subject: "Bridge", object: "B1"},
		{predicateType: "Proposes", subject: "B1", object: "B2"},
	}
	adj := buildAdjacency(claims)

	vulns := findBridgeEntities(entities, claims, adj)

	found := false
	for _, v := range vulns {
		if v.Entity == "Bridge" {
			found = true
			if v.Type != "bridge_entity" {
				t.Errorf("vuln type = %q, want %q", v.Type, "bridge_entity")
			}
		}
	}
	if !found {
		t.Error("Bridge should be flagged as a bridge entity")
	}
}

func TestAnalyzeRealCorpus(t *testing.T) {
	root := repoRoot(t)
	report, err := analyze(root, 250)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	if report.Entities < 200 {
		t.Errorf("entities = %d, expected at least 200 (corpus has ~233)", report.Entities)
	}
	if report.Claims < 200 {
		t.Errorf("claims = %d, expected at least 200 (corpus has ~281)", report.Claims)
	}
	if report.Edges < 100 {
		t.Errorf("edges = %d, expected at least 100", report.Edges)
	}
	if report.Clusters < 2 {
		t.Errorf("clusters = %d, expected at least 2", report.Clusters)
	}
	if len(report.Vulnerabilities) == 0 {
		t.Error("expected at least some vulnerabilities in the real corpus")
	}

	// At least one bridge entity should exist in a multi-cluster corpus
	foundBridge := false
	for _, v := range report.Vulnerabilities {
		if v.Type == "bridge_entity" {
			foundBridge = true
			break
		}
	}
	if !foundBridge {
		t.Error("expected at least one bridge entity")
	}

	t.Logf("analyze: %d entities, %d claims, %d edges, %d clusters, %d vulns",
		report.Entities, report.Claims, report.Edges, report.Clusters, len(report.Vulnerabilities))
}

func TestCountClusters(t *testing.T) {
	cases := []struct {
		name string
		adj  map[string]map[string]bool
		entities []entityInfo
		want int
	}{
		{
			name: "two disconnected components",
			entities: []entityInfo{
				{name: "A"}, {name: "B"}, {name: "C"}, {name: "D"},
			},
			adj: map[string]map[string]bool{
				"A": {"B": true},
				"B": {"A": true},
				"C": {"D": true},
				"D": {"C": true},
			},
			want: 2,
		},
		{
			name: "fully connected",
			entities: []entityInfo{
				{name: "A"}, {name: "B"}, {name: "C"},
			},
			adj: map[string]map[string]bool{
				"A": {"B": true, "C": true},
				"B": {"A": true, "C": true},
				"C": {"A": true, "B": true},
			},
			want: 1,
		},
		{
			name: "all isolated",
			entities: []entityInfo{
				{name: "A"}, {name: "B"}, {name: "C"},
			},
			adj: map[string]map[string]bool{},
			// Isolated nodes have no edges → 0 clusters. countClusters counts
			// connected components of size >= 2; singletons are not clusters.
			want: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countClusters(tc.entities, tc.adj)
			if got != tc.want {
				t.Errorf("countClusters() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestFindBridgeEntities already exists above (line 106).
// Additional coverage via TestCountClusters.
