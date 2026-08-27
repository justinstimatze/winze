package main

import "testing"

func TestThinConjectureRuleFailsOnEmptyRationale(t *testing.T) {
	src := `package winze

var noReason = Conjecture{GeneratedBy: "trip", From: "lexicon:lex-0165", Rationale: ""}
`
	if rc := thinConjectureRule(writeLintFixture(t, src)); rc != 1 {
		t.Errorf("empty Rationale must fail (rc=1), got rc=%d", rc)
	}
}

func TestThinConjectureRuleFailsOnOmittedRationale(t *testing.T) {
	// Omitting the key entirely is the same zero value as Rationale: "" —
	// the rule must not require the key to be present to notice it's missing.
	src := `package winze

var noKeyAtAll = Conjecture{GeneratedBy: "trip", From: "lexicon:lex-0165"}
`
	if rc := thinConjectureRule(writeLintFixture(t, src)); rc != 1 {
		t.Errorf("omitted Rationale key must fail (rc=1), got rc=%d", rc)
	}
}

func TestThinConjectureRulePassesOnFilledRationale(t *testing.T) {
	src := `package winze

var reasoned = Conjecture{GeneratedBy: "trip", From: "lexicon:lex-0165", Rationale: "both cite the same 1962 survey independently"}
var unrelatedType = Provenance{Origin: "arXiv:1234", Quote: "exact source text"}
`
	if rc := thinConjectureRule(writeLintFixture(t, src)); rc != 0 {
		t.Errorf("filled Rationale must pass (rc=0), got rc=%d", rc)
	}
}

// TestThinConjectureRuleFindsNestedConjecture pins the shape that actually
// occurs in the corpus: cmd/add's --conjecture mode inlines the Conjecture
// into the claim's Prov field (var X = TheoryOf{..., Prov: Conjecture{...}}),
// never as a standalone var. A shallower version of this rule that only
// checked a var's own top-level type found zero hits against the real
// corpus despite a dozen-plus trip-promoted files plainly having them — this
// is the regression test for that miss.
func TestThinConjectureRuleFindsNestedConjecture(t *testing.T) {
	src := `package winze

var TripConnection = TheoryOf{
	Subject: SomeEntity,
	Object:  OtherEntity,
	Prov: Conjecture{
		GeneratedBy: "metabolism-trip",
		Rationale:   "",
	},
}
`
	if rc := thinConjectureRule(writeLintFixture(t, src)); rc != 1 {
		t.Errorf("nested Conjecture with empty Rationale must fail (rc=1), got rc=%d", rc)
	}
}

func TestThinConjectureRulePassesOnFilledNestedConjecture(t *testing.T) {
	src := `package winze

var TripConnection = TheoryOf{
	Subject: SomeEntity,
	Object:  OtherEntity,
	Prov: Conjecture{
		GeneratedBy: "metabolism-trip",
		Rationale:   "both operate at the boundary between representation and reality",
	},
}
`
	if rc := thinConjectureRule(writeLintFixture(t, src)); rc != 0 {
		t.Errorf("nested Conjecture with a filled Rationale must pass (rc=0), got rc=%d", rc)
	}
}
