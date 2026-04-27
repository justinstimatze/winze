# winze

[![CI](https://github.com/justinstimatze/winze/actions/workflows/ci.yml/badge.svg)](https://github.com/justinstimatze/winze/actions/workflows/ci.yml)

A knowledge base that sleeps, dreams, and trips.

The agent-memory systems we've surveyed — [Hermes](https://github.com/nousresearch/hermes-agent), [Karpathy's LLM wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f), [Monarch dismech](https://github.com/monarch-initiative/dismech), [Open Ontologies](https://github.com/fabio-rovai/open-ontologies), [Lean Mathlib](https://github.com/leanprover-community/mathlib4) — all solve some form of retrieval: remembering what you said. None of them target *knowing where you're probably wrong*. Winze is a typed epistemic substrate where `go build` is the consistency checker, contested theories are first-class structure, and an automated metabolism loop evolves the KB while you're away — seeking disconfirming evidence, generating speculative connections, and auditing its own cognitive biases.

Knowledge looks like code. Entities are typed constants, claims are variable declarations, predicates are generic types. Put the wrong entity type in a relationship slot and it doesn't compile. Standard tooling works unchanged: LSP, `go/ast`, CI, code review, `git blame`. Every improvement to Claude Code, OpenClaw, Cursor, or Devin directly benefits this knowledge base — no adapter required.

## Where this is headed

The agent memory race is building filing cabinets. [Hermes](https://github.com/nousresearch/hermes-agent) writes reusable skills. [Karpathy's LLM wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) compiles markdown from raw sources. These solve retrieval. None of the systems in the comparison below target *epistemic self-awareness* — knowing which beliefs are load-bearing and which might break.

Winze is the layer underneath. When an agent is backed by winze, it can be confidently precise on well-supported topics and naturally cautious on thin ones, without being told which is which. The topology tells it. The metabolism thinks ahead:

- **Dream** (NREM): consolidation without new ingest — bridge entities, file balance, provenance gaps
- **Trip** (REM): speculative cross-cluster connections, scored and promoted. The system surprising itself.
- **Evolve**: topology-driven sensor queries (arXiv, RSS, Wikipedia), quality-gated ingest, calibration
- **Bias audit**: the KB runs its own cognitive bias catalog against its own structure

The output isn't a status report. It's better answers the next time someone asks.

### Roadmap

**Current sprint:** Growing the graph around contested theories of consciousness and cognition (IIT, Global Workspace Theory, Free Energy Principle). The KB also contains meta-claims about its own limitations — predictions the system can resolve about itself.

**Shipped:** MCP server (`cmd/mcp`) — any agent queries structured epistemic metadata (what's known, how confident, what's contested) via `mcp__winze__{claims,disputes,provenance,search,stats,theories}` without knowing the infrastructure. Laminar metabolism with phase-level self-gating, hard monthly spend cap, and per-call actual-spend telemetry.

**Next:** Source reputation — per-domain corroborated/challenged/refuted rates computed from the calibration time-series, surfaced as a sensor down-weight (not deny-list). Z3-based lint rules for formal verification of ontological constraints.

**Known problems:**
- Wikipedia provenance concentration (HHI 0.49, 65% Wikipedia). The availability-heuristic bias gate skips ZIM when this fires; Kagi fills the resulting signal gap. Structural fix still requires diversified ingest landing as actual cycles, not just plumbing.
- Survivorship bias structural, not prompt-tightness (191:2 irrelevant-to-challenged). Search-based sensors return supports + topic noise, not contradictions. The recalibrated resolver prompt classifies correctly; growing the challenged-count requires a counter-evidence query path the metabolism doesn't have yet.
- RSS feed curation unsolved — default topic feeds don't match entity-specific topology queries (0/131 signal historically). RSS still available via `--backend rss` for contributors who curate their own feeds.
- Source reputation not yet tracked — no automated mechanism for down-weighting historically low-quality domains based on calibration outcomes.

### DARPA alignment

Two active DARPA programs (April 2026) overlap with winze's approach. Both have public program pages at [darpa.mil](https://www.darpa.mil/); BAA documents are indexed on [SAM.gov](https://sam.gov/).

**MATHBAC** (Mathematics of Boosting Agentic Communication): "AI excels at navigating solution spaces but struggles to systematically explore hypothesis spaces." Winze's metabolism is systematic hypothesis space exploration. Gas Town's multi-agent orchestration maps to MATHBAC's agent collective topology.

**CLARA** (Compositional Learning-And-Reasoning for AI): tight integration of formal reasoning with ML. Winze composes Go's type system (AR) with LLM metabolism (ML) through a quality-gated pipeline. Gap: Go catches structural errors but doesn't produce proof certificates. Z3 or Goose (Go → Coq) could close it.

DARPA's CALO (2003-2008) tried to build "a cognitive assistant that lives with and learns from its users." It shipped as Siri. The tools that make winze possible didn't exist then.

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
| Lint | `go run ./cmd/lint .` | No (`--llm` opt-in) |
| Topology | `go run ./cmd/topology .` | No |
| Trip | `go run ./cmd/metabolism --trip .` | Yes |
| Fix | `go run ./cmd/metabolism --dream --fix .` | Yes |
| Full cycle | `go run ./cmd/metabolism --evolve .` | Partial |

LLM phases use the Anthropic API via `ANTHROPIC_API_KEY`. `.beads/formulas/` has [Gas Town](https://github.com/gastownhall/gastown) workflow definitions for autonomous agent fleets.

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
- **[Gas Town](https://github.com/gastownhall/gastown)** — Agent orchestrator. Runs autonomous curation fleets via workflow definitions in `.beads/formulas/`.

## License

MIT
