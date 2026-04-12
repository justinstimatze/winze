package winze

// B-shape role types: authorial, design, and meta-claim atoms.
//
// The A-shape roles in roles.go (Person, Place, Event, etc.) fit concrete
// entity-relation prose. The B-shape roles fit design-doc prose where
// "claims" are rules, commitments, and structural assertions about a
// creative work rather than facts about things in the world.
//
// A and B share the Entity atom and Provenance/TemporalMarker substrate
// but operate under disjoint predicate families — a Policy cannot WorksFor
// an Organization, and a Person cannot be Prohibited. The compiler enforces
// the split for free because the slot types differ.

// CreativeWork is a published or authored artifact (game, book, film, etc.).
// Schema.org CreativeWork is the external anchor (see external.go).
type CreativeWork struct{ *Entity }

// DesignLayer is an interpretive layer of a creative work (e.g., Layer 1
// / Layer 2 / Layer 3 of a game design document).
type DesignLayer struct{ *Entity }

// Phase is a temporally- or arc-scoped production stage of a creative work
// (e.g., Phase A, Phase D, "Phase G aftermath").
type Phase struct{ *Entity }

// ProtectedLine is a specific authored string that is load-bearing across
// multiple interpretive layers and must not be edited. It carries its own
// source location (file path, line number) distinct from Provenance,
// because Provenance points at the reference-doc the claim was extracted
// from, while SourceRef points at the creative-work file the line lives in.
type ProtectedLine struct {
	*Entity
	SourceRef string // e.g. "engine.py:1076"
	Text      string
}

// NeverAnswered marks a question the creative work commits to never
// resolving. The Question field is the human phrasing; the entity ID is
// the stable handle.
type NeverAnswered struct {
	*Entity
	Question string
}

// AuthorialPolicy is a rule applied at authoring time that rejects or
// revises content matching a condition. Action is one of "cut", "revise",
// "reconsider", "consider-adding". Rationale is the doc's stated reason.
type AuthorialPolicy struct {
	*Entity
	Action    string
	Rationale string
}

// Reading is a reification: a specific interpretation of a ProtectedLine
// at a given DesignLayer or Phase. Reified because readings are 3-ary
// (line × context × text) and BinaryRelation only holds 2 slots.
type Reading struct {
	*Entity
	Text string
}
