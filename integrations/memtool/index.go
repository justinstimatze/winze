package memtool

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

func loadIndex(path string) (*index, error) {
	ix := &index{path: path, entries: map[string]indexEntry{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ix, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading index at %s: %w", path, err)
	}
	if len(data) == 0 {
		return ix, nil
	}
	if err := json.Unmarshal(data, &ix.entries); err != nil {
		return nil, fmt.Errorf("parsing index at %s: %w", path, err)
	}
	return ix, nil
}

// delete removes path, and everything nested under it as a directory, from
// the index only. The underlying winze entities are untouched -- winze
// refuses hard deletion by design, and this stays true to that: a "deleted"
// memory-tool file just becomes unreachable through view/create, not gone.
func (ix *index) delete(path string) {
	delete(ix.entries, path)
	prefix := strings.TrimSuffix(path, "/") + "/"
	for p := range ix.entries {
		if strings.HasPrefix(p, prefix) {
			delete(ix.entries, p)
		}
	}
}

func (ix *index) get(path string) (indexEntry, bool) {
	v, ok := ix.entries[path]
	return v, ok
}

// listPrefix returns every path under dir, sorted, for a view command
// targeting a directory rather than a single file.
func (ix *index) listPrefix(dir string) []string {
	prefix := strings.TrimSuffix(dir, "/") + "/"
	var out []string
	for p := range ix.entries {
		if strings.HasPrefix(p, prefix) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// rename moves path (and everything nested under it, for a directory rename)
// to newPath. Index-only, like delete: no winze entity is touched, so a
// rename costs nothing on the winze side. Returns how many entries moved.
func (ix *index) rename(oldPath, newPath string) int {
	renamed := 0
	if v, ok := ix.entries[oldPath]; ok {
		delete(ix.entries, oldPath)
		ix.entries[newPath] = v
		renamed++
	}
	oldPrefix := strings.TrimSuffix(oldPath, "/") + "/"
	newPrefix := strings.TrimSuffix(newPath, "/") + "/"
	var matches []string
	for p := range ix.entries {
		if strings.HasPrefix(p, oldPrefix) {
			matches = append(matches, p)
		}
	}
	for _, p := range matches {
		v := ix.entries[p]
		delete(ix.entries, p)
		ix.entries[newPrefix+strings.TrimPrefix(p, oldPrefix)] = v
		renamed++
	}
	return renamed
}

func (ix *index) save() error {
	data, err := json.MarshalIndent(ix.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ix.path, data, 0o644)
}

func (ix *index) set(path string, entry indexEntry) {
	ix.entries[path] = entry
}

// index maps a memory-tool virtual path (e.g. "/memories/preferences.md") to
// the winze entity that holds it, and persists that mapping beside the
// winze store itself. Winze has no directory concept and no path identifier
// of its own -- this file is the only place that translation lives, and it
// never touches winze's corpus or schema.
type index struct {
	path    string
	entries map[string]indexEntry
}

// indexEntry pairs a winze entity's var name with the title it was created
// under. creationTitle is kept separate from the (renamable) virtual path
// because winze has no rename primitive: an entity's Name is fixed at
// creation, so a lookup must always search by the title winze actually has,
// not by whatever path currently maps to it.
type indexEntry struct {
	Var           string `json:"var"`
	CreationTitle string `json:"creation_title"`
}
