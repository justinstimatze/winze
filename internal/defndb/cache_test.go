package defndb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A stale index that silently answers with old content is the worst failure
// mode a knowledge base can have — worse than being slow, because the caller
// cannot tell. Every test here is about that: the cached answer must equal the
// answer computed from scratch, under every mutation the corpus undergoes.

const corpusA = `package winze

type Person struct{ *Entity }
type Concept struct{ *Entity }

var Alice = Person{&Entity{ID: "alice", Name: "Alice", Brief: "A person."}}
var Ideas = Concept{&Entity{ID: "ideas", Name: "Ideas", Brief: "A concept."}}

var srcOne = Provenance{Origin: "book one", Quote: "quoted text one"}

//winze:contested
var AliceProposesIdeas = Proposes{Subject: Alice, Object: Ideas, Prov: srcOne}
`

const corpusB = `package winze

var Bob = Person{&Entity{ID: "bob", Name: "Bob", Brief: "Another person."}}

var srcTwo = Provenance{Origin: "book two", Quote: "quoted text two"}

var BobProposesIdeas = Proposes{Subject: Bob, Object: Ideas, Prov: srcTwo}
`

func setup(t *testing.T, files map[string]string) (dir string) {
	t.Helper()
	dir = t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("WINZE_NO_INDEX", "")
	return dir
}

// snapshot renders the whole index as a comparable string.
func snapshot(t *testing.T, dir string) string {
	t.Helper()
	c, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := c.load()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, rt := range idx.roleTypes {
		b.WriteString("role " + rt.Name + " " + filepath.Base(rt.SourceFile) + "\n")
	}
	for _, ev := range idx.entityVars {
		b.WriteString("entity " + ev.VarName + " " + ev.RoleType + "\n")
	}
	for _, d := range idx.defs {
		b.WriteString("def " + d.Name + "\n")
	}
	idx.eachLiteral(func(f *LiteralField) bool {
		b.WriteString("lit " + f.DefName + "." + f.FieldName + "=" + f.FieldValue + " [" + f.TypeName + "]\n")
		return true
	})
	for _, p := range idx.pragmas {
		b.WriteString("pragma " + p.DefName + " " + p.Key + "\n")
	}
	return b.String()
}

// uncached computes the same snapshot with the cache disabled — the ground
// truth every cached result is checked against.
func uncached(t *testing.T, dir string) string {
	t.Helper()
	t.Setenv("WINZE_NO_INDEX", "1")
	defer t.Setenv("WINZE_NO_INDEX", "")
	return snapshot(t, dir)
}

// touch rewrites a file and forces its mtime forward, so the test does not
// depend on filesystem timestamp granularity.
func touch(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(p, future, future); err != nil {
		t.Fatal(err)
	}
}

func TestCachedMatchesUncached(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA, "b.go": corpusB})
	want := uncached(t, dir)

	first := snapshot(t, dir) // cold: parses, writes cache
	second := snapshot(t, dir)
	if first != want {
		t.Errorf("cold index differs from uncached:\n%s\n---\n%s", first, want)
	}
	if second != want {
		t.Errorf("warm index differs from uncached:\n%s\n---\n%s", second, want)
	}
}

func TestEditedFileIsReparsed(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA, "b.go": corpusB})
	snapshot(t, dir) // prime

	touch(t, dir, "b.go", strings.Replace(corpusB, `Name: "Bob"`, `Name: "Roberta"`, 1))

	got := snapshot(t, dir)
	if !strings.Contains(got, ".Name=Roberta ") {
		t.Errorf("edit not picked up:\n%s", got)
	}
	// Only the Name field changed; the var is still Bob, so the Subject field
	// legitimately still reads Bob. Assert on the field that actually changed.
	if strings.Contains(got, ".Name=Bob ") {
		t.Errorf("stale Name survived:\n%s", got)
	}
	if got != uncached(t, dir) {
		t.Error("index after edit differs from uncached")
	}
}

// A same-size edit is the case (size, mtime) invalidation has to catch via the
// timestamp alone.
func TestSameSizeEditIsReparsed(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA})
	snapshot(t, dir)

	edited := strings.Replace(corpusA, `Name: "Alice"`, `Name: "Alicf"`, 1)
	if len(edited) != len(corpusA) {
		t.Fatal("test setup: edit changed the file size")
	}
	touch(t, dir, "a.go", edited)

	if got := snapshot(t, dir); !strings.Contains(got, "Alicf") {
		t.Errorf("same-size edit not picked up:\n%s", got)
	}
}

func TestAddedAndDeletedFiles(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA})
	snapshot(t, dir)

	touch(t, dir, "b.go", corpusB)
	got := snapshot(t, dir)
	if !strings.Contains(got, "entity Bob") {
		t.Errorf("added file not picked up:\n%s", got)
	}
	if got != uncached(t, dir) {
		t.Error("index after add differs from uncached")
	}

	if err := os.Remove(filepath.Join(dir, "b.go")); err != nil {
		t.Fatal(err)
	}
	got = snapshot(t, dir)
	if strings.Contains(got, "entity Bob") {
		t.Errorf("deleted file still in index:\n%s", got)
	}
	if got != uncached(t, dir) {
		t.Error("index after delete differs from uncached")
	}
	// A second warm read must also not resurrect it from a stale cache.
	if got2 := snapshot(t, dir); got2 != got {
		t.Error("deleted file reappeared on the next read")
	}
}

// A role type added in one file changes how vars in *other* files are
// classified. That is why classification is deferred to the merge instead of
// being cached per fragment — this test is the reason.
func TestRoleTypeAddedElsewhereReclassifiesCachedFiles(t *testing.T) {
	dir := setup(t, map[string]string{
		"roles.go": "package winze\n\ntype Person struct{ *Entity }\n",
		"data.go":  "package winze\n\nvar Zed = Hypothesis{&Entity{ID: \"zed\", Name: \"Zed\"}}\n",
	})
	if got := snapshot(t, dir); strings.Contains(got, "entity Zed") {
		t.Fatalf("Hypothesis is not a role type yet:\n%s", got)
	}

	// data.go is unchanged and will be served from cache; roles.go is not.
	touch(t, dir, "roles.go", "package winze\n\ntype Person struct{ *Entity }\ntype Hypothesis struct{ *Entity }\n")

	got := snapshot(t, dir)
	if !strings.Contains(got, "entity Zed Hypothesis") {
		t.Errorf("cached file was not reclassified after a role type was added elsewhere:\n%s", got)
	}
	if got != uncached(t, dir) {
		t.Error("index differs from uncached after role change")
	}
}

// A cache is an optimisation. Corruption must cost a reparse, never an error
// and never a wrong answer.
func TestCorruptCacheFallsBackToParsing(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA})
	want := snapshot(t, dir)

	p := cachePath(dir)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("cache was not written: %v", err)
	}
	for _, junk := range [][]byte{
		{},
		[]byte("not a winze index"),
		[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	} {
		if err := os.WriteFile(p, junk, 0o644); err != nil {
			t.Fatal(err)
		}
		if got := snapshot(t, dir); got != want {
			t.Errorf("corrupt cache (%d bytes) produced a wrong index", len(junk))
		}
	}

	// Truncation partway through a valid file is the realistic corruption.
	full, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, frac := range []int{2, 3, 4, 8} {
		if err := os.WriteFile(p, full[:len(full)/frac], 0o644); err != nil {
			t.Fatal(err)
		}
		if got := snapshot(t, dir); got != want {
			t.Errorf("cache truncated to 1/%d produced a wrong index", frac)
		}
	}
}

// A cache written for another corpus must never be served, even if the path
// hash collides or WINZE_INDEX is pointed at the wrong file.
func TestCacheIsRejectedForADifferentCorpus(t *testing.T) {
	dirA := setup(t, map[string]string{"a.go": corpusA})
	shared := filepath.Join(t.TempDir(), "shared.bin")
	t.Setenv("WINZE_INDEX", shared)

	wantA := snapshot(t, dirA)

	dirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirB, "b.go"), []byte(corpusB), 0o644); err != nil {
		t.Fatal(err)
	}
	gotB := snapshot(t, dirB)
	if strings.Contains(gotB, "entity Alice") {
		t.Errorf("another corpus's cache was served:\n%s", gotB)
	}
	// And dirA still answers correctly after dirB overwrote the shared file.
	if got := snapshot(t, dirA); got != wantA {
		t.Error("corpus A's index was corrupted by corpus B sharing the cache path")
	}
}

func TestVersionMismatchDiscardsCache(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA})
	want := snapshot(t, dir)

	p := cachePath(dir)
	full, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	full[0] = byte(cacheVersion + 7) // first varint is the version
	if err := os.WriteFile(p, full, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, dir); got != want {
		t.Error("version mismatch did not force a reparse")
	}
}

// The cache must be a pure optimisation: unreadable or unwritable cache
// locations degrade to parsing, never to failure.
func TestUnwritableCacheDirIsNotFatal(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA})
	want := uncached(t, dir)

	t.Setenv("WINZE_INDEX", filepath.Join(dir, "nonexistent-subdir", "sub", "index.bin"))
	if got := snapshot(t, dir); got != want {
		t.Error("index was wrong when the cache could not be written")
	}

	// A directory where the cache file should be: open succeeds, read fails.
	block := filepath.Join(t.TempDir(), "index.bin")
	if err := os.MkdirAll(block, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WINZE_INDEX", block)
	if got := snapshot(t, dir); got != want {
		t.Error("index was wrong when the cache path was a directory")
	}
}

func TestUnparseableFileDoesNotPoisonTheCache(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA})
	want := snapshot(t, dir)

	// A corpus mid-edit. The valid file must still answer, and the broken file
	// must not be cached as empty.
	touch(t, dir, "broken.go", "package winze\n\nvar X = Person{&Entity{")
	if got := snapshot(t, dir); !strings.Contains(got, "entity Alice") {
		t.Errorf("valid files stopped answering because another file was broken:\n%s", got)
	}

	touch(t, dir, "broken.go", "package winze\n\nvar Y = Person{&Entity{ID: \"y\", Name: \"Y\"}}\n")
	got := snapshot(t, dir)
	if !strings.Contains(got, "entity Y") {
		t.Errorf("repaired file was not picked up — it had been cached as empty:\n%s", got)
	}
	if !strings.Contains(got, "entity Alice") {
		t.Error("original content lost")
	}
	_ = want
}

func TestRoundTripCodec(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA, "b.go": corpusB})
	paths, keys, err := scanFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	frags, err := parseFragments(paths, keys)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.CreateTemp(t.TempDir(), "idx")
	if err != nil {
		t.Fatal(err)
	}
	if err := encodeCache(f, dir, paths, frags); err != nil {
		t.Fatal(err)
	}
	f.Close()

	in, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	gotDir, gotFrags, err := decodeCache(in)
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != dir {
		t.Errorf("dir = %q, want %q", gotDir, dir)
	}
	if len(gotFrags) != len(frags) {
		t.Fatalf("fragment count %d, want %d", len(gotFrags), len(frags))
	}
	for p, want := range frags {
		got := gotFrags[p]
		if got.Key != want.Key {
			t.Errorf("%s: key %+v, want %+v", p, got.Key, want.Key)
		}
		if len(got.Literals) != len(want.Literals) {
			t.Fatalf("%s: %d literals, want %d", p, len(got.Literals), len(want.Literals))
		}
		for i := range want.Literals {
			if got.Literals[i] != want.Literals[i] {
				t.Errorf("%s literal %d:\n got %+v\nwant %+v", p, i, got.Literals[i], want.Literals[i])
			}
		}
		if len(got.Pragmas) != len(want.Pragmas) {
			t.Errorf("%s: %d pragmas, want %d", p, len(got.Pragmas), len(want.Pragmas))
		}
		if len(got.RoleTypes) != len(want.RoleTypes) || len(got.Defs) != len(want.Defs) ||
			len(got.Candidates) != len(want.Candidates) {
			t.Errorf("%s: fragment shape mismatch", p)
		}
	}
}

// The encoded cache must be byte-identical for an unchanged corpus, or every
// read rewrites it and the file churns for no reason.
func TestEncodingIsDeterministic(t *testing.T) {
	dir := setup(t, map[string]string{"a.go": corpusA, "b.go": corpusB})
	paths, keys, _ := scanFiles(dir)
	frags, _ := parseFragments(paths, keys)

	var prev []byte
	for i := 0; i < 3; i++ {
		p := filepath.Join(t.TempDir(), "idx")
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := encodeCache(f, dir, paths, frags); err != nil {
			t.Fatal(err)
		}
		f.Close()
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && string(b) != string(prev) {
			t.Fatal("encoding is not deterministic")
		}
		prev = b
	}
}

// Index order feeds query output order, and the previous implementation walked
// Go maps directly.
func TestIndexOrderIsStable(t *testing.T) {
	files := map[string]string{}
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		files[n+".go"] = strings.ReplaceAll(corpusB, "Bob", strings.ToUpper(n)+"ob")
	}
	files["roles.go"] = "package winze\n\ntype Person struct{ *Entity }\n"
	dir := setup(t, files)

	first := uncached(t, dir)
	for i := 0; i < 5; i++ {
		if got := uncached(t, dir); got != first {
			t.Fatal("index order varies between runs")
		}
	}
}
