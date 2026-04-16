# winze

[![CI](https://github.com/justinstimatze/winze/actions/workflows/ci.yml/badge.svg)](https://github.com/justinstimatze/winze/actions/workflows/ci.yml)

A knowledge base that sleeps, dreams, and trips — audits itself for cognitive biases, predicts where it's wrong, and gets less wrong over time.

## Why

Your agents are getting better at writing code. They're not getting better at remembering. Context windows are a bottleneck, RAG is lossy, vector search doesn't know what a contradiction is, and a folder of markdown files doesn't get smarter over time. Meanwhile, every coding agent on earth already knows how to navigate, edit, lint, test, and review Go source files.

Winze makes knowledge look like code. Entities are typed constants, predicates are generic types, claims are variable declarations. `go build` is the consistency checker — put the wrong entity type in a relationship slot and it doesn't compile. Standard tooling works unchanged: LSP, `go/ast`, CI, code review, `git blame`. Every improvement to Claude Code, OpenClaw, Cursor, or Devin directly benefits this knowledge base — no adapter required.

Winze is a crude, slow, expensive approximation of a system that maintains a model of reality, notices when that model is fragile, seeks disconfirming evidence, and updates its confidence. Built on Go source files and [Gas Town](https://github.com/gastownhall/gastown).

## What it does

**Catches knowledge errors at compile time.** Put a Person where a Hypothesis belongs — `go build` fails pointing at the exact claim.

**Audits itself for cognitive biases.** Nine auditors, each mapping a bias the KB catalogs to a structural check on itself. Two currently triggered — we built them to find exactly this.

**Predicts where it's wrong.** Topology analysis identifies structurally fragile hypotheses. The metabolism loop polls arXiv, RSS, and Wikipedia for evidence. 40 hypotheses tracked, 8 corroborated, 1 challenged autonomously.

**Runs while you sleep.** Sense → evaluate → ingest → dream → trip → calibrate. One command, nightly cron, any agent runtime. Most phases need no API key. [Gas Town](https://github.com/gastownhall/gastown) workflow definitions ship in `.beads/formulas/` for autonomous agent fleets — curation patrols, health checks, depth-first ingest.

**Ingests your Obsidian vault.** Markdown notes → typed entities with provenance, navigable by agents, auditable by the compiler.

## Quick start

Requires Go 1.26+.

```bash
git clone https://github.com/justinstimatze/winze.git && cd winze
go build ./...                   # type-check the seed corpus
go test ./...                    # invariant tests
go run ./cmd/query --stats .     # what's in the KB
go run ./cmd/lint .              # structural health
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

Open `starter.go`:

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

## Bias audit

| Auditor | What it measures | Finding |
|---------|-----------------|---------|
| Confirmation bias | Corroboration rate among signal cycles | 58% (PASS) |
| Anchoring | File age vs. claim density correlation | rho = -0.02 (PASS) |
| Clustering illusion | File grouping vs. topology cluster overlap | 44% Jaccard (PASS) |
| Availability heuristic | Provenance source concentration | **0.70 HHI — 83% Wikipedia (TRIGGERED)** |
| Survivorship bias | Irrelevant-to-challenged ratio | **8:0 — zero challenges (TRIGGERED)** |
| Framing effect | Evaluative language in Briefs | 3% (PASS) |
| Dunning-Kruger | Low-complexity entities escaping detection | 74% vs 37% gap (PASS) |
| Base rate neglect | Predicate distribution entropy | 4.3 bits (PASS) |
| Premature closure | Cliches + DAG leaf detection | 31 structural (PASS) |

## Metabolism

| Phase | Command | Needs LLM? |
|-------|---------|------------|
| Sense | `go run ./cmd/metabolism .` | No |
| Dream | `go run ./cmd/metabolism --dream .` | No |
| Bias audit | `go run ./cmd/metabolism --bias .` | No |
| Calibrate | `go run ./cmd/metabolism --calibrate .` | No |
| Lint | `go run ./cmd/lint .` | No (`--llm` opt-in) |
| Topology | `go run ./cmd/topology .` | No |
| Trip | `go run ./cmd/metabolism --trip .` | Yes |
| Fix | `go run ./cmd/metabolism --dream --fix .` | Yes |
| Full cycle | `go run ./cmd/metabolism --evolve .` | Partial |

**Dream** (NREM): consolidation without new ingest — bridge entities, file balance, provenance gaps. **Trip** (REM): speculative cross-cluster connections scored across temperature and prompt type. **Fix**: Brief tightening with quality gates and auto-revert.

LLM phases use the Anthropic API via `ANTHROPIC_API_KEY`. `CLAUDE.md` has agent instructions for Claude Code — adapt to your runtime's format. `.beads/formulas/` has [Gas Town](https://github.com/gastownhall/gastown) workflow definitions as readable TOML.

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

- **[defn](https://github.com/justinstimatze/defn)** — AI-native code database for Go. Winze uses it as an MCP server for structured queries across the KB: multi-hop entity lookups, cross-file claim analysis, provenance tracing.
- **[adit](https://github.com/justinstimatze/adit-code)** — Structural analysis for AI-edited codebases. Scores corpus files for agent-writability during dream cycles and flags maintenance priorities.
- **[slimemold](https://github.com/justinstimatze/slimemold)** — Reasoning topology observer. Monitors the epistemic support graph for load-bearing claims that have never been challenged.
- **[plancheck](https://github.com/justinstimatze/plancheck)** — Predicts which files agents will miss. Used to validate implementation plans before execution.
- **[Gas Town](https://github.com/gastownhall/gastown)** — Agent orchestrator. Runs autonomous curation fleets against the KB via workflow definitions in `.beads/formulas/`.

## Vision & roadmap

### Where this is headed

The agent memory race of 2026 is building filing cabinets. MemPalace stores conversations verbatim. Hermes writes reusable skills. Karpathy's LLM wiki compiles markdown from raw sources. They all solve retrieval — remembering what you said. None of them solve *epistemic self-awareness* — knowing where you're probably wrong and doing something about it.

Winze is the layer underneath. When an agent is backed by winze, it doesn't just remember — it knows which of its beliefs are load-bearing and which might break. It can be confidently precise on well-supported topics and naturally cautious on thin ones, without being told which is which. The topology tells it.

The metabolism isn't a maintenance cron job. It's the system thinking ahead:
- **Dream** consolidates: finds structural weaknesses, bridge entities, provenance gaps
- **Trip** speculates: generates unexpected connections between distant clusters, scores them, promotes the good ones
- **Evolve** deepens: topology-driven sensor queries, quality-gated ingest, calibration
- **Bias audit** self-corrects: the KB runs its own cognitive bias catalog against its own structure

The output of all this isn't a status report. It's better answers the next time someone asks.

### Roadmap

**Current sprint:** Grow the graph around contested theories of consciousness and cognition. IIT, Global Workspace Theory, and Free Energy Principle are being seeded via PKM ingest and deepened by the metabolism. The KB also contains meta-claims about its own limitations (reification risk, finite ontology incompleteness, self-audit epiphenomenalism) — predictions the system can resolve about itself.

**Next:** Winze as an MCP server. Any agent queries the KB for structured epistemic metadata — what's known, how confident, what's contested — without knowing the infrastructure exists. Z3-based lint rules for stronger formal verification of ontological constraints (cycle detection, temporal ordering, functional uniqueness).

**Side project — blind field discovery:** Can the metabolism autonomously map a scientific field it doesn't know? Phase 1: rediscover periodicity from raw NIST atomic data (verifiable calibration run). Phase 2: map quantum computing from arXiv papers only, with Wikipedia blinders on. Compare output to Wikipedia's article structure after the experiment. The payoff is both a validation of the approach and a useful domain expert as a byproduct.

**Known problems to fix:**
- Calibration loop is near-tautological (searching Wikipedia for Wikipedia-sourced topics). Needs `gap_confirmed` vs `no_gap` distinction instead of just "articles found."
- Triggered bias auditors (availability heuristic, survivorship bias) don't yet gate the next metabolism phase. Self-audit that doesn't change behavior is epiphenomenal.
- 88% Wikipedia provenance concentration. The metabolism should prioritize arXiv and RSS backends when availability heuristic triggers.

### DARPA alignment

Two active DARPA programs (April 2026) have significant overlap with winze's approach:

**MATHBAC** (Mathematics of Boosting Agentic Communication): "AI excels at navigating solution spaces but struggles to systematically explore hypothesis spaces." Winze's topology-driven metabolism is systematic hypothesis space exploration. The typed claim format maps to MATHBAC's "compact, generalizable nuggets." Gas Town's multi-agent orchestration (polecats, witness, mayor) maps to MATHBAC's agent collective topology.

**CLARA** (Compositional Learning-And-Reasoning for AI): Tight integration of formal reasoning with ML, demonstrably trustworthy. Winze composes Go's type system (formal reasoning) with LLM-powered metabolism (ML) through a quality-gated pipeline. The gap: Go's type system catches structural errors but doesn't produce formal proof certificates. Z3 SMT solver bindings (go-z3) or Goose (Go → Coq translation, MIT PDOS) could close this gap.

DARPA's CALO project (2003-2008) tried to build "a cognitive assistant that lives with and learns from its users." It shipped as Siri. The tools that make winze possible — LLMs, full encyclopedias on a laptop, embedding search — didn't exist then.

## Prior art

| Project | Substrate | Consistency | Contestation? | Self-calibrating? |
|---------|-----------|-------------|--------------|-------------------|
| [MemPalace](https://github.com/milla-jovovich/mempalace) | ChromaDB + SQLite | None (verbatim storage) | No | No |
| [Hermes Agent](https://github.com/nousresearch/hermes-agent) | Skill documents | Behavioral testing | No | No |
| [Karpathy LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) | Markdown | String-level lint | No | No |
| [Monarch dismech](https://github.com/monarch-initiative/dismech) | YAML + LinkML | Schema validation + CI | No | No |
| [Open Ontologies](https://github.com/fabio-rovai/open-ontologies) | RDF/OWL | OWL2-DL reasoning | No | No |
| Prolog/Datalog | Logic programs | Inference engine | No | No |
| [Lean Mathlib](https://github.com/leanprover-community/mathlib4) | Dependent types | Proof checker | No | No |
| **winze** | Go source | `go build` + 7 lint rules + 9 bias auditors | **Yes (typed TheoryOf/Disputes)** | **Yes (topology → predict → calibrate)** |

## License

MIT
