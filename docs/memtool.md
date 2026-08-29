# `integrations/memtool`: winze as the backend for Anthropic's native memory tool

Phase 4 of the winze-agent improvement plan. Full detail, the command-mapping
table, and what's built versus what's left live in
`integrations/memtool/README.md` (package docs, not scanned by this repo's
doc-coverage gate — this file is the pointer that satisfies it and the entry
point for anyone starting from `docs/`).

In one sentence: a Claude agent that only knows Anthropic's standard
`memory_20250818` tool protocol (`view`/`create`/`str_replace`/`insert`/
`delete`/`rename` over a virtual `/memories` tree) gets winze's typed,
provenanced storage underneath, with zero awareness winze exists.

- `index.go` — the `path -> {var, creation title}` manifest translating the
  memory tool's file-tree model onto winze's flat entity graph.
- `backend.go` — reaches winze through `winze-agent call`, the same
  non-MCP-host seam `integrations/hermes/winze/` already uses.
- `executor.go` — `Executor.Execute` maps each of the six memory-tool
  commands onto `winze_remember`/`winze_update`/the index.
- `loop.go` — `Loop`, the hand-rolled `tool_use` conversation loop the native
  memory tool needs (it has no `input_schema` of its own, so it doesn't fit
  the SDK's generic `BetaToolRunner`).
- `cmd/memtooldemo` — a live, two-conversation demo: one conversation writes
  a fact via the memory tool, a second conversation sharing no message
  history recalls it. Requires `ANTHROPIC_API_KEY` and makes real, billed API
  calls — it is written and build-gated but has not yet been run.

## See also

- `docs/agent.md` — the four `winze-agent` tools this backend calls through.
- `docs/agent-identity-integration.md` — the precondition-1 validation shape
  `cmd/memtooldemo` is a live instance of, once actually run.
