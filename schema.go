package winze

// QuoteMandateDate is the date after which all Provenance records must have
// a non-empty Quote. Before metabolism cycle 6 (2026-04-13), Quote was
// optional and some early provenance vars use string concatenation that
// extractStringField can't parse. The corpus_test.go enforcement boundary
// uses this constant.
const QuoteMandateDate = "2026-04-13"

// DefaultEntityCap is the default maximum entity count for the KB.
// Topology suppresses breadth sensor targets above this threshold;
// metabolism refuses ingest/pipeline above it. Depth-first: deepen
// thin contested neighborhoods before expanding.
// Override with WINZE_ENTITY_CAP env var or --entity-cap flag.
const DefaultEntityCap = 300

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
	Origin     string // human hint: "Wikipedia 2025-12 / Tunguska_event", "conversation 2026-04-11"
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

// CodeRef is a typed citation from a knowledge entity to a live code symbol —
// the doc→code form of winze's typed-citation primitive. Symbol holds the
// actual Go symbol (e.g. a function value), so `go build` type-checks the
// citation: rename or remove the symbol and the corpus stops compiling. A
// documentation reference that cannot go stale silently, because staleness is
// a build error rather than something a reviewer might notice.
//
// Path is the human-readable label for rendering and query (e.g.
// "internal/corpuslock.Acquire"); it is derivable from Symbol via
// runtime.FuncForPC — a later refinement so Path itself can't drift from
// Symbol — kept explicit for now. Only entities that document code carry
// CodeRefs; corpus concepts do not, so the field lives on SourceDoc, not on
// every Entity.
type CodeRef struct {
	Symbol any    // the real code symbol — compile-checked existence
	Path   string // human label, e.g. "internal/corpuslock.Acquire"
	Note   string // what the citing entity asserts about this symbol
}

// SourceDoc is a knowledge entity that documents part of a codebase: an
// ordinary *Entity (identity + prose Brief) plus typed code citations. Prose
// for meaning, typed references for the links — the same split the rest of the
// corpus uses for concept→concept claims, pointed at code instead. See
// winze_self.go for the corpus's self-documentation and docs/typed-citation.md
// for the thesis.
type SourceDoc struct {
	*Entity
	Refs []CodeRef
}
