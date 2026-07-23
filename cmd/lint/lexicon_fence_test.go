package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLintFixture(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLexiconFenceFailsOnQuotedLexicon(t *testing.T) {
	// A Provenance whose Origin or Quote references a lexicon locator is a leak.
	src := `package winze

var leakOrigin = Provenance{Origin: "lexicon:lex-0165", Quote: "some text"}
var leakQuote = Provenance{Origin: "arXiv:1234", Quote: "borrowed from lex-0788 gloss"}
`
	if rc := lexiconFenceRule(writeLintFixture(t, src)); rc != 1 {
		t.Errorf("lexicon-quoting Provenance must fail (rc=1), got rc=%d", rc)
	}
}

func TestLexiconFencePassesSanctionedAndClean(t *testing.T) {
	// The sanctioned path (Conjecture carrying a lexicon locator — no Quote
	// field exists on it) and ordinary sourced Provenance both pass. A
	// Conjecture is not a Provenance, so the rule never scans it.
	src := `package winze

var normalSource = Provenance{Origin: "Wikipedia / Apophenia", Quote: "exact source text"}
var sparked = Conjecture{GeneratedBy: "trip", From: "lexicon:lex-0165", Rationale: "sparked by the pattern"}
`
	if rc := lexiconFenceRule(writeLintFixture(t, src)); rc != 0 {
		t.Errorf("sanctioned Conjecture + clean Provenance must pass (rc=0), got rc=%d", rc)
	}
}

func TestLexiconLocatorPrecision(t *testing.T) {
	// The locator pattern must catch atom-ids and the lexicon: prefix without
	// firing on the ordinary English word "lexicon" in prose.
	hits := []string{"lex-0165", "lex-788", "see lexicon:lex-0001", "LEX-1234"}
	for _, s := range hits {
		if !lexiconLocator.MatchString(s) {
			t.Errorf("should match locator %q", s)
		}
	}
	misses := []string{"a lexicon of terms", "lexical priming", "lexicographer", "index lex of terms"}
	for _, s := range misses {
		if lexiconLocator.MatchString(s) {
			t.Errorf("should NOT match ordinary prose %q", s)
		}
	}
}
