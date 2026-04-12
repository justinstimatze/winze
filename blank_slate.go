package winze

// Pinker's The Blank Slate ingest: Wikipedia's article on the 2002 book.
// Chosen specifically to test the seed-and-wait pattern — HumanCognition
// was seeded as a contested-target-ready Concept by Mattson's SPP thesis
// in mattson_pattern_processing.go (session 5 slice 2), with one TheoryOf
// claim waiting for a rival. This slice provides the rival: Pinker's
// evolutionary-psychology thesis that human behaviour is significantly
// shaped by evolved psychological traits, which is a different framing
// from Mattson's cortical-expansion-for-pattern-processing.
//
// Schema forcing functions earned by this slice: NONE. Authored, Proposes,
// and TheoryOf carry all three claims. This is the eighth consecutive
// slice (since session 3) where an ingest in a previously-explored
// source-shape neighbourhood earns zero new primitives.
//
// Seed-and-wait demonstration: HumanCognition now has two TheoryOf
// claims from two different files (mattson_pattern_processing.go and
// blank_slate.go), written in two different sessions, with zero
// coordination between the slices. The contested-concept rule fires
// automatically on the lint run after this file is added. This is the
// first time the pattern has been demonstrated end-to-end.
//
// Cross-file bridges:
//   - HumanCognition (mattson_pattern_processing.go) — the TheoryOf
//     target, referenced from this file without editing the source.
//   - DonaldEBrown (human_universals.go) — Brief-level adjacency only.
//     The Blank Slate Wikipedia article's See Also section lists Brown
//     and Human Universals, but the article body does not commit to the
//     influence relationship at the level InfluencedBy requires. The
//     DePaul handout (human_universals.go provenance) explicitly says
//     Brown's list was "republished in Pinker 2002 The Blank Slate" —
//     a future edit to human_universals.go could wire InfluencedBy using
//     that source's commitment. This file declines to fabricate the
//     bridge from a See Also link.
//
// Scope discipline: the article contains extensive political-philosophy
// content (fears of inequality, determinism, nihilism, totalitarianism).
// All declined — same discipline as the UDHR normative deferral. The
// cognitive-science thesis and its provenance are extracted; the
// political arguments are not winze's domain.
//
// Reception: substantial positive (Buss, Dawkins, Dennett, Bloom,
// Grayling) and negative (Schlinger, Ludvig, Dupré, Orr, Eriksen,
// Bateson, Menand) reviews documented in the Wikipedia article. None
// reified — no CritiquesOrDisputes predicate exists, and the critical
// responses do not force one. If a future slice ingests a specific
// rebuttal paper, that paper's own Proposes + TheoryOf claims would
// naturally enter the graph via the existing vocabulary.

var blankSlateSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Blank_Slate",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 6 (blank slate first slice)",
	Quote: "The Blank Slate: The Modern Denial of Human Nature is a best-selling 2002 book " +
		"by cognitive psychologist Steven Pinker, in which he argues against tabula rasa models " +
		"in the social sciences, claiming that human behaviour is significantly shaped by evolved " +
		"psychological traits. [...] Pinker argues that modern science has challenged three " +
		"'linked dogmas' that constitute the dominant view of human nature in intellectual life: " +
		"The blank slate — the idea that the mind has no innate traits (empiricism); " +
		"The noble savage — the belief that people are born good and corrupted by society (romanticism); " +
		"The ghost in the machine — the notion that each person has a soul making choices independent " +
		"of biology (dualism). [...] he argues that political equality does not require sameness, " +
		"but policies that treat people as individuals with rights; that moral progress does not " +
		"require the human mind to be naturally free of selfish motives, only that it has other " +
		"motives to counteract them.",
}

// -----------------------------------------------------------------------------
// Entities. Tight scope: the book (Concept), the author (Person), and the
// thesis (Hypothesis). TabulaRasa as a concept Pinker disputes is mentioned
// in Brief but not reified — no ArguesAgainst predicate exists and the
// article's treatment is "Pinker argues against X" rather than "X is a
// theory of Y", so there is no honest TheoryOf claim to wire for it.
// -----------------------------------------------------------------------------

var (
	BlankSlate2002 = Concept{&Entity{
		ID:      "concept-blank-slate-2002",
		Name:    "The Blank Slate",
		Kind:    "concept",
		Aliases: []string{"The Blank Slate: The Modern Denial of Human Nature"},
		Brief: "Steven Pinker's 2002 book arguing against tabula rasa models in the " +
			"social sciences. Claims human behaviour is significantly shaped by evolved " +
			"psychological traits and challenges three 'linked dogmas': the blank slate " +
			"(no innate traits), the noble savage (born good, corrupted by society), and " +
			"the ghost in the machine (soul independent of biology). Nominated for the " +
			"Aventis Prize and finalist for the Pulitzer Prize. The book's appendix " +
			"republishes Donald Brown's human universals list from Brown 1991 — the " +
			"same list ingested in human_universals.go via the DePaul handout source. " +
			"That source commits to the Pinker-Brown relationship; the Blank Slate " +
			"Wikipedia article's See Also section lists Brown and Human Universals but " +
			"the body text does not commit to the influence at InfluencedBy level.",
	}}

	StevenPinker = Person{&Entity{
		ID:      "steven-pinker",
		Name:    "Steven Pinker",
		Kind:    "person",
		Aliases: []string{"Pinker"},
		Brief: "Canadian-American cognitive psychologist and author, known for The " +
			"Language Instinct (1994), How the Mind Works (1997), The Blank Slate (2002), " +
			"The Better Angels of Our Nature (2011), and Enlightenment Now (2018). " +
			"Argues that human behaviour has significant innate, evolved structure — an " +
			"evolutionary-psychology framing of human cognition that is a rival to " +
			"Mattson's cortical-expansion-for-pattern-processing framing. The Wikipedia " +
			"article's See Also lists Donald Brown and Human Universals; Pinker " +
			"republished Brown's human universals list as an appendix to The Blank Slate.",
	}}

	PinkerHumanNatureThesis = Hypothesis{&Entity{
		ID:   "hypothesis-pinker-human-nature",
		Name: "Pinker evolutionary-psychology thesis of human nature",
		Kind: "hypothesis",
		Brief: "The thesis, advanced by Steven Pinker in The Blank Slate (2002), that " +
			"human behaviour is significantly shaped by evolved psychological traits and " +
			"that the 'blank slate' doctrine — the idea that the mind has no innate " +
			"traits — is refuted by modern science. Pinker's framing is evolutionary-" +
			"psychological: domain-specific evolved modules shape cognition, morality, " +
			"and social behaviour. This is a rival to Mattson's SPP thesis (superior " +
			"pattern processing as the fundamental basis of human cognition, locating " +
			"the substrate in cortical expansion of specific brain regions rather than " +
			"in domain-specific evolutionary psychology). Both are TheoryOf HumanCognition; " +
			"the contested-concept rule fires on HumanCognition when this claim is added.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Three claims, each load-bearing:
//   1. Authored — Pinker wrote the book
//   2. Proposes — Pinker advances the thesis
//   3. TheoryOf — the thesis is a theory of HumanCognition (fires contested)
// -----------------------------------------------------------------------------

var (
	PinkerAuthoredBlankSlate = Authored{
		Subject: StevenPinker,
		Object:  BlankSlate2002,
		Prov:    blankSlateSource,
	}

	PinkerProposesHumanNatureThesis = Proposes{
		Subject: StevenPinker,
		Object:  PinkerHumanNatureThesis,
		Prov:    blankSlateSource,
	}

	// This claim fires the contested-concept rule on HumanCognition.
	// HumanCognition was seeded in mattson_pattern_processing.go with
	// MattsonSPPThesisTheoryOfHumanCognition as its sole TheoryOf claim.
	// Adding this second TheoryOf from a different file, written in a
	// different session, makes HumanCognition the 6th contested target
	// in winze and the first to demonstrate the seed-and-wait pattern.
	PinkerThesisTheoryOfHumanCognition = TheoryOf{
		Subject: PinkerHumanNatureThesis,
		Object:  HumanCognition,
		Prov:    blankSlateSource,
	}
)
