package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justinstimatze/winze/internal/defndb"
)

// writeFile is the fixture shorthand these tests use to lay out a corpus dir.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// shortType is the last dot-segment of a type name. defn qualifies with the
// module path (…/winze/corpus.Entity) and the AST reader has no module to
// qualify with, which is why every consumer of TypeName already splits on "."
// and takes the tail.
func shortType(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}

// bothSources opens the real corpus through each backend, skipping when defn is
// not reachable — CI ingests before it tests, a bare checkout does not.
func bothSources(t *testing.T) (*defndb.Client, *astSource) {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "corpus")
	client, err := defndb.New(dir)
	skipIfNoDefnDB(t, err)
	if err != nil {
		t.Fatalf("defndb.New: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	src, err := newASTSource(dir)
	if err != nil {
		t.Fatalf("newASTSource: %v", err)
	}
	return client, src
}

// TestASTSourceMatchesDefn is the differential a second backend needs: the same
// corpus read both ways yields the same fields, in the same order, with the
// same values.
//
// It matters because the AST reader is not a fallback that only has to be
// close. A meld is read through it and nothing else, so anywhere it disagrees
// with defn is a place a melded store answers differently from the store it was
// melded from — silently, and with no other test in a position to notice.
func TestASTSourceMatchesDefn(t *testing.T) {
	client, src := bothSources(t)

	collect := func(s corpusSource) []string {
		t.Helper()
		var out []string
		if err := s.EachField(nil, func(f *defndb.LiteralField) bool {
			out = append(out, fmt.Sprintf("%s|%s|%s|%s:%d|%q",
				f.DefName, shortType(f.TypeName), f.FieldName, f.SourceFile, f.Line, f.FieldValue))
			return true
		}); err != nil {
			t.Fatalf("EachField: %v", err)
		}
		return out
	}

	want, got := collect(client), collect(src)
	if len(want) == 0 {
		t.Fatal("defn returned no fields — the differential proves nothing")
	}

	shown := 0
	for i := 0; i < len(want) && i < len(got); i++ {
		if want[i] == got[i] {
			continue
		}
		if shown++; shown > 5 {
			t.Errorf("...and more; %d fields compared", min(len(want), len(got)))
			break
		}
		t.Errorf("field %d differs:\n  defn: %s\n   ast: %s", i, want[i], got[i])
	}
	if len(want) != len(got) {
		t.Errorf("field count: defn %d, ast %d", len(want), len(got))
	}
}

// TestASTSourceRoleTypesMatchDefn checks the set that decides which vars count
// as entities at all. A role type missing here demotes every var built from it
// to a claim, and the whole read side follows.
func TestASTSourceRoleTypesMatchDefn(t *testing.T) {
	client, src := bothSources(t)

	want, err := client.RoleTypeSet()
	if err != nil {
		t.Fatalf("RoleTypeSet: %v", err)
	}
	got, _ := src.RoleTypeSet()
	for r := range want {
		if !got[r] {
			t.Errorf("ast reader missing role type %q", r)
		}
	}
	for r := range got {
		if !want[r] {
			t.Errorf("ast reader invented role type %q", r)
		}
	}
}

// TestASTSourceEntityVarsMatchDefn covers the call the entity index is seeded
// from. It is separate from the field stream on purpose: a var absent here is an
// entity missing from every read path even when all of its fields came through.
func TestASTSourceEntityVarsMatchDefn(t *testing.T) {
	client, src := bothSources(t)

	wantVars, err := client.EntityVarsWithRoles()
	if err != nil {
		t.Fatalf("EntityVarsWithRoles: %v", err)
	}
	gotVars, _ := src.EntityVarsWithRoles()

	index := func(vs []defndb.VarRoleInfo) map[string]string {
		m := make(map[string]string, len(vs))
		for _, v := range vs {
			m[v.VarName] = v.RoleType + " in " + filepath.Base(v.SourceFile)
		}
		return m
	}
	want, got := index(wantVars), index(gotVars)
	for name, role := range want {
		switch g, ok := got[name]; {
		case !ok:
			t.Errorf("ast reader missing entity var %s (%s)", name, role)
		case g != role:
			t.Errorf("entity var %s: defn says %s, ast says %s", name, role, g)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("ast reader invented entity var %s (%s)", name, got[name])
		}
	}
}

// TestOpenCorpusPicksTheASTReaderForAMeld pins the routing. The manifest is the
// whole signal: melds are the corpora defn cannot ingest, and choosing on a
// failed defn open instead would turn a stale index or a broken ingest into a
// quiet switch to a different reader.
func TestOpenCorpusPicksTheASTReaderForAMeld(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "x.go"), "package winze\n\nvar Zero = Concept{&Entity{ID: \"z\"}}\n")
	writeFile(t, filepath.Join(dir, meldManifestName), `{"version":1,"stores":[],"primary":"x"}`)

	src, err := openCorpus(dir)
	if err != nil {
		t.Fatalf("openCorpus: %v", err)
	}
	defer src.Close()
	if _, ok := src.(*astSource); !ok {
		t.Fatalf("openCorpus returned %T for a meld, want *astSource", src)
	}

	var ids []string
	if err := src.EachField([]string{"ID"}, func(f *defndb.LiteralField) bool {
		ids = append(ids, f.DefName+"="+f.FieldValue)
		return true
	}); err != nil {
		t.Fatalf("EachField: %v", err)
	}
	if len(ids) != 1 || ids[0] != "Zero=z" {
		t.Errorf("fields from the meld = %v, want [Zero=z]", ids)
	}
}

// TestFieldValueUnquotesStringsAndKeepsSourceOtherwise pins the one rendering
// rule the read paths depend on: Subject and Object are matched by value
// prefix, so an identifier has to come through bare.
func TestFieldValueUnquotesStringsAndKeepsSourceOtherwise(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "c.go"), `package winze

var Src = Provenance{Origin: "Wikipedia", Quote: "a" + "b"}

var C = TheoryOf{
	Subject: Alpha.Entity,
	Object:  Beta,
	Prov: Conjecture{
		GeneratedBy: "trip",
	},
}
`)
	writeFile(t, filepath.Join(dir, meldManifestName), `{"version":1}`)

	src, err := newASTSource(dir)
	if err != nil {
		t.Fatalf("newASTSource: %v", err)
	}
	got := map[string]string{}
	types := map[string]string{}
	if err := src.EachField(nil, func(f *defndb.LiteralField) bool {
		got[f.FieldName] = f.FieldValue
		types[f.FieldName] = f.TypeName
		return true
	}); err != nil {
		t.Fatalf("EachField: %v", err)
	}

	for field, want := range map[string]string{
		"Origin":      "Wikipedia",
		"Quote":       "ab",
		"Subject":     "Alpha.Entity",
		"Object":      "Beta",
		"GeneratedBy": "trip",
	} {
		if got[field] != want {
			t.Errorf("%s = %q, want %q", field, got[field], want)
		}
	}
	// The inline literal reports its own type, not the claim's, which is what
	// lets a provenance read find it.
	if types["GeneratedBy"] != "Conjecture" {
		t.Errorf("GeneratedBy TypeName = %q, want Conjecture", types["GeneratedBy"])
	}
	if types["Subject"] != "TheoryOf" {
		t.Errorf("Subject TypeName = %q, want TheoryOf", types["Subject"])
	}
	if !strings.HasPrefix(got["Prov"], "Conjecture{") {
		t.Errorf("Prov = %q, want the inline literal's source text", got["Prov"])
	}
}
