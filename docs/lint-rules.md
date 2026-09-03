# Lint rules

`go run ./cmd/lint corpus` runs the deterministic rules; add `--llm --llm-max-calls=5`
for the LLM contradiction check.

The rules: naming-oracle, orphan-report, value-conflict, contested-concept,
brief-check, provenance-split, llm-contradiction, brief-drift, structural-dedup,
lexicon-fence, thin-conjecture, dated-measurement, coderef-mutual-exclusion,
coderef-span, coderef-existence.

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

## thin-conjecture

`thin-conjecture` flags a `Conjecture` literal whose `Rationale` is the empty
string (key omitted counts the same as `Rationale: ""`). A `Conjecture` is
uncitable by construction — no `Quote` field, closed by the compiler — but its
honesty depends on `Rationale` actually carrying winze's own reasoning for the
connection; an empty one is generated knowledge with no reasoning recorded at
all, and Go's zero value lets that compile silently. Hard failure (exit 1): a
missing `Rationale` has no legitimate reading the way a bare Brief mention does
for brief-drift.

Walks every expression under each var decl looking for a `Conjecture` literal
at any depth, not just the var's own top-level type — `cmd/add --conjecture`
inlines the literal into the claim's `Prov` field
(`var X = TheoryOf{..., Prov: Conjecture{...}}`), it is never a standalone
top-level var, so a shallower check finds nothing against the real corpus.

Deliberately narrow: only the empty string is flagged. A short-but-present
`Rationale` ("TBD") is a real quality question too, but there's no
forcing-function instance of it yet to say what "thin" means in practice — see
the project's own schema-accretion discipline — so this rule waits for one
rather than inventing a threshold.

Motivated by a dispatch exchange with onsetter (justinstimatze/onsetter):
onsetter's write-time CLAUDE.md ask-block layer was proposed as the gate, but
its `PreToolUse` hook is hardcoded to `Write|Edit` and every corpus `.go`
mutation here goes through the `defn` MCP tool per this project's own
CLAUDE.md — the ask would never fire. A deterministic lint rule sees the AST
on disk regardless of which tool wrote it, so it's the mechanism that actually
covers winze's real write path.

## dated-measurement

`dated-measurement` flags a `Brief`, `Rationale`, or `Quote` that reads as a
measurement or a claim about present state (matching a bare number-plus-unit
like `24GB`/`97%`/`12ms`, or a present-tense status word — `currently`, `now`,
`is backed by`, `links in`, `takes`) with no ISO date (`\d{4}-\d{2}-\d{2}`)
anywhere in the same string. This is README known-problem 5 made concrete:
`internal/defndb`'s package doc asserted defn was Dolt-backed and too heavy to
link, that stopped being true, and nothing noticed until an agent read the
stale claim as a live measurement and built ~500 lines on top of it.

Advisory (returns 0 regardless of hits), not blocking: a real run against the
corpus (2026-09-02) found 10 of 496 strings scanned, and the pattern is
deliberately loose (README known-problem 5's own "first cut") — most of the
real hits were historical decades ("in the 1970s") inside sourced `Quote`
text, which cannot take an ISO date without either inventing one or rewriting
the citation's exact source text, and one was a verbatim user remark
containing the word "now." A precise-enough check to gate on belongs in CI,
not as a loose heuristic; this stays a report until the false-positive rate
says otherwise.

## coderef-mutual-exclusion

`coderef-mutual-exclusion` flags a `CodeRef` literal that sets both `Symbol`
and `Client` — see `corpus/schema.go`'s `CodeRef` doc for why that's a
structural contradiction: a citation is either this store's own module
(compiler-checked via `Symbol`) or an external client (lint-checked via
`Client`), never ambiguously either. Always on, zero cost — no client
resolution or file I/O needed — so it runs unconditionally like
`naming-oracle`. Hard failure (exit 1): there's no legitimate reading of a
citation claiming to be both at once.

## coderef-span

`coderef-span` is the mechanism for the citation shape `Symbol` can never
reach: a non-Go target, or any citation where byte-exact content rather than
mere existence is the asserted property (`corpus/schema.go`'s `CodeSpan`,
`docs/typed-citation.md`, `FEEDBACK-2026-09-02.md#1`). A `CodeRef` with
`Span` set is checked by reading the exact line of text at `Span.Line` in the
cited file and comparing its sha256 hash to `Span.Hash`.

A `Client == ""` span (a citation to a file inside the store's own repo) is
always checked, resolved relative to the store's root. A `Client != ""` span
(an external target) is checked against the path named for that client via:

```
--clients=name=path,name=path              # inline, comma-separated
--clients-file=<path to JSON {"name":"path"}>
$WINZE_CLIENTS_FILE                         # same JSON shape, env fallback
.winze-clients.json                         # default file in the working directory, gitignored
```

With no client configured for a given `Client`-bearing `Span` ref, the check
skips cleanly (same posture as `--llm`) rather than failing — client paths
are machine-local and there's nothing wrong with a corpus authored without
them checked out.

**Authoring a `Span` citation**: `cmd/add` has no flag for this today —
`CodeRef`/`SourceDoc` are hand-authored Go literals, same as the two
same-module citations that predate `Client`/`Span`. Compute `Span.Hash` with
the exact recipe `lineAt`/`hashLine` (`cmd/lint/coderef_span.go`) reproduce,
byte-for-byte, no trailing newline:

```
awk 'NR==713{sub(/\r$/,""); printf "%s", $0}' path/to/file | sha256sum
```

If a third citation ever needs this, promote it to a `cmd/lint
--hash-line=path:N` convenience flag — not before, per this project's own
third-occurrence promotion discipline.

A `CodeRef` with `Client != ""` and `Span == nil` (an existence-checked
citation to a Go symbol in a different module) is a distinct shape this rule
does not check — see `coderef-existence` below.

## coderef-existence

`coderef-existence` is milestone 2: the Go-to-Go half `coderef-span`
deliberately doesn't handle. A `CodeRef` with `Client != ""` and `Span == nil`
names a package-qualified Go symbol (e.g.
`"github.com/user/otherproject/internal/foo.Bar"`) in an external client's
module. The rule loads that client with
[`golang.org/x/tools/go/packages`](https://pkg.go.dev/golang.org/x/tools/go/packages)
(`packages.Load` with `NeedName|NeedTypes`, `./...`), builds a
`pkgPath.Name` set from each loaded package's `Types.Scope().Names()`, and
flags any cited symbol not in that set — a rename or deletion, caught the
same way `go build` would catch it in-module.

Deliberately not hash-based, unlike `coderef-span`: `go/packages` correctly
ignores a harmless `gofmt` pass or a comment edit that a hash would flag as
drift. Same `--clients`/`--clients-file` resolution as `coderef-span` (see
above), same posture — skips cleanly with nothing configured.

**Known, deliberate limitation**: `Types.Scope().Names()` only sees
package-level declarations, not methods (`(*Foo).Bar`) — fine for the
motivating case (a plain function), a real gap if a method citation shows up
later. Not solved speculatively; extend when a real citation needs it.
