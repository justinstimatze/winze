# winze-meld (read-only union of stores)

`winze-meld` bridges two or more winze stores into a single read-only
"mind-meld" directory you can point the query tools at, then dissolve when done.

```bash
winze-meld --into /tmp/meld  winze-defn winze-memory .    # union of three stores
winze-meld --dissolve /tmp/meld                            # tear it down
```

The meld is a FROZEN snapshot: each store is materialized via `git archive` at a
pinned SHA (HEAD by default), so the meld never couples to a store's live working
tree and can be reproduced from the manifest. It is read-only by construction —
the union of two `package winze` stores cannot `go build` (duplicate
identifiers), so top-level `*.go` files are copied namespace-prefixed
(`<ns>__memory.go`) with one canonical `predicates.go`. A `.winze-meld.json`
manifest records the pinned SHAs and guards `--dissolve`.

Use it to run a query across a fleet of winze instances — the read side sees the
union, each store keeps its own history and write path.
