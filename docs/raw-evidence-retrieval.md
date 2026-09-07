# Raw-evidence retrieval: `winze_recall_raw` and `--raw`

Every memory system this project has checked — winze included — retrieves
over something authored first: promoted, extracted, or hand-written. Nobody
does automatic, derived retrieval over raw session history with no authoring
step (see `docs/sota-memory-systems-survey-2026-08-31.md`'s comparison
table). This is winze's first move toward closing that gap, scoped narrowly
on purpose: deterministic keyword search over evidence that already existed,
returning source text, never a claim.

## What already existed

`cmd/agent/rawlog.go`'s `appendRawLog` has, since Phase 3a, written every
`winze_remember`/`winze_update` call's raw note text — verbatim, timestamped,
tagged with the tool and var name — to `raw.jsonl` beside `memory.go`. It
writes *before* dedup, before the onsetter check, before the typed write even
attempts. A note that gets dedup-blocked, fails the build gate, or lands in a
`Brief` a later truncation cuts short is not gone; it was already one grep
away in this file. See `docs/agent.md`'s `winze_remember` section for the
full rationale.

What was missing: nothing could *query* `raw.jsonl`. It was a write-only
recovery log — useful to a human running `grep`, invisible to an agent.

## What's new

- **`winze-query --raw <query> <dir>`** (`cmd/query/rawfulltext.go`): BM25
  keyword search over `raw.jsonl`'s `Note` field, reusing the same engine
  `--fulltext`/`--hybrid` already use (`cmd/query/fulltext.go`'s
  `ftIndex`/`search`/`tokenize`) — a raw log entry is a third document
  `kind` alongside `"entity"` and `"provenance"`, with zero changes to the
  existing entity/provenance path. Runs before the typed corpus index is
  built (like `--docs-recall`), since raw-evidence retrieval never depends on
  whether the corpus itself compiles.
- **`winze_recall_raw(query, limit?)`** (`cmd/agent/recall_raw.go`): the MCP
  surface, shelling out to `winze-query --raw` the same way `winze_recall`
  shells out to `--hybrid`. Returns `{time, tool, var, note, score}` per hit —
  verbatim text with a timestamp, not an entity headline and brief.

## Why a new tool, not a new mode of `winze_recall`

`winze_recall` returns curated entity headlines and briefs — a typed claim's
public face. A raw hit is a different object entirely: what was actually
said, unedited, with no claim made about it. Conflating the two risks
changing `winze_recall`'s measured, working contract by accident. Keeping
them separate keeps both surfaces — and both self-recall numbers — honest and
independently measurable.

## Why this doesn't conflict with mirror-source-commitments

`CLAUDE.md`'s mirror-source-commitments discipline governs what may be
*encoded* as a claim: only what a source explicitly commits to. A raw-tier
hit is never encoded as anything — it is returned as source text on a query
and nothing more. This was the one open risk named in
`docs/memory-first-repositioning-2026-09-07.md`'s first open question: winze
already reverted an automated writer that promoted speculative hits straight
into fabricated `Provenance.Quote` claims
(`feedback_trip_promotion_fabrication`). This tier avoids that failure by
construction: it sits *below* the claim graph, not into it, so a raw hit is
never encoded as anything a later reader could mistake for a sourced claim.
Eywa's framing (arXiv 2605.30771) — "extraction is an index, not the
memory" — and a private sibling project's independent rejection of a
harvest-into-winze feature both land on the same resolution. It's a new
object class, not a new way to populate the existing one.

## Honest limits

- **One shared `raw.jsonl` per store**, same as today — no per-session
  partition. `docs/agent-identity-integration.md`'s `session_<nick>.go` shape
  would need to land first; not this plan's job to unblock.
- **BM25 keyword only.** Eywa's read path fuses four channels (vector, BM25,
  temporal, entity-graph) with weighted reciprocal rank fusion. Building all
  four for a tier that didn't exist in any form yet was over-scoped for a
  first cut — BM25 is deterministic (no LLM, no embedding call), reuses
  machinery already in this repo, and is enough to measure whether raw-tier
  retrieval moves the self-recall number at all. Semantic, temporal, and
  entity-graph channels remain candidate next milestones.
- **Not build-gated, not typed, by design.** There is nothing to type-check —
  a raw hit carries no relationship to any other entity, on purpose.

## The number

Measured 2026-09-07 against the same self-recall harness that produced the
57%/47.5% later-probe hit rates already in `README.md`'s Known-problems
section (`TestSelfRecallDecaysWithCorpusGrowth`, `cmd/longmemeval`,
`WINZE_SELFRECALL_N=150`, one-note-per-session, 150 real transcript
sessions): the raw tier recalled 41/125 (32.8%, mean rank 5.85) on the later
probe, against 51/125 (40.8%, mean rank 5.35) for the existing typed-store
hybrid search over the same sessions and questions in the same run.

**It doesn't close the gap — not yet, and the reason is legible from the
number itself.** `winze_recall` fuses BM25 and semantic (embedding) search
with reciprocal rank fusion; this first cut of the raw tier is BM25 only,
exactly as scoped above. Hybrid retrieval beat keyword-only retrieval on this
corpus regardless of which tier it ran over, which makes this a measurement
of retrieval method, not of typed claim versus raw text. Eywa's own
architecture fuses four channels for exactly this reason. The honest
milestone-2 candidate this result names directly: add semantic ranking to
`buildRawFTIndex`'s output (reusing `cmd/query/hybrid.go`'s existing
`semanticRank`/`rrfFuse`, the same machinery `--hybrid` already uses) before
drawing any conclusion about whether removing the authoring step helps or
hurts recall on its own merits.

## See also

- `docs/agent.md` — the four (now five) `winze-agent` tools.
- `docs/query.md` — the full `winze-query` command list.
- `docs/memory-first-repositioning-2026-09-07.md` — the repositioning memo
  this plan closes the roadmap item from (local, untracked; not linked
  publicly since it also discusses a private sibling project).
- `docs/sota-memory-systems-survey-2026-08-31.md` — the field comparison that
  first named this gap.
