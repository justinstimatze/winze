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

   **Measured 2026-08-03 (`longmemeval --probe`, all 500 questions, oracle set,
   no API):** one candidate cause of extraction misses — the lens never seeing
   the evidence because `renderSession` truncates it — is *mostly ruled out*. Of
   896 gold-answer turns, truncation drops only **8 (in 8/500 questions, 1.6%)**;
   the per-session cap fires on 253 answer-bearing sessions but the answer
   survives in 245 of them (the head-keeping bet holds 97% of the time). So
   extraction recall is predominantly a **lens** matter, not a truncation one —
   the confidence bar is about lens scoping, not evidence delivery. The 8 are a
   bounded, separable substrate bug: `renderSession` does a blind positional
   `s[:16000]` cut while its comment claims to keep user turns and shed long
   assistant monologues, so a role-aware render would recover most of them (they
   skew to late-in-session facts: 3 single-session-preference, 3 multi-session,
   2 temporal-reasoning). Re-run: `cmd/longmemeval/fetch-data.sh` then
   `longmemeval --probe --dataset data/longmemeval_oracle.json --work /tmp/x
   -temporal 1000 -knowledge 1000 -single 1000 -multi 1000 -assistant 1000
   -preference 1000`.

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
     stops `./...` from reaching the sibling tooling. **Measured 2026-08-03**
     (min-of-6, full `db.Sync`): whole-module 2580 ms vs corpus-scoped 925 ms —
     **2.79× faster, 36% of the work**, better than the conservatively-projected
     half. (Host load ~6–7, so absolutes are contention-inflated; the ratio
     cancels the shared tax and is the robust figure — see
     `docs/defn-migration.md`.)
   - **The defn scoped-sync feature shipped but is moot here.** defn v0.26.0 added
     `db.SyncPattern(projectDir, pattern)`, but post-move `SyncPattern(repo, ".")`
     errors — the module root no longer holds a package to scope to. The
     subdirectory move already delivers the corpus-proportional ingest this
     precondition required; a DefIDs IN-predicate would further scope the
     DB-insert half *within* the corpus only for very large stores, which is
     optimization, not a gate.

   With ingest now corpus-scoped, a cold woken agent's first read is bounded by
   corpus-only ingest, not whole-module. This precondition is substantially
   cleared; remaining perf headroom (the defn ask, incremental sync) is
   optimization, not a blocker.

**First live measurement, 2026-08-03 → 2026-08-29 (`0/8 → 8/8`, one session, not
a green light on its own):** the write path this precondition actually gates —
an agent writes its own session-end context via `winze_remember`, a genuinely
cold session (a fresh subagent, zero inherited context) recalls it later via
`winze_recall`, scored against 8 questions with ground truth locked before the
write — had never been run end to end until today. First pass: **0/8**
correct with `winze_recall`'s then-defaults. Root cause was mechanical, not
epistemic: 3/8 misses were the right entity ranked #1 but truncated below the
answer (`brief_chars` defaulted to 240, with a bare `…` giving no signal that
more existed); 4/8 misses were a single multi-topic session-summary entity
losing BM25/semantic rank to small, unrelated-but-lexically-adjacent existing
entities on narrow queries; 1/8 was a stale-but-correct partial hit from the
same dilution effect. Fixed both causes (`recallDefaultBriefChars` 240→500
plus an actionable truncation hint; split the session note into 7 atomic
entities) and re-ran the identical 8 questions on a second fresh subagent:
**8/8**, all high confidence. See `docs/agent.md`'s `winze_recall` entry.

This is a first data point, not the confidence bar itself: one session,
self-authored ground truth, one fresh-subagent trial, no multi-day gap (the
"cold" session was seconds later, not days), and no test under real corpus
growth or concurrent writers. What it does establish: the failure mode found
was retrieval ergonomics and write hygiene, not a defect in the
typed-extraction model itself — and both are now fixed defaults/discipline
rather than open questions.

**Status: precondition 2 (perf) substantially cleared; precondition 1
(memory-confidence) has a first positive result, not yet a cleared gate.** The
design is fixed enough that the pointer question is answered and won't churn.
With ingest now corpus-scoped, the wake-time read is viable; precondition 1
now has real evidence in its favor (see above) rather than none, but a single
same-day N=1 trial is not the bar this section originally called for — a
multi-session, multi-day version of the same test (real elapsed time, real
corpus growth in between) is the next step before treating this as cleared.

### Measured 2026-09-02: retrieval does not decay; the write gate does

`TestSelfRecallDecaysWithCorpusGrowth` (`cmd/longmemeval`) is the multi-session,
multi-day version this section asked for. It replays 20 real sessions
stratified across 104 days of the `~/Documents` project into a scratch store,
oldest-first, so each note is written into a store holding every earlier note
and none of the later ones. Then it probes each note with its own session title
once the store is at full size. No answerer and no judge: rank is deterministic,
and putting an LLM between the write and the number would cost money to find a
ranking bug a sort can show.

**Retrieval: 18/18 stored notes came back at rank 1. Mean rank 1.00, zero
misses.** The oldest note had 19 later writes stacked on top of it and still
ranked first. Corpus growth was the mechanism this section worried about, and
on this corpus it does not bite.

**The write gate does. 2 of 20 writes (10%) were refused before storage**, at
cosine 0.74 and 0.73:

| refused | blocked against | days apart |
|---|---|---|
| "Recover Claude sessions from crashed machine" | "Resume lost Claude code sessions from transcripts" | 35 |
| "RAM and load investigation" | "Investigate RAM and disk space usage" | 59 |

Neither pair is a duplicate. They are the same *topic* recurring weeks apart —
two separate crashes, two separate RAM investigations. Dedup cannot separate
recurrence from duplication, and for session notes recurrence is the normal
case, not the exception. The cost is worse than a dropped write: the second
incident vanishes and the first stands as though it were the only time, so a
later reconstruction is wrong rather than merely thin. This is the "write
hygiene" the N=1 trial named, now with a rate and a mechanism.

**The raw tier caught both.** `appendRawLog` runs on the fourth line of
`handleRemember`, before the dedup gate, and the replay confirms it: 20 raw
entries for 20 attempted writes, both refusals recoverable. This is the first
time Phase 3a has been exercised at all — before this replay `raw.jsonl` had
never captured a single entry in production, because nothing had called
`winze_remember` since it shipped. Its design claim held on its first real test.

**What this does not establish.** Rank is findability, not sufficiency: nothing
here checks whether a recalled note *says enough* to reconstruct the session.
That needs the answerer and the judge, and is worth paying for now that the
cheap question is answered. The corpus is also one project's sessions on one
host, and the notes are title-plus-opening-ask rather than what an agent would
compose at a session's end.

## See also

- `docs/multi-session-write-shape.md` — the shared-brain write shape this reuses.
- `docs/defn-migration.md` — the ingest-cost measurements and the gap that
  gates precondition 2.
