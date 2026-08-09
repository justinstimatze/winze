# Feedback: onsetter should gate winze-agent writes, not just CLAUDE.md

Not yet implemented. This is a forward note for when memory moves fully onto
`winze-agent` — recorded here rather than in CLAUDE.md's doc index because
there's no code behind it yet.

## The gap

[onsetter](https://github.com/justinstimatze/onsetter) intercepts `Write`/`Edit`
calls that touch a `CLAUDE.md` path and match rule-like phrasing (`always`,
`never`, and similar). When it matches, it quotes the line back and asks
whether the rule should be a hook instead of prose — because a `CLAUDE.md`
line is read inconsistently (only when the file happens to be in context),
while a hook fires on every matching tool call. See `stull`
(github.com/justinstimatze/stull) for the enforcement side: a `CLAUDE.md` rule
that's actually a deterministic guard belongs in a `Machine`, not a bullet
point that has to be re-read into context to matter.

Memory is moving off CLAUDE.md's `auto memory` convention and onto
`winze-agent` (`winze_remember` / `winze_recall` / `winze_update` /
`winze_link` — see `docs/agent.md`). Once that's the live substrate, a
`type: feedback` memory that reads as an enforceable rule ("always do X",
"never do Y") can get written straight into a winze store via
`winze_remember`, completely outside onsetter's current gate — onsetter only
watches `CLAUDE.md` file writes. The rule-vs-recall distinction onsetter
exists to catch doesn't go away when the storage moves; the interception
point onsetter currently uses does.

## What's needed

onsetter's gate needs a second surface: the `winze-agent` write path
(`winze_remember`, and `winze_update` where it rewrites a claim's text),
checked with the same content pattern it already applies to `CLAUDE.md` diffs.
A memory that matches should get the same "want a hook instead?" prompt,
scoped to whichever project the memory's context implies (or asked, if
ambiguous) — same "gate on path AND content, quote the match back, end with
an explicit if-this-is-fine-continue" shape onsetter already uses, just with
`winze_remember`'s call site as the second gate instead of a file write.

Whether that's a change inside `winze-agent`'s MCP server (call out to
onsetter before a write commits) or a change inside onsetter (subscribe to
winze-agent's write path the way it currently watches file writes) is an
onsetter-side design call, not a winze one — flagging the gap here since it's
winze's migration that opens it.

## Status: the onsetter-side blocker cleared (2026-08-09)

onsetter reported over dispatch that its ask-matching engine is now importable
from outside onsetter — `github.com/justinstimatze/onsetter/ask`, formerly
`internal/ask`. Two specifics that decide the shape of the work above:

- `ask.Match(e Edit)` now accepts `Edit.Path == ""`, so `when:` / `not:` /
  `has:` headers run against note-only content. An ask that also sets `in:` /
  `not-in:` / `untouched:` rejects with a Result saying it needs a path — by
  design, so those headers are simply not usable on a path-less call.
- `discover.Roots(path)` was **not** changed and still needs a real path to
  climb from, which a `winze_remember` call does not have. So the open question
  is no longer "can onsetter see a path-less write" but "which `CLAUDE.md`
  governs one" — the store's own, the calling cwd's, or an explicit setting.
  That is a winze-side decision, and it is the only thing still unanswered.

Full writeup in onsetter's `INTEGRATIONS.md`.
