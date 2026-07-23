# Predicate-gap surfacer

```bash
go run ./cmd/predicates-suggest .                             # default: Sonnet, min-cluster 3
go run ./cmd/predicates-suggest --min-cluster 4 --model haiku .
go run ./cmd/predicates-suggest --dry-run .                   # print prompt, no API call
```

Reads `.metabolism-trip-isolated.jsonl` (the trip-cycle log of generations
whose rationale states no existing predicate captures the connection),
filters by score, and asks an LLM to cluster entries by predicate-shape.
For each cluster ≥ `--min-cluster`, proposes a candidate predicate (name,
slot types, sample claims, rationale). Empty array is a valid answer when
the existing vocabulary already absorbs the observed gap — which was
confirmed on the first real run: all 51 score≥3 entries are absorbed by
`StructurallyAnalogousTo`.

Candidates write to `.metabolism-predicate-candidates.json` (gitignored).
Human review is mandatory — the tool surfaces candidates, never promotes.
This preserves the "do NOT invent predicates speculatively" discipline
while closing the predicate-ontology gap signal loop.
