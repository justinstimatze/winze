# winze-meld (read-only union of stores)

`winze-meld` bridges two or more winze stores into a single read-only
"mind-meld" directory you can point the query tools at, then dissolve when done.

```bash
winze-meld winze-memory .                    # union of two stores, into a mktemp dir
winze-meld --out /tmp/meld  a b c            # ...or a dir you name
winze-meld a@v0.3.0 b@HEAD~5                 # pin either store at a ref
winze-meld --dissolve /tmp/meld              # tear it down
```

## What counts as a store

meld does not assume a layout. A store's corpus is **the one directory in its
tree whose `.go` files declare `package winze`** — `corpus/` in a winze
checkout, the repo root in a store `winze-agent init` scaffolded. Both are
correct, and the meld dir flattens either.

The two shapes exist for different reasons and neither is going away. This repo
keeps its corpus under `corpus/` so defn ingest stops indexing `cmd/` and
`internal/` on every write. A store scaffolded by `winze-agent init` keeps the
same files flat at its root, which is what makes `go build .` there a working
consistency gate — nesting it would cost that gate and buy nothing, since a
store has no tooling to exclude in the first place.

Detecting rather than configuring is also the only thing that survives a pinned
SHA: a layout marker committed today does not exist at last month's commit, and
a meld's promise is that its manifest reproduces it. `winze-agent` already
treats the question the same way — `findCorpusSource` (`cmd/agent/init.go`)
looks for the corpus instead of naming it.

The package clause is the marker because Go allows exactly one per directory, so
it cannot drift within the corpus, and because two `package winze` sets
colliding is precisely what makes a meld read-only. A store with no such
directory is rejected; so is one with more than one, listed by name rather than
guessed at.

## What a meld is

A FROZEN snapshot. Each store is materialized via `git archive` at a pinned SHA
(HEAD by default), so the meld never couples to a store's live working tree.
`.winze-meld.json` records each store's path, SHA, namespace, and detected
corpus dir, and it is the guard `--dissolve` checks before removing anything.

Corpus files are copied namespace-prefixed (`<ns>__memory.go`), the prefix
derived from the store's directory name. It survives into query results as the
source label, so a hit tells you which store it came from. One canonical
`predicates.go` is kept from the primary (first) store so `LoadPredicates`
resolves. Cross-store var-name collisions are surfaced, not merged — both
entities appear, each tagged by store.

Read-only by construction, not by policy: two stores both declaring
`package winze` and both defining `Entity`, `Claim`, and the predicate vars mean
the union cannot `go build`. The write path (`winze-add` / `winze-edit`) runs
that build as its gate, so it does not apply to a meld.

## Known gap: nothing queries a meld today

`winze-query` reaches a corpus through `defndb.New(dir)` in every mode, and
defn's ingest runs `packages.Load("./...")`, which needs a module that
type-checks. A meld is a directory that deliberately does not:

- no `go.mod` (add one and you get past this to the next two)
- duplicate identifiers across the melded stores, which is the whole design
- `winze__winze_self.go` imports `winze/internal/corpuslock` and
  `internal/corpusparse`, unresolvable outside the winze module

So `winze-meld` builds a correct union dir and there is currently no reader for
it. This predates corpus detection and is not caused by it — the package doc
here used to claim winze-query "AST-scrapes composite literals without
type-checking", which described the pre-defn query. Closing it means either an
AST-only read path in `cmd/query`, or an ingest that tolerates package errors
and indexes syntax.
