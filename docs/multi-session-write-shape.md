# Multi-session write shape: winze as a shared, low-drift brain

## The goal

Point several agent sessions (parallel Claude Code windows, a team, the
metabolism loop) at one winze repo and use it as shared knowledge storage —
a structured `CLAUDE.md` that doesn't drift, because every cross-reference is
a typed var the compiler validates instead of a prose mention nobody checks.

For that to be worth the friction over markdown, two things have to hold:

1. **Reads stay near-markdown speed.**
2. **Concurrent writes don't corrupt the corpus or fight each other in git.**

Both hold. This note records the shape and the measured numbers behind it.

## Measured baseline (43 files / 12,882 lines, warm, compiled binaries)

| operation | latency | notes |
|---|---|---|
| `cat` one corpus file | 2.8 ms | markdown-read baseline |
| winze single-entity query | 24 ms | parses + indexes the **whole** corpus every call |
| winze `--stats` (aggregate) | 25 ms | same |
| write gate `go build . && go vet .` | 91 ms | the per-write cost |
| `go build .` alone | 25 ms | |
| `gofmt -w` one file | 2.5 ms | |

Reads are ~8× a `cat` but 24 ms is imperceptible interactively, and parse cost
is linear in total lines — so there is roughly **10× headroom** (to ~250 ms,
the interactive threshold) before reads need a persistent index. Reads are not
the blocker. The markdown gap lives entirely on the **write** path: the 91 ms
gate is the compiler refusing to accept a dangling reference. That cost *is*
the anti-drift feature; you pay it on writes, not reads.

## The shape: per-session files as a write-ahead log

winze already uses this pattern without naming it: `metabolism_cycle*.go` and
`pkm_*.go` are cases of *one automated writer owns a filename-prefix namespace
and only appends to its own files*. Per-session generalizes it to N writers.

```
winze/
  apophenia.go          <- curated topic files   = the compacted store
  consciousness.go      <-   (consolidation owns these)
  ...
  session_a1b2.go       <- session A's log        = write-ahead log
  session_c3d4.go       <- session B's log          (each session owns one)
  metabolism_cycle*.go  <- the metabolism writer's log (already exists)
```

All `package winze`, all in the repo root (a Go package is one directory, so
the separation is a filename prefix, not a subdirectory). Because it is one
flat package, **a claim in `session_a1b2.go` that references `Apophenia`
(declared in `apophenia.go`) type-checks across the file boundary for free.**
That cross-session referential integrity is the thing an Obsidian vault of
`[[wikilinks]]` cannot enforce.

### Why this removes both concurrency failure modes

| failure mode | one shared file | per-session file |
|---|---|---|
| git conflict (two sessions append at EOF) | conflicts every time | disjoint files -> `git pull --rebase` is clean |
| gate race (concurrent `go build .` in one tree) | one session sees another's half-written append | each session in its own worktree -> gate runs isolated |

Worktree-per-session is the *conflict-free* path: disjoint files and isolated
gates mean zero contention. Gas Town polecats already clone into
`polecats/<name>/winze/`.

### Sharing one worktree: the corpus lock

Not every multi-session user sets up a worktree per session — the simplest
"point all my parallel windows at one repo" shape has N sessions writing the
**same** working tree. There the gate race is real: the gate is `go build .`
over the whole directory, so one session running its gate while another is
mid-write sees the other's half-written file, and three things break — a lost
update (B's write overwrites A's from a stale backup), a revert clobber (A's
failed gate reverts to A's backup, wiping B's committed write), and a
cross-file **false revert** (B's gate fails on A's syntactically-broken
intermediate and B reverts its *own valid* change, even for a write to a
different file).

`internal/corpuslock` closes all three: every mutator (`cmd/add`,
`cmd/add --batch`, `cmd/edit rename`/`merge`) takes a corpus-wide exclusive
`flock(2)` on `.winze.lock` around its whole read→gate→commit section, so
shared-tree writers serialize instead of racing. The lock is per-open-file-
description, so the kernel releases it if a holder crashes — no stale-lock
recovery. Uncontended cost is a sub-microsecond syscall, so the single-writer
path is unaffected.

The false-revert case is the one that matters most when the author is an
**agent**: an agent takes a build error at face value and immediately tries to
"fix" its claim — so a phantom error from another session's in-flight write
sends it into a repair spiral on code that was already correct. Serialization
guarantees any gate failure is attributable to the writer's own change, which
is also what makes "revert on gate fail" a *sound* policy again.

So there are two safe multi-session shapes: worktree-per-session (no
contention, needs `git worktree` setup) and shared-tree-with-lock (serialized
writes, zero setup). Pick by whether write concurrency is high enough that
serialization latency bites.

## Burst writes: batch the gate (`cmd/add --batch`)

The gate (91 ms) dominates; appending (2.5 ms) is noise. A session that learned
several things in one turn should commit them under **one** gate, not one gate
per claim. `cmd/add --batch <file.jsonl>` does this — K claims across their
target files, a single `go build . && go vet .`, all-or-nothing revert.

Measured, 5-claim burst against the real corpus + real gate:

| path | latency | |
|---|---|---|
| 5 sequential `winze-add` (gate ×5) | **621 ms** | |
| 1 `winze-add --batch` of 5 (gate ×1) | **119 ms** | **5.2× faster** |

Input is JSONL (the corpus's native log shape), one claim per line:

```jsonl
{"to":"session_a1b2.go","name":"C1","predicate":"Accepts","subject":"MichaelShermer","object":"ShermerPatternicityFraming","provenance_var":"apopheniaSource"}
{"to":"session_a1b2.go","name":"C2","predicate":"Proposes","subject":"KlausConrad","object":"ConradApopheniaClinicalFraming","quote":"exact source text","origin":"Wikipedia / Apophenia"}
```

`--batch -` reads from stdin. `--dry-run` renders without touching files or
running the gate. Every field mirrors the single-claim flags.

## The one real cost: eventual consistency on the entity namespace

Per-session writes trade one guarantee for throughput. **Referential integrity
stays strong** — the gate never lets a dangling reference through. But entity
*identity* goes **eventual**: two sessions can independently coin the same
concept.

- **Same var name, same concept** (`var Apophenia` in two session files) ->
  the package won't compile at merge -> loud, forced resolution. The compiler
  turns Obsidian's silent duplicate-stub problem into a build error you cannot
  ignore.
- **Different name, same concept** (`KlausConrad` vs `KonradKlaus`) -> builds
  fine -> `rot-probe`'s `duplicate` detector catches it on the next pass.
- Different name, different concept -> fine.

So the architecture is **log-structured (LSM-style)**: fast uncoordinated
appends to per-session write-ahead files (hot path, ~119 ms for a burst, zero
cross-session locking) plus a periodic **compaction** pass that folds session
logs into curated topic files and dedups (cold path, coherence).

Compaction is not new machinery — it is the existing maintenance tools:

- `rot-probe` — find duplicate / drifted entities (the compaction *lens*)
- `winze-edit merge` — fold two entities into one (the compaction *primitive*, built)
- `winze-edit rename` — referential cleanup (built)
- `metabolism --dream` — Brief quality, file balance

This is the real mandate for the `merge` tool: it is the compaction half of a
log-structured multi-session KB, not an occasional convenience for scattered
`rot-probe` hits.

## Honest limits

- **Worktree-per-session is mandatory** for the gate-isolation math. Without
  it you are back to a shared-tree gate race.
- **Compaction must actually run**, or session files accumulate and the entity
  namespace fragments — the write-ahead log grows unbounded without a
  compactor. `dream` + `merge` become operationally required, not polish.
- **Read-before-write is the cheap discipline** that keeps compaction light: a
  24 ms query to reuse an existing entity avoids coining a duplicate. Worth
  making a convention in the session write protocol.

## What needs zero new machinery to start

Session files are a naming convention over the write path that already exists;
`--batch` is the one hot-path win and it is built. What the shape makes
load-bearing is the compaction half — the `merge` work, now with a real reason
to exist.
