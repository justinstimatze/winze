"""winze as a Hermes memory provider.

Drop-in for NousResearch/hermes-agent's MemoryProvider ABC. Everything reaches
winze through `winze-agent call`, which dispatches to the same handlers the MCP
server registers — so the dedup check and the corpus build gate apply here
exactly as they do to an editor session.

The one place this provider deliberately differs from its siblings: sync_turn
does nothing. Most memory providers treat every completed turn as something to
absorb, because their store is a transcript index and more text is strictly
more recall. winze's store is a typed corpus that has to compile, every write
runs `gofmt && go build && go vet` with a revert on failure, and every claim
carries an attribution it must be able to defend. Feeding it raw turns would
either fail the gate constantly or fill it with untyped noise, and the second
is worse. Writing is a decision the agent makes by calling winze_remember, not
a side effect of talking.

Install
-------
Point HERMES at this file and set WINZE_BIN to the directory holding the winze
binaries (or put winze-agent on PATH). The store itself resolves the way it
does everywhere: $WINZE_STORE, then `git config --get winze.store`, then
~/winze-memory. The older $WINZE_MEMORY and winze.memory are still honoured.
"""

from __future__ import annotations

import json
import os
import queue
import shutil
import subprocess
import threading
from typing import Any, Dict, List, Optional

try:
    from agent.memory_provider import MemoryProvider
except ImportError:  # pragma: no cover - lets the module import standalone
    MemoryProvider = object  # type: ignore[assignment,misc]


# Recall is on the turn path, so it gets a short leash. A slow store should
# cost the turn nothing but its own recall, and a hung one should not wedge the
# agent — prefetch returns "" and the turn proceeds without memory.
RECALL_TIMEOUT_S = 10
# Writes run the build gate, which compiles the corpus. That is seconds, not
# milliseconds, and it is the reason winze writes are trustworthy.
WRITE_TIMEOUT_S = 120

_WRITE_TOOLS = frozenset({"winze_remember", "winze_update", "winze_link"})


class WinzeMemoryProvider(MemoryProvider):
    """Typed, gated memory backed by a winze corpus."""

    def __init__(self) -> None:
        self._session_id = ""
        self._agent_context = "primary"
        self._prefetch_q: "queue.Queue[str]" = queue.Queue(maxsize=4)
        self._pending: Optional[str] = None
        self._worker: Optional[threading.Thread] = None
        self._stop = threading.Event()

    # -- Core lifecycle ----------------------------------------------------

    @property
    def name(self) -> str:
        return "winze"

    def is_available(self) -> bool:
        """True when the binary is reachable. No network calls, per the ABC.

        Store reachability is deliberately not checked here: resolving it can
        touch git config and the filesystem, and a store that exists at init
        can still vanish. Recall failures degrade to empty context anyway, so
        the expensive check buys nothing the cheap one does not.
        """
        return self._binary() is not None

    def initialize(self, session_id: str, **kwargs: Any) -> None:
        self._session_id = session_id
        self._agent_context = kwargs.get("agent_context", "primary")
        self._stop.clear()
        self._worker = threading.Thread(
            target=self._prefetch_loop, name="winze-prefetch", daemon=True
        )
        self._worker.start()

    def shutdown(self) -> None:
        self._stop.set()
        # Unblock the worker's get() so the thread can observe the stop flag.
        try:
            self._prefetch_q.put_nowait("")
        except queue.Full:
            pass
        if self._worker is not None:
            self._worker.join(timeout=2)

    # -- Recall ------------------------------------------------------------

    def system_prompt_block(self) -> str:
        return (
            "You have a winze memory: a typed knowledge base that compiles. "
            "Recalled memories arrive as context automatically; you do not need "
            "to ask for them. Call winze_remember when you learn something "
            "durable that the code and the conversation would not already tell "
            "a later session — a decision and its reason, a constraint, a "
            "preference. Every write is type-checked and refused if it "
            "duplicates an existing memory, so write the fact, not a summary of "
            "the conversation."
        )

    def prefetch(self, query: str, *, session_id: str = "") -> str:
        """Return the recall queued last turn. Never blocks on the store."""
        out, self._pending = self._pending, None
        return out or ""

    def queue_prefetch(self, query: str, *, session_id: str = "") -> None:
        try:
            self._prefetch_q.put_nowait(query)
        except queue.Full:
            # Recall is best-effort context, so dropping one is correct.
            # Blocking the turn to guarantee it is not.
            pass

    def _prefetch_loop(self) -> None:
        while not self._stop.is_set():
            try:
                query = self._prefetch_q.get(timeout=0.5)
            except queue.Empty:
                continue
            if self._stop.is_set() or not query.strip():
                continue
            self._pending = self._recall(query)

    def _recall(self, query: str) -> str:
        raw = self._call(
            "winze_recall",
            {"query": query, "limit": 5, "brief_chars": 240},
            timeout=RECALL_TIMEOUT_S,
        )
        try:
            hits = json.loads(raw).get("hits", [])
        except (json.JSONDecodeError, AttributeError):
            # A non-JSON body is the tool's own error text. It belongs in the
            # log, not in the model's context as if it were a memory.
            return ""
        if not hits:
            return ""
        lines = ["## Recalled from winze memory\n"]
        for h in hits:
            lines.append(f"- **{h.get('name', '?')}** ({h.get('role_type', '?')}) — {h.get('brief', '')}")
        return "\n".join(lines)

    def sync_turn(
        self,
        user_content: str,
        assistant_content: str,
        *,
        session_id: str = "",
        messages: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        """Intentionally empty — see the module docstring."""

    # -- Tools -------------------------------------------------------------

    def get_tool_schemas(self) -> List[Dict[str, Any]]:
        return [
            {
                "name": "winze_remember",
                "description": (
                    "Save a durable fact to memory. Refused if it duplicates an "
                    "existing memory, and type-checked before it lands. Write the "
                    "fact and why it matters, not a recap of the conversation."
                ),
                "parameters": {
                    "type": "object",
                    "properties": {
                        "note": {"type": "string", "description": "The fact, in prose."},
                        "title": {"type": "string", "description": "Short headline. Derived from the note if omitted."},
                        "role": {"type": "string", "description": "Entity role: Concept, Person, Hypothesis, Place, Event. Defaults to Concept."},
                        "force": {"type": "boolean", "description": "Write anyway despite a near-duplicate. Use only after reading what it matched."},
                    },
                    "required": ["note"],
                },
            },
            {
                "name": "winze_recall",
                "description": (
                    "Search memory directly. Relevant memories are already injected "
                    "each turn, so reach for this only when you need something the "
                    "automatic recall missed, or the full text of a truncated one "
                    "(tighten the query and set brief_chars to 0)."
                ),
                "parameters": {
                    "type": "object",
                    "properties": {
                        "query": {"type": "string", "description": "Natural language or keywords."},
                        "limit": {"type": "number", "description": "Max memories to return. Default 5."},
                        "brief_chars": {"type": "number", "description": "Truncate each brief to this many chars. 0 for full text."},
                    },
                    "required": ["query"],
                },
            },
            {
                "name": "winze_update",
                "description": "Revise an existing memory in place, keeping its identity and links. Use when a fact changed rather than writing a second, contradicting memory.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "var": {"type": "string", "description": "The memory's var_name, as returned by winze_recall."},
                        "note": {"type": "string", "description": "The corrected fact, in full. Replaces the old text."},
                        "title": {"type": "string", "description": "New headline, if it should change."},
                    },
                    "required": ["var", "note"],
                },
            },
            {
                "name": "winze_link",
                "description": "Record a typed relationship between two memories, with your reason for it. This is what makes recall associative rather than a keyword index.",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "from": {"type": "string", "description": "Source memory var_name."},
                        "to": {"type": "string", "description": "Target memory var_name."},
                        "rationale": {"type": "string", "description": "Why these are related. Required — an unexplained link is not usable later."},
                        "relation": {"type": "string", "description": "Predicate name. Defaults to a general relation."},
                        "name": {"type": "string", "description": "Explicit var name for the link."},
                    },
                    "required": ["from", "to", "rationale"],
                },
            },
        ]

    def handle_tool_call(self, tool_name: str, args: Dict[str, Any], **kwargs: Any) -> str:
        if tool_name in _WRITE_TOOLS and self._agent_context != "primary":
            # Per the ABC: a cron or subagent context writing to the user's
            # memory corrupts it with machinery talking to itself.
            return json.dumps({"error": f"winze is read-only in agent_context={self._agent_context!r}"})
        timeout = WRITE_TIMEOUT_S if tool_name in _WRITE_TOOLS else RECALL_TIMEOUT_S
        return self._call(tool_name, args, timeout=timeout)

    # -- Subprocess --------------------------------------------------------

    # winze-agent was the old name for this binary. It is still accepted so a
    # host with an older winze checkout keeps working instead of reporting the
    # store as simply unavailable.
    _BINARY_NAMES = ("winze-agent", "winze-mem")

    def _binary(self) -> Optional[str]:
        bindir = os.environ.get("WINZE_BIN")
        for name in self._BINARY_NAMES:
            if bindir:
                candidate = os.path.join(bindir, name)
                if os.access(candidate, os.X_OK):
                    return candidate
            found = shutil.which(name)
            if found:
                return found
        return None

    def _call(self, tool: str, args: Dict[str, Any], *, timeout: int) -> str:
        binary = self._binary()
        if binary is None:
            return json.dumps({"error": "winze-agent not found (set WINZE_BIN or put it on PATH)"})
        try:
            proc = subprocess.run(
                [binary, "call", tool, json.dumps(args)],
                capture_output=True,
                text=True,
                timeout=timeout,
                # WINZE_BIN is inherited: winze-agent shells out to winze-query
                # and winze-add, and finds them the same way this does.
                env=os.environ.copy(),
            )
        except subprocess.TimeoutExpired:
            return json.dumps({"error": f"{tool} timed out after {timeout}s"})
        except OSError as exc:
            return json.dumps({"error": f"{tool} failed to start: {exc}"})
        if proc.returncode != 0:
            # winze-agent exits non-zero on a tool error and puts the reason on
            # stdout, so the body is the useful part either way.
            return json.dumps({"error": (proc.stdout or proc.stderr).strip()})
        return proc.stdout.strip()


# ---------------------------------------------------------------------------
# Plugin entry point
# ---------------------------------------------------------------------------

def register(ctx) -> None:
    """Register winze as a memory provider plugin."""
    ctx.register_memory_provider(WinzeMemoryProvider())
