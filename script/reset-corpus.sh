#!/bin/bash
set -euo pipefail

# Reset winze to a fresh KB by removing all example corpus files.
# Framework files (schema, roles, predicates, lint, sensor) are kept.
# A minimal starter.go is created to show the pattern.
#
# After running:
#   1. Edit starter.go or create new .go files with your own entities
#   2. go build ./...
#   3. go run ./cmd/lint .

cd "$(dirname "$0")/.."

FRAMEWORK_FILES=(
  schema.go
  bootstrap.go
  roles.go
  design_roles.go
  design_predicates.go
  predicates.go
  external.go
  disputes.go
)

echo "Removing example corpus files..."

for f in *.go; do
  skip=false
  for fw in "${FRAMEWORK_FILES[@]}"; do
    if [ "$f" = "$fw" ]; then
      skip=true
      break
    fi
  done
  if $skip; then
    echo "  keep: $f (framework)"
    continue
  fi
  echo "  remove: $f"
  rm "$f"
done

echo ""
echo "Creating starter.go..."

cat > starter.go << 'GOEOF'
package winze

// Starter corpus file. Replace this with your own entities and claims.
//
// Pattern:
//   1. Declare a Provenance (where the knowledge came from)
//   2. Declare entities (Person, Concept, Hypothesis, Place, Event, ...)
//   3. Declare claims (BinaryRelation or UnaryClaim linking entities)
//   4. Run: go build ./...
//   5. Run: go run ./cmd/lint .

var starterSource = Provenance{
	Origin:     "your source here (URL, book citation, paper DOI, etc.)",
	IngestedAt: "2026-01-01",
	IngestedBy: "your-name",
	Quote:      "the specific passage supporting the claim",
}

var ExamplePerson = Person{&Entity{
	ID:    "example-person",
	Name:  "Ada Lovelace",
	Kind:  "person",
	Brief: "Mathematician, first computer programmer. Wrote the first algorithm intended for machine processing.",
}}

var ExampleConcept = Concept{&Entity{
	ID:    "example-concept",
	Name:  "Algorithm",
	Kind:  "concept",
	Brief: "A finite sequence of well-defined instructions for solving a class of problems or performing a computation.",
}}

var ExampleHypothesis = Hypothesis{&Entity{
	ID:    "lovelace-computing-thesis",
	Name:  "Lovelace computing thesis",
	Kind:  "hypothesis",
	Brief: "Machines can be made to compute any calculable function given appropriate instructions.",
}}

var ExampleClaim = Proposes{
	Subject: ExamplePerson,
	Object:  ExampleHypothesis,
	Prov:    starterSource,
}
GOEOF

echo ""
echo "Done. Run 'go build ./...' to verify, then start adding your own knowledge."
