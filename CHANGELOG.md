# Changelog

## v0.4.0 — 2026-09-07

The full history (419 commits) is in `git log v0.3.0..v0.4.0`; grouped by
theme here instead.

### Added

- MCP server (`cmd/mcp`): structured epistemic queries (`claims`, `disputes`,
  `provenance`, `search`, `stats`, `theories`) for any agent, without the
  agent knowing the underlying infrastructure.
- `winze-agent` (renamed from `winze-mem`): `winze_remember` / `winze_recall`
  / `winze_update` / `winze_link` tools, gated through onsetter's ask engine,
  with recurrence-vs-duplicate detection replacing a pure cosine-threshold
  dedup gate.
- OKF v0.2 export (`cmd/okf`) — projects the typed corpus into a conformant
  Open Knowledge Format bundle; trust tiers fall out of the `Provenance`/
  `Conjecture` split rather than being asserted.
- Vault ingest (`--pkm`) — a markdown vault's `[[wikilinks]]` become typed,
  compiled claims.
- Multi-store meld (`cmd/meld`) — read-only union of two or more winze
  stores for cross-store query, then dissolved.
- `winze-observatory` — standalone live fleet dashboard with a typed
  frontend and a real event stream.
- Anthropic `memory_20250818` native-tool backend (`integrations/memtool`)
  and a Hermes `MemoryProvider` integration.
- `winze-benchmark` — retrieval benchmark harness (grep/BM25/defn/AST
  comparison).
- LongMemEval-based self-recall measurement harness (`cmd/longmemeval`) —
  replays real transcript sessions to measure cross-session recall under
  corpus growth.
- Laminar metabolism: phase-level self-gating, a hard monthly spend cap,
  per-call actual-spend telemetry, and typed `LearningGoal` self-directed
  curiosity.
- Trip-cycle `Conjecture` attribution — a compiler-enforced fence against
  a generated claim wearing a fabricated `Provenance` — plus
  structural-affinity/bridge-edge-aware sampling and an adversarial critic
  pass.
- New lint rules: dated-measurement, thin-conjecture, brief-drift,
  lexicon-fence, structural dedup.
- `CodeRef` / `SourceDoc` / `StructurallyAnalogousTo` schema primitives;
  `Client`/`Span` fields for cross-repo, content-hash-checked citations to
  non-Go targets.
- `--docs-recall` — semantic recall over `docs/*.md`; `CLAUDE.md` split into
  topic-scoped, recallable docs.
- CI: build, vet, staticcheck, test, lint, and topology on every push; the
  defn CLI is installed and the corpus ingested in-workflow so integration
  tests do real work instead of skipping.
- `SECURITY.md` and Dependabot, ahead of the initial public release.

### Changed

- defn migration: moved off an internal Dolt-backed index cache onto defn's
  own SQLite-backed graph (`internal/defndb` thinned to a client).
- `cmd/mem` renamed to `cmd/agent` (`winze-mem` → `winze-agent`), so the
  binary stops colliding with the store's own name.
- README repositioned to lead with the memory-continuity gap ("nobody does
  automatic, derived retrieval over a user's raw session history with no
  authoring step") over "epistemic self-awareness" as the primary pitch.
  Roadmap, OKF interop, DARPA alignment, and known problems moved out of
  README into a standalone `ROADMAP.md`.
- Corpus moved into `corpus/` so ingest (defn's and winze's own parser)
  scopes to it rather than the whole module.
- Gas Town / beads / Dolt scheduler integration removed; the metabolism
  stays scheduler-agnostic.

### Removed

- The raw-evidence retrieval tier (`raw.jsonl`, `winze_recall_raw`,
  `winze-query --raw`) — built, measured against the typed store on a
  shared harness, then retired in three phases once a `Documented` claim
  made it a structurally lossless replacement. See
  `docs/raw-evidence-retrieval.md`.
