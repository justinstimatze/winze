+++
name = "kb-health"
description = "Periodic knowledge base health check: lint stats, thin neighborhoods, stale provenance"
version = 1

[gate]
type = "cooldown"
duration = "2h"

[tracking]
labels = ["plugin:kb-health", "category:audit"]
digest = true

[execution]
timeout = "5m"
notify_on_failure = true
severity = "low"
+++

## KB Health Check

Run the winze lint suite and report structural health.

### Steps

1. Run lint and capture output:
```bash
cd /home/gas6amus/Documents/winze
go run ./cmd/lint . 2>&1
```

2. Parse output for:
   - Total entity count and role type count
   - Orphaned entities (entities with zero claims)
   - Contested concepts and their theory counts
   - Value conflicts (suppressed and unsuppressed)

3. Report findings to the bead. If any of these are true, file a new bead:
   - New orphaned entities appeared since last check
   - A contested concept has only 2 theories (could benefit from a third source)
   - An unsuppressed value conflict exists

4. Do NOT run the LLM rule during patrol (cost control). LLM checks are
   for ingest-time validation, not periodic health checks.
