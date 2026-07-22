# Typed citation: the primitive under winze

Winze has been described as a typed epistemic knowledge base, a "commonplace
book that typechecks," a codebase-documentation layer. Those are all facets of
one primitive: the **typed citation** — a reference whose target existence is
checked by the compiler, so it cannot go stale silently.

Three references in the KB are the same shape:

- **concept → concept** — a claim's `Subject`/`Object` name other entities. Put
  the wrong entity type in a slot and it does not compile.
- **claim → source** — every claim carries `Provenance{Quote, Origin}`; a claim
  without a source is not a valid record.
- **doc → code** — a documentation entity cites a live code symbol *by value*
  (`internal/…` symbol held in a field), so renaming or deleting the symbol
  breaks the build.

Each is a citation with a type the compiler keeps honest. The knowledge base is
what accumulates when enough typed citations pile up; retrieval, topology, and
the metabolism loop are what you can do once they have.

## Doc → code, demonstrated

`winze_self.go` documents winze's own internals and cites real symbols:

```go
CorpusLockDoc = SourceDoc{
    Entity: &Entity{Name: "Corpus write lock", Kind: "sourcedoc",
        Brief: "Every winze write path takes a corpus-wide advisory flock…"},
    Refs: []CodeRef{{
        Symbol: corpuslock.Acquire,          // the real symbol, held as a value
        Path:   "internal/corpuslock.Acquire",
    }},
}
```

`Symbol` holds the actual function, so `go build .` type-checks it. The full
loop:

| step | action | result |
|------|--------|--------|
| valid | cite `corpuslock.Acquire` | `go build .` passes |
| drift | rename the symbol, leave the doc | `undefined: corpuslock.Acquire` at the citation |
| heal | update the citation to match | build passes |

Standard Go rename tooling (`gopls`) propagates a symbol rename into the
citation automatically, so the healthy case is a non-event; only a citation
whose target genuinely no longer exists fails. This is the same enforcement the
KB already applies to concept links, pointed at the codebase.

Contrast a prose wiki, a README, or a design doc: a reference to `handleAuth()`
survives long after `handleAuth` is gone. Here it cannot.

## Enforcement, not detection

The knowledge-management field around this problem splits on one axis. One side
*detects* drift after the fact — an LLM watches the docs (and, in the better
tools, the connected codebase) and drafts a fix routed through human approval;
nothing auto-applies, but nothing is prevented either, and a stale reference is
live until the model happens to flag it. (Slite's self-maintaining knowledge
base documents exactly this posture in its own material —
<https://slite.com/blog/slite-announcing-self-maintaining-knowledge-base> — as
does the "knowledge as code" plain-text-plus-validator pattern at
<https://knowledge-as-code.com/>, whose cross-reference checker resolves links
by name at review time rather than at build time.)

Winze sits at the other end: a stale reference does not exist, because it does
not compile. Prevention rather than probabilistic detection; deterministic
rather than semantic. The detection tools bolt a watcher onto the outside of a
prose store and point it at the code; winze moves the boundary so the reference
*is* code and there is nothing left to watch.

Detection and enforcement are complementary — an LLM re-reading whether a doc's
*narrative* still matches the code it cites is real work a compiler cannot do
(the compiler checks that every symbol a doc names exists, not that the story is
still accurate). Winze's own division of labor is: compiler for
reference-integrity, metabolism/LLM loop for narrative accuracy.

## Typing serves retrieval too

A verified type is not only an integrity check; it is a retrieval signal with
zero classification error. Because an entity's role and a claim's predicate
*compiled*, retrieval can filter or weight by them without the misclassification
noise that soft, model-assigned tags carry.

The read side has content retrieval (BM25 fulltext, local-embedding semantic
search, reciprocal-rank hybrid) and the type signal fused on top:

- `query --hybrid "consciousness" --type Hypothesis` filters the fused ranking
  to a verified role. The role was type-checked at build time, so nothing
  misclassified leaks the wrong kind in or a right one out — a lower-ranked
  hypothesis surfaces past a higher-ranked concept exactly when you asked for
  hypotheses.
- `query --hybrid "apophenia" --expand` appends each hit's typed claim
  neighborhood — `predicate → neighbor (neighbor's verified role)`, with edge
  direction read off the graph. Downstream reasoning gets the relationships,
  not just a prose snippet, which is the concrete answer to "terminology
  mismatch ruins downstream reasoning": you hand the model the typed subgraph.

The same type declaration pays twice — once for integrity, once for retrieval —
because both read the one source of truth the compiler already verified.

## The format is an implementation detail

The load-bearing consequence: **a user never needs to know it is code.** The
compiler is invisible infrastructure. A person adds or queries knowledge by
talking to an agent; the agent maps the utterance to a typed claim, the build
gate validates it, and the agent confirms the change against the typed,
provenanced diff. The human sees a conversation and a sourced answer — never Go,
never a build error.

That decouples two things a document format always fused: the human authoring
surface and the storage medium. Markdown optimized "a non-coder can edit this
directly." Once the authoring surface is a conversation, that constraint moves
off the storage entirely, and the storage is free to be the medium with the
best machine properties — enforcement, typed retrieval, integrity — which is
code. The result dominates the older trade on both axes at once: an easier
interface (talking beats formatting) and a stronger guarantee (a compiler beats
a validator script).

Emitting Markdown or HTML for whoever wants to read the raw store is a
read-only projection — genuinely optional. The product is the guarantee
(nothing stale, everything sourced), and the code underneath is *how*, not
*what*. Every other tool in this space has to market its format because the
format leaks into the experience; winze's does not, so winze sells the outcome.

## The two write paths

There are two authors, one enforced store:

- **agent-as-author** — an agent already fluent in the schema writes claims
  directly; the build gate is the check. No proposal layer needed.
- **human-via-agent** — a person describes something in natural language; an
  agent proposes the typed claim, the gate validates, the person confirms in
  chat. This is the path that makes winze usable by anyone, coder or not.

Both land in the same corpus behind the same gate, and concurrent writers
serialize on the corpus-wide advisory lock (see
[multi-session-write-shape.md](multi-session-write-shape.md)).
