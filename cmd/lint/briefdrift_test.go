package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// collectMentionPragmas must read //winze:mentions from all three positions:
// a doc comment on the spec, a trailing line comment, and (for a single-spec
// `var x = ...`) the GenDecl doc comment.
func TestCollectMentionPragmas(t *testing.T) {
	src := `package winze

var (
	// winze:mentions HardProblemOfConsciousness,IntegratedInformationTheory
	DocForm = Person{&Entity{Name: "Doc Form"}}

	LineForm = Person{&Entity{Name: "Line Form"}} //winze:mentions Advaita
)

//winze:mentions KurtGodel
var SingleSpec = Person{&Entity{Name: "Single Spec"}}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := collectMentionPragmas(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got["DocForm"]["HardProblemOfConsciousness"] || !got["DocForm"]["IntegratedInformationTheory"] {
		t.Errorf("DocForm exemptions wrong: %v", got["DocForm"])
	}
	if !got["LineForm"]["Advaita"] {
		t.Errorf("LineForm (trailing comment) not parsed: %v", got["LineForm"])
	}
	if !got["SingleSpec"]["KurtGodel"] {
		t.Errorf("SingleSpec (GenDecl doc) not parsed: %v", got["SingleSpec"])
	}
	if len(got["LineForm"]) != 1 {
		t.Errorf("LineForm should have exactly one exemption, got %v", got["LineForm"])
	}
}

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
