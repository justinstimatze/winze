# MCP tools available (for any agent or human session)

- **defn** (`mcp__defn__code`): SQL-backed code database. Query entities, claims,
  provenance across the entire KB. Use for multi-hop queries, aggregation, and
  cross-file analysis. The ingest pipeline can use this to check entity existence,
  find claim context, and validate predicates.
- **adit** (`mcp__adit__*`): Code quality scoring. `adit_score_file` rates
  agent-writability of corpus files. `adit_blast_radius` measures change impact.
  Use during dream phase to assess KB health and prioritize maintenance.
- **wikipedia-zim** (`mcp__wikipedia-zim__*`): Direct ZIM article access. Search,
  read articles, extract links. Use for sensor queries and article exploration
  beyond the metabolism CLI.

These MCP servers are in the global config — all sessions inherit them.
