package winze

// Tenth public-corpus ingest: Wikipedia's article on Carl Sagan's
// The Demon-Haunted World: Science as a Candle in the Dark (1995),
// combined with targeted facts from the Michael Shermer Wikipedia
// article. Paired intentionally to earn the first slice in winze
// that deliberately shops for *existing* entities to reference
// rather than adding content in isolation.
//
// Motivation: the query session run immediately before this slice
// (see the end of session 4 findings) exposed a surprising gap in
// the graph's density claims. Winze's "cross-ingest density" was
// real at the predicate-type level — 20+ predicates in active
// cross-file use, contested-concept rule firing on 4 targets
// across 3 ingests, IsCognitiveBias crossing file boundaries for
// the first time in the apophenia slice — but at the *entity*
// level only two user-content vars in the entire graph were
// referenced across file boundaries: Stope (bootstrap.go →
// stope_constraints.go, founding-session scaffolding) and
// AndyClark (predictive_processing.go → quantum_thief.go, the
// InfluencedBy bridge wired earlier this session). Every other
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
	IngestedBy: "winze session 4 (skepticism ingest, first deliberate cross-file-bridge slice)",
	Quote:      "The Demon-Haunted World: Science as a Candle in the Dark is a 1995 book by the astronomer and science communicator Carl Sagan. (Four of the 25 chapters were written with Ann Druyan.) In it, Sagan aims to explain the scientific method to laypeople and to encourage people to learn critical and skeptical thinking. [...] Sagan presents a set of tools for skeptical thinking that he calls the 'baloney detection kit'. [...] As an example of skeptical thinking, Sagan offers a story concerning a fire-breathing dragon who lives in his garage. [...] Sagan concludes by asking: 'Now what's the difference between an invisible, incorporeal, floating dragon who spits heatless fire and no dragon at all? If there's no way to disprove my contention, no conceivable experiment that would count against it, what does it mean to say that my dragon exists?'",
}

var shermerBioSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Michael_Shermer",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 4 (skepticism ingest, cross-file bridge source)",
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
		Brief: "American astronomer, planetary scientist, and science communicator (1934–1996). Author of The Demon-Haunted World: Science as a Candle in the Dark (1995), one of the most widely-read popular-science treatments of scientific skepticism, the scientific method, and the distinction between valid science and pseudoscience. Invented the 'baloney detection kit' framing (co-attributed in a 2020 Skeptical Inquirer interview by Ann Druyan to Sagan's friend Arthur Felberbaum), the 'dragon in my garage' thought experiment illustrating falsifiability, and a catalogue of twenty logical fallacies to avoid in skeptical reasoning. Hub entity: positioned so subsequent ingests on skepticism, philosophy of science, pseudoscience debunking, or the biographies of Hume / Popper / Russell / Feyerabend can cite him cheaply.",
	}}

	AnnDruyan = Person{&Entity{
		ID:    "ann-druyan",
		Name:  "Ann Druyan",
		Kind:  "person",
		Brief: "American author and science communicator, Carl Sagan's wife and frequent collaborator. Co-wrote four of the 25 chapters of The Demon-Haunted World (1995). In a 2020 Skeptical Inquirer interview she attributed the original framing of the 'baloney detection kit' phrase to a friend of hers named Arthur Felberbaum, not to Sagan — a minor attribution disagreement recorded here for the benefit of a future slice that wants to surface competing origin claims for named concepts.",
	}}

	DemonHauntedWorld = Concept{&Entity{
		ID:      "concept-demon-haunted-world",
		Name:    "The Demon-Haunted World: Science as a Candle in the Dark",
		Kind:    "concept",
		Aliases: []string{"Demon-Haunted World"},
		Brief:   "Carl Sagan's 1995 Random House book on scientific skepticism, the scientific method, and the demarcation of science from pseudoscience. Four chapters co-written with Ann Druyan. Contains the 'baloney detection kit' catalogue of skeptical-thinking tools and logical-fallacy avoidance list, the 'dragon in my garage' falsifiability thought experiment, and extended critiques of specific pseudoscience claims (UFOs, ESP, faith healing, repressed-memory therapy, alien abduction). The book is a real-world creative work but not fiction — winze currently has no IsNonfictionWork tag because one-off tag invention is premature; the name and Brief do the content work until a second non-fiction book forces the predicate.",
	}}

	ScientificSkepticism = Concept{&Entity{
		ID:      "concept-scientific-skepticism",
		Name:    "Scientific skepticism",
		Kind:    "concept",
		Aliases: []string{"scientific skeptic movement", "skepticism"},
		Brief:   "A position in the philosophy of science that holds empirical claims should be approached with critical inquiry and tested by the standards of the scientific method, with demonstrable belief proportional to evidence. The umbrella concept Sagan's Demon-Haunted World defends for a popular audience. Also the organising concept of the Skeptics Society, founded by Michael Shermer and Pat Linse in 1991 — the Skeptics Society's AffiliatedWith claim on the existing apophenia.go MichaelShermer entity is the first real cross-file entity reference in winze outside the Clark bridge.",
	}}

	Falsifiability = Concept{&Entity{
		ID:    "concept-falsifiability",
		Name:  "Falsifiability",
		Kind:  "concept",
		Brief: "The principle, most famously associated with Karl Popper, that a hypothesis is scientific if and only if it is possible in principle to devise an observation that would count against it. The demarcation criterion separating science from non-science. Sagan's dragon-in-garage thought experiment is a popular-audience illustration of the principle: a hypothesis whose defender can always invent a new reason why any proposed test would not work has no empirical content, and is therefore not scientific regardless of how sincerely it is held.",
	}}

	BaloneyDetectionKit = Concept{&Entity{
		ID:      "concept-baloney-detection-kit",
		Name:    "Baloney detection kit",
		Kind:    "concept",
		Aliases: []string{"BDK"},
		Brief:   "Sagan's popular-audience catalogue of skeptical-thinking tools, introduced in The Demon-Haunted World. Nine constructive tools (independent confirmation, encouraging debate, recognising argument-from-authority is unreliable, considering multiple hypotheses, avoiding bias toward one's own hypothesis, quantification, chain-of-argument validation, Occam's razor, falsifiability) plus twenty named logical fallacies to avoid (ad hominem, argument from authority, appeal to ignorance, special pleading, begging the question, observational selection, statistics of small numbers, non sequitur, post hoc ergo propter hoc, excluded middle, slippery slope, confusion of correlation and causation, straw man, suppressed evidence, weasel word, among others). Attribution of the phrase itself is contested: Ann Druyan in 2020 credited the original framing to Arthur Felberbaum, though the catalogue as published is Sagan's.",
	}}

	WhyPeopleBelieveWeirdThings = Concept{&Entity{
		ID:      "concept-why-people-believe-weird-things",
		Name:    "Why People Believe Weird Things",
		Kind:    "concept",
		Aliases: []string{"WPBWT"},
		Brief:   "Michael Shermer's 1997 book, full title Why People Believe Weird Things: Pseudoscience, Superstition, and Other Confusions of Our Time. The Michael Shermer Wikipedia article explicitly attributes the first use of 'patternicity' to this book (1997), disagreeing with the Apophenia Wikipedia article which dates the coinage to 2008. Winze records both datings by preserving the two source quotes in their respective Provenance records and flagging the dispute in the ShermerPatternicityFraming Brief in apophenia.go; no functional predicate is earned for this single occurrence.",
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
		Brief: "American non-profit organisation founded in 1991 by Michael Shermer and Pat Linse in Los Angeles, promoting scientific skepticism and debunking pseudoscience. Publishes Skeptic magazine and organises the Caltech Lecture Series. Had over 50,000 members as of 2017. The Skeptics Society's relation to the broader Scientific Skepticism concept is via its role as the organising institution that crystallised the skeptical-movement network Sagan had popularised four years earlier in The Demon-Haunted World.",
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
		Brief: "Sagan's illustration of falsifiability via a staged dialogue about a fire-breathing dragon living in his garage. The dragon is invisible, floats in the air, and breathes heatless fire — each new test proposed by the visitor is countered by a new reason the test will not work. Sagan concludes: 'Now what's the difference between an invisible, incorporeal, floating dragon who spits heatless fire and no dragon at all? ... Your inability to invalidate my hypothesis is not at all the same thing as proving it true.' The argument is presented in The Demon-Haunted World as a popular-audience illustration of the Popperian demarcation criterion, not as a novel philosophical contribution — Sagan's value-add is pedagogical rather than theoretical.",
	}}

	BaloneyDetectionKitThesis = Hypothesis{&Entity{
		ID:    "hyp-baloney-detection-kit-thesis",
		Name:  "A catalogue of fallacy-detection tools and named logical fallacies is a practical method by which laypeople can distinguish valid scientific claims from pseudoscience",
		Kind:  "hypothesis",
		Brief: "The central thesis of the baloney detection kit section of The Demon-Haunted World. Sagan proposes that the tools of scientific thinking — typically learned implicitly through research training — can be made explicit and taught to laypeople as a bounded checklist, and that doing so enables ordinary readers to distinguish empirical claims from pseudoscientific assertions without needing specialist expertise. The thesis is a popular-education claim, not a claim about the philosophy of science — compare with Hempel, Popper, Kuhn, or Feyerabend, who make foundational claims Sagan's book takes as given.",
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
