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

// Attribution is the epistemic backing of a claim. Every claim is either
// SOURCED (Provenance — mirror-source-commitments, Quote holds the exact source
// fragment) or a CONJECTURE (Conjecture — winze's own generation, uncitable by
// construction). The interface is sealed (unexported method), so only these two
// can back a claim: the compiler thus guarantees every claim declares which
// kind of knowledge it is, and nothing can pose as a third, ambiguous state.
type Attribution interface {
	isAttribution()
	// Conjectural reports whether this backing is winze's own generation rather
	// than a sourced record. Sourced provenance returns false; a conjecture
	// returns true.
	Conjectural() bool
}

func (Provenance) isAttribution()    {}
func (Provenance) Conjectural() bool { return false }

// Conjecture is the honest backing for knowledge winze GENERATED rather than
// sourced: trip-cycle connections, cross-cluster analogies, synthesis. It is
// uncitable by construction — it has NO Quote field, so a generated claim can
// never wear a fabricated source attribution. That is the whole point:
// `Conjecture{Quote: "..."}` does not compile, so the failure mode where a
// speculative claim is dressed as sourced fact is closed by the type system,
// not by a lint rule someone might disable.
//
// A conjecture records its OWN honest origin instead — which winze process
// produced it, from which entities, with what generation parameters and score.
// Rationale is winze's own reasoning for the connection, explicitly not a
// source's words. A conjecture may later be PROMOTED to a Provenance if a real
// source is found, or pruned if it fails to corroborate; that lifecycle is what
// makes winze's generation a reasoning process rather than unlabelled
// invention. See docs/typed-citation.md.
type Conjecture struct {
	GeneratedBy      string    // the winze process, e.g. "metabolism-trip", "synthesis"
	From             []*Entity // the entities the conjecture was generated from
	CycleN           int       // metabolism cycle that produced it (0 if not applicable)
	Temperature      float64   // generation temperature (the drug-profile wildness axis)
	PromptType       string    // "analogy", "contradiction", "genealogy", "synthesis", ...
	Score            int       // interestingness score (>=3 interesting, >=4 promote)
	Rationale        string    // winze's OWN reasoning for the connection — never a source quote
	GeneratedAt      string    // ISO-8601 date
	GeneratedByAgent string    // worker id that ran the generation
}

func (Conjecture) isAttribution()    {}
func (Conjecture) Conjectural() bool { return true }

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
	Prov    Attribution
}

// UnaryClaim is the generic base for single-slot predicates (Is, Has, etc.).
// Same discipline as BinaryRelation: named distinct types are declared at
// the call site.
type UnaryClaim[S any] struct {
	Subject S
	When    *TemporalMarker
	Prov    Attribution
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
