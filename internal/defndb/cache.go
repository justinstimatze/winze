package defndb

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The persistent index cache.
//
// Every winze CLI invocation is a fresh process, so the sync.Once memoisation
// on Client only ever helped within one run. A query against a 250k-line corpus
// paid the full parse every time — measured at ~236ms after parallelising the
// parser, and ~360ms before. That is fine for a batch phase and far too slow to
// feel interactive during content generation, which is the workload a game-dev
// vault actually creates.
//
// The cache is per-FILE, not per-corpus, because that is what makes the common
// case cheap. Editing one corpus slice — which is what `winze-add` and a hand
// edit both do — invalidates one fragment and reparses one file; everything
// else is decoded. A whole-corpus cache would be correct but would throw away
// all of it on every write.
//
// Invalidation is by (size, mtime), the same signal `go build` itself trusts.
// Content hashing would be stricter and would mean reading every file to decide
// whether to read every file — the cost the cache exists to avoid. A corpus
// file rewritten within a filesystem timestamp tick, to the same size, with
// different content, would go unnoticed; the guard against that is the build
// gate, which reads the real files regardless of what any cache believes.

// cacheVersion is bumped whenever the fragment layout changes. A cache written
// by an older winze decodes into the wrong shape or the wrong semantics, so the
// version mismatch discards it rather than trusting it.
const cacheVersion = 2

// fileKey identifies a file's content cheaply.
type fileKey struct {
	Size  int64
	MTime int64 // UnixNano
}

// fragment is one file's contribution to the index. Everything here is derived
// from that file alone — entity-var classification, which needs the corpus-wide
// role set, is deliberately deferred to the merge so a fragment never depends
// on another file and can be cached independently.
type fragment struct {
	Key        fileKey
	RoleTypes  []RoleType
	Defs       []SearchResult
	Candidates []VarRoleInfo // RoleType holds the raw composite type name, unclassified
	Literals   []LiteralField
	Pragmas    []Pragma
}

// cachePath resolves where this corpus's index lives. $WINZE_INDEX overrides;
// otherwise it is under the XDG cache dir, keyed by a hash of the corpus's
// absolute path so two corpora never collide and neither pollutes a repo.
func cachePath(dir string) string {
	if v := os.Getenv("WINZE_INDEX"); v != "" {
		return v
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	sum := sha256.Sum256([]byte(abs))
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "winze", "index-"+hex.EncodeToString(sum[:8])+".bin")
}

// cacheDisabled reports whether the cache is switched off. Set WINZE_NO_INDEX
// to bypass it — for benchmarking the cold path, and as the escape hatch if a
// cache is ever suspected of lying.
func cacheDisabled() bool {
	return os.Getenv("WINZE_NO_INDEX") != ""
}

// scanFiles returns the corpus's .go files with their cheap identity, sorted.
// Sorted because the merged index's slice order becomes query output order, and
// map iteration made that vary between runs.
func scanFiles(dir string) ([]string, map[string]fileKey, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var paths []string
	keys := map[string]fileKey{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		p := filepath.Join(dir, e.Name())
		paths = append(paths, p)
		keys[p] = fileKey{Size: info.Size(), MTime: info.ModTime().UnixNano()}
	}
	sort.Strings(paths)
	return paths, keys, nil
}

// loadCache reads the cache, returning an empty set on any problem. A cache is
// an optimisation: a missing, corrupt, or version-mismatched one must cost a
// reparse, never an error.
func loadCache(dir string) map[string]fragment {
	if cacheDisabled() {
		return nil
	}
	p := cachePath(dir)
	if p == "" {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()

	cachedDir, frags, err := decodeCache(f)
	if err != nil {
		return nil
	}
	abs, _ := filepath.Abs(dir)
	if cachedDir != abs {
		// A hash collision, or $WINZE_INDEX pointed at another corpus's cache.
		return nil
	}
	return frags
}

// storeCache writes the cache atomically: a temp file in the same directory
// followed by a rename, so a reader never observes a partial index and two
// winze processes writing at once cannot interleave. A failure here is
// deliberately silent — the answer has already been computed correctly, and
// failing a query because a cache could not be written would be the wrong
// trade.
func storeCache(dir string, paths []string, frags map[string]fragment) {
	if cacheDisabled() {
		return
	}
	p := cachePath(dir)
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	abs, _ := filepath.Abs(dir)

	tmp, err := os.CreateTemp(filepath.Dir(p), ".winze-index-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if err := encodeCache(tmp, abs, paths, frags); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
	}
}
