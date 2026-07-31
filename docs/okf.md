# OKF export (`winze-okf`)

`winze-okf` projects the corpus into an [Open Knowledge Format][spec] bundle —
Google Cloud's OKF v0.2, a directory of markdown files with YAML frontmatter,
published June 2026.

[spec]: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md

```bash
winze-okf --out ./bundle .        # emit a bundle from the corpus
winze-okf --validate ./bundle     # check conformance (exit 1 if not conformant)
winze-okf --out ./bundle --force  # overwrite a directory winze did not generate
```

## Why export

OKF and winze made opposite bets from the same premise. Both hold that
agent-facing knowledge needs curation, provenance, and a generated-vs-sourced
distinction — OKF v0.2's `generated` / `verified` trust tiers arrived at that
independently of winze's `Attribution` sum. They part on enforcement. OKF's
conformance floor is one required field (`type`), and the spec forbids consumers
from rejecting unknown types, unknown keys, or broken links; it is buying
adoption. Winze's floor is `go build`; it is buying integrity.

Those bets compose in one direction. A corpus that satisfies the compiler can
always be projected down to a bundle that satisfies the floor; the reverse does
not hold. So winze exports, and the export is the argument in a form other tools
can read — the bundle renders on GitHub, indexes in any search tool, and loads
into any OKF consumer, while the corpus keeps the gate.

The strategic reading: if OKF becomes the interchange format for agent
knowledge, a Go-only KB is an island. The exporter turns that risk into a
distribution channel without conceding anything to it.

## What the projection preserves

**Edge types.** OKF concepts link with ordinary markdown links, and the spec is
explicit that the *kind* of a relationship is carried by surrounding prose, not
the link. That is the one thing winze cannot afford to drop, so claims are
emitted under `## <Predicate>` headings inside `# Claims`. Unknown headings are
conformant, so the bundle stays legal for any consumer while a consumer that
looks recovers the full typed edge. The object's document gets the inverse edge
under `# Referenced by`.

**The attribution split, structurally.** This is the part that matters:

| winze | OKF output |
|---|---|
| claim backed by `Provenance` | contributes a `sources:` entry; the claim link carries that source's footnote id; the exact `Quote` lands in the footnote definition |
| claim backed by `Conjecture` | contributes **nothing** to `sources:`; emitted under `# Conjectures` with winze's own rationale, generator, cycle and score; carries no footnote marker |

`corpusparse.Conjecture` has no `Quote` field for the same reason `schema.go`'s
does not. There is no value the exporter could put in a source entry for a
generated claim even by mistake — the trip-fabrication failure mode stays closed
across the projection, not by a rule in the exporter but by an absent field.
`TestConjectureNeverReachesSources` pins it.

**Trust tiers, derived rather than asserted.** OKF's three tiers fall out of the
split instead of being declared alongside it:

- `generated: {by, at}` records who produced the document. Always a winze
  process today; `corpusparse.Actor` maps a winze identity into OKF's
  `process:` / `human:` convention, defaulting to `process:` so an export
  cannot quietly promote machine ingest to human review.
- `verified: [{by: process:winze-build-gate, at}]` is emitted only for
  documents with at least one Quote-bearing sourced claim → **machine-confirmed**.
  The actor is the gate, not the ingester: what was verified is that a
  Quote-bearing provenance type-checked and passed lint.
- A document backed only by conjecture gets no `verified` entry and
  `status: draft` → **unverified**. Which is what it is.

**Quotes over URLs.** OKF's `sources[].resource` is a URI, and the spec tells
consumers to tolerate link rot. Winze's `Origin` is a free-form hint that was
never required to resolve — the `Quote` is the audit record — so `resource`
takes a URL only when the Origin actually contains one, and otherwise becomes a
`winze:source/<id>` scope descriptor, which the spec permits alongside URLs and
bundle paths. Synthesizing a plausible URL here would be the same fabrication
the type system exists to prevent. The Quote travels in the footnote regardless,
so the bundle's audit trail survives its sources going offline.

**Typed code citations** (`SourceDoc.Refs`) are emitted under
`# Code references`, labelled with the fact that the compile-time guarantee does
*not* travel: in the corpus a renamed symbol breaks the build, in a markdown
bundle it is prose. Exporting a guarantee you cannot keep is how a bundle starts
lying.

## What does not survive

Honest accounting of the loss:

- **Per-claim trust is flattened at the frontmatter level.** OKF's tiers are
  document-scoped, so `status` / `verified` describe the whole file. The
  per-claim distinction survives only in the body, via footnote markers and the
  `# Conjectures` split. A consumer reading frontmatter alone sees less than the
  corpus knows.
- **Disagreement has no OKF vocabulary.** `Disputes` claims and
  `//winze:contested` concepts export as ordinary predicate headings, which is
  lossless in the body and invisible to a frontmatter-only consumer.
- **The gate itself.** A bundle is markdown. Nothing downstream stops an editor
  from writing a `sources:` entry under a conjecture by hand. Round-tripping
  a bundle back into the corpus therefore belongs on the untrusted path
  (`docs/skeptical-ingest.md`), not the trusted one.

## Bundle layout

```
bundle/
├── index.md              # okf_version: "0.2", links every section
├── log.md                # date-grouped history, newest first
├── concepts/
│   ├── index.md          # progressive disclosure: title - description
│   └── survivorship-bias.md
├── hypotheses/ people/ events/ places/ organizations/ …
└── .winze-okf.json       # generation marker (see below)
```

Directories group by role for progressive disclosure only — OKF puts no meaning
in the path, and the role also survives verbatim in each document's `type`.

Documents carry two extra frontmatter keys, `winze_var` and `winze_file`, which
join a concept back to the Go declaration it came from. Unknown keys are
conformant and consumers must tolerate them.

## Determinism

Emission is fully deterministic: entities, sections, sources and footnotes are
sorted, slug collisions resolve in sorted var order, and nothing carries a
wall-clock stamp. Re-exporting an unchanged corpus produces a byte-identical
bundle, so a committed bundle diffs against the corpus change that caused it and
nothing else.

`.winze-okf.json` marks a directory as winze-generated. A re-export overwrites a
marked directory freely and refuses an unmarked non-empty one unless `--force` —
winze may replace what winze wrote, and nothing else.

## Validation

`--validate` checks the spec's conformance floor as **errors**:

1. every non-reserved `.md` file has parseable YAML frontmatter
2. every frontmatter block has a non-empty `type`
3. reserved filenames follow their structure — `log.md` uses ISO 8601 date
   headings, and only the bundle-root `index.md` may declare `okf_version`

It reports as **warnings** the things the spec forbids *consumers* from
rejecting: broken cross-links, missing indexes, out-of-order log entries. That
is a rule about reading, not writing. A producer that emits a broken link has a
bug, and "the consumer must tolerate it" is not a reason to ship it. The
exporter's own output is asserted to validate with zero warnings.

## Upstream gap

The exporter's `## <Predicate>` convention is a workaround for a real hole in
the spec: OKF has a knowledge graph with no edge types. Winze's predicate
vocabulary is precisely the missing piece, and a backward-compatible `relations:`
frontmatter family would close it. v0.1 → v0.2 absorbed the whole
provenance/trust surface in three months under PR/issue governance, so the spec
is visibly still forming — this is the highest-leverage contribution available.
