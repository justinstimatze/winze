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

## Reading a meld

`winze-query` reads a meld through a parse tree instead of through defn, and
picks which on the presence of `.winze-meld.json`. Every mode works —
`--stats`, `--claims`, `--theories`, `--provenance`, `--disputes`,
`--decisions`, `--fulltext`, `--hybrid`:

```bash
winze-query --hybrid "how minds fail at modeling reality" /path/to/meld
winze-query --claims SomeEntity /path/to/meld
```

defn cannot serve a meld and never will be able to. Its ingest runs
`packages.Load`, which wants a module that type-checks, and a meld is built not
to be one: no `go.mod`, duplicate identifiers across the melded stores (which is
what makes it read-only), and `winze_self.go` importing `winze/internal` from
outside the winze module. A parser is stopped by none of the three.

The seam is `corpusSource` in `cmd/query/corpussource.go` — the six calls every
mode reaches its data through. `*defndb.Client` already satisfied it, so the AST
reader is one implementation of the same interface producing the same
`LiteralField` stream, and no mode had to change. Its correctness is pinned by a
differential rather than by inspection: `TestASTSourceMatchesDefn` reads the real
corpus both ways and requires the two streams to be identical, field for field,
including the source file and line. On the same corpus every mode's output is
byte-identical between the backends.

The routing is on the manifest and not on defn failing to open. A fallback would
turn a stale index or a broken ingest into a silent switch to a different reader
— same command, different answers, no symptom.
