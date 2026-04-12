package winze

// Metabolism cycle 2 ingest: corroboration and dispute claims surfaced by
// topology-driven sensor queries against arXiv and Wikipedia ZIM backends.
//
// Cycle context: topology identified 5 structurally fragile hypotheses
// (single-source AND uncontested). Metabolism ran improved queries (proposer
// names + concept phrases) against both backends. ZIM produced strong signal
// for 4 of 5 targets. This file encodes the two claims where Wikipedia
// sources explicitly commit to a corroboration or dispute relationship.
//
// Structural effect:
//   - ChalmersHardProblemThesis: was single-source (only Chalmers proposes).
//     Adding Dennett's Disputes claim means the hypothesis now has both a
//     proposer and a disputant — no longer structurally fragile.
//   - BrownHumanUniversalsThesis: was single-source (only Brown proposes).
//     Adding Pinker's Proposes claim means the hypothesis now has two
//     independent proposers — no longer structurally fragile.
//
// Targets with no actionable ingest from this cycle:
//   - BaloneyDetectionKitThesis: ZIM found The Demon-Haunted World article
//     (already ingested source). No new corroborating or disputing voice.
//   - ConstructionistThesis: ZIM found "Mystical or religious experience"
//     which confirms Katz's constructionism "became dominant during the
//     1970s" but names no second independent proposer.
//   - ConradApopheniaClinicalFraming: no signal from either backend.
//
// Schema forcing functions earned: NONE. Disputes and Proposes are both
// existing predicates. One new Person entity (DanielDennett). Pinker
// already exists in blank_slate.go.
//
// Cross-file bridges wired:
//   - DennettDisputesHardProblemThesis references ChalmersHardProblemThesis
//     from hard_problem.go. Cross-file entity bridge.
//   - PinkerProposesBrownHumanUniversalsThesis references both StevenPinker
//     from blank_slate.go and BrownHumanUniversalsThesis from
//     human_universals.go. Double cross-file bridge in a single claim.

// Provenance for Dennett's dispute — drawn from the same Wikipedia article
// that sourced hard_problem.go, but the specific dispute text was
// previously deferred as "parasitic without own article ingests." The
// metabolism cycle's structural-fragility finding overrides that deferral:
// breaking single-source vulnerability is the explicit purpose of the
// sensor loop.
var dennettDisputeSource = Provenance{
	Origin: "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 2 (sensor: topology-driven ZIM query " +
		"'Chalmers hard problem', corroboration ingest)",
	Quote: "Its existence is rejected by other philosophers of mind, such as " +
		"Daniel Dennett, Massimo Pigliucci, Thomas Metzinger, Patricia " +
		"Churchland, and Keith Frankish [...] Daniel Dennett and Patricia " +
		"Churchland, among others, believe that the hard problem is best " +
		"seen as a collection of easy problems that will be solved through " +
		"further analysis of the brain and behaviour.",
}

// Provenance for Pinker's corroboration — from the Human_Universals
// Wikipedia article, a new source found by the ZIM "Brown Human
// Universals" query. The previous ingest (human_universals.go) used
// the DePaul course page which only cited Pinker as publication venue
// without editorial endorsement. This Wikipedia source explicitly
// commits to Pinker seeing the list as evidence.
var pinkerHumanUniversalsSource = Provenance{
	Origin: "Wikipedia (zim 2025-12) / Human_Universals",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 2 (sensor: topology-driven ZIM query " +
		"'Brown Human Universals', corroboration ingest)",
	Quote: "Steven Pinker lists all Brown's universals in the appendix of " +
		"his book The Blank Slate. The list is seen by Brown (and Pinker) " +
		"to be evidence of mental adaptations to communal life in our " +
		"species' evolutionary history.",
}

// -----------------------------------------------------------------------------
// New entity: Daniel Dennett. Minimal but non-parasitic — Dennett is a
// hub entity who future ingests on consciousness, philosophy of mind,
// or cognitive science will naturally reference.
// -----------------------------------------------------------------------------

var DanielDennett = Person{&Entity{
	ID:      "daniel-dennett",
	Name:    "Daniel Dennett",
	Kind:    "person",
	Aliases: []string{"Dennett"},
	Brief: "American philosopher and cognitive scientist (1942–2024), " +
		"prominent critic of the hard problem of consciousness. Dennett " +
		"argued that the hard problem is best seen as a collection of " +
		"easy problems solvable through further analysis of the brain " +
		"and behaviour — that subjective experience will be explained " +
		"once all the functional and mechanistic questions are answered, " +
		"leaving no residual 'hard' problem. Author of Consciousness " +
		"Explained (1991), Darwin's Dangerous Idea (1995), and From " +
		"Bacteria to Bach and Back (2017). His rejection of irreducible " +
		"qualia places him in direct opposition to Chalmers. Hub entity " +
		"for future philosophy-of-mind and cognitive-science ingests.",
}}

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	// Dennett disputes Chalmers' hard problem thesis. This breaks the
	// single-source vulnerability: ChalmersHardProblemThesis now has
	// a proposer (Chalmers, hard_problem.go) and a disputant (Dennett,
	// this file). The Wikipedia article names additional disputants
	// (Churchland, Frankish, Metzinger, Pigliucci, Dehaene, Baars,
	// Seth, Damasio) and supporters (Levine, McGinn, Block, Tononi,
	// Koch) — each could be a future cycle's ingest target.
	DennettDisputesHardProblemThesis = Disputes{
		Subject: DanielDennett,
		Object:  ChalmersHardProblemThesis,
		Prov:    dennettDisputeSource,
	}

	// Pinker proposes (endorses) the human universals thesis. Pinker
	// already exists in blank_slate.go with PinkerInfluencedByBrown
	// (human_universals.go). This new Proposes claim is sourced from
	// the Wikipedia Human_Universals article which explicitly commits
	// to Pinker seeing the list "as evidence of mental adaptations" —
	// stronger than the DePaul course page's publication-venue citation
	// that justified only InfluencedBy. The thesis now has two
	// proposers (Brown and Pinker), breaking its single-source
	// vulnerability.
	PinkerProposesBrownHumanUniversalsThesis = Proposes{
		Subject: StevenPinker,
		Object:  BrownHumanUniversalsThesis,
		Prov:    pinkerHumanUniversalsSource,
	}
)
