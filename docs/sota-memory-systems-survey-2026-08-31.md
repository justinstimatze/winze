# SOTA memory/continuity systems — survey (2026-08-31)

Compiled while scoping wanigan's own continuity claim
(`~/Documents/wanigan/CONTINUITY-PLAN.md` §1a/§4); cross-posted here since
winze faces the adjacent question of where an *authored*, curated memory
store sits relative to the field. Every row below reflects a doc/source
actually checked on 2026-08-31, not general training-time recall — confidence
is marked per row, and anything not directly checked says so.

## The landscape

| System | Continuity mechanism | Authoring required | Retrieval over prior sessions | Confidence |
|---|---|---|---|---|
| Claude Code | `/compact` summary, `--resume`, `CLAUDE.md`, auto-memory | human writes memory files | no | high — it is the host |
| Codebase-index IDEs | semantic index over **code** | none | no — indexes code, not conversation | high |
| Letta / MemGPT-style | tiered memory (core/recall/archival), agent promotes to long-term | the agent authors each memory | partial, over authored memories only | medium — from training, not independently verified this pass |
| winze (this project) | typed store with provenance, recall hook | human/agent curates entries | over the curated store | high — its own source |
| mem0 | vector store, agent/pipeline extracts facts | pipeline extracts (not the base agent, but still an authoring step) | yes, over extracted facts | high — checked their published benchmark page 2026-08-31 |
| Anthropic `memory_20250818` | plain file read/write in `/memories` | human or agent writes/edits files | no — view/create/str_replace only, no search | high — checked the tool spec 2026-08-31 |
| Cursor / Windsurf | static rules; Windsurf adds short auto-notes (Cascade Memories) | none (rules) / auto (notes) | no — no retrieval over conversation history either way | high — checked Windsurf's own docs 2026-08-31 |

**The gap nobody in this table fills:** automatic, derived, semantic
retrieval over a user's own prior sessions, keyed on live intent, with no
authoring step. Letta needs the agent to write memories. winze needs
curation. `CLAUDE.md` needs a human. Code indexes do not index
conversations. Every system that *does* retrieve does it over something
that was authored first — extracted (mem0), promoted (Letta), or written
(Anthropic's tool, Windsurf's notes). None retrieves over the raw
transcript itself with the query seen before an answer gets compressed.

## mem0 — the one with a published number, and its caveat

mem0 publishes 92.5 overall on LoCoMo (single-hop 94.6, multi-hop 95.4,
temporal 82.3) and 94.4 on LongMemEval, at roughly 6,956 mean tokens per
retrieval against 25,000+ for full-context baselines. That's a real, public
number — but LoCoMo (arXiv 2402.17753: 1,986 QA pairs over 10 synthetic
conversations) is a synthetic, adversarially-constructed, multi-persona
corpus, built specifically to compare products at scale, not a real user's
actual session history. Porting a LoCoMo number onto a single-user, one-host
system is a fabricated comparison, not a measurement — the benchmark and the
deployment target aren't the same population. Worth remembering before
citing any LoCoMo/LongMemEval number as if it transfers.

## Letta / MemGPT — agent-authored tiers

The agent itself calls a tool, mid-reasoning, to decide what's worth
keeping (core/recall/archival tiers, agent-edited). No published recall
benchmark from Letta itself was found during this check — the "medium"
confidence above reflects that the architecture description is
well-established but the quality claim isn't independently verified.

## Anthropic `memory_20250818` — the shipped baseline, not a search tool

Plain-file read/write (`view`/`create`/`str_replace`) in a `/memories`
directory at the API layer — the same authoring-required shape as a human
editing `CLAUDE.md` at the CLI layer, just agent-driven instead of
human-driven. No search, no ranking: the agent has to already know what to
reread, or content not already summarized is simply gone. This is the
mechanism any retrieval-based system is actually competing against for
Anthropic's own users, not a synthetic leaderboard.

## Cursor / Windsurf — the closest a mainstream product gets, and still short

Windsurf's Cascade Memories are short, auto-generated, workspace-scoped
notes. Windsurf's own documentation describes them as useful for stable
preferences, explicitly *not* useful for "remember the plan we built
yesterday." That's a genuine, vendor-acknowledged limit, not a claim of
weakness — a note compressed without the future query in mind can't answer
that query, no matter how good the summarizer is. It's independent
confirmation, from a different team shipping a different product, of the
same failure mode any purely-summarization-based continuity mechanism runs
into.

## Methodology note

Every "high confidence" row above reflects a source actually read on
2026-08-31 (official docs, a published benchmark page, or the vendor's own
product page), not recalled from training. Anything marked lower confidence
says so rather than presenting architecture-from-memory as a verified claim.
