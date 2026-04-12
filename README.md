# winze

Go's type system as a knowledge-base consistency checker.

## The idea

Source code IS knowledge. Entities are typed constants, predicates are generic types, claims are variable declarations. `go build` is the consistency checker — it enforces referential integrity and type safety across every claim in the graph. No binary is produced.

Standard tooling works unchanged: LSP navigation, `go/ast` analysis, CI pipelines, code review, `git diff`, `git blame`. Every improvement to agentic coding tools (Claude Code, Cursor, Devin, etc.) directly improves the fleet's ability to maintain the knowledge base, because the knowledge base *is* code.

The 2026 agentic coding wave is building infrastructure for LLM agents to collaboratively write and maintain code. Winze repurposes that same infrastructure for collaborative knowledge maintenance.

## Quick start

```bash
git clone https://github.com/justinstimatze/winze.git
cd winze
go build ./...              # verify the seed corpus compiles
go run ./cmd/lint .         # see the health dashboard
```

Study a corpus file (e.g., `tunguska.go`) to see the pattern: Provenance, entities, claims.

To start fresh with your own domain:

```bash
./script/reset-corpus.sh    # removes seed corpus, creates starter.go
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

var AliceProposes = Proposes[Person, Hypothesis]{
    Subject: Alice,
    Object:  AttentionHypothesis,
    Prov:    mySource,
}
```

Then: `go build ./...` to verify, `go run ./cmd/lint .` to check health.

## Lint rules

```bash
go run ./cmd/lint .                         # 4 deterministic rules
go run ./cmd/lint . --llm --llm-max-calls=5 # + LLM contradiction check
```

| Rule | What it checks |
|------|---------------|
| naming-oracle | Role types must be grounded in ExternalTerms (Schema.org, WordNet, Wikidata) |
| orphan-report | Entities with zero claim references |
| value-conflict | `//winze:functional` predicates with contradictory values |
| contested-concept | `//winze:contested` predicates with multiple theories (informational, not failure) |
| llm-contradiction | LLM-detected semantic contradictions across claim neighborhoods (opt-in, needs `ANTHROPIC_API_KEY`) |

## Sensor (optional)

Poll arXiv and Semantic Scholar for new papers matching your KB's domain:

```bash
go run ./cmd/sensor --backend arxiv --limit 5 --json
go run ./cmd/sensor --backend scholar --limit 5 --json  # needs SEMANTIC_SCHOLAR_API_KEY
go run ./cmd/sensor --backend all --limit 5 --json
```

Edit the `queries` slice in `cmd/sensor/main.go` to match your domain. State is tracked in `.sensor-state.json` (gitignored) to avoid re-reporting seen papers.

### Wikipedia ZIM setup (recommended)

The metabolism loop can use a local Wikipedia ZIM file for offline fulltext search — no rate limits, better recall for non-STEM domains than arXiv.

1. Download a ZIM file from [download.kiwix.org](https://download.kiwix.org/zim/wikipedia/):
   ```bash
   # ~20GB, nopic variant (text only, recommended)
   wget https://download.kiwix.org/zim/wikipedia/wikipedia_en_all_nopic_2025-12.zim
   mkdir -p /opt/zim && mv wikipedia_en_all_nopic_2025-12.zim /opt/zim/
   ```

2. Install Python libzim (used by `script/zim-search.py`):
   ```bash
   pip install libzim    # or: uvx install openzim-mcp /opt/zim
   ```

3. Run a metabolism cycle with the ZIM backend:
   ```bash
   go run ./cmd/metabolism --backend zim --zim /opt/zim/wikipedia_en_all_nopic_2025-12.zim .
   ```

   If libzim is in a virtualenv, set `WINZE_ZIM_PYTHON` to that interpreter:
   ```bash
   export WINZE_ZIM_PYTHON=/path/to/venv/bin/python
   go run ./cmd/metabolism --backend zim --zim /opt/zim/wikipedia_en_all_nopic_2025-12.zim .
   ```

4. Run both backends together:
   ```bash
   go run ./cmd/metabolism --backend all --zim /opt/zim/wikipedia_en_all_nopic_2025-12.zim .
   ```

## Gas Town integration (optional)

Multi-agent curation workflow for autonomous KB maintenance via [Gas Town](https://github.com/gastownhall/gastown):

- **Formula:** `.beads/formulas/mol-curate.formula.toml` — 5-step curation workflow (load-context, source-analysis, ingest, validate, submit)
- **Patrol plugin:** `plugins/kb-health/plugin.md` — periodic lint health check (2h cooldown)
- **Interactive skill:** `.claude/skills/curate/SKILL.md` — `/curate status|ingest|audit|predict|resolve|sensor`

## Design principles

- **Mirror-source-commitments:** Only encode claims the source explicitly commits to. Use `Provenance.Quote` with exact source text.
- **Schema accretion:** Don't invent predicates speculatively. Wait for the forcing function — a source that explicitly commits to a relationship no existing predicate captures.
- **Prose is I/O not state:** Ingest workers consume prose and produce typed claims. Source documents are transient; the KB is the canonical representation.
- **LLM as expensive lint rule:** LLM judgment is one lint rule among many, not a separate architectural stage. Runs opt-in with an explicit token budget.
- **Reification over schema extension:** Handle competing theories via Hypothesis entities + TheoryOf, not new role types.

## Roadmap

- **PKM / second-brain ingest:** Import from Obsidian vaults, Logseq graphs, Roam JSON exports, Notion exports, plain markdown Zettelkasten, and plain text directories. An `ingest` command that reads a directory of markdown notes, extracts entities and relationships via LLM, and emits `.go` corpus files with provenance pointing back to the source note. Goal: dump your existing knowledge base into winze in one pass, then let the type system and lint rules surface what's inconsistent.
- **Query generation improvements:** Topology-derived sensor queries need domain-aware phrasing (current: literal keyword concatenation produces museum results when searching for anthropological theses).
- **Pure Go ZIM reader ([gozim](https://github.com/justinstimatze/gozim)):** Eliminate the Python bridge for Wikipedia search. Bleve-backed fulltext index, zero CGO. In development.
- **Prediction schema:** `Predicts[Hypothesis, Event]`, `Credence[Hypothesis]` — encode falsifiable predictions and track calibration over time.
- **Live visualization:** Gource-style real-time dashboard of agents committing to the KB.

## Prior art

| Project | Substrate | Consistency mechanism | Agent-writable? |
|---------|-----------|----------------------|----------------|
| [Karpathy LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) | Markdown | String-level lint | Yes |
| [Monarch dismech](https://github.com/monarch-initiative/dismech) | YAML + LinkML | Schema validation + CI | Yes |
| [Open Ontologies](https://github.com/fabio-rovai/open-ontologies) | RDF/OWL via Oxigraph | OWL2-DL reasoning | Via MCP |
| Prolog/Datalog | Logic programs | Inference engine | Limited |
| [Lean Mathlib](https://github.com/leanprover-community/mathlib4) | Dependent types | Proof checker | Limited |
| **winze** | Go source code | `go build` + lint rules | Yes (mainstream language) |

## License

MIT
