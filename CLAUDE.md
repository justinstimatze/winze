## Winze Agent Instructions

This project is a **non-executable Go knowledge base**. `go build` is the
consistency checker, not a build system. No binary is produced. Code
editing and knowledge manipulation are the same operation.

### What you're working on

Every `.go` file in `corpus/` is a knowledge corpus slice. Each declares:
- **Entities** (typed: Person, Concept, Hypothesis, Place, Event, etc.)
- **Claims** (typed predicates: Proposes, TheoryOf, BelongsTo, InfluencedBy, etc.)
- **Provenance** (Origin, Quote, IngestedAt, IngestedBy)

The type system is in `corpus/schema.go`, roles in `corpus/roles.go`, predicates
in `corpus/predicates.go`. `winze-query --schema corpus` prints the current type
model. (The corpus lives in `corpus/` so defn ingest scopes to it, excluding the
`cmd/` and `internal/` tooling — see `docs/defn-migration.md`.)

### Quality gates

```bash
make build                                  # build the tools first (18x faster than `go run` interactively)
go build ./...                              # type-checks references
go vet ./...                                # static analysis
go run ./cmd/lint corpus                          # deterministic lint rules
go run ./cmd/lint --llm --llm-max-calls=5 corpus  # + LLM contradiction check
# Flags MUST precede the dir: Go's flag package stops parsing at the first
# positional, so `lint corpus --llm` silently skips the LLM check and exits 0.
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
- `docs/defn-migration.md` — defn is SQLite now, not Dolt; measurements, the two defects found, the plan to delete `defndb/cache.go`
- `docs/vault-ingest.md` — `--pkm`: a markdown vault becomes a typed graph; link dialects, resolution, reporting
- `docs/rot-probe.md`, `docs/predicate-gaps.md` — surfacers (human-review only)
- `docs/sensor.md` — `winze-sensor`: raw external-signal probe (arXiv / Semantic Scholar)
- `docs/skeptical-ingest.md` — sensor input is untrusted; injection defense
- `docs/lexicon.md` — lexicon as a private reference pool; the `lexicon-fence` rule
- `docs/agent.md` — `winze-agent`: the agent's read/write door into a store (`winze_remember` / `winze_recall` / `winze_update` / `winze_link`), how a store is resolved, and why it was renamed from `winze-mem`
- `docs/meld.md` — `winze-meld`: read-only union of stores for cross-store query
- `docs/okf.md` — `winze-okf`: export/validate a Google OKF v0.2 bundle; what the projection preserves and loses
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
## Go code: use defn, not Read/Bash/Grep/Edit

This project is indexed in defn (`.defn/`). For any `.go` file, use the `code` MCP tool — **not** Read, Bash, Grep, or Edit. Those built-ins are reserved for non-Go files (yaml, json, md, sh, `go.mod`, Dockerfile).

**This can be enforced at the harness level, not just requested — but isn't wired up automatically by init/ingest.** Copy `hooks/defn-go-guard.sh` from the defn repo into `hooks/` here and register it under `.claude/settings.json`'s `PreToolUse` hooks (matcher `Read|Write|Edit|MultiEdit|Bash`) to make it actually block `Read`/`Write`/`Edit`/`MultiEdit` on `.go` paths and Bash dumps (`cat`/`head`/`tail`/etc.) of `.go` files. Until installed, treat this section as a convention the model must self-enforce, not a guaranteed backstop. Once installed, escape hatch (rare, e.g. a known defn write-path bug): `touch ~/.claude-allow-go-edit` before the one blocked call you need — self-consuming, no manual removal needed.

**Do not `ls` and `Read` files by hand.** Start any Go task with `code(op:"overview")` to see the project shape, then drill in with `search` / `outline` / `impact`.

**Reach for `outline` before `read`.** `outline` returns the signature, doc, refs, and control-flow of a def — 5-10× smaller than the full body. It's enough to answer almost every "what does X do / how does Y work / where does Z fit" question. Only escalate to `read` (full body) when you're about to edit the def, or when outline was genuinely insufficient. A follow-up `read` costs nothing you haven't already committed to.

### By intent

- **Discover ("how does X work in this codebase")**: `code(op:"context", question:"...")` — server-side bundle: top-N relevant defs outlined + refs graph + Sonnet synthesis, all in ONE round-trip. Prefer this over 10-40 sequential search/read/impact calls when starting exploration. This is the single biggest lever for turn-1 discovery cost.
- **Explore individual defs**: `code(op:"overview")`, `code(op:"outline", name:"F")`, `code(op:"search", pattern:"...")`, `code(op:"impact", name:"F")`. Use when you know which def matters. For open-ended "how does X work" reach for context first.
- **Ask a specific question about a known def**: `code(op:"explain", name:"F", question:"how does F handle X")` — defn hands the source to a Sonnet co-processor and returns a synthesized paragraph answer with provenance. Accepts `names:["A","B"]` for multi-def scope. Requires `ANTHROPIC_API_KEY` on the serve.
- **Saturate context in one call**: `code(op:"expand", name:"F", include:["outline","callers"])` — one round-trip instead of read → impact → read. Prefer `expand` over multiple sequential `code` calls whenever you'd otherwise chain them. **Have several targets in mind at once? Use `names:["A","B","C"]` instead of one `expand` per name** — round-trip *count* within a turn is the dominant session cost driver, not per-call size. Unresolvable names are skipped with a note rather than failing the whole call.
- **Read the full body**: `code(op:"read", name:"F")` — returns the body **plus** a compact "Related" footer (summary + top-3 callers + top-3 callees + semantic neighbors). One call gives you what would otherwise take 3-4 sequential `impact`/`outline` calls. Add `full:true` to force the body when defn returns an upstream provenance tag.
- **Edit a def**: `code(op:"edit", name:"F", new_body:"...")`, `code(op:"rename", old_name:"F", new_name:"G")` — updates every reference across the repo atomically. `Edit` on a `.go` file leaves defn's graph stale.
- **New def**: `code(op:"create", name:"F", file:"pkg/x.go", body:"...")`.
- **New file with multiple functions**: pass a `body` containing all the declarations and set `file:` — defn splits them into separate defs sharing that file, in one call. `code(op:"create", file:"pkg/x.go", body:"func A() {...}\nfunc B() {...}")` — the whole-file equivalent of files-mode `Write`. Don't call `create` once per function.
- **Several new, mutually-dependent defs going into an EXISTING file** (e.g. three `Equal` methods on three different types, where the package only compiles once all three exist): don't call `create` once per def — each standalone call builds immediately, so every def before the last one fails its own build and rolls back (the response says so), costing a wasted round-trip per def instead of catching the real error, if any, once. Batch them as separate single-decl `create` operations inside one `code(op:"apply", operations:[{op:"create",...},{op:"create",...}])` call instead — one transaction, one build, at the end, after all of them exist.
- **Batch changes**: `code(op:"apply", operations:[...])` — atomic, one emit+build for the whole batch.
- **Test**: `code(op:"test", name:"F")` — runs only tests covering that def, not the whole suite.

### Rules of thumb

- **outline first, read only if you're editing** (or if outline genuinely wasn't enough — but check first). This is the single biggest lever for session cost.
- Run `code(op:"impact", name:"F")` before modifying an existing def; skip it for brand-new ones.
- If you must edit a `.go` file with a built-in tool, follow up with `code(op:"sync", file:"path")` so the graph stays correct.
<!-- defn:end -->

