---
name: curate
description: >
  Interactive knowledge base curation for winze. Ingest sources, audit
  neighborhoods, check predictions, view KB health. Use when working
  on winze as a crew member or in interactive sessions.
allowed-tools: "Bash(go *), Bash(git *), Bash(gt *), Bash(bd *), Read, Write, Edit, Glob, Grep, Agent"
version: "1.0.0"
author: "winze"
argument-hint: "<command> [args] — status, ingest <source>, audit [entity], predict, resolve, sensor"
---

# /curate — Winze KB Curation Skill

Route on the first argument:

## `/curate status`

Show current KB health dashboard:
```bash
go run ./cmd/lint . 2>&1
echo "---"
git log --oneline -5
echo "---"
cat .sensor-state.json 2>/dev/null || echo "No sensor state yet"
```

Report: entity count, role types, lint findings, recent commits, last sensor poll.

## `/curate ingest <source>`

Guided ingest of a source into winze. The `<source>` argument is a URL,
DOI, Wikipedia article name, or file path.

Steps:
1. Fetch/read the source
2. Identify extractable entities and claims
3. Check schema fitness (do existing predicates suffice?)
4. Write the .go file with Provenance, entities, claims
5. Run quality gates: `go build ./...`, `go vet ./...`, `go run ./cmd/lint .`
6. Report findings and ask user to review before committing

Follow mirror-source-commitments: only encode what the source commits to.

## `/curate audit [entity]`

Audit a specific entity's neighborhood, or the whole KB if no entity given.

For a specific entity:
1. Find all claims where entity is Subject or Object
2. Check Brief quality (is it substantive?)
3. Check cross-file bridges (does this entity connect to other ingests?)
4. Identify thin spots (claims that could benefit from additional sources)

For whole KB:
```bash
go run ./cmd/lint .
go run ./cmd/lint . --llm --llm-max-calls=10
```

## `/curate predict`

Record a prediction about the KB or external signal. Prompts for:
- What is being predicted (natural language)
- Credence (0-1)
- Resolution date
- What signal would resolve it

Currently records as a comment in BOOTSTRAP.md pending the Prediction
type addition to predicates.go.

## `/curate resolve`

Check pending predictions against current state. Review any predictions
whose resolution date has passed or whose signal has arrived.

## `/curate sensor`

Run the Semantic Scholar sensor:
```bash
go run ./cmd/sensor --dir . --json
```

Show new papers found. For each paper matching a winze entity or concept,
suggest whether it's worth ingesting.
