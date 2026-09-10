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
- **The LATER-PROBE ceiling was partly a benchmark-modeling artifact, not a real production ceiling — measured, not assumed, 2026-09-08.** `cmd/longmemeval`'s own `noteFor` (the harness's synthetic memory-note builder) uses title + the literal opening question, capped 1200 chars — and its own doc comment already explains why: re-feeding the transcript to a model to compose a richer note "would simulate something that never happens... paying ~33k tokens per session to reconstruct what the writer already knew would measure a pipeline nobody runs." That reasoning is sound, but it doesn't mean the opening-question shape is a *faithful* proxy for what a real `winze_remember` call writes — checked directly against this project's own real (private) dogfood store: real Briefs measured n=95, median 571 chars, mean 885, 27% exceeding the harness's 1200-char cap outright, and qualitatively dense with dates, commit hashes, percentages, and named mechanisms — an already-reached *outcome*, never a raw question. A new `WINZE_NOTE_SHAPE=outcome` (`cmd/longmemeval/transcript.go`'s `midpointOutcome`) tests that gap the same zero-extra-cost way `noteFor`'s existing shapes do — no new LLM call, just a different turn already on disk: the assistant's own last substantial response strictly before the probe turn, safe by construction (the walk breaks the instant it reaches the probe, the same exclusion pattern `ArcAsks` already used for the "arc" shape). Measured on the same N=150 harness, rerank held on (production's real default): LATER-PROBE hit@5 rose from 34% to **51%**, mean rank fell 38.78→18.57, median 18.0→**5.0** — nearly matching the reranker's own effect size, stacked on top of it. Sample size shifted slightly (125→119 later-probe sessions; the real transcript pool this harness draws from keeps growing as more sessions accumulate) so this isn't a byte-for-byte paired comparison, but the effect size is far too large to be pool drift. Practical reading: production's *real* LATER-PROBE-equivalent quality is likely much closer to 51% than to the previously-reported 34%, since real `winze_remember` calls already write outcome-shaped notes, not opening questions — the harness was undermeasuring winze, not winze undermeasuring the store. Open question, not yet resolved: should `outcome` become the harness's reported default going forward, with `open` kept only as the documented worst-case floor?

A costrel consult (fable, 2026-09-08) on "what's next given this" named the concrete engineering move: ship `outcome`'s derivation as a *production* capture path (auto-index the assistant's substantial turns from real session transcripts into recall, zero LLM, alongside authored Briefs) — the outcome-shape note is itself the exact zero-authoring, zero-LLM derived-retrieval capability the SOTA survey says nobody fills, and it currently only exists in this test harness, not in production. First step named: a cheap, no-LLM miss audit — for each LATER-PROBE miss, grep whether the probe's content ever reached the derived note at all, splitting coverage misses (not there) from retrieval misses (there, ranked poorly). Run 2026-09-08 on the N=150 manifest (`manifest-outcome-audit-1788883487.jsonl`): of 56 misses, **39 coverage, 4 retrieval, 13 ambiguous** — coverage dominates roughly 10:1, exactly fable's predicted branch. Reading the coverage misses by hand: a session drifts across topics, and the single last-assistant-turn capture often reflects a different stretch of conversation than whatever the later probe happens to ask about.

The obvious next move — widen `outcomeTurns` to capture every substantial assistant turn before the probe instead of just the last one — was built and measured the same day, and **made it worse**: hit@5 51-53%→38%, median rank 5.0→11.0 (same N=150, rerank held on). Reverted (never committed) rather than left in place as a regression. Inferred, not independently verified, cause: the multi-turn join truncated on a chronological budget (fills from the earliest pre-probe turn forward), so a long pre-probe run of turns pushed out the turn closest to the probe — the single turn the proven single-turn version was actually relying on. "Wider" wasn't the lever; "closest to the probe" was, and widening without also reordering by recency threw that away. That truncate-from-oldest-end fix was built and measured the same day (`outcomeTurns` widened to return every substantial pre-probe assistant turn, `outcomeNote` filling its 1500-char budget from the newest turn backward, then restoring chronological order for display) — **partial recovery, not a win**: hit@5 38%→**42%** (still N=150, rerank held on), confirming "closest to the probe survives" is directionally the right lever, but short of the single-turn baseline's 51-53%. Reverted (never committed), same as the prior attempt. Reading: a multi-turn join, even ordered correctly, still dilutes the single strongest signal (the one turn nearest the probe) with weaker ones sharing its budget — for this harness's note sizes, one turn beats several. Coverage misses (39/56 per the audit above) remain real and unaddressed by any turn-selection tweak; the fix that would actually reach them is picking *which* turn based on topical relevance to the probe, not proximity — genuinely bigger scope (needs some notion of what the probe is about before the note is written), not a cheap next experiment. Parked, not pursued further this session.

A second costrel consult (fable, 2026-09-08) on this exact plateau reframed the problem: both failed widening attempts kept the constraint of *one note per session* — production evidence (this project's own `winze-memory` commit history, full repo not just `memory.go`, checked directly) shows real sessions routinely write several distinct memories (4 in 52 minutes on 2026-09-02, 6+ within 15 minutes on 2026-08-29), so the harness's one-note assumption may itself be the artifact, the same shape as the opening-question-vs-outcome discovery above. Fable's proposed test: one retrieval unit per assistant turn instead of one note per session, scored at session level (any of a session's units in top-5 counts as a hit). Checked directly before building anything: `probeAll` already does exactly that scoring (`bestRankOf` over a session's full var list, its own doc comment says so), and it's already exercised today by `WINZE_NOTE_SHAPE=claims` — an LLM (Haiku) splits a session's user-turn asks into atomic facts, one entity per fact, N vars per session. Cost was also cheaper than fable assumed: `sessionExcerpt` (the input to the Haiku call) is bounded to the same ~3200-char excerpt the `arc` shape already uses, not a full-transcript re-summarization — nowhere near the "pipeline nobody runs" `OpeningAsk` rejects. So `claims` doubles as a free, already-built test of the multi-unit hypothesis, on the opposite speaker basis (user asks, not assistant replies) from `outcome`.

Run 2026-09-08, N=150, rerank on: LATER-PROBE hit@5 **50%**, mean rank 35.84, median rank 6.0 (117/117 recalled, 0 never surfaced) — statistically tied with `outcome`'s 51-53%, not a clear win by fable's own bar ("clearing 53% by more than run-to-run spread" — the 51-53% range is itself ~2 points of noise). TITLE PROBE also slipped, 100%/median rank 1.0 → 96%/median rank 2.08 — more candidate entities per session costs a little even on the easy probe. Write-rejection rose to 13% (73/581 attempted writes across 150 sessions, ~3.9 facts/session) against `outcome`'s single-write-per-session rate. Fable pre-committed the read for this outcome: "at or below baseline: you saved the production build and learned coverage misses aren't a unit problem." Applying that directly — this result argues *against* building the bigger zero-LLM per-assistant-turn `writeSessions` change fable proposed, since the cheaper multi-unit implementation already available (`claims`) didn't clear the bar either. Single-turn `outcome` (51-53% hit@5) remains the current best and the harness's recommended default. This note-shape/unit-count axis is now reasonably exhausted for cheap iteration; the coverage-miss population (39/56) stands unresolved, and the two live options are the larger production-capture-path build (real engineering scope, not a harness tweak) or shifting effort to a different SOTA lever entirely.
- **Hash-checked evidence archiving shipped for curated-corpus claims — the session-derived half of the same ask stays open.** `Provenance.EvidenceHash` (`corpus/schema.go`), a new `cmd/lint` rule (`evidence-span`, `docs/lint-rules.md`), and a content-addressed `evidence/<hash>.txt` archive close the gap named in the raw-evidence-tier consult above for winze's own hand-curated claims: `Provenance.Origin`'s own doc comment already called external sources "transient... never required to resolve to a live file," and now a named `Provenance` var can additionally carry a hash-checked, verbatim-checked archive of its `Quote` — `evidence-span` fails the build gate if the archive goes missing, is corrupted, or `Quote` drifts from what was actually archived. This does **not** move the LATER-PROBE self-recall number above: checked directly this session, `handleRemember`/`execDocument` (`cmd/agent/mcp.go`) already attach the *same string* as both a session-derived claim's `Brief` and its `Quote` — there is no richer text sitting behind the Brief for a citation mechanism to point at instead. The real gap for that population is upstream of winze: `cmd/longmemeval`'s own `noteFor` writes a deliberately partial note (title + opening ask only, capped at 1200 chars, "the cheapest note that could work"), so most of a real session never reaches `winze_remember` at all — fixing that needs a richer write-time capture mechanism, not a citation primitive, and is real, separate, unscoped future work. Also scoped narrower than it might sound: archiving is deliberately limited to the exact `Quote` text (never a full external source document — copyright and repo-growth reasons, named in the design), and only visible to lint on a *named, top-level* `Provenance` var, since `cmd/add`'s default inline mode (`renderClaim`, `cmd/add/main.go`) nests the literal inside the claim itself, invisible to the AST walker (`internal/corpusparse.ParseCorpusFull`) this rule reads from — checked directly rather than assumed, which is why there's no new `cmd/add` flag for this (archiving is hand-authored, same posture as `coderef-span`'s existing `Span` citations).
- **Richness-over-recency and multi-note capture both tried, both worse — the "note-shape axis exhausted" verdict above holds even more firmly now, 2026-09-09.** Scoping a production session-end auto-capture feature (`winze-agent session-capture`, planned but not yet built) surfaced a real observation first: run against a real transcript, `outcome`'s "last substantial assistant turn" captured a thin closing status check while a session's actual substance (a disk-cleanup job, 1023 chars) sat several turns earlier — a real session can wander, and recency can miss the point. The natural-seeming fix — pick the *longest* substantial turn instead of the last one — was built as a new zero-LLM `WINZE_NOTE_SHAPE=topk` (`topKNotes`, `cmd/longmemeval/selfrecall_corpus_test.go`, top-`$WINZE_TOPK_N` longest assistant turns strictly before the held-out later-ask, mirroring `midpointOutcome`'s exact walk/stop condition so the comparison is fair) and measured directly against the same N=150 LATER-PROBE harness, same rerank-on config: **`topk=1` (single longest turn) scored 26% hit@5, mean rank 29.11 — dramatically worse than `outcome`'s 51-53%, and `topk=3` (three independent notes, never combined into one budget) scored 45% — better than `topk=1` but still short.** TITLE PROBE collapsed too (62% and 90% respectively, against a near-100% baseline). Root cause, once measured: `midpointOutcome`'s "last turn" isn't really about recency at all — it picks the assistant turn *immediately adjacent to the held-out probe*, which buys conversational proximity (shared vocabulary/topic with whatever query later tries to retrieve it). Ranking by length alone throws that proximity away in exchange for content that can be topically distant from the probe. Proximity-to-query beats size-of-content on this specific task. This does not resolve the original wandering-session observation, it just shows the LATER-PROBE metric measures something narrower than "does the note represent the session" — it measures "can a nearby, differently-worded query retrieve this note," and a thin-but-proximate note can win that while genuinely failing to represent the session. `outcome` remains the clear best measured shape; richness-based and multi-note zero-LLM capture are now both closed as production candidates, not just LLM-based `claims`.
- **`winze_recall`'s JSON response always reports `score: 0`.** Found while building the link-edge fixture above: `handleRecall` (`cmd/agent/mcp.go`) decodes `--hybrid`'s JSON into `queryHit`, whose `Score` field is tagged for `--hybrid`'s never-emitted `score` key — `--hybrid` only ever emits `rrf`/`lex_rank`/`sem_rank`. Production's dedup and link-suggestion logic (`checkDedup` → `nearestMemories`) is unaffected, since it queries `--semantic` directly and gets real cosine values; only the score a caller of `winze_recall` itself sees is dead. Not fixed here — out of scope for the milestone that found it — but worth knowing before trusting that field.
- Wikipedia provenance concentration (HHI 0.49, 65% Wikipedia). The availability-heuristic bias gate skips ZIM when this fires; Kagi fills the resulting signal gap. Structural fix still requires diversified ingest landing as actual cycles, not just plumbing.
- Survivorship bias structural, not prompt-tightness (191:2 irrelevant-to-challenged). Search-based sensors return supports + topic noise, not contradictions. The recalibrated resolver prompt classifies correctly; growing the challenged-count requires a counter-evidence query path the metabolism doesn't have yet.
- RSS feed curation unsolved — default topic feeds don't match entity-specific topology queries (0/131 signal historically). RSS still available via `--backend rss` for contributors who curate their own feeds.
- Source reputation not yet tracked — no automated mechanism for down-weighting historically low-quality domains based on calibration outcomes.
- **Prose claims about external dependencies rot with nothing checking them.** `internal/defndb`'s package doc asserted that defn was Dolt-backed and too heavy to link; that stopped being true and nothing noticed, and an agent later read it as a live measurement and built ~500 lines on top of it. `CodeRef` makes a doc→code reference fail the build when it goes stale, but `Symbol` is only checked when the store and the cited code share a module — it can't reach a client repo winze-agent serves, and never reaches non-Go code. `CodeRef` now has `Client`/`Span` fields for exactly that reach, checked by `cmd/lint --clients` (a content-hash check for non-Go targets, shipped 2026-09-02) rather than the compiler — see [docs/typed-citation.md](docs/typed-citation.md) for what that trades away. The narrower half — a bare undated claim, independent of whether it cites real code — is also now caught: `cmd/lint`'s dated-measurement rule flags a `Brief`, `Rationale`, or `Quote` that reads as measurement-shaped with no date (advisory, shipped 2026-09-02).
- **External review feedback, 2026-09-02 — two of eight items already resolved, five still open.** A sibling private project evaluating winze as its own decision record filed eight numbered findings (originally kept as a standalone `FEEDBACK-2026-09-02.md`; folded in here and the file removed 2026-09-08, since the two items it prompted already shipped and the rest belongs with the rest of this list, not in a separate untracked log). Two were the direct cause of the fixes in the bullet just above: the typed-citation module-boundary gap became `CodeRef.Client`/`Span`, and the dated-measurement gap became `cmd/lint`'s `dated-measurement` rule — both shipped the same day the feedback was filed. Five remain open, none yet built:
  1. `winze-agent init --link` sets `git config winze.store` but does not register the MCP server for the current project — a new adopter gets no `winze_remember` tool until someone runs that registration separately, and `docs/agent.md` doesn't currently say so.
  2. Two shapes for "decision" coexist unreconciled: `corpus/schema.go`'s bootstrap-era `Decision`/`FailureMode`/`Mitigation`/`OpenQuestion` types vs. `docs/decisions.md`'s actual model (a `Concept` with a `Supersedes` lifecycle). Either mark the four record types legacy with a pointer to `docs/decisions.md`, or give `winze_remember` a `--role Decision` that converges the two shapes.
  3. No generic per-predicate markdown projection exists (something like `winze-query --predicate X --md .`) for a caller who wants a hand-readable list of one predicate's claims — e.g. to generate a `BOUNDARY.md` from the store rather than maintain it beside it.
  4. `init`'s corpus-search fallback (`--from`, then `$WINZE_SRC`, then the `corpus/` beside the running binary — `docs/agent.md:146`) fails for a `go install`ed binary in `~/go/bin`, which never has a `corpus/` beside it. A module-cache lookup (`go list -m -f '{{.Dir}}' github.com/justinstimatze/winze`) or the build's own `vcs.*` info via `debug.ReadBuildInfo` would close it before giving up.
  5. No warning when a binary predates the checkout `--from` points at — a stale binary produces a build-gate error (e.g. a type mismatch a newer commit already auto-resolves) that reads as a real bug rather than a version-skew message.

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

### The real benchmark says winze isn't in the ballpark yet — measured 2026-09-08

Everything above reasoned about SOTA from architecture and a self-authored
dogfood harness. `docs/benchmark.md` already had the actual number and it
had gone unquoted here: on LongMemEval's oracle set (distractors removed,
the *easy* tier, `k=120`, 2026-08-07 sweep), winze scores **431/500** —
and a raw control that skips winze entirely and hands the chat history
straight to the answerer scores **443/500**. Winze currently loses to
doing nothing, on the cheap tier of its own chosen benchmark, by 12 net —
well outside the doc's own ~10-question noise floor. The full
`longmemeval_s` haystack (the actual hard test, with distractors) has
never been scored at all; it prices at ~$138 and ~5.6h at concurrency 8,
and paying for it before clearing the raw control on oracle would just be
buying a confirmation of the same result for free money.

It isn't a uniform loss. Per-type, winze *beats* the raw control on
knowledge-update (71/78 vs 67/78) and temporal (121/133 vs 115/133) — where
typed structure and supersession help — and loses badly on assistant-recall
(39/56 vs 53/56) and multi-session (107/133 vs up to 133), where the answer
is a verbatim quote or scattered raw text a typed `Fact` representation is
lossy for. Of the multi-session failures where every needed fact was
*already* in the answerer's context, 0-1 of 16 recover at any window
size — an extraction-quality problem, not a retrieval one, per the doc's
own `k`-sweep analysis.

A third costrel consult (fable) on "given the note-shape axis is plateaued,
what's next" landed on: **drop "close the authoring-gap" as a spending
axis entirely.** Not because two dogfood-harness attempts (the retired
raw-evidence tier, today's `claims` multi-unit test) both tied rather than
beat their baselines — that alone wouldn't settle it — but because winning
that axis is unclaimable under a cheap/fast/scalable constraint regardless:
the field's own yardstick for "closed the gap" is a LoCoMo/LongMemEval
score, and this doc's own survey already ruled porting those numbers onto
a single-user deployment "a fabricated comparison, not a measurement." A
dogfood-harness number moving has never been able to support that claim
either way. The oracle-set number above makes the same point more sharply
without needing the argument: winze doesn't need a philosophical reason to
stop claiming SOTA on retrieval — it's currently behind a dumb baseline on
the actual benchmark, measured.

Proposed replacement, not yet built: a **revision/temporal probe** on the
existing self-recall harness — a fact stated in session N, updated in
session N+j, probed at N+k, checking whether recall returns the current
version and beats naive most-recent-match. Targets the one thing the real
benchmark already confirms is winze's actual edge (knowledge-update,
temporal), rather than the axis (verbatim/multi-session) where a raw dump
keeps winning. The typed `Supersedes` graph this would exercise has never
been measured this way.

**Corrected from an earlier draft of this session's own conversation, not
this doc:** multi-writer shared memory is not a hypothetical winze doesn't
have proof of yet. It's running now: a separate real project shares one
winze store across 11 concurrent git worktrees, 58 real commits across
distinct work threads — confirmed directly 2026-09-08, not a config stub,
and not the Gas Town citation `docs/multi-session-write-shape.md` used to
carry (that integration was dropped 2026-07-22 and never actually
exercised this path; see that doc for the corrected citation, kept
deliberately without the other project's internal paths/codenames/ticket
IDs — none of that specificity is load-bearing for the claim). None of the
five systems in this section's own survey (Letta, mem0, Claude Code's own
memory, Anthropic's tool, Cursor/Windsurf) are multi-writer at all, so this
is a real, running, currently-uncontested differentiator — with one honest
limit: every commit in that store carries the same git author, so
concurrent *sessions* writing safely is demonstrated, concurrent *distinct
human contributors* is not, yet.

**Where this leaves priority, given both findings together:** the
highest-leverage next work is fixing the diagnosed oracle-set gap
(assistant-recall's verbatim-quote representation, the multi-session
extraction-quality cohort that already had every fact in context and
still lost) — a real, measured deficit with a named root cause, not
another dogfood-harness note-shape tweak and not the expensive full-
haystack run yet. The multiplayer angle is a second, separate, real
differentiator worth writing up on its own terms; it doesn't compete with
the extraction fix for the same next slot of work.

### The rerank-vs-Eywa tension, resolved without a consult — 2026-09-08

Before the extraction-gap work started, a real tension surfaced and got
resolved directly rather than dispatched: the survey above (2026-08-31,
expanded 09-07) names Eywa's "zero LLM calls inside retrieval" as
concretely ahead of winze. The same week, independently, winze shipped an
LLM listwise reranker into `winze_recall`'s default path (`6e800fa`,
`dde0c9c`, `72163ec`, `f634aa7`) for a large measured win — LATER-PROBE
hit@5 14%→34%, later 51% with richer note-shapes. Two decisions pointing
opposite directions, never weighed against each other in one place.

Reconciled: Eywa's abstract frames the payoff as retrieval and
answer-generation being decoupled, so the same retrieved set holds
regardless of which model answers from it. Winze's rerank already has
that property — it runs once, before any answer model sees anything, and
doesn't vary by which model is chosen downstream. What it does *not* have
is bit-for-bit determinism of the ranking call itself (`Temperature: 0`
already documented above as not fully deterministic on the Anthropic
API). That's a real but much smaller gap than "swappable answer models,"
and not one worth trading a measured 14%→51% hit@5 gain to close.

The doc's own stated reason for an Anthropic call over a local one —
"Ollama has no rerank endpoint and Cohere/Voyage have no self-host
option" — is true but not the full search: Hugging Face's
`text-embeddings-inference` self-hosts cross-encoder rerank models
(e.g. `bge-reranker`) behind a `/rerank` endpoint, no Ollama involved.
**Unverified, recalled from training, not checked against their repo/docs
this session** — flagged here so a future session doesn't have to
re-derive it from scratch, not presented as confirmed. Worth a real A/B
only if a concrete reason to want local/deterministic reranking shows up
(cost at scale, offline operation, killing the fail-open-on-API-down
path) — none of which currently apply to a single-user store. Until then:
park this, don't chase zero-LLM-retrieval as a goal in itself, and the
extraction-gap fix stands as the next work, unchanged from the verdict
above.

### The extraction-gap fix, measured — 2026-09-08

Acted on the priority above rather than reasoning from the aggregate:
re-ran the exact slice the gap lives in (56 assistant-recall + 133
multi-session, `k=120`) fresh, read the real failing question/gold/answer
triples instead of the fact counts, and found two genuinely separate,
independently fixable defects — not one diffuse "extraction is bad."

**Assistant-recall (18 of 56 failures, 11 of them zero-fact extractions):**
the lens's own worked examples for "specifics the assistant supplied" were
all personal-recommendation shaped — "a venue, a product, a colour." The
failing sessions were about something else entirely: a novel's plot (the
Library of Babel), a paper's sample size, a chess move, a historical
cartoon, a technical paper's framerate number — none about the user's own
life, all a concrete, nameable detail the assistant stated. Neither
`lensSystem` rule 2 nor `lensRetrySystem` rule 1 named "explained a fact
from an article" or "produced content on request" as the same kind of
specific, so the lens read these sessions as having nothing worth keeping.
Fixed by broadening both rules explicitly (`lensVersion` v9→v10,
`cmd/longmemeval/lens.go`).

**Multi-session (25 of 133 failures, mostly the "everything was already in
context and still lost" cohort):** most are genuine miscounts, unresolved
cross-session duplicates, and stale-vs-updated value confusion — a harder,
still-open population. But a real, separate, mechanical defect sat in
`judgeSystem`'s rubric: "INCORRECT if... it says it doesn't know," with no
carve-out for a question whose *gold answer itself* is an abstention ("the
information provided is not enough"). Measured: 8 of 12 abstention
questions passed anyway (the judge is an LLM applying the rubric with some
independent judgment, not a literal string match), but at least one
(`eeda8a6d_abs`) was a clean case of a correct abstention marked wrong.
Also fixed: a gold answer offering more than one acceptable value
("Pilsner or Lager") now accepts either — a real failure, `16c90bf4`,
named only "Pilsner" and was marked incorrect.

**Net effect, same 189-question set, cold, before vs. after
(`cmd/longmemeval/baselines/v10-k120-asst-multi.jsonl`):** 146/189 (77%)
→ 164/189 (87%). Per-type: assistant-recall 38/56→52/56 (zero-fact count
11→2), multi-session 108/133→112/133, abstention questions 8/12→9/12.
Question-level: 25 recovered, 7 regressed — net +18, checked individually
rather than just netted away. Six of the seven regressions read as
ordinary run-to-run answerer/extraction variance, the same non-determinism
already documented elsewhere on this page (`Temperature: 0` is not
bit-identical on the Anthropic API); one (`3249768e`) is a real, narrow
side effect of broadening extraction — a second, unrelated enumerated list
in the same session got captured this time, and the answerer picked the
wrong one. Worth watching if the pattern recurs, not worth reverting the
fix over a single case against 25 recoveries.

**Not yet done:** re-measuring at `k=120` on the full 500-question oracle
set. This fix was diagnosed and measured on the assistant+multi-session
189-question slice only; knowledge-update, temporal, single-user, and
preference were untouched by these prompt changes and should be
unaffected, but that is an assumption, not yet checked. The number that
actually answers whether winze closed *the* gap this section opened with
is the full-500 re-run against the raw-context-dump control (431/500 vs
443/500) — not yet run.

### The full-500 re-run, fair comparison — 2026-09-08

Both sides re-run under the current code (`k=120`, concurrency 4, same
fixed `judgeSystem` on both — the old 443/500 control number was scored
under the buggy rubric too, so it needed re-scoring, not just winze):
`cmd/longmemeval/baselines/v10-k120-full500.jsonl` (winze) and
`v10-raw-control-full500.jsonl` (control).

**winze: 444/500 (88.8%). Raw control: 452/500 (90.4%).** Still losing,
by a narrower margin than before (net −8 vs. the old −12) — winze is not
past this line yet. Per-type, winze vs. control: knowledge-update
72/78 vs 66/78 (+6), single-user 69/70 vs 67/70 (+2), preference 28/30 vs
27/30 (+1), multi-session 113/133 vs 114/133 (≈even), assistant-recall
51/56 vs 56/56 (−5, the control's ceiling — raw transcript access beats a
`Fact` representation on verbatim recall by construction), **temporal
111/133 vs 122/133 (−11)**.

**The temporal number is a reversal worth flagging plainly, not
attributing to today's fix.** The 2026-08-07 sweep had winze *beating*
the control on temporal, 121/133 vs 115/133 — a documented edge. Today
that edge is gone and reversed. Checked before writing this down: the
harness's retrieval (`syncAndRetrieve`/`rankFacts`) is plain term-overlap
ranking, unchanged, and does **not** go through `winze_recall`'s LLM
reranker at all — that's a different tool on a different code path, so
the reranker shipped this week is not a candidate explanation. Reading
the actual failures: almost all are answerer-side date arithmetic ("Nov
29 − Nov 15 = 14 days" when the source says "a week before," multi-step
relative-date subtraction, chronological-ordering mistakes across many
facts) — not extraction gaps shaped like anything today's lens change
touched. But the comparison against the 2026-08-07 baseline is confounded
by every lens version between then and now (v7→v10, not just today's
bump), so **this is not a clean isolated measurement of today's fix** —
only the 189-question assistant+multi-session result above is. Whether
temporal got worse specifically *because* of v10, or was already this
weak before v10 and the 2026-08-07 number reflected a different, since-
changed pipeline, is an open question. Next step, not yet done: hold
today's code fixed and check whether reverting just `lensVersion` to v9
changes the temporal score — the controlled test this finding needs
before assigning it a cause.

### The temporal regression, root-caused and fixed — 2026-09-08

Ran the controlled test named above. Swapped `cmd/longmemeval/lens.go`
back to its pre-v10 content (`git show 22fa3a5:cmd/longmemeval/lens.go`),
built a separate binary, ran the same 133 temporal questions, everything
else held fixed. **v9: 122/133 — matches the 2026-08-07 baseline almost
exactly. v10: 111/133.** Paired on the identical questions: 12 regressed,
1 improved. This confirms it, not just implicates it: today's v10 change
caused the temporal regression, full stop, and the earlier "confounded by
every version since 2026-08-07" hedge no longer applies to this specific
finding.

Mechanism, read from the actual extractions: v10's broadened rule 2 ran
in the **primary** lens pass, which fires on every session — including
temporal ones that already had real dated-event facts. So a temporal
session now also picked up tangential content (recommendations,
informational asides) it never used to, and that competed with the real
events for the same `k=120` retrieval window. Regressed questions show
materially more extracted facts under v10 (61→90, 71→96, 130→200 in
three examples) with the wrong specific event surfacing in the answer.

The fix (`lensVersion` v10→v11) is narrower than reverting v10 outright:
revert the **primary** pass's rule 2 to its v9 wording, keep the **retry**
pass's broadening. This isn't a guess — checked directly: of the 18
original assistant-recall failures, 15 were rescued by `lensRetrySystem`
specifically, which only fires when the primary pass returns nothing. A
temporal session with real facts never reaches the retry path, so it was
never the source of the assistant-recall gain and never needed the
broadening it was paying for. Measured on the same 322 non-trivial
questions (temporal + multi-session + assistant-recall), v11 vs v10:
temporal 111→118 (nearly back to 122), assistant-recall held exactly at
51/56, multi-session 113→111 (inside the noise floor).

**The full-500 re-run landed: winze 447/500 (89.4%)** vs. the same raw
control from before, 452/500 (90.4% — control doesn't touch extraction
at all, so it didn't need re-running under v11). This supersedes the
v10 full-500 line above. The gap has now narrowed three times running:
−12 (pre-fix) → −8 (v10) → **−5 (v11)**. Per-type, winze vs. control:
knowledge-update 73/78 vs 66/78 (+7), single-user 68/70 vs 67/70 (+1),
temporal 120/133 vs 122/133 (−2, recovered from v10's −11), multi-session
111/133 vs 114/133 (−3), assistant-recall 51/56 vs 56/56 (−5, the
structural ceiling — a raw transcript beats a compressed `Fact` on
verbatim recall by construction), single-session-preference 24/30 vs
27/30 (−3).

One new, small, honestly-unexplained wobble: preference went 28/30
under v10 to 24/30 under v11 — a 4-question drop on a 30-question type,
which clears this doc's own "only trust a move of three or more" bar but
wasn't chased down tonight. Rule 2's classic worked example
("recommended the Hotel Meridien in Lyon") is preference-shaped, so
narrowing rule 2 back to v9 wording plausibly touches it, but that's an
inference, not a checked mechanism — logged here for whoever picks this
up next rather than run to ground in an already-long session.

**The methodology question this raises, and the user's own answer to
it:** should every `lensVersion` bump require a full-500 regression check
before shipping, given a targeted fix silently broke a different question
type six versions running and nothing caught it until today? Decided:
not a blocking gate on every targeted fix — that's overkill for a cheap,
narrow rediagnosis — but run the full 500 more often than "only when
something feels off," which is the cadence that let this one ride for
however many versions it actually rode for (still unknown; v7 through v9
were never checked against the full set either).

### What the field's numbers actually look like, and a cautionary tale — 2026-09-08

Corrected a too-hasty finding from earlier today: a research pass first
concluded no LongMemEval leaderboard exists at all (checked paperswithcode
and the official GitHub repo, both genuinely empty of one). Pushed on
directly — a real, if messy, *vendor self-comparison* culture exists that
those two sources don't surface.

**mem0's own 2026 leaderboard post** (mem0.ai/blog/ai-memory-benchmarks-in-2026,
curled directly) publishes a table — ByteRover 92.8%, Mem0 94.4%, Zep
71.2% — and the post itself is honest about what the table is worth:
"None of these numbers were generated using the same model stack, judge
model, or retrieval configuration... treat this table as a starting
point, not a settled ranking." Mem0's own per-category LongMemEval
breakdown: single-session-user 98.6, single-session-assistant 98.2,
knowledge-update 93.6, multi-session 88.0 — temporal and preference
aren't reported at all. Confirmed from their own text this is scored on
the full `longmemeval_s` haystack (~40 sessions/user, ~115K tokens), not
the oracle set.

**OMEGA** (omegamax.co/benchmarks, another vendor's own page) reports
95.4% "task-averaged" but 466/500 raw (93.2%) — the two numbers disagree
because task-averaging weights a 30-question category equally with a
133-question one, worth knowing before quoting either figure alone.
Per-category, same haystack tier: single-session-recall 125/126,
preference 30/30, multi-session 111/133 (83%), knowledge-update 75/78
(96%), temporal 125/133 (94%). Their own comparison table lists Mastra
94.9%, Emergence AI 86%, Zep 71.2%, and marks Mem0/Letta "N/A" — disputing
mem0's self-reported number, not confirming it. Two vendors, two
self-published tables, disagreeing with each other about a third vendor's
score — that is the actual state of the field's "leaderboard."

**The number that matters for winze: every one of these is scored on the
full haystack. Winze has only ever scored the oracle set** (distractors
removed — see "The real benchmark says winze isn't in the ballpark yet"
above). The original paper's own Figure 3(b) shows oracle scores drop
~30% relative to full-haystack for the same system. So winze's 444/500
oracle number isn't just behind these — it isn't eligible for this
comparison at all yet. Winze has never attempted the harder test. That is
the honest headline, not a caveat to soften.

**MemPalace, read as a cautionary tale on purpose, not a drive-by
mention.** An independent critique (arXiv 2604.21284, "Spatial Metaphors
for LLM Memory," full PDF read) covers a system that launched April 2026,
hit 47,900 GitHub stars in two weeks, and claimed 96.6% Recall@5 on
LongMemEval — "higher than any extraction-based competitor" — attributed
to its "method of loci" spatial architecture (Wings→Rooms→Closets→Drawers).
An independent audit (GitHub Issue #29, dial481) found: the 96.6% is the
performance of **ChromaDB's stock all-MiniLM-L6-v2 embedding model on
verbatim text — reproducible with a minimal ChromaDB setup, no palace
structure required at all.** Recall@5 also isn't answer accuracy; real
end-to-end QA accuracy was ~67.2%, a huge gap from the marketed number.
Five more claims fell the same way on inspection: a "100%" LongMemEval
score hid undisclosed iterative LLM reranking; a "100%" LoCoMo score used
k=50 (functionally the whole conversation handed back); "30x compression,
zero information loss" was lossy summarization; "semantic contradiction
detection" was exact-match dedup; a "+34% boost" was ordinary metadata
filtering under a new name. The maintainer acknowledged all six points
and retired the disputed numbers — the right ending, but only after
external audit pressure forced it, not before.

**The lesson to actually hold onto, not just note:** MemPalace's failure
wasn't fabrication — every debunked number came from a real run — it was
publishing a headline metric (a generous retrieval score) standing in
for a harder one (answer accuracy) without saying so, and crediting a
fancy narrative (the spatial metaphor) for what a boring, standard
component (an off-the-shelf embedding model) was actually doing. That is
a structural risk this project has brushed against more than once this
same session — the 57%/52%/47.5% figures that turned out to be a
hardcoded result-cap bug, the raw-tier "statistically indistinguishable"
comparison that was measured hours before the fix for that same bug. The
difference so far is that winze's numbers get re-verified and corrected
in this document when the gap is found, not left standing until an
outsider audits them. Keeping that difference real — publishing the
caveat *with* the number the first time, not after being asked, and
periodically checking whether a measured win (knowledge-update, temporal)
is really coming from the typed/provenance architecture or from
something a plain baseline would also get right — is the actual, ongoing
work this cautionary tale asks for. Not a one-time note that MemPalace
was overstated.

### Full failure-review pass on the oracle set, and the answerer fix it produced — 2026-09-08

New standing practice, decided this session: analyze every failure on a
full-500 run, not a sample — "it's just not that many," and a sampled
read already missed a real mechanism earlier tonight. A fork read all 53
losses on `v11-k120-full500.jsonl` and classified each one: 24 extraction
misses, 4 truncation, 19 answerer reasoning errors on facts that were
already correctly retrieved, 3 cross-session duplicate/conflict, 3
judge-or-gold artifacts. The 19 were the standout — no re-extraction
needed, four repeatable, independently-nameable shapes:

1. **Premise-mismatch answered as if true** (4 failures) — `a96c20ee_abs`
   answered "Harvard University" for a poster presentation no fact
   mentions. A direct violation of the existing "do not invent" rule,
   just never named for this specific shape.
2. **Refusal when a missing piece is legitimately zero, not unknown**
   (2) — `7024f17c` refused to total jogging+yoga hours because yoga was
   never logged that week, when "never logged" means zero contribution.
3. **Date-arithmetic errors** (6) — `982b5123` conflated "booked three
   months in advance" with "how many months ago," a different quantity
   built from the same two dates.
4. **Preference under-application** (4) — `54026fce` drew on one stored
   preference and ignored the rest of what was retrieved.

Checked and dropped as a candidate for this batch: whether the answerer
uses a fact's verbatim `Quote` over its paraphrased `Value` when a
question asks for exact wording. `Quote` is already in the prompt
(`answer()`'s `%q` formatting) but `answerSystem` never told the model to
prefer it — a real gap, just not the one explaining any of these 53. Every
remaining assistant-recall failure was upstream of that choice (the fact
was never extracted at all, in either field).

**Fixed `answerSystem` with four new rules, one per shape above.** Tested
narrow first: 16 named failures, `--only`, no re-extraction. 7 of 16
flipped, all 3 pure date-arithmetic cases among them. Two things surfaced
in that same narrow run, not papered over: `gpt4_93159ced_abs` correctly
named its premise mismatch ("your employer is NovaTech, not Google") and
then answered a hypothetical anyway — the rule said to name the mismatch,
never said to stop there. Fixed with one more line. `a96c20ee_abs`
(Harvard) still hallucinates despite the rule naming this exact shape —
left open; not every instance of this pattern is closable by prompt
instruction alone.

**Full-500 re-run, both sides** (`answerSystem` is shared by `answer()`
and `answerRaw()`, so the raw control needed re-scoring too, same as the
judge fix earlier): **winze 450/500 (90%), raw control 455/500 (91%).**
Both sides gained exactly 3 questions — the fix improved answerer
reasoning generically, which helps a raw-transcript answerer exactly as
much as a typed-fact one. The absolute numbers are real; **the relative
gap did not move, still −5.** Per-type, winze vs. control: knowledge-
update 74/78 vs 69/78 (+5), single-user 66/70 vs 67/70 (−1), preference
28/30 vs 23/30 (+5 — the unexplained wobble logged two sections up is
resolved, in winze's favor, plausibly by the new "apply every preference
fact" rule, though that's inference not a checked mechanism), temporal
118/133 vs 121/133 (−3), assistant-recall 50/56 vs 55/56 (−5, unchanged —
the structural ceiling), multi-session 114/133 vs 120/133 (−6, worse
than before — the raw control gained more from this fix here than winze
did, not chased down tonight).

**Two standing practices from tonight, both the user's own call:** run
the full 500 more often, not just when something feels off — a targeted
fix silently broke temporal for however many `lensVersion` bumps it
actually rode, unnoticed. And always read every failure on a full run,
not a sample — the sample-based reads earlier tonight missed the
Quote-vs-Value question entirely and would have missed the premise-
mismatch pattern's actual frequency.

**Filed separately:** three concrete defn usability papercuts hit while
doing all of the above (`code`'s cache-hit response ignoring `full:true`
when the cached response already was full; `sync` rejecting a directory
`overview` itself accepts; `search` silently treating a regex-shaped
pattern as a dead literal with no "not regex" hint) — dropped as
`FEEDBACK-winze-session-2026-09-08.md` in defn's own project directory,
per the standing cross-project feedback convention.

### Two fixes tried on multi-session and preference, both reverted — 2026-09-08

The 450/500 above split unevenly: multi-session sat at 114/133 against raw
control's 120/133 (-6, the widest per-type gap), preference already led
control 28/30 to 23/30 (+5, not a gap). Read the 19 multi-session losses
against real session text before touching anything, following tonight's
own standing practice. Ten of them were questions a raw-transcript control
got right that winze did not — meaning the needed fact was in the session,
never reached the retrieved-facts pipeline. Three confirmed directly: a
5-gallon betta tank named only in a "by the way" aside inside a session
about a *different* tank's nitrite levels (`46a3abf7`, gold 3 tanks, winze
saw 2); a BBQ party at a friend's place mentioned in passing while planning
an unrelated potluck (`60159905`, gold three dinner parties, winze saw
two); a Jimmy Choo heels' $500 retail price dropped as an aside in a
session about affordable fashion brands, sessions away from the $200
purchase price it needed to pair with (`bb7c3b45`, gold $300 saved, winze
had no retail price to subtract from). All three: a single-sentence pivot
into a fact the surrounding session isn't about, missed because the lens
reads for a session's dominant topic rather than every turn.

**Fix 1 — lens rule 1a, told the primary extraction pass to read every turn
for these asides.** A 21-question targeted `--only` re-run flipped 7 of 19
named failures to correct, all three of the above included — the mechanism
was real. The full-500 re-run told a different story: multi-session 114/133
-> 113/133, preference 28/30 -> 26/30. Both **worse**, not better, on
exactly the categories this targeted. Fact counts explain why: they rose
on nearly every one of the eight NEW regressions this introduced (110->130,
112->124, 157->173...), several crossing the k=120 retrieval cap for the
first time, and the newly-competing facts produced their own wrong answers
— an extra "fourth road trip" the answerer couldn't reconcile with a stated
three-trip total, a grapefruit garnish mention double-counted as a cocktail
ingredient, a preference session whose extraction came back with 1 fact
instead of 15 for reasons that don't trace to the rule's own wording and
looks like plain non-determinism instead. This is the identical mechanism
that broke temporal under lens v10 back in the first half of this same
session: more real, correctly-extracted facts still compete for the same
fixed k=120 window, so "the model missed a real fact" is not by itself a
reason to extract more of them — reverted, full account in `lens.go`'s
`lensVersion` changelog.

**Fix 2 — two answerSystem rules**, tried independent of the lens change:
scoping "the most recent value wins" to require the two facts describe the
same context (aimed at `a4996e51`, 45 vs gold 50 — the rule had fired
across an unrelated "some weeks go up to 45" aside and a specific "peak
season +10 hours" fact as if one updated the other), and a rule against
substituting outside/general knowledge for a missing specific number
(aimed at `09ba9854`/`09ba9854_abs`, which fabricate plausible Narita
airport transit fares instead of saying I don't know). Isolated on v11's
lens — warm cache, byte-identical facts to the 450/500 run — the two rules
alone took the full 500 to **441/500**. Every type went flat or down:
knowledge-update 74->71, multi-session 114->113, preference 28->25,
temporal 118->116, assistant and single-user unchanged. 19 questions
flipped from correct to wrong against only 10 the other way, and unlike
the lens regression, there is no one clean mechanism behind the 19 — some
read as ordinary judge/sampling variance on answers no worse in substance
(`38146c39`'s reworded but equivalent turbinado-sugar suggestion flipped to
wrong with no visible content difference), and at least one genuinely
changed shape for the worse in a way neither new rule explains directly
(`1c0ddc50`'s answer started restating retrieved true-crime/self-
improvement facts as generic options, exactly what the gold answer says
the user does not want). Reverted; full account in `answerSystem`'s own
changelog.

**Both baselines are in git** (`v12-lens1a-attempt-full500.jsonl`,
`v11-lens-answersys-attempt-full500.jsonl`) for the per-question record,
per the baselines README.

**The honest total: multi-session and preference are exactly where they
were at 450/500** — 114/133 and 28/30, gap -6 and +5 respectively. Two
plausible, independently-reasoned fixes, each validated on a narrow slice
before being trusted with real API spend, both lost on the full 500 for
different reasons. This is the same lesson lens v10 already taught,
applying again to a different rule and a different mechanism: a change
that flips its own named failures under a targeted check is not evidence
it helps, and the full 500 has been the only number in this file that's
ever told the truth. What's left unresolved from this pass: 10 of the 19
multi-session losses are still real, still confirmed against source text,
still winze-specific — the aside-extraction problem is real, a narrow
lens-prompt fix for it just isn't.

### The multi-session k increase, tried and closed — 2026-09-08

Named above as the next thing to try: a larger retrieval window specifically
for multi-session questions, leaving `k=120` for everything else so no other
type pays for it. Added `-k-multi` (`main.go`) — 0 by default, overrides `-k`
only when `q.QuestionType == "multi-session"` — so this could be tested
without repeating lens v10/rule 1a's mistake of a change that touches every
type at once.

Ran the 133 multi-session questions alone at `k=200` and `k=300` against
today's pipeline (v11 lens, warm cache — extraction is `k`-independent, so
this cost answer+judge only). **112/133 and 111/133, both worse than the
114/133 reference at `k=120`.** This replicates `docs/benchmark.md`'s
2026-08-07 finding — recovery non-monotone past ~120, extra slots
displacing useful facts rather than adding capacity — on today's extraction,
isolated so there's no possible confound from other types riding along.
The ceiling here was never capacity; it's the term-overlap ranker admitting
lower-relevance facts ahead of the one that matters as the window grows.
More `k` cannot fix a ranking problem, scoped or not.

**Closed, not open.** `-k-multi` stays in the code, off by default — real,
harmless tooling even though the hypothesis it was built to test failed.
The actual fix for the 10 confirmed aside-extraction losses is a genuine
retrieval-time or ranking-time change (surface an aside-shaped fact higher
when a session's main-topic facts are already well covered), not a window-
size knob in either direction. Not attempted tonight.

### The ranking-time fix, checked and closed before it cost a full run — 2026-09-08

Named above as the next real lever. Before writing it, checked which multi-
session losses the k=120 cutoff could possibly explain: across the full 500,
only 18 questions ever have `facts > retrieved` at all (k genuinely binds),
7 of those wrong. Of the three multi-session ones in that 7, two turned out
to be answerer failures with nothing to do with ranking — `46a3abf7` (tanks)
and `60159905` (dinner parties) both have their needed fact sitting inside
the retrieved set with room to spare (57/57 and 94/94, no truncation at
all); the model just didn't use it. `bb7c3b45` (Jimmy Choo) is a plain
extraction gap — no fact anywhere in the 32 extracted mentions a retail
price. Confirmed directly against the warm v11 extraction cache with a
throwaway test, not by re-reading the prose account of these three from
earlier tonight, which had assumed retrieval was where they broke.

That left exactly one confirmed case where a real fact was extracted and cut
by rank: `bf659f65` ("how many albums or EPs have I purchased"), 143 facts
extracted, the one fact describing an actual EP purchase
(`whiskey_wanderers_ep_midnight_sky`) ranked #128, eight facts past the
k=120 cutoff. Wrote a same-session marginal-relevance rerank — discount a
candidate by how much of its own vocabulary is already covered by facts
already picked from its own session, so a ninth near-duplicate stops
crowding out something that says something new — scoped to fire only when
`facts > k`, a strict no-op on the 482/500 questions that never truncate.

Checked it against the one case it was built for before running anything:
it can't fix this one. The EP-purchase fact scores exactly **0** against the
question — none of its terms (`whiskey`, `wanderers`, `ep`, `midnight`,
`sky`, `bought`, `festival`, `merchandise`, `booth`) overlap the question's
(`how`, `many`, `music`, `albums`, `eps`, `purchased`, `downloaded`); `ep`
vs `eps` is an exact-token near-miss with zero credit. The 80 facts that
outrank it mostly score exactly 1, for one reason: this session's lens
output happened to name most of its attributes with a `music_` prefix
(`music_review_tip_8`, `music_recommendation_3`, ...), so "music" alone
buys a point regardless of relevance — 77 of 143 facts share it. A
same-session redundancy discount can only ever roughly halve a positive
score; it cannot manufacture credit for a fact that shares zero literal
tokens with the question. No discount curve rescues a true zero sitting
below 80 unrelated ones scored higher by an accident of naming.

The real fix for this exact case is a scoring-function change — normalize
plurals so `eps`/`ep` and `albums`/`album` match, or move off exact-token
overlap toward something that credits meaning over spelling — not a
redistribution of a fixed ranking. That change touches `terms()`, which
every question of every type scores through, not just the 18 that ever
truncate — a strictly larger blast radius than either of the two changes
already reverted tonight, for a confirmed population of one question out of
500. Reverted the rerank (`store.go` diffs clean to the pre-session byte),
deleted the throwaway diagnostic test. Not worth a full-500 run to confirm
what the cached-extraction check already showed for free: this lever's
real, addressable population is smaller than the noise floor, and the fix
that would actually reach it is broader than the two blanket changes this
session already measured as net losses. Multi-session and preference stand
exactly where the night started: 114/133 and 28/30.

### The assistant-recall retry-prompt fix, tried and reverted — 2026-09-08

The one gap left after the ranking-time close above: `single-session-
assistant` sits at 50/56 against the reference control's 55/56. Read all 6
real losses against actual session text and extracted facts (`e3fc4d6e`,
`352ab8bd`, `18dcd5a5`, `dc439ea3`, `c8f1aeed`, `16c90bf4`), each a distinct
mechanism. Two looked reachable without a primary-lens change:
`352ab8bd` (a first-turn paper review, buried under revision requests that
followed) and `e3fc4d6e` (facts stated in pasted article text, not the
model's own prose) both zero-fact on the primary pass, both routed through
`lensRetrySystem`. Added two rules there — pasted/quoted source material
counts as fact-bearing on its own (1a), and a session's first substantive
answer isn't displaced by a long tail of near-identical revisions after it
(1b) — and bumped `lensVersion` `v11`→`v13`. Scoped to the retry prompt
specifically because this session had already burned two primary-lens
changes (v10, the reverted "v12" rule) on exactly this kind of regression.

Narrow check on the 6 target qids looked clean: `18dcd5a5` flipped to
correct, `e3fc4d6e` went from 0 to 5 extracted facts (still wrong, but
moving), `352ab8bd` unchanged, the other three untouched as expected (they
need a primary-lens change, not attempted). Established first that retry
touches roughly 50 of 500 questions dataset-wide, so a full run was run
before shipping rather than trusting the narrow six.

**Full 500 came back 444/500, down from the v11 reference's 450.**
Per-type against the v11 baseline (`cmd/longmemeval/baselines/v11-answersys-k120-full500.jsonl`):

| type | v11 | v13 |
|---|---|---|
| knowledge-update | 74/78 | 72/78 |
| multi-session | 114/133 | 117/133 |
| single-session-assistant | **50/56** | **48/56** |
| single-session-preference | 28/30 | 25/30 |
| single-session-user | 66/70 | 66/70 |
| temporal-reasoning | 118/133 | 116/133 |

The one type this was built to fix moved backward. Diffed qid-by-qid
against v11 (28 flips: 17 to wrong, 11 to correct) and split by whether the
question's session actually routed through retry, using the per-question
log lines:

- **Retry-exposed flips, net −2**: 4 down (`1568498a`, `ceb54acb`,
  `eaca4986`, `5a4f22c0`), 2 up (`18dcd5a5`, `ec81a493`). `ceb54acb` is the
  clearest case — under v11's retry prompt it extracted 5 facts and
  answered correctly; under v13's it extracted **0** and the answerer got
  nothing. `1568498a` similarly dropped 4 facts to 2. Both previously-
  working retry extractions, broken by the new rules, and neither is one of
  the 6 sessions the rules were written for. `5a4f22c0` shows the same
  regression reaching outside the target type entirely, into
  knowledge-update.
- **Non-retry flips, net −4**: 13 down, 9 up, none of them touching the
  retry pass at all — pure primary-pass Haiku extraction landing
  differently between runs. `lensVersion`'s cache key covers the whole
  extraction (`sha256(lensVersion + model + sessionBody)`), so bumping it
  busts and cold-reruns every session's primary extraction too, not just
  the ~50 that ever reach retry. Haiku isn't perfectly deterministic on
  identical primary-prompt text, and that alone reshuffled the k=120
  ranking window on unrelated questions across knowledge-update,
  preference, temporal, and multi-session in both directions.

Two findings worth keeping past this one experiment:

1. **"Retry-only is safe" was half right.** It's true a retry-only change
   can't compete with an already-correct primary-pass session for the
   k=120 window — that part held. What it doesn't protect against: some
   retry-touched sessions were *already succeeding* under the old retry
   prompt, and a retry-prompt change can regress those exactly the way a
   primary-lens change regresses primary successes — same mechanism, just
   bounded to the retry population (~50 questions) instead of all 500.
   `ceb54acb` is that failure mode, not a coincidence.
2. **A `lensVersion` bump is not a clean A/B for a retry-only prompt
   change.** The shared-version cache key means every full-500 validation
   after a version bump carries primary-pass re-extraction noise as a
   baseline cost — measured here at −4 net, larger than the −2 the actual
   rule change cost within its own scope. A future retry-only or otherwise
   narrowly-scoped lens change needs either a repeat-run noise floor
   (re-run the unchanged baseline once more under a fresh bust) or a
   version key that can invalidate the retry path alone, before a single
   full-500 run can be trusted to isolate the real effect.

Reverted `lens.go` (`git checkout`, confirmed byte-clean diff against
`HEAD`, `code(op:"sync")` to reconcile the graph, build gate clean).
`single-session-assistant` stands at 50/56. `dc439ea3`, `c8f1aeed`, and
`16c90bf4` remain the open losses, all needing a primary-lens change this
session has now measured three separate times (v10, "v12", and the retry-
population regression above) as more likely to cost than to gain — not
attempted tonight.

### The primary-lens fix for the last three assistant-recall losses, shipped — 2026-09-09

Read all three remaining losses against real session text and a warm-cache
extraction dump. Three distinct mechanisms, not one diffuse gap:

- `dc439ea3` (gold: "Hoop Dance") — the session's *first* assistant turn is a
  numbered 1-7 list of traditional powwow games, item 7 being Hoop Dance.
  Zero facts extracted from it. Two nearly-identical numbered lists *later*
  in the same session — venue recommendations, packing tips — extracted one
  fact per item perfectly. The only difference: those are framed as
  "recommendations"/"tips"; the games list answers a plain factual question.
- `16c90bf4` (gold: "Pilsner or Lager") — the assistant says "a pilsner or
  lager would work well." The lens captured `beer_type = "Pilsner"`, sourced
  from the *user's* next turn ("I'll try this with a Pilsner"), losing
  "lager" and attributing the fact to the wrong turn entirely.
- `c8f1aeed` (gold: "Pennsylvania") — the assistant names Pennsylvania as the
  example state, inside an explanatory paragraph about EPA/state
  groundwater-monitoring rules. Zero assistant facts extracted from the
  whole session; only the user's own stated opinions got captured.

`c8f1aeed`'s mechanism — a concrete nameable detail buried in unstructured
explanatory prose, not a list, not a recommendation — is the exact shape
`v10` already broadened for and got reverted over: a confirmed -11
temporal-reasoning regression on a larger diagnosed set (189 questions),
via primary-pass k=120 dilution. Re-attempting that specific broadening for
one more question is a worse trade than v10 already made and lost, so it
was deliberately left alone. The other two have a different, narrower
shape: `dc439ea3` is rule 3a (never-collapse-an-enumeration) not firing on
a list that isn't framed as a personal recommendation; `16c90bf4` is a
quote-attribution bug that doesn't add any new fact, it just points an
already-would-be-extracted fact at the wrong turn.

Added two rules to `lensSystem` (the primary pass): 2a states that rule
3a's enumeration mandate applies to any list "regardless of recommendation-
framing or position in the session"; 2b says the assistant's original
multi-option statement wins over a user's later, narrower echo of it.
`lensVersion` v11 -> v14 (v12, v13 already used and reverted earlier
tonight).

Narrow `-only` check on the 3 target qids: all three flipped correct,
including `c8f1aeed` — but reading the extraction, that one went through
`lensRetrySystem` (unchanged, untouched by this fix), because the primary
pass unexpectedly returned zero facts this run where v11 had returned two.
The win there isn't from rules 2a/2b; it's `lensRetrySystem`'s existing,
already-effective assistant-output framing catching a session that this run
happened to starve on the primary pass. Also visible in the same check: real
collateral volume growth from 2a even on the two *already-working* lists in
`dc439ea3`'s session — both picked up a redundant "list summary" fact they
didn't have before (rule 3a always asked for one; the model just wasn't
reliably producing it, and 2a's insistence made it more consistent
everywhere, not only on the target list). 14 facts -> 24 on that one session
alone. That's real k=120-dilution material spread wider than the two target
sessions, so the full 500 was run before shipping regardless of how clean
the narrow check looked.

**Full 500: 451/500, up from v11's 450.** `single-session-assistant`
52/56 (v11: 50), all three target qids correct and holding. No category
collapsed: multi-session held exactly at 114/133, temporal-reasoning held
at 118/133 (v10's shape of failure did not reproduce), knowledge-update
73/78 (-1), preference 25/30 (-3), single-session-user 69/70 (+3). Diffed
28 qid flips (22 down, 23 up) against v11: multi-session alone accounts for
10 down/10 up, net zero — the same pattern of pure re-extraction noise the
`v13` experiment already measured from any `lensVersion` bump forcing a
cold Haiku re-run dataset-wide, not a change attributable to rules 2a/2b.
One flip worth checking directly: `eaca4986` (single-session-assistant)
regressed, the same qid that separately regressed under `v13`'s unrelated
retry change. Confirmed it's the same non-determinism, not a rule effect —
extracted fact count is identical (17/17) between v11 and v14; only the
answerer's phrasing and the judge's leniency on a genuine abstention
differ.

Shipped. `lensVersion = "v14"`, commit follows this entry.

### The retrieval-mechanism swap: LLM rerank vs. term overlap, a real trade, not a win — 2026-09-09

Asked directly: every fix this session had been a hand-tuned prompt rule or
a `k` knob, all epicycles on `rankFacts`, which scores retrieval by literal
token overlap and nothing else. `winze_recall` (the production path) already
ships an LLM listwise reranker measured elsewhere in this codebase at
LATER-PROBE hit@5 14% -> 51%. This benchmark harness has never used it.

Ported the mechanism: `cmd/longmemeval/rerank.go`, a `-rerank` flag (off by
default), `rerankFacts`/`callFactRerank` mirroring `cmd/query`'s
`rerankTop`/`callRerank` shape — one Haiku call per question, given up to
`rerankCap`=200 candidate facts, asked to return them ordered by relevance;
fails open to `rankFacts` on any error. Reuses the already-extracted,
already-cached fact set, so on a warm extraction cache (no `lensVersion`
change) this costs one small Haiku call per question and zero
re-extraction — extraction is 97% of a run's spend per `main.go`'s own
`-batch` flag comment, so this is the cheapest way to test whether retrieval
quality, not the extraction prompt, is the actual ceiling.

Checked it first against `bf659f65` — the one case the earlier MMR-rerank
attempt (see "The ranking-time fix" section above) proved could *never*
work, because the target fact scored an exact 0 against the question and no
same-session redundancy discount can lift a true zero. The LLM reranker
surfaced it prominently in the answer on the first try. The question still
scored wrong (gold is 3 purchases, the answer found 2 — a separate,
unrelated extraction gap), but the ranking mechanism itself worked exactly
where the literal-overlap approach was mathematically incapable of it.

First full-500 attempt died at question 389/500: the Anthropic workspace ran
out of credit balance mid-run (`400 Bad Request... credit balance is too
low`), not a code or rate-limit problem. Re-ran clean after the user topped
up.

**Full 500: 451/500 — an exact tie with v14's 451/500.** Per-type against
v14: knowledge-update 73/78 (flat), single-session-assistant 52/56 (flat),
single-session-preference 25/30 (flat), single-session-user 69->67 (-2),
multi-session 114->112 (-2), temporal-reasoning 118->122 (+4).

The aggregate tie is two real effects canceling, confirmed by reading actual
answer content on both sides rather than trusting counts:

- **temporal-reasoning: 0 down, 4 up.** A clean, one-directional gain with
  no collateral inside the type.
- **The two single-session-user "losses" are not a rerank effect.** Both
  have `facts == retrieved` in both versions (nothing was ever cut, so
  reordering had nothing to act on) and near-identical answer text between
  versions (`58ef2f1c`: "Love is in the Air... in February" both times,
  missing the same "14th"). Pure answerer/judge sampling noise, the same
  noise floor `v13`'s experiment already characterized.
- **multi-session's 9 down / 7 up hides a real, bidirectional mechanism
  effect that has nothing to do with truncation.** Only 2 of those 16 flips
  (`88432d0a` down, `c4a1ceb8` up) involve a fact count that actually
  exceeds k=120. The rest changed with `facts == retrieved` unchanged on
  both sides — meaning the swap altered PRESENTATION ORDER among facts that
  all reach the answerer either way, and that alone was enough to flip real
  answers, not noise:
  - Fixed (up): `60159905` now finds all 3 dinner parties instead of
    missing the BBQ-at-Mike's aside that was already inside the retrieved
    set the whole time; `6c49646a` replaced a fabricated "fourth road trip"
    with the real Yellowstone leg and got the correct 3,000-mile total;
    `c4a1ceb8` correctly restricted the citrus count to actual recipes
    instead of suggestions. All three are named failures from the reverted
    `v12` aside-extraction attempt — reached here through retrieval-order,
    with none of `v12`'s extraction-volume dilution risk.
  - Broken (down): `92a0aa75` replaced a correct subtraction (3y9m total
    minus 2y4m as Coordinator = the gold 1y5m) with a bare, wrong duration;
    `aae3761f` replaced a correct 3-destination sum (4+5+6=15 hours,
    matching gold exactly) with a confused answer naming different
    destinations entirely. Literal term overlap turns out to be more
    reliable than LLM relevance judgment specifically on precise multi-fact
    arithmetic, where reordering can bury or conflate the exact few facts a
    calculation depends on.

**Not shipped — `-rerank` stays off by default, kept as tooling like
`-k-multi`.** This isn't a win by this session's bar (a clear gain with no
new collateral); it's a real, validated axis of the solution space: LLM
reranking reaches a failure class term overlap structurally cannot (facts
buried by presentation order, independent of any cutoff), at the cost of a
new failure class term overlap didn't have (precise arithmetic confused by
reordering). A hybrid — term overlap as the primary score, LLM rerank only
breaking ties among close scores or only applied when a session's fact
count actually exceeds k — is the natural next move to capture the aside
fix without the arithmetic cost, and is real, uncommitted scope: not
attempted tonight.

Caching side-note from the same session: audited all five system prompts in
this file against Anthropic's cacheable-prefix floors (Haiku ~2048 tok,
Sonnet ~1024 tok) — `lensSystem` 1499, `lensRetrySystem` 894, `answerSystem`
822, `judgeSystem` 234, `factRerankSystem` 87 tokens. Every one is under its
floor; `cache_control` is present but a no-op on all five, confirmed by
`cached=0` on every run tonight. Not worth padding artificially. The real
cost lever already exists and went unused all night: `-batch`, a flat 50%
off on extraction specifically (97% of a run's spend) — didn't matter for
tonight's rerank runs (warm cache, no re-extraction) but should be the
default for the next `lensVersion` bump's cold run.

### Reading every remaining failure directly: the biggest single gain of the night — 2026-09-09

Pushed to actually mine tonight's own trajectory instead of proposing another
benchmark run: read all 45 of the 49 remaining v14 failures directly against
real question/gold/answer content (all of temporal-reasoning's 15, 14 of
multi-session's 19 not already read via the rerank diff, all 5 preference,
all 5 knowledge-update, the 1 single-session-user). Distinct, well-evidenced
mechanisms, not one diffuse gap:

- **Undercounting/miscounting on "how many X" questions — the strongest
  signal of the night, 8+ instances**: `gpt4_ab202e7f` found 3 kitchen items
  against a gold of 5, `gpt4_7fce9456` 3 properties against 4,
  `gpt4_15e38248` 3 furniture pieces against 4, `a08a253f` 3 fitness days
  against 4, `0a995998` 2 clothing items against 3, `45dc21b6` 2 recipes
  against 3, `e3038f8c` summed to 100 against a gold of 99 by including an
  item whose own count was never stated, `681a1674` overcounted 4 Marvel
  rewatches against a gold of 2. Most have `facts == retrieved` — nothing
  cut by k=120 — so this isn't the extraction-dilution story chased
  elsewhere; the answerer has the full list and doesn't work through it.
- **Existing rules with live, current violations, not just historical
  ones**: the premise-mismatch "stop there" rule is violated by
  `2133c1b5_abs` (names Harajuku-not-Shinjuku correctly, then answers the
  Shinjuku duration anyway) and by `a96c20ee_abs` (the same Harvard
  hallucination this file already flagged unresolved back in the
  2026-09-08 entry above — still unresolved). The most-recent-value rule is
  violated by `852ce960` (picked a stale $350k pre-approval over an
  explicitly updated $400k). The judge's "extra detail is fine" line is
  violated by `gpt4_93f6379c` and `89941a94`, both matching gold's actual
  conclusion exactly plus additional correct detail, both marked INCORRECT.
- **Preference answers lead with generic advice instead of the personalized
  fact**: `09d032c9` opens with five generic battery tips before mentioning
  the user's own power bank; `57f827a0` opens "I don't have enough context"
  and only recovers with genuinely good, specific content afterward.
- Two single-instance temporal gaps read directly: `gpt4_cd90e484` has both
  numbers needed (binoculars bought "three weeks ago," walk "a week ago")
  and refuses instead of subtracting (3-1=2, matching gold); `gpt4_d31cdae3`
  treats a trip only ever mentioned as planned/future as unlocatable in
  time instead of inferring it must come after an already-happened trip.

Four changes, in one pass since they're all `answerSystem`/`judgeSystem`
scope (no extraction, no lensVersion bump, so no re-extraction cost and no
k=120-dilution risk of the kind that burned every lens-side attempt
tonight): a counting-discipline rule (scan every retrieved fact before
answering a count, and flag rather than silently include an uncertain
candidate); rewrote "most recent value wins" into a name-candidates-then-
choose shape; rewrote the premise-mismatch rule so the mismatch sentence is
declared the ENTIRE response, not just "stop there"; rewrote the judge's
extra-detail line to name the pass condition (does the actual conclusion
match) rather than what to tolerate. The shape-over-prose bet is deliberate
— mirrors `lensSystem`'s own history, where "never collapse an enumeration"
didn't land until v4 turned it into a literal output-shape requirement
rather than a stronger sentence saying the same thing.

Narrow check on 13 named target qids: only 3 flipped clean
(`gpt4_15e38248` counting, `2133c1b5_abs` premise-mismatch,
`89941a94` judge extra-detail) — most of the counting targets turned out to
be genuine extraction gaps (the missing item was never captured as a fact
at all), not answering-discipline gaps, a real correction to the
hypothesis caught before spending on anything broader. `852ce960` and
`a96c20ee_abs` didn't move — consistent with this file's own note that some
instances of the hallucination pattern aren't closable by prompt alone.

**Full 500: 460/500, up 9 from v14's 451 — the largest single gain of the
night.** Per-type: knowledge-update 73->76, multi-session 114->113,
single-session-assistant 52->53, preference 25->28, single-session-user
69->67, temporal-reasoning 118->123. Diffed all 25 flips (8 down, 17 up)
before trusting the aggregate. The 17 gains map directly onto the
mechanisms above: `2133c1b5_abs`, `89941a94`, `gpt4_93f6379c`,
`gpt4_15e38248` confirm the four rule rewrites; `09d032c9`, `a89d7624`,
`d6233ab6` confirm the preference lead-with-personalization fix;
`gpt4_cd90e484` and `gpt4_d31cdae3` are the two temporal gaps read directly
above, fixed on the first attempt. Of the 8 regressions, `58ef2f1c` and
`66f24dbb` have byte-identical answer text to v14 and flipped purely on
judge inconsistency — confirmed noise, not a rule effect. The other six are
real, honestly-costed trade-offs: `51c32626` got newly over-cautious and
refused a question it previously answered correctly; `9ee3ecd6` dropped the
final computed number (300-200=100) after restating only the intermediate
total; `gpt4_2f8be40d` shows the counting rule's real cost directly — v14
correctly filtered out uncertain-dated weddings to land on 3 (matching
gold), the new "scan everything" instruction made the model include all 5
mentioned regardless of date confidence, overcounting past gold; and
`gpt4_f420262c`/`28dc39ac` remain genuine k=120 truncation cases the answer
prompt can't reach either way.

No category collapsed — worst per-type delta is -2, nothing like v10's -11
or v13's -6. Shipped. `answerSystem` and `judgeSystem` both updated in
place (no version bump — neither is part of the extraction cache), commit
follows this entry. The counting rule's overcounting trade-off on
`gpt4_2f8be40d` is a known, accepted cost, not silently papered over: it
increases recall on the undercounting cases (the dominant failure shape)
at some real cost to precision on cases needing selective filtering. Not
tuned further tonight given the net is strongly positive; a sharper
selection criterion (something closer to "count only if the confidence and
date match the question's own scope" rather than "scan everything") is the
natural next refinement, not attempted.

### Three named next-moves chased directly, all closed with a negative result — 2026-09-09

All three checked against real data already on disk (jsonl diffs, one
`-only` run), zero full-500 spend. None panned out as scoped; all three are
now closed rather than left as open TODOs someone re-proposes later.

**`gpt4_2f8be40d`'s "date confidence" fix, named just above, is wrong about
the mechanism.** Ran `-only gpt4_2f8be40d` against the current shipped build
and read the actual extracted facts (`facts.go` for the run), not just the
answer text. There is no date-confidence signal to filter on at all: the
overcount is two independent things. First, an entity-coreference gap —
`attended_college_roommate_wedding` ("rooftop garden ceremony... in the
city") and `friend_emily_wedding` ("married partner Sarah") are two facts
from adjacent quotes in the same turn describing what the gold answer treats
as one event (the roommate is Emily), extracted as two because nothing in
either quote states the link explicitly. Second, a genuine distractor —
`attended_cousin_emily_wedding` ("my cousin Emily's wedding in the city") is
a different Emily, deliberately similar-sounding, and correctly excluded by
gold. Neither is a prompt-level fix: the first needs cross-fact identity
resolution during or after extraction, the second is already being read
correctly as a distinct event by the model (which is exactly why it gets
counted) — the model has no textual basis to know gold excludes it. Not a
near-zero-cost fix. Dropped rather than force a rewrite at the wrong layer.

**The hybrid retrieval trigger doesn't survive the full flip set.** The plan
was: term overlap by default, LLM rerank only when a session's fact count
exceeds `k` or scores are close. A fact-count threshold around 70-75 does
separate the 5 named qids cleanly (wins at 78/88/151 facts, losses at 26/67)
— but that's fitting a rule to 5 hand-picked examples. Diffed v14 vs. the
full rerank run on every multi-session flip instead (16 total, not 5): fact
counts are 20-135 on the LOSS side and 34-151 on the WIN side, fully
overlapping, no threshold separates them. Question shape doesn't separate
them either — both sides are dominated by "how many X" counting questions,
with an arithmetic question on each side too (`aae3761f` loses, `a1cc6108`
wins). There is no cheap, static per-question signal here; whether rerank
helps a specific multi-session question looks like it depends on the actual
content, not anything visible before running it. A real hybrid would need
its own classification call, which is a different, bigger experiment than
"a knob on the existing retrieval," not this session's scrappy check.
Dropped in its scoped form.

**The preference wobble (28/30 v10 -> 24/30 v11) reads as judge noise, not
the rule-2 narrowing this file guessed at.** Diffed all 6 v10/v11 flips on
`single-session-preference` directly (`caf03d32`, `54026fce`, `09d032c9`,
`38146c39`, `95228167`, `b0479f84`). Four of the five DOWN flips
(`caf03d32`, `38146c39`, `95228167`, `b0479f84`) are near-paraphrases of
each other — same specific advice, same personalization, reworded — with no
content a judge should score differently. This matches the noise pattern
already named elsewhere in this file (10/500 questions flip at identical
config; `58ef2f1c`/`66f24dbb` flipped on byte-identical answer text in the
460 diff). Closing this as noise, not a live regression to chase; the
"plausibly the same rule-2 narrowing" guess above was never checked until
now, and checking it doesn't hold up.

### The first real full-haystack numbers, and rerankCap's blind spot found and (attempted-fix reverted) — 2026-09-09

Winze has never once run the actual `longmemeval_s` haystack (distractors
included) in this project's history — every number in this file above is
oracle-set. Built a 30-question stratified sample (5 per type) against
`cmd/longmemeval/data/longmemeval_s_cleaned.json` instead of the full 500:
`-dry-run` and `-probe` first (free — confirmed session counts, 44-57/user,
and zero gold-evidence truncation before spending anything), then a real
run. Extraction cache is global and content-addressed
(`os.UserCacheDir()/winze-longmemeval/extractions`, keyed by session
content, per `main.go`'s own comment that oracle and full-haystack share
evidence sessions byte-for-byte) — so this and every follow-up below reused
warm extraction, paying only for retrieval+answer+judge each time.

**Term overlap, default retrieval: 23/30 (76.7%).** Milder than the
original paper's ~30%-relative-drop figure would project from tonight's 92%
oracle number (~64%) — not a paired comparison (different questions), just
a first real read. Per type: single-session-user 5/5, single-session-
assistant 5/5, temporal-reasoning 5/5, knowledge-update 4/5, multi-session
3/5, **single-session-preference 1/5** — preference is the standout
casualty. The number that explains the rest: `retrieved` hit exactly 120
(=k) on **all 30 questions, no exceptions** — extracted fact counts run
623-1120/question, so 85-93% of everything extracted gets cut every single
time. Nothing like this exists on the oracle set, where a question rarely
fills the window at all; the k=120 dilution mechanism chased narrowly
elsewhere in this file is universal here instead of occasional.

**Rerank, same sample, same warm cache: 23/30, identical per-type
breakdown, 0 of 30 verdicts different from term overlap.** Not a near-tie —
literally the same correctness on every question. Root cause, found by
reading `rerankFacts` rather than assuming the tie meant "no effect":
`rerankCap = 200` (`cmd/longmemeval/rerank.go`) prefilters by term overlap
down to 200 candidates before the LLM ever scores anything, whenever a
session has more than 200 facts. On the oracle set that never fires
(45-90 facts/question). On this sample it fires on all 30 (623-1120 facts
each) — term overlap was making the real candidate-inclusion decision every
time, and the LLM was only ever reordering within term overlap's own top-200
pick. The rerank mechanism never got a chance to rescue anything outside
that window; the earlier oracle-set rerank tie (451/500, see above) never
exposed this because the cap was never binding there.

**Raised `rerankCap` to 1500, re-ran the same sample: 17/30 (56.7%), net -6,
knowledge-update collapsing 4/5 -> 0/5. Reverted.** Checked before blaming
the wrong thing: `callFactRerank`'s own `MaxTokens` guard returns an
explicit error (fail-open to term overlap) on a truncated response, so this
isn't silent corruption, and the pre-existing truncated-extraction cohort
(8 of 30 questions, present identically and unaffected by this change
across all three runs) only accounts for 2 of the 6 net regressions. The
rest is Haiku's own ranking quality degrading as the candidate list grows
from 200 to up to 1500 items in a single call — a bigger context window
fitting the prompt is not the same as the model reasoning well over it.
`rerankCap` is back at 200, with both findings in its doc comment: it's a
real, confirmed ceiling on the full haystack, but raising it blindly is a
worse ceiling, not a fix. A smarter fix (chunked reranking, a mid-size cap
tried directly rather than jumping 200->1500, or restricting rerank to a
pre-filtered relevant subset larger than 200 but well short of the full
extraction) is real, uncommitted scope — not attempted tonight.

**Where this leaves the SOTA question directly:** winze has a real first
data point on the harder tier now, at real cost (one 30-question sample,
extraction paid once, three retrieval-mode passes on the warm cache) rather
than the $138/70M-token full-500 run this file was talking itself into
earlier tonight. 76.7% on this sample sits behind the field's own
self-reported range (Zep 71.2% floor, mem0 94.4%, OMEGA 93.2%) but isn't
embarrassing — and preference's 1/5 plus the universal k=120 saturation are
now two concrete, evidenced levers for whoever picks this up next, not
guesses.

### Preference's 1/5, read directly: three different causes, not one — 2026-09-09

Read all 5 `single-session-preference` questions from the haystack sample
against their actual extracted facts (grep on the gold-specific term, not
just the answer text), same discipline as `gpt4_2f8be40d` above. "1/5" hid
three unrelated mechanisms, only one of which any retrieval-side change can
touch:

- **Extraction gap** (`0edc2aef`, Miami hotel): zero mentions of "Miami"
  anywhere in 734 extracted facts. The model correctly says so and points at
  the Seattle trip it did find instead of inventing one. No retrieval fix
  reaches this — the fact never existed to retrieve.
- **Retrieval-volume gap** (`8a2466db`, Adobe Premiere Pro resources):
  "premiere"/"adobe" each hit exactly once in 911 facts — a real needle,
  genuinely present but crowded out by everything else competing for the
  window.
- **Relevance-judgment gap** (`35a27287`, language-practice preference for
  "cultural events this weekend"): 47 "french" hits and 37 "language" hits —
  abundant signal, not buried at all. Still wrong even when the LLM reranker
  saw the full 840-fact pool directly (`rerankCap=1500`, no term-overlap
  prefilter in the way). The question's own words share no vocabulary with
  the preference; neither term overlap nor an LLM judging relevance over
  everything bridges that gap. No cap size touches this one.
- `75832dbd` (AI-in-healthcare publication interest) stayed wrong across
  every configuration tried, with only weak, ambiguous signal ("medical"
  hits 4 times, "healthcare" 0) — likely a partial extraction gap, not
  chased further tonight.

**A midpoint `rerankCap=500` confirmed the retrieval-volume mechanism
directly: 23/30 overall, an exact tie with the 200 baseline, but a
different composition.** `8a2466db` (the needle case) flipped correct, as
the mechanism above predicts — and unlike `rerankCap=1500`, this didn't cost
`06878be2` (the one preference question term overlap already had right).
The tie instead came from a new, real regression: `51a45a95`
(single-session-user, "$5 coupon on coffee creamer") flipped wrong under
500 — checked the actual answer text, not just the flag: term overlap
confidently says "Target (via Cartwheel, Target's app)", matching gold;
`cap500` retrieves the same Cartwheel fact but hedges, "doesn't specify
which store," an inference the reordering apparently no longer had enough
supporting context to make confidently. Net zero on n=30 is not evidence
500 beats 200 — it's evidence the mechanism runs in both directions, real
each way. A policy that targets rerank at specifically-diagnosed needle
cases rather than blanket-raising the cap for every question is the shape
a real fix would need — not attempted with the blanket constant, but see
below.

### rerankCap scoped by question type, not blanket — the night's first real net win on the haystack axis — 2026-09-09

The blanket 500 test's regression (`51a45a95`) was `single-session-user` —
outside both categories the needle mechanism was actually diagnosed in
(`single-session-preference`'s `8a2466db`, `multi-session`'s
`gpt4_59c863d7`, read directly above and in the preference section).
Threaded `Question.QuestionType` through `runQuestion` ->
`syncAndRetrieve` -> `rerankFacts` (build gate + full `go test
./cmd/longmemeval/...` green) and added `rerankCapWide = 500`, used only
for `multi-session` and `single-session-preference`; every other type keeps
`rerankCap = 200`.

**Same warm-cache sample: 24/30 (80.0%), up from term overlap's 76.7% —
the first clean net gain reranking has produced all night, after three
straight ties or losses (the oracle-set tie at 451/500, this sample's
cap=200 tie, cap=1500's regression).** `8a2466db` flipped correct exactly
as predicted. `51a45a95` — the collateral regression from the blanket test
— stayed correct, since it's outside both scoped types and still runs at
cap=200.

Two things temper this before calling it settled, both checked rather than
assumed:

- `06878be2` (the preference question that broke under the blanket 500)
  stayed correct here too, even though it's inside the scoped type and
  genuinely runs at cap=500 in this version. Same cap, same question,
  different outcome from the earlier run — Haiku's rerank call is
  `Temperature: 0` but not bit-identical on the Anthropic API (already
  documented elsewhere in this file), so this could be the scoping working
  as intended, or it could be ordinary run-to-run noise on a borderline
  case. Not distinguishable from one more run each; not chased further
  tonight.
- `gpt4_59c863d7` (the Tiger I tank case that motivated widening
  `multi-session`'s cap at all) **did not flip** — still 4/5, still missing
  the tank. Checked why: at 840 total facts, `rerankCap=500` still
  prefilters by term overlap before the LLM sees anything, and a fact with
  only 2 "tiger" hits in the whole set apparently doesn't clear the top-500
  by literal term overlap either, only the (regression-prone) top-1500.
  Different needles sit at different depths; one scoped cap value doesn't
  rescue all of them, and there's no way to know how deep without checking
  case by case.

Left `-rerank` off by default, as it already was — this is a refinement of
an opt-in path, not a change to the harness's shipped default. n=30 makes
80.0% a real, checked signal, not a number to generalize from; a larger
sample is what would turn "first net win" into an actual verdict.

### `gpt4_59c863d7` closed: not a cap problem at all, an extraction naming gap — 2026-09-09

Asked to try a wider cap specifically for this qid. Didn't need to run
anything: it has 840 total facts, and `rerankCap=1500` (already run above)
never engages its term-overlap prefilter below 840 — the LLM already saw
every fact directly, with nothing hidden, and still answered 4/5. There is
no wider cap than "everything," and "everything" already failed. This
qid is a relevance-judgment gap like `35a27287`'s language case, not a
volume gap like `8a2466db`'s — mis-sorted into the volume category
initially by pattern-matching to `8a2466db` rather than checking each case.

Read the actual extracted fact instead of guessing further: the four kits
the model finds are each backed by 2-3 facts under a consistent
`model_kit_*` attribute prefix (`model_kit_b29_bomber`, `model_kit_camaro_scale`,
`model_kit_revell_f15_eagle`, `model_kit_spitfire_mk_v` — 10 facts total,
following the convention). The missing fifth kit — a 1/16 scale German
Tiger I tank — was extracted as a single fact under `Attribute:
"diorama_scale"`, quoting "I also started working on a diorama featuring a
1/16 scale German Tiger I tank." The lens keyed off "diorama" as the
session's salient noun and never gave this one a `model_kit_*` attribute
like its siblings. No retrieval or reranking mechanism can be expected to
count this reliably against a "how many model kits" question when its own
extracted framing doesn't signal "model kit" the way every other instance
in the same store does — this is a lens naming-consistency gap, not
something `k`, `rerankCap`, or the retrieval mechanism touches. Genuinely
closed for the retrieval axis; a real fix here would be lens-side (teaching
extraction to tag an item by what it structurally is, not by whichever noun
a sentence happened to lead with), out of scope for tonight's retrieval
work.

### `06878be2`'s noise flag, resolved — 2026-09-09

The type-scoping section above left one thing open: `06878be2` ran correct
at `rerankCapWide=500` even though the same cap value had broken it under
the earlier blanket-500 test, and the gap could have been the scoping
itself or ordinary Anthropic API non-determinism on a borderline case.
Reran it three more times, independently, same warm cache, same build:
3/3 correct. The three answers paraphrase differently (different accessory
lists, different ordering) but every one stays inside "Sony-compatible
gear" and never recommends a competing brand, which is what the judge is
actually scoring. Four consecutive correct runs at this cap settles it —
not a fragile pass, and nothing left to chase on this axis tonight. This
closes out the type-scoped `rerankCap` work started above: one clean net
win (76.7%→80.0%), one case resolved as a lens-naming gap no cap can touch,
and now this one confirmed stable rather than lucky.

### Tier-2 transcript retrieval shipped; the benchmark extension can't yet exercise it — 2026-09-09

`internal/transcript`, `cmd/query/transcript.go`, and `winze_recall_transcript`
shipped this session (`24c0778`, `bbe8bee`) per the plan at
`~/.claude/plans/floofy-giggling-jellyfish.md`: BM25 search over a session's
own raw Claude Code transcript, keyed by the session-id already sitting in a
`session-end-capture` entry's `Origin`. Two real bugs caught before shipping,
neither hypothetical: `flag.Parse()` stops at the first bare positional, so
a `--transcript <id> <query>` design would have silently dropped `--json`
(and anything else) appended after it by the MCP call path — fixed by making
the query its own `--transcript-query` flag instead of a positional; and
`runCall`'s handler map is separate from `runServe`'s tool registration,
so registering the new tool in `runServe` alone left `winze-agent call
winze_recall_transcript` failing with "unknown tool" until added there too.

The benchmark extension (`tier2Recovery` in
`selfrecall_corpus_test.go`) ran twice against real transcripts and neither
run gives tier-2 anything to recover:

- n=7 (winze + lexicon, `WINZE_TRANSCRIPT_DIR` pointed at a scratch symlink
  dir combining both projects' `.jsonl` files): TITLE 100%, LATER PROBE 7/7
  recalled, hit@5 57%, **0 absolute misses**.
- n=15 (same plus `cope` and `capitulant`, both public repos, no shared
  `winze.store` with winze's own): TITLE 93% (1 of 15 outside top-5, still
  found), LATER PROBE 13/13 recalled, hit@5 54%, **0 absolute misses**
  again, across a noticeably more heterogeneous topic mix (fusion energy,
  Buddhism, Picasso, auditory entrainment, alongside winze/lexicon's own
  material).

Winze's own two-transcript corpus was tried first, as planned, and hard-skips
(`only 1 sessions carry both a title and an opening ask`, harness needs
`>=4`) — not a small-sample caveat, a total skip.

`tier2Recovery` only fires on a LATER-PROBE session that scores rank 0 (never
found by the typed store's search at all) — hit@5 dropping to 54% at n=15
means more sessions fell *outside the top 5*, not that any went unfound.
Across both runs, and now 4 different real projects' transcripts, that
never happened once. Read plainly: at this scale and for this shape of query,
the typed store's hybrid search practically always surfaces *something* for
a real held-out question — the measured failure mode is rank degradation,
not absolute miss. That's a positive finding about the existing mechanism,
not a negative one about tier-2, but it does mean the benchmark still hasn't
produced the one number the plan was written to get: does searching the raw
transcript recover a session that the typed store missed outright. The two
manual checks from earlier in this session — `ffa99662`'s recovered
disk-cleanup content, and a live self-query recovering pre-compaction detail
from this session's own transcript — remain the only direct evidence tier-2
does what it's for. Getting a scored number would need either a much larger
combined corpus or a corpus with harder, more paraphrased LATER-PROBE
queries; not attempted tonight given the hour.

**Third run, same night, corpus widened again — the finding holds and now
reads as structural, not a sample-size gap.** Corrected an over-restrictive
scoping call: the earlier two runs used only public-GitHub sibling projects,
conflating "safe to quote into this public repo" with "safe to read into a
local, ephemeral benchmark store." The two are different bars — nothing
about this harness commits, publishes, or quotes session content anywhere;
it computes an aggregate number and deletes the scratch store. Corrected,
the corpus widened to 6 sources (the 4 above plus `freshet`, private, and
the harness's own default `~/.claude/projects/-home-gas6amus-Documents`
directory, 168 general-session transcripts) — 217 transcripts total, still
deliberately excluding `stope` and `publicai`, both of which carry their own
standing rule beyond ordinary repo-privacy. Result at n=40 (of 173 usable,
the harness's own per-run cap): TITLE PROBE 40/40 recalled (hit@5 75%),
LATER PROBE 34/34 recalled, hit@5 41%, mean rank 13.26 — **still 0 absolute
misses.** Three runs now (n=7, n=15, n=40), monotonically more diverse and
harder, and the miss count hasn't moved off zero while mean rank has climbed
4.00 → 5.46 → 13.26. That pattern reads as structural rather than
under-sampled: at max store size 40, "found at some rank" only requires a
nonzero BM25/semantic score against a session's own later-question, which a
session's own note is, almost by construction, unlikely to score a true
zero against — the harness may need many hundreds of notes in the store
before an absolute miss becomes reachable at all, at which point widening
public/private *scope* further stops being the lever; store *size* is.
Not chased further tonight. The real next move, named but not built: measure
whether tier-2 transcript search ranks *higher* than the typed store for the
~59% of LATER-PROBE sessions landing outside top-5 at n=40 — a real,
already-populated comparison, unlike the always-empty absolute-miss
population three runs have now confirmed.

**Fourth run: built the divergence comparison named above, caught a
methodology mistake spanning all three runs above, and got a real number.**
Two things worth naming plainly rather than folding quietly into a bigger
number.

First: none of the three runs above set `$WINZE_NOTE_SHAPE`, so all three
replayed under the harness's *default* shape, `openNote` (title + opening
ask) — not `outcome` (last substantial turn before the probe), the shape
`ROADMAP.md` already measured as best in earlier sessions and the one
production `session-capture` actually ships. The TITLE/LATER PROBE numbers
above are real numbers for `openNote` specifically, not a general result —
worth flagging since nothing in the text above named the shape.

Second, and caught before it shipped as a finding: `tier2NoteDivergence`
(new, `6e9a558`) compares a session's note against `winze_recall_transcript`'s
top hit for the same query, to test the plan's actual thesis directly — does
raw-transcript search find something note-compression left out? First run
came back 29/29 (100%), which is too clean to trust blind. Added a debug
flag, inspected real pairs, and found every `noteFor` shape prepends a
`Session YYYY-MM-DD (id): Title` header before the captured content — the
comparison's fixed prefix window was matching the header against the raw
turn on every session, guaranteed to differ regardless of actual content.
Fixed (`stripNoteHeader`), sanity-checked on a small corpus first (6/7, not
degenerate), then re-run properly.

Corrected final run, `WINZE_NOTE_SHAPE=outcome`, same 218-transcript 6-source
corpus, n=40 (of 174 usable): TITLE PROBE 68% (down from openNote's 75% on
this corpus — the two shapes trade wins by corpus, already an established
pattern, not new), LATER PROBE 29/29 recalled, hit@5 28%, mean rank 10.76 —
**still 0 absolute misses, a fourth run confirming the same structural
finding.** But the new number: **TIER-2 NOTE DIVERGENCE 25/29 (86%)** — for
86% of probed sessions, the raw transcript's top-matching turn for the
held-out query is a *different* turn than the one the note already captured.
That's the first real, mechanistic, non-degenerate evidence for the plan's
core thesis from this harness (rather than the two earlier manual checks
alone): compressing a note ahead of the query throws away content that
query-time search over the raw transcript recovers, for the large majority
of real sessions tried — even though the typed store's cross-session search
still, separately, almost never returns an absolute zero for the same query.
The two findings aren't in tension: a session can rank low-but-nonzero
against its own compressed note while the raw transcript still holds a
better-matching turn the note never captured at all.

### `k=120` checked against tonight's actual misses: not the mechanism — 2026-09-09

Before touching the constant, read the 6 wrong answers from tonight's best
haystack run (`haystack-rerank-scoped-1788992545.log`, 24/30, 80%) against
their own extracted-fact pools (each question's `facts.go`) and, where still
unresolved, the raw haystack source itself. `retr` hit exactly 120 on all 30
questions in this run too, the same universal saturation named earlier
tonight. None of the 6 misses trace to that cut.

- `35a27287`, `gpt4_59c863d7`, `0edc2aef`, `75832dbd` — already closed above
  (relevance-judgment gap, lens-naming gap, extraction gap, weak/ambiguous
  partial extraction).
- **`852ce960` (Wells Fargo pre-approval, gold $400,000), new**: the raw
  haystack has two mentions — an earlier "$350,000," and a later "remember
  when I got pre-approved for $400,000" buried inside an unrelated cable/TV
  setup sentence — a knowledge-update case exactly like the yoga-classes and
  Korean-restaurant questions winze already gets right elsewhere in this
  sample. But `facts.go` has zero occurrences of "400,000" or "400000"
  anywhere across 646 extracted facts — the lens only ever captured the
  superseded $350,000 mention. A pure extraction gap: the correct value
  never entered the candidate pool to be cut from.
- **`0a995998` (clothing pickups/returns, gold 3), new**: both source items
  — the blazer dry-cleaning pickup and the Zara boots exchange — are present
  in `facts.go` (`navy_blazer_*`, `boots_exchange*`), so nothing was cut or
  missed. The gold key counts the boots exchange as two items (return the
  old pair, pick up the new pair); the model counted it as one transaction,
  landing on 2 instead of 3. Same shape as the 10/500-flip noise already
  named elsewhere in this file — a gold-answer counting convention, not a
  retrieval failure, and no retrieval mechanism touches it.

**Where this leaves last turn's proposed next move**: raising or
type-scoping `k` the way `rerankCap` was scoped tonight has nothing to
recover here. Not one of the 6 misses in the current-best run has its gold
fact sitting in the extracted pool, cut only by the final top-120 selection
— `k=120`'s universal saturation is real as a raw statistic but isn't, on
this evidence, where tonight's wrongness actually comes from. The two levers
that do show up are both upstream of retrieval: extraction completeness (two
of six misses tonight are the lens silently dropping a true fact) and a
semantic/embedding channel for vocabulary-mismatched relevance judgment (one
of six, the same mechanism `35a27287` already named — `rankFacts` is
confirmed pure term-overlap counting, no semantic scoring anywhere in this
pipeline). Neither is a constant to tune.

### Extraction-completeness fix for the mortgage case: deferred, not built — 2026-09-09

Before writing a lens rule for `852ce960`'s shape (a value named as an aside
inside an unrelated sentence, not the session's dominant topic), checked
`lensVersion`'s own changelog first. This exact fix has been tried twice
already, both measured net-negative on the full 500:

- **v10** broadened the primary pass's assistant-specifics rule. Fixed 11/18
  targeted assistant-recall failures; regressed temporal-reasoning
  122/133 -> 111/133 on the same 133 questions, because the broadened rule
  ran on every session, not just the starved ones, and the extra captured
  content competed for the same fixed k=120 window.
- **v12** tried a narrower version, specifically for "by the way" asides —
  the exact shape `852ce960` needs. Flipped 7/19 targeted multi-session
  failures; full-500 re-run came back multi-session 114/133 -> 113/133,
  preference 28/30 -> 26/30 — net *worse* on the categories it targeted.
  Reverted same session.

Both entries end on the same lesson: any extraction change that adds real
facts still competes for k=120, so a targeted fix can't be judged from the
cases it targets — only from a full-500 re-run, since the damage lands
elsewhere. There's no cheap way to check this on `852ce960` alone; a
single-question or small-sample test is structurally blind to the exact
failure mode being guarded against.

Asked directly rather than silently building it: don't fund a standalone
full-500 run just to re-test a fix shape that's already failed twice.
Decision: leave the lens alone for now, and don't chase a retrieval-side
supersession detector as a dedicated task either — batch a check of this
mortgage-style gap into whatever full-500 run happens next for other
reasons, rather than paying for one on its own.

### Semantic-fusion retrieval channel shipped, measured: 83% (25/30), up from 80% — 2026-09-09

Built the other half of the two-lever finding above: `rankFacts`/`rerankFacts`
had zero semantic scoring anywhere in the pipeline (`rankFacts` is pure term-
overlap counting, confirmed by reading it directly), which is exactly why
`35a27287` stayed wrong even with the LLM seeing every fact directly — real
signal, zero vocabulary overlap with the question's own words. Added
`(*runner).semanticFuseFacts` (`cmd/longmemeval/semantic.go`, new), a fourth
retrieval mode (`-semantic`, alongside default/`-rerank`) that RRF-fuses
`scoreByOverlap`'s term-overlap ranking (extracted from `rankFacts` as a
shared, now-testable core) with embedding cosine similarity over every fact,
local ollama (`all-minilm`, same model and score distribution as the rest of
winze), disk-cached by content hash exactly like `extractSession`'s own
cache — same content-hash-plus-atomic-rename shape, same reason: a
concurrent question loop can miss the same key at once, and per-key files
make that race harmless. Deliberately not imported from `cmd/query`, which
has the same mechanics (`embed`, `bestCosine`, `rrfFuse`) already shipped —
this package keeps its own copies of shared mechanics rather than depending
on shipped production code, the same boundary the tier-2 transcript plan
drew. Pure logic (`fuseRankMaps`, the RRF core) covered by unit tests before
any real run; `scoreByOverlap`'s extraction verified against `rankFacts`'s
prior behavior.

Checked the single diagnosed case first, cheap, before spending on the
full sample: `-only 35a27287 -semantic` on the warm extraction cache flipped
it correct. Then the full same-30-question sample used for every other
number tonight, cold embedding cache: **83% (25/30), up from the type-scoped
rerank's 80% (24/30) and term-overlap's 76.7% (23/30).**

Two real, mechanistically distinct recoveries, not one:

- `35a27287` (the French/language preference case) — the one this channel
  was built for. Fixed as predicted.
- `gpt4_59c863d7` (the Tiger I tank model-kit count) — previously declared
  "closed for the retrieval axis" earlier tonight, on the reasoning that its
  extraction tagged the tank under `diorama_scale` instead of `model_kit_*`
  like its four siblings, so no retrieval mechanism should be expected to
  connect it. That reasoning held for term overlap and for LLM rerank (which
  still starts from a term-overlap-prefiltered candidate pool) but not for
  embeddings: "a diorama featuring a 1/16 scale German Tiger I tank" sits
  close enough to "model kit" in embedding space to surface anyway, no
  attribute-name match needed. Retrieval recovered what extraction mis-named,
  which the earlier verdict didn't consider because it was only checking
  whether term-overlap-based mechanisms could reach it.

One real, new regression, checked rather than waved away: `6d550036`
("how many projects have I led or am currently leading?", gold 2) flipped
from correct to wrong. The semantic channel pulled a third, previously
out-of-window fact into the top-120 — a "rural water access project...
planning/leading" — semantically close enough to "projects I lead" to rank
in, and the answerer counted it despite gold treating a planning-stage
mention as not yet a led project. Same shape as every other retrieval change
tonight: a fix for one case is a plausible new failure mode for an adjacent
one, not a free lift. Net on this sample: +2 fixed, -1 regressed, +1 overall.

Real cost, measured rather than assumed: the cold run took ~20 minutes at
concurrency 8 (host load 14-15 from concurrent sessions tonight — a factor,
not isolated), embedding ~900 facts/question the first time each is seen.
205MB / 25,806 embeddings now sit in the disk cache
(`~/.cache/winze-longmemeval/embeddings`, sibling to the extraction cache),
content-hash keyed the same way, so a warm rerun pays only answer+judge —
the same warm/cold shape the extraction cache already has, not a new cost
story. `-semantic` is off by default, same posture as `-rerank`: a refinement
to test, not a change to the shipped default. n=30 is a real, checked signal
on this sample, not a number to generalize to the full 500 from.
