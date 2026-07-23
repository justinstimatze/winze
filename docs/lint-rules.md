# Lint rules

`go run ./cmd/lint .` runs the deterministic rules; add `--llm --llm-max-calls=5`
for the LLM contradiction check.

The rules: naming-oracle, orphan-report, value-conflict, contested-concept,
brief-check, provenance-split, llm-contradiction, brief-drift, structural-dedup,
lexicon-fence.

## structural-dedup

`structural-dedup` flags probable duplicate entities by SHARED
claim-neighborhood — same role, same predicates to/from the same neighbors —
not by prose (two entities that are the same concept may have Briefs written
nothing alike). This is the calque-faithful check: index the edges, not the
representation, catching the duplicate-entity defect the build gate is blind to
(two same-type entities both type-check). Rarity-weighted (idf) so taxonomic
siblings fall out, and symmetric (both neighborhoods must be near-identical) so
it flags twins, not category-mates. Advisory and deliberately high-threshold:
on a clean corpus the honest answer is few or none, and dense sibling clusters
(a fiction cast, a bias taxonomy) can still appear — a real duplicate ranks
above them. Point it at one entity with `query --dupes NAME` (the coin-time
query: "does something structurally identical already exist?"). See
`internal/dedup`.

## brief-drift

`brief-drift` reports entities whose Brief names another entity with no claim
path to it within two hops. Each hit is an **assertion-candidate**: prose that
may claim a relationship the claim graph does not encode. Two ways to resolve
one: **add the claim** (if the Brief asserts a real relationship — the prose
was ahead of the structure), or **annotate `//winze:mentions Target`** (if the
Brief names it for context only — mirror-source-commitments permits Brief
references a source does not commit to). Marked mentions are exempted and
counted separately.

Advisory by default (a bare Brief mention is often legitimate, so hard-failing
all of them would be the over-strict trap). `lint --brief-strict` turns it into
a gate (exit 1) on any unexempted assertion-candidate — for a triaged corpus
where every Brief mention is either claimed or explicitly acknowledged. Two
hops rather than one because the house pattern routes a person to a concept
through an intermediate framing entity.

## lexicon-fence

`lexicon-fence` keeps lexicon's private, non-redistributable content out of the
public corpus. A `Provenance` whose Origin or Quote references a lexicon locator
(`lex-NNNN` or `lexicon:`) is a hard failure (exit 1): lexicon is a *stimulus*
winze reads to spark connections, never a *source* it quotes. The correct
attribution for anything lexicon-derived is a `Conjecture` (`From:
"lexicon:lex-NNNN"`), which carries no `Quote` by design, so nothing leaks. The
compiler closes the `Conjecture` side; this rule closes the free-string
`Provenance` side it can't reach. Matched precisely enough that the ordinary
word "lexicon" in prose doesn't trip it. See `docs/lexicon.md`.
