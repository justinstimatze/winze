package winze

// KnownDispute marks a (subject, predicate) pair as an intentional live
// dispute in the knowledge base, rather than an ingest error. The
// value-conflict lint rule consults KnownDispute declarations and
// suppresses flagged groups that a dispute covers — the conflict is
// *recorded* at that point, not actionable.
//
// This is a meta-annotation, not an ontological entity: KnownDispute
// does NOT embed *Entity, so the naming-oracle lint skips it. A dispute
// is about the graph, not about the world the graph represents.
//
// SubjectRef is declared as `any` rather than a specific role type
// because disputes can cover subjects of any role (Place, Person,
// Event, etc.). The compiler cannot enforce that PredicateType is a
// real predicate name — it is a string for lint-time matching — but
// the naming-oracle + orphan-report rules together catch most typos
// by flagging the entire claim group as ungrounded.
//
// The Rationale field is prose for humans reading lint output. It does
// not feed back into the graph.
type KnownDispute struct {
	ID            string
	Name          string
	SubjectRef    any
	PredicateType string
	Rationale     string
}
