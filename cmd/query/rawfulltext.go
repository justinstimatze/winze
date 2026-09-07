package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
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
