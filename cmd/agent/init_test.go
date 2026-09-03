package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The valuable test here is the end-to-end one: a scaffolded store must
// actually compile. Everything else in init is arrangement; "go build ." in a
// fresh store passing is the only thing that says the scaffold works, and it
// is exactly what nobody could check while the procedure was folklore.

// gitIdentity names an author for a subprocess that will commit. init creates
// the store's repo itself, so the `git config user.email` the other tests here
// run against an existing repo has nothing to configure yet — and a machine
// with no global identity fails inside git rather than in anything these tests
// are about. That is what held CI red from 2026-08-09 while every developer
// box, each carrying a ~/.gitconfig, stayed green. Reproduce the CI shape with
// GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null go test ./cmd/agent/.
var gitIdentity = []string{
	"GIT_AUTHOR_NAME=winze test",
	"GIT_AUTHOR_EMAIL=test@example.com",
	"GIT_COMMITTER_NAME=winze test",
	"GIT_COMMITTER_EMAIL=test@example.com",
}

func winzeRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd() // cmd/agent
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

func TestInitProducesAStoreThatCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := filepath.Join(t.TempDir(), "test-store")

	// runInit exits the process on failure, so drive the built binary rather
	// than calling it in-process — a t.Fatal is more useful than os.Exit(1)
	// taking the whole test binary with it.
	// Both binaries go in one dir. winze_remember below shells out to winze-add
	// through WINZE_BIN, and pointing that at the repo's own bin/ only works on
	// a box that has run `make build` — bin/ is gitignored, so on a clean
	// checkout the store's first write dies at fork/exec instead of testing
	// anything.
	tools := t.TempDir()
	bin := filepath.Join(tools, "winze-agent")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building winze-agent: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "build", "-o", filepath.Join(tools, "winze-add"), "../add").CombinedOutput(); err != nil {
		t.Fatalf("building winze-add: %v\n%s", err, out)
	}
	cmd := exec.Command(bin, "init", dir, "--private", "--from", winzeRoot(t))
	cmd.Env = append(os.Environ(), gitIdentity...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	for _, f := range append(canonicalFiles, "go.mod", ".gitignore", contentFile) {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("scaffold is missing %s", f)
		}
	}
	// bootstrap.go is winze's own founding record. Copying it is how the
	// existing memory store ended up holding 25 entities about defn and Cyc.
	if _, err := os.Stat(filepath.Join(dir, "bootstrap.go")); err == nil {
		t.Error("scaffold copied bootstrap.go — a new store inherits winze's founding entities")
	}

	// The gate. init runs this itself before committing; running it again here
	// is what makes this test worth having.
	build := exec.Command("go", "build", ".")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("the scaffolded store does not compile:\n%s", out)
	}
	vet := exec.Command("go", "vet", ".")
	vet.Dir = dir
	if out, err := vet.CombinedOutput(); err != nil {
		t.Fatalf("the scaffolded store does not vet:\n%s", out)
	}

	// A store that compiles but cannot be written to is not a store. The first
	// version of init passed every other assertion here and still produced one:
	// it named the content file after the module, while winze_remember writes
	// to memory.go, so the scaffold rejected its first memory. Compiling is not
	// the same as working, and only a real write says which one this is.
	remember := exec.Command(bin, "call", "winze_remember",
		`{"note":"A scaffolded store accepts its first memory through the build gate.","title":"Scaffold write check"}`)
	remember.Env = append(os.Environ(), "WINZE_STORE="+dir, "WINZE_BIN="+tools)
	remember.Env = append(remember.Env, gitIdentity...) // the write path commits too
	if out, err := remember.CombinedOutput(); err != nil || strings.Contains(string(out), "failed") {
		t.Fatalf("a fresh store rejected its first memory: %v\n%s", err, out)
	}
	if b, err := os.ReadFile(filepath.Join(dir, contentFile)); err != nil || !strings.Contains(string(b), "Scaffold write check") {
		t.Errorf("the memory did not land in %s: %v", contentFile, err)
	}

	// --private must actually block, not merely exist.
	hook := filepath.Join(dir, ".git", "hooks", "pre-push")
	fi, err := os.Stat(hook)
	if err != nil {
		t.Fatalf("--private installed no pre-push hook: %v", err)
	}
	if fi.Mode()&0o111 == 0 {
		t.Error("pre-push hook is not executable — it would never run")
	}
	if h := exec.Command(hook); h.Run() == nil {
		t.Error("pre-push hook exited 0 — it must block the push")
	}

	// One commit, and it should record where the schema came from.
	log := exec.Command("git", "log", "--oneline")
	log.Dir = dir
	lb, err := log.CombinedOutput()
	if err != nil || !strings.Contains(string(lb), "winze store") {
		t.Errorf("git history is not what init claims to have written: %v\n%s", err, lb)
	}
}

func TestInitRefusesANonEmptyDirectory(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "winze-agent")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, "init", dir, "--from", winzeRoot(t)).CombinedOutput()
	if err == nil {
		t.Fatal("init overwrote a non-empty directory")
	}
	if !strings.Contains(string(out), "not empty") {
		t.Errorf("refusal does not say why: %s", out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "notes.md")); string(b) != "mine" {
		t.Error("init touched an existing file before refusing")
	}
}

func TestModuleNameSurvivesRealStoreNames(t *testing.T) {
	for in, want := range map[string]string{
		"winze-memory":    "winzememory",
		"publicai-memory": "publicaimemory",
		"My Store":        "mystore",
		"2026-notes":      "s2026notes", // a module path may not start with a digit
		"---":             "winzestore",
	} {
		if got := moduleName(in); got != want {
			t.Errorf("moduleName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFindCorpusSourceReportsWhereItLooked(t *testing.T) {
	t.Setenv("WINZE_SRC", "")
	_, err := findCorpusSource(filepath.Join(t.TempDir(), "nowhere"))
	if err == nil {
		t.Fatal("found a corpus that does not exist")
	}
	// A path in the error is the difference between a one-line fix and a hunt —
	// the same lesson as the recall error message.
	if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("error names no path: %v", err)
	}
}

// TestFindCorpusSourceFallsBackToModuleCache proves the fallback FEEDBACK-2026-09-02.md#6
// asked for: a repo that pulled winze in as a dependency (or `go get`) has a
// real corpus/schema.go sitting in the module cache even when it never
// cloned the source, so `init` should find it there rather than giving up.
func TestFindCorpusSourceFallsBackToModuleCache(t *testing.T) {
	t.Setenv("WINZE_SRC", "")
	cacheDir := t.TempDir()
	pkgDir := filepath.Join(cacheDir, "github.com", "justinstimatze", "winze@v0.0.0-fake")
	if err := os.MkdirAll(filepath.Join(pkgDir, "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "corpus", "schema.go"), []byte("package winze\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMODCACHE", cacheDir)
	// Outside any module, so the earlier `go list -m` fallback cannot resolve
	// winze and this test is actually exercising the module-cache glob, not
	// riding along on the main-module case (this checkout's own go.mod IS
	// github.com/justinstimatze/winze, so a from-here `go list -m` on it would
	// otherwise succeed for the wrong reason).
	t.Chdir(t.TempDir())

	got, err := findCorpusSource("")
	if err != nil {
		t.Fatalf("findCorpusSource: %v", err)
	}
	if want := filepath.Join(pkgDir, "corpus"); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestInitWarnsWhenSourceIsNewerThanTheBinary is FEEDBACK-2026-09-02.md#7:
// a compile error from a schema fix the running binary predates read as a
// bug, not a stale install, and cost a round-trip to diagnose. init already
// knows both paths at this point, so it should say so.
func TestInitWarnsWhenSourceIsNewerThanTheBinary(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// A throwaway checkout with the real schema files, committed at a
	// deliberately future date — the shape of a `git pull` landing after this
	// binary was last built.
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range canonicalFiles {
		b, err := os.ReadFile(filepath.Join(winzeRoot(t), "corpus", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, "corpus", f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(env []string, name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = src
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	base := append(os.Environ(), gitIdentity...)
	run(base, "git", "init", "-q")
	run(base, "git", "add", ".")
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	run(append(append([]string{}, base...), "GIT_AUTHOR_DATE="+future, "GIT_COMMITTER_DATE="+future),
		"git", "commit", "-q", "-m", "future schema change")

	tools := t.TempDir()
	bin := filepath.Join(tools, "winze-agent")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building winze-agent: %v\n%s", err, out)
	}

	dir := filepath.Join(t.TempDir(), "test-store")
	cmd := exec.Command(bin, "init", dir, "--from", src)
	cmd.Env = base
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "has commits newer than this binary") {
		t.Errorf("no stale-binary advisory in init output:\n%s", out)
	}
}

// TestInitLinkPrintsMCPRegistrationCommand is FEEDBACK-2026-09-02.md#2:
// --link makes a store resolvable but the calling repo still has no
// winze_remember tool until an MCP server is registered for it, and nothing
// said so. init should print the registration line rather than run it —
// adding an MCP server is a standing config change this command should not
// make on a caller's behalf.
func TestInitLinkPrintsMCPRegistrationCommand(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	tools := t.TempDir()
	bin := filepath.Join(tools, "winze-agent")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building winze-agent: %v\n%s", err, out)
	}

	// --link runs `git config winze.store` in the caller's cwd — give it a
	// disposable repo rather than letting it touch this checkout's own config.
	callerRepo := t.TempDir()
	initRepo := exec.Command("git", "init", "-q")
	initRepo.Dir = callerRepo
	if out, err := initRepo.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	dir := filepath.Join(t.TempDir(), "test-store")
	cmd := exec.Command(bin, "init", dir, "--link", "--from", winzeRoot(t))
	cmd.Dir = callerRepo
	cmd.Env = append(os.Environ(), gitIdentity...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "claude mcp add") {
		t.Errorf("--link did not print an MCP registration command:\n%s", out)
	}
	if !strings.Contains(string(out), bin) {
		t.Errorf("registration command does not name this binary (%s):\n%s", bin, out)
	}
}
