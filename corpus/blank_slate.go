package winze

// Pinker's The Blank Slate ingest: Wikipedia's article on the 2002 book.
// Chosen specifically to test the seed-and-wait pattern — HumanCognition
// was seeded as a contested-target-ready Concept by Mattson's SPP thesis
// in mattson_pattern_processing.go, with one TheoryOf
// claim waiting for a rival. This slice provides the rival: Pinker's
// evolutionary-psychology thesis that human behaviour is significantly
// shaped by evolved psychological traits, which is a different framing
// from Mattson's cortical-expansion-for-pattern-processing.
//
// Schema forcing functions earned by this slice: NONE. Authored, Proposes,
// and TheoryOf carry all three claims. This is the eighth consecutive
// slice where an ingest in a previously-explored
// source-shape neighbourhood earns zero new primitives.
//
// Seed-and-wait demonstration: HumanCognition now has two TheoryOf
// claims from two different files (mattson_pattern_processing.go and
// blank_slate.go), written independently, with zero
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
	IngestedBy: "winze",
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
		Brief:   "2002 book by Steven Pinker rejecting tabula rasa models and arguing human behavior is shaped by evolved psychological traits, challenging blank slate, noble savage, and mind-body dualism theories.",
	}}

	StevenPinker = Person{&Entity{
		ID:      "steven-pinker",
		Name:    "Steven Pinker",
		Kind:    "person",
		Aliases: []string{"Pinker"},
		Brief:   "Canadian-American cognitive psychologist and author of The Language Instinct, How the Mind Works, and The Blank Slate. Known for arguing that human behavior has significant innate, evolved structure grounded in evolutionary psychology.",
	}}

	PinkerHumanNatureThesis = Hypothesis{&Entity{
		ID:    "hypothesis-pinker-human-nature",
		Name:  "Pinker evolutionary-psychology thesis of human nature",
		Kind:  "hypothesis",
		Brief: "Evolutionary psychologist Steven Pinker's thesis that human behavior stems from evolved psychological modules rather than a blank-slate mind, challenging the doctrine that the mind lacks innate traits.",
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
	// different ingest, makes HumanCognition the 6th contested target
	// in winze and the first to demonstrate the seed-and-wait pattern.
	PinkerThesisTheoryOfHumanCognition = TheoryOf{
		Subject: PinkerHumanNatureThesis,
		Object:  HumanCognition,
		Prov:    blankSlateSource,
	}
)
