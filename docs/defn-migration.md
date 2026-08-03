# defn: measurements, defects found, and the migration plan

**Measured 2026-08-01 against defn at `4d9d0a0`.** Everything here is a dated
observation, not a standing fact. Re-measure before trusting it.

## Why this document exists

`internal/defndb` carried a package comment explaining that defn was backed by
Dolt, linked ~190 indirect modules, and made `go build` cost minutes. That was
true when it was written and had silently stopped being true. It was read as a
live measurement rather than a dated decision, and on the strength of it winze
grew a persistent index cache (`internal/defndb/cache.go`, `codec.go`) that
reimplements a subset of what defn already does.

That is the failure mode this corpus is *about*. A claim with no live check on
it rots, and prose that reads like a measurement should carry the date it was
taken. The typed-citation primitive exists precisely so a reference cannot go
stale silently — and the thing that rotted was a paragraph of prose no citation
covered.

## What defn actually is now

| | then (the stale comment) | now, measured |
|---|---|---|
| engine | Dolt (go-mysql-server, vitess) | `modernc.org/sqlite`, pure Go, no cgo |
| module graph | ~190 indirect | 102 total |
| build cost | "minutes, 1–2 GB RSS at link" | **0.859s** for a binary linking `defn/db`, warm cache |

`defn/dbclient` — the MySQL-wire client whose own doc says it was split out *for
winze* — is now dead in both directions: nothing inside defn imports it, and its
premise (talking to a Dolt sql-server to avoid linking Dolt) no longer applies.

## The API is a superset of what defndb hand-rolls

| winze `defndb` | defn `db` |
|---|---|
| `ClaimFields()` | `LiteralFields(LiteralFieldFilter{FieldNames: []string{"Subject","Object","Prov"}})` |
| `EntityFields()` | `LiteralFields(LiteralFieldFilter{FieldNames: ...})` |
| `LiteralFieldsForType(p)` | `LiteralFields(LiteralFieldFilter{TypeName: p})` |
| `Pragmas(prefix)` | `Pragmas("winze:%")` |
| `Search(pattern)` | `Definitions(DefinitionFilter{Name: pattern})` / `Search()` |
| *(nothing)* | `Refs()`, `Traverse(name, direction, kinds, depth)` — multi-hop |
| *(nothing)* | `StaleFiles(dir)` — incremental invalidation |

Filtering happens in SQL, so a narrow question does not materialise the whole
table first. `Traverse` is multi-hop graph walking winze does not have at all.

## Two defects found while profiling it

Ingesting the 5000-note synthetic corpus (250k lines, 30287 claims) took 45s.
Then the query winze runs interactively — "what claims involve entity X",
which filters `literal_fields.field_value` because a claim's Subject and Object
hold the var names they point at:

```go
d.LiteralFields(db.LiteralFieldFilter{Value: "Note00042%"})
```

**9 rows out of 132733, in 193ms.** `EXPLAIN QUERY PLAN` said `SCAN`.

1. **No index on `field_value`.** `type_name`, `field_name`, `def_id` and
   `(type_name, field_name)` are all indexed; the column that answers "what
   refers to X" was not.
2. **And the index alone does not help.** SQLite cannot use an index for `LIKE`
   while `case_sensitive_like` is OFF, which is the default defn runs with.
   Adding the index left the plan at `SCAN`. An anchored prefix is equivalent to
   a half-open range, and a range *does* use the index.

Fixed in defn: added `idx_litfield_value`, and `likePrefixRange` rewrites
`field_value LIKE 'Foo%'` to `>= 'Foo' AND < 'Fop'` in `QueryLiteralFields`.
Only provably-equivalent patterns are rewritten; `'%foo%'` and interior `_`
fall through unchanged.

```
one entity, 9 rows of 132733:  193ms -> 0.14ms   (through db.LiteralFields)
raw SQL, same predicate:      36.9ms -> 0.06ms
```

Status: committed on defn branch `claude/google-markdown-yaml-comparison-k59ii3`,
**not yet pushed** — the repo was attached read-only.

## The gap that remains, honestly

A naive swap would be *slower* for winze's current access pattern. Bulk
`LiteralFields` over 90841 rows costs 875ms through defn vs 102ms for winze's
whole warm query. Isolated on the same corpus:

| | cost |
|---|---|
| unconditional `ORDER BY type_name, field_name` | ~103 ms |
| unconditional `LEFT JOIN definitions` for `DefName` | ~210 ms |
| row materialisation | remainder |

Both want an opt-in on `LiteralFieldFilter` rather than a silent change, since
dropping the ORDER BY alters result order for existing callers.

**But that gap is mostly winze's fault.** `buildIndexDefn` loads every claim,
entity and provenance to answer `--claims Tunguska`. Against defn that is one
indexed lookup. The migration is not "point defndb at defn" — it is *stop
building a whole-corpus index to answer a question about one entity*.

## Ingest is corpus-scoped (2026-08)

A second whole-corpus waste was in the *write* path. `defndb.ensureFresh` fires
`db.Sync` — `packages.Load("./...")` — on any `.go` change. While the corpus
lived at the module root, `./...` matched the whole module: the ~47-file corpus
*plus* all of `cmd/` and `internal/`. Measured split (warm, min-of-6): ~1991 ms
per write = type-check 593 ms + DB insert/resolve 1398 ms, ~half of it indexing
tooling the queries never read.

The fix, applied winze-side: the corpus was moved into a `corpus/` subdirectory
of the same module, and every ingest/gate points at `corpus/`.
`packages.Load("./...", Dir=corpus/)` matches only the corpus package — sibling
`cmd/`/`internal/` trees are excluded from both the type-check target and the DB
insert (`IngestPackages` iterates only matched packages). The corpus stays **one
Go package**, so the free cross-file referential integrity (the reason for the
single-package shape) is untouched; `source_file` stays a bare basename because
the corpus sits flat in the projectDir (`filepath.Rel` reduces to the basename).

**Measured 2026-08-03** (min-of-6, fresh `.defn` each iteration, full `db.Sync`):

| | min | vs whole-module |
|---|---|---|
| whole-module `Sync(repo)` | 2580 ms | — |
| corpus-scoped `Sync(corpus)` | 925 ms | **2.79× faster, 36% of the work** |

Indexing `cmd/`+`internal/` was ~64% of every write's ingest — the move saves
*more* than the conservatively-projected half. The host was not quiet (load
~6–7), so the **absolutes are contention-inflated**; the 925 ms-under-load is
consistent with the ~850 ms quiet-box projection. The **ratio is the robust
figure** — both halves pay the same contention tax, so it cancels.

Note: defn v0.26.0 *did* ship the scoped-sync ask (`db.SyncPattern(projectDir,
pattern)`), but it is moot for winze now — `SyncPattern(repo, ".")` errors with
`no Go files in .../winze` because the module root no longer holds a package to
scope to. The subdirectory move stands on its own.

## The plan

1. ~~Push the defn index fix; land it upstream.~~ **Done 2026-08-01, defn main
   `7093ab9`** (`store: index literal_fields.field_value + opt-out ORDER BY/DefName
   for bulk callers`).
2. ~~Add opt-in `OrderBy` / `WithDefName` to `LiteralFieldFilter`.~~ **Done, but
   as opt-*out* `SkipOrderBy` / `SkipDefName` (same commit).** An opt-in bool
   defaults false the moment it ships, silently dropping ordering and DefName for
   every existing `LiteralFieldFilter` caller — the regression this doc's own
   constraint was guarding against. Opt-out makes the fast path something a caller
   must ask for. The bulk call site is
   `LiteralFields(LiteralFieldFilter{Value: "X%", SkipOrderBy: true, SkipDefName: true})`.
3. ~~Rewrite winze's read paths to ask narrow questions.~~ **Done 2026-08-01.**
   `--claims X`, `--provenance X`, `--theories X` now resolve one target and
   stream only the claims that name it, via `defndb.EachFieldWithValuePrefix` +
   `EachFieldForDefs` (the seams that map onto the `Value:"X%"` / `def_id IN (...)`
   SQL filters). Still over the AST backend — the whole-corpus parse is unchanged
   until step 4 — but the query shape is now narrow and the map funnel is gone.
   That funnel was also a live bug: `--claims Chalmers` resolved to a *different
   entity* each run (Go map order), so the read paths returned non-reproducible
   answers. They are deterministic now, and `--theories` stopped resolving to
   trip-cycle bookkeeping entities as a side effect.
4. ~~`internal/defndb` becomes a thin defn client. **Delete `cache.go` and
   `codec.go`**.~~ **Done 2026-08-02, against defn main `2ec0eaa`** (which adds
   `db.Sync` — a full `packages.Load` re-ingest that sets `last_ingest`).
   `cache.go`, `codec.go`, and `cache_test.go` (~950 lines) are gone. `defndb`
   now opens `.defn`, and `New` ingests on first use / re-ingests when
   `db.StaleFiles` reports a changed `.go` file, so an unedited corpus pays only
   the stat walk. The whole consumer API (`RoleTypes`, `EntityVarsWithRoles`,
   `ClaimFields`, `EachField…`, `Pragmas`, `Search`) is preserved over SQL, with
   `SkipOrderBy`/`SkipDefName` set on every bulk `LiteralFields` call.

   Three things surfaced only once the client drove real ingest over the whole
   winze module, none of them in the plan:

   - **`db.Sync` assumes a single writer.** Two overlapping full ingests collide
     on the `definitions` unique index inside `UpsertDefinitionsBulk`. Serialized
     it is clean (a re-Sync even prunes stale rows). `defndb` now takes an OS
     advisory lock on `.defn/sync.lock` around the check-and-sync, which also
     protects concurrent winze processes; reported to defn as a `db.Sync`
     property.
   - **defn records pragma doc-comments as file-level** (`def_id` NULL) instead
     of binding them to the definition they sit directly above, so `db.Pragmas`
     returns an empty `DefName` for every `//winze:functional` / `//winze:contested`
     pragma. `Pragmas` reconstructs the binding client-side (`defAfter`: the
     nearest definition starting after the pragma line) until defn's ingest links
     them; `TestConcordance_Pragmas` now asserts `DefName` is populated so the gap
     can't hide again. Reported to defn.
   - **A defn client needs a real module, not loose files.** `packages.Load`
     requires `go.mod`; the AST loaders parsed any directory. The mcp
     auto-refresh test grew a bare `go.mod` in its fixture. In production the
     corpus is always the winze module, so this is a test-only adjustment.

   Separately, regenerating `predictions.go` for this work exposed a **metabolism
   codegen bug**: the external-sensor reify loop never ran `sanitizeIdent`, so a
   `goal:Foo` LearningGoal hypothesis name leaked its colon into a Go identifier
   (`var EvidenceSearchgoal:Goal…`) and broke the build. Fixed in
   `cmd/metabolism/reify.go` — the KB-internal loop already sanitized; the
   external loop now does too.
5. ~~Keep `astutil.ParseCorpus`.~~ **Kept.** The write path's build gate parses
   real files regardless of any database, and that is correct: the gate must
   never trust a cache.

## What to keep from the stopgap

Not the code — the measurements, which transfer to whatever backs the index:

- **String interning matters at this shape.** 132733 literal records are six
  mostly-repeated strings each; gob wrote 34 MB and cost 357ms to encode, 136ms
  to decode, against a 193ms parse. A cache that costs more than the work it
  replaces is not a cache.
- **Iterators beat filtered copies.** `ClaimFields` built a 90841-record copy
  the caller walked exactly once.
- **Index order feeds output order.** Walking Go maps made query output vary
  between runs; that fix is worth keeping wherever the index lives.
