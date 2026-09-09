# Baselines

One file per scored configuration: the per-question outcome of a longmemeval
run, as JSONL. Emitted by `--baseline <path>`.

```bash
/tmp/lme --dataset … --work … --cache … -k 60 \
  --temporal 10 --knowledge 10 --single 10 --multi 10 --assistant 10 --preference 10 \
  --baseline cmd/longmemeval/baselines/v5-k60.jsonl
```

## Why these are in git

The aggregate score cannot carry a comparison on its own. Two byte-identical
runs — same lens version, same `k`, every session served from the extraction
cache — scored 43/60 and 44/60, so a two-question move between configurations
is inside the noise floor. Which *specific* questions flipped is not. Three
assistant questions turning correct while one temporal turns wrong is a
mechanism you can go read; +2 overall is nothing.

That comparison needs the previous configuration's per-question outcomes, and
regenerating them costs real API spend: extraction caches to disk, answers and
judgements do not. So the baseline is committed and the run log is not.

## Diffing

Rows are in run order, which is stable: `selectSubset` takes the first N of
each type in dataset file order, so the same quota returns the same sixty qids
every time. Line N of one baseline is the same question as line N of another.

```bash
# which questions changed verdict
jq -r '[.qid, .type, (.correct|tostring)] | @tsv' old.jsonl > /tmp/a
jq -r '[.qid, .type, (.correct|tostring)] | @tsv' new.jsonl > /tmp/b
delta /tmp/a /tmp/b
```

## What is not here

Timings. They differ on every run and would rewrite all sixty rows each time,
burying the handful of lines that carry a finding. They stay in the run log,
where `docs/benchmark.md` records that numbers off a loaded box are ceilings
rather than measurements.

Errored questions are absent rather than recorded as wrong — a harness failure
is not a memory failure, and a diff that shows one is misleading. Check the
`scored / errored / attempted` line in the run's own report.

## Files

- `v4-k60.jsonl` — lens v4, `k=60`, oracle set, 47/60. The best configuration
  as of 2026-08-06 and the current default (`-k` defaults to 60). Reconstructed
  from that run's log, which predates the `--baseline` flag; the reconstruction
  reproduces the reported 47 correct and the per-type row exactly.
- `v12-lens1a-attempt-full500.jsonl` — a lens rule tried and reverted the same
  night (2026-09-08): read every turn for an unrelated aside instead of the
  session's dominant topic, aimed at multi-session/preference failures where
  a raw-transcript control had the fact and winze's extraction didn't. 450/500
  overall (unchanged from `v11-answersys-k120-full500.jsonl`), but worse on
  both targeted categories — multi-session 114->113, preference 28->26 — via
  the same k=120 dilution mechanism that broke temporal under lens v10.
  Superseded by reverting to v11's lens; kept for the per-question record.
- `v11-lens-answersys-attempt-full500.jsonl` — v11's lens (byte-identical,
  warm cache) with two answerSystem rules tried and reverted the same night:
  context-scoping "most recent wins", and a rule against substituting outside
  knowledge for a missing specific number. 441/500 — every type flat or down
  from the 450/500 reference, none improved. Isolates the answerSystem-only
  effect from the lens change above; both were reverted, see `ROADMAP.md`.
- `v11-multi-only-k200.jsonl`, `v11-multi-only-k300.jsonl` — the 133
  multi-session questions only, `-k-multi` scoped so no other type is
  affected, testing whether a larger retrieval window helps multi-session
  specifically. 112/133 and 111/133 against the 114/133 reference at k=120 —
  worse at both points, replicating the 2026-08-07 k-sweep's non-monotone
  finding (`docs/benchmark.md`) on today's pipeline. The `-k-multi` flag
  stays in the code (off by default) since the tooling is real even though
  the hypothesis it tests failed; see `ROADMAP.md`.
