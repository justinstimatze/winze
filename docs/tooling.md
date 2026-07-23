# Build the tools first (interactive speed)

```bash
make build      # -> ./bin/winze-{query,lint,topology,metabolism,add,...}
make install    # -> $GOBIN (defaults to ~/go/bin)
```

`go run ./cmd/foo` recompiles on every invocation and costs ~0.5s before the
tool starts. Measured on this corpus: `go run ./cmd/query --stats .` is 497ms
against 27ms for the built binary — an 18x tax on the operation a knowledge
base exists to make cheap. The `go run` forms in the other docs still work and
are fine for batch phases; use the built binaries for anything interactive.

Reference timings (built binaries, warm): query 12-47ms (stats/hybrid),
topology ~90ms, lint ~260ms, the per-claim gate (`go build . && go vet .`)
~90ms warm (build ~37ms + vet ~55ms) and up to ~400ms cold or under load.
`go run` adds ~0.5s compile on top — use the built binaries for anything
interactive.
