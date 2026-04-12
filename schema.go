package winze

// Provenance is the audit trail for a claim. Source documents are transient
// in winze's workflow (the KB is the canonical representation, not a mirror
// of external files), so there is no live link to verify against. The Quote
// field holds the specific fragment the claim was extracted from — when the
// source is gone, that quote IS the audit record.
//
// Origin is a free-form human hint about where the claim came from — a
// corpus name, a URL, a conversation timestamp, a book citation. It is
// never required to resolve to a live file.
type Provenance struct {
	Origin     string // human hint: "Stope reference corpus / npc-reggie-tsosie.md", "conversation 2026-04-11"
	IngestedAt string // ISO-8601 date of ingest
	IngestedBy string // worker id or author name
	Quote      string // the specific source fragment the claim was extracted from
}

// TemporalMarker places a claim in time. v0 is deliberately coarse: Era is
// a free-string tag and Ordinal is an optional tie-breaker. The schema will
// churn here as soon as real claims want intervals, relative ordering, or
// world-time vs story-time — that churn is the signal to refine, not to
// pre-design.
type TemporalMarker struct {
	Era     string
	Ordinal int
}

// BinaryRelation is the generic base for two-slot predicates. Concrete
// predicates are named distinct types over instantiations, e.g.
//
//	type WorksFor BinaryRelation[*Entity, *Entity]
//
// so that each predicate is its own first-class type in defn's graph and
// //winze:disjoint pragmas can bind type pairs for contradiction lint.
type BinaryRelation[S, O any] struct {
	Subject S
	Object  O
	When    *TemporalMarker
	Prov    Provenance
}

// UnaryClaim is the generic base for single-slot predicates (Is, Has, etc.).
// Same discipline as BinaryRelation: named distinct types are declared at
// the call site.
type UnaryClaim[S any] struct {
	Subject S
	When    *TemporalMarker
	Prov    Provenance
}

// Scene groups claims that share a setting. Claims is []any because
// predicate types are heterogeneous; a Claim interface can replace this
// once query patterns are known.
type Scene struct {
	ID     string
	Where  *Entity
	When   *TemporalMarker
	Claims []any
}
