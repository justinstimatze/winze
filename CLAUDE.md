## Winze Agent Instructions

This project is a **non-executable Go knowledge base**. `go build` is the
consistency checker, not a build system. No binary is produced. Code
editing and knowledge manipulation are the same operation.

### What you're working on

Every `.go` file in the root is a knowledge corpus slice. Each declares:
- **Entities** (typed: Person, Concept, Hypothesis, Place, Event, etc.)
- **Claims** (typed predicates: Proposes, TheoryOf, BelongsTo, InfluencedBy, etc.)
- **Provenance** (Origin, Quote, IngestedAt, IngestedBy)

The type system is in `schema.go`, roles in `roles.go`, predicates in
`predicates.go`. `winze-query --schema .` prints the current type model.

### Quality gates

```bash
make build                                  # build the tools first (18x faster than `go run` interactively)
go build ./...                              # type-checks references
go vet ./...                                # static analysis
go run ./cmd/lint .                         # deterministic lint rules
go run ./cmd/lint . --llm --llm-max-calls=5 # + LLM contradiction check
```

The build gate (`gofmt -w && go build . && go vet .`, revert on failure) is the
load-bearing discipline this project was built around. Every write path runs it.
Do NOT relax that path.

### Mirror-source-commitments

Only encode claims the source explicitly commits to. Use `Provenance.Quote`
with exact source text. Do not fabricate relationships. Brief-level references
are fine for connections the source doesn't explicitly make.

A claim's `Prov` is an `Attribution` — either a sourced `Provenance` (Quote =
exact source text) or a `Conjecture` (winze's OWN generation: trip cycles,
cross-cluster analogy, synthesis). `Conjecture` has **no `Quote` field by
design** — the compiler forbids a generated claim from wearing a fabricated
source, which closes the trip-fabrication failure mode structurally rather than
by lint. When winze generates a speculative connection, back it with
`Conjecture` (`GeneratedBy`, `Rationale`, …), never a `Provenance` with an
invented Quote.

### Schema accretion

Do NOT invent predicates speculatively. Wait for the forcing function: a source
that explicitly commits to a relationship no existing predicate captures. When a
third occurrence of a pattern surfaces, promote it to a named discipline.

### Domain boundary

The KB's domain is the epistemology of minds — how minds (human and artificial)
build, validate, and fail at modeling reality. Concepts are in-domain when they
illuminate how knowledge is constructed, contested, or mistaken. Ingest that
doesn't serve this domain is bloat. The metabolism loop is depth-first: deepen
thin contested neighborhoods before expanding to new hypotheses.

### Detailed reference (recalled on demand)

The command references and deep-dives live in `docs/` and are surfaced per-prompt
by the docs-recall hook (`winze-query --docs-recall "<prompt>" .`) — this file
stays small so it can stay current. To pull a topic yourself:

- `docs/tooling.md` — building the binaries, timings
- `docs/authoring.md` — `cmd/add`: inline / provenance-var / conjecture / batch / propose
- `docs/editing.md` — `cmd/edit`: rename, merge, concurrent-write safety
- `docs/query.md` — the read side (`cmd/query`), all modes
- `docs/decisions.md` — the decision log: `--decisions` over the Supersedes graph
- `docs/lint-rules.md` — the lint rules; structural-dedup and brief-drift in depth
- `docs/pragmas.md` — `//winze:contested`, `//winze:functional`, `//winze:mentions`
- `docs/predicates.md` — the predicate families
- `docs/topology.md` — structural vulnerability analysis
- `docs/metabolism.md` — the `--evolve` loop, phases, gating, budget, sharing
- `docs/rot-probe.md`, `docs/predicate-gaps.md` — surfacers (human-review only)
- `docs/sensor.md` — `winze-sensor`: raw external-signal probe (arXiv / Semantic Scholar)
- `docs/skeptical-ingest.md` — sensor input is untrusted; injection defense
- `docs/meld.md` — `winze-meld`: read-only union of stores for cross-store query
- `docs/observatory.md` — `winze-observatory`: standalone fleet dashboard
- `docs/benchmark.md` — `winze-benchmark`: retrieval benchmark (grep/bm25/defn/ast)
- `docs/mcp-tools.md` — defn / adit / wikipedia-zim
- `docs/multi-session-write-shape.md`, `docs/typed-citation.md` — the write shape
- `docs/docs-recall.md` — how this recall works

### Session completion

Commit and push code changes when a unit of work is done — work isn't complete
until `git push` succeeds (never leave it stranded locally). Run the quality
gates first (tests, `go build ./...`, `go vet ./...`, lint) if code changed.
That's it: no ticketing system, no separate data plane.

<!-- defn:begin -->
## Code Navigation and Editing

This project is indexed in defn. Use the `code` MCP tool for **Go code**:

```
code(op: "read", name: "handleEdit")           -- full source by name
code(op: "read", name: "server.go:272")        -- or by file:line
code(op: "impact", name: "Render")             -- blast radius + test coverage
code(op: "edit", name: "Foo", new_body: "...") -- edit, auto-emit + build
code(op: "search", pattern: "%Auth%")          -- name pattern (% wildcard)
code(op: "search", pattern: "authentication")  -- body text search
code(op: "test", name: "Render")               -- run affected tests only
code(op: "sync")                               -- re-ingest after file edits
```

All ops: read, search, impact, explain, untested, edit, create, delete, rename, move, test, apply, diff, history, find, sync, query, overview, patch.

**Both editing paths work.** `code(op:"edit")` updates the database, emits files, and rebuilds references automatically. File tools (Read, Edit) work too — call `code(op:"sync")` after editing Go files.

Prefer defn for Go code (fewer steps, auto-build verification). Use Read/Edit/Grep for non-Go files.

**Rule of thumb:** Always run impact before modifying an existing definition. Skip it for brand-new definitions.
<!-- defn:end -->
