# Metabolism (automated evolution)

```bash
# === THE DEFAULT: one command, full KB evolution ===
go run ./cmd/metabolism --evolve .                     # full KB evolution: bias-audit → sense → evaluate → ingest → trip (promote) → dream (cleanup) → calibrate
go run ./cmd/metabolism --evolve --dry-run .            # preview what would happen

# === Individual phases (for debugging/manual control) ===
go run ./cmd/metabolism .                              # sensor-only (arXiv backend)
go run ./cmd/metabolism --backend all .                # sensor-only (all backends)
go run ./cmd/metabolism --pipeline .                   # quality-gated ingest from corroborated cycles
go run ./cmd/metabolism --dream .                      # consolidation: topology+lint+adit analysis
go run ./cmd/metabolism --dream --fix --tighten .      # auto-fix overlong Briefs via LLM
go run ./cmd/metabolism --trip .                       # speculative cross-cluster connections
go run ./cmd/metabolism --calibrate .                  # prediction accuracy analysis
go run ./cmd/metabolism --durability .                 # re-run KB-internal resolvers against current corpus; report drift
go run ./cmd/metabolism --durability --write .         # also append _recheck entries to the log
go run ./cmd/metabolism --reify .                      # generate predictions.go from log
go run ./cmd/metabolism --bias .                       # cognitive bias self-audit
go run ./cmd/metabolism --trip --temperature 0.9 --prompt-type contradiction --pairs 10 .  # custom drug profile
go run ./cmd/metabolism --trip --prompt-type confluence --pairs 5 .   # 3-entity narrative convergence (REBUS-weighted)
go run ./cmd/metabolism --trip --prompt-type synthesis --temperature 1.0 --pairs 5 .  # 3-5 entity concept genesis (includes isolates)
go run ./cmd/metabolism --pkm /path/to/vault .          # PKM ingest: markdown notes → typed Go corpus files
go run ./cmd/metabolism --pkm /path/to/vault --dry-run . # show what would be generated
go run ./cmd/metabolism --entity-cap 250 .             # refuse ingest/pipeline above entity cap (default 250)
go run ./cmd/metabolism --json .                       # JSON output
```

One cycle: topology identifies fragile hypotheses → sensor queries (arXiv
and/or Wikipedia ZIM) for external signal → results logged to
`.metabolism-log.json` → calibration tracks whether structural fragility
predicts curation gaps. Topology reads the log to deprioritize already-queried
hypotheses (fresh ones get sensor attention first). Zero-paper cycles
auto-resolve as `no_signal`. `--suggest` generates ingest templates from
corroborated results. `--reify` generates `predictions.go` encoding metabolism
predictions as first-class Predicts/ResolvedAs claims (the KB becomes self-aware
about its own epistemic performance). `--pipeline` runs the full automated quality pipeline:
ingest → go build → go vet → deterministic lint → LLM contradiction check →
commit if all pass, reject if any gate fails. Exit 2 = quality rejection.
ZIM backend uses gozim (pure Go, no Python needed). Builds a Bleve fulltext
index on first use (persisted to `<zimfile>.bleve/`).
RSS backend fetches Atom/RSS 2.0 feeds and filters entries by query terms.
Default feeds cover cognitive science (Nature Reviews Neuroscience, Trends in
Cognitive Sciences), AI (arXiv cs.AI, cs.CL), and philosophy of mind (PhilPapers).
Custom feeds via `--rss-feeds URL1,URL2,...`. No API key needed.

## Sleep-cycle phases (dream, trip)

`--dream` runs a consolidation cycle (NREM-like): no new ingest, analyzes
existing KB health via topology + lint + adit and reports maintenance
opportunities (bridge entities, file balance, provenance splits, brief quality).
`--dream --fix --tighten` auto-fixes overlong Briefs via LLM (Haiku), with
quality gates (build + vet + lint) and automatic revert on failure.

`--trip` runs a speculative cross-cluster connection cycle (REM-like): picks
entity pairs from different topology clusters, LLM generates and scores
speculative connections. Two orthogonal axes: `--temperature` (0.0-1.0,
wildness) and `--prompt-type` (analogy, contradiction, genealogy, prediction,
confluence, synthesis). Pair-based types (analogy/contradiction/genealogy/prediction)
pick 2 cross-cluster entities. Group-based types pick 3-5 entities with REBUS-like
flattened priors (thin entities boosted, not penalized). Confluence picks 3 entities
cross-cluster and generates ~150 word narratives about structural convergence.
Synthesis picks 3-5 entities including isolates and generates ~200 word narratives
that name a new concept the KB doesn't have yet. Group modes use Sonnet for
narrative generation and Haiku for scoring (two-pass).
Together they form a "drug profile." Score >= 3 = interesting, >= 4 = promote.

## Bias self-audit

`--bias` runs cognitive bias self-audit: the KB's own bias catalog (confirmation
bias, anchoring, clustering illusion, availability heuristic, survivorship bias)
applied as deterministic auditors checking KB structure. Runs standalone or as
part of dream (`--dream --bias`). Each auditor reports a metric, threshold,
and whether the bias was triggered. The KB eats its own dogfood.
Bias auditors also gate `--evolve` as Phase 0. When AvailabilityHeuristic
triggers (provenance HHI > 0.25, i.e. too concentrated on one source), the
cycle skips the ZIM backend so querying Wikipedia-while-Wikipedia-dominant
doesn't deepen the concentration. Other triggers currently surface only;
wiring more gates belongs in cmd/metabolism/bias_gates.go.

## Calibration and durability

`--calibrate` includes prediction accuracy scoring per hypothesis with
hit rate, precision, and efficiency metrics, plus a post-hoc provenance-overlap
scan (gap_confirmed / mixed_overlap / no_gap). A corroborated cycle whose
sources are all already in the corpus provenance is tautological (no_gap);
one with at least one novel source is real external signal. Recomputed each
run because novelty is a moving target as the corpus grows.

The prediction-type breakdown splits by whether a prediction can be wrong.
**Falsifiable predictions** (`structural_fragility`: topology flags a fragile
hypothesis, then the sensor either finds curation evidence or doesn't) carry a
real hit rate — that number is the calibration. **Durability checks** (`trip_*`:
a claim already promoted through the build gate is "predicted" to pass that
gate) sit at ~100% by construction, so their hit rate is not calibration — it is
reported as a "still hold" rate, and their real signal is drift on the
`--durability` recheck, where a verdict can flip as the corpus and resolver code
evolve. Reporting the tautological 100% as a hit rate would flatter the
calibration, so the two are labeled apart.

`--irrelevance-audit` re-classifies a sample of "irrelevant" cycles under
a neutral prompt (no "default to irrelevant" framing) and reports the
flip rate. Diagnostic only — does not mutate the log. Flags:
`--audit-n=N` (default 10), `--audit-haiku=true` (default), and
`--audit-require-snippet` (only sample cycles whose papers carry at
least one snippet). When the flip rate is non-trivial the production
prompt is likely over-strict.

`--durability` re-runs KB-internal resolvers (lint, functional, build-gate)
against the current corpus and reports drift vs historical verdicts:
stable, flipped_to_confirmed, flipped_to_refuted, now_ambiguous,
unresolvable (claim var gone), resolver_changed (oracle code digest moved
but verdict unchanged — attribution guard). Each cycle records
OracleCommit (git HEAD SHA) and OracleDigest (sha256 over resolver source
files) so drift can be attributed to corpus-change vs oracle-code-change.
Read-only by default; `--durability --write` appends _recheck entries to
the log so next `--calibrate` sees time-series signal.

## PKM ingest

`--pkm` reads a directory of markdown notes (Obsidian vaults, Zettelkasten, plain
markdown) and generates typed Go corpus files (`pkm_*.go`). Mechanical extraction:
parses titles, authors, wikilinks, and `**Prediction**:` lines. No LLM needed.
Generated files are clearly separated by `pkm_` prefix, cleanly removable
(`rm pkm_*.go && go build ./...`), and repeatable (re-running regenerates from
scratch). The demo vault is at `../winze-demo-vault/` (separate repo, separate
git history). Use `--entity-cap 300` when ingesting alongside the starter corpus.

## Multi-timescale laminar cycles (scheduler-agnostic)

Cadence belongs to whatever scheduler fires the loop (cron, a CI job, a
systemd timer); phase granularity and self-gating belong to winze. The
pattern: a scheduler runs `go run ./cmd/metabolism --evolve .` on a regular
clock (hourly is a reasonable default), and winze's per-phase gates decide
what's actually worth running on each tick. Cost tracks opportunity, not the
clock.

`--evolve` runs eight named phases: `bias`, `sense`, `resolve`, `ingest`,
`trip`, `dream`, `calibrate`, `reify`. Cheap phases (`bias`, `dream`,
`calibrate`) emit telemetry every tick — they have no LLM or sensor cost and
produce the signal the dynamic gates read. Expensive phases (`sense`, `resolve`,
`ingest`, `trip`, plus dream's optional brief-tightening) self-gate against that
telemetry. `calibrate` is read-only analysis; `reify` is the one that *writes* —
it regenerates the tracked `predictions.go` from the log, so it is its own phase
and a read-only run (a `--phases` list without `reify`) leaves the working tree
clean. A bare `--evolve` (no `--phases`) runs all eight, reify included.

A scheduler can also fire a phase subset directly with `--phases`:

```bash
go run ./cmd/metabolism --evolve --phases=cheap .   # bias + dream + calibrate (no LLM, no sensors)
go run ./cmd/metabolism --evolve --phases=llm .     # resolve + ingest + trip + dream
go run ./cmd/metabolism --evolve --phases=sense,resolve .
```

Default gate thresholds (sensible for a ~300-cycle corpus on an hourly tick;
override with the matching flag, `=0` to disable that gate):

| Phase   | Default gate                                   | Flag |
|---------|------------------------------------------------|------|
| sense   | last sensor cycle ≥ 4h ago                     | `--sense-min-hours` |
| resolve | ≥3 hypotheses with 3+ unresolved signal cycles | `--resolve-min-unresolved` |
| trip    | last `metabolism_cycle*.go` mtime ≥ 24h ago    | `--trip-min-hours` |
| ingest  | ≥1 corroborated-but-uningested cycle           | `--ingest-min-corroborated` |

Gate decisions print as `[gate] <phase>: ALLOW|SKIP: <reason>` so formula logs
make every firing decision auditable. The `--phases` filter, the dynamic gate,
and the budget guard all stack independently — a phase needs all three to
greenlight before it runs.

## Budget guard

Set `METABOLISM_BUDGET_CENTS` to cap estimated monthly LLM/API spend. Per-phase
estimates: sense 13¢, resolve 10¢, ingest 20¢, trip 15¢, dream-fix 1¢. Per
`--evolve` run when all expensive phases fire: ~59¢; on the gated rhythm most
ticks are mostly-cheap. State persists in `.metabolism-budget.json` with monthly
auto-reset. Set `METABOLISM_BUDGET_CENTS=0` (or unset) to disable. Estimates are
approximations — actual spend depends on response length; the bookkeeping is
conservative.

`go run ./cmd/metabolism --calibrate-trend .` prints the rolling time series
from `.metabolism-calibration.jsonl` (one row per past `--calibrate`
invocation) — answers "is useful signal trending up?" without re-running
calibrate.

**What to ignore.** Do NOT add cron, systemd, or any cadence-management
machinery inside winze itself. The scheduler owns cadence; winze owns
self-gating. If the schedule needs a different rhythm, change the scheduler
or set the `--*-min-*` thresholds — never grow winze a scheduler. (Per-instance
activation via systemd timers lives in `cmd/metabolize`, outside the metabolism
engine.)

## Sharing & autonomous metabolism (the vision)

Winze's north star: instances **evolve themselves** (the `--evolve` loop) and
**share what they learn**. Both are infrastructure-light:

- **Autonomous metabolism** is scheduler-agnostic. Winze owns the phase
  granularity, self-gating, and budget guard; *any* scheduler — cron, a CI job,
  a systemd timer — fires `metabolism --evolve .` on a clock. Do NOT grow winze
  its own scheduler; the cadence lives outside.
- **Sharing** is plain GitHub: a winze instance is a fork, and instances share
  corpus/tooling improvements by opening **pull requests between forks**. The
  build gate is the review — a PR that doesn't compile doesn't merge.
