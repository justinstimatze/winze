# Winze: a commonplace book that typechecks

> A 250-entity Go knowledge base, structured the way Renaissance scholars
> kept personal notebooks. Sourced quotations, indexed headings, the
> keeper's own hunches marked as such. The trick is that this old form
> turns out to be a good substrate for LLM consumption — sourced quotes
> + indexed headings + marked-as-speculative residue lets a small model
> synthesize without confabulating.

## What's a commonplace book

The term is a calque of Latin *locus communis* — "common place," meaning
a topic of general application. Aristotle's *Topics* is the ancestor.
Erasmus's *De Copia* (1512) taught the practice; Locke's *A New Method
of Making Common-Place-Books* (1706) is the canonical how-to. Marcus
Aurelius, Milton, Bacon, Newton, Jefferson, Twain, and Auden all kept
one. They were as common as having a notebook is now.

A commonplace book is **thematic**, not chronological. Entries are
filed under topical headings, with sourced quotations and the keeper's
own observations. Entries say "Locke says X" without claiming X is true.
The keeper's connections between entries are marked *as the keeper's*,
not promoted to canonical status.

Modern descendants — Luhmann's Zettelkasten, Obsidian, Roam, the
"second brain" / PKM movement — all sit in this family. Winze sits
there too, with two additions: typecheck-as-consistency-gate (Go's
compiler verifies references), and explicit speculative residue
(speculative cross-cluster connections kept as JSONL, separate from
typed claims).

## Convention-by-convention fit

| Commonplace book convention | Winze instantiation |
|---|---|
| Thematic, not chronological | Entity/concept files (`hard_problem.go`, `apophenia.go`), not log entries |
| Every entry sourced with quotation | `Provenance.Quote` is mandatory on every claim |
| Topical index keyed by heading | Go's typed entity + predicate system |
| Personal, not authoritative | Mirror-source-commitments — only what the source says |
| Keeper's connections marked as such | NONE-predicate residue in `.metabolism-trip-isolated.jsonl` |
| Working tool, not encyclopedia | A working KB, not a reference work |

## The demo: with vs without speculative residue

Winze runs a "trip cycle" — REM-like speculative cross-cluster
connection generation. Pairs of entities from different topology
clusters are scored by an LLM for whether they share an interesting
structural pattern. Connections that score well but don't fit any
canonical predicate (`Proposes`, `BelongsTo`, `InfluencedBy`, etc.)
accrete into `.metabolism-trip-isolated.jsonl` — the keeper's
hunches, marked as such.

The `--ask` query path reads that JSONL and surfaces it in a
"Speculative Cross-Cluster Connections" section of the LLM context.
The model is told these are speculative, not canonical.

We ran a 10-question battery (5 should-help, 3 negative-control,
2 mixed) twice — once with the JSONL present, once with it stashed
away — through Haiku 4.5. Two trials per condition. Full report:
[`jsonl-validation-2026-04-28.md`](./jsonl-validation-2026-04-28.md).

### What the residue surfaced

For "What does Gödel's incompleteness theorem have in common with
cognitive bias detection?":

- **WITH residue**: cited specific scored speculative pairs —
  *"Both Gödel's incompleteness and Sagan's baloney detection reveal
  fundamental boundaries where domain-specific mastery cannot be fully
  compressed into teachable mechanistic procedures that guarantee
  correctness at every instance" (score 3/5)*. Names the pair, cites
  the score, hedges as speculative.
- **WITHOUT residue**: fell back to typed-corpus inference —
  named `FiniteOntologyIncompleteness` from `theory_seeds.go`, which
  is a real claim but a less direct answer.

For "How does Searle relate to Kahneman?":

- **WITH residue**: surfaced a score-4/5 speculative analogy —
  *"Both Searle's Biological Naturalism and Kahneman's Dual-Process
  Framework constrain valid models of cognition to mechanisms that
  reflect specific architectural features rather than functional
  equivalence alone."* Hedged as machine-generated, not promoted.
- **WITHOUT residue**: explicitly answered *"there is no direct
  documented relationship"*, which is the right answer if you
  haven't seen the speculative residue.

Both behaviors are honest. The residue gives the model a *third
path* between "answer from typed claims" and "honest non-answer" —
a path that's marked as speculative on its way out the door.

### Aggregate result

5 should-help questions: 4 wins for WITH on synthesis specificity,
1 tie. 3 negative-control questions (typed-only content): 3 ties —
the residue doesn't pollute non-speculative answers. 2 mixed
questions: WITH adds orthogonal info on both.

The model never confabulated. When the residue had relevant content,
it cited it with scores and hedged appropriately. When it didn't,
the model fell back honestly.

## Why this is worth showing

A 250-entity knowledge base is genuinely commonplace-book-scale. It
is not encyclopedia-scale, and it is certainly not a god-mind. But
the *form* — sourced quotation, indexed headings, marked speculative
residue — turns out to map almost exactly onto what an LLM needs to
synthesize without confabulating: trustable substrate plus separately-
flagged hunches.

Vector-DB filing cabinets do not have this structure. Everything
retrieved is presented as equivalent fact. Without the speculative-
vs-canonical separation, an LLM can't distinguish the keeper's
hunches from the keeper's sources, so either it hedges everything
into uselessness or it confabulates on synthesis.

The commonplace book form is centuries older than databases, and it
solved this exact problem for human keepers using nothing but a
heading system and an attribution rule. The Go compiler enforces the
heading system. The trip cycle generates the keeper's hunches and
files them under their own roof.

## Caveats

- **Sample is small.** 53 speculative connections from one polecat
  run, 10 questions in the battery, 2 trials per condition. The
  result is suggestive, not proof.
- **Haiku is non-deterministic.** Two trials per condition smooths
  this for qualitative scoring but is not power analysis.
- **The residue has hub-bias.** Gödel, Kahneman, and Hilbert dominate
  current speculative pairs (rich conceptual handles attract
  connections). This isn't fixed yet; structural-affinity weighting
  was added but conceptual-utility hubs are a separate effect.
- **The win is conditional on JSONL coverage.** Questions about
  domains the residue doesn't span degrade to "tie with typed-only"
  outcomes.

## Reproducing

```bash
# With residue present
ls .metabolism-trip-isolated.jsonl
go run ./cmd/query --ask "What does Gödel's incompleteness theorem have in common with cognitive bias detection?" .

# Without residue
mv .metabolism-trip-isolated.jsonl /tmp/stash.jsonl
go run ./cmd/query --ask "What does Gödel's incompleteness theorem have in common with cognitive bias detection?" .
mv /tmp/stash.jsonl .metabolism-trip-isolated.jsonl

# Full battery
TRIALS=2 ./scripts/validate_jsonl_help.sh
```

The full validation report scores 10 questions across 3 classes.
