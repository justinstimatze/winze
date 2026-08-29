package memtool

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// TestExecutorCreateOverwritesExisting pins the approved design decision: a
// create on a path that already exists is a hard overwrite via winze_update,
// not a new entity plus a Supersedes link.
func TestExecutorCreateOverwritesExisting(t *testing.T) {
	e, fb := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/a.md", FileText: "v1"})
	if len(fb.content) != 1 {
		t.Fatalf("after first create: %d vars, want 1", len(fb.content))
	}
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/a.md", FileText: "v2"})
	if len(fb.content) != 1 {
		t.Fatalf("after overwrite: %d vars, want 1 (hard overwrite, no new entity)", len(fb.content))
	}
	got, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/a.md"})
	if isErr || got != "v2" {
		t.Errorf("view after overwrite = (%q, err=%v), want v2", got, isErr)
	}
}

func TestExecutorCreateThenView(t *testing.T) {
	e, _ := newTestExecutor(t)
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "create", Path: "/memories/a.md", FileText: "hello",
	}); isErr {
		t.Fatal("create returned an error")
	}
	got, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "view", Path: "/memories/a.md",
	})
	if isErr {
		t.Fatalf("view returned an error: %s", got)
	}
	if got != "hello" {
		t.Errorf("view = %q, want %q", got, "hello")
	}
}

func TestExecutorDeleteThenViewNotFound(t *testing.T) {
	e, _ := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/a.md", FileText: "hi"})
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "delete", Path: "/memories/a.md"}); isErr {
		t.Fatal("delete of an existing path returned an error")
	}
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/a.md"}); !isErr {
		t.Error("view after delete should error (not found)")
	}
}

func TestExecutorInsertAtLine(t *testing.T) {
	e, _ := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/a.md", FileText: "one\ntwo\nthree"})

	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "insert", Path: "/memories/a.md", InsertLine: 1, InsertText: "inserted",
	}); isErr {
		t.Fatal("insert at a valid line returned an error")
	}
	got, _ := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/a.md"})
	want := "one\ninserted\ntwo\nthree"
	if got != want {
		t.Errorf("content after insert = %q, want %q", got, want)
	}
}

func TestExecutorRenameThenView(t *testing.T) {
	e, _ := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/old.md", FileText: "hi"})
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "rename", OldPath: "/memories/old.md", NewPath: "/memories/new.md",
	}); isErr {
		t.Fatal("rename returned an error")
	}
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/old.md"}); !isErr {
		t.Error("old path should 404 after rename")
	}
	got, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/new.md"})
	if isErr || got != "hi" {
		t.Errorf("view of new path after rename = (%q, err=%v), want hi", got, isErr)
	}
}

func TestExecutorStrReplaceRequiresUniqueMatch(t *testing.T) {
	e, _ := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/a.md", FileText: "foo bar foo"})

	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "str_replace", Path: "/memories/a.md", OldStr: "missing", NewStr: "x",
	}); !isErr {
		t.Error("str_replace with no match should error")
	}
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "str_replace", Path: "/memories/a.md", OldStr: "foo", NewStr: "x",
	}); !isErr {
		t.Error("str_replace matching twice should error, not pick one arbitrarily")
	}
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "str_replace", Path: "/memories/a.md", OldStr: "bar", NewStr: "baz",
	}); isErr {
		t.Error("str_replace with exactly one match should succeed")
	}
	got, _ := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/a.md"})
	if got != "foo baz foo" {
		t.Errorf("content after str_replace = %q, want %q", got, "foo baz foo")
	}
}

func TestExecutorUnknownCommand(t *testing.T) {
	e, _ := newTestExecutor(t)
	if _, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "wat"}); !isErr {
		t.Error("an unrecognized command should report an error, not succeed silently")
	}
}

func TestExecutorViewDirectoryListsChildren(t *testing.T) {
	e, _ := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/dir/a.md", FileText: "a"})
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/dir/b.md", FileText: "b"})
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/other.md", FileText: "c"})

	got, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "view", Path: "/memories/dir"})
	if isErr {
		t.Fatalf("view of a directory returned an error: %s", got)
	}
	for _, want := range []string{"/memories/dir/a.md", "/memories/dir/b.md"} {
		if !contains(got, want) {
			t.Errorf("directory listing %q missing %q", got, want)
		}
	}
	if contains(got, "/memories/other.md") {
		t.Errorf("directory listing %q leaked an unrelated file", got)
	}
}

func TestExecutorViewRangeSlicesLines(t *testing.T) {
	e, _ := newTestExecutor(t)
	e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{Command: "create", Path: "/memories/a.md", FileText: "one\ntwo\nthree\nfour"})

	got, isErr := e.Execute(anthropic.BetaMemoryTool20250818CommandUnion{
		Command: "view", Path: "/memories/a.md", ViewRange: []int64{2, 3},
	})
	if isErr {
		t.Fatalf("view with a range returned an error: %s", got)
	}
	if want := "two\nthree"; got != want {
		t.Errorf("ranged view = %q, want %q", got, want)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{content: map[string]string{}, title: map[string]string{}}
}

func newTestExecutor(t *testing.T) (*Executor, *fakeBackend) {
	t.Helper()
	fb := newFakeBackend()
	ix := &index{path: filepath.Join(t.TempDir(), "index.json"), entries: map[string]indexEntry{}}
	return &Executor{ix: ix, be: fb}, fb
}

func (f *fakeBackend) fullBrief(title string) (string, bool, error) {
	for v, t := range f.title {
		if t == title {
			return f.content[v], true, nil
		}
	}
	return "", false, nil
}

func (f *fakeBackend) remember(note, title string) (string, error) {
	if f.rememberErr != nil {
		return "", f.rememberErr
	}
	f.nextID++
	v := fmt.Sprintf("Mem%d", f.nextID)
	f.content[v] = note
	f.title[v] = title
	return v, nil
}

func (f *fakeBackend) update(varName, note string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.content[varName]; !ok {
		return fmt.Errorf("no such var %s", varName)
	}
	f.content[varName] = note
	return nil
}

// fakeBackend is an in-memory stand-in for cliBackend, so command logic is
// testable without a real winze binary or store.
type fakeBackend struct {
	nextID      int
	content     map[string]string // varName -> content
	title       map[string]string // varName -> title used at creation
	rememberErr error
	updateErr   error
}
