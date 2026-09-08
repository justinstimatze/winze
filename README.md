# winze

[![CI](https://github.com/justinstimatze/winze/actions/workflows/ci.yml/badge.svg)](https://github.com/justinstimatze/winze/actions/workflows/ci.yml)

A knowledge base that sleeps, dreams, and trips.

The agent-memory field splits on one line: everything that retrieves does it over something authored first. [Letta/MemGPT](https://github.com/letta-ai/letta) promotes to a tier, [mem0](https://github.com/mem0ai/mem0) extracts facts through a pipeline, Claude Code's own memory and Anthropic's `memory_20250818` tool wait on a human or agent to write a file, Cursor and Windsurf keep short auto-notes explicitly not meant to answer "what did we do yesterday." Nobody does automatic, derived retrieval over a user's raw session history with no authoring step — and winze doesn't either; it's a curated store like the rest of the field (see [docs/sota-memory-systems-survey-2026-08-31.md](docs/sota-memory-systems-survey-2026-08-31.md)). What curation buys, with `go build` as the enforcement mechanism instead of a database schema: a citation to a code symbol or another claim fails the build the moment it goes stale, contested theories are first-class structure instead of prose, and an automated metabolism loop evolves the KB while you're away — seeking disconfirming evidence, generating speculative connections, and auditing its own cognitive biases. None of the systems above are multi-writer at all — winze is: concurrent agent sessions across separate git worktrees can write into one shared store without corrupting it, because the same compiler gate that catches a stale reference also turns a silent duplicate concept into a build error instead of two competing wiki pages. Running today, not hypothetical: a real project shares one store across 11 concurrent worktrees. See [docs/multi-session-write-shape.md](docs/multi-session-write-shape.md).

Knowledge looks like code. Entities are typed constants, claims are variable declarations, predicates are generic types. Put the wrong entity type in a relationship slot and it doesn't compile. Standard tooling works unchanged: LSP, `go/ast`, CI, code review, `git blame`. Every improvement to Claude Code, OpenClaw, Cursor, or Devin directly benefits this knowledge base — no adapter required.

## Typed citation

The primitive under all of it is the **typed citation**: a reference whose
target existence is checked by the compiler, so it cannot go stale silently.
Concept→concept links, claim→source provenance, and doc→code references are the
same shape — a citation with a type the build keeps honest. A documentation
entity can cite a live code symbol *by value* (`winze_self.go` does this on
winze's own internals): rename or delete the symbol and the KB stops compiling.
Most tools in this space *detect* drift after the fact and route a proposed fix
through human approval; winze *prevents* it — a stale reference is a build
error, not something a reviewer might notice. The format is an implementation
detail: a user adds and queries knowledge by talking to an agent that maps the
conversation to a typed claim behind the gate, and never needs to know it is
code. See [docs/typed-citation.md](docs/typed-citation.md).

## Where this is headed

The metabolism is what curation compounds into once it's run long enough. A resumed thread that can't tell which of its own beliefs have gone stale is confident wrongness with better latency — so dream, trip, and evolve spend idle time precomputing exactly that. When an agent is backed by winze, it can be confidently precise on well-supported topics and naturally cautious on thin ones, without being told which is which. The topology tells it.

- **Dream** (NREM): consolidation without new ingest — bridge entities, file balance, provenance gaps
- **Trip** (REM): speculative cross-cluster connections, scored and promoted. The system surprising itself.
- **Evolve**: topology-driven sensor queries (arXiv, RSS, Wikipedia), quality-gated ingest, calibration
- **Bias audit**: the KB runs its own cognitive bias catalog against its own structure

The output isn't a status report. It's better answers the next time someone asks.

Current sprint, what's shipped and next, OKF interop, known problems, and DARPA alignment all live in [ROADMAP.md](ROADMAP.md).

DARPA's CALO (2003-2008) tried to build "a cognitive assistant that lives with and learns from its users." It shipped as Siri. The tools that make winze possible didn't exist then.

## Quick start

Requires Go 1.26+.

```bash
git clone https://github.com/justinstimatze/winze.git && cd winze
go build ./...                   # type-check the seed corpus
go test ./...                    # invariant tests
go run ./cmd/query --stats .     # what's in the KB
go run ./cmd/lint corpus              # structural health
go run ./cmd/metabolism --bias . # bias self-audit
go run ./cmd/metabolism --evolve . # full autonomous cycle
```

```bash
go run ./cmd/query "consciousness" .        # search
go run ./cmd/query --theories "apophenia" .  # competing theories
go run ./cmd/query --disputes .              # active disputes
go run ./cmd/query --ask "What theories compete on consciousness?" .  # LLM-powered
```

### Your domain in 5 minutes

```bash
./script/reset-corpus.sh  # removes seed corpus, keeps schema + starter.go
```

Open `corpus/starter.go`:

```go
package winze

var bachSource = Provenance{
    Origin:     "Wikipedia 2025-12 / Well-Tempered_Clavier",
    IngestedAt: "2026-04-13",
    IngestedBy: "your-name",
    Quote:      "Bach wrote the collection to demonstrate the musical possibilities of well temperament.",
}

var Bach = Person{&Entity{
    ID:    "johann-sebastian-bach",
    Name:  "Johann Sebastian Bach",
    Kind:  "person",
    Brief: "Baroque composer. Wrote the Well-Tempered Clavier to prove equal temperament works.",
}}

var WellTemperedThesis = Hypothesis{&Entity{
    ID:    "well-tempered-thesis",
    Name:  "Well-tempered thesis",
    Kind:  "hypothesis",
    Brief: "All 24 major and minor keys are musically viable in a single tuning system.",
}}

var BachProposesWT = Proposes{
    Subject: Bach,
    Object:  WellTemperedThesis,
    Prov:    bachSource,
}
```

`go build ./...` passes. Add a competing theory for the same concept and the lint rule fires — it's contested now.

### Obsidian vault ingest

```bash
go run ./cmd/metabolism --pkm /path/to/vault .  # markdown → typed Go
go run ./cmd/metabolism --dream --bias .         # contradictions + blind spots
```

## Prior art

| Project | Substrate | Consistency | Contestation? | Self-calibrating? |
|---------|-----------|-------------|--------------|-------------------|
| [Hermes Agent](https://github.com/nousresearch/hermes-agent) | Skill documents | Behavioral testing | No | No |
| [Karpathy LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) | Markdown | String-level lint | No | No |
| [Monarch dismech](https://github.com/monarch-initiative/dismech) | YAML + LinkML | Schema validation + CI | No | No |
| [Open Ontologies](https://github.com/fabio-rovai/open-ontologies) | RDF/OWL | OWL2-DL reasoning | No | No |
| Prolog/Datalog | Logic programs | Inference engine | No | No |
| [Lean Mathlib](https://github.com/leanprover-community/mathlib4) | Dependent types | Proof checker | No | No |
| **winze** | Go source | `go build` + 7 lint rules + 9 bias auditors | **Yes (typed TheoryOf/Disputes)** | **Yes (topology → predict → calibrate)** |

## Metabolism

| Phase | Command | Needs LLM? |
|-------|---------|------------|
| Sense | `go run ./cmd/metabolism .` | No |
| Dream | `go run ./cmd/metabolism --dream .` | No |
| Bias audit | `go run ./cmd/metabolism --bias .` | No |
| Calibrate | `go run ./cmd/metabolism --calibrate .` | No |
| Lint | `go run ./cmd/lint corpus` | No (`--llm` opt-in) |
| Topology | `go run ./cmd/topology .` | No |
| Trip | `go run ./cmd/metabolism --trip .` | Yes |
| Fix | `go run ./cmd/metabolism --dream --fix .` | Yes |
| Full cycle | `go run ./cmd/metabolism --evolve .` | Partial |

LLM phases use the Anthropic API via `ANTHROPIC_API_KEY`. Autonomous operation is scheduler-agnostic: any cron/CI/systemd timer fires `metabolism --evolve .` on a clock, and winze owns the per-phase gating and budget guard (see `script/setup-autonomous.sh`).

## Bias audit

| Auditor | What it measures | Finding (live, ~300-cycle corpus) |
|---------|-----------------|---------|
| Confirmation bias | Corroboration rate among signal cycles | 13% (PASS) |
| Anchoring | File age vs. claim density correlation | rho = 0.32 (PASS) |
| Clustering illusion | File grouping vs. topology cluster overlap | 17% Jaccard (PASS) |
| Availability heuristic | Provenance source concentration | **0.49 HHI — 65% Wikipedia (TRIGGERED)** |
| Survivorship bias | Irrelevant-to-challenged ratio | **191:2 — 95.5x ratio (TRIGGERED)** |
| Framing effect | Evaluative language in Briefs | 2% (PASS) |
| Dunning-Kruger | Low-complexity entities escaping detection | 85% vs 37% gap (PASS) |
| Base rate neglect | Predicate distribution entropy | 4.06 bits (PASS) |
| Premature closure | Cliches + DAG leaf detection | 34 info-level (PASS) |

## Schema

**Entity:** ID, Name, Kind, Brief, Aliases.

**Roles:** 16 — Person, Organization, Place, Event, Facility, Substance, Instrument, Hypothesis, Concept (grounded in Schema.org / WordNet / Wikidata), plus 7 design roles for `--pkm` creative-work analysis.

**Predicates:** `BinaryRelation[S, O]` and `UnaryClaim[S]`, 40+ across families:

| Family | Predicates |
|--------|-----------|
| Attribution | Proposes, Disputes, ProposesOrg, DisputesOrg |
| Theory | TheoryOf (`//winze:contested`), HypothesisExplains |
| Taxonomy | BelongsTo, DerivedFrom, IsCognitiveBias, IsPolyvalentTerm |
| Authorship | Authored, AuthoredOrg, CommentaryOn, AppearsIn |
| Spatial | LocatedIn, LocatedNear, OccurredAt |
| People | InfluencedBy, WorksFor, AffiliatedWith, InvestigatedBy |
| Prediction | Predicts, Credence, ResolvedAs (`//winze:functional`) |
| Design | AppliesToWork, WorkHasLayer, WorkHasPhase, WorkHasProtectedLine, WorkCommitsToNeverAnswering |

**Provenance:** Every claim carries origin, ingest date, ingester, exact source text.

**Annotations:** `//winze:contested` (competing theories expected), `//winze:functional` (one value per subject).

## Design principles

- **Mirror-source-commitments:** Only encode what the source explicitly states.
- **Schema accretion:** Don't invent predicates speculatively. Wait for the forcing function.
- **Prose is I/O not state:** Source documents are transient; the KB is canonical.
- **LLM as expensive lint rule:** Opt-in, budgeted, one rule among many.
- **Depth over breadth:** Deepen thin contested neighborhoods before expanding.

## Built with

- **[defn](https://github.com/justinstimatze/defn)** — AI-native code database for Go. Structured queries across the KB: multi-hop entity lookups, cross-file claim analysis, provenance tracing.
- **[adit](https://github.com/justinstimatze/adit-code)** — Structural analysis for AI-edited codebases. Scores corpus files for agent-writability.
- **[slimemold](https://github.com/justinstimatze/slimemold)** — Reasoning topology observer. Monitors the epistemic support graph for load-bearing unchallenged claims.
- **[plancheck](https://github.com/justinstimatze/plancheck)** — Predicts which files agents will miss. Validates implementation plans before execution.

## License

MIT
