package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo commits files into a fresh repo and returns its path and HEAD SHA.
func gitRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-q", "-m", "fixture")
	return dir, run("rev-parse", "HEAD")
}

const corpusFile = "package winze\n\nvar A = 1\n"

// TestDetectCorpusAcrossLayouts pins the contract docs/meld.md states: a store's
// corpus is the one directory declaring `package winze`, wherever it sits. The
// two layouts here are the two that exist — this repo nests its corpus under
// corpus/ so defn ingest skips cmd/ and internal/, and a store scaffolded by
// `winze-agent init` keeps the same files flat so `go build .` at its root is a
// working gate. A meld has to take either without being told which.
func TestDetectCorpusAcrossLayouts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		files   map[string]string
		wantDir string
		wantN   int
	}{
		{
			name: "nested under corpus/",
			files: map[string]string{
				"corpus/schema.go": corpusFile,
				"corpus/roles.go":  corpusFile,
				"cmd/tool/main.go": "package main\n\nfunc main() {}\n",
				"internal/x/x.go":  "package x\n",
				"go.mod":           "module example.com/w\n\ngo 1.26\n",
				"docs/notes.md":    "not go\n",
			},
			wantDir: "corpus",
			wantN:   2,
		},
		{
			name: "flat at the repo root",
			files: map[string]string{
				"schema.go": corpusFile,
				"memory.go": corpusFile,
				"go.mod":    "module example.com/store\n\ngo 1.26\n",
			},
			wantDir: ".",
			wantN:   2,
		},
		{
			// The corpus dir is still found from a _test.go declaring the same
			// package, but the file itself is not melded — a meld carries
			// corpus content, not the store's own gate.
			name: "test files locate the dir without being copied",
			files: map[string]string{
				"corpus/schema.go":      corpusFile,
				"corpus/schema_test.go": corpusFile,
			},
			wantDir: "corpus",
			wantN:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, sha := gitRepo(t, tc.files)
			got, files, err := detectCorpus(dir, sha)
			if err != nil {
				t.Fatalf("detectCorpus: %v", err)
			}
			if got != tc.wantDir {
				t.Errorf("corpus dir = %q, want %q", got, tc.wantDir)
			}
			if len(files) != tc.wantN {
				t.Errorf("got %d corpus files %v, want %d", len(files), files, tc.wantN)
			}
			for _, f := range files {
				if strings.HasSuffix(f, "_test.go") {
					t.Errorf("melded a test file: %s", f)
				}
			}
		})
	}
}

// TestDetectCorpusRefusesRatherThanGuesses covers the two shapes meld cannot
// resolve. Both must name what they found: an error that says only "failed"
// leaves you unable to tell a non-store from an ambiguous one.
func TestDetectCorpusRefusesRatherThanGuesses(t *testing.T) {
	t.Run("no package winze anywhere", func(t *testing.T) {
		dir, sha := gitRepo(t, map[string]string{"main.go": "package main\n\nfunc main() {}\n"})
		_, _, err := detectCorpus(dir, sha)
		if err == nil {
			t.Fatal("want an error for a repo that is not a winze store")
		}
		if !strings.Contains(err.Error(), "not a winze store") {
			t.Errorf("error %q does not say the repo is not a store", err)
		}
	})

	t.Run("two candidate directories", func(t *testing.T) {
		dir, sha := gitRepo(t, map[string]string{
			"corpus/schema.go": corpusFile,
			"vendored/old.go":  corpusFile,
		})
		_, _, err := detectCorpus(dir, sha)
		if err == nil {
			t.Fatal("want an error rather than a pick between two corpora")
		}
		for _, want := range []string{"corpus", "vendored"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not name candidate %q", err, want)
			}
		}
	})
}
