package winze

// Role types wrap *Entity with a distinct Go type so predicate slot
// signatures can enforce ontology sanity at compile time. Each role name
// is grounded in external.go's ExternalTerms — Person/Organization/Place/
// Event from Schema.org, Facility from WordNet, Substance from Wikidata —
// so the naming oracle decision (dec-no-upper-ontology) has real teeth
// from day one.
//
// These wrappers apply to corpus entities (ingested from source documents).
// winze's self-tracking entities in bootstrap.go (tools, projects, papers)
// remain plain *Entity — they describe the project itself, not claims
// from external content, and do not benefit from role typing.
//
// An entity is declared once in its primary role. If a later claim needs
// it in a different role, we cross-wrap explicitly (e.g. a Person that
// is also treated as an Organization-like agent for one claim). Forcing
// the cross-wrap to be visible at the call site is a feature: it surfaces
// ontology stretching instead of hiding it.

type Person struct{ *Entity }
type Organization struct{ *Entity }
type Place struct{ *Entity }
type Event struct{ *Entity }
type Facility struct{ *Entity }
type Substance struct{ *Entity }
type Instrument struct{ *Entity }

// Hypothesis: a scientific claim advanced about some phenomenon,
// attributed to a proposer, potentially disputed, potentially about
// something. Added for the Tunguska ingest where competing-cause
// explanations are the central structure of the corpus. A Hypothesis
// entity carries no truth commitment — it is the reified "X thinks Y"
// node that Proposes / Disputes / HypothesisExplains relations attach to.
type Hypothesis struct{ *Entity }

// Concept: a term or idea that is itself the subject of claims — as
// opposed to a concrete referent in the world. Added for the Nondualism
// ingest, where the article is literally about a "polyvalent term" and
// the load-bearing entities (Nondualism, Advaita, Advaya, Brahman,
// Śūnyatā, prapanca) are none of Person/Place/Event/... — they are
// the terms whose definitions authors argue over. The role is
// deliberately permissive: anything from a Sanskrit root to a
// contemporary philosophical position can live here. The
// disambiguation work — whose sense of "advaita"? — is not done by
// the role type but by the claims that attach to concept entities
// (in particular, typology Hypothesis entities that attribute
// distinct senses to distinct authors or traditions).
type Concept struct{ *Entity }
