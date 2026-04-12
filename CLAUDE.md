## Winze Agent Instructions

This project is a **non-executable Go knowledge base**. `go build` is the
consistency checker, not a build system. No binary is produced. Code
editing and knowledge manipulation are the same operation.

### What you're working on

Every `.go` file in the root is a knowledge corpus slice. Each declares:
- **Entities** (typed: Person, Concept, Hypothesis, Place, Event, etc.)
- **Claims** (typed predicates: Proposes, TheoryOf, BelongsTo, InfluencedBy, etc.)
- **Provenance** (Origin, Quote, IngestedAt, IngestedBy)

The type system is in `schema.go`, roles in `roles.go`, predicates in `predicates.go`.

### Quality gates

```bash
go build ./...                              # type-checks references
go vet ./...                                # static analysis
go run ./cmd/lint .                         # 4 deterministic lint rules
go run ./cmd/lint . --llm --llm-max-calls=5 # + LLM contradiction check
```

Lint rules: naming-oracle, orphan-report, value-conflict, contested-concept, llm-contradiction.

### Mirror-source-commitments

Only encode claims the source explicitly commits to. Use `Provenance.Quote`
with exact source text. Do not fabricate relationships. Brief-level references
are fine for connections the source doesn't explicitly make.

### Schema accretion

Do NOT invent predicates speculatively. Wait for the forcing function: a
source that explicitly commits to a relationship no existing predicate
captures. When a third occurrence of a pattern surfaces, promote it to a
named discipline.

### Existing predicate families

**Attribution:** Proposes, Disputes, ProposesOrg, DisputesOrg
**Theory:** TheoryOf (//winze:contested), HypothesisExplains
**Taxonomy:** BelongsTo, DerivedFrom, IsCognitiveBias, IsPolyvalentTerm
**Authorship:** Authored, AuthoredOrg, CommentaryOn, AppearsIn
**Fiction:** IsFictionalWork, IsFictional
**Spatial:** LocatedIn, LocatedNear, OccurredAt
**People:** InfluencedBy, WorksFor, AffiliatedWith, InvestigatedBy
**Functional (//winze:functional):** FormedAt, EnergyEstimate, EnglishTranslationOf

### Curation formula

If you're a Gas Town polecat, your workflow is defined by the `mol-curate`
formula in `.beads/formulas/`. Run `gt prime` to see your checklist.
