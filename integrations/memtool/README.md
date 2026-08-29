# memtool

Backs Anthropic's native memory tool (`memory_20250818`) with a winze store —
Phase 4 of the winze-agent improvement plan. Any Claude agent using the
standard memory-tool protocol gets winze's typed, provenanced guarantees,
with zero awareness that winze exists underneath.

## Why this exists

Anthropic's memory tool models a hierarchical file tree: `view`/`create`/
`str_replace`/`insert`/`delete`/`rename` on paths under `/memories`. Winze
models a flat graph of typed atomic entities with whole-`Brief` replacement
and no directory concept at all. This package is the translation layer
between the two, and it never touches winze's own corpus or schema — the
mapping lives entirely in `index.go`'s `path -> {var, creation title}`
manifest, persisted as its own JSON file beside the winze store.

Reaches winze through `winze-agent call`, the same non-MCP-host path the
Python Hermes integration (`integrations/hermes/winze/`) already uses.
`cmd/agent`'s handlers are unexported in `package main`, so this could not
import them directly even if it wanted to.

## Command mapping

| memory-tool command | winze operation |
|---|---|
| `create` (new path) | `winze_remember(force:true)` + add to the index |
| `create` (existing path) | `winze_update` — hard overwrite, no history kept |
| `view` (directory) | list index entries by path prefix |
| `view` (file) | full `Brief`, untruncated, optionally sliced by `view_range` |
| `str_replace` | fetch full `Brief`, single-occurrence text replace, `winze_update` |
| `insert` | fetch full `Brief`, line-based insert, `winze_update` |
| `delete` | remove the path from the index only — the winze entity is untouched |
| `rename` | update the index key(s) only |

`delete`/`rename` never touch winze's corpus: winze refuses hard deletion by
design, and this package respects that by construction rather than adding a
special case.

## The conversation loop

The native `memory_20250818` tool has no `input_schema` — Anthropic's API
supplies it server-side — so it does not fit the SDK's generic `BetaTool`/
`BetaToolRunner` abstraction, which always requires one. `loop.go`'s `Loop`
is the hand-rolled equivalent for this one tool: it declares
`anthropic.BetaMemoryTool20250818Param` in the request's `tools`, and for
each returned `"memory"` tool_use block, re-marshals its `Input` (an `any`)
to JSON and unmarshals it into `anthropic.BetaMemoryTool20250818CommandUnion`
before handing it to `Executor.Execute` and appending the `tool_result`.
`loop_test.go` exercises that JSON round-trip directly — the closest a unit
test gets to what a live response actually hands back — without touching the
network.

## What's not built yet

An end-to-end demo proving one live Claude conversation round-trips through
this into winze — a fact written via the memory tool in one conversation,
recalled correctly by a second conversation that shares no message history —
lives at `cmd/memtooldemo` and is written, but has not yet been *run*: it
costs real, billed API calls (`ANTHROPIC_API_KEY` required), so building and
testing it is as far as this went without that explicit go-ahead. Running it
is the actual precondition-1-shaped validation
`docs/agent-identity-integration.md` calls for — passing unit tests on
`Executor` and `Loop` is not the same claim as a live round trip succeeding.
