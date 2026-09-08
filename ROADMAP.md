# Roadmap

The metabolism is what curation compounds into once it's run long enough. A resumed thread that can't tell which of its own beliefs have gone stale is confident wrongness with better latency — so dream, trip, and evolve spend idle time precomputing exactly that. When an agent is backed by winze, it can be confidently precise on well-supported topics and naturally cautious on thin ones, without being told which is which. The topology tells it.

- **Dream** (NREM): consolidation without new ingest — bridge entities, file balance, provenance gaps
- **Trip** (REM): speculative cross-cluster connections, scored and promoted. The system surprising itself.
- **Evolve**: topology-driven sensor queries (arXiv, RSS, Wikipedia), quality-gated ingest, calibration
- **Bias audit**: the KB runs its own cognitive bias catalog against its own structure

The output isn't a status report. It's better answers the next time someone asks.

**Current sprint:** Growing the graph around contested theories of consciousness and cognition (IIT, Global Workspace Theory, Free Energy Principle). The KB also contains meta-claims about its own limitations — predictions the system can resolve about itself.

**Shipped:** MCP server (`cmd/mcp`) — any agent queries structured epistemic metadata (what's known, how confident, what's contested) via `mcp__winze__{claims,disputes,provenance,search,stats,theories}` without knowing the infrastructure. Laminar metabolism with phase-level self-gating, hard monthly spend cap, and per-call actual-spend telemetry. OKF export (`cmd/okf`) — see below. Vault ingest (`--pkm`) — point winze at a pile of markdown and its `[[wikilinks]]` become typed, compiled claims; see [docs/vault-ingest.md](docs/vault-ingest.md). Multi-store meld (`cmd/meld`) — bridge two or more winze stores into a single read-only union for cross-store query, then dissolve it; see [docs/meld.md](docs/meld.md). Dated-measurement lint (`cmd/lint`'s `datedMeasurementRule`) — flags a `Brief`, `Rationale`, or `Quote` that reads as a measurement or a live-state claim with no date attached, the exact shape of the stale `internal/defndb` incident below; advisory, since a first run against the real corpus (2026-09-02) mostly caught historical decades in sourced `Quote` text that cannot carry an ISO date without either inventing one or rewriting the citation. Raw-evidence retrieval (`winze_recall_raw`, `winze-query --raw`) — **retired**; kept here as the historical record its retirement was based on. It was hybrid BM25 + semantic + temporal search over `raw.jsonl`, the verbatim, timestamped log every `winze_remember`/`winze_update` call wrote before dedup or typing, returning source text, never a claim. Measured on the same self-recall harness as the typed store, on level retrieval footing: 40.0% later-probe hit rate against the typed store's 40.8% — statistically indistinguishable at this sample size, once BM25-only was replaced with the same BM25+semantic fusion `--hybrid` already uses (an earlier BM25-only cut of this tier measured 32.8%). **Correction, 2026-09-08:** this comparison ran 2026-09-07 between 11:30 and 14:32, hours before that same day's 20:40 result-cap fix (`a49fe0e`) — the same bug already flagged below as invalidating the 57%/47.5% figures. "Indistinguishable" isn't a confirmed post-fix fact; see [docs/raw-evidence-retrieval.md](docs/raw-evidence-retrieval.md)'s own correction. The retirement still stands on the second-surface maintenance cost alone, but raw-plus-rerank was never measured. Adding a third, temporal channel moved this number by exactly zero — the harness's one-content-question-per-session probes rarely carry the date language the channel looks for, an honest null rather than a broken feature. The entity-graph channel Eywa's own architecture also uses was never built: the only harness that could validate it produced no linked or Var-tagged raw entries to walk. **All three retirement phases have shipped:** an acid test found real memories in a live store with `winze_remember` attempts unrecoverable anywhere in the typed store, so a new `Documented` claim (`corpus/predicates.go`) attached an entity's exact source text as real `Provenance` on every successful write, every update's outgoing text before it was overwritten, and a dedup-blocked write's text on the entity it collided with, instead of discarding it (phase 1); a one-time migration closed the gap for existing content too, via two exact database joins plus a replay of the small genuine residue through the real write path (phase 2); and `raw.jsonl`, `winze_recall_raw`, `--raw`, every file that existed only to serve them, and the self-recall harness's raw-tier comparison probe are now deleted (phase 3), leaving exactly one write path and one retrieval path. Verified live against the real store before deletion: both originally-lost attempts surfaced via `--fulltext`, and every one of the one real store's 13 `raw.jsonl` lines was independently recoverable via a current `Brief` or `Documented` `Quote`. See [docs/raw-evidence-retrieval.md](docs/raw-evidence-retrieval.md) for the full history.

**Next — the read side moves onto [defn](https://github.com/justinstimatze/defn).** defn moved off Dolt to SQLite, links in 0.859s, and its query API is a superset of `internal/defndb` — including multi-hop `Traverse` and incremental `StaleFiles`, which winze has no equivalent for. The real work is not swapping the backend but changing the question: `--claims X` currently builds a whole-corpus index to answer about one entity, where defn answers it with one indexed lookup (0.14ms at 30k claims, after a defn-side fix this work produced). Full measurements in [docs/defn-migration.md](docs/defn-migration.md).

**Also next:** Source reputation — per-domain corroborated/challenged/refuted rates computed from the calibration time-series, surfaced as a sensor down-weight (not deny-list). Z3-based lint rules for formal verification of ontological constraints. An OKF `relations:` proposal upstream, and OKF *import* on the untrusted path so other producers' bundles are ingestible as sensor input rather than as fact. A raw-evidence retrieval tier — closing the zero-authoring-step gap named in [README.md](README.md) (nobody in the field does automatic derived retrieval over raw session history with no authoring step) — requires claims to cite into stored evidence by a hash-checked reference rather than copy source text inline, so the tier can't quietly become a second copy of the typed store the way the retired one did; not yet scoped as an implementation plan (see the "Lead with memory, not epistemics" section below).

## Interop: Google's Open Knowledge Format

[OKF v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md) (Google Cloud, June 2026) is a directory of markdown files with YAML frontmatter, and it reached winze's central thesis independently: v0.2 added `generated` / `verified` trust tiers over a `sources` family — a generated-vs-sourced distinction, arrived at from the same premise. Where it parts is enforcement. OKF's conformance floor is one required field, and consumers are forbidden from rejecting unknown types or broken links; it is buying adoption. Winze's floor is `go build`; it is buying integrity.

Those bets compose in one direction: a corpus that satisfies the compiler always projects down to a bundle that satisfies the floor, and never the reverse. So `winze-okf --out` emits a conformant bundle — 322 concepts, zero warnings — and the projection is the argument in a form other tools can read.

Two things survive that the format alone would drop. OKF links carry no relationship kind (the spec puts it in "surrounding prose"), so claims are emitted under `## <Predicate>` headings: conformant, and a consumer that looks recovers the typed edge. And the attribution split holds structurally — a `Provenance`-backed claim contributes a `sources:` entry and a footnote carrying its exact Quote, while a `Conjecture`-backed claim contributes *nothing* to `sources:` and lands in its own `# Conjectures` section. Not by a rule in the exporter: `Conjecture` has no `Quote` field, so there is no value to put there. Trust tiers then fall out rather than being asserted — sourced documents reach machine-confirmed via the build gate, conjecture-only documents stay unverified and draft. [docs/okf.md](docs/okf.md) covers what the projection loses, too.

## Known problems

- **Cross-session self-recall has a real number, and it's improving slowly.** The 57%/47-52% figures previously reported here were a measurement artifact — a hardcoded result cap was hiding true ranks past 15. Fixed 2026-09-07: the real number is 100% recall, mean rank ~40 on a genuine question, versus ~2 for the note's own wording — the store finds everything, it just doesn't rank a real question near the top yet. Two follow-ups made it worse (a graph-proximity channel, an alternate embedding model); a third — generating a handful of plausible retrieval questions per note at index time and matching against those too — helped, a real but modest 5-8% drop in mean rank. Still experimental, off by default. A fourth follow-up did move the number that matters for a live caller: LLM listwise reranking over the fused candidate pool, inserted before the staleness downrank so a reorder can never re-promote a superseded entity (`WINZE_RERANK_TOPK`=40, `WINZE_RERANK_MODEL` defaults to Haiku; on by default for `winze_recall` as of 2026-09-08, not via the env var — see below for why). Ollama has no rerank endpoint and Cohere/Voyage have no self-host option, so this reranks via a bounded Anthropic call rather than a local cross-encoder — the first Anthropic call in `winze_recall`'s synchronous read path, fail-open on any error or timeout so a slow or down API degrades to unranked order instead of hanging the call. Measured on the same N=150 harness, 2026-09-07: against production's real default limit of 5, LATER-PROBE hit@5 rose from 14% to 34% and median rank fell from 25.0 to 18.0; TITLE-PROBE hit@5 rose from 80% to 99%. Misses already ranked past `WINZE_RERANK_TOPK` are untouched by design — a bounded-window reranker can't rescue a long tail it never sees. The rerank system prompt sits under the model's cache floor (confirmed live: `written=0 read=0` on two identical back-to-back calls), so it never reads from cache; the candidate list dominates per-call cost regardless and was never going to cache.

A second costrel consult (fable, 2026-09-08) said flip it on: the miss-decomposition gate the first consult set had closed (a mixed 42%-inside-window/58%-outside-window split at `WINZE_RERANK_TOPK`=40, not the "mostly inside" case fable named as the flip condition), and a direct `--hybrid` query against the worst unreranked TITLE-PROBE miss showed the mechanism it fixes concretely — "Investigate monitor tearing and artifacts after 26.04 upgrade" buried at rank 16 under fifteen unrelated "Investigate high RAM/CPU/memory/swap usage" session titles, a real near-duplicate-title-clutter defect in the deterministic first stage, not an RRF-weighting bug. But `WINZE_RERANK` as a single process-wide env var can't be "on for reads only" — `currentBrief` (`cmd/agent/mcp.go`), called from `winze_update`'s write handler, shares the exact same `--hybrid` mode and process environment as `winze_recall`'s read path, with no per-call override anywhere between them. Fixed with a `--rerank` CLI flag threaded through `runHybrid`/`rerankFused` (`rerankShouldRun`, OR'd with the existing env check): `winze_recall` now passes it explicitly, `currentBrief`'s identity lookup never does, and the env var still works unchanged for the harness. Two separately-tested levers before landing here both came back null: bumping `WINZE_RERANK_TOPK` 40→80 converted zero of the 26 newly-windowed candidates (window size wasn't it), and a cross-session Jaccard-overlap filter excluding recurring probe-text rituals ("pick a name, check for collisions," paraphrased across 5 of 150 sessions) moved hit@5 only 14%→16% (inside this project's own documented noise floor — probe quality wasn't the dominant driver either). Fable's own read on what's next, offered as inference rather than something it checked: the topK null plus the inside-window 42% also not converting together suggest the LATER-PROBE gap isn't a ranking problem at all — the reranker sees name+brief, and whatever a LATER probe needs may live in body/evidence text that never reaches it, pointing back at the still-unbuilt evidence-citation tier rather than more ranking work.

The same harness measured the raw-evidence tier (above) three times on 2026-09-07 at N=150, on identical sessions and questions: BM25-only, 41/125 (32.8%, mean rank 5.85); BM25+semantic, 50/125 (40.0%, mean rank 5.16); BM25+semantic+temporal, 50/125 (40.0%, mean rank 5.16, unchanged) — against 51/125 (40.8%, mean rank 5.35) for the typed store, itself unchanged and an exact repeat of its own milestone-2 figure. Adding the semantic channel closed nearly the whole gap; adding the temporal channel closed none of it, a real null rather than noise (the untouched typed-store row reproducing exactly rules out harness drift as the explanation) — see [docs/raw-evidence-retrieval.md](docs/raw-evidence-retrieval.md) for what all three numbers do and don't mean, measured before that tier's later retirement.
- **Rerank A/B is significant but not yet paired — medium-term confidence-building, not blocking.** The 14%→34% hit@5 jump above clears an unpaired two-proportion z-test (pooled p=0.24, z≈3.7, p≈0.0002) checked 2026-09-08 directly against the raw logs (`~/.cache/winze-selfrecall/selfrecall-median-check-n150-1788841845.log`, `selfrecall-rerank-n150-1788844230.log`). No on-disk manifest currently pairs the same 125 LATER-PROBE questions across both arms for an exact McNemar's test — the closest manifest (`manifest-rerank-1788845805.jsonl`, from the topK-decomposition run) doesn't reproduce the reported run's aggregate closely enough to stand in for it. Not worth blocking on given the unpaired result already clears significance by a wide margin; worth doing the next time this number needs re-confirming (a model swap, a topK change, or enough corpus regrowth to matter) — rerun both arms same-day with manifest-dumping on both sides.
- **The LATER-PROBE ceiling was partly a benchmark-modeling artifact, not a real production ceiling — measured, not assumed, 2026-09-08.** `cmd/longmemeval`'s own `noteFor` (the harness's synthetic memory-note builder) uses title + the literal opening question, capped 1200 chars — and its own doc comment already explains why: re-feeding the transcript to a model to compose a richer note "would simulate something that never happens... paying ~33k tokens per session to reconstruct what the writer already knew would measure a pipeline nobody runs." That reasoning is sound, but it doesn't mean the opening-question shape is a *faithful* proxy for what a real `winze_remember` call writes — checked directly against this project's own real dogfood store (`git config winze.store` → `~/Documents/winze-memory/memory.go`): real Briefs measured n=95, median 571 chars, mean 885, 27% exceeding the harness's 1200-char cap outright, and qualitatively dense with dates, commit hashes, percentages, and named mechanisms — an already-reached *outcome*, never a raw question. A new `WINZE_NOTE_SHAPE=outcome` (`cmd/longmemeval/transcript.go`'s `midpointOutcome`) tests that gap the same zero-extra-cost way `noteFor`'s existing shapes do — no new LLM call, just a different turn already on disk: the assistant's own last substantial response strictly before the probe turn, safe by construction (the walk breaks the instant it reaches the probe, the same exclusion pattern `ArcAsks` already used for the "arc" shape). Measured on the same N=150 harness, rerank held on (production's real default): LATER-PROBE hit@5 rose from 34% to **51%**, mean rank fell 38.78→18.57, median 18.0→**5.0** — nearly matching the reranker's own effect size, stacked on top of it. Sample size shifted slightly (125→119 later-probe sessions; the real transcript pool this harness draws from keeps growing as more sessions accumulate) so this isn't a byte-for-byte paired comparison, but the effect size is far too large to be pool drift. Practical reading: production's *real* LATER-PROBE-equivalent quality is likely much closer to 51% than to the previously-reported 34%, since real `winze_remember` calls already write outcome-shaped notes, not opening questions — the harness was undermeasuring winze, not winze undermeasuring the store. Open question, not yet resolved: should `outcome` become the harness's reported default going forward, with `open` kept only as the documented worst-case floor?

A costrel consult (fable, 2026-09-08) on "what's next given this" named the concrete engineering move: ship `outcome`'s derivation as a *production* capture path (auto-index the assistant's substantial turns from real session transcripts into recall, zero LLM, alongside authored Briefs) — the outcome-shape note is itself the exact zero-authoring, zero-LLM derived-retrieval capability the SOTA survey says nobody fills, and it currently only exists in this test harness, not in production. First step named: a cheap, no-LLM miss audit — for each LATER-PROBE miss, grep whether the probe's content ever reached the derived note at all, splitting coverage misses (not there) from retrieval misses (there, ranked poorly). Run 2026-09-08 on the N=150 manifest (`manifest-outcome-audit-1788883487.jsonl`): of 56 misses, **39 coverage, 4 retrieval, 13 ambiguous** — coverage dominates roughly 10:1, exactly fable's predicted branch. Reading the coverage misses by hand: a session drifts across topics, and the single last-assistant-turn capture often reflects a different stretch of conversation than whatever the later probe happens to ask about.

The obvious next move — widen `outcomeTurns` to capture every substantial assistant turn before the probe instead of just the last one — was built and measured the same day, and **made it worse**: hit@5 51-53%→38%, median rank 5.0→11.0 (same N=150, rerank held on). Reverted (never committed) rather than left in place as a regression. Inferred, not independently verified, cause: the multi-turn join truncated on a chronological budget (fills from the earliest pre-probe turn forward), so a long pre-probe run of turns pushed out the turn closest to the probe — the single turn the proven single-turn version was actually relying on. "Wider" wasn't the lever; "closest to the probe" was, and widening without also reordering by recency threw that away. That truncate-from-oldest-end fix was built and measured the same day (`outcomeTurns` widened to return every substantial pre-probe assistant turn, `outcomeNote` filling its 1500-char budget from the newest turn backward, then restoring chronological order for display) — **partial recovery, not a win**: hit@5 38%→**42%** (still N=150, rerank held on), confirming "closest to the probe survives" is directionally the right lever, but short of the single-turn baseline's 51-53%. Reverted (never committed), same as the prior attempt. Reading: a multi-turn join, even ordered correctly, still dilutes the single strongest signal (the one turn nearest the probe) with weaker ones sharing its budget — for this harness's note sizes, one turn beats several. Coverage misses (39/56 per the audit above) remain real and unaddressed by any turn-selection tweak; the fix that would actually reach them is picking *which* turn based on topical relevance to the probe, not proximity — genuinely bigger scope (needs some notion of what the probe is about before the note is written), not a cheap next experiment. Parked, not pursued further this session.

A second costrel consult (fable, 2026-09-08) on this exact plateau reframed the problem: both failed widening attempts kept the constraint of *one note per session* — production evidence (this project's own `winze-memory` commit history, full repo not just `memory.go`, checked directly) shows real sessions routinely write several distinct memories (4 in 52 minutes on 2026-09-02, 6+ within 15 minutes on 2026-08-29), so the harness's one-note assumption may itself be the artifact, the same shape as the opening-question-vs-outcome discovery above. Fable's proposed test: one retrieval unit per assistant turn instead of one note per session, scored at session level (any of a session's units in top-5 counts as a hit). Checked directly before building anything: `probeAll` already does exactly that scoring (`bestRankOf` over a session's full var list, its own doc comment says so), and it's already exercised today by `WINZE_NOTE_SHAPE=claims` — an LLM (Haiku) splits a session's user-turn asks into atomic facts, one entity per fact, N vars per session. Cost was also cheaper than fable assumed: `sessionExcerpt` (the input to the Haiku call) is bounded to the same ~3200-char excerpt the `arc` shape already uses, not a full-transcript re-summarization — nowhere near the "pipeline nobody runs" `OpeningAsk` rejects. So `claims` doubles as a free, already-built test of the multi-unit hypothesis, on the opposite speaker basis (user asks, not assistant replies) from `outcome`.

Run 2026-09-08, N=150, rerank on: LATER-PROBE hit@5 **50%**, mean rank 35.84, median rank 6.0 (117/117 recalled, 0 never surfaced) — statistically tied with `outcome`'s 51-53%, not a clear win by fable's own bar ("clearing 53% by more than run-to-run spread" — the 51-53% range is itself ~2 points of noise). TITLE PROBE also slipped, 100%/median rank 1.0 → 96%/median rank 2.08 — more candidate entities per session costs a little even on the easy probe. Write-rejection rose to 13% (73/581 attempted writes across 150 sessions, ~3.9 facts/session) against `outcome`'s single-write-per-session rate. Fable pre-committed the read for this outcome: "at or below baseline: you saved the production build and learned coverage misses aren't a unit problem." Applying that directly — this result argues *against* building the bigger zero-LLM per-assistant-turn `writeSessions` change fable proposed, since the cheaper multi-unit implementation already available (`claims`) didn't clear the bar either. Single-turn `outcome` (51-53% hit@5) remains the current best and the harness's recommended default. This note-shape/unit-count axis is now reasonably exhausted for cheap iteration; the coverage-miss population (39/56) stands unresolved, and the two live options are the larger production-capture-path build (real engineering scope, not a harness tweak) or shifting effort to a different SOTA lever entirely.
- **Hash-checked evidence archiving shipped for curated-corpus claims — the session-derived half of the same ask stays open.** `Provenance.EvidenceHash` (`corpus/schema.go`), a new `cmd/lint` rule (`evidence-span`, `docs/lint-rules.md`), and a content-addressed `evidence/<hash>.txt` archive close the gap named in the raw-evidence-tier consult above for winze's own hand-curated claims: `Provenance.Origin`'s own doc comment already called external sources "transient... never required to resolve to a live file," and now a named `Provenance` var can additionally carry a hash-checked, verbatim-checked archive of its `Quote` — `evidence-span` fails the build gate if the archive goes missing, is corrupted, or `Quote` drifts from what was actually archived. This does **not** move the LATER-PROBE self-recall number above: checked directly this session, `handleRemember`/`execDocument` (`cmd/agent/mcp.go`) already attach the *same string* as both a session-derived claim's `Brief` and its `Quote` — there is no richer text sitting behind the Brief for a citation mechanism to point at instead. The real gap for that population is upstream of winze: `cmd/longmemeval`'s own `noteFor` writes a deliberately partial note (title + opening ask only, capped at 1200 chars, "the cheapest note that could work"), so most of a real session never reaches `winze_remember` at all — fixing that needs a richer write-time capture mechanism, not a citation primitive, and is real, separate, unscoped future work. Also scoped narrower than it might sound: archiving is deliberately limited to the exact `Quote` text (never a full external source document — copyright and repo-growth reasons, named in the design), and only visible to lint on a *named, top-level* `Provenance` var, since `cmd/add`'s default inline mode (`renderClaim`, `cmd/add/main.go`) nests the literal inside the claim itself, invisible to the AST walker (`internal/corpusparse.ParseCorpusFull`) this rule reads from — checked directly rather than assumed, which is why there's no new `cmd/add` flag for this (archiving is hand-authored, same posture as `coderef-span`'s existing `Span` citations).
- **`winze_recall`'s JSON response always reports `score: 0`.** Found while building the link-edge fixture above: `handleRecall` (`cmd/agent/mcp.go`) decodes `--hybrid`'s JSON into `queryHit`, whose `Score` field is tagged for `--hybrid`'s never-emitted `score` key — `--hybrid` only ever emits `rrf`/`lex_rank`/`sem_rank`. Production's dedup and link-suggestion logic (`checkDedup` → `nearestMemories`) is unaffected, since it queries `--semantic` directly and gets real cosine values; only the score a caller of `winze_recall` itself sees is dead. Not fixed here — out of scope for the milestone that found it — but worth knowing before trusting that field.
- Wikipedia provenance concentration (HHI 0.49, 65% Wikipedia). The availability-heuristic bias gate skips ZIM when this fires; Kagi fills the resulting signal gap. Structural fix still requires diversified ingest landing as actual cycles, not just plumbing.
- Survivorship bias structural, not prompt-tightness (191:2 irrelevant-to-challenged). Search-based sensors return supports + topic noise, not contradictions. The recalibrated resolver prompt classifies correctly; growing the challenged-count requires a counter-evidence query path the metabolism doesn't have yet.
- RSS feed curation unsolved — default topic feeds don't match entity-specific topology queries (0/131 signal historically). RSS still available via `--backend rss` for contributors who curate their own feeds.
- Source reputation not yet tracked — no automated mechanism for down-weighting historically low-quality domains based on calibration outcomes.
- **Prose claims about external dependencies rot with nothing checking them.** `internal/defndb`'s package doc asserted that defn was Dolt-backed and too heavy to link; that stopped being true and nothing noticed, and an agent later read it as a live measurement and built ~500 lines on top of it. `CodeRef` makes a doc→code reference fail the build when it goes stale, but `Symbol` is only checked when the store and the cited code share a module — it can't reach a client repo winze-agent serves, and never reaches non-Go code. `CodeRef` now has `Client`/`Span` fields for exactly that reach, checked by `cmd/lint --clients` (a content-hash check for non-Go targets, shipped 2026-09-02) rather than the compiler — see [docs/typed-citation.md](docs/typed-citation.md) for what that trades away. The narrower half — a bare undated claim, independent of whether it cites real code — is also now caught: `cmd/lint`'s dated-measurement rule flags a `Brief`, `Rationale`, or `Quote` that reads as measurement-shaped with no date (advisory, shipped 2026-09-02).

## DARPA alignment

Two active DARPA programs (April 2026) overlap with winze's approach. Both have public program pages at [darpa.mil](https://www.darpa.mil/); BAA documents are indexed on [SAM.gov](https://sam.gov/).

**MATHBAC** (Mathematics of Boosting Agentic Communication): "AI excels at navigating solution spaces but struggles to systematically explore hypothesis spaces." Winze's metabolism is systematic hypothesis space exploration. Multiple instances (forks sharing improvements via pull requests, each running the metabolism loop on its own schedule) map to MATHBAC's agent collective topology.

**CLARA** (Compositional Learning-And-Reasoning for AI): tight integration of formal reasoning with ML. Winze composes Go's type system (AR) with LLM metabolism (ML) through a quality-gated pipeline. Gap: Go catches structural errors but doesn't produce proof certificates. Z3 or Goose (Go → Coq) could close it.

## Lead with memory, not epistemics

Working notes: a repositioning argument and what's come of it since, kept
here to stand on its own for a session picking this up cold. Read
`docs/sota-memory-systems-survey-2026-08-31.md` first; this section builds
on it and doesn't repeat it.

As of 2026-09-07: the README changes described below have shipped, Eywa's
been read in full, and every open question raised along the way is
resolved — including, in this section's own closing part, whether the new
raw-evidence tier named above in "Also next" is a real architectural class
or a repeat of the tier this same day's phase 3 just retired.

### The correction this is recording

Winze's own README pitches its differentiator as epistemic self-awareness —
"knowing where you're probably wrong" — with retrieval-style continuity
treated as a solved-elsewhere problem winze deliberately isn't targeting. The
actual argument for that framing: this project should be judged first as a
**world-class 2026-standard memory system**, and epistemic correctness
(calibration, contested-claim tracking, staleness) should be positioned as
what a *good* memory system produces as a consequence of doing memory right
— not as a separate, co-equal pitch competing with continuity for attention
and roadmap time.

That's not fully true of every mechanism in the stack (see below — some of
winze's correctness machinery is genuinely separate engineering, not a free
byproduct), but the framing correction stands regardless: lead with memory
quality, subordinate the epistemics story to it.

### Where winze already agrees with this, undocumented as agreement

`docs/sota-memory-systems-survey-2026-08-31.md` already did most of the hard
part of this repositioning, seven days before this section, without labeling it
as one. It's a real, dated, confidence-marked comparison against Claude
Code's own memory, Letta/MemGPT, mem0, Anthropic's `memory_20250818` tool,
and Cursor/Windsurf — and its central finding is stated in memory terms, not
epistemics terms: *"the gap nobody in this table fills: automatic, derived,
semantic retrieval over a user's own prior sessions, keyed on live intent,
with no authoring step."* Winze requires curation; every competitor that
retrieves does so over something authored first (extracted, promoted, or
written). That's a memory-continuity gap statement, not an epistemics one.

Also already shipped, and already memory-framed, not epistemics-framed:

- `docs/memtool.md` — winze is the backend for Anthropic's own native
  `memory_20250818` tool (`integrations/memtool`, live round-trip demo at
  `cmd/memtooldemo`). This is a real, already-built continuity mechanism,
  not a hypothetical.
- `docs/benchmark.md` — a retrieval benchmark harness (`winze-benchmark`,
  grep/bm25/defn/ast comparison) already exists.
- A `longmemeval` binary sits at the repo root (built 2026-09-02, ~25MB) —
  LongMemEval tooling is already present, even if the survey doc is
  explicitly cautious about porting its numbers onto a single-user system
  (see the survey's mem0 section: a LoCoMo/LongMemEval score "is a real,
  public number" for a synthetic multi-persona benchmark, and "porting [it]
  onto a single-user, one-host system is a fabricated comparison, not a
  measurement" — a caution worth keeping, not overriding, in whatever
  benchmarking work follows from this section).

None of that needed inventing. It needed noticing that it's already the
strongest evidence for the repositioning, and that the README's public-facing
identity doesn't currently lead with it.

### What's new since the 2026-08-31 survey

The survey compared winze against shipped *products*. It's thin on the
research-literature side, understandably — that wasn't its brief. A search
run 2026-09-07 for the current state of AI agent memory research surfaces
material the survey doesn't cite:

- **Three problems the field names as the hardest open ones in 2026:
  cross-session identity, temporal abstraction at scale, and memory
  staleness** (per LoCoMo/LongMemEval/BEAM-benchmark-era summaries — worth
  independently verifying against the primary benchmark papers before citing
  further, this is a search-result characterization, not a primary-source
  read). All three map directly onto winze's own existing machinery: typed
  citation is a staleness mechanism (a stale reference is a build error, not
  a maybe — `docs/typed-citation.md`), and the metabolism's dream/trip/evolve
  cycle is a temporal-abstraction mechanism (consolidation, speculative
  cross-links, topology-driven re-query) already built for reasons that
  happened to also be memory reasons.
- **Eywa — "Provenance-Grounded Long-Term Memory for AI Agents"**
  (arXiv 2605.30771, https://arxiv.org/abs/2605.30771, Resham Joshi, single-
  author preprint — **abstract read in full 2026-09-07, body not read**;
  don't extend the claims below past what the abstract actually states).

  Its architecture is stated as "evidence before belief": immutable source
  evidence is stored before canonical facts are derived from it, extracted
  memories are validated against typed signals and source support, and
  retrieval runs through a "deterministic multi-route read path with zero
  LLM calls inside retrieval" — retrieved context comes back separate from
  answer instructions, so the same memory substrate can be evaluated under
  different answer models (frontier, budget, local) without changing what
  gets retrieved. Reported numbers, under a frozen retrieval config: 90.19%
  judge accuracy on LoCoMo's C1-C4 split (Claude Sonnet 4.6 write+QA roles),
  88.2% retrieval-sufficiency on LongMemEval-S, 81.45% mean nugget score /
  85.29% pass@≥0.5 on BEAM (a 700-question technical-memory stress
  benchmark). Per-question artifacts (questions, gold answers, model
  answers, retrieved context, labels) are published.

  **Where this validates winze's approach independently.** "Evidence before
  belief" is the same move as winze's `Provenance`/`Conjecture` split
  (`CLAUDE.md`): a claim's attribution is either a sourced `Provenance` with
  an exact `Quote`, or a `Conjecture` that structurally cannot carry a
  `Quote` field at all — the compiler forecloses a generated claim wearing a
  fabricated source. Two independently-arrived-at systems landing on
  provenance-gates-belief as the mechanism is real evidence for the "world-
  class memory absorbs correctness for free" argument this section is making —
  Eywa isn't pitched as an epistemics project at all, it's pitched as a
  memory-infrastructure project, and provenance-grounding is just how it
  gets built.

  **Where it's ahead of winze, concretely.** Winze's read side is mixed, not
  uniformly deterministic: `docs/query.md` documents both a structured query
  mode and a `--ask` mode that explicitly "sends the full KB context to an
  LLM for natural language answers" (`docs/query.md:22,28`). Eywa's claim is
  that retrieval itself — including, per the abstract, answering — never
  calls an LLM inside the retrieval path; the LLM only sits at the answer-
  generation step, decoupled and swappable. That's a sharper commitment than
  winze currently has documented anywhere, and it's the thing making Eywa's
  multi-answer-model evaluation possible at all. Also ahead: three external,
  named benchmark numbers with published per-question artifacts. Winze's own
  `docs/benchmark.md` retrieval benchmark exists but nothing in this pass
  confirmed it publishes artifacts at that granularity — worth checking, not
  assumed either way.

  **Where they're not actually substitutable.** Eywa reads as scoped to
  session-scoped agent memory for conversational QA — the abstract's whole
  frame is "AI agents that persist across sessions need memory they can
  retrieve, audit, update, and erase." Nothing in the abstract suggests it
  has anything like contested-claims-as-first-class-structure, a calibration
  time-series, or an ongoing metabolism (dream/trip/evolve) — it's a
  retrieval/evidence pipeline, not a standing, curated, self-auditing
  epistemic corpus. Winze is solving a different-shaped problem (a durable
  KB a human curates and the system tends over time) that happens to share
  Eywa's core mechanism at the provenance layer. Worth reading the full
  paper before concluding more than that; the abstract alone doesn't settle
  whether Eywa's read path is a better version of something winze already
  does, or a different tool entirely.

  **The caution that still applies.** The 2026-08-31 survey doc already
  flagged, about mem0, that a LoCoMo/LongMemEval number is measured on a
  synthetic, adversarially-constructed, multi-persona corpus built to
  compare products at scale — porting it onto a single-user, one-host system
  "is a fabricated comparison, not a measurement." That caution applies to
  Eywa's numbers exactly as much as it applied to mem0's; Eywa is measured
  on the same benchmark population. Publishing per-question artifacts is a
  real transparency improvement over mem0's headline-number-only approach,
  but it doesn't change what population the number describes.
- Other 2026 research worth a scan, same caveat (titles from search, content
  unread): A-MEM (autonomous memory organize/update/prune), EverMemOS
  ("self-organizing memory operating system for structured long-horizon
  reasoning" — January 2026), MAGMA (multi-graph agentic memory
  architecture). None of these were cross-checked against winze's own
  design; flagging them as a reading list, not a verified comparison.

### The Tao mapping, since it's what prompted this

Terence Tao, "The paradox at the heart of AI and science" (Big Think Clips,
2026-09-03, https://youtu.be/svl_1upFpQo). His description of attunement
with a longtime collaborator spans three timescales without naming them as
such: sub-second (completing a sentence before it's finished), day-to-day
(resuming a dropped thread instantly), and multi-year (picking up something
dormant with someone not spoken to in a long time). He's explicit that
current AI does none of these the way a person does — it "can make notes and
kind of simulate this memory," but that isn't attunement.

Map that against the newly-surfaced field framing: his three timescales are
close cousins of the field's three named hard problems — multi-year
dormant-thread revival is a cross-session-identity problem, day-to-day
resume is temporal abstraction at a shorter scale, and sub-second attunement
is unreachable at all without staleness-awareness (you cannot confidently
resume anything, at any timescale, without knowing which of your beliefs
about it have gone stale). That last point is the real justification for
"epistemic correctness rides along for free": *at the long-horizon end of
memory, staleness-tracking is not a separate feature, it's the mechanism
that makes long-horizon resume honest instead of confidently wrong.* Winze
already has that mechanism, built for a different stated reason. It should
be re-described, in the README and in roadmap priority, as core memory
infrastructure — not as the standalone "epistemic self-awareness" pitch it
currently reads as.

### What is NOT free, and shouldn't be pitched as if it were

Being honest about the boundary matters more than the pivot itself: typed
citation and contested-claim representation are genuinely memory-integrity
mechanics — build honest
reference-tracking and honest representation-of-disagreement, and
correctness falls out, no separate design required. But the **calibration
time-series, per-source corroborated/challenged/refuted counting, the
Wikipedia-concentration (HHI) down-weighting, and the bias-audit-against-own-
structure pass** (all in `docs/metabolism.md`'s evolve phase) are deliberately
engineered analysis passes sitting on top of stored claims. A system that
only resumed threads fast and attuned well would not spontaneously start
counting how often a source gets contradicted. Keep building these — they're
real and they're good — but justify and prioritize them by how much they
serve memory quality (do they make a resumed thread more trustworthy?), not
as a parallel headline feature competing with continuity for the pitch.

### What shipped

The README's first paragraph now leads with the continuity/retrieval
framing — the survey's own "gap nobody fills" line is a stronger hook than
"knowing where you're probably wrong," and it's already true, dated
evidence, not aspirational (`fee1cbb`, `9ad8164`; the second commit also
caught the same stale framing in "Where this is headed," which the first
pass missed). "Where this is headed" now justifies dream/trip/evolve by
resumed-thread trustworthiness instead of pitching them as their own
headline — the epistemics work stays, just reframed as infrastructure.
That reframe only touched the top-level pitch; it hasn't been swept through
`docs/metabolism.md` or the other internal docs, which is narrower than
what this section originally reached for.

Eywa got read in full (arXiv 2605.30771, via the ar5iv HTML mirror, per
this project's own curl-first citation discipline) rather than left at the
abstract — see the expanded section below.

The survey's named gap — no system does automatic derived retrieval over
raw session history with zero authoring step — got named in `ROADMAP.md`'s
"Also next" as the next priority. It's not scoped as an implementation
plan: that's real multi-file architecture (a second storage/retrieval
subsystem alongside the typed corpus) and needs plan mode + `/check-plan`,
not a freehand build under a documentation task.

### Eywa, now read in full (2026-09-07)

The abstract read (original section below) held up; the full paper adds
mechanism and an honest limitations section worth carrying into winze's own
thinking.

**Failure taxonomy (§3, Table 1) is the sharpest reusable artifact.** Eywa
names eight ways a memory-backed answer can fail and insists on keeping them
apart rather than collapsing them into one pass/fail number: coverage gap
(evidence was never captured), grounding gap (a derived fact isn't actually
supported by its source), revision gap (a superseded fact still answers),
scope gap (the wrong person's memory answers), temporal gap (right memory,
wrong time window), retrieval gap (the fact exists but isn't surfaced),
synthesis gap (surfaced but the answer model still gets it wrong), and
measurement gap (the metric, not the system, is what disagrees). Applied to
winze's own open self-recall problem ([[project_selfrecall_next_steps_2026_09_02]]):
the 47.5-52% later-probe miss rate is almost certainly a **retrieval gap**
specifically — the fact is written and stored (ruled out coverage gap via
the raw-tier check), the dedup cross-reference ruled out a grounding/
rejection-driven explanation, and rank dilution (the leading remaining
hypothesis) is a textbook retrieval-gap shape: present in the store, buried
by ranking. Worth adopting this vocabulary in winze's own docs instead of
inventing parallel terms.

**Architecture: evidence, signals, beliefs — three objects, one direction.**
(§3, §4.1-4.3) Evidence is immutable raw turns. Signals are deterministic,
LLM-free typed detections over evidence (dates, entities, amounts, URLs —
Tier 0, no model call). Beliefs are LLM-extracted, validated, revisable
facts linked back to their evidence by a provenance edge — and the paper is
explicit that **extraction is an index, not the memory**: if a belief is
missing or wrong, the authoritative evidence underneath it is still there to
repair from. This is winze's own `Provenance.Quote` discipline generalized
one level down: winze already refuses to let a claim exist without its
exact source quote; Eywa's move is to also keep the *entire session*, not
just the quoted fragment, as an addressable, immutable substrate underneath
the claim layer.

**Read path: deterministic, multi-route, zero LLM calls at query time.**
(§4.5) A hand-coded query-shape planner (not a learned classifier, not an
LLM call) weights four retrieval channels — vector cosine, keyword BM25,
temporal date-range, entity/graph traversal — per query shape (inference,
contradiction/update, multi-session/relation, summary/recap, explicit-date,
list/aggregation), fuses them with weighted reciprocal rank fusion, then
applies two deterministic post-fusion filters: a person-scope demotion
(×0.05 for facts scoped to someone else) and a preservation floor (a
validated candidate can't be reranked below rank 25). This is a sharper
commitment than winze has anywhere today — `docs/query.md`'s `--ask` mode
explicitly sends full KB context to an LLM for the answer itself, not just
for planning. Eywa keeps the LLM entirely out of retrieval and only lets it
touch the final answer step, which is what makes its multi-answer-model
evaluation (frontier/budget/local, same retrieval) possible at all.

**Honest limitations that read like a template winze could reuse.** 16 named
limitations (§8), several with a real number attached rather than a
qualitative hedge: a 143-sample write-path audit found 67.4% of candidate
facts carried no hard anchor (checked instead by support-overlap, subject,
and polarity rules) and rejected 11 of 132 candidates, mostly for
insufficient source overlap or invented values; in the actual LoCoMo
observation database, only 59 of 2,541 persisted facts contained a hard
anchor in the fact text at all. That's the same "state the actual rejection
rate, not just the mechanism" discipline this project's own dated-measurement
and dedup-cross-reference work already follows — Eywa runs it at a
comparable scale and reports it with comparable candor.

**Benchmark numbers, same caveat this project's own survey already named.**
90.19% LoCoMo C1-C4 (Claude Sonnet 4.6 write+QA), 88.2% LongMemEval-S
retrieval-sufficiency, 81.45%/85.29% on BEAM (Eywa's own new 700-question
stress benchmark, not yet independently validated per the paper's own §7/§8).
The paper's own Threats to Validity section makes the same point
`docs/sota-memory-systems-survey-2026-08-31.md` made about mem0: these are
synthetic, adversarially-constructed, multi-persona corpora built to compare
products at scale, reported without confidence intervals in this version,
and not a controlled head-to-head against competitors. Worth citing Eywa's
architecture, not its leaderboard position, for exactly the reason this
project already declined to cite mem0's.

**Where Eywa is ahead of winze, concretely, now confirmed rather than
guessed:** the zero-LLM-read commitment above, the explicit failure
taxonomy, and the person-scope + preservation-floor read-time mechanics —
none of which winze has today. **Where they're not substitutable:** nothing
in the full paper changes the abstract-stage read that Eywa is a
session-scoped conversational-QA memory pipeline, not a standing,
curated, self-auditing epistemic corpus with contested claims and a
metabolism. The provenance mechanism generalizes; the product shape doesn't.

### Open questions — resolved 2026-09-07

1. **Does closing the authoring-required gap conflict with `CLAUDE.md`
   mirror-source-commitments? Resolved: it can, and already did once — but
   only if the mechanism is built the wrong way.** Winze already reverted an
   automated writer that emitted fabricated `Quote` fields
   ([[feedback_trip_promotion_fabrication]]) — the exact failure mode any
   automatic derived-retrieval mechanism risks if its raw hits get promoted
   straight into typed `Provenance`-backed claims. Eywa's "extraction is an
   index, not the memory" principle and a related private project's own
   explicit rejection of a similar `harvest`-into-winze feature (see
   question 3) both land on the same resolution independently: **close the
   gap below the claim graph, not
   into it.** A raw-evidence retrieval tier — verbatim session text, hashed,
   timestamped, deterministically searchable, structurally separate from
   `Entity`/claim types — returns *source text* on a query, not a claim.
   Nothing gets encoded, so mirror-source-commitments (which governs what
   may be asserted as a claim) doesn't apply to it at all. It's a new object
   class, not a new way to populate the existing one.
2. **Eywa's full paper — read.** See the expanded section above.
3. **Is there a related-project connection worth formalizing? Resolved:
   yes, and it's already flowing one direction, undocumented.** A private
   sibling project working on cross-session handoff continuity already
   names winze directly in its own design notes, sources its SOTA
   comparison from the same 2026-08-31 survey this section's predecessor
   cites (not two independent analyses — the same one reused across both),
   and already rejected a harvest-into-winze feature for the exact same
   fabricated-`Quote` reason named in question 1 above — its authors
   reasoned through this independently and reached the same conclusion. It
   has also already folded in a methodology fix from winze's own side, on
   how it measures whether superseded content gets dropped across a
   handoff. And it's the concrete, already-built, already-measured instance
   of "the gap nobody fills" this section's survey named only as an
   abstract absence: automatic, zero-authoring retrieval over real (not
   synthetic) raw sessions, outperforming Claude Code's own `/compact` by a
   wide margin on its own internal benchmark. Whatever winze builds for its
   own raw-evidence tier should study that project's design before
   inventing a parallel one — the mechanism (deterministic retrieval with
   zero authoring step) is the same problem shape, even though its target
   (session-handoff continuity) and winze's (a curated, contested-claims
   corpus) are different products.

### Is the new raw-evidence tier real, or `raw.jsonl` again? — resolved 2026-09-07

Phase 3 of retiring the OLD raw-evidence tier (`raw.jsonl`, `winze_recall_raw`,
`--raw` — `docs/raw-evidence-retrieval.md`) shipped the same day as this section
named a NEW raw-evidence tier the top priority for closing the authoring
gap above. Same name, opposite direction, and nothing above had stated why this tier's
required separateness from typed claims is permanent rather than the same
temporary crutch `raw.jsonl` turned out to be.

Ran a costrel consult (fable) on the question directly. Verdict: **build
it — it's a permanent class, under one property this section hadn't stated.**
The property: **claims cite into the tier, they don't copy out of it.** A
session-derived claim's `Provenance` should carry a hash-checked reference
to a span in the evidence tier, not an inlined `Quote` string, so that
removing the tier breaks the build gate rather than a coverage check. Their
test for which one's been built: *could a string-coverage check ever
retire this tier?* If yes, it's `raw.jsonl` again — that is literally the
mechanism phase 3 used to retire the old one. Two implementation details
this leaves genuinely open, not yet checked: whether `Provenance.Origin` is
a structured locator today or a free string (the reference needs the
former), and whether `080d961` (coderef's content-hash-checked `Span`
citations, cross-repo/non-Go) is the same primitive to reuse.

This also reframes Open question 1's fabrication-risk justification above.
Fabrication is a property of one promoter (the trip cycle), not of the
tier's shape — a promoter constrained to emit hash-checked span references
can't fabricate regardless of where it writes. If separateness rests on
fabrication risk alone, "safe promoter exists → promote everything → tier's
covered → delete" is exactly `raw.jsonl`'s trajectory. The real, permanent
reason: a session transcript has no durable external referent the way a
paper or a Wikipedia article does (those already exist outside winze; a
compacted transcript doesn't) — winze has to be the thing holding it, the
same way it already holds nothing else the source itself preserves.

**Carry forward into the eventual plan, so it isn't lost:** an immutable
evidence tier grows unbounded and will need its own pruning/retention
policy (age, never-retrieved) — a within-tier decision, separate from
whether the tier should exist at all. It shouldn't get scoped as part of
"should we build this," and it shouldn't get forgotten just because nothing
forces the question until the store is already large.

---
Drafted 2026-09-07, prompted by Tao's interview and how it maps onto
saturday, ettle, and winze. The field-framing claims above (named
hard problems, paper titles) come from a single search pass the same day,
not primary-source reads — treat them as a pointer to check, not a verified
survey, until someone reads the actual papers.
