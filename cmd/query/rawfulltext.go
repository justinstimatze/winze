package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/justinstimatze/winze/internal/cliutil"
)

// buildRawFTIndex indexes each raw log entry's Note as a document, reusing
// the same BM25 engine buildFTIndex builds for entities and provenance
// (cmd/query/fulltext.go) -- ftDoc's kind+ref pair is already opaque to the
// scoring math, so "raw" slots in as a third kind with no change to ftIndex,
// ftDoc, ftHit, or search.
func buildRawFTIndex(docs []rawDoc) *ftIndex {
	fi := &ftIndex{df: map[string]int{}}
	total := 0
	for i, d := range docs {
		toks := tokenize(d.Note)
		if len(toks) == 0 {
			continue
		}
		tf := make(map[string]int, len(toks))
		for _, t := range toks {
			tf[t]++
		}
		fi.docs = append(fi.docs, ftDoc{kind: "raw", ref: i, terms: tf, len: len(toks)})
		for t := range tf {
			fi.df[t]++
		}
		total += len(toks)
	}
	fi.n = len(fi.docs)
	if fi.n > 0 {
		fi.avgLen = float64(total) / float64(fi.n)
	}
	return fi
}

// loadRawDocs reads a store's raw.jsonl -- the write-side evidence log
// cmd/agent/rawlog.go's appendRawLog maintains, one JSON object per line. A
// missing file is not an error: a store with no raw writes yet has nothing
// to load.
func loadRawDocs(dir string) ([]rawDoc, error) {
	f, err := os.Open(filepath.Join(dir, "raw.jsonl"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var docs []rawDoc
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var d rawDoc
		if err := json.Unmarshal(line, &d); err != nil {
			continue // best-effort read, same posture as the writer
		}
		docs = append(docs, d)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return docs, nil
}

// runRawFulltext answers --raw: BM25 search over a store's raw.jsonl
// evidence log. Unlike every other query mode, this does not need the typed
// corpus index at all -- raw.jsonl is written independently of whether the
// corpus builds -- so main dispatches this before buildIndex, the same
// early-dispatch treatment --docs-recall already gets.
func runRawFulltext(dir, query string, jsonOut bool) {
	docs, err := loadRawDocs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query --raw: %v\n", err)
		os.Exit(1)
	}
	fi := buildRawFTIndex(docs)
	hits := fi.search(query, 15)

	if jsonOut {
		out := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			d := docs[h.ref]
			out = append(out, map[string]any{
				"score": h.score,
				"time":  d.Time,
				"tool":  d.Tool,
				"var":   d.Var,
				"note":  d.Note,
			})
		}
		printJSON(map[string]any{"query": query, "count": len(hits), "hits": out})
		return
	}

	if len(hits) == 0 {
		fmt.Printf("No raw-evidence matches for %q\n", query)
		return
	}
	fmt.Printf("Raw-evidence matches for %q (%d):\n\n", query, len(hits))
	for _, h := range hits {
		d := docs[h.ref]
		fmt.Printf("  [%.2f] %s  %s (%s)\n", h.score, d.Time, d.Tool, d.Var)
		fmt.Printf("        %s\n", cliutil.Truncate(d.Note, 200))
	}
}

// rawDoc mirrors cmd/agent/rawlog.go's rawLogEntry wire shape by field name,
// not by import -- cmd/query and cmd/agent are sibling main packages that
// don't import each other, the same pattern internal/corpusparse's CodeRef
// mirror already uses against corpus/schema.go's CodeRef.
type rawDoc struct {
	Time string `json:"time"`
	Tool string `json:"tool"`
	Var  string `json:"var"`
	Note string `json:"note"`
}
