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

// -----------------------------------------------------------------------------
// Core record types. These live here rather than in bootstrap.go so that a new
// store can copy the schema files alone: bootstrap.go is winze's own founding
// content, and a fresh store should inherit the types without the entities.
// -----------------------------------------------------------------------------

// Entity is a named thing the KB tracks. Kind is an open string; current
// values include "tool", "project", "concept", "paper", "person",
// "character", "place", "organization", "event", "instrument". Aliases
// holds surface-form variants an ingest worker resolved to this entity.
type Entity struct {
	ID      string
	Name    string
	Kind    string
	Brief   string
	Aliases []string
}

// Decision is a locked-in architectural choice. Decisions are load-bearing:
// the rest of the KB assumes they hold. Changing one is a branch-level event.
//
// This is winze's own fixed-schema architecture record (used in
// corpus/bootstrap.go), not the general decision-log mechanism every winze
// store gets for free. A store recording its own project decisions — the
// germline/aipotluck.org shape — should use a Concept memory plus a
// Supersedes chain instead; see docs/decisions.md. The two are named alike on
// purpose (both answer "why is it this way") but serve different corpora:
// this one only ever describes winze's own build.
type Decision struct {
	ID        string
	Title     string
	Rationale string
}

// FailureMode is a known way the architecture can break. Severity is 1 (low)
// to 3 (dealbreaker).
type FailureMode struct {
	ID          string
	Title       string
	Severity    int
	Description string
}

// Mitigation is a defense against a FailureMode. Automated names the intended
// mechanism, not deployment status: true means the mitigation is designed to
// run as a deterministic lint rule, false means a procedural discipline
// followed by hand. Whether an Automated mitigation has actually shipped is
// stated in its own Rule text (see mit-naming-oracle's "Implemented in
// cmd/lint") or, where it hasn't yet, noted there directly.
type Mitigation struct {
	ID        string
	Addresses *FailureMode
	Rule      string
	Automated bool
}

// OpenQuestion is something not yet resolved. Blocking questions should be
// answered before implementation work depends on their outcome. When resolved,
// set Resolution to describe the outcome and Blocking to false.
type OpenQuestion struct {
	ID         string
	Title      string
	Blocking   bool
	Resolution string // empty = still open; non-empty = resolved
}

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
// the doc→code form of winze's typed-citation primitive.
//
// Two independent axes, not every combination valid:
//
//   - WHERE the target lives: Client == "" (the only shape before this pair of
//     fields existed) means the citation targets this store's own module —
//     Symbol, if set, is the compile-checked existence proof, and `go build`
//     type-checks it: rename or remove the symbol and the corpus stops
//     compiling. Client != "" names an external client repo (resolved by
//     cmd/lint's --clients/--clients-file, see docs/lint-rules.md) that this
//     store's go.mod deliberately does NOT require — many-repos-to-one-store
//     is the documented norm (docs/agent.md), so requiring every client would
//     be exactly the fragmentation that norm avoids. Symbol MUST be nil
//     whenever Client != "", even if the target happens to be vendored into
//     this module incidentally — a citation is either "this store's own
//     module, compiler-checked" or "an external client, lint-checked," never
//     ambiguously either. cmd/lint's coderef-mutual-exclusion rule enforces
//     this as a hard failure.
//
//   - HOW existence is checked: Span == nil means existence-checked — a
//     same-module Symbol (checked by `go build`), or, when Client != "", a
//     package-qualified Go symbol name in Path (e.g.
//     "github.com/user/otherproject/internal/foo.Bar") resolved against the
//     named client with golang.org/x/tools/go/packages (cmd/lint's
//     coderef-existence rule) — deliberately NOT hash-based, since go/packages
//     correctly ignores harmless reformatting/comment changes a hash would
//     false-positive on. Span != nil means content-hash-checked: the ONLY
//     mechanism for a non-Go target (a C symbol, e.g. Path
//     "src/nvim/register.c" with Span.Line 713) or any citation where
//     byte-exact content, not mere existence, is the asserted property.
//     cmd/lint's coderef-span rule reads Path in the named client's checkout
//     (or, when Client == "", relative to the store's own repo), hashes the
//     exact text at Span.Line, and compares to Span.Hash.
//
// A third combination — Client != "" && Span == nil — is valid (an
// existence-checked external citation) but is not checked by any rule until
// cmd/lint's coderef-existence rule ships; don't mistake an unimplemented
// check for a passing one.
//
// This is a flat struct with a lint-enforced exclusion, not a sum type like
// Provenance/Conjecture, which the compiler makes mutually exclusive by
// construction. Deliberate: a real sum type here would need the AST-only
// internal/corpusparse mirror to do actual Go type resolution, which it
// doesn't and shouldn't start doing for this, and it would force every
// existing CodeRef literal to change shape.
//
// Path is the human-readable label for rendering and query, and doubles as
// the machine-checked locator: a same-module dotted symbol name
// ("internal/corpuslock.Acquire"), a client-scoped dotted symbol name when
// Client != "" && Span == nil, or a client-relative file path
// ("src/nvim/register.c") when Span != nil — the line itself lives in
// Span.Line, not appended to Path.
type CodeRef struct {
	Symbol any       // the real code symbol — compile-checked existence. Must be nil when Client != "".
	Client string    // "" = same-module citation (unchanged); non-empty names an external client checked by cmd/lint --clients
	Path   string    // human label / locator — see doc above for the three shapes
	Span   *CodeSpan // nil = existence-checked (Symbol or go/packages); non-nil = content-hash-checked (the only mechanism for non-Go targets)
	Note   string    // what the citing entity asserts about this symbol
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

// CodeSpan pins a single line of text in a client's file so a citation that
// cannot be checked by the Go compiler (a non-Go target, or a Go symbol in a
// different module) can still detect drift — by content, not by symbol
// existence. Hash is the sha256 (hex-encoded) of the exact line's text at
// citation time; cmd/lint's coderef-span rule re-hashes the live file's line
// and flags a mismatch, a missing file, or a line number past EOF.
//
// Single-line only: no forcing-function case yet for a range citation (see
// this project's own schema-accretion discipline), so no Range field exists
// until one shows up.
type CodeSpan struct {
	Line int    // 1-indexed line number in the client's file at Path
	Hash string // sha256 (hex) of that exact line's text, captured at citation time
}
