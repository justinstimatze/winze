package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodeRefExistenceRuleFlagsRenamedSymbol(t *testing.T) {
	modDir := writeScratchGoModule(t, "func Baz() {}") // renamed from Bar
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "other", Path: "example.com/scratchmod/pkg.Bar"}},
}
`
	if rc := codeRefExistenceRule(writeLintFixture(t, src), map[string]string{"other": modDir}); rc != 1 {
		t.Errorf("a renamed symbol must be flagged (rc=1), got rc=%d", rc)
	}
}

func TestCodeRefExistenceRuleFlagsUnconfiguredClient(t *testing.T) {
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "other", Path: "example.com/otherproj/pkg.Bar"}},
}
`
	// A non-empty clients map that just doesn't name this ref's Client is a
	// real misconfiguration, distinct from "nothing configured at all".
	clients := map[string]string{"unrelated": "/somewhere"}
	if rc := codeRefExistenceRule(writeLintFixture(t, src), clients); rc != 1 {
		t.Errorf("an unconfigured named client must be flagged (rc=1), got rc=%d", rc)
	}
}

// TestCodeRefExistenceRulePassesOnExistingSymbol and
// TestCodeRefExistenceRuleFlagsRenamedSymbol are the requested end-to-end
// scenario: a real scratch Go module, a citation that resolves, then a
// rename that must flip the rule from pass to fail.
func TestCodeRefExistenceRulePassesOnExistingSymbol(t *testing.T) {
	modDir := writeScratchGoModule(t, "func Bar() {}")
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "other", Path: "example.com/scratchmod/pkg.Bar"}},
}
`
	if rc := codeRefExistenceRule(writeLintFixture(t, src), map[string]string{"other": modDir}); rc != 0 {
		t.Errorf("an existing symbol must pass (rc=0), got rc=%d", rc)
	}
}

func TestCodeRefExistenceRuleSkipsWithNoClients(t *testing.T) {
	src := `package winze

var ext = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "other", Path: "example.com/otherproj/pkg.Bar"}},
}
`
	if rc := codeRefExistenceRule(writeLintFixture(t, src), nil); rc != 0 {
		t.Errorf("no clients configured must skip cleanly (rc=0), got rc=%d", rc)
	}
}

// writeScratchGoModule creates a throwaway Go module at module path
// example.com/scratchmod with a single package pkg containing funcSrc, and
// returns the module's root directory.
func writeScratchGoModule(t *testing.T, funcSrc string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/scratchmod\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte("package pkg\n\n"+funcSrc+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
