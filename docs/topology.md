# Topology analysis

```bash
go run ./cmd/topology .              # structural vulnerability report
go run ./cmd/topology --json .       # JSON with sensor_targets for automation
go run ./cmd/topology --export-kb .  # slimemold-compatible KBClaim JSON
go run ./cmd/topology --dot .        # epistemic support DAG as Graphviz DOT
go run ./cmd/topology --why NAME .   # trace epistemic support chain with provenance
```

Detectors: single-source, uncontested, thin-provenance, bridge-entity,
concentration-risk. Sensor targets are topology-derived queries for the
most structurally fragile hypotheses.
