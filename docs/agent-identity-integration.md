# Agent-identity integration: winze as the woken agent's memory

**Status: proposed, gated. Do not implement yet.** This design answers one
architectural question now — the one that is cheap to decide today and annoying
to migrate later — and records the preconditions that must clear before any of
it gets built. See [Gating](#gating).

## The seam

mcp-dispatch shipped a lifecycle supervisor (`dispatch-supervise`): when mail
arrives for a durable nick with no live session, it cold-starts that agent's
runtime from an operator allowlist. That closes the "durable identity but nobody
answers" gap and opens a new one — **a woken agent wakes cold.** It inherits its
unread mail and nothing else: not what it was doing, not its standing
commitments, not who it works with.

dispatch's backlog had "design the per-agent memory file" for this. It is
retargeted to winze rather than built as a competing store, because winze
already is the shared-brain shape (`docs/multi-session-write-shape.md`) and
already exposes claims / provenance / disputes over MCP. dispatch stays the
identity and transport layer; winze is the memory.

## What dispatch owns, and what it deliberately does not

dispatch keeps a durable nick registry at `{relay}/.agents/<nick>.json`, one
record per teammate:

| field | meaning |
|---|---|
| `nick` | the id with the pid suffix stripped — **stable across restarts** |
| `first_seen` / `last_seen` | operational timestamps |
| `sessions` / `last_session_id` | run bookkeeping |
| `channels` | standing channel subscriptions |

Two properties matter for the join: the nick is **stable across restart**, and
it is the **same identity the git transport uses cross-host**, so anything keyed
on it survives a machine change. The registry is explicitly *operational
bookkeeping, not knowledge* — dispatch does not want to hold the agent's
context, only its address.

## The decision: where does the pointer live?

The open question was framed as "does the nick→context pointer live in the
dispatch registry or in winze?" The useful move is to drop the assumption that a
pointer must be *stored* at all.

**The nick is already the join key. Both sides hold it for free.** dispatch
knows a woken agent's nick (it is the agent's identity). winze can store that
agent's context as an entity whose stable slug *is* the nick. Then:

- dispatch adds **no** `winze_ref` field — its registry stays pure operational
  bookkeeping, exactly as it wants.
- winze does **not** key on a dispatch-internal pointer — it keys on the nick,
  which is a legitimate stable public identity, not operational leakage.
- Resolution at session start is one winze lookup: *"recall the Agent entity for
  nick X."* No pointer to keep in sync, nothing to migrate if either registry
  changes shape.

So: **neither registry stores a pointer. The nick is the shared key, resolved by
a winze query at wake time.** This is the answer to dispatch's open question, and
it is the cheap-now / costly-later call worth making today.

## What winze stores per agent

A typed `Agent` entity, keyed by nick, plus the claims that agent authored. This
is not new schema machinery — it is the existing entity/claim/provenance model
with `Provenance.IngestedBy = <nick>` doing the authorship join that already
exists on every claim.

A woken session reconstructs "who am I and what was I doing" from three reads:

1. **The `Agent` entity for its nick** — role, standing commitments, who it works
   with. One entity.
2. **Recent claims it authored** — `IngestedBy == nick`, most-recent first. What
   it last concluded.
3. **Open disputes it is party to** — the contested neighborhood it left behind.

All three are ordinary winze queries over the existing model. The write path is
the session write-ahead-log shape already documented: the agent appends to its
own `session_<nick>.go` (or a durable `agent_<nick>.go`), the gate enforces
referential integrity, compaction folds it in later.

## Gating

This integration must not ship until two preconditions clear. They are the same
two the project is already working, which is why this waits rather than jumps the
queue.

1. **Confidence in winze-as-memory.** The memory-store use (winze holding an
   agent's durable context and returning it faithfully) has to be trusted before
   a cold agent relies on it to know who it is. The LongMemEval-S spike measured
   the *retrieval and substrate* side and found them sound (defn retrieval
   1.8–6.75 ms/question, flat under load); the open edge it exposed is
   **extraction recall**, and an agent writing its *own* context back is a
   different, likely easier, write path than an LLM lens extracting from
   third-party chat logs — but it still needs its own confidence bar, not an
   assumption borrowed from the read-side numbers.

2. **The ingest-scoping perf work.** This is the hard gate, and the honest
   shape of it is a *cold-store write-path* cost, not a read cost.
   - **Reads are already solved.** A single-entity lookup — exactly the agent-wake
     "who am I" query — is an indexed **0.14 ms** hit on a warm store
     (`docs/defn-migration.md`), and the scoped read seams (`EachFieldWithValuePrefix`
     / `EachFieldForDefs`) already stream only one target's claims. A woken agent
     reading its own nick's slice off a warm store is free.
   - **The cost was the full-*module* re-ingest on any write.** `defndb.ensureFresh`
     calls `db.Sync` — `packages.Load("./...")` — whenever *any* `.go` file has
     changed. When the corpus lived at the module root, `./...` matched the whole
     module: the ~47-file corpus *plus* all of `cmd/` and `internal/`. Measured
     split (warm, min-of-6): ~1991 ms = type-check 593 ms + DB insert/resolve
     1398 ms, of which ~46% of DB rows and most of the type-check were tooling,
     not corpus — roughly half of every write's ingest spent indexing code the
     queries never read.
   - **This is now scoped, winze-side.** The corpus was moved into a `corpus/`
     subdirectory of the same module (2026-08). Ingest points at `corpus/`, so
     `packages.Load("./...", Dir=corpus/)` matches only the corpus package —
     `cmd/` and `internal/` are excluded from *both* the type-check target and
     the DB insert (`IngestPackages` iterates only the matched packages). The
     corpus stays **one Go package** (`docs/multi-session-write-shape.md`), so the
     free cross-file referential integrity is preserved; the subdirectory simply
     stops `./...` from reaching the sibling tooling. Expected per-write ingest
     drops ~1991 ms → ~850 ms.
   - **The defn ask still helps at scale.** The scope-able-ingest ask filed with
     defn (scoped sync + a DefIDs IN-predicate, dispatch msg-438f91df, still
     pending) would further scope the DB-insert half *within* the corpus for very
     large stores. It is no longer the gating item — the subdirectory move
     delivers the corpus-proportional ingest that this precondition required.

   With ingest now corpus-scoped, a cold woken agent's first read is bounded by
   corpus-only ingest, not whole-module. This precondition is substantially
   cleared; remaining perf headroom (the defn ask, incremental sync) is
   optimization, not a blocker.

**Status: precondition 2 (perf) substantially cleared; precondition 1
(memory-confidence) remains.** The design is fixed enough that the pointer
question is answered and won't churn. With ingest now corpus-scoped, the
wake-time read is viable; what still gates the build is confidence in
winze-as-memory — the extraction-recall edge the LongMemEval-S spike exposed, on
a write path (an agent writing its own context) that is different from and likely
easier than the third-party-log extraction the spike measured, but which still
needs its own confidence bar before a cold agent relies on it to know who it is.

## See also

- `docs/multi-session-write-shape.md` — the shared-brain write shape this reuses.
- `docs/defn-migration.md` — the ingest-cost measurements and the gap that
  gates precondition 2.
