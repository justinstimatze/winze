# Raw-evidence retrieval: `winze_recall_raw` and `--raw`

**Retired as of phase 3 below.** `winze_recall_raw`, `--raw`, and
`raw.jsonl` no longer exist — everything through "What's new" describes a
mechanism that shipped and was later deleted once `Documented` claims made
it redundant. Kept as the historical record the retirement was based on.

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
  `cmd/query/rawhybrid.go`, `cmd/query/rawtemporal.go`): three-channel
  hybrid search over `raw.jsonl`'s `Note`/`Time` fields — BM25, semantic
  (embedding), and temporal (date/relative-date matching) — fused by
  reciprocal rank fusion. BM25 and semantic reuse the same pipeline
  `--hybrid` uses over the typed corpus rather than reimplementing it:
  `cmd/query/fulltext.go`'s `ftIndex`/`search` and `cmd/query/semantic.go`'s
  `embedSegments`/`vecCache` are both fully generic over an opaque document
  index, so a raw log entry slots in as a document exactly the way an entity
  or a provenance record already does. The temporal channel
  (`parseTemporalRange`/`temporalRank`) is new, pure, deterministic logic —
  it recognizes a small set of date expressions (`today`, `yesterday`, `N
  days/weeks/months ago`, `last/this week/month`, an explicit ISO date) and
  ranks in-window docs by recency; a query with no temporal language leaves
  the channel silent rather than injecting a recency bias. Fusion for the
  raw tier is a dedicated `rrfFuseRaw` (`cmd/query/rawhybrid.go`), not
  `--hybrid`'s own `rrfFuse` — kept separate so `--hybrid`'s already-shipped
  display and JSON code never has to change shape for a channel it doesn't
  have.
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
  the semantic channel (milestone 2) trades that purity for recall — the
  same trade `--hybrid` already made for the typed corpus, extended to a
  second surface. `--raw` now depends on a local ollama instance the way
  `--hybrid` and `--semantic` already do, and fails hard rather than
  silently degrading to BM25-only if it's unreachable. The temporal channel
  (milestone 3) stays at zero marginal dependency: pure stdlib string/time
  logic.
- **Entity-graph channel remains unbuilt, and not by oversight.** Eywa's
  read path fuses four channels (vector, BM25, temporal, entity-graph); this
  tier now has three of the four. Entity-graph was investigated for
  milestone 3 and deliberately deferred: the only field that could seed a
  graph walk over raw docs is `rawDoc.Var`, and `winze_remember` (the tool
  every self-recall-harness write goes through) always logs it empty — the
  var doesn't exist yet at the point the raw entry is written, by the same
  before-typing invariant described above.
- **Milestone 4 fixed half of that blocker, not both.** Claim edges between
  memory entities only ever existed if `winze_link` was actually called, and
  `handleRemember`'s `suggestLinks` only ever *suggests* a call for a
  human/agent to issue deliberately — it never auto-links, so the harness
  had zero edges to walk. Milestone 4 gave the harness real ones: a
  post-write pass mirrors production's own link-suggestion mechanism
  (`checkDedup` → `nearestMemories`, `cmd/agent/mcp.go` — cosine via
  `--semantic`, not `--hybrid`) and calls `winze_link` for real, at
  production's real 0.45 floor. Measured at N=150: 142 claim edges across
  150 sessions. That unblocks an entity-graph channel for the **typed
  store's** `--hybrid` — real edges now exist to walk. It does *not* unblock
  one for **this raw tier**: `rawDoc.Var` is still always empty from
  `winze_remember`, so a raw-tier entity-graph channel would need to resolve
  entity mentions from `.Note` text directly rather than from `.Var`, a
  different, still-unbuilt design. That's milestone 5's scope, not this
  one's.

## The number

Measured against the same self-recall harness that produced the 57%/47.5%
later-probe hit rates already in `README.md`'s Known-problems section
(`TestSelfRecallDecaysWithCorpusGrowth`, `cmd/longmemeval`,
`WINZE_SELFRECALL_N=150`, one-note-per-session, 150 real transcript
sessions), across three runs on the identical sessions and questions:

| | later-probe hit rate | mean rank (found) |
|---|---|---|
| Raw tier, BM25 only (2026-09-07) | 41/125 — 32.8% | 5.85 |
| Raw tier, BM25 + semantic (2026-09-07) | 50/125 — 40.0% | 5.16 |
| Raw tier, BM25 + semantic + temporal (2026-09-07) | 50/125 — 40.0% | 5.16 |
| Typed store, BM25 + semantic (2026-09-07) | 51/125 — 40.8% | 5.35 |

Adding the semantic channel closed nearly the entire gap: 32.8% → 40.0%, a
7.2-point jump from one change. Against the typed store's 40.8%, the
remaining 0.8-point gap (one question out of 125) is well inside binomial
noise at this sample size — on this measure, at this scale, raw-evidence
retrieval and the typed claim graph now perform indistinguishably. The raw
tier's mean rank among the questions it *did* surface (5.16) is marginally
better than the typed store's (5.35).

**Adding the temporal channel moved nothing — same 50/125, same 5.16 mean
rank, to the decimal.** Not a small move; no move. This is worth taking at
face value rather than writing around: the typed-store row above, untouched
by this change, reproduced its own milestone-2 figure exactly (51/125,
mean rank 5.35), which rules out the harness itself drifting between runs —
the raw tier's flat result holds up as a genuine null. The smoke test in
`cmd/query/rawtemporal_test.go` and a manual run against the live
`winze-memory` store both confirm `parseTemporalRange`/`temporalRank` fire
correctly and rank in-window docs by recency when a query does carry
recognized date language, so the mechanism isn't the problem. The language
it looks for is what's scarce in this specific probe set: a rough incidence check (`rg` across the transcript corpus for
the same phrases the parser recognizes) found them common across full
transcripts generally, but `LaterAsk` holds back exactly one specific
mid-session turn per session — the odds that phrase-bearing lines and the
one held-out turn coincide, across 125 probes, are apparently low enough to
land at zero this run. That's an inference from the incidence check, not a
per-probe count — worth an independent look before trusting the exact
odds, but not before trusting the null result itself, which the identical
typed-store control makes solid.

**What this does and doesn't settle.** It resolves the retrieval-method
confound the first number had: once both tiers use the same fusion
mechanism, object class (typed claim vs. raw text) stops being the
explanation for the gap, because there mostly isn't one left. It does not
yet settle whether raw retrieval is *sufficient on its own* — this was
measured with the raw tier's per-line documents (one `winze_remember` note
per line), not against a larger, less-curated raw corpus, and the
entity-graph channel Eywa's own architecture also uses is still missing
from both tiers equally (see Honest limits above for why). Nor does the
temporal channel's null result here settle whether it's worth having: this
harness's probe shape (one held-out content question per session) is a poor
fixture for a channel built for date-anchored queries ("what did we discuss
last week") — a fixture that actually asked date-anchored questions would
be a fairer test, and doesn't exist yet. Worth an independent read before
leaning on any of these numbers for more than "the authoring step is not
obviously buying accuracy that the retrieval mechanism doesn't already
provide."

## Retiring this tier (in progress)

A consult and a direct acid test (2026-09-07) settled a question this doc
left open — whether the raw tier earns a permanent place, or is standing in
for a gap the typed store could close at the source. The acid test came
down on the second answer: two of three `raw.jsonl` lines behind one
existing memory
(`DedupBlocksRecurrenceNotJustDuplication` in `~/Documents/winze-memory`) —
two earlier `winze_remember` attempts at the same finding, reworded — exist
nowhere in the typed store, confirmed by direct `grep`. Only the third
attempt, a `winze_update`, survived, and only because it happened to become
the entity's current `Brief`. `raw.jsonl` was the only place either lost
attempt's exact wording still existed.

**Phase 1 (shipped): make the write path structurally lose nothing.** A new
`Documented` unary claim (`corpus/predicates.go`) carries an entity's exact
source text as real `Provenance` — `Quote`, `Origin`, `IngestedAt` — not
just a `Brief` summary. `winze_remember` attaches one on every successful
write; `winze_update` snapshots the outgoing `Brief` into one before
overwriting it; and — the fix that actually closes the gap the acid test
found — a dedup-blocked write's text is now attached as a `Documented` claim
on the entity it collided with, instead of being refused and discarded. All
three verified live. A follow-on fix landed the same day: the dedup-block
path called `execDocument` but never `gitCommitMemory`, so a blocked write's
`Documented` claim sat as an uncommitted file change until some unrelated
later write happened to sweep it up — found while verifying phase 2, fixed
by committing after a successful block-path attach the same way the
success path already does.

**Phase 2 (shipped): backfill existing content.** A throwaway migration
script — never committed, deleted after use, per CLAUDE.md's "migration is
throwaway work" — closed the gap for the one real store that had it: two
exact database joins (an `update`-tool `raw.jsonl` line already carries its
resolved var; a `remember`-tool line's exact text matches some entity's
*current* `Brief`, since `winze-add` stores it verbatim) accounted for
almost everything, and the small genuine residue — text matching no current
`Brief` at all, meaning it was rejected before anything recorded
rejections — was replayed through the real `winze_remember` tool so phase
1's own logic decided its fate. Verified against the real store: both
`winze_remember` attempts behind `DedupBlocksRecurrenceNotJustDuplication`
this doc's acid test found are now retrievable via `--fulltext`/`--hybrid`
as real `Documented` claims — one via the exact-`Brief` join, one via the
live dedup-block-attach path firing for real during replay. A real, older
store's schema copy had also drifted from the canonical `predicates.go` and
carried a deliberate local `RelatesTo`/`Supersedes` divergence
(`memory_predicates.go`) — a wholesale schema refresh would have silently
collided with it; `Documented` was added as a surgical, additive insertion
into the existing file instead, once `cmd/add`'s any-role resolution
(`anyRoleSlots`, which only ever reads `predicates.go`, never other files)
made clear it couldn't live anywhere else either.

**Phase 3 (shipped): delete, not demote.** A repo-wide sweep (via `defn`'s
call-graph search, not assumption) confirmed `cmd/meld`, `cmd/memtool`,
`cmd/observatory`, `cmd/benchmark`, `cmd/mcp`, and `cmd/edit` had zero
raw-tier references — only `cmd/agent/mcp.go` (the two `appendRawLog` call
sites, the `winze_recall_raw` tool registration), `cmd/agent/call.go` (its
CLI dispatch case), and `cmd/query/main.go` (the `--raw` flag) did. All
three were edited rather than deleted wholesale; `appendRawLog`,
`rawLogPath`, `rawLogEntry`, `handleRecallRaw`, `buildRawFTIndex`,
`loadRawDocs`, `runRawHybrid`, `rrfFuseRaw`, `semanticRankRaw`,
`parseTemporalRange`, and `temporalRank` — and the files that existed only
to hold them (`cmd/agent/rawlog.go`, `cmd/agent/recall_raw.go`,
`cmd/query/rawfulltext.go`/`rawhybrid.go`/`rawtemporal.go`) — are gone. One
real dependency the phase-2 plan hadn't anticipated: the self-recall
benchmark harness (`cmd/longmemeval/selfrecall_corpus_test.go`) ran every
probe against `winze_recall_raw` as a second, comparable channel — the
mechanism behind the 40.0%/40.8% comparison below. Once the tool is gone,
calling it fails the test outright — there's no graceful skip — so
`probeAll` was simplified back to a single-tier probe and
`assertRawTier`/`rawRankOf`/
`bestRawRankOf`/`rawHits` were deleted outright rather than stubbed —
no live failure mode was left to justify keeping a comparison against a
retrieval path that no longer exists. The one real store's `raw.jsonl`
(`~/Documents/winze-memory`, 13 lines) was deleted after two separate
runs of the same check — every line's note text against every current
`Brief` and every `Documented` claim's `Quote` — both came back 13/13
covered, zero missing.

**Not yet built:**
- Fix `winze_recall`'s `score: 0` bug (README's Known problems) — still
  open, unrelated to this tier's retirement.

## See also

- `docs/agent.md` — the `winze-agent` tools.
- `docs/query.md` — the full `winze-query` command list.
- `docs/memory-first-repositioning-2026-09-07.md` — the repositioning memo
  this closes the roadmap item from (local, untracked; not linked publicly
  since it also discusses a private sibling project).
- `docs/sota-memory-systems-survey-2026-08-31.md` — the field comparison that
  first named this gap.
