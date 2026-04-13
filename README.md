# winze

A knowledge base that audits itself for cognitive biases, predicts where it's wrong, and is right 88% of the time.

Built on Go's type system. The compiler catches ontological errors. No new query language. No new runtime. Standard tooling works unchanged.

## What this is

189 entities. 242 claims. 132 directed epistemic support edges. 9 cognitive bias auditors. An 88% hit rate on structural fragility predictions.

Winze is a knowledge base where source code IS the knowledge. Entities are typed constants, predicates are generic types, claims are variable declarations. `go build` is the consistency checker — it enforces referential integrity and type safety across every claim in the graph. No binary is produced.

```bash
go build ./...              # the compiler catches ontological errors
go run ./cmd/lint .         # 7 lint rules check structural health
go run ./cmd/metabolism --bias .  # the KB audits itself for cognitive biases
```

## Why this exists

The 2026 agentic coding wave is building infrastructure for LLM agents to collaboratively write and maintain code. Winze repurposes that infrastructure for collaborative knowledge maintenance. Every improvement to Claude Code, Cursor, or Devin directly improves the fleet's ability to maintain this knowledge base — because the knowledge base *is* code.

Standard tooling works unchanged: LSP navigation, `go/ast` analysis, CI pipelines, code review, `git diff`, `git blame`.

## Five claims

### 1. The compiler catches knowledge errors

Change a Person entity to a Concept where a Person is required. `go build` fails with a type error pointing to the exact claim that references it. No YAML validator, no custom schema language — Go's type system enforces ontological constraints at compile time.

```go
// This builds:
var AliceProposes = Proposes{Subject: Alice, Object: AttentionHypothesis, Prov: mySource}

// Change Alice from Person to Concept → build fails:
//   cannot use Alice (variable of type Concept) as Person value
```

### 2. It finds its own gaps and goes looking

The topology analyzer identifies structurally fragile hypotheses — single-source, uncontested, thin-provenance. The metabolism loop generates sensor queries for each fragile hypothesis, polls external sources (arXiv, Wikipedia), and logs what it finds.

```bash
go run ./cmd/topology .              # 5 vulnerability detectors
go run ./cmd/metabolism .            # query arXiv for fragile hypotheses
go run ./cmd/metabolism --calibrate . # measure prediction accuracy
```

Current state: 10 hypotheses tracked, 7 confirmed by external sources, 1 irrelevant, 2 pending.

### 3. It audits itself for cognitive biases — using biases it catalogs

The KB contains entities for cognitive biases (confirmation bias, anchoring, availability heuristic, Dunning-Kruger, etc.). It applies those same biases as deterministic auditors checking its own structure.

```bash
go run ./cmd/metabolism --bias .
```

9 auditors. Each maps a bias entity in the KB to a structural check:

| Auditor | What it measures | Current finding |
|---------|-----------------|-----------------|
| Confirmation bias | Corroboration rate among signal cycles | 58% (PASS) |
| Anchoring | Correlation between file age and claim density | rho = -0.02 (PASS) |
| Clustering illusion | Overlap between file grouping and topology clusters | 44% Jaccard (PASS) |
| Availability heuristic | Provenance source concentration (HHI) | **0.70 — 83% Wikipedia (TRIGGERED)** |
| Survivorship bias | Irrelevant-to-challenged resolution ratio | **8:0 — zero challenges found (TRIGGERED)** |
| Framing effect | Evaluative language in entity Briefs | 3% (PASS) |
| Dunning-Kruger | Low-complexity entities escaping vulnerability detection | 74% vs 37% gap (PASS) |
| Base rate neglect | Predicate distribution entropy | 4.3 bits (PASS) |
| Premature closure | Thought-terminating cliches + epistemic DAG leaf detection | 31 structural (PASS) |

The epistemic support DAG (132 directed edges) is inferred from predicate semantics: `Proposes` flows from Person to Hypothesis, `DerivedFrom` flows from source to derived concept, `Disputes` is a contra edge excluded from support.

Two findings are real: the KB is 83% sourced from Wikipedia (availability heuristic), and the metabolism loop has never classified any source as a challenge (survivorship bias). The KB found its own biases.

### 4. It consolidates, speculates, and tightens

Three maintenance modes, modeled on the biological sleep cycle:

- **Dream** (NREM consolidation): analyzes topology + lint + adit without new ingest. Reports bridge entities, file balance, provenance splits, brief quality.
- **Trip** (REM speculation): picks entity pairs from different topology clusters, LLM generates and scores speculative connections. Two axes: temperature (0.0–1.5) × prompt type (analogy, contradiction, genealogy, prediction).
- **Fix**: LLM-assisted Brief tightening with quality gates and automatic revert on failure.

```bash
go run ./cmd/metabolism --dream .                    # consolidation report
go run ./cmd/metabolism --dream --bias .             # + self-audit
go run ./cmd/metabolism --dream --fix --tighten .    # auto-fix with quality gates
go run ./cmd/metabolism --trip --temperature 1.0 .   # speculative connections
```

### 5. It makes predictions and tracks whether they're right

The metabolism loop generates falsifiable predictions: "this hypothesis is structurally fragile → querying external sources should find evidence." The calibration system tracks whether those predictions were confirmed.

```bash
go run ./cmd/metabolism --calibrate --json .
```

Current calibration: 88% hit rate among hypotheses that received signal. Per-hypothesis scorecards with precision, cycles-to-verdict, and signal quality decomposition. The KB reifies its own predictions as first-class claims (`predictions.go`), creating a self-referential feedback loop.

*The same architecture that powers prediction markets — falsifiable claims, resolution criteria, tracked accuracy — except the currency is epistemic accuracy, not money.*

## What's next

The system today is self-contained: it curates the epistemology-of-minds corpus, audits itself, and tracks its own predictions. The roadmap is about opening it up:

- **PKM ingest**: Point it at your Obsidian vault. It converts your notes to typed entities with provenance. The metabolism loop finds gaps, the lint rules catch contradictions, the bias auditors flag systematic blind spots. Your notes work for you.
- **Autonomous operation**: The full sleep cycle (metabolism → dream → trip → bias audit) on a timer. Leave it running. Come back to a morning report of what it found, what it fixed, what predictions resolved.
- **Live-world sensors**: News feeds, economic data, event streams. The predictions become testable against reality, not just against what Wikipedia already documented.

## Quick start

```bash
git clone https://github.com/justinstimatze/winze.git
cd winze
go build ./...              # verify the seed corpus compiles
go run ./cmd/lint .         # see the health dashboard
go run ./cmd/topology .     # structural vulnerability report
go run ./cmd/metabolism --bias .  # cognitive bias self-audit
```

Study a corpus file (e.g., `tunguska.go`) to see the pattern: Provenance, entities, claims.

To start fresh with your own domain:

```bash
./script/reset-corpus.sh    # removes seed corpus, keeps schema
go build ./...              # verify the framework compiles
```

## Schema

### Entity

Named knowledge atom. Fields: ID, Name, Kind, Brief, Aliases.

```go
var AndyClark = Person{&Entity{
    ID:    "andy-clark",
    Name:  "Andy Clark",
    Kind:  "person",
    Brief: "Philosopher and cognitive scientist. Author of the predictive processing framework.",
}}
```

### Roles

16 role types grounded in external vocabulary (Schema.org, WordNet, Wikidata):

**Corpus roles:** Person, Organization, Place, Event, Facility, Substance, Instrument, Hypothesis, Concept

**Design roles:** CreativeWork, DesignLayer, Phase, ProtectedLine, NeverAnswered, AuthorialPolicy, Reading

### Predicates

`BinaryRelation[S, O]` (two-slot) and `UnaryClaim[S]` (one-slot). 30+ predicates across families:

| Family | Predicates |
|--------|-----------|
| Attribution | Proposes, Disputes, ProposesOrg, DisputesOrg |
| Theory | TheoryOf (`//winze:contested`), HypothesisExplains |
| Taxonomy | BelongsTo, DerivedFrom, IsCognitiveBias, IsPolyvalentTerm |
| Authorship | Authored, AuthoredOrg, CommentaryOn, AppearsIn |
| Fiction | IsFictionalWork, IsFictional |
| Spatial | LocatedIn, LocatedNear, OccurredAt |
| People | InfluencedBy, WorksFor, AffiliatedWith, InvestigatedBy |
| Prediction | Predicts, Credence, ResolvedAs (`//winze:functional`) |
| Functional | FormedAt, EnergyEstimate, EnglishTranslationOf (`//winze:functional`) |

### Provenance

Every claim carries an audit trail: where the knowledge came from, when it was ingested, who ingested it, and the specific source fragment supporting the claim.

```go
var source = Provenance{
    Origin:     "Wikipedia 2025-12 / Tunguska_event",
    IngestedAt: "2026-04-11",
    IngestedBy: "winze session 2",
    Quote:      "The explosion over the sparsely populated Eastern Siberian Taiga...",
}
```

### Annotations

- `//winze:contested` on a predicate type means multiple subjects per object is expected (e.g., multiple theories explaining the same concept)
- `//winze:functional` on a predicate type means only one value per subject is correct; violations are flagged by the value-conflict lint rule

## Writing a corpus file

```go
package winze

var mySource = Provenance{
    Origin:     "https://example.com/paper",
    IngestedAt: "2026-04-12",
    IngestedBy: "your-name",
    Quote:      "exact text from source supporting the claim",
}

var Alice = Person{&Entity{
    ID:    "alice",
    Name:  "Alice Researcher",
    Kind:  "person",
    Brief: "Cognitive scientist studying attention.",
}}

var AttentionHypothesis = Hypothesis{&Entity{
    ID:    "attention-bottleneck",
    Name:  "Attention bottleneck hypothesis",
    Kind:  "hypothesis",
    Brief: "Cognitive attention is a serial bottleneck, not a parallel resource.",
}}

var AliceProposes = Proposes{
    Subject: Alice,
    Object:  AttentionHypothesis,
    Prov:    mySource,
}
```

Then: `go build ./...` to verify, `go run ./cmd/lint .` to check health.

## Lint rules

```bash
go run ./cmd/lint .                         # 7 deterministic rules
go run ./cmd/lint . --llm --llm-max-calls=5 # + LLM contradiction check
```

| Rule | What it checks |
|------|---------------|
| naming-oracle | Role types must be grounded in ExternalTerms (Schema.org, WordNet, Wikidata) |
| orphan-report | Entities with zero claim references |
| value-conflict | `//winze:functional` predicates with contradictory values |
| contested-concept | `//winze:contested` predicates with multiple theories (informational, not failure) |
| brief-check | Missing or overlong entity Briefs |
| provenance-split | Same origin cited differently across files |
| llm-contradiction | LLM-detected semantic contradictions across claim neighborhoods (opt-in, needs `ANTHROPIC_API_KEY`) |

## Metabolism

The epistemic metabolism loop: topology identifies fragile hypotheses → sensors query external sources → results are logged → calibration tracks accuracy.

```bash
go run ./cmd/metabolism .                              # arXiv backend (default)
go run ./cmd/metabolism --backend zim --zim FILE .     # Wikipedia ZIM backend
go run ./cmd/metabolism --backend all --zim FILE .     # both backends
go run ./cmd/metabolism --dry-run .                    # show targets without querying
go run ./cmd/metabolism --calibrate .                  # analyze accumulated cycle log
go run ./cmd/metabolism --suggest .                    # generate ingest template from corroborated results
go run ./cmd/metabolism --ingest --zim FILE .          # LLM-assisted ingest from corroborated ZIM cycles
go run ./cmd/metabolism --pipeline --zim FILE .        # full quality pipeline: ingest → build → lint → llm → commit/reject
go run ./cmd/metabolism --reify .                      # generate predictions.go from metabolism log
go run ./cmd/metabolism --dream .                      # consolidation cycle (no new ingest)
go run ./cmd/metabolism --dream --bias .               # + cognitive bias self-audit
go run ./cmd/metabolism --dream --fix --tighten .      # auto-fix overlong Briefs via LLM
go run ./cmd/metabolism --bias .                       # standalone bias audit
go run ./cmd/metabolism --trip .                       # speculative cross-cluster connections
go run ./cmd/metabolism --entity-cap 250 .             # refuse ingest above entity cap (default 250)
go run ./cmd/metabolism --json .                       # JSON output
```

### Wikipedia ZIM setup

The metabolism loop can use a local Wikipedia ZIM file for offline fulltext search — no rate limits, better recall for non-STEM domains.

1. Download a ZIM file from [download.kiwix.org](https://download.kiwix.org/zim/wikipedia/)
2. Run: `go run ./cmd/metabolism --backend zim --zim /path/to/file.zim .`
3. First run builds a Bleve fulltext index (persisted to `<zimfile>.bleve/`)

## Design principles

- **Mirror-source-commitments:** Only encode claims the source explicitly commits to. Use `Provenance.Quote` with exact source text.
- **Schema accretion:** Don't invent predicates speculatively. Wait for the forcing function — a source that explicitly commits to a relationship no existing predicate captures.
- **Prose is I/O not state:** Ingest workers consume prose and produce typed claims. Source documents are transient; the KB is the canonical representation.
- **LLM as expensive lint rule:** LLM judgment is one lint rule among many, not a separate architectural stage. Runs opt-in with an explicit token budget.
- **Reification over schema extension:** Handle competing theories via Hypothesis entities + TheoryOf, not new role types.
- **Depth over breadth:** Prioritize deepening thin contested neighborhoods over expanding to new hypotheses. Entity count is capped at 250 by default.

## Prior art

| Project | Substrate | Consistency | Agent-writable? | Predictions? |
|---------|-----------|-------------|----------------|-------------|
| [Karpathy LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) | Markdown | String-level lint | Yes | No |
| [Monarch dismech](https://github.com/monarch-initiative/dismech) | YAML + LinkML | Schema validation + CI | Yes | No |
| [Open Ontologies](https://github.com/fabio-rovai/open-ontologies) | RDF/OWL | OWL2-DL reasoning | Via MCP | No |
| Prolog/Datalog | Logic programs | Inference engine | Limited | No |
| [Lean Mathlib](https://github.com/leanprover-community/mathlib4) | Dependent types | Proof checker | Limited | No |
| **winze** | Go source | `go build` + 7 lint rules + 9 bias auditors | Yes | **Yes (88% hit rate)** |

## License

MIT
