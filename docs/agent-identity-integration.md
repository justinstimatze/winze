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
   - **The cost is the full re-ingest on any write.** `defndb.ensureFresh` calls
     `db.Sync` — a full `packages.Load("./...")` re-type-check of the whole module
     — whenever *any* `.go` file has changed, and discards the stale-file set it
     already computed. Cost is roughly linear in total corpus size: ~2 s at the
     current ~12k lines, ~45 s at a 250k-line / 30k-claim store. A woken agent on
     a **cold** store (never ingested, or invalidated by another session's write)
     pays that before its first read.
   - **The lever is only half winze's to pull.** The scope-able-ingest ask filed
     with defn (scoped sync + a DefIDs IN-predicate, dispatch msg-438f91df, still
     pending) scopes the *DB insert* half. But the corpus is **one Go package by
     design** (`docs/multi-session-write-shape.md` — that is what makes
     cross-file references type-check for free), and Go's compilation unit is the
     package, so the *type-check* half cannot be scoped below the whole package
     without partitioning the corpus into multiple packages — which would forfeit
     the free cross-reference integrity the whole shape rests on. Which half
     dominates the re-ingest is unmeasured (blocked on an idle box); that
     measurement is the first concrete perf task, because it decides whether the
     defn ask alone is sufficient or whether the single-package design itself is
     the scaling ceiling.

   Until the re-ingest cost is characterized and reduced, a cold woken agent's
   first read is bounded by full-corpus ingest, and the integration waits on it.

**Do not build until both clear.** The design is fixed enough that the pointer
question is answered and won't churn; the implementation waits on perf and
memory-confidence, in that dependency order — perf first, because without scoped
ingest the wake-time read is not viable at all.

## See also

- `docs/multi-session-write-shape.md` — the shared-brain write shape this reuses.
- `docs/defn-migration.md` — the ingest-cost measurements and the gap that
  gates precondition 2.
