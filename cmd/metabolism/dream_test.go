package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func dreamRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

func TestAnalyzeFileBalance(t *testing.T) {
	root := dreamRepoRoot(t)
	findings := analyzeFileBalance(root)

	// The real corpus has files of varying sizes — should produce some findings
	if len(findings) == 0 {
		t.Log("no file balance findings (all files are reasonably balanced)")
	}

	for _, f := range findings {
		if f.Category == "" {
			t.Errorf("finding has empty Category: %+v", f)
		}
		if f.Description == "" {
			t.Errorf("finding has empty Description: %+v", f)
		}
	}
	t.Logf("analyzeFileBalance: %d findings", len(findings))
}
