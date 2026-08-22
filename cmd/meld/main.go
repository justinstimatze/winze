// Command winze-meld bridges two or more winze stores into a single
// read-only "mind-meld" directory you can point the query tools at, then
// dissolve when done.
//
//	winze-meld <store1> <store2> [store3...]   # create a meld, print its path
//	winze-meld <store>@<sha> <store2>          # pin a store at a specific commit
//	winze-meld --out DIR <store1> <store2>     # meld into a chosen dir
//	winze-meld --dissolve DIR                  # tear a meld down (rm)
//
// The meld is a FROZEN snapshot: each store is materialized via
// `git archive` at a pinned SHA (HEAD by default), so the meld never
// couples to a store's live working tree and can be reproduced from the
// manifest. It is read-only by construction — the union of two
// `package winze` stores cannot `go build` (duplicate identifiers), so the
// write path (winze-add / winze-edit), which runs that build as its gate, does
// not apply.
//
// Nothing reads a meld today either. cmd/query reaches a corpus through
// defndb.New in every mode, and defn's ingest wants a module that type-checks,
// which a meld deliberately is not. See docs/meld.md; this doc used to claim
// winze-query AST-scrapes without type-checking, which was true before the defn
// migration and is not now.
//
// A store's corpus is found, not assumed: it is the one directory in the tree
// at that SHA whose .go files declare `package winze` (see detectCorpus). This
// repo keeps its corpus under corpus/; a store scaffolded by `winze-agent init`
// keeps the same files flat at its root. Both are correct, and the meld dir
// flattens either.
//
// Those corpus files are copied in namespace-prefixed (`<ns>__memory.go`); the
// prefix survives into query results as the source label, so a hit tells you
// which store it came from. One canonical predicates.go is kept from the
// primary (first) store so cmd/predicates-suggest still resolves. Cross-store
// var-name collisions are surfaced, not merged — namespacing is deferred by
// design; both entities appear, each tagged by store.
package main

import (
	"archive/tar"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justinstimatze/winze/internal/events"
)

// manifestName marks a directory as a winze-meld and records what it was
// built from. Its presence is the guard that --dissolve checks before rm.
const manifestName = ".winze-meld.json"

type manifest struct {
	Version int          `json:"version"`
	Stores  []storeEntry `json:"stores"`
	Primary string       `json:"primary"` // namespace whose predicates.go is canonical
}

type storeEntry struct {
	Path      string `json:"path"`
	SHA       string `json:"sha"`
	Namespace string `json:"namespace"`
	// CorpusDir is where the corpus was found in this store's tree at SHA:
	// "corpus" in a winze checkout, "." in a store `winze-agent init`
	// scaffolded flat. Recorded because the pinned SHA alone no longer says
	// where to look.
	CorpusDir string `json:"corpus_dir"`

	// files are the corpus paths detectCorpus already found, carried to
	// copyStoreGoFiles so the detection grep runs once per store rather than
	// twice. Kept out of the manifest deliberately: SHA plus CorpusDir
	// reproduces it, and a stored file list would only rot.
	files []string
}

func main() {
	var (
		out      = flag.String("out", "", "meld into this dir (default: a fresh mktemp dir)")
		dissolve = flag.String("dissolve", "", "tear down the meld at this dir (validates the manifest, then rm)")
		quiet    = flag.Bool("quiet", false, "print only the meld dir path")
	)
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: winze-meld <store1> <store2> [store@sha ...]\n"+
			"       winze-meld --dissolve <dir>\n\n"+
			"Bridge winze stores into a read-only meld dir; query with winze-query <dir>.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *dissolve != "" {
		if err := runDissolve(*dissolve); err != nil {
			fatal(err)
		}
		return
	}

	stores := flag.Args()
	if len(stores) < 2 {
		flag.Usage()
		os.Exit(2)
	}
	if err := runMeld(stores, *out, *quiet); err != nil {
		fatal(err)
	}
}

func runMeld(specs []string, out string, quiet bool) error {
	entries, err := resolveStores(specs)
	if err != nil {
		return err
	}

	dir := out
	if dir == "" {
		dir, err = os.MkdirTemp("", "winze-meld-*")
		if err != nil {
			return err
		}
	} else {
		if empty, err := isEmptyOrAbsent(dir); err != nil {
			return err
		} else if !empty {
			return fmt.Errorf("--out %s exists and is not empty; refusing to overwrite", dir)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	for _, e := range entries {
		n, err := copyStoreGoFiles(e, dir)
		if err != nil {
			return fmt.Errorf("meld %s@%s: %w", e.Path, e.SHA[:min(7, len(e.SHA))], err)
		}
		if n == 0 {
			return fmt.Errorf("meld %s@%s: corpus %s archived no files", e.Path, e.SHA[:min(7, len(e.SHA))], e.CorpusDir)
		}
	}

	// The primary store's predicates.go is copied canonically (un-prefixed)
	// so LoadPredicates (cmd/predicates-suggest) still resolves against the meld.
	if err := copyCanonicalPredicates(entries[0], dir); err != nil {
		return err
	}

	m := manifest{Version: 1, Stores: entries, Primary: entries[0].Namespace}
	if err := writeManifest(dir, m); err != nil {
		return err
	}
	emitMeldEvent("meld", entries[0].Path, entries, dir)

	if quiet {
		fmt.Println(dir)
		return nil
	}
	fmt.Printf("melded %d stores into %s\n", len(entries), dir)
	for _, e := range entries {
		fmt.Printf("  %-16s %s @ %s  corpus=%s\n", e.Namespace, e.Path, e.SHA[:min(12, len(e.SHA))], e.CorpusDir)
	}
	fmt.Printf("\nquery it:   winze-query --hybrid \"<q>\" %s\n", dir)
	fmt.Printf("dissolve:   winze-meld --dissolve %s\n", dir)
	return nil
}

// resolveStores parses `path[@ref]` specs into pinned entries with unique
// filesystem-safe namespaces derived from each store's directory name, and
// locates each store's corpus at the SHA it pinned.
func resolveStores(specs []string) ([]storeEntry, error) {
	seen := map[string]int{}
	var entries []storeEntry
	for _, spec := range specs {
		path, ref := spec, "HEAD"
		if i := strings.LastIndex(spec, "@"); i >= 0 {
			path, ref = spec[:i], spec[i+1:]
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
			return nil, fmt.Errorf("store %s: not a directory", path)
		}
		sha, err := gitRevParse(abs, ref)
		if err != nil {
			return nil, fmt.Errorf("store %s: resolve %q: %w", path, ref, err)
		}
		corpusDir, files, err := detectCorpus(abs, sha)
		if err != nil {
			return nil, fmt.Errorf("store %s: %w", path, err)
		}
		ns := uniqueNamespace(sanitizeNS(filepath.Base(abs)), seen)
		entries = append(entries, storeEntry{Path: abs, SHA: sha, Namespace: ns, CorpusDir: corpusDir, files: files})
	}
	return entries, nil
}

var nsBad = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeNS(base string) string {
	ns := nsBad.ReplaceAllString(strings.ToLower(base), "-")
	ns = strings.Trim(ns, "-")
	if ns == "" {
		ns = "store"
	}
	return ns
}

func uniqueNamespace(ns string, seen map[string]int) string {
	if n := seen[ns]; n > 0 {
		seen[ns] = n + 1
		return fmt.Sprintf("%s-%d", ns, n+1)
	}
	seen[ns] = 1
	return ns
}

func gitRevParse(dir, ref string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", ref)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// copyStoreGoFiles archives e's detected corpus files at e.SHA and writes each
// into dir as `<ns>__<base>.go`, flattening whatever directory the corpus lived
// in. Returns the count written.
//
// The archive names those files explicitly rather than naming a directory, so a
// store whose corpus sits at its repo root does not drag its whole tree through
// the tar — and nothing needs filtering on the way out, since nothing was asked
// for that should not be copied.
func copyStoreGoFiles(e storeEntry, dir string) (int, error) {
	args := append([]string{"-C", e.Path, "archive", "--format=tar", e.SHA, "--"}, e.files...)
	cmd := exec.Command("git", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	count := 0
	tr := tar.NewReader(stdout)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			cmd.Wait()
			return count, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue // git archive emits the intermediate directory entries too
		}
		dst := filepath.Join(dir, e.Namespace+"__"+filepath.Base(hdr.Name))
		if err := writeFileFrom(tr, dst); err != nil {
			cmd.Wait()
			return count, err
		}
		count++
	}
	if err := cmd.Wait(); err != nil {
		return count, fmt.Errorf("git archive: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return count, nil
}

// copyCanonicalPredicates copies the primary store's predicates.go into dir
// un-prefixed, so LoadPredicates(dir) resolves. Absent predicates.go is fine.
func copyCanonicalPredicates(e storeEntry, dir string) error {
	cmd := exec.Command("git", "-C", e.Path, "show", e.SHA+":"+filepath.Join(e.CorpusDir, "predicates.go"))
	out, err := cmd.Output()
	if err != nil {
		return nil // no predicates.go at that commit — not fatal
	}
	return os.WriteFile(filepath.Join(dir, "predicates.go"), out, 0o644)
}

func writeFileFrom(r io.Reader, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	return f.Close()
}

func runDissolve(dir string) error {
	mf := filepath.Join(dir, manifestName)
	data, err := os.ReadFile(mf)
	if err != nil {
		return fmt.Errorf("%s is not a winze-meld dir (no %s): %w", dir, manifestName, err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("%s has a corrupt %s: %w", dir, manifestName, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	primary := dir
	if len(m.Stores) > 0 {
		primary = m.Stores[0].Path
	}
	emitMeldEvent("unmeld", primary, m.Stores, dir)
	fmt.Printf("dissolved meld at %s (%d stores)\n", dir, len(m.Stores))
	return nil
}

// emitMeldEvent records a meld/unmeld to the fleet event stream so a dashboard
// can render two winzes bridging (and then dissolving) from real events. The
// store basenames are what a consumer matches against its instance tiles.
func emitMeldEvent(kind, primary string, stores []storeEntry, meldDir string) {
	names := make([]string, len(stores))
	for i, s := range stores {
		names[i] = filepath.Base(s.Path)
	}
	events.Emit(primary, kind, map[string]any{"stores": names, "dir": meldDir})
}

func writeManifest(dir string, m manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, manifestName), append(data, '\n'), 0o644)
}

func isEmptyOrAbsent(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "winze-meld:", err)
	os.Exit(1)
}

// detectCorpus locates the corpus in store at sha: the one directory whose .go
// files declare `package winze`, plus the paths of those files.
//
// It looks rather than assuming because there is no single right layout. This
// repo keeps its corpus under corpus/ so defn ingest stops indexing cmd/ and
// internal/ on every write; a store scaffolded by `winze-agent init` keeps the
// same files flat at its root, which is what makes `go build .` there a working
// gate. Nesting a flat store would cost it that gate and buy it nothing.
//
// Looking is also the only thing that works at an arbitrary pinned SHA. A
// layout marker committed today does not exist at last month's commit, and a
// meld's promise is that its manifest reproduces it.
//
// The package clause is the marker for two reasons: Go allows exactly one per
// directory, so it cannot drift within the corpus; and two `package winze` sets
// colliding is precisely what makes a meld read-only. The same grep hands back
// the file list, because every .go file in a directory declares that
// directory's package — whatever named the directory has already named its
// contents.
func detectCorpus(store, sha string) (string, []string, error) {
	short := sha[:min(7, len(sha))]
	cmd := exec.Command("git", "-C", store, "grep", "-l", "^package winze$", sha, "--", "*.go")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		// git grep exits 1 for "matched nothing", which is a fact about the
		// store rather than a git failure. Anything else is git going wrong.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", nil, fmt.Errorf("no `package winze` directory at %s — not a winze store", short)
		}
		return "", nil, fmt.Errorf("git grep: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	dirs, files := corpusMatches(out, sha)
	switch {
	case len(dirs) == 0:
		return "", nil, fmt.Errorf("no `package winze` directory at %s — not a winze store", short)
	case len(dirs) > 1:
		return "", nil, fmt.Errorf("ambiguous corpus at %s: `package winze` in %s — meld will not pick for you",
			short, strings.Join(dirs, ", "))
	case len(files) == 0:
		return "", nil, fmt.Errorf("corpus %s at %s holds only _test.go files", dirs[0], short)
	}
	return dirs[0], files, nil
}

// corpusMatches splits `git grep -l` output into the directories that declared
// `package winze` and the corpus files to actually meld.
//
// A directory is claimed by any match in it, _test.go included — a test file
// declaring `package winze` sits in the corpus by Go's own rule. The file list
// drops those: a meld carries corpus content, not the store's own gate.
func corpusMatches(out []byte, sha string) (dirs, files []string) {
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		path, ok := strings.CutPrefix(line, sha+":")
		if !ok {
			continue
		}
		if d := filepath.Dir(path); !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
		if !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
	}
	return dirs, files
}
