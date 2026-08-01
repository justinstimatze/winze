# Query interface

```bash
go run ./cmd/query "consciousness" .              # substring search entities by name/brief/alias
go run ./cmd/query --fulltext "pattern detection failure" .  # BM25 fulltext ranking over Briefs + provenance Quotes
go run ./cmd/query --semantic "machine that seems to understand but does not" .  # embedding search via local ollama all-minilm (~53ms cached)
go run ./cmd/query --hybrid "confirmation bias" .   # reciprocal-rank-fusion of BM25 + semantic into one list
go run ./cmd/query --hybrid "consciousness" --type Hypothesis .  # type-aware: filter hybrid results to a verified role (zero-classification-error)
go run ./cmd/query --hybrid "apophenia" --expand .  # append each hit's typed claim neighborhood (predicate → neighbor + role) — reasoning-ready context
go run ./cmd/query --dupes ConfirmationBias .       # structural twins: same-role entities sharing this one's claim-neighborhood (coin-time dedup)
go run ./cmd/query --theories "apophenia" .        # competing theories of a concept
go run ./cmd/query --claims "Chalmers" .           # all claims involving an entity
go run ./cmd/query --provenance "Sagan" .          # provenance trail for a source
go run ./cmd/query --disputes .                    # all active disputes
go run ./cmd/query --stats .                       # KB summary statistics
go run ./cmd/query --schema .                       # the type model: roles, predicate signatures, attribution modes
go run ./cmd/query --reverie .                      # associative walk over the claim graph (random start; add a seed to steer); the grounded "trip"
go run ./cmd/query --decisions .                    # the decision log: current vs superseded, over the Supersedes graph (see docs/decisions.md)
go run ./cmd/query --docs-recall "<prompt>" .       # semantic recall over docs/*.md: the file#anchor sections a prompt implicates
go run ./cmd/query --json "consciousness" .        # JSON output
go run ./cmd/query --ask "What theories compete on consciousness?" .  # LLM-powered natural language query
go run ./cmd/query --ask .                         # interactive REPL mode
```

The read side of the KB. Parses corpus `.go` files with `go/ast`, builds
an in-memory index of entities, claims, and provenance, and answers queries.
`--ask` mode sends the full KB context to an LLM for natural language answers
(needs ANTHROPIC_API_KEY). `--docs-recall` operates on the prose docs, not the
corpus — it is the read side of the split-CLAUDE.md docs (see
`docs/docs-recall.md`). For richer queries (multi-hop, aggregation), use
`defn` MCP directly in a Claude Code session.

## The index cache

Every winze CLI invocation is a fresh process, so nothing survived between
queries: a `--claims` lookup reparsed the whole corpus, every time. On a small
corpus that is invisible. On a 250k-line one it was ~360ms per query regardless
of what was asked, which is fine for a batch phase and too slow to feel
interactive during content generation.

The read side now keeps a persistent index under
`$XDG_CACHE_HOME/winze/index-<hash>.bin`, keyed by the corpus's absolute path.
`$WINZE_INDEX` overrides the location; `WINZE_NO_INDEX=1` bypasses it entirely,
which is both the benchmarking switch and the escape hatch if a cache is ever
suspected of lying.

Measured on a 5328-entity / 30287-claim corpus (250k lines, 51 files):

| | |
|---|---|
| before | ~360 ms |
| parallel parse, no object resolution | 236 ms |
| + warm index cache | **~102 ms** |
| cold (cache miss, full parse + write) | 235 ms |

The real corpus (322 entities) answers in ~6 ms.

### How it invalidates

Per **file**, not per corpus, because that is what makes the common case cheap:
`winze-add` and a hand edit both touch one corpus slice, so one fragment is
reparsed and the other fifty are decoded. A whole-corpus cache would be correct
and would throw all of it away on every write.

The signal is `(size, mtime)` — what `go build` itself trusts. Content hashing
would mean reading every file to decide whether to read every file, which is the
cost the cache exists to avoid. A file rewritten inside one timestamp tick, to
the same size, with different content would slip past; the guard against that is
the build gate, which reads the real files regardless of what any cache believes.

Entity-role classification is deliberately **not** cached per file. A role type
declared in `roles.go` decides how a var in another file is read, so that
decision is recomputed at merge time from the corpus-wide role set — otherwise
adding a role would leave every already-cached file misclassified, invisibly.

### Why not gob

It was the first attempt and measured as a net loss: 34 MB, 357 ms to encode and
136 ms to decode, against a 193 ms parse. A cache that costs more than the work
it replaces is not a cache.

The cost is structural. An index record is six mostly-repeated strings —
`DefName` repeats once per field in a var, `TypeName` is one of a few dozen
predicate names, `FieldName` is one of about ten, `SourceFile` is constant
within a file, and even `FieldValue` repeats, because a claim's Subject and
Object are var names that recur throughout the corpus. Any encoding that writes
those inline pays for the same bytes 130k times.

The format now writes each distinct string once into a table and refers to it by
varint index: 4.4 MB, 16 ms to decode. Format details are winze-internal and
versioned; a cache that fails to decode for any reason is discarded and rebuilt,
so changing it needs only a version bump.

### Correctness

A stale index that silently answers with old content is worse than a slow one,
because the caller cannot tell. `internal/defndb/cache_test.go` asserts the
cached index is byte-identical to the uncached one across every mutation the
corpus undergoes — edits (including same-size edits), additions, deletions, role
types added in another file, unparseable files mid-edit — and that corruption,
truncation, version mismatch, an unwritable location, and a cache belonging to a
different corpus all degrade to a reparse rather than to a wrong answer.
