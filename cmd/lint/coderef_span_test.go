package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCodeRefSpanRuleChecksInStoreRefWithoutClients proves a Client==""
// span (a hash-checked citation to a file inside the store's own repo) is
// checked even with no --clients configured — it never needed external
// resolution. The target file lives beside the corpus/ directory, not
// inside it, matching Path's existing "relative to the store's own root"
// convention (corpus.CodeRef's doc comment, and goDirective in
// cmd/agent/init.go for the same filepath.Dir(corpusDir) pattern).
func TestCodeRefSpanRuleChecksInStoreRefWithoutClients(t *testing.T) {
	storeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(storeRoot, "target.txt"), []byte("line one\nline two\nline three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	corpusDir := filepath.Join(storeRoot, "corpus")
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hash := hashLine("line two")
	src := `package winze

var inStore = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Path: "target.txt", Span: &CodeSpan{Line: 2, Hash: "` + hash + `"}}},
}
`
	if err := os.WriteFile(filepath.Join(corpusDir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := codeRefSpanRule(corpusDir, nil); rc != 0 {
		t.Errorf("a matching in-store span must pass (rc=0), got rc=%d", rc)
	}
}

// TestCodeRefSpanRuleFlagsHashMismatch is the requested end-to-end scenario:
// a real client checkout, a citation that matches it, then a mutation that
// must flip the rule from pass to fail.
func TestCodeRefSpanRuleFlagsHashMismatch(t *testing.T) {
	clientDir := t.TempDir()
	targetFile := filepath.Join(clientDir, "register.c")
	if err := os.WriteFile(targetFile, []byte("line one\nfree_list_invariant();\nline three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash := hashLine("free_list_invariant();")

	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "wovim", Path: "register.c", Span: &CodeSpan{Line: 2, Hash: "` + hash + `"}}},
}
`
	corpusDir := writeLintFixture(t, src)
	clients := map[string]string{"wovim": clientDir}

	if rc := codeRefSpanRule(corpusDir, clients); rc != 0 {
		t.Fatalf("matching hash must pass (rc=0), got rc=%d", rc)
	}

	if err := os.WriteFile(targetFile, []byte("line one\nrenamed_fn();\nline three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := codeRefSpanRule(corpusDir, clients); rc != 1 {
		t.Errorf("a mutated line must be flagged (rc=1), got rc=%d", rc)
	}
}

func TestCodeRefSpanRuleFlagsLineOutOfRange(t *testing.T) {
	clientDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(clientDir, "short.c"), []byte("one line only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "wovim", Path: "short.c", Span: &CodeSpan{Line: 99, Hash: "whatever"}}},
}
`
	if rc := codeRefSpanRule(writeLintFixture(t, src), map[string]string{"wovim": clientDir}); rc != 1 {
		t.Errorf("a line past EOF must be flagged (rc=1), got rc=%d", rc)
	}
}

func TestCodeRefSpanRuleFlagsMissingFile(t *testing.T) {
	clientDir := t.TempDir()
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "wovim", Path: "does-not-exist.c", Span: &CodeSpan{Line: 1, Hash: "whatever"}}},
}
`
	if rc := codeRefSpanRule(writeLintFixture(t, src), map[string]string{"wovim": clientDir}); rc != 1 {
		t.Errorf("a missing file must be flagged (rc=1), got rc=%d", rc)
	}
}

func TestCodeRefSpanRulePassesWithNoSpanRefs(t *testing.T) {
	src := `package winze

var ok = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Symbol: something.Thing, Path: "pkg.Thing"}},
}
`
	if rc := codeRefSpanRule(writeLintFixture(t, src), nil); rc != 0 {
		t.Errorf("no Span refs must pass (rc=0), got rc=%d", rc)
	}
}

func TestCodeRefSpanRuleSkipsUnconfiguredExternalClient(t *testing.T) {
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "wovim", Path: "register.c", Span: &CodeSpan{Line: 1, Hash: "whatever"}}},
}
`
	if rc := codeRefSpanRule(writeLintFixture(t, src), nil); rc != 0 {
		t.Errorf("an unconfigured external client must skip cleanly (rc=0), got rc=%d", rc)
	}
}
