# Query interface

```bash
go run ./cmd/query "consciousness" .              # substring search entities by name/brief/alias
go run ./cmd/query --fulltext "pattern detection failure" .  # BM25 fulltext ranking over Briefs + provenance Quotes
go run ./cmd/query --semantic "machine that seems to understand but does not" .  # embedding search via local ollama all-minilm (~53ms cached)
go run ./cmd/query --hybrid "confirmation bias" .   # reciprocal-rank-fusion of BM25 + semantic into one list
go run ./cmd/query --hybrid "consciousness" --type Hypothesis .  # type-aware: filter hybrid results to a verified role (zero-classification-error)
go run ./cmd/query --hybrid "apophenia" --expand .  # append each hit's typed claim neighborhood (predicate → neighbor + role) — reasoning-ready context
go run ./cmd/query --dupes ConfirmationBias .       # structural twins: same-role entities sharing this one's claim-neighborhood (coin-time dedup)
go run ./cmd/query --theories "apophenia" .        # competing theories of a concept
go run ./cmd/query --claims "Chalmers" .           # all claims involving an entity
go run ./cmd/query --provenance "Sagan" .          # provenance trail for a source
go run ./cmd/query --disputes .                    # all active disputes
go run ./cmd/query --stats .                       # KB summary statistics
go run ./cmd/query --schema .                       # the type model: roles, predicate signatures, attribution modes
go run ./cmd/query --reverie .                      # associative walk over the claim graph (random start; add a seed to steer); the grounded "trip"
go run ./cmd/query --decisions .                    # the decision log: current vs superseded, over the Supersedes graph (see docs/decisions.md)
go run ./cmd/query --docs-recall "<prompt>" .       # semantic recall over docs/*.md: the file#anchor sections a prompt implicates
go run ./cmd/query --json "consciousness" .        # JSON output
go run ./cmd/query --ask "What theories compete on consciousness?" .  # LLM-powered natural language query
go run ./cmd/query --ask .                         # interactive REPL mode
```

The read side of the KB. Parses corpus `.go` files with `go/ast`, builds
an in-memory index of entities, claims, and provenance, and answers queries.
`--ask` mode sends the full KB context to an LLM for natural language answers
(needs ANTHROPIC_API_KEY). `--docs-recall` operates on the prose docs, not the
corpus — it is the read side of the split-CLAUDE.md docs (see
`docs/docs-recall.md`). For richer queries (multi-hop, aggregation), use
`defn` MCP directly in a Claude Code session.
