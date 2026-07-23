# Rot probe

```bash
go run ./cmd/rot-probe --n 10 --model haiku .              # default: 10 entities, Haiku
go run ./cmd/rot-probe --n 20 --model sonnet .             # deeper sample, deeper model
go run ./cmd/rot-probe --n 5 --seed 42 --dry-run .         # preview prompt, no API call
```

Samples a random subset of corpus entities and asks an LLM to flag potential
rot signals: `duplicate` (two entities likely the same thing), `contradiction`
(claims that can't all be true), `brief_drift` (Brief text no longer matches
the entity's claims), `trip_attractor` (entity has more trip-cycle-generated
claims than source-grounded claims, with the trip claims attaching concepts
the Brief does not anticipate — a topology signal, not necessarily a defect).
Findings are surfaced for human review only — the tool NEVER auto-fixes.
Output appends to `.metabolism-rot-probe.jsonl` (gitignored) for time-series.
Empty findings on a small sample is a valid answer; not a green light to skip
the next probe.

Trip-cycle claims are detected by var-name heuristic (`^TripCycle\d`),
serialized to the LLM prompt with a `[trip]` tag so the model can reason
about source-grounded vs trip-promoted claim ratios per entity.

The whole point of the typed substrate is to surface what inspection misses.
Until rot surfacing actually surfaces things, the "is the typed gate worth
its friction" question lives on faith. Periodic rot-probe runs convert that
question into evidence.
