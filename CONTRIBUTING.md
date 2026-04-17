# Contributing to winze

## Before you start

winze has two layers. The `.go` files in the root are the knowledge base — entities, claims, provenance. The `cmd/` directory is the tooling that analyzes, audits, and maintains that knowledge. Both are real Go code, but `go build` on the root package produces no binary — it's a consistency check.

Read `tunguska.go` or `nondualism.go` to see the pattern: Provenance, entities, claims. The walkthrough in the README shows the full cycle in 5 minutes.

## Quality gates

Every PR must pass:

```bash
go build ./...              # type-checks all entity references
go vet ./...                # standard static analysis
go test ./...               # corpus invariant tests
go run ./cmd/lint .         # 7 deterministic lint rules
```

CI runs these automatically. If the compiler rejects your claim, the claim is wrong.

## Domain boundary

The KB's domain is the **epistemology of minds** — how minds (human and artificial) build, validate, and fail at modeling reality. A concept is in-domain when it illuminates how knowledge is constructed, contested, or mistaken.

**In-domain examples:** Tunguska event (competing hypotheses about physical evidence), anchoring bias (systematic reasoning failure), predictive coding (how brains model reality).

**Out-of-domain examples:** A recipe database (no epistemic dimension), a pure programming language comparison (unless it's about how language shapes thinking), sports statistics (unless examining how fans construct false narratives from data).

Contributions outside this domain will be declined. If you're unsure, open an issue first.

## Adding knowledge

### New entities and claims

1. Pick or create a `.go` file for the topic
2. Declare a `Provenance` with the exact source (URL, DOI, book citation)
3. Add entities with the appropriate role type (`Person`, `Concept`, `Hypothesis`, `Place`, `Event`, etc.)
4. Add claims using existing predicates (`Proposes`, `TheoryOf`, `BelongsTo`, `InfluencedBy`, etc.)
5. Run the quality gates

Rules:

- **Mirror-source-commitments.** Only encode claims the source explicitly states. `Provenance.Quote` must contain the exact text supporting the claim. No inference, no extrapolation.
- **Entity cap.** The KB targets depth over breadth. The default cap is 300 entities. Deepen thin contested neighborhoods before expanding.
- **Brief quality.** Entity Briefs should be one sentence, under 300 characters. State what the entity *is*, not what it *does in this KB*.

### New predicates

Do not propose new predicates speculatively. The threshold is a **forcing function**: three occurrences of the same relationship pattern from different sources, where no existing predicate captures it.

Open an issue with:
- The three examples
- Which existing predicates you considered and why they don't fit
- The proposed type signature (`BinaryRelation[Subject, Object]` or `UnaryClaim[Subject]`)

### Disputes

If a source contradicts an existing claim, encode the dispute:

```go
var SmithDisputesJonesHypothesis = Disputes{
    Subject: Smith,
    Object:  JonesHypothesis,
    Prov:    smithSource,
}
```

Disputes are first-class. The topology analyzer tracks them. The survivorship bias auditor flags their absence.

## Reporting bugs

For the tooling (`cmd/lint`, `cmd/topology`, `cmd/metabolism`):

1. What command you ran
2. What you expected
3. What happened (paste the output)
4. Your Go version (`go version`)

For the KB itself (incorrect claims, missing provenance, type errors):

1. Which entity or claim
2. What's wrong
3. A source that demonstrates the error

## Code style

The tooling in `cmd/` is standard Go. The corpus files in the root follow the patterns established by existing files. When in doubt, match the existing style.
