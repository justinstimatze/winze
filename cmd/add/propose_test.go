package main

import (
	"testing"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

func TestExtractJSON(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                        `{"a":1}`,
		"```json\n{\"a\":1}\n```":        `{"a":1}`,
		"Here you go:\n{\"a\":1}\nDone.": `{"a":1}`,
		"no json here":                   "no json here",
	}
	for in, want := range cases {
		if got := extractJSON(in); got != want {
			t.Errorf("extractJSON(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIdentTokens(t *testing.T) {
	got := identTokens("SelectiveEvidence")
	if !got["selective"] || !got["evidence"] || len(got) != 2 {
		t.Errorf("SelectiveEvidence tokens = %v", got)
	}
	// Spaces, hyphens, and single-letter noise are handled.
	got = identTokens("Clustering illusion")
	if !got["clustering"] || !got["illusion"] {
		t.Errorf("spaced tokens = %v", got)
	}
}

func TestNearestEntitiesFindsOverlap(t *testing.T) {
	ents := []corpusparse.Entity{
		{VarName: "ConfirmationBias", Name: "Confirmation bias"},
		{VarName: "SelectiveEvidence", Name: "Selective evidence"},
		{VarName: "Apophenia", Name: "Apophenia"},
	}
	// "SelectiveEvidenceBias" shares tokens with SelectiveEvidence (2) and
	// ConfirmationBias (1); the stronger overlap must rank first.
	near := nearestEntities("SelectiveEvidenceBias", ents, 3)
	if len(near) == 0 || near[0] != "SelectiveEvidence" {
		t.Errorf("nearest = %v, want SelectiveEvidence first", near)
	}
}

func TestSanitizeIdent(t *testing.T) {
	cases := map[string]string{
		"GoodName":     "GoodName",
		"has spaces":   "hasspaces",
		"weird-chars!": "weirdchars",
		"9leading":     "leading", // a leading digit is stripped, the usable rest kept
		"99":           "",        // nothing usable survives → empty (caller falls back)
		"":             "",
	}
	for in, want := range cases {
		if got := sanitizeIdent(in); got != want {
			t.Errorf("sanitizeIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueNameFallbackAndCollision(t *testing.T) {
	used := map[string]bool{"ShermerProposes": true, "ShermerProposes2": true}
	// Empty name falls back to Subject+Predicate, then suffixes past collisions.
	got := uniqueName("", proposal{Subject: "Shermer", Predicate: "Proposes"}, used)
	if got != "ShermerProposes3" {
		t.Errorf("uniqueName collision = %q, want ShermerProposes3", got)
	}
	// A fresh name passes through unchanged.
	if got := uniqueName("FreshClaim", proposal{}, used); got != "FreshClaim" {
		t.Errorf("uniqueName fresh = %q, want FreshClaim", got)
	}
}
