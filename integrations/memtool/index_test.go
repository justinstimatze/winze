package memtool

import (
	"path/filepath"
	"testing"
)

func TestIndexDeleteRemovesDirectoryContents(t *testing.T) {
	ix := &index{entries: map[string]indexEntry{
		"/memories/dir/a.md": {Var: "MemA"},
		"/memories/dir/b.md": {Var: "MemB"},
		"/memories/other.md": {Var: "MemOther"},
	}}
	ix.delete("/memories/dir")
	if _, ok := ix.get("/memories/dir/a.md"); ok {
		t.Error("/memories/dir/a.md survived a directory delete")
	}
	if _, ok := ix.get("/memories/dir/b.md"); ok {
		t.Error("/memories/dir/b.md survived a directory delete")
	}
	if _, ok := ix.get("/memories/other.md"); !ok {
		t.Error("/memories/other.md was removed by an unrelated delete")
	}
}

func TestIndexListPrefix(t *testing.T) {
	ix := &index{entries: map[string]indexEntry{
		"/memories/dir/a.md":  {Var: "MemA"},
		"/memories/dir/b.md":  {Var: "MemB"},
		"/memories/other.md":  {Var: "MemOther"},
		"/memories/dir2/c.md": {Var: "MemC"},
	}}
	got := ix.listPrefix("/memories/dir")
	want := []string{"/memories/dir/a.md", "/memories/dir/b.md"}
	if len(got) != len(want) {
		t.Fatalf("listPrefix(/memories/dir) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("listPrefix[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIndexRenameMovesDirectoryContents(t *testing.T) {
	ix := &index{entries: map[string]indexEntry{
		"/memories/old/a.md": {Var: "MemA"},
		"/memories/old/b.md": {Var: "MemB"},
		"/memories/other.md": {Var: "MemOther"},
	}}
	n := ix.rename("/memories/old", "/memories/new")
	if n != 2 {
		t.Fatalf("rename moved %d entries, want 2", n)
	}
	if _, ok := ix.get("/memories/old/a.md"); ok {
		t.Error("old path still present after rename")
	}
	if v, ok := ix.get("/memories/new/a.md"); !ok || v.Var != "MemA" {
		t.Errorf("get(/memories/new/a.md) = (%+v, %v), want Var=MemA", v, ok)
	}
	if v, ok := ix.get("/memories/new/b.md"); !ok || v.Var != "MemB" {
		t.Errorf("get(/memories/new/b.md) = (%+v, %v), want Var=MemB", v, ok)
	}
	if v, ok := ix.get("/memories/other.md"); !ok || v.Var != "MemOther" {
		t.Error("unrelated entry disturbed by rename")
	}
}

func TestIndexRoundTripsThroughDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.json")

	ix, err := loadIndex(p)
	if err != nil {
		t.Fatalf("loadIndex on a missing file: %v", err)
	}
	ix.set("/memories/a.md", indexEntry{Var: "MemA", CreationTitle: "/memories/a.md"})
	if err := ix.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := loadIndex(p)
	if err != nil {
		t.Fatalf("loadIndex after save: %v", err)
	}
	if v, ok := reloaded.get("/memories/a.md"); !ok || v.Var != "MemA" {
		t.Errorf("reloaded.get(/memories/a.md) = (%+v, %v), want Var=MemA", v, ok)
	}
}
