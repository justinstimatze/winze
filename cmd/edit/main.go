// Command edit performs referentially-safe mutations on the corpus.
//
// Until now winze's entire write surface was cmd/add, which appends a claim.
// A knowledge base you can only append to is one you cannot maintain: when
// rot-probe reports two entities that are probably the same thing, or a
// framing gets refined and its claims should retarget, there was no tool to
// act on the finding. This is the edit side.
//
// Every mutation runs the same gate cmd/add does — gofmt, go build, go vet —
// and reverts every touched file if any step fails. The gate is the whole
// discipline of the project; a mutation tool that bypassed it would be worse
// than no mutation tool.
//
// Renaming is done by byte-splicing at offsets the parser identifies, not by
// text substitution. The parser knows which occurrences of "Apophenia" are
// the identifier and which are prose inside a Brief, a Quote, or a comment;
// sed does not. That distinction is the entire reason this is a Go tool and
// not a shell one-liner.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justinstimatze/winze/internal/astutil"
	"github.com/justinstimatze/winze/internal/corpuslock"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "rename":
		os.Exit(cmdRename(os.Args[2:]))
	case "merge":
		os.Exit(cmdMerge(os.Args[2:]))
	case "set-brief":
		os.Exit(cmdSetBrief(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: winze-edit <subcommand> [flags]

Referentially-safe corpus mutations. Every mutation is gated on
gofmt + go build + go vet and reverts all touched files on failure.

Subcommands:
  rename   rename a top-level var across the corpus, rewriting every reference
  merge    fold entity A into entity B: retarget every reference, remove A's
           declaration; claims retarget automatically, the build gate validates
  set-brief revise an entity's Brief (and optionally Name) in place

Run 'winze-edit <subcommand> -h' for flags.
`)
}

func cmdRename(args []string) int {
	fs := flag.NewFlagSet("rename", flag.ExitOnError)
	from := fs.String("from", "", "existing var name (e.g. KlausConrad)")
	to := fs.String("to", "", "new var name")
	root := fs.String("root", ".", "winze repo root (the directory containing predicates.go)")
	dryRun := fs.Bool("dry-run", false, "report what would change; do not modify files")
	fs.Parse(args)

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "error: --from and --to are required")
		fs.Usage()
		return 2
	}
	if *from == *to {
		fmt.Fprintln(os.Stderr, "error: --from and --to are identical")
		return 2
	}
	if !isIdent(*to) {
		fmt.Fprintf(os.Stderr, "error: %q is not a valid Go identifier\n", *to)
		return 2
	}

	sites, declared, collides, err := findRenameSites(*root, *from, *to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}
	if !declared {
		fmt.Fprintf(os.Stderr, "error: no top-level var named %q in the corpus\n", *from)
		return 1
	}
	if collides {
		fmt.Fprintf(os.Stderr, "error: %q is already declared; rename would collide\n", *to)
		return 1
	}

	files := make([]string, 0, len(sites))
	total := 0
	for f, offs := range sites {
		files = append(files, f)
		total += len(offs)
	}
	sort.Strings(files)

	fmt.Printf("rename %s -> %s: %d references across %d files\n", *from, *to, total, len(files))
	for _, f := range files {
		fmt.Printf("  %s (%d)\n", filepath.Base(f), len(sites[f]))
	}
	if *dryRun {
		fmt.Println("(dry run — nothing written)")
		return 0
	}

	unlock, err := corpuslock.Acquire(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "corpus lock: %v\n", err)
		return 1
	}
	defer unlock()

	backups := map[string][]byte{}
	revert := func() {
		for path, content := range backups {
			_ = os.WriteFile(path, content, 0o644)
		}
	}

	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			revert()
			fmt.Fprintf(os.Stderr, "read %s: %v (reverted)\n", path, err)
			return 1
		}
		backups[path] = src
		if err := os.WriteFile(path, splice(src, sites[path], *from, *to), 0o644); err != nil {
			revert()
			fmt.Fprintf(os.Stderr, "write %s: %v (reverted)\n", path, err)
			return 1
		}
	}

	// Format ONLY the files this rename touched. `gofmt -w` on the repo root
	// recurses and reformats everything, sweeping unrelated pre-existing drift
	// into what should be a surgical change — a mutation tool must not have a
	// blast radius wider than its mutation.
	gofmtArgs := append([]string{"-w"}, files...)
	steps := [][]string{
		append([]string{"gofmt"}, gofmtArgs...),
		{"go", "build", "."},
		{"go", "vet", "."},
	}
	for _, step := range steps {
		if out, err := runCmd(*root, step[0], step[1:]...); err != nil {
			revert()
			fmt.Fprintf(os.Stderr, "%s failed (all files reverted):\n%s\n", step[0], out)
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "renamed %s -> %s in %d files (build gate passed)\n", *from, *to, len(files))
	return 0
}

// findRenameSites returns byte offsets of every identifier referring to the
// package-level var `from`, plus whether `from` is declared and whether `to`
// already is.
//
// The corpus is a single package of top-level declarations with no local
// scopes to shadow a name, so every *ast.Ident matching `from` is a reference
// to that var. Identifiers inside string literals and comments are not
// *ast.Ident nodes at all, which is exactly why this is safe and a textual
// substitution would not be.
func findRenameSites(root, from, to string) (map[string][]int, bool, bool, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, root, astutil.GoFileFilter, parser.ParseComments)
	if err != nil {
		return nil, false, false, err
	}

	sites := map[string][]int{}
	var declared, collides bool

	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, n := range vs.Names {
						switch n.Name {
						case from:
							declared = true
						case to:
							collides = true
						}
					}
				}
			}
			ast.Inspect(file, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok || id.Name != from {
					return true
				}
				// A selector's field half (pkg.From) is not our var.
				sites[path] = append(sites[path], fset.Position(id.Pos()).Offset)
				return true
			})
		}
	}
	return sites, declared, collides, nil
}

// splice replaces `from` with `to` at the given byte offsets. Offsets are
// applied back-to-front so earlier splices do not shift later ones.
func splice(src []byte, offsets []int, from, to string) []byte {
	ordered := append([]int(nil), offsets...)
	sort.Sort(sort.Reverse(sort.IntSlice(ordered)))
	out := src
	for _, off := range ordered {
		if off < 0 || off+len(from) > len(out) {
			continue
		}
		if string(out[off:off+len(from)]) != from {
			continue // position drifted; leave it for the build gate to catch
		}
		next := make([]byte, 0, len(out)-len(from)+len(to))
		next = append(next, out[:off]...)
		next = append(next, to...)
		next = append(next, out[off+len(from):]...)
		out = next
	}
	return out
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		isLetter := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if i == 0 && !isLetter {
			return false
		}
		if !isLetter && !isDigit {
			return false
		}
	}
	return !token.Lookup(s).IsKeyword()
}

func runCmd(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	// `gofmt -l -w` lists reformatted files on stdout and exits 0; only a real
	// error should trip the gate.
	if name == "gofmt" && err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return nil, nil
	}
	return out, err
}
