# Raw-evidence retrieval: `winze_recall_raw` and `--raw`

Every memory system this project has checked — winze included — retrieves
over something authored first: promoted, extracted, or hand-written. Nobody
does automatic, derived retrieval over raw session history with no authoring
step (see `docs/sota-memory-systems-survey-2026-08-31.md`'s comparison
table). This is winze's move toward closing that gap: retrieval over
evidence that already existed, returning source text, never a claim.

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

- **`winze-query --raw <query> <dir>`** (`cmd/query/rawfulltext.go`,
  `cmd/query/rawhybrid.go`): hybrid BM25 + semantic search over `raw.jsonl`'s
  `Note` field, fused by reciprocal rank fusion — the same pipeline
  `--hybrid` uses over the typed corpus, reused rather than reimplemented.
  `cmd/query/fulltext.go`'s `ftIndex`/`search` (BM25) and
  `cmd/query/semantic.go`'s `embedSegments`/`vecCache` (embeddings, cached by
  content hash) are both fully generic over an opaque document index; a raw
  log entry slots in as a document exactly the way an entity or a provenance
  record already does, with zero changes to either path.
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
  would need to land first; not this doc's job to unblock.
- **Not build-gated, not typed, by design.** There is nothing to type-check —
  a raw hit carries no relationship to any other entity, on purpose.
- **No longer fully deterministic.** Milestone 1 shipped BM25-only
  specifically because it needed no LLM call and no embedding call. Adding
  the semantic channel (below) trades that purity for recall — the same
  trade `--hybrid` already made for the typed corpus, extended to a second
  surface. `--raw` now depends on a local ollama instance the way `--hybrid`
  and `--semantic` already do, and fails hard rather than silently degrading
  to BM25-only if it's unreachable.
- **Temporal and entity-graph channels remain unbuilt.** Eywa's read path
  fuses four channels (vector, BM25, temporal, entity-graph); this tier now
  has two of the four.

## The number

Measured against the same self-recall harness that produced the 57%/47.5%
later-probe hit rates already in `README.md`'s Known-problems section
(`TestSelfRecallDecaysWithCorpusGrowth`, `cmd/longmemeval`,
`WINZE_SELFRECALL_N=150`, one-note-per-session, 150 real transcript
sessions), across two runs on the identical sessions and questions:

| | later-probe hit rate | mean rank (found) |
|---|---|---|
| Raw tier, BM25 only (2026-09-07) | 41/125 — 32.8% | 5.85 |
| Raw tier, BM25 + semantic (2026-09-07) | 50/125 — 40.0% | 5.16 |
| Typed store, BM25 + semantic (2026-09-07) | 51/125 — 40.8% | 5.35 |

Adding the semantic channel closed nearly the entire gap: 32.8% → 40.0%, a
7.2-point jump from one change. Against the typed store's 40.8%, the
remaining 0.8-point gap (one question out of 125) is well inside binomial
noise at this sample size — on this measure, at this scale, raw-evidence
retrieval and the typed claim graph now perform indistinguishably. The raw
tier's mean rank among the questions it *did* surface (5.16) is marginally
better than the typed store's (5.35).

**What this does and doesn't settle.** It resolves the retrieval-method
confound the first number had: once both tiers use the same fusion
mechanism, object class (typed claim vs. raw text) stops being the
explanation for the gap, because there mostly isn't one left. It does not
yet settle whether raw retrieval is *sufficient on its own* — this was
measured with the raw tier's per-line documents (one `winze_remember` note
per line), not against a larger, less-curated raw corpus, and the temporal
and entity-graph channels Eywa's own architecture also uses are still
missing from both tiers equally. Worth an independent read before leaning on
this number for anything beyond "the authoring step is not obviously
buying accuracy that the retrieval mechanism doesn't already provide."

## See also

- `docs/agent.md` — the `winze-agent` tools.
- `docs/query.md` — the full `winze-query` command list.
- `docs/memory-first-repositioning-2026-09-07.md` — the repositioning memo
  this closes the roadmap item from (local, untracked; not linked publicly
  since it also discusses a private sibling project).
- `docs/sota-memory-systems-survey-2026-08-31.md` — the field comparison that
  first named this gap.
