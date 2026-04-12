---
name: New predicate request
about: Propose a new relationship type
title: "[predicate] "
labels: schema
---

**Three examples** of this relationship pattern from different sources:

1.
2.
3.

**Existing predicates considered** and why they don't fit:


**Proposed type signature**
```go
type NewPredicate BinaryRelation[SubjectType, ObjectType]
```

**Contested or functional?**
- [ ] `//winze:contested` — multiple subjects per object expected
- [ ] `//winze:functional` — only one value per subject is correct

