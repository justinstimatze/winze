# Docs recall (how the detailed reference reaches you)

CLAUDE.md is loaded into every session in this project regardless of what the
session is doing. Everything that has to be resident competes for that space, so
the file grows until nobody adds to it and new tools go undocumented. The fix is
the shape winze-memory already runs: keep a small always-loaded core, move the
reference into `docs/`, and surface only the sections a given prompt implicates.

## The mechanism

`winze-query --docs-recall "<prompt>"` chunks `docs/*.md` (and the top-level
`README`/`CONTRIBUTING`/…) by H2 section, embeds each chunk with the same local
ollama `all-minilm` and content-addressed vector cache the corpus `--semantic`
search uses, and prints the top few `file#anchor` sections above a cosine floor.
CLAUDE.md itself is excluded — it is the resident core, not recall material.

```bash
winze-query --docs-recall "how do I merge two duplicate entities" .
winze-query --docs-recall "concurrent writes from multiple sessions" --json .
winze-query --docs-recall "trip vs reverie" --docs-top 5 --docs-floor 0.25 .
```

A chunk is one H2 section, prefixed with its file's H1 for standalone context
(so "Batch mode" recalls as "Authoring helper › Batch mode"). Recall points at
the section, not the whole file, so the reader opens `docs/editing.md#merge`
rather than a 200-line file.

## Wiring

A `UserPromptSubmit` hook runs `--docs-recall` on the prompt and injects the
pointers, the same way the winze-memory recall hook injects memories. It never
blocks a prompt: if ollama is down the embed fails, the tool prints nothing and
exits 0. Precision-first — a marginal section shown every prompt trains the
reader to ignore the banner — so the floor is set to surface only genuine hits.

Tuning: `WINZE_DOCS_RECALL_MIN` overrides the cosine floor; `--docs-top` caps
how many sections surface. The vector cache in `.winze-embed/` is shared with
`--semantic` and content-addressed, so an unchanged section never re-embeds and
editing one doc only re-embeds that doc's changed chunks.
