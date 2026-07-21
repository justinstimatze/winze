package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The corpus shape that makes textual rename unsafe: the identifier appears
// as a var name, as a reference, as a substring of a longer identifier, and
// inside prose (a Brief, a quoted source, and a comment).
const fixture = `package winze

// Apophenia is discussed in the comment here.
var Apophenia = Concept{&Entity{
	Name:  "Apophenia",
	Brief: "Apophenia is the tendency to perceive connections.",
}}

var ApopheniaClinicalFraming = Concept{&Entity{Name: "framing"}}

var SomeClaim = TheoryOf{
	Subject: ApopheniaClinicalFraming,
	Object:  Apophenia,
	Prov:    &Provenance{Quote: "Apophenia, as Conrad named it"},
}
`

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFindRenameSites_IgnoresProseAndSubstrings(t *testing.T) {
	dir := writeFixture(t)

	sites, declared, collides, err := findRenameSites(dir, "Apophenia", "Pareidolia")
	if err != nil {
		t.Fatal(err)
	}
	if !declared {
		t.Fatal("expected Apophenia to be declared")
	}
	if collides {
		t.Error("Pareidolia is not declared; should not report a collision")
	}

	total := 0
	for _, offs := range sites {
		total += len(offs)
	}
	// Exactly two *identifiers*: the declaration and the single reference
	// (Object: Apophenia). The Name string, the Brief, the Quote, the
	// comment, and ApopheniaClinicalFraming must all be untouched — those are
	// the five sites a textual rename would corrupt.
	if total != 2 {
		t.Errorf("got %d identifier sites, want 2 (declaration + 1 reference)", total)
	}
}

func TestFindRenameSites_DetectsCollision(t *testing.T) {
	dir := writeFixture(t)
	_, _, collides, err := findRenameSites(dir, "Apophenia", "SomeClaim")
	if err != nil {
		t.Fatal(err)
	}
	if !collides {
		t.Error("renaming onto an existing var should report a collision")
	}
}

func TestSplice_RewritesOnlyGivenOffsets(t *testing.T) {
	dir := writeFixture(t)
	src, err := os.ReadFile(filepath.Join(dir, "corpus.go"))
	if err != nil {
		t.Fatal(err)
	}
	sites, _, _, err := findRenameSites(dir, "Apophenia", "Pareidolia")
	if err != nil {
		t.Fatal(err)
	}
	got := string(splice(src, sites[filepath.Join(dir, "corpus.go")], "Apophenia", "Pareidolia"))

	for _, want := range []string{
		`Name:  "Apophenia"`,                     // string literal untouched
		`Brief: "Apophenia is the tendency`,      // prose untouched
		`Quote: "Apophenia, as Conrad named it"`, // quoted source untouched
		`// Apophenia is discussed`,              // comment untouched
		`var ApopheniaClinicalFraming`,           // substring identifier untouched
		`var Pareidolia = Concept`,               // declaration renamed
		`Object:  Pareidolia,`,                   // reference renamed
	} {
		if !contains(got, want) {
			t.Errorf("expected output to contain %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestIsIdent(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"Foo", true},
		{"_foo9", true},
		{"", false},
		{"9Foo", false},
		{"has-dash", false},
		{"has space", false},
		{"func", false}, // keyword
	} {
		if got := isIdent(tc.in); got != tc.want {
			t.Errorf("isIdent(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
