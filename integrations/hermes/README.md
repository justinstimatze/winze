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
