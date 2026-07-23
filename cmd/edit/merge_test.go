package main

import (
	"os"
	"path/filepath"
	"testing"
)

// mergeFixture exercises both declaration shapes (a grouped var block and a
// standalone var), a claim whose Subject/Object must retarget, and prose that
// contains the entity's textual name and must NOT be touched.
const mergeFixture = `package winze

var (
	KlausConrad = Person{&Entity{
		Name: "Klaus Conrad",
	}}

	MichaelShermer = Person{&Entity{
		Name: "Michael Shermer",
	}}
)

var LoneConcept = Concept{&Entity{
	Name: "Lone",
}}

var CanonConcept = Concept{&Entity{
	Name: "Canon",
}}

var ConradClaim = Proposes{
	Subject: KlausConrad,
	Object:  LoneConcept,
	Prov:    &Provenance{Quote: "Klaus Conrad named it; LoneConcept appears here as prose"},
}
`

func writeMergeFixture(t *testing.T) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "corpus.go")
	if err := os.WriteFile(path, []byte(mergeFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func applyPlanToFixture(t *testing.T, dir, path, from, into string) string {
	t.Helper()
	plan, err := planMerge(dir, from, into)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.fromDeclared {
		t.Fatalf("%s should be declared", from)
	}
	if !plan.intoDeclared {
		t.Fatalf("%s should be declared", into)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(applyEdits(src, plan.edits[path]))
}

func TestPlanMerge_GroupSpecRemoval(t *testing.T) {
	dir, path := writeMergeFixture(t)
	got := applyPlanToFixture(t, dir, path, "KlausConrad", "MichaelShermer")

	mustContain := []string{
		`MichaelShermer = Person`,       // survivor's declaration intact
		`Subject: MichaelShermer,`,      // claim retargeted from KlausConrad
		`Object:  LoneConcept,`,         // unrelated reference untouched
		`Quote: "Klaus Conrad named it`, // prose untouched (string literal)
	}
	for _, w := range mustContain {
		if !contains(got, w) {
			t.Errorf("expected output to contain %q\n--- got ---\n%s", w, got)
		}
	}
	mustNotContain := []string{
		`KlausConrad = Person`, // declaration removed
		`Name: "Klaus Conrad"`, // decl body removed
		`Subject: KlausConrad`, // no stale reference
	}
	for _, w := range mustNotContain {
		if contains(got, w) {
			t.Errorf("expected output NOT to contain %q\n--- got ---\n%s", w, got)
		}
	}
	// MichaelShermer's block must still be inside the group parens.
	if !contains(got, "var (") || !contains(got, ")") {
		t.Errorf("group parens should survive removing one spec:\n%s", got)
	}
}

func TestPlanMerge_StandaloneRemoval(t *testing.T) {
	dir, path := writeMergeFixture(t)
	got := applyPlanToFixture(t, dir, path, "LoneConcept", "CanonConcept")

	if contains(got, "var LoneConcept = Concept") {
		t.Errorf("standalone declaration should be removed:\n%s", got)
	}
	if !contains(got, "var CanonConcept = Concept") {
		t.Errorf("survivor standalone declaration should remain:\n%s", got)
	}
	if !contains(got, "Object:  CanonConcept,") {
		t.Errorf("claim Object should retarget to CanonConcept:\n%s", got)
	}
	// The exact ident "LoneConcept" appears inside a Quote string — must survive.
	if !contains(got, "LoneConcept appears here as prose") {
		t.Errorf("prose containing the ident name must be untouched:\n%s", got)
	}
}

func TestPlanMerge_IntoNotDeclared(t *testing.T) {
	dir, _ := writeMergeFixture(t)
	plan, err := planMerge(dir, "KlausConrad", "NoSuchEntity")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.fromDeclared {
		t.Error("KlausConrad should be declared")
	}
	if plan.intoDeclared {
		t.Error("NoSuchEntity should not be declared")
	}
}

func TestPlanMerge_FromNotDeclared(t *testing.T) {
	dir, _ := writeMergeFixture(t)
	plan, err := planMerge(dir, "NoSuchEntity", "MichaelShermer")
	if err != nil {
		t.Fatal(err)
	}
	if plan.fromDeclared {
		t.Error("NoSuchEntity should not be declared")
	}
}

func TestApplyEdits_DeletionAndRename(t *testing.T) {
	src := []byte("ABCDEFGHIJ")
	// Delete "CD" (offset 2, len 2) and rename "GH" (offset 6, len 2) -> "xyz".
	out := string(applyEdits(src, []edit{
		{offset: 2, length: 2, repl: ""},
		{offset: 6, length: 2, repl: "xyz"},
	}))
	if out != "ABEFxyzIJ" {
		t.Errorf("got %q, want %q", out, "ABEFxyzIJ")
	}
}

func TestLineHelpers(t *testing.T) {
	src := []byte("line0\nline1\nline2\n")
	if got := lineStart(src, 8); got != 6 { // offset 8 is inside line1
		t.Errorf("lineStart=%d, want 6", got)
	}
	if got := lineStart(src, 2); got != 0 { // inside line0
		t.Errorf("lineStart=%d, want 0", got)
	}
	if got := lineEndAfterNewline(src, 8); got != 12 { // past line1's newline
		t.Errorf("lineEndAfterNewline=%d, want 12", got)
	}
	if got := lineEndAfterNewline(src, 100); got != len(src) { // clamps at EOF
		t.Errorf("lineEndAfterNewline=%d, want %d", got, len(src))
	}
}
