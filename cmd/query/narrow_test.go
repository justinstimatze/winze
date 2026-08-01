package main

import (
	"testing"

	"github.com/justinstimatze/winze/internal/defndb"
)

// claimKey identifies a claim for set comparison independent of slice order.
func claimKey(c claimRecord) string {
	return c.VarName + "|" + c.Predicate + "|" + c.Subject + "|" + c.Object + "|" + c.ProvRef
}

// TestNarrowClaimsMatchFullIndex proves the single-entity narrow path
// (claimRecordsInvolving) returns exactly the claims the whole-corpus index path
// (claimsInvolving over buildIndex) returns, for every entity var. That equality
// is what lets --claims stop building the full index: the narrow query is a
// faithful substitute, not an approximation.
func TestNarrowClaimsMatchFullIndex(t *testing.T) {
	root := repoRoot(t)
	client, err := defndb.New(root)
	skipIfNoDefnDB(t, err)
	if err != nil {
		t.Fatalf("defndb.New: %v", err)
	}
	defer client.Close()

	kb, err := buildIndex(root)
	if err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	tested, nonEmpty := 0, 0
	for _, e := range kb.Entities {
		v := e.VarName
		want := claimsInvolving(kb, v)
		got, err := claimRecordsInvolving(client, v)
		if err != nil {
			t.Fatalf("claimRecordsInvolving(%q): %v", v, err)
		}

		wantSet := make(map[string]bool, len(want))
		for _, c := range want {
			wantSet[claimKey(c)] = true
		}
		gotSet := make(map[string]bool, len(got))
		for _, c := range got {
			gotSet[claimKey(c)] = true
		}
		for k := range wantSet {
			if !gotSet[k] {
				t.Errorf("%s: narrow path missing claim %s", v, k)
			}
		}
		for k := range gotSet {
			if !wantSet[k] {
				t.Errorf("%s: narrow path has extra claim %s", v, k)
			}
		}
		tested++
		if len(got) > 0 {
			nonEmpty++
		}
	}
	if nonEmpty < 20 {
		t.Errorf("only %d entities had claims; expected many — the narrow path may not be exercised", nonEmpty)
	}
	t.Logf("compared %d entity vars, %d with claims", tested, nonEmpty)
}

// TestLoadOrderDeterministic locks in that the read-path loaders return the same
// order across independent Client instances. The map-funnelled predecessor
// inherited Go's randomised iteration order, which made a fuzzy --claims target
// resolve to a different entity from one run to the next.
func TestLoadOrderDeterministic(t *testing.T) {
	root := repoRoot(t)

	c1, err := defndb.New(root)
	skipIfNoDefnDB(t, err)
	if err != nil {
		t.Fatalf("defndb.New: %v", err)
	}
	e1, _, err := loadEntities(c1)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}
	cl1, err := loadClaims(c1)
	if err != nil {
		t.Fatalf("loadClaims: %v", err)
	}

	c2, err := defndb.New(root)
	if err != nil {
		t.Fatalf("defndb.New: %v", err)
	}
	e2, _, err := loadEntities(c2)
	if err != nil {
		t.Fatalf("loadEntities: %v", err)
	}
	cl2, err := loadClaims(c2)
	if err != nil {
		t.Fatalf("loadClaims: %v", err)
	}

	if len(e1) != len(e2) {
		t.Fatalf("entity count differs across instances: %d vs %d", len(e1), len(e2))
	}
	for i := range e1 {
		if e1[i].VarName != e2[i].VarName {
			t.Fatalf("entity order differs at %d: %q vs %q", i, e1[i].VarName, e2[i].VarName)
		}
	}
	if len(cl1) != len(cl2) {
		t.Fatalf("claim count differs across instances: %d vs %d", len(cl1), len(cl2))
	}
	for i := range cl1 {
		if cl1[i].VarName != cl2[i].VarName {
			t.Fatalf("claim order differs at %d: %q vs %q", i, cl1[i].VarName, cl2[i].VarName)
		}
	}
}
