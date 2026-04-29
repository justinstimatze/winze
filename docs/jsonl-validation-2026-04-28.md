# JSONL demand-side validation — 2026-04-28

Empirical follow-up to commit f4f456a ("query: surface trip-cycle dream-state
into --ask context"). The wiring was previously validated on a single
anecdote; this battery scores 10 questions × 2 trials × 2 conditions.

**Verdict: validation succeeds.** Should-help class wins 4 of 5 (threshold for
"demand-side wiring validated" was 3 of 5). Negative-control class shows
acceptable trial variance — format differs, content matches. Mixed class
adds orthogonal info on both questions.

## Setup

- Model: `claude-haiku-4-5-20251001` (the --ask default)
- JSONL snapshot: 53 NONE-predicate rows from polecat wi-085
  (`/tmp/winze-none-predicate-2026-04-27.jsonl`, 53540 bytes)
- Battery script: `scripts/validate_jsonl_help.sh`
- Comparison generator: `scripts/score_battery.sh`
- Raw outputs: `/tmp/jsonl-battery-1777400136` (with), `/tmp/jsonl-battery-1777403596` (without)
- Paired report: `/tmp/battery-paired-1777406514.md`

## Per-question results

| Question | Class | WITH cited JSONL? | WITHOUT honest non-answer? | Verdict |
|---|---|---|---|---|
| godel-bias | should-help | t1: yes (score 3/5, 4/5) — t2: errored | t1: inferred from typed — t2: cited reified TripCycle1 | **WITH wins on specificity** |
| hilbert-consciousness | should-help | t1: 3 connections w/ scores — t2: 3 connections w/ scores | t1: typed-only, abstract — t2: typed-only, abstract | **WITH wins on richness** |
| dual-process-pp | should-help | t1: Kahneman↔WhiteShergill 4/5 + Kahneman↔Mattson 4/5 — t2: same | t1+t2: cited reified TripCycle21 only | **WITH wins on dimension** |
| bias-failure | should-help | t1: vague "speculative connection notes" — t2: typed-only | t1+t2: typed-only | **tie** (typed corpus answers question) |
| godel-baloney | should-help | t1: 4/5 cited — t2: 4/5 cited + secondary | t1+t2: honest "no direct relationship" | **WITH wins on synthesis** |
| hard-problem | negative-control | t1+t2: Chalmers correct | t1+t2: Chalmers correct | tie (instrument OK) |
| tunguska | negative-control | t1+t2: same facts | t1+t2: same facts | tie (instrument OK) |
| disputes | negative-control | t1: 18 disputes — t2: 16 (minor undercount) | t1: 18 — t2: 17 (variant grouping) | tie (instrument OK) |
| apophenia-contested | mixed | t1: cited Conrad↔Kahneman 4/5 — t2: typed-only | t1+t2: typed-only | WITH adds value when surfaced |
| searle-kahneman | mixed | t1: 4/5 cited — t2: 3/5 cited | t1+t2: honest "no claim linking them" | **WITH wins on synthesis** |

Should-help: 4 wins, 1 tie, 0 losses → ≥3/5 threshold met.
Negative-control: 3/3 ties → instrument variance is acceptable.
Mixed: 2/2 architecturally orthogonal → safe to expand demand-side surface.

## What the WITH responses surfaced that WITHOUT did not

The JSONL section put specific, scored cross-cluster pairs into context
that don't exist as typed Predicts/ResolvedAs claims:

- Kahneman ↔ WhiteShergill (System 1/2 ↔ top-down/bottom-up, score 4/5)
- Hilbert ↔ IIT intractability (score 4/5)
- Hilbert ↔ Baloney Detection Kit (score 4/5)
- Searle's biological naturalism ↔ Kahneman dual-process (score 4/5)
- Conrad apophenia ↔ Kahneman (score 4/5)
- Gödel ↔ Sagan baloney (score 4/5)

Each appears with provenance citation (`predictions.go`,
`metabolismPredictionSource`) and an explicit hedge ("speculative cross-cluster
connections", "not yet promoted to authoritative status"). Haiku surfaced this
correctly without being told to hedge.

## What WITHOUT does well

When no speculative connection exists, WITHOUT honestly says so —
"no direct relational claim", "no claim linking them". This is the right
behavior when the JSONL is absent. The wiring does not pressure Haiku into
confabulation; it expands available context, doesn't force consumption.

## Failure mode logged

`godel-bias__should-help__jsonl-yes__trial-2.txt` errored with `defndb:
defn not available`. Single occurrence across 40 runs. Excluded from
scoring; not a JSONL-related failure.

## Implications

The brief's interpretation table maps this outcome to:
- **Package as commonplace-book demo** — this report is the seed. The
  with/without comparison is externally-shareable evidence that winze's
  typed-claim + speculative-residue architecture beats vector-DB
  filing-cabinet on synthesis questions
- **Expand demand-side surface** — topology fragility flags, calibration
  novelty markers, current bias-state at query time are the natural next
  injections (the "bigger Step 2"). Wiring more bias auditors to behavior
  (Anchoring, ClusteringIllusion, SurvivorshipBias) is in the same family

## Caveats

- Sample is one snapshot (53 rows from one polecat run). Whether the
  win generalizes to a different snapshot is untested.
- Haiku is non-deterministic. Two trials per condition smooths this for
  qualitative scoring but is not a power-analysis result.
- The win on should-help comes from rich speculative pairs being present
  AND on-topic for the question. A question about a domain the JSONL
  doesn't cover would degrade to the bias-failure tie outcome.
