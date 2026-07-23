# Pragma annotations

Pragma comments control lint behavior for specific declarations. They are placed
as line comments on the var declaration, not on the type.

- `//winze:contested` — marks a TheoryOf claim as expected to have competing
  theories. The contested-concept lint rule counts these to identify concepts
  with thin contestation (only 2 competing theories vs 3+). Apply to TheoryOf
  claims where the concept is genuinely contested in the literature.
- `//winze:functional` — marks a predicate as functional (each Subject has at
  most one Object). The value-conflict lint rule flags multiple functional
  claims with the same Subject as a potential contradiction. Apply to predicates
  like FormedAt, EnergyEstimate, ResolvedAs where uniqueness is expected.
- `//winze:mentions Target1,Target2` — on an ENTITY var declaration, marks the
  named target entities as contextual mentions in that entity's Brief, exempting
  them from brief-drift. Use when the Brief names an entity for context, not to
  assert a relationship the claim graph should encode. Accepted on the spec's
  doc comment, its trailing line comment, or (for a single-spec `var x = …`) the
  declaration's doc comment.
