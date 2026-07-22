package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// The lint binary runs 9 rules, and before this each rule (and the defndb AST
// shim, twice) re-walked and re-parsed the ~30 corpus files independently — 17
// full parses per run, most of the wall time. These two dir-keyed caches parse
// each corpus ONCE per process and share it: corpusFiles for the go/ast
// collectors, parseCorpusCached for the corpusparse-based rules. Keyed by dir
// so tests with distinct temp dirs stay isolated; process-lifetime, since a
// lint run reads a fixed snapshot of the corpus.

type parsedCorpus struct {
	fset  *token.FileSet
	files []*ast.File
	err   error
}

var (
	filesMu    sync.Mutex
	filesCache = map[string]*parsedCorpus{}
)

// corpusFiles parses every non-test .go file in dir once (with comments, so
// pragma-reading rules work) and returns the shared FileSet and files. All
// callers must use the returned FileSet for position info — the *ast.File nodes
// are tied to it.
func corpusFiles(dir string) (*token.FileSet, []*ast.File, error) {
	filesMu.Lock()
	defer filesMu.Unlock()
	if pc, ok := filesCache[dir]; ok {
		return pc.fset, pc.files, pc.err
	}
	pc := &parsedCorpus{fset: token.NewFileSet()}
	entries, err := os.ReadDir(dir)
	if err != nil {
		pc.err = err
	} else {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			f, perr := parser.ParseFile(pc.fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
			if perr != nil {
				pc.err = perr
				pc.files = nil
				break
			}
			pc.files = append(pc.files, f)
		}
	}
	filesCache[dir] = pc
	return pc.fset, pc.files, pc.err
}

type parsedEntities struct {
	entities []corpusparse.Entity
	claims   []corpusparse.Claim
	err      error
}

var (
	cpMu    sync.Mutex
	cpCache = map[string]*parsedEntities{}
)

// parseCorpusCached memoizes corpusparse.ParseCorpus per dir, so the rules that
// use it (brief-drift, structural-dedup) share one parse instead of three.
func parseCorpusCached(dir string) ([]corpusparse.Entity, []corpusparse.Claim, error) {
	cpMu.Lock()
	defer cpMu.Unlock()
	if pc, ok := cpCache[dir]; ok {
		return pc.entities, pc.claims, pc.err
	}
	ents, claims, err := corpusparse.ParseCorpus(dir)
	cpCache[dir] = &parsedEntities{entities: ents, claims: claims, err: err}
	return ents, claims, err
}
