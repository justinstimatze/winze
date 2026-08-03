package winze

// Tenth public-corpus ingest: Wikipedia's article on Carl Sagan's
// The Demon-Haunted World: Science as a Candle in the Dark (1995),
// combined with targeted facts from the Michael Shermer Wikipedia
// article. Paired intentionally to earn the first slice in winze
// that deliberately shops for *existing* entities to reference
// rather than adding content in isolation.
//
// Motivation: a query run immediately before this slice
// exposed a surprising gap in
// the graph's density claims. Winze's "cross-ingest density" was
// real at the predicate-type level — 20+ predicates in active
// cross-file use, contested-concept rule firing on 4 targets
// across 3 ingests, IsCognitiveBias crossing file boundaries for
// the first time in the apophenia slice — but at the *entity*
// level only two user-content vars in the entire graph were
// referenced across file boundaries:
// AndyClark (predictive_processing.go → quantum_thief.go, the
// InfluencedBy bridge wired earlier). Every other
// slice was a silo connected to the rest of winze only by shared
// predicate types, not by shared entities. The "graph density"
// claim needed correction: predicate density was real, entity
// density was a stated goal that had not actually been achieved.
//
// This slice is the first deliberate corrective. The scope was
// chosen specifically because Sagan's book is adjacent to three
// already-ingested domains (cognitive biases and apophenia via
// the fallacy catalogue; predictive processing via the pattern-
// matching discussion; misconceptions via the general debunking
// mode) AND because the Michael Shermer Wikipedia article cites
// facts about Shermer — who already exists as a Person entity in
// apophenia.go — that can be honestly promoted from Brief-level
// prose to claim-level assertions. Two new claims target the
// existing MichaelShermer entity from inside this file: the first
// real cross-file entity reference in winze that is NOT the Clark
// bridge. Entity-level density begins to earn itself here.
//
// Schema forcing functions earned by this slice:
//
//   - None. Fifth consecutive slice to earn zero new primitives.
//     All eight claims reuse Authored, Proposes, TheoryOf, and
//     AffiliatedWith. The vocabulary-fit pattern is now unmistakable:
//     winze's predicate accretion rate has converged, at least for
//     concept-adjacent ingests in the academic/popular-science
//     neighbourhood. The interesting question is whether this is
//     because the vocabulary is actually complete for this
//     neighbourhood or because the slices keep choosing sources
//     that fit the existing vocabulary — a selection-bias
//     question a slice from a genuinely new domain (scientific
//     paper, legal document, recipe corpus) would answer.
//
// Cross-file bridges wired by this slice:
//
//   - `ShermerAffiliatedWithSkepticsSociety` — the claim lives in
//     demon_haunted.go but references MichaelShermer defined in
//     apophenia.go. First real cross-file claim in winze whose
//     subject is an entity from a different file.
//
//   - `ShermerAuthoredWhyPeopleBelieveWeirdThings` — second cross-
//     file claim in the same slice, same subject. Two claims with
//     cross-file subjects plus the pre-existing AndyClark bridge
//     moves winze from "one cross-file bridge in the entire graph"
//     to "three", which is the earliest point the entity-density
//     pattern can honestly be called a pattern rather than an
//     accident.
//
//   - Planted: CarlSagan, DemonHauntedWorld, and the baloney
//     detection kit / dragon-in-garage / falsifiability cluster
//     are new entities in their own right, but they are
//     *positioned* so future slices that cite Sagan (a Sagan
//     biography ingest; any Hume / Popper / Russell / Feyerabend
//     slice that connects skepticism and philosophy of science; a
//     dedicated pseudoscience ingest on UFOs, ESP, or faith
//     healing) can reference them cheaply. Sagan is the kind of
//     hub entity that a claim graph in this subject neighbourhood
//     will want to reference many times.
//
// Factual dispute noted but not reified:
//
//   - The Apophenia article and the Michael Shermer biography
//     disagree on when Shermer coined "patternicity". The
//     Apophenia article says 2008; the Shermer biography
//     attributes the term to his 1997 Why People Believe Weird
//     Things book. The disagreement is real but small, and
//     reifying it into a CoinedIn[Concept, TemporalMarker]
//     functional predicate would invent schema for a single
//     occurrence — exactly the kind of premature predicate-family
//     that winze's discipline forbids. Instead the disagreement
//     is recorded only in the Brief of ShermerPatternicityFraming
//     (updated in the apophenia.go slice pass to note the
//     competing dates) and in this file's header comment. A
//     future slice with a second coined-term dating dispute can
//     earn the predicate.

var sagan1995Source = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Demon-Haunted_World (Carl Sagan, 1995, Random House)",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "The Demon-Haunted World: Science as a Candle in the Dark is a 1995 book by the astronomer and science communicator Carl Sagan. (Four of the 25 chapters were written with Ann Druyan.) In it, Sagan aims to explain the scientific method to laypeople and to encourage people to learn critical and skeptical thinking. [...] Sagan presents a set of tools for skeptical thinking that he calls the 'baloney detection kit'. [...] As an example of skeptical thinking, Sagan offers a story concerning a fire-breathing dragon who lives in his garage. [...] Sagan concludes by asking: 'Now what's the difference between an invisible, incorporeal, floating dragon who spits heatless fire and no dragon at all? If there's no way to disprove my contention, no conceivable experiment that would count against it, what does it mean to say that my dragon exists?'",
}

var shermerBioSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Michael_Shermer",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "Michael Brant Shermer is an American science writer, historian of science, executive director of The Skeptics Society, and founding publisher of Skeptic magazine, a publication focused on investigating pseudoscientific and supernatural claims. [...] In 1991, Shermer and Pat Linse co-founded the Skeptics Society in Los Angeles. [...] Writing in Why People Believe Weird Things: Pseudoscience, Superstition, and Other Confusions of Our Time (1997), Shermer refers to 'patternicity', his term for pareidolia and apophenia or the willing suspension of disbelief.",
}

// -----------------------------------------------------------------------------
// New real-world entities introduced by this slice. Sagan and Druyan are
// genuinely new; the DemonHauntedWorld book concept is new; baloney detection
// kit, dragon-in-garage argument, scientific skepticism and falsifiability
// are new conceptual/hypothesis entities. All are deliberately positioned
// so that future ingests can reference them cheaply.
// -----------------------------------------------------------------------------

var (
	CarlSagan = Person{&Entity{
		ID:    "carl-sagan",
		Name:  "Carl Sagan",
		Kind:  "person",
		Brief: "American astronomer and science communicator (1934–1996) who advanced scientific skepticism through works like The Demon-Haunted World and concepts such as the \"baloney detection kit\" and \"dragon in my garage\" thought experiment.",
	}}

	AnnDruyan = Person{&Entity{
		ID:    "ann-druyan",
		Name:  "Ann Druyan",
		Kind:  "person",
		Brief: "American author and science communicator; wife and frequent collaborator of Carl Sagan. Co-wrote four chapters of The Demon-Haunted World (1995).",
	}}

	DemonHauntedWorld = Concept{&Entity{
		ID:      "concept-demon-haunted-world",
		Name:    "The Demon-Haunted World: Science as a Candle in the Dark",
		Kind:    "concept",
		Aliases: []string{"Demon-Haunted World"},
		Brief:   "1995 book by Carl Sagan and Ann Druyan on scientific skepticism and the demarcation of science from pseudoscience, featuring the \"baloney detection kit\" and critiques of UFOs, ESP, and other pseudoscience claims.",
	}}

	ScientificSkepticism = Concept{&Entity{
		ID:      "concept-scientific-skepticism",
		Name:    "Scientific skepticism",
		Kind:    "concept",
		Aliases: []string{"scientific skeptic movement", "skepticism"},
		Brief:   "Philosophical position holding that empirical claims should be tested by scientific method and belief proportioned to evidence. Foundational concept of the Skeptics Society.",
	}}

	Falsifiability = Concept{&Entity{
		ID:    "concept-falsifiability",
		Name:  "Falsifiability",
		Kind:  "concept",
		Brief: "The principle that a scientific hypothesis must be testable and potentially disprovable through observation. A demarcation criterion distinguishing science from non-science, most famously articulated by Karl Popper.",
	}}

	BaloneyDetectionKit = Concept{&Entity{
		ID:      "concept-baloney-detection-kit",
		Name:    "Baloney detection kit",
		Kind:    "concept",
		Aliases: []string{"BDK"},
		Brief:   "A set of nine critical-thinking tools and twenty logical fallacies presented by Carl Sagan in The Demon-Haunted World to evaluate claims and detect faulty reasoning.",
	}}

	WhyPeopleBelieveWeirdThings = Concept{&Entity{
		ID:      "concept-why-people-believe-weird-things",
		Name:    "Why People Believe Weird Things",
		Kind:    "concept",
		Aliases: []string{"WPBWT"},
		Brief:   "Michael Shermer's 1997 book exploring why people accept pseudoscience, superstition, and irrational beliefs. Often credited as the first use of the term \"patternicity.",
	}}
)

// -----------------------------------------------------------------------------
// New organization: the Skeptics Society, referenced by a cross-file
// claim against the existing MichaelShermer entity.
// -----------------------------------------------------------------------------

var (
	SkepticsSociety = Organization{&Entity{
		ID:    "org-skeptics-society",
		Name:  "The Skeptics Society",
		Kind:  "organization",
		Brief: "American non-profit organization founded in 1991 by Michael Shermer and Pat Linse that promotes scientific skepticism and debunks pseudoscience through Skeptic magazine and the Caltech Lecture Series.",
	}}
)

// -----------------------------------------------------------------------------
// Hypotheses — reified arguments from Sagan's book.
// -----------------------------------------------------------------------------

var (
	DragonInGarageArgument = Hypothesis{&Entity{
		ID:    "hyp-dragon-in-garage",
		Name:  "A hypothesis whose defender always invents a reason why any proposed test would not work has no empirical content and is not scientific",
		Kind:  "hypothesis",
		Brief: "Thought experiment by Carl Sagan illustrating falsifiability: a dragon in his garage with undetectable properties (invisible, incorporeal, heatless fire) that cannot be tested, demonstrating why unfalsifiable claims lack scientific validity.",
	}}

	BaloneyDetectionKitThesis = Hypothesis{&Entity{
		ID:    "hyp-baloney-detection-kit-thesis",
		Name:  "A catalogue of fallacy-detection tools and named logical fallacies is a practical method by which laypeople can distinguish valid scientific claims from pseudoscience",
		Kind:  "hypothesis",
		Brief: "Sagan's thesis that scientific thinking tools can be made explicit as a checklist and taught to laypeople to distinguish empirical claims from pseudoscience without specialist expertise.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Seven within-file claims plus two cross-file bridges referencing
// MichaelShermer (defined in apophenia.go). The cross-file claims are the
// slice's sharpest deliverable — first real entity-level bridge in winze
// outside the pre-existing Clark link.
// -----------------------------------------------------------------------------

var (
	SaganAuthoredDemonHauntedWorld = Authored{
		Subject: CarlSagan,
		Object:  DemonHauntedWorld,
		Prov:    sagan1995Source,
	}

	// Co-authorship is recorded as a second Authored claim with the
	// same Object, which the non-functional Authored predicate
	// accepts without complaint. Druyan co-wrote only four of the 25
	// chapters, but winze has no chapter-level granularity and
	// recording the partial contribution at brief-level is honest.
	DruyanCoAuthoredDemonHauntedWorld = Authored{
		Subject: AnnDruyan,
		Object:  DemonHauntedWorld,
		Prov:    sagan1995Source,
	}

	SaganProposesBaloneyDetectionKit = Proposes{
		Subject: CarlSagan,
		Object:  BaloneyDetectionKitThesis,
		Prov:    sagan1995Source,
	}

	SaganProposesDragonInGarage = Proposes{
		Subject: CarlSagan,
		Object:  DragonInGarageArgument,
		Prov:    sagan1995Source,
	}

	DragonInGarageTheoryOfFalsifiability = TheoryOf{
		Subject: DragonInGarageArgument,
		Object:  Falsifiability,
		Prov:    sagan1995Source,
	}

	BaloneyDetectionKitThesisTheoryOfScientificSkepticism = TheoryOf{
		Subject: BaloneyDetectionKitThesis,
		Object:  ScientificSkepticism,
		Prov:    sagan1995Source,
	}

	// BaloneyDetectionKit the concept BelongsTo DemonHauntedWorld as
	// a proper part of that book. Reuse of the BelongsTo pattern in
	// a fifth subject domain (after cognitive bias families, the
	// Jean le Flambeur trilogy series, the cognitive-biases umbrella,
	// and the apophenia sub-types).
	BaloneyDetectionKitBelongsToDemonHauntedWorld = BelongsTo{
		Subject: BaloneyDetectionKit,
		Object:  DemonHauntedWorld,
		Prov:    sagan1995Source,
	}

	// -------------------------------------------------------------------------
	// CROSS-FILE BRIDGES. The two claims below target MichaelShermer,
	// defined in apophenia.go. They are the first real cross-file
	// entity references in winze outside the AndyClark bridge, and
	// they use the shermerBioSource Provenance because their content
	// comes from the Shermer biography Wikipedia article, not from
	// The Demon-Haunted World itself.
	// -------------------------------------------------------------------------

	ShermerAffiliatedWithSkepticsSociety = AffiliatedWith{
		Subject: MichaelShermer,
		Object:  SkepticsSociety,
		Prov:    shermerBioSource,
	}

	ShermerAuthoredWhyPeopleBelieveWeirdThings = Authored{
		Subject: MichaelShermer,
		Object:  WhyPeopleBelieveWeirdThings,
		Prov:    shermerBioSource,
	}
)
