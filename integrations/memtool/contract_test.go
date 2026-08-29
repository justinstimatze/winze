package memtool

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd() // integrations/memtool
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

// gitEnv names an author for the subprocess git commits that a real
// winze-agent init/remember/update makes along the way.
var gitEnv = []string{
	"GIT_AUTHOR_NAME=winze test",
	"GIT_AUTHOR_EMAIL=test@example.com",
	"GIT_COMMITTER_NAME=winze test",
	"GIT_COMMITTER_EMAIL=test@example.com",
}

// TestCLIBackendCreateAndRecallAgainstARealStore is the contract test this
// package didn't have: every other test here drives Executor against
// fakeBackend, which encodes the winze-agent CLI's output shape as an
// assumption, never a verified fact. createdVar (a line-scraping parser
// duplicated from cmd/agent's own copy, because cmd/agent's handlers are
// unexported in package main and this package cannot import them) and
// fullBrief's assumption about winze_recall's JSON shape both drift silently
// if the real CLI's output ever changes. This builds a real winze-agent and
// its sibling tools, scaffolds a real scratch store, and drives cliBackend
// against them for real, so that drift shows up as a failing test here, not
// as a silent break the first time this package is actually used.
//
// Uses a path-shaped title ("/memories/...") rather than a friendly one,
// matching how Executor.create actually calls remember: with the memory
// tool's virtual path as the title.
//
// The sleep-then-touch before recalling is not padding: it works around a
// real, root-caused bug in the pinned defn library
// (github.com/justinstimatze/defn v0.26.31, db.(*DB).StaleFiles,
// db/db.go:364) that this test surfaced. StaleFiles compares file mtimes to
// the last-ingest time using Unix-second resolution with a strict `>`
// (`info.ModTime().Unix() > lastIngest`), so a write landing in the same
// wall-clock second as the ingest that recorded last_ingest is invisible to
// the staleness check forever after -- winze_recall then serves a pre-write
// view with zero explanation, and no amount of waiting before the *next*
// query fixes it, because memory.go's mtime and the recorded last_ingest are
// both already-fixed past timestamps that a later delay cannot change. The
// only fix from outside defn is to force memory.go's mtime to a point that
// is unambiguously past last_ingest: sleep past the current second, then
// os.Chtimes it to "now". Remove this once the upstream comparison uses
// sub-second resolution or a content hash instead of Unix()-truncated mtimes.
func TestCLIBackendCreateAndRecallAgainstARealStore(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	root := repoRoot(t)
	tools := t.TempDir()
	agentBin := filepath.Join(tools, "winze-agent")
	if out, err := exec.Command("go", "build", "-o", agentBin, filepath.Join(root, "cmd", "agent")).CombinedOutput(); err != nil {
		t.Fatalf("building winze-agent: %v\n%s", err, out)
	}
	for _, sub := range []string{"add", "edit", "query"} {
		out, err := exec.Command("go", "build", "-o", filepath.Join(tools, "winze-"+sub), filepath.Join(root, "cmd", sub)).CombinedOutput()
		if err != nil {
			t.Fatalf("building winze-%s: %v\n%s", sub, err, out)
		}
	}

	storeDir := filepath.Join(t.TempDir(), "test-store")
	initCmd := exec.Command(agentBin, "init", storeDir, "--from", root)
	initCmd.Env = append(os.Environ(), gitEnv...)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	t.Setenv("WINZE_STORE", storeDir)
	t.Setenv("WINZE_BIN", tools)
	for _, kv := range gitEnv {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}

	be := cliBackend{BinPath: agentBin}
	const path = "/memories/contract-test.md"

	varName, err := be.remember("A contract test wrote this through the real CLI.", path)
	if err != nil {
		t.Fatalf("remember: %v", err)
	}
	if varName == "" {
		t.Fatal("remember returned an empty var name -- createdVar's parsing likely drifted from winze-agent's real success-line format")
	}

	time.Sleep(1100 * time.Millisecond) // let wall-clock time move past the write's second
	memPath := filepath.Join(storeDir, "memory.go")
	now := time.Now()
	if err := os.Chtimes(memPath, now, now); err != nil {
		t.Fatalf("touching memory.go to defeat defn's same-second staleness bug: %v", err)
	}

	brief, found, err := be.fullBrief(path)
	if err != nil {
		t.Fatalf("fullBrief: %v", err)
	}
	if !found {
		t.Fatal("fullBrief did not find the entity it just created -- winze_recall's real JSON shape likely drifted from what fullBrief expects")
	}
	if !strings.Contains(brief, "A contract test wrote this through the real CLI.") {
		t.Errorf("fullBrief returned %q, want it to contain the note we wrote", brief)
	}
}
