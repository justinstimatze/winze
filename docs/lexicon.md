# Lexicon as a reference pool (stimulus, not source)

Lexicon is a private substrate of ~459 cognitive-pattern atoms, each with
verbatim lineage from primary sources acquired by various means. It is not
public and its refs are not redistributable. It is also a rich pool to trip
against — the patterns are squarely in winze's domain (the epistemology of
minds). The problem: how does winze draw on it without leaking it into a corpus
that pushes to GitHub?

The answer is winze's existing shape. Winze is a typed graph of lightweight
pointers, not a document store — an entity is a Name plus a one-line Brief, and
`Provenance.Origin` is a *locator*, not the source text. The heavy content
already lives outside; winze holds the edge and knows how to find it. Lexicon
just makes that pointer a private-read one. (This is the anti-Confluence:
Confluence embeds everything and searches badly; winze holds edges and traverses
to the source.)

## The rules

1. **Reference, never copy.** Winze holds a lexicon atom's *ID* (`lex-0165`) as a
   locator — like a DOI — never its gloss and never the acquired primary-source
   text. The ID is how winze finds it via the lexicon MCP when needed.

2. **Stimulus, not source.** Winze reads lexicon (via `lexicon_read`, at
   trip time, ephemerally) to spark connections among its *own* entities. The
   output is a `Conjecture`, and the compiler forbids a `Conjecture` from
   carrying a `Quote`. So a lexicon-sparked claim *structurally cannot* contain
   lexicon's text. The locator lives in the Conjecture's `From`
   (`Conjecture{From: "lexicon:lex-0165", Rationale: ...}`).

3. **Never the raw ref dir.** The "acquired by various means" material is the
   real risk and never touches a repo that pushes to GitHub. Winze reaches
   lexicon only through the MCP's gloss/ID layer.

4. **No lexicon atoms as winze entities.** A pattern stays a *reference*, not an
   ingested node. If one genuinely deserves a winze entity, re-derive it from a
   *public* primary source with its own real provenance — never copy lexicon's.

5. **Don't recreate lexicon.** Winze should not become a shadow copy. Where an
   existing winze entity really is a lexicon pattern, it can be *thinned* to a
   node plus a locator, trusting lexicon to hold the content — winze keeps the
   claims (its structural value), lexicon keeps the prose.

## The fence (`lexicon-fence` lint rule)

The compiler closes most of this — `Conjecture` has no `Quote`. The one hole it
can't reach: `Provenance.Origin`/`.Quote` are free strings, so nothing stops a
careless paste of a lexicon gloss into a `Provenance`. The `lexicon-fence` lint
rule is that guard: a `Provenance` whose Origin or Quote references a lexicon
locator (`lex-NNNN` or `lexicon:`) is a hard failure, because the correct
attribution for anything lexicon-derived is a `Conjecture`, which cannot quote.
Deterministic, and matched precisely enough that the ordinary word "lexicon" in
prose does not trip it.

So a careless future ingest that tries to paste lexicon text into the public
corpus fails the gate instead of leaking — safe by construction, not by care.

## North star: lexicon as its own winze, melded while tripping

The clean end state: lexicon becomes its *own* winze store. Then tripping winze
against lexicon is just `winze-meld` — a read-only, pinned-SHA, dissolvable union
of the two stores — with `trip` run across the melded graph, then dissolved. The
meld is read-only by construction (two `package winze` stores can't compile
merged, so it is materialized frozen and namespace-prefixed), so lexicon's
content is never written into winze's committed corpus. No new integration
mechanism: `meld` and `trip` already compose, and the `lexicon-fence` still
guards any connection that gets promoted back. Until then, the MCP-stimulus path
above is the same discipline with a lighter join.
