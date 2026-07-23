package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReifyEntityCount_Fixture(t *testing.T) {
	dir := t.TempDir()
	// Two entity decls (Event, Hypothesis, both RoleType{&Entity{...}}) and one
	// claim decl (Predicts, which references entities but embeds no &Entity).
	// The counter must count the two entities and ignore the claim.
	src := `package winze

var EvidenceSearchFoo = Event{&Entity{ID: "e1", Name: "Foo", Kind: "event"}}
var MetaHypoBar = Hypothesis{&Entity{ID: "h1", Name: "Bar", Kind: "hypothesis"}}
var PredFoo = Predicts{Subject: MetaHypoBar, Object: EvidenceSearchFoo}
`
	if err := os.WriteFile(filepath.Join(dir, "predictions.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := reifyEntityCount(dir); got != 2 {
		t.Errorf("reifyEntityCount = %d, want 2 (two entities, claim ignored)", got)
	}
}

func TestReifyEntityCount_AbsentFileFailsOpen(t *testing.T) {
	if got := reifyEntityCount(t.TempDir()); got != 0 {
		t.Errorf("no predictions.go → want 0, got %d", got)
	}
}

// TestReifyEntityCountRepo pins the real corpus: predictions.go's reify Events
// are the reason the entity count sits over cap, and this is the number the
// cap fix subtracts. A change here means reify's output shape shifted.
func TestReifyEntityCountRepo(t *testing.T) {
	n := reifyEntityCount(repoRoot(t))
	if n < 50 {
		t.Errorf("reifyEntityCount(repo) = %d, expected the ~97 reify Events — did predictions.go move or shrink?", n)
	}
	t.Logf("reify bookkeeping entities in predictions.go: %d", n)
}
