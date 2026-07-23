# Authoring helper (cmd/add)

```bash
# Inline-source mode (one-off claim from a freshly-quoted source):
go run ./cmd/add \
  --to apophenia.go \
  --name MyNewClaim \
  --predicate Proposes \
  --subject KlausConrad \
  --object ConradApopheniaClinicalFraming \
  --quote "exact source text" \
  --origin "Wikipedia (zim 2025-12) / Apophenia"

# Reuse-named-source mode (preferred when adding multiple claims from
# the same source — keeps provenance unfragmented):
go run ./cmd/add \
  --to apophenia.go \
  --name AnotherClaim \
  --predicate Accepts \
  --subject MichaelShermer \
  --object ShermerPatternicityFraming \
  --provenance-var apopheniaSource
```

Appends a typed claim declaration, runs `gofmt -w`, then `go build . && go vet .`
as the gate. Reverts the file on failure. Use `--unary` for `UnaryClaim`
predicates (omit `--object`); `--dry-run` to preview the render without
touching the file. `--provenance-var <name>` reuses an existing `Provenance`
var instead of inlining one (mutually exclusive with `--quote`/`--origin`);
the build gate validates that the named var exists. `--conjecture` (with a
required `--rationale`, and optional `--generated-by`) attributes the claim as a
`Conjecture` — winze's OWN assertion, carrying no source `Quote` — for claims
winze generates rather than ingests (e.g. a memory-to-memory link). The three
attribution modes (inline `--quote`/`--origin`, `--provenance-var`,
`--conjecture`) are mutually exclusive; this is the tool-side honoring of the
mirror-source-commitments fence — a generated claim can never wear a fabricated
source. The tool does no slot-type checking of its own — the build gate is what
validates the claim, which is the load-bearing discipline this project was built
around. Do NOT relax that path.

## Batch mode

`--batch <file.jsonl>` (or `-` for stdin) appends many claims under a single
build gate — the burst-write path. The gate (~91ms warm) dominates per-claim
cost; running it once for K claims measured **5.2× faster** on a 5-claim burst
(621ms → 119ms) against this corpus. Each JSONL line is one claim with fields
mirroring the flags (`to`, `name`, `predicate`, `subject`, `object`, `quote`,
`origin`, `ingested_by`, `provenance_var`, `unary`). All-or-nothing: every
touched file reverts if any record fails validation or the gate. `--dry-run`
renders without touching files. This is the write path the multi-session
shared-KB shape relies on (see `docs/multi-session-write-shape.md`).

## Propose mode

`--propose "<rough note>"` is the human-via-agent write path: an LLM (Haiku by
default, `--model` to override) maps a natural-language note onto the EXISTING
predicate/entity vocabulary and proposes one typed claim (predicate + subject +
object), reusing existing entity vars. The proposal is validated against the
corpus and rendered; a referenced entity that doesn't exist is reported with
nearest-existing suggestions (coin-time dedup nudge) rather than silently
coined. Target file is inferred from the subject's file unless `--to` is given.
**Provenance is never invented** — the LLM proposes structure only;
`--quote`/`--origin` (or `--provenance-var`) still supply the source, and
`--commit` is refused without them. Default is preview; `--commit` routes the
claim through the same build gate as direct add. The note is treated as
untrusted data (mapped, not obeyed). The stable vocabulary prefix is
`cache_control`-marked, so repeated proposals in a session read it at ~10%
input cost. See `docs/typed-citation.md` (the two write paths).
