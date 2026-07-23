# winze-sensor (external signal probe)

`winze-sensor` queries external sources for papers matching a set of query terms
and reports new results, filtering out ones already seen. It is the raw sensor
probe the metabolism loop wraps — useful standalone to check what a topology
target would surface before committing a full cycle.

```bash
winze-sensor "predictive processing" "active inference" .   # arXiv (default)
winze-sensor --backend all "confirmation bias" .            # arXiv + Semantic Scholar
winze-sensor --backend scholar --limit 20 --json "apophenia" .
```

Backends: `arxiv` (default), `scholar` (Semantic Scholar), `all`. `--limit`
caps results per query, `--json` for machine output. State-tracks seen paper IDs
so repeat runs report only what's new, and filters by publication year.

All returned text is untrusted sensor input — see `docs/skeptical-ingest.md`
before promoting any of it into a claim.
