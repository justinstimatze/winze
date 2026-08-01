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

## The plan

1. Push the defn index fix; land it upstream.
2. Add opt-in `OrderBy` / `WithDefName` to `LiteralFieldFilter` in defn so bulk
   callers stop paying for a sort and a join they discard.
3. Rewrite winze's read paths to ask narrow questions. `--claims X`,
   `--provenance X`, `--theories X` are all single-entity queries that currently
   build the entire index first.
4. `internal/defndb` becomes a thin defn client. **Delete `cache.go` and
   `codec.go`** — roughly 500 lines of index cache and wire format that exist
   only because of the stale comment.
5. Keep `astutil.ParseCorpus` (parallel, no object resolution). The write path's
   build gate parses real files regardless of any database, and that is correct:
   the gate must never trust a cache.

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
