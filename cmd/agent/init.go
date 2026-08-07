package main

// init — scaffold a new winze store.
//
// This existed only as folklore until 2026-08-07: "copy five files by hand,
// write a go.mod, git init." That is a poor first step for anything, and it
// made every proposal that begins with "create a store" impossible to estimate
// honestly, because its first step had no cost anyone could quote. Reported by
// the wanigan session as a blocker on exactly that.
//
// The scaffold copies the canonical schema from a winze checkout rather than
// embedding a copy in this binary. Embedding would put a second copy of the
// schema in the tree, and a second copy is the drift this project spends its
// build gate preventing. The cost is that init needs to find the source, which
// it does the same way everything else here finds things: an env var, then a
// sensible default, then a clear error.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// canonicalFiles are the schema files every store holds a copy of. The copy is
// deliberate: it makes a store a standalone Go package that compiles without
// importing winze, which is what makes `go build .` in the store directory a
// working consistency gate.
//
// bootstrap.go is deliberately NOT here. It is winze's own founding record, and
// copying it was how the existing memory store ended up carrying 25 entities
// about defn, Dolt and Cyc that have nothing to do with its owner. The types it
// used to declare now live in schema.go for exactly this reason.
var canonicalFiles = []string{"schema.go", "roles.go", "predicates.go", "external.go"}

// contentFile is where a store's memories live. The name is not cosmetic:
// handleRemember and handleUpdate pass `--to memory.go`, so a store whose
// content file is called anything else compiles fine and then rejects every
// write.
const contentFile = "memory.go"

// storeGitignore keeps derived state out of the store's history. Every entry is
// something a winze tool regenerates: the lock, the embedding cache, telemetry,
// the defn index.
const storeGitignore = `.winze.lock
.winze-embed/
.winze-usage.jsonl
.defn/
bin/
*.db
`

const privatePushHook = `#!/bin/sh
echo "pre-push BLOCKED: this winze store is local-only. Never push private working memory." >&2
exit 1
`

// initArgs is what `init` was asked to do, separated from doing it so the
// parsing is readable on its own and the orchestration below reads as a list of
// steps rather than a switch wrapped around one.
type initArgs struct {
	dir     string
	from    string
	private bool
	link    bool
}

func parseInitArgs(args []string) initArgs {
	var a initArgs
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--private":
			a.private = true
		case arg == "--link":
			a.link = true
		case arg == "--from" && i+1 < len(args):
			i++
			a.from = args[i]
		case strings.HasPrefix(arg, "--from="):
			a.from = strings.TrimPrefix(arg, "--from=")
		case strings.HasPrefix(arg, "-"):
			fatalf("init: unknown flag %q", arg)
		default:
			a.dir = arg
		}
	}
	if a.dir == "" {
		fmt.Fprintln(os.Stderr, "usage: winze-agent init <dir> [--private] [--link] [--from <winze-checkout>]")
		fmt.Fprintln(os.Stderr, "  --private  install a pre-push hook that blocks every push")
		fmt.Fprintln(os.Stderr, "  --link     point the current repo at the new store (git config winze.store)")
		os.Exit(2)
	}
	return a
}

func runInit(argv []string) {
	opts := parseInitArgs(argv)
	dir, from, private, link := opts.dir, opts.from, opts.private, opts.link

	abs, err := filepath.Abs(dir)
	if err != nil {
		fatalf("init: %v", err)
	}
	// Refuse rather than merge. A store half-created over an existing directory
	// is worse than no store: the build gate would pass on whatever was already
	// there and the missing half would surface much later.
	if entries, err := os.ReadDir(abs); err == nil && len(entries) > 0 {
		fatalf("init: %s already exists and is not empty", abs)
	}
	src, err := findCorpusSource(from)
	if err != nil {
		fatalf("init: %v", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		fatalf("init: %v", err)
	}

	name := moduleName(filepath.Base(abs))
	for _, f := range canonicalFiles {
		b, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			fatalf("init: reading %s from %s: %v", f, src, err)
		}
		if err := os.WriteFile(filepath.Join(abs, f), b, 0o644); err != nil {
			fatalf("init: %v", err)
		}
	}
	write := func(rel, content string, mode os.FileMode) {
		if err := os.WriteFile(filepath.Join(abs, rel), []byte(content), mode); err != nil {
			fatalf("init: writing %s: %v", rel, err)
		}
	}
	write("go.mod", fmt.Sprintf("module %s\n\ngo %s\n", name, goDirective(src)), 0o644)
	// memory.go, not <name>.go: winze_remember and winze_update write to that
	// exact filename. Naming it anything else produces a store that compiles
	// and then refuses its first memory — which is what the first version of
	// this did, and what the compile-only test failed to catch.
	write(contentFile, fmt.Sprintf(`package winze

// %s — a winze store. Every declaration here is knowledge, and `+"`go build .`"+`
// is the consistency checker: a claim referencing an entity that does not exist
// will not compile.
//
// Do not hand-edit to add a memory. Use winze_remember / winze_update /
// winze_link, or winze-add, so the gate and the dedup check run.
`, name), 0o644)
	write(".gitignore", storeGitignore, 0o644)

	// The gate before the commit, so a store that cannot compile is never
	// recorded as a working one.
	if out, err := runIn(abs, "go", "build", "."); err != nil {
		fatalf("init: the new store does not compile — the copied schema is broken:\n%s", out)
	}

	if out, err := runIn(abs, "git", "init", "-q", "-b", "main"); err != nil {
		fatalf("init: git init: %s", out)
	}
	if private {
		write(filepath.Join(".git", "hooks", "pre-push"), privatePushHook, 0o755)
	}
	if out, err := runIn(abs, "git", "add", "."); err != nil {
		fatalf("init: git add: %s", out)
	}
	msg := fmt.Sprintf("winze store %s: schema from %s", name, shortRev(src))
	if out, err := runIn(abs, "git", "commit", "-q", "-m", msg); err != nil {
		fatalf("init: git commit: %s", out)
	}

	fmt.Printf("store ready at %s\n", abs)
	fmt.Printf("  schema: %s (copied from %s)\n", strings.Join(canonicalFiles, ", "), src)
	fmt.Printf("  content: %s — empty, written through the tools\n", contentFile)
	if private {
		fmt.Println("  private: pre-push hook installed, every push blocked")
	}
	if link {
		if out, err := runIn(".", "git", "config", "winze.store", abs); err != nil {
			fmt.Fprintf(os.Stderr, "  (could not link this repo: %s)\n", out)
		} else {
			fmt.Println("  linked: this repo now resolves to it (git config winze.store)")
		}
		return
	}
	fmt.Printf("\nPoint a repo at it from that repo's root:\n  git config winze.store %s\n", abs)
}

// findCorpusSource locates the canonical schema: --from, then $WINZE_SRC, then
// the corpus beside this binary (the `make build` layout), then give up loudly
// rather than scaffold a store from files that are not there.
func findCorpusSource(from string) (string, error) {
	var tried []string
	consider := func(p string) (string, bool) {
		if p == "" {
			return "", false
		}
		c := p
		if filepath.Base(c) != "corpus" {
			c = filepath.Join(c, "corpus")
		}
		tried = append(tried, c)
		if _, err := os.Stat(filepath.Join(c, "schema.go")); err == nil {
			return c, true
		}
		return "", false
	}
	if p, ok := consider(from); ok {
		return p, nil
	}
	if from != "" {
		return "", fmt.Errorf("--from %s holds no corpus/schema.go", from)
	}
	if p, ok := consider(os.Getenv("WINZE_SRC")); ok {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		if p, ok := consider(filepath.Dir(filepath.Dir(exe))); ok {
			return p, nil
		}
	}
	return "", fmt.Errorf("no winze corpus found (looked in %s) — set $WINZE_SRC to a winze checkout or pass --from",
		strings.Join(tried, ", "))
}

// moduleName turns a directory name into a Go module path. Store directories
// are named for people and projects ("winze-memory", "publicai-memory"), and a
// hyphen is legal in a module path but the name also has to survive being a
// filename, so it is normalised once here and reused for both.
var nonModuleChar = regexp.MustCompile(`[^a-z0-9]+`)

func moduleName(base string) string {
	s := nonModuleChar.ReplaceAllString(strings.ToLower(base), "")
	if s == "" {
		return "winzestore"
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "s" + s
	}
	return s
}

// goDirective reads the `go` line from the source module so a new store targets
// the same toolchain as the schema it just copied, rather than a version pinned
// in this file that would rot.
func goDirective(corpusDir string) string {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(corpusDir), "go.mod"))
	if err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
				return strings.TrimSpace(v)
			}
		}
	}
	return "1.24"
}

// shortRev records which winze revision the schema was copied from, so a store
// that later fails its gate after a sync can be traced to the version it
// started from.
func shortRev(corpusDir string) string {
	out, err := runIn(filepath.Dir(corpusDir), "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "unknown revision"
	}
	return strings.TrimSpace(out)
}

func runIn(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
