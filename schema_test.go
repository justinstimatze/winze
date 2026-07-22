package winze

import (
	"reflect"
	"testing"
)

// A sourced Provenance and a Conjecture report their epistemic status
// distinctly, and both satisfy Attribution so either can back a claim.
func TestAttributionConjectural(t *testing.T) {
	var sourced Attribution = Provenance{Origin: "Wikipedia", Quote: "..."}
	var conj Attribution = Conjecture{GeneratedBy: "metabolism-trip"}
	if sourced.Conjectural() {
		t.Error("Provenance must report Conjectural() == false")
	}
	if !conj.Conjectural() {
		t.Error("Conjecture must report Conjectural() == true")
	}
}

// A real predicate accepts a Conjecture in its Attribution slot: the
// generated-knowledge write path type-checks exactly like the sourced one.
func TestConjectureBacksAClaim(t *testing.T) {
	h1 := Hypothesis{&Entity{ID: "h1", Name: "A"}}
	h2 := Hypothesis{&Entity{ID: "h2", Name: "B"}}
	c := StructurallyAnalogousTo{
		Subject: h1,
		Object:  h2,
		Prov: Conjecture{
			GeneratedBy: "metabolism-trip",
			From:        []*Entity{h1.Entity, h2.Entity},
			PromptType:  "analogy",
			Score:       4,
			Rationale:   "both hypotheses share the same epistemic structure",
		},
	}
	if !c.Prov.Conjectural() {
		t.Error("a claim backed by a Conjecture should report Conjectural()")
	}
}

// The fence. Conjecture has NO Quote field, so a generated claim can never wear
// a fabricated source attribution — `Conjecture{Quote: "..."}` does not compile.
// This test guards that structural property: a future edit that reintroduces a
// Quote field (reopening the trip-fabrication failure mode) breaks the build.
func TestConjectureHasNoQuoteField(t *testing.T) {
	if _, ok := reflect.TypeOf(Conjecture{}).FieldByName("Quote"); ok {
		t.Fatal("Conjecture must NOT have a Quote field — that field's absence is the fence against fabricated attribution")
	}
}
