package winze

// Starter predicates. Each predicate is a named distinct type over a
// BinaryRelation instantiation, so defn registers it as a first-class
// type and //winze:disjoint pragmas can bind type pairs for deterministic
// contradiction lint.
//
// Slot types are role wrappers (see roles.go), not raw *Entity. This is
// what catches ontology misuse at compile time: a WorksFor with a Place
// or Event in the Subject slot will not build.

// WorksFor: a person is employed by an organization.
type WorksFor BinaryRelation[Person, Organization]

// HoldsContractWith: one organization holds a contract from another.
type HoldsContractWith BinaryRelation[Organization, Organization]

// LocatedNear: one place is geographically near another. v0 conflates
// "near" with any spatial adjacency; a HasDistance predicate with a unit
// is the likely refine when a claim demands precision.
type LocatedNear BinaryRelation[Place, Place]

// MonitoredBy: a place is monitored (environmentally, by sampling) by a
// person acting as agent. MonitoredByOrg is the sibling for
// organization-as-agent. The split reflects Go's lack of sum types;
// widening to an Agent interface is a future refactor that will collapse
// both predicates once enough claims demand it.
type MonitoredBy BinaryRelation[Place, Person]

// MonitoredByOrg: a place is monitored by an organization (community
// monitoring, government monitoring, etc.).
type MonitoredByOrg BinaryRelation[Place, Organization]

// ShipsSamplesTo: a person routinely ships physical samples to an
// organization (field monitoring → lab).
type ShipsSamplesTo BinaryRelation[Person, Organization]

// Contaminates: an event deposited a polluting substance or influence
// into a place. v0 slot shape is Event→Place; substance involvement is
// captured by a sibling Released claim that attaches the substance.
type Contaminates BinaryRelation[Event, Place]

// Operates: a person routinely handles an instrument. Institutional
// operation of a facility is RunsFacility, not Operates — the asymmetry
// is deliberate because the English verb spans two different relation
// shapes that would otherwise blur.
type Operates BinaryRelation[Person, Instrument]

// RunsFacility: an organization is the operator of a facility.
type RunsFacility BinaryRelation[Organization, Facility]

// CausedEvent: a facility (or another event) is the origin of an event.
// Subject is Facility for the motivating Church Rock case; widen to a
// sum-typed origin slot when a non-facility origin shows up.
type CausedEvent BinaryRelation[Facility, Event]

// Released: an event released a substance into the environment.
type Released BinaryRelation[Event, Substance]

// Proposes: a person advances a hypothesis. Attribution relation — winze
// does not endorse the hypothesis, only records who put it forward.
type Proposes BinaryRelation[Person, Hypothesis]

// Disputes: a person argues against a hypothesis. Live scientific
// disagreement is a first-class citizen; the asymmetry with Proposes is
// deliberate so disputes do not collapse into "the rival hypothesis was
// proposed" — a disputant need not have a replacement.
type Disputes BinaryRelation[Person, Hypothesis]

// HypothesisExplains: a hypothesis is advanced as the explanation for an
// event. The Object slot is Event rather than a wider "phenomenon" role
// because the Tunguska ingest only needed events; widen when a hypothesis
// turns out to be about a Place or a Substance.
type HypothesisExplains BinaryRelation[Hypothesis, Event]

// LocatedIn: geographic containment (Vanavara in Siberia). Distinct from
// LocatedNear, which is adjacency. The compiler cannot enforce the
// containment reading, but separating the predicates lets a future lint
// rule catch LocatedIn cycles — a real containment graph is a forest.
type LocatedIn BinaryRelation[Place, Place]

// OccurredAt: an event's spatial anchor.
type OccurredAt BinaryRelation[Event, Place]

// LedExpedition: a person led another event (field expedition, mission).
// The Object slot is Event so field expeditions can be first-class
// entities with their own dates, sites, and claims, rather than being
// buried inside free-text on Kulik's Person entity.
type LedExpedition BinaryRelation[Person, Event]

// InvestigatedBy: an event was the object of scientific investigation
// by a person. Mirror of a LedExpedition that is about the event.
type InvestigatedBy BinaryRelation[Event, Person]

// ProposesOrg: an organization collectively proposes a hypothesis
// (research team, institutional group). The split from Proposes mirrors
// MonitoredBy vs MonitoredByOrg — both exist because Go has no sum type
// to cover "agent", and Tunguska's cause debate has organization-level
// attributions ("a 2020 Russian team", "the Bologna group") that do not
// cleanly nominate a single author.
type ProposesOrg BinaryRelation[Organization, Hypothesis]

// DisputesOrg: an organization collectively rejects a hypothesis.
type DisputesOrg BinaryRelation[Organization, Hypothesis]

// Accepts: a person accepts a hypothesis without having originated it.
// Weaker than Proposes — records agreement, not authorship. The source
// must use language like "accepted" or "agreed with." Earned by five
// instances in the hard-problem ingest where the source says subjects
// "accepted" the thesis.
type Accepts BinaryRelation[Person, Hypothesis]

// AcceptsOrg: organization variant (Go sum-type workaround).
type AcceptsOrg BinaryRelation[Organization, Hypothesis]

// EarlyFormulationOf: a person advanced an earlier version of an argument
// that contributed to a hypothesis, without having proposed the hypothesis
// itself. Earned by 3 instances where Kirk (1974), Campbell (1970), and
// Nagel (1970s) preceded Chalmers' 1994 formulation.
type EarlyFormulationOf BinaryRelation[Person, Hypothesis]

// FundedBy: an event (typically a field expedition or research project)
// was financially supported by an organization.
type FundedBy BinaryRelation[Event, Organization]

// AffiliatedWith: a person has an institutional affiliation with an
// organization — academic appointment, fellowship, long-term membership.
// Distinct from WorksFor, which is ordinary employment; the compiler
// cannot enforce the distinction, but the split keeps query intent clear
// (academic affiliation graphs and employment graphs tend to be queried
// by different consumers).
type AffiliatedWith BinaryRelation[Person, Organization]

// -----------------------------------------------------------------------------
// Unary style-claim predicates — each a distinct type so the predicate
// type NAME is the content. No value field on UnaryClaim by design;
// adding one would muddy the "predicate as type" spirit. A future slice
// can either accrete more style-claim types or add a structured value
// field if the content starts needing structure.
//
// ---- Taxonomy-tag UnaryClaim discipline (named pattern) ----
//
// Across early ingests, a specific sub-shape of the unary style-claim
// idea accreted to the point of being worth naming explicitly in
// predicates.go rather than in inline slice comments. Call it the
// **taxonomy-tag UnaryClaim pattern**:
//
//     type IsX UnaryClaim[Role]
//
// where X is the taxonomic category and Role is the permissive base
// role the claim wraps (almost always Concept, occasionally
// Hypothesis). The pattern is validated by five predicates and 20+
// claims across four slices:
//
//   - IsPolyvalentTerm             (nondualism.go, forecasting.go)
//   - IsCognitiveBias              (cognitive_biases.go, apophenia.go — first cross-file use)
//   - IsFictionalWork              (quantum_thief.go: 4 books)
//   - IsFictional                  (quantum_thief.go: 6 in-fiction concepts)
//   - CorrectsCommonMisconception  (misconceptions.go — wraps Hypothesis instead of Concept)
//
// When the pattern is the right call:
//
//   1. The claim is an unconditional fact about a concept's category
//      membership in a named family, not a relation to another
//      concept. A relation would be BelongsTo instead.
//   2. The family is not a Role in the naming-oracle sense — no
//      forcing case has demanded a dedicated role type, and creating
//      one would multiply entities without new queries.
//   3. The set of predicates that can meaningfully apply to tagged
//      subjects is unbounded — a Concept role is permissive enough
//      that future ingests can attach any predicate shape without
//      re-splitting the role hierarchy.
//   4. The predicate name is **the entire definition**. No value
//      field, no parameters, no attributes. If the tag needs a
//      value (like EnergyEstimate or EnglishTranslationOf) it is
//      not a taxonomy tag, it is a functional binary relation.
//
// When the pattern is the WRONG call:
//
//   - When the category membership IS a relation to another named
//     parent — use BelongsTo instead. Example: ClusteringIllusion
//     BelongsTo Apophenia, not a hypothetical IsSubtypeOfApophenia
//     tag. The rule of thumb: if the parent deserves its own entity,
//     the membership deserves a BelongsTo edge.
//
//   - When the tag's presence implies a role split is actually earned.
//     If every tagged subject ends up only appearing as Subject or
//     Object in predicates that no untagged subject ever appears in,
//     the tag is secretly a Role and the discipline win is to
//     promote it. None of the current five have crossed that line.
//
//   - When the tag is actually a value that wants attribution (who
//     says so? when?). Provenance on the claim covers "who says
//     so" at the claim level, but a structured value field would
//     cross the line from style-claim into typed quantity — use
//     a functional BinaryRelation with a value struct in the
//     Object slot (see EnergyEstimate / EnglishTranslationOf).
//
// The naming convention is `IsX` or `CorrectsY` — present-tense
// verbless tags that read as assertions about the subject. The
// slot is almost always Concept; Hypothesis appears once
// (CorrectsCommonMisconception) because the misconception slice's
// unit of correction is a hypothesis-as-fact, not a concept-as-
// category-member. Future slices that want Place, Event, or Person
// tags should stop and ask whether the tag is secretly about an
// implicit Concept the ingest is refusing to reify — in winze's
// experience so far, the answer has always been yes.
//
// This discipline block exists because the pattern crossed the
// "four occurrences worth naming explicitly" threshold called out
// in cognitive_biases.go's inline comment during early ingests, and
// the next ingest added the fifth occurrence without pushing back on
// the shape. Promoting it here closes the inline-comment → first-
// class-discipline loop before the next ingest faces the same
// role-split-vs-tag question and has to re-derive it.
// -----------------------------------------------------------------------------

// GrantsBroadAuthorityOverWinze: the subject has granted broad decision
// authority over winze direction to the assistant running it.
type GrantsBroadAuthorityOverWinze UnaryClaim[Person]

// PrefersTerseResponses: the subject prefers short, direct responses
// from assistants without trailing summaries of diffs they can read.
type PrefersTerseResponses UnaryClaim[Person]

// PushesBackOnOverengineering: the subject immediately pushes back on
// ceremony, premature abstraction, or pre-planned vocabulary families.
type PushesBackOnOverengineering UnaryClaim[Person]

// PrefersOrganicSchemaGrowth: the subject prefers that winze's predicate
// vocabulary accrete strictly per ingest need rather than be designed
// as predicate families in anticipation of future content.
type PrefersOrganicSchemaGrowth UnaryClaim[Person]

// IsPolyvalentTerm: the subject concept has multiple incompatible
// technical meanings across traditions or authors and no single
// canonical sense. Added for the Nondualism ingest, where the source
// article opens with the declaration that "nondualism" is a
// polyvalent term — the first time winze encodes a concept's
// contested-meaning status as a claim rather than leaving it as an
// untyped property of the Brief.
type IsPolyvalentTerm UnaryClaim[Concept]

// IsCognitiveBias: the subject concept is a named cognitive bias — a
// systematic pattern of deviation from norm and/or rationality in
// judgment. Added for the Wikipedia List of cognitive biases ingest
// so ingest workers can query "what biases does winze know about"
// with a one-line claim instead of a role-type lookup. Parallel
// shape to IsPolyvalentTerm: UnaryClaim[Concept] with name-is-content
// discipline. Uses the permissive Concept role rather than a
// dedicated CognitivePhenomenon role because biases are one of many
// possible named-cognitive-pattern categories and no forcing case
// has demanded a role split.
type IsCognitiveBias UnaryClaim[Concept]

// BelongsTo: the subject concept is a member of a parent concept
// representing a family, category, or grouping. Added for the
// cognitive-biases ingest where each individual bias belongs to
// one of six task-based families (Estimation, Decision, etc.) per
// the Dimara et al. 2020 classification. Not the same as
// DerivedFrom, which tracks etymological/technical lineage; this
// predicate tracks taxonomic membership. Not functional — a
// concept can legitimately belong to multiple overlapping
// families (a bias might operate in both Estimation and Decision
// tasks, or a term might be a member of both a domain-specific
// and a general-purpose family). Cycle detection is a candidate
// for a future lint rule but isn't forced by this ingest — for
// now the orphan-report and contested-concept rules are enough.
type BelongsTo BinaryRelation[Concept, Concept]

// InfluencedBy: the subject person's work draws on the object person as
// an intellectual or artistic influence. Added for the predictive
// processing ingest where Rajaniemi's own acknowledgments in the Fractal
// Prince cite Andy Clark (on the 'mind as self-loop' idea) and
// Douglas Hofstadter; backfilled into the Quantum Thief slice for
// Maurice Leblanc, on whose Arsène Lupin the protagonist Jean le
// Flambeur is explicitly modelled. Person→Person by design: work-level
// influences (Hofstadter's I Am a Strange Loop, Yates's Art of Memory,
// the Arabian Nights) are routed through their author to avoid a
// second Person-vs-Concept slot split that no ingest has forced yet.
// Not functional — a working author can cite arbitrarily many
// influences and they're all simultaneously true. The predicate is
// the first real cross-ingest bridge in winze: the same Rajaniemi
// entity that carries InfluencedBy{Rajaniemi, Leblanc} also carries
// InfluencedBy{Rajaniemi, Clark}, and the Clark target is the same
// Person entity defined in the predictive processing slice — proving
// that the non-executable discipline lets separate ingests meet at a
// shared entity without either slice having to know about the other.
type InfluencedBy BinaryRelation[Person, Person]

// IsFictionalWork: the subject concept is a creative work of fiction —
// a novel, short story, film, game, or other authored fictional artefact.
// Added for the Quantum Thief ingest as the "real-world tag" half of
// the in-fiction/real-world split. The work itself exists in the real
// world (Rajaniemi really wrote it, Gollancz really published it), so
// it is NOT IsFictional — that tag is reserved for the characters,
// places, and concepts that exist only within the work's frame. The
// two tags are deliberately orthogonal: IsFictionalWork marks "this
// is a work of fiction"; IsFictional marks "this is a thing that only
// exists inside a work of fiction". A concept can legitimately carry
// either, both (a fictional book within a fictional book), or neither.
type IsFictionalWork UnaryClaim[Concept]

// IsFictional: the subject concept exists only within the frame of
// some fictional work. Any claim about the subject should be read as
// "...within the fiction" rather than "...in the real world". Added
// for the Quantum Thief ingest to keep in-world facts (Jean le
// Flambeur is a thief, the Oubliette is a Martian city) out of the
// same query space as real-world facts without introducing a
// dedicated FictionalEntity role. The tag pattern mirrors the
// IsCognitiveBias and IsPolyvalentTerm precedent: a unary claim on
// Concept rather than a role split, because the set of real-world
// predicates that could meaningfully apply to a fictional entity is
// unbounded and no single role wrapping would capture it.
type IsFictional UnaryClaim[Concept]

// AppearsIn: the subject concept appears in the object fictional
// work. Not functional — a character or invented vocabulary item can
// appear in multiple books (Jean le Flambeur appears in all three
// novels of the trilogy), and a second slice can wire those
// additional appearances without any schema change. The Object slot
// is Concept rather than a dedicated FictionalWork role for the same
// reason IsFictional uses Concept: a role split has not been forced
// yet. The lint rule that would catch "AppearsIn whose Object is not
// tagged IsFictionalWork" is a natural future addition once the
// pragma vocabulary has a "tag-implies-tag" primitive, but is not
// forced by this slice.
type AppearsIn BinaryRelation[Concept, Concept]

// CommentaryOn: one creative work is a scholarly commentary on another.
// Added for the White & Shergill 2012 "Using Illusions to Understand
// Delusions" ingest (white_shergill_commentary.go), which is explicitly
// a Frontiers in Psychology commentary on Clark 2013 "Whatever next?".
// Paper-to-paper by design: the relationship is between the commentary
// artifact and its target artifact, not between their authors or their
// theses. Could have been collapsed into InfluencedBy at the author
// level, but that loses the structural fact that this specific paper
// responds to that specific paper — and once the reader wants to ask
// "what commentaries does winze know about?" the author-level
// collapse has no predicate to answer with. First schema-forcing
// ingest in five slices; breaks the vocabulary-fit streak exactly at
// the point a new corpus shape (peer-reviewed commentary) earns it.
// Not functional: a single commentary can address multiple target
// papers, and a single target paper attracts many commentaries.
//
// The Subject slot is the commentary work and the Object slot is the
// target work — the English verb "A comments on B" orders the slots
// the same way. Both slots use Concept because papers-as-entities are
// represented via the same Concept-with-IsScientificPaper-tag pattern
// that books use with IsFictionalWork (see quantum_thief.go). A
// dedicated Paper role has not been forced yet and would double the
// graph for a single-slice savings.
type CommentaryOn BinaryRelation[Concept, Concept]

// Authored: a person authored a creative work. Distinct from Proposes
// because authorship of a novel is not the advancement of a factual
// claim — a hypothesis is asserted to be true of the world, whereas a
// novel is constructed and need not correspond to anything. The
// split also mirrors Proposes/ProposesOrg: the UDHR slice earned
// AuthoredOrg as the institutional-authorship sibling when a
// committee-drafted document (the UN Commission on Human Rights'
// drafting of the Universal Declaration of Human Rights) made
// per-person Authored claims semantically wrong — attributing the
// declaration to Roosevelt, Cassin, Humphrey, Chang, Malik, or
// Mehta individually would erase the institutional character of
// the act, and attributing it to nobody would drop the authorship
// claim entirely.
type Authored BinaryRelation[Person, Concept]

// AuthoredOrg: an organization collectively authored a creative work
// or document. The institutional-authorship sibling of Authored,
// earned by the UDHR 1948 slice (udhr.go) as the first forcing case
// where committee authorship was load-bearing and per-person
// attribution would have been wrong. Parallel split to
// Proposes/ProposesOrg and MonitoredBy/MonitoredByOrg — the Go
// lack-of-sum-types work-around that has paid off three times
// running. Non-functional: a document can plausibly be co-authored
// by multiple organizations (the UDHR itself was drafted by the UN
// Commission on Human Rights and adopted by the General Assembly,
// which are two different organizations with different formal
// roles in the document's history; this slice uses AuthoredOrg for
// the drafting role and ProposesOrg on the article-level
// Hypotheses for the adoption role, keeping the two speech acts
// structurally distinguished).
type AuthoredOrg BinaryRelation[Organization, Concept]

// CorrectsCommonMisconception: the subject hypothesis is the factual
// correction of a widely-held false belief about the same topic.
// Added for the Wikipedia List of common misconceptions ingest, whose
// source article explicitly states that "each entry is worded as a
// correction; the misconceptions themselves are implied rather than
// stated." Winze follows the source's discipline: the corrected fact
// is the Hypothesis, its name IS the correction, and the existence
// of a common misbelief is encoded only as this unary tag. A future
// ingest that wants structured access to the *content* of common
// misbeliefs (rather than just their existence) will need to
// introduce a separate false-belief representation — but no such
// ingest has forced it yet, and inventing one now would fabricate
// content the source does not provide.
type CorrectsCommonMisconception UnaryClaim[Hypothesis]

// TheoryOf: a hypothesis is a theory *of* a concept — a typology, a
// meta-claim about definition, a proposed partition, or any structural
// account that takes the concept itself as its subject. Parallel shape
// to HypothesisExplains[Hypothesis, Event], split for the same
// Go-lacks-sum-types reason as MonitoredBy/MonitoredByOrg. The
// Nondualism ingest attaches four distinct theories-of-nondualism
// here: Murti's advaita/advaya binary, Loy's five flavors, Volker's
// three types, and the perennialist common-core thesis; a future
// ingest can promote widely-adopted theories out of Hypothesis into
// a dedicated role if one shows up enough.
//
// The //winze:contested pragma tells the contested-concept lint rule
// to group TheoryOf claims by Object (the concept being theorised
// about) and emit an advisory report whenever two or more distinct
// subject Hypothesis entities point at the same concept. Unlike
// //winze:functional, contested is not a failure condition: multiple
// theories of the same concept are a normal state of affairs in
// philosophy and in any field with live intellectual disagreement.
// The rule's job is to surface the landscape of disagreement, not to
// demand its resolution.
//
//winze:contested
type TheoryOf BinaryRelation[Hypothesis, Concept]

// DerivedFrom: one concept is etymologically or technically derived
// from another. Not functional — a concept can descend from multiple
// roots (English "nondualism" is derived from both Sanskrit advaita
// and Sanskrit advaya, which are distinct terms with distinct
// technical histories). The relation is intentionally about concept
// lineage rather than about translation.
type DerivedFrom BinaryRelation[Concept, Concept]

// EnglishRendering is a value-with-attribution object for claims
// about how a non-English concept should be rendered in English.
// Non-entity struct (no *Entity embed), mirroring TemporalMarker and
// EnergyReading, so it lives outside the naming-oracle's role-type
// world. The attribution is not optional because the dispute is
// specifically about which translator/tradition's rendering is
// canonical — a Value without a By is worthless for this predicate.
type EnglishRendering struct {
	Value string // e.g. "Monism", "nondualism", "that which has no second beside it"
	By    string // short translator/tradition tag, e.g. "Max Müller 1879"
}

// EnglishTranslationOf: the canonical English rendering of a
// non-English source concept. Functional: winze asserts that a
// source concept has at most one "right" English rendering, which
// is exactly the claim Müller makes (advaita = Monism) and that
// Watts and some subsequent scholars explicitly reject. Three
// rival renderings coexist in the literature, so this becomes the
// third `//winze:functional` predicate after FormedAt and
// EnergyEstimate, and stresses the value-conflict rule on a new
// predicate shape (Concept, not Place or Event, as subject). The
// dispute is recorded at both levels of winze: the translator-
// proposal layer (Müller proposes an `AdvaitaAsMonismTranslation`
// Hypothesis via Proposes) and the value layer (three rival
// `EnglishTranslationOf` claims annotated by a KnownDispute).
// Having both layers is deliberate — the Hypothesis layer carries
// attribution + dispute structure, the functional layer carries
// the rival values themselves, and each answers questions the
// other cannot.
//
//winze:functional
type EnglishTranslationOf BinaryRelation[Concept, *EnglishRendering]

// FormedAt: the formation date of a place, encoded as a TemporalMarker
// value in the Object slot. Distinct from the claim's own When field,
// which records when the *claim* was made; here the Object is the value
// being asserted about the subject. A place has at most one formation
// date, so FormedAt is functional — two claims with the same subject
// and different objects are a value conflict, not a legitimate
// many-relation. The pragma below tells the value-conflict lint rule
// to flag exactly that pattern.
//
//winze:functional
type FormedAt BinaryRelation[Place, *TemporalMarker]

// EnergyReading is a value-with-attribution object for explosive-energy
// claims. Kept as a non-entity struct (no *Entity embed) so it lives
// outside the naming-oracle's role-type world, mirroring TemporalMarker.
// Value is a free-text range because winze v0 does not model units or
// intervals; a future refinement can promote this to a typed quantity
// when a second energy-estimate ingest demands it.
type EnergyReading struct {
	Value string // "~10-15 Mt TNT", "~3-5 Mt TNT", etc.
	By    string // short attribution tag, e.g. "Ben-Menahem 1975 seismic"
}

// EnergyEstimate: the released energy of an event, as a reading value.
// Functional: an event has one true energy, so rival readings are a
// value conflict to be flagged unless annotated as a KnownDispute. The
// second functional predicate in winze after FormedAt, added to stress
// the value-conflict rule with a three-way (rather than two-way) live
// scientific disagreement.
//
//winze:functional
type EnergyEstimate BinaryRelation[Event, *EnergyReading]

// -----------------------------------------------------------------------------
// Prediction and calibration predicates.
//
// Earned by the README roadmap item, with the deferred-schema surface
// from forecasting.go providing the design. The forecasting
// ingest established the conceptual vocabulary (Tetlock's calibration
// framing, forecasting-as-concept); these predicates are the structural
// machinery for encoding specific dated predictions and tracking their
// resolution over time.
//
// The prediction loop: Hypothesis --Predicts--> Event (future observable),
// with a Credence value attaching a probability and a Resolution value
// recording the ground-truth outcome once time passes.
// -----------------------------------------------------------------------------

// Predicts: a hypothesis generates a testable prediction about an
// observable event. The Subject is a Hypothesis (not a Person) because
// attribution is already handled by Proposes — a person proposes a
// hypothesis, and the hypothesis predicts an event. This keeps the
// prediction graph structural rather than personal, paralleling
// HypothesisExplains[Hypothesis, Event] for past events.
//
// Not functional: a single hypothesis can generate multiple distinct
// testable predictions about different events. For example, a climate
// hypothesis might predict both sea-level rise events and temperature
// threshold events independently.
type Predicts BinaryRelation[Hypothesis, Event]

// CredenceLevel is a value-with-attribution object for probability
// assignments on hypotheses. Parallel to EnergyReading and
// EnglishRendering: a non-entity struct that lives outside the
// naming-oracle's role-type world. The By field is not optional
// because the whole point of credence tracking is calibration per
// forecaster — a credence without attribution cannot be scored.
type CredenceLevel struct {
	Value string // free-text probability, e.g. "0.70", "70%", "likely (>0.6)"
	By    string // who assigned this credence, e.g. "Tetlock 2015", "winze 2026-04-13"
}

// Credence: a probability assignment on a hypothesis. NOT functional —
// different forecasters legitimately assign different credences to the
// same hypothesis, and the calibration loop needs all of them to score
// each forecaster's accuracy. This is the deliberate opposite of
// EnergyEstimate (functional: one true energy) — there is no single
// "true" credence, only well-calibrated or poorly-calibrated ones.
type Credence BinaryRelation[Hypothesis, *CredenceLevel]

// ResolutionOutcome records the ground-truth result of a prediction
// once the predicted event's time horizon has passed. Result values:
//
//   confirmed  — prediction confirmed (evidence found, whether corroborating or challenging)
//   refuted    — prediction refuted (no signal: no papers found at all)
//   ambiguous  — inconclusive (papers found but irrelevant — sensor miscalibration, not prediction failure)
//
// The meta-prediction is "structural fragility predicts findable external
// evidence." Both corroborated and challenged metabolism resolutions confirm
// this prediction. Irrelevant results are ambiguous (the sensor, not the
// prediction, may be at fault). Only no_signal genuinely refutes.
type ResolutionOutcome struct {
	Result   string // "confirmed", "refuted", "ambiguous"
	Evidence string // source text establishing the outcome
}

// ResolvedAs: the ground-truth outcome of a hypothesis's prediction.
// Subject is the evidence-search Event (unique per hypothesis), Object
// is the ResolutionOutcome. Functional: each evidence search has exactly
// one resolution. This is the first functional predicate whose value is
// temporally gated — FormedAt, EnergyEstimate, and EnglishTranslationOf
// are atemporal (the true value just IS), whereas a ResolvedAs value only
// becomes assertable once the prediction's time horizon passes.
//
// Design note: the Subject is an Event (not a Hypothesis) because the
// resolution is about a specific evidence search, not the hypothesis
// itself. A hypothesis can have multiple evidence searches with different
// outcomes across different cycles. The Predicts claim (Hypothesis→Event)
// connects the resolution back to the hypothesis.
//
//winze:functional
type ResolvedAs BinaryRelation[Event, *ResolutionOutcome]
