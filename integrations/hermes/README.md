# winze as a Hermes memory provider

`winze_provider.py` implements the `MemoryProvider` ABC from
[NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent), so a
Hermes agent can use a winze corpus as its long-term memory.

## Setup

Build the winze binaries and point `WINZE_BIN` at them:

```bash
make build
export WINZE_BIN=$PWD/bin
```

The store resolves the way it does everywhere else — `$WINZE_MEMORY`, then
`git config --get winze.memory`, then `~/winze-memory` — so a repo that already
has a winze memory needs no further configuration.

## What it exposes

The same four tools the MCP server registers: `winze_remember`, `winze_recall`,
`winze_update`, `winze_link`. Everything goes through `winze-mem call`, which
dispatches to those same handlers, so the dedup check and the corpus build gate
apply here exactly as they do to an editor session.

Recall is automatic. `queue_prefetch` runs the search on a background thread
after each turn and `prefetch` hands the result to the next one without ever
blocking on the store; a slow or missing store costs the turn its memory and
nothing else.

## One deliberate difference

`sync_turn` does nothing.

Most providers absorb every completed turn, because their store is a transcript
index and more text is strictly more recall. winze's store is a typed corpus
that has to compile: each write runs `gofmt && go build && go vet` and reverts
on failure, and each claim carries an attribution it must be able to defend.
Feeding it raw turns would either fail the gate constantly or fill it with
untyped noise, and the second is the worse outcome — it is how a memory becomes
something you stop trusting.

So writing is a decision the agent makes by calling `winze_remember`, not a
side effect of talking. The cost is that nothing is remembered unless the agent
chooses to remember it. The benefit is that everything in the store was worth
a decision, and the `system_prompt_block` tells the model what clears that bar.

Writes are refused outside `agent_context="primary"`. A cron or subagent
context writing into the user's memory fills it with machinery talking to
itself.

## Conformance

`winze_provider.py` falls back to `MemoryProvider = object` when Hermes is not
importable, so it loads and smoke-tests on a machine with no Hermes at all.
That fallback means nothing in this repo enforces the contract — abstractness
is not checked and neither are signatures.

`conformance.py` does, wherever Hermes actually is:

```bash
PYTHONPATH=/path/to/hermes-agent python3 integrations/hermes/conformance.py
```

Exits non-zero on failure and refuses to pass silently when Hermes is absent.

Verified 2026-08-06 against `agent/memory_provider.py` from the upstream main
branch: the real ABC binds, no abstract method is left unimplemented, and every
override is call-compatible. Two differences are ignored by design —
`initialize` and `handle_tool_call` annotate `**kwargs: Any` where the base
leaves it bare, which changes nothing about how the host calls them.

**Not verified:** that Hermes's own provider discovery finds and loads the
class. That needs a running agent, and there is no Hermes install on the
machine this was written on. Treat the provider as contract-correct and
end-to-end untested until someone runs it under a real one.
