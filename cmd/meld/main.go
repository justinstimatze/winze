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
// write path (winze-add / winze-edit) does not apply. Only the read path
// (winze-query), which AST-scrapes composite literals without type-checking,
// operates over a meld.
//
// Every store's top-level *.go files are copied in namespace-prefixed
// (`<ns>__memory.go`); the prefix survives into query results as the source
// label, so a hit tells you which store it came from. One canonical
// predicates.go is kept from the primary (first) store so cmd/predicates-suggest
// still resolves. Cross-store var-name collisions are surfaced, not merged —
// namespacing is deferred by design; both entities appear, each tagged by store.
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
			return fmt.Errorf("meld %s@%s: no top-level .go files at that commit", e.Path, e.SHA[:min(7, len(e.SHA))])
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
		fmt.Printf("  %-16s %s @ %s\n", e.Namespace, e.Path, e.SHA[:min(12, len(e.SHA))])
	}
	fmt.Printf("\nquery it:   winze-query --hybrid \"<q>\" %s\n", dir)
	fmt.Printf("dissolve:   winze-meld --dissolve %s\n", dir)
	return nil
}

// resolveStores parses `path[@ref]` specs into pinned entries with unique
// filesystem-safe namespaces derived from each store's directory name.
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
		ns := uniqueNamespace(sanitizeNS(filepath.Base(abs)), seen)
		entries = append(entries, storeEntry{Path: abs, SHA: sha, Namespace: ns})
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

// copyStoreGoFiles streams `git archive <sha>` and writes each top-level
// non-test .go file into dir as `<ns>__<name>`. Returns the count written.
func copyStoreGoFiles(e storeEntry, dir string) (int, error) {
	cmd := exec.Command("git", "-C", e.Path, "archive", "--format=tar", e.SHA)
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
		name := hdr.Name
		if strings.Contains(name, "/") { // top-level only
			continue
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		dst := filepath.Join(dir, e.Namespace+"__"+name)
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
	cmd := exec.Command("git", "-C", e.Path, "show", e.SHA+":predicates.go")
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
