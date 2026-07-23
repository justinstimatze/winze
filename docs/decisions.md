# Decision log (the Supersedes graph)

A decision log answers "why is it this way, and is that still the current call?"
Git log answers the first badly and the second not at all. Winze answers both
without a new store, because a decision is not a separate ontological category —
it's a memory with a **lifecycle**, and the lifecycle is the typed part.

```bash
winze-query --decisions .            # current decisions, tracing what each superseded
winze-query --decisions --json .     # machine form
```

## How it works

A decision is recorded like any durable project fact (`winze_remember`, or a
memory entry). When a later decision replaces an earlier one, that relationship
is a typed `Supersedes` claim — winze-memory's predicate, written through the
build gate:

```
winze_link(from=NewerDecision, to=OlderDecision, relation="Supersedes",
           rationale="why the newer one replaces it")
```

`--decisions` walks the `Supersedes` graph and shows each **current** decision
(one nothing supersedes) followed by the chain it replaced:

```
decisions — 1 current, tracing what each superseded:

  ● Winze sprint vision and positioning
    ↑ supersedes  Winze strategic positioning and sprint vision
    ↑ which superseded  Strategic positioning April 2026
```

## Why this shape

- **The compiler gates the graph.** A `Supersedes` pointing at a deleted memory
  won't build, so "what's current" is a query over real edges, not a grep over
  prose that may have gone stale.
- **One home per decision.** Decisions are the subset of project-memory that
  gets superseded, so they live in the memory store that already holds them. A
  separate decisions store would duplicate the same decision in two places — the
  coin-time-dedup rot winze exists to prevent.
- **Supersession, not deletion.** Three near-identical positioning memories
  aren't a duplicate to merge away (that loses the history) — they're a chain.
  The `Supersedes` links mark which is current while keeping the older reasoning
  readable and reachable.

The link is a `Conjecture` (winze's own assertion about which decision is
current), not a `Provenance` — it carries a `Rationale`, no source `Quote`.
