# winze-observatory (fleet dashboard)

`winze-observatory` serves an ambient fleet dashboard as a standalone local web
app — the not-a-Claude-artifact version. It reads the live corpus of each
registered instance (or the dirs you pass) and serves an always-on organism view
you can leave on a spare monitor.

```bash
winze-observatory .                       # serve the current store at http://127.0.0.1:7777
winze-observatory --open winze-memory .   # multiple stores, open a browser on start
winze-observatory --addr 127.0.0.1:8080 . # custom listen address
```

Loopback by default (`127.0.0.1:7777`) — it renders private memory stores, so it
does not listen on a public interface. Endpoints: `/` (the view), `/app.js`,
`/api/fleet.json` (nodes/edges/communities per instance), `/api/events.json`
(the live metabolism/meld event stream). Rendering is honest-only: nodes come
from entities that appear in real claims, edges are typed claims plus resolved
`[[wikilinks]]`, recency glow is real time-since-metabolism, and animation is
driven by the real event stream — no fabricated phases or melds.

Per-instance activation (systemd timers, autonomy tiers) is `winze-metabolize`;
the observatory only reads.
