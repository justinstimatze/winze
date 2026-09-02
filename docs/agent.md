# `winze-agent`: the agent's read/write door into a store

`winze-agent` (`cmd/agent`) is how an agent remembers and recalls. It is the
second of the two MCP surfaces this repo builds, and the distinction between
them is capability, not subject matter:

| binary | surface | tools |
|---|---|---|
| `winze-mcp` | analytic reads over a corpus | `search`, `claims`, `provenance`, `disputes`, `stats`, `theories` |
| `winze-agent` | an agent's own reads and writes | `winze_remember`, `winze_recall`, `winze_update`, `winze_link` |

Everything `winze-agent` does runs through the built winze binaries
(`winze-query`, `winze-add`, `winze-edit`). It orchestrates; it does not
reimplement. That is why a write here gets the same `gofmt && go build && go
vet` gate, the same corpus lock, and the same revert-on-failure as a write from
an editor session.

## Four faces

```
winze-agent init <dir>      # scaffold a new store, gate-checked before its first commit
winze-agent serve           # MCP server — the four tools above
winze-agent recall-hook     # SessionStart / UserPromptSubmit: injects associative recall
winze-agent capture-guard   # PreToolUse: blocks native-memory writes where a store exists
winze-agent call <tool> '<json>'   # the same handlers from a shell, for non-MCP hosts
```

`call` is what the Hermes provider in `integrations/hermes/winze/` shells out
to, so a non-MCP host gets the identical handlers rather than a second
implementation that drifts.

## The tools

- **`winze_remember(note, role?, title?, force?)`** — store a durable fact as a
  typed entity. Build-gated, auto-committed, and refused when it duplicates an
  existing memory. `force` writes anyway, and is meant to be used after reading
  what it matched, not instead of reading it.

  A high cosine alone no longer refuses. Above the block threshold the note is
  checked for **specifics its nearest neighbour lacks** — an ISO date, a commit
  sha or session id, a multi-digit number. A reworded duplicate restates a fact
  and introduces none; a recurrence arrives with its own particulars. When any
  are found the note stores with a warning naming them, rather than being
  turned away.

  A date on its own does not count. Every session note carries a fresh
  timestamp, and winze's own memory convention tells agents to write absolute
  dates, so a bare ISO date or year says only that the day changed — which a
  rewording written today says just as loudly. Before this exclusion landed,
  replaying 140 sessions refused zero of 140 writes: the gate had stopped
  gating. A date now needs some other particular beside it to read as a new
  occurrence. Confirmed live both ways — a restatement dated today is refused
  at cosine 0.79, while one carrying a commit sha and a byte count stores at
  cosine 0.92. The higher-scoring note is the one that passes, which is the
  whole point: cosine is not the discriminator.

  This came from a measured false-refusal rate, not a hunch. Replaying 20 real
  sessions across 104 days (`TestSelfRecallDecaysWithCorpusGrowth`, 2026-09-02)
  refused 2 of 20 writes at cosine 0.74 and 0.73, and neither was a duplicate:
  a note about recovering from a crashed machine was blocked by one 35 days
  older about losing sessions, and a RAM investigation by one 59 days older
  about RAM. Same topic, different incident. For session notes, recurrence is
  the normal case. With the check in place the same replay stores 20/20 and
  still recalls every note at rank 1, while a reworded restatement carrying no
  new specifics is still refused at cosine 0.76 — higher than either
  recurrence, which is why raising the threshold would not have worked.

  A refusal is the expensive mistake here. An accepted duplicate is visible
  noise the operator can merge; a refusal deletes the second occurrence and
  leaves the first standing as though it were the only one, so the store comes
  out confidently wrong rather than merely cluttered.
- **`winze_recall(query, limit?, brief_chars?)`** — hybrid BM25 + semantic
  associative search, returning compact headlines. Defaults: `limit=5`,
  `brief_chars=500`. A brief cut short by `brief_chars` ends with `… [N more
  chars — retry with brief_chars:M for the full text]` rather than a bare
  ellipsis — a live cold-recall test (2026-08-29) found a caller that hit the
  right entity at rank 1 and still reported "not found," because nothing said
  digging further with a bigger `brief_chars` was possible. Write memories as
  one fact per entity, not a multi-topic session dump: the same test found a
  single blob diluted BM25/semantic ranking enough that unrelated small
  entities outranked it on narrow queries — atomic entities fixed that half
  of the failure, the hint fixed the other half. Together they took the same
  8-question cold-recall test from 0/8 to 8/8. See
  `docs/agent-identity-integration.md`'s precondition 1.
- **`winze_update(var, note)`** — revise a memory's `Brief` in place, through
  the gate. The alternative it exists to prevent is storing a near-duplicate
  and leaving both.
- **`winze_link(from, to, relation, rationale)`** — relate two memories with a
  typed claim (`RelatesTo`, `Supersedes`, …). The link is winze's own assertion,
  so it is written as a `Conjecture` and carries no source quote by
  construction. `winze-query --schema <store>` lists the predicates.

`winze_remember` and `winze_update` also append the raw note text, verbatim,
to `raw.jsonl` beside `memory.go` — before dedup, before the onsetter check,
before the typed write even attempts. One shared file for the whole store
today (no per-nick session files exist yet; see
`docs/agent-identity-integration.md`). This is the MemPalace-shaped recovery
path: a note that gets dedup-blocked, fails the build gate, or lands in a
`Brief` a later `brief_chars` truncates is not gone, it is one grep away in
this file. Best-effort by construction — a write failure here never blocks
the real tool response — and it carries no schema and no build gate, so it
costs nothing to have and nothing to maintain (`cmd/agent/rawlog.go`).

`winze_remember` and `winze_update` also run the note past onsetter's ask
engine before committing — advisory, never blocking — so a rule-shaped memory
("always do X", "never do Y") gets the same "want a hook instead?" prompt a
`CLAUDE.md` edit would. See `docs/onsetter-integration.md` for which
`CLAUDE.md` governs this and how to point it elsewhere with
`$WINZE_AGENT_CLAUDE_MD` or `winze-agent serve --onsetter-check=<path>`.

## Creating one

```
winze-agent init <dir> [--private] [--link] [--from <winze-checkout>]
```

Copies the canonical schema (`schema.go`, `roles.go`, `predicates.go`,
`external.go`), writes a `go.mod` and an empty `memory.go`, runs `go build .`
to prove the result compiles, then `git init` and commits. `--private` installs
a pre-push hook that blocks every push; `--link` points the current repo at the
new store.

Two details that are not arbitrary:

**`bootstrap.go` is not copied.** It is winze's own founding record — 25
entities about defn, Dolt and Cyc. It used to also hold the core record types
(`Entity`, `Decision`, `FailureMode`, `Mitigation`, `OpenQuestion`), which is
why every store copied it and inherited that content along with the schema.
Those types now live in `schema.go`, so a store can take the schema without the
history. An existing store that still has `bootstrap.go` should delete it after
its next sync; the build gate will say immediately if anything referenced it.

**The content file is `memory.go`, not `<store>.go`.** `winze_remember` and
`winze_update` pass `--to memory.go`. The first version of `init` named it
after the module and produced a store that compiled cleanly and then rejected
its first write — which the compile-only test did not catch. There is now a
test that performs a real write.

The schema is copied from a winze checkout rather than embedded in the binary,
because embedding would put a second copy of the schema in the tree and a
second copy is the drift the build gate exists to prevent. The source resolves
as `--from`, then `$WINZE_SRC`, then the `corpus/` beside the running binary.

## Which store it serves

There is no single store. Several are live at once — one per team or project,
each shared by every repo that names it — so the binary resolves its target
rather than hard-coding one:

1. `$WINZE_STORE`
2. `git config --get winze.store` in the working directory
3. `~/winze-memory`, the historical default

The git step is the interesting one. Git keeps `--local` config in the common
dir, so every worktree of a repo reads the same value with no per-worktree
setup, and two different repos can deliberately name the same store.
Many-repos-to-one-store costs one `git config winze.store <path>`. The
alternative — deriving the store from the working directory — gives each
worktree its own pile and each clone another one, which is the fragmentation
this is built to avoid.

`storeRootConfigured` answers the question `storeRoot` cannot: opted in, versus
handed back the last-resort default. `capture-guard` needs that distinction,
because blocking a native-memory write is only sound advice where a store
actually exists to redirect it to. That is what makes the hooks safe to install
user-wide.

There is no alias. `$WINZE_MEMORY` and `git config winze.memory` were honoured
for one day and then removed: winze has a single user, so a deprecation window
priced for strangers is cost with no beneficiary. Before the removal, every
repo naming a store — `town/winze`, `publicai/aipotluck.org`,
`publicai/agent-service` — was migrated to `winze.store` and the old key
unset, because hooks fail open and a missed one would have looked like silence
rather than an error.

## Naming

This command was `cmd/mem` → `winze-mem` until 2026-08. Three things were
called nearly the same thing: `winze-memory` the store directory, `winze-mem`
the binary, and `winze-memory` the MCP server registration. The rename leaves
the noun to the store.

The collision was not only a reading problem. Every other binary in this repo
is named for what it does — `winze-query`, `winze-lint`, `winze-edit`,
`winze-meld` — and `winze-mem` was the only one named after a store, which it
was going to collide with the moment a second store existed. One did:
`~/Documents/publicai-memory` serves aipotluck.org and agent-service.

It also fooled the repo's own documentation gate. `TestDocsCoverageThisRepo`
requires every `cmd/` to be named in a doc, and it checks with a substring
match — so `winze-mem` was reported as documented because four unrelated docs
mention `winze-memory` in passing. The command had no doc at all. This file is
that doc, and it exists because the rename removed the accidental cover.

## See also

- `docs/agent-identity-integration.md` — the gated design where a woken agent
  reads its own context back by dispatch nick. This binary is the surface it
  would read through.
- `docs/multi-session-write-shape.md` — how N sessions write to one store
  without corrupting it or fighting in git.
- `docs/mcp-tools.md` — the other MCP servers this project consumes.
- `docs/onsetter-integration.md` — the write-time quality gate on
  `winze_remember`/`winze_update`.
