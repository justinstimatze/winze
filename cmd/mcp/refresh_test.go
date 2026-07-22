package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHandler_IndexAutoRefresh proves the MCP server reflects a corpus write
// without a restart: a second session appending an entity becomes visible to
// the next query. Without the mtime-triggered rebuild in handler.index(), the
// server would answer from the snapshot built at startup forever.
func TestHandler_IndexAutoRefresh(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "corpus.go")

	writeCorpus := func(names ...string) {
		var b strings.Builder
		b.WriteString("package winze\n")
		b.WriteString("type Entity struct {\n\tID, Name, Kind, Brief string\n\tAliases []string\n}\n")
		b.WriteString("type Concept struct{ *Entity }\n")
		for _, n := range names {
			fmt.Fprintf(&b, "var %s = Concept{&Entity{Name: %q}}\n", n, n)
		}
		if err := os.WriteFile(file, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	entityCount := func(h *handler) int {
		stats, ok := h.coreStats().(map[string]any)
		if !ok {
			t.Fatalf("coreStats returned %T, want map", h.coreStats())
		}
		n, ok := stats["entities"].(int)
		if !ok {
			t.Fatalf("entities field is %T, want int", stats["entities"])
		}
		return n
	}

	writeCorpus("Foo")
	kb, err := buildIndex(dir)
	if err != nil {
		t.Fatalf("initial buildIndex: %v", err)
	}
	h := &handler{dir: dir}
	h.cur.Store(kb)
	h.builtAt.Store(corpusMaxMtime(dir))

	if got := entityCount(h); got != 1 {
		t.Fatalf("initial entity count = %d, want 1", got)
	}

	// A different session appends an entity. Bump mtime explicitly into the
	// future so the test does not depend on filesystem mtime granularity.
	writeCorpus("Foo", "Bar")
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(file, future, future); err != nil {
		t.Fatal(err)
	}

	if got := entityCount(h); got != 2 {
		t.Fatalf("after corpus write, entity count = %d, want 2 — auto-refresh did not fire", got)
	}
}

// TestCorpusMaxMtime_IgnoresNonGo confirms the mtime sweep only considers the
// corpus .go files, so a touch of an unrelated file does not trigger rebuilds.
func TestCorpusMaxMtime_IgnoresNonGo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package winze\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := corpusMaxMtime(dir)
	if base == 0 {
		t.Fatal("expected non-zero mtime for a .go file")
	}
	// A newer non-.go file must not raise the max.
	future := time.Now().Add(time.Hour)
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(readme, future, future); err != nil {
		t.Fatal(err)
	}
	if got := corpusMaxMtime(dir); got != base {
		t.Errorf("non-.go file changed the corpus mtime: got %d, base %d", got, base)
	}
}
