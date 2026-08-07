"""Check WinzeMemoryProvider against the real Hermes MemoryProvider ABC.

winze_provider.py falls back to `MemoryProvider = object` when Hermes is not
importable, so it loads and its tests pass on a machine with no Hermes at all.
That fallback is deliberate — the module should be readable and smoke-testable
standalone — but it means nothing in the winze repo ever enforces the contract.
Abstractness is not checked, and neither are signatures.

Run this wherever Hermes actually is:

    PYTHONPATH=/path/to/hermes-agent python3 integrations/hermes/conformance.py

Exits non-zero on any failure. What it cannot check is the last mile: whether
Hermes's own provider discovery finds and loads the class. That needs a running
agent.
"""

from __future__ import annotations

import inspect
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "winze"))

try:
    from agent.memory_provider import MemoryProvider
except ImportError:
    sys.exit("hermes not importable — set PYTHONPATH to a hermes-agent checkout")

import importlib
wp = importlib.import_module("__init__")

failures: list[str] = []

if wp.MemoryProvider is not MemoryProvider:
    failures.append("winze_provider fell back to its stub ABC; the real one was not imported")

unimplemented = set(getattr(wp.WinzeMemoryProvider, "__abstractmethods__", frozenset()))
if unimplemented:
    failures.append(f"abstract methods not implemented: {sorted(unimplemented)}")

# Python does not check override signatures, so a wrong one passes abstractness
# and fails at call time instead. Annotation-only differences are ignored: they
# do not change how the host can call the method.
for name, mine in inspect.getmembers(wp.WinzeMemoryProvider, callable):
    base = getattr(MemoryProvider, name, None)
    if name.startswith("_") or base is None or not callable(base):
        continue
    try:
        a, b = inspect.signature(mine), inspect.signature(base)
    except (TypeError, ValueError):
        continue
    if [(p.name, p.kind) for p in a.parameters.values()] != [
        (p.name, p.kind) for p in b.parameters.values()
    ]:
        failures.append(f"{name} signature differs: base {b}, ours {a}")

try:
    wp.WinzeMemoryProvider()
except Exception as exc:  # noqa: BLE001 - any failure here is a conformance failure
    failures.append(f"cannot instantiate: {exc}")

if failures:
    for f in failures:
        print(f"FAIL {f}", file=sys.stderr)
    sys.exit(1)
print("conforms: real ABC bound, all abstract methods implemented, signatures compatible")
