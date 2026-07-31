# Vault ingest — a pile of markdown becomes a typed graph

```bash
winze-metabolism --pkm /path/to/vault --dry-run .   # report, generate nothing
winze-metabolism --pkm /path/to/vault --entity-cap 6000 .
winze-metabolism --pkm /path/to/vault --json .      # machine-readable summary
```

This is the answer to "why not just keep the markdown". A folder of notes — or
an OKF bundle, which is the same thing with frontmatter — already holds a graph:
the links between the notes. What it does not hold is anything that can *check*
that graph. Winze's claim is that compiling it is worth the trip.

## What it was doing before

Worth recording, because the gap was total. The previous extraction was written
against one demo vault and encoded that vault literally:

- categories came from a hardcoded `{books, productivity, coffee}` map, which
  materialised three entities in **every** vault whether or not those
  directories existed
- author extraction only ran inside a directory named `books`
- every note became a `Concept` regardless of content
- **wikilinks were parsed into a struct field that nothing ever read**

Measured on a six-note game-design vault with thirteen links between the notes:
**nine entities, zero claims** — a disconnected bag plus three phantom
categories from someone else's coffee notes. Every note in the vault knew what
it pointed at, and none of that survived. The same vault now yields nine
entities and fifteen claims, with three unresolved links reported by name.

## The mapping

Only what a note actually commits to:

| vault | corpus |
|---|---|
| note | entity; role from frontmatter `type` when it names a real winze role, else `Concept` |
| `[[link]]`, `[text](other.md)` | `References` claim, provenance quoting the linking note |
| frontmatter `aliases` | link-resolution keys |
| frontmatter `title` / H1 / filename | entity `Name` (in that precedence) |
| first prose sentence | entity `Brief` |
| directory | `FiledUnder` a category entity, one per directory that exists |

Nothing is inferred past that. A link under `## Depends on` is *evidence* for a
typed `DependsOn`, and the section is recorded and reported as a promotion
candidate — but the mechanical path never invents the predicate. A wikilink
commits to the reference and deliberately not to its kind, and manufacturing a
`DependsOn` from a heading would be fabricating a commitment the note never
made. Promoting a `References` edge to a typed predicate is a normal corpus
edit, done deliberately.

### Why `References` and `FiledUnder` exist

Both were forced, not invented. `References` has `*Entity` slots because a link
says nothing about what either end *is* — a design note may link a system to a
creature to a location, and `Concept/Concept` slots would either reject real
links or lie about the roles.

`FiledUnder` came out of the build gate rejecting the first attempt, which used
`BelongsTo`. `BelongsTo` is `Concept/Concept`, so a note typed `character` (a
`Person`) did not fit — and the meaning was wrong anyway: a creature filed under
`entities/` does not *belong to* the concept "Entities". The compiler caught an
ontology error in generated code and the ingest reverted the files. That is the
whole thesis in one incident.

## Link dialects

Real piles mix them, so both are handled:

- `[[Target]]`, `[[Target|display]]`, `[[Target#Heading]]`, `[[dir/Target]]`
- `[display](./other.md)`, `[display](other.md)`

Links inside fenced code blocks and inline code spans are **not** links. A
game-design vault quotes code constantly and `[[0]]` in a Go snippet is an array
index; treating it as an edge would put fabricated structure into the corpus.

Resolution tries, in order: exact relative path, path relative to the linking
note, filename, title, alias — all normalized, so `[[Combat Resolution]]`,
`[[combat-resolution]]` and `[[Combat_Resolution]]` name the same note.

## What it tells you

The summary is deliberately loud about what did **not** convert:

```
  5000 notes parsed in 90ms, 319 existing KB entities
  24832 links: 24832 resolved, 0 unresolved, 0 ambiguous
  5006 entities, 29832 claims (0 notes already in KB)
  5000 notes declared a frontmatter type
  link sections (promotion candidates): Depends on×24832
```

- **unresolved** — the link names no note. Usually a note not yet written, which
  is a real fact about the vault.
- **ambiguous** — two notes claim one name. Resolved to *neither*, and both are
  named in the report. Picking a winner would put a wrong edge in the graph, and
  a wrong edge is worse than a missing one.
- **orphans** — notes with no resolved link in either direction.
- **link sections** — where a typed predicate could be promoted from.

A summary that only counted successes would hide all four.

## Re-ingest

Ingest is idempotent and additive. A note whose ID already exists in the KB is
skipped, and new links are wired to the **existing** entity rather than coining
a duplicate — so a vault can be re-ingested as it grows, and hand-edits to
entities that came from it survive.

Output is deterministic: sorted, and carrying no generation timestamp. Ingesting
an unchanged vault twice leaves a clean tree.

## The gate

Generated files go through the same `go build` / `go vet` gate as every other
write path, and are **deleted** if it fails. There is no mode in which a bad
ingest lands in the corpus.

## Measured behaviour

On a synthetic 5000-note vault (20 MB, 24832 links, six directories):

| stage | time |
|---|---|
| parse + resolve 5000 notes | 90–112 ms |
| full ingest, dry run | 274 ms |
| ingest + gofmt + `go build` + `go vet` on 29832 claims | 15.3 s |
| `winze-query` on the resulting 5328-entity / 30287-claim corpus | ~360 ms |

The parse side is fast and scales linearly — notes are parsed in parallel and
resolution is a map lookup per link, not a scan. The 15 s is Go compiling 35k
declarations, paid once per ingest.

The ~360 ms query is the honest weak spot: `winze-query` re-parses the corpus
AST on every invocation, so at this scale every query pays a fixed ~350 ms
regardless of what it asks for. Fine for a batch phase, too slow to feel
interactive during content generation. A persistent index is the fix, and it is
not built yet.
