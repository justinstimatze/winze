# Demand-Signal Validation — 2026-04-29

Validation battery for bias-state and calibration-marker context injections in `--ask`.

## Setup

- Script: `scripts/validate_demand_signals.sh`
- 6 questions × 2 trials × 2 conditions (signals present / stashed)
- State files tested: `.metabolism-bias-state.json`, `.metabolism-calibration-state.json`
- KB state: AvailabilityHeuristic + SurvivorshipBias triggered; DimaraTaskBasedClassification + HypothesisLakeChekoIsCrater challenged

## Question Design

| Tag | Class | Rationale |
|-----|-------|-----------|
| consciousness-reliability | bias-benefit | Wikipedia concentration should prompt hedging |
| predictive-processing-evidence | bias-benefit | Single-source risk; no external corroboration |
| dimara-classification | cal-benefit | 1 challenge, 0 corroborations in sensor log |
| lake-cheko | cal-benefit | 1 challenge, 0 corroborations in sensor log |
| hard-problem-support | cal-benefit | 3 corroborations / 6 cycles |
| luhmann-zettelkasten | negative-control | Neither signal should change this answer |

## Results

| Question | Class | Verdict | Notes |
|----------|-------|---------|-------|
| consciousness-reliability | bias-benefit | WEAK + | Both conditions detect concentration — it's visible in the provenance data. WITH adds audit framing + counts; WITHOUT wrote a *longer* concentration analysis. |
| predictive-processing-evidence | bias-benefit | MODERATE + | WITH correctly names "zero direct external corroborations across 290-cycle sensor log." WITHOUT says "absent" with no count. Calibration data is the differentiator, not bias framing. |
| dimara-classification | cal-benefit | **STRONG +** | WITHOUT misreads the evidence outcome as positive ("15+ results... suggesting external validation"). WITH correctly reports "1 challenge, 0 corroborations." Clearest signal in the battery. |
| lake-cheko | cal-benefit | **STRONG +** | WITH gives "1 challenge, 0 corroborations across 35 cycles" + directional interpretation. WITHOUT presents dispute neutrally, can't weight sides. |
| hard-problem-support | cal-benefit | MODERATE + | WITH reports "3 corroborations / 6 cycles" + notes Wikipedia-63% bias flag. WITHOUT says "resolved" with no counts. |
| luhmann-zettelkasten | negative-control | **PASS** | Answers content-equivalent, one short paragraph each. Instrument is clean. |

## Conclusions

**Calibration injection earns its keep.** Hypothesis-level challenge/corroboration counts are not derivable from the typed KB claims. Without the injection, the model gets Dimara's evidence direction wrong. For challenged hypotheses, the injection gives directional guidance; without it, the model hedges symmetrically.

**Bias injection is weaker than expected.** Source concentration is already visible in the provenance data — the model reads it from the claims themselves. The bias audit adds audit-language framing and counts, but the WITHOUT condition for consciousness-reliability was actually more thorough on concentration analysis.

**Negative control is clean.** No injection pollution on unrelated queries.

**Decision:** Keep both injections. Calibration is clearly load-bearing. Bias is marginal but free (no API cost, just string injection), and the clean negative control shows negligible noise risk.

## What This Does NOT Test

- Whether the KB's factual claims are correct (the calibration state reports what the sensor found, not ground truth)
- Multi-turn or interactive REPL behavior
- Queries where neither signal is relevant (only one negative control tested)
- Effect on downstream ingest decisions
