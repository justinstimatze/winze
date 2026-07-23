package main

import "testing"

// TestDocsCoverageThisRepo is the gate as a test: every cmd/ binary in this
// very repo must be named in a doc. It fails the moment someone adds a cmd/
// without a doc mention — the coverage gate, enforced by `go test`.
func TestDocsCoverageThisRepo(t *testing.T) {
	root := repoRoot(t)
	bins, err := listBinaries(root)
	if err != nil {
		t.Fatalf("listBinaries: %v", err)
	}
	if len(bins) == 0 {
		t.Fatal("no binaries found — wrong repo root?")
	}
	docText, err := gatherDocText(root)
	if err != nil {
		t.Fatalf("gatherDocText: %v", err)
	}
	for _, b := range bins {
		if !contains(docText, "winze-"+b) && !contains(docText, "cmd/"+b) {
			t.Errorf("cmd/%s is documented nowhere — add a `winze-%s` or `cmd/%s` mention to a doc", b, b, b)
		}
	}
}
