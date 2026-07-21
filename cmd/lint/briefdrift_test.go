package main

import (
	"testing"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

func TestBuildMentionMatchers_SurfaceSelection(t *testing.T) {
	entities := []corpusparse.Entity{
		{VarName: "StevenPinker", Name: "Steven Pinker", Aliases: []string{"Pinker"}},
		{VarName: "TheSelf", Name: "Self"},                                  // too short
		{VarName: "LongHypothesis", Name: longName()},                       // sentence-length Name
		{VarName: "TripLintFoo", Name: "Trip Lint Foo"},                     // reify machinery
		{VarName: "Dup", Name: "Duplicate", Aliases: []string{"Duplicate"}}, // repeated surface
	}

	byVar := map[string]mentionMatcher{}
	for _, m := range buildMentionMatchers(entities) {
		byVar[m.varName] = m
	}

	if _, ok := byVar["TheSelf"]; ok {
		t.Error("surface shorter than minSurfaceLen should be skipped")
	}
	if _, ok := byVar["LongHypothesis"]; ok {
		t.Error("sentence-length Name should be skipped: it is prose, not a handle")
	}
	if _, ok := byVar["TripLintFoo"]; ok {
		t.Error("reify machinery should be skipped")
	}

	m, ok := byVar["StevenPinker"]
	if !ok {
		t.Fatal("expected a matcher for StevenPinker")
	}
	for _, tc := range []struct {
		brief string
		want  string
	}{
		{"argued against by Steven Pinker in 2002", "Steven Pinker"},
		{"Pinker disputes this", "Pinker"},
		{"Pinkerton is unrelated", ""}, // word boundary
		{"see also PINKER", ""},        // case-sensitive: entity names are proper nouns
	} {
		if got := m.re.FindString(tc.brief); got != tc.want {
			t.Errorf("brief %q: got %q, want %q", tc.brief, got, tc.want)
		}
	}
}

func longName() string {
	// 60+ chars: the shape hypothesis entities use, where Name is a full claim.
	return "Structural fragility of a hypothesis predicts where curation gaps appear"
}
