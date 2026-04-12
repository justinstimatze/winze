// Package winze is a non-executable knowledge base built on Go's type system.
//
// Entities are typed constants (Person, Concept, Hypothesis, etc.), predicates
// are generic types (Proposes, TheoryOf, BelongsTo, etc.), and claims are
// variable declarations. The compiler catches ontological errors: if a claim
// references the wrong type of entity, it doesn't compile.
//
// The package produces no binary. go build is the consistency checker.
// Standard Go tooling works unchanged: LSP navigation, go/ast analysis,
// code review, git blame.
//
// Schema types are in schema.go. Role wrappers are in roles.go and
// design_roles.go. Predicates are in predicates.go and design_predicates.go.
// Corpus data lives in per-topic .go files (tunguska.go, nondualism.go, etc.).
//
// See the README for CLI tools: query, lint, topology, metabolism.
package winze
