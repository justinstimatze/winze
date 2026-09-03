package main

import (
	"path/filepath"
	"testing"
)

func TestCodeRefMutualExclusionFailsOnSymbolAndClient(t *testing.T) {
	src := `package winze

var bad = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Symbol: something.Thing, Client: "other-repo", Path: "pkg.Thing"}},
}
`
	if rc := codeRefMutualExclusionRule(writeLintFixture(t, src)); rc != 1 {
		t.Errorf("Symbol+Client must fail (rc=1), got rc=%d", rc)
	}
}

func TestCodeRefMutualExclusionPassesOnClientOnly(t *testing.T) {
	src := `package winze

var ok = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Client: "other-repo", Path: "pkg.Thing"}},
}
`
	if rc := codeRefMutualExclusionRule(writeLintFixture(t, src)); rc != 0 {
		t.Errorf("Client alone must pass (rc=0), got rc=%d", rc)
	}
}

// TestCodeRefMutualExclusionPassesOnRealCorpus pins the 2 real CodeRef
// instances (corpus/winze_self.go) forever. Uses filepath.Join(repoRoot(t),
// "corpus") rather than bare repoRoot(t) — see FEEDBACK-2026-09-02 plan notes
// on why repoRoot(t) alone silently scans zero files since the corpus/
// directory move.
func TestCodeRefMutualExclusionPassesOnRealCorpus(t *testing.T) {
	if rc := codeRefMutualExclusionRule(filepath.Join(repoRoot(t), "corpus")); rc != 0 {
		t.Errorf("real corpus must pass (rc=0), got rc=%d", rc)
	}
}

func TestCodeRefMutualExclusionPassesOnSymbolOnly(t *testing.T) {
	src := `package winze

var ok = SourceDoc{
	Entity: &Entity{ID: "x", Name: "x", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{{Symbol: something.Thing, Path: "pkg.Thing"}},
}
`
	if rc := codeRefMutualExclusionRule(writeLintFixture(t, src)); rc != 0 {
		t.Errorf("Symbol alone must pass (rc=0), got rc=%d", rc)
	}
}
