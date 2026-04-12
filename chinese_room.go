package winze

// Fifteenth public-corpus ingest, session 7 slice 1: Wikipedia's article
// on the Chinese room thought experiment. Chosen specifically to pressure-
// test two assumptions: (1) that the predicate vocabulary handles
// philosophical dispute representation without an ArguesAgainst predicate,
// and (2) that the existing role types (Concept, Hypothesis) can handle
// thought experiments and philosophical arguments — objects that are
// neither theories nor hypotheses in the usual sense.
//
// Schema forcing functions earned by this slice: NONE. Authored, Proposes,
// and TheoryOf carry all claims. Concept handles the thought experiment
// and the paper. Hypothesis handles Searle's positive thesis (biological
// naturalism). This is the ninth consecutive slice in an explored
// source-shape neighbourhood to earn zero new primitives.
//
// Pressure test results:
//
//   1. Dispute representation WITHOUT ArguesAgainst. Searle's Chinese
//      room is philosophy's most famous dispute — directed against
//      functionalism, computationalism, and strong AI. The system
//      represents this by encoding Searle's POSITIVE thesis (biological
//      naturalism: consciousness requires specific biological machinery)
//      as a Hypothesis with a TheoryOf(Consciousness) claim, which
//      competes with Watts's evolutionary-dead-end thesis from
//      blindsight.go. The "argues against" framing is the same thesis
//      in negative form — it does not need its own predicate. Competing
//      TheoryOf claims suffice for dispute representation.
//
//   2. Vocabulary boundary holds. A thought experiment is a Concept —
//      the Chinese room is an intellectual artifact, same pattern as
//      BlankSlate2002 or BlindsightNovel. A philosophical thesis is a
//      Hypothesis — biological naturalism is a positive claim about
//      consciousness. No ThoughtExperiment or Argument role type needed.
//
//   3. Consciousness fires as 7th contested target. Two TheoryOf
//      subjects now: WattsConsciousnessAsDeadEndThesis (blindsight.go)
//      and SearleBiologicalNaturalism (this file). Cross-file bridge
//      to blindsight.go's Consciousness entity, second demonstration
//      of the seed-and-wait pattern (first was HumanCognition in
//      session 6).
//
// Cross-file bridges:
//   - Consciousness (blindsight.go:105) — TheoryOf target, fires
//     contested-concept. Referenced from this file without editing the
//     source, same cross-file pattern as HumanCognition in blank_slate.go.
//
// Brief-level adjacency (no claim-level bridges):
//   - Rorschach (blindsight.go) dramatises the Chinese room: Watts's
//     alien vessel communicates in human languages without understanding,
//     explicitly modelled on Searle's thought experiment. But the
//     Wikipedia Chinese room article does not cite Watts, so no bridge.
//   - StevenPinker (blank_slate.go) is mentioned in the "carbon
//     chauvinism" section suggesting a counter thought experiment, but
//     this is a one-sentence reference — not InfluencedBy level.
//   - AndyClark (predictive_processing.go) is listed in the article's
//     philosopher sidebar but has no substantive mention in the body.
//   - DavidChalmers is mentioned arguing that "consciousness is at the
//     root of the matter" and that future LLMs might achieve
//     consciousness. Not reified — one-sentence engagement, same
//     discipline as Tenenbaum in mattson_pattern_processing.go.
//
// Deliberate exclusions:
//   - Strong AI / functionalism / computationalism — not reified as
//     Concepts. They are targets of Searle's argument, not positive
//     theses the article commits to at TheoryOf level. If a future
//     slice ingests the Stanford Encyclopedia's article on
//     computationalism, that source would commit at the required level.
//   - Individual respondents (Dennett, Minsky, Block, Boden,
//     Hofstadter, Harnad) — not reified. Each would need their own
//     article ingest to carry load-bearing claims. Reifying them here
//     creates parasitic Person entities with no claims beyond "responded
//     to Searle."
//   - Dennett's eliminative materialism — tempting as a 3rd
//     TheoryOf(Consciousness), but the article's treatment is "Dennett
//     describes consciousness as a 'user illusion'" — one sentence, not
//     the commitment level TheoryOf requires. Defer to a dedicated
//     Dennett ingest.
//   - Applied ethics section — declined (winze scope discipline).
//   - Computer science section — technical context on Turing
//     completeness and symbol processing, not claim-bearing.
//   - Turing test, China brain, brain replacement scenario — mentioned
//     but not reified (no load-bearing claims beyond existence).
//   - Leibniz's mill, Block's China brain, Dneprov's "The Game" — prior
//     art mentioned in the History section. Not reified because the
//     article positions them as precursors to Searle's version, not as
//     independent arguments with their own thesis commitments.
//
// Scope discipline: the article contains extensive discussion of the
// replies and counter-arguments (systems reply, robot reply, brain
// simulator, speed/complexity, other minds). All declined — the
// replies are responses to Searle, not independent theses the article
// commits to. The positive thesis (biological naturalism) and its
// provenance are extracted; the debate choreography is not winze's
// domain.

var chineseRoomSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Chinese_room",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 7 (chinese room first slice, dispute + vocabulary pressure test)",
	Quote: "The Chinese room argument holds that a computer executing a program cannot have a " +
		"mind, understanding, or consciousness, regardless of how intelligently or human-like " +
		"the program may make the computer behave. The argument was presented in a 1980 paper " +
		"by the philosopher John Searle entitled 'Minds, Brains, and Programs' and published " +
		"in the journal Behavioral and Brain Sciences. [...] The argument is directed against " +
		"the philosophical positions of functionalism and computationalism, which hold that the " +
		"mind may be viewed as an information-processing system operating on formal symbols. " +
		"[...] Searle holds a philosophical position he calls 'biological naturalism': that " +
		"consciousness and understanding require specific biological machinery that is found " +
		"in brains. He writes 'brains cause minds' and that 'actual human mental phenomena " +
		"[are] dependent on actual physical–chemical properties of actual human brains'. [...] " +
		"Searle does not disagree with the notion that machines can have consciousness and " +
		"understanding, because, as he writes, 'we are precisely such machines'. Searle holds " +
		"that the brain is, in fact, a machine, but that the brain gives rise to consciousness " +
		"and understanding using specific machinery. [...] David Chalmers writes, 'it is " +
		"fairly clear that consciousness is at the root of the matter' of the Chinese room. " +
		"[...] It eventually became the journal's 'most influential target article', " +
		"generating an enormous number of commentaries and responses in the ensuing decades.",
}

// -----------------------------------------------------------------------------
// Entities. Tight scope: the thought experiment (Concept), the paper
// (Concept), the author (Person), and the positive thesis (Hypothesis).
//
// The Chinese room argument is a Concept, not a Hypothesis — it is a
// thought experiment, an intellectual artifact used to argue FOR a
// thesis (biological naturalism). The thesis itself is the Hypothesis.
// This distinction is the vocabulary pressure test: Concept handles
// thought experiments without needing a ThoughtExperiment role type.
// Same pattern as BlankSlate2002 (the book is a Concept, the thesis
// is a Hypothesis).
// -----------------------------------------------------------------------------

var (
	ChineseRoomArgument = Concept{&Entity{
		ID:      "concept-chinese-room",
		Name:    "Chinese room",
		Kind:    "concept",
		Aliases: []string{"Chinese room argument", "Chinese room thought experiment"},
		Brief: "John Searle's 1980 thought experiment arguing that a computer executing " +
			"a program cannot have a mind, understanding, or consciousness. A person in a " +
			"room follows English instructions to manipulate Chinese symbols, producing " +
			"outputs indistinguishable from a native speaker, yet understands nothing — " +
			"demonstrating (Searle argues) that syntax alone cannot produce semantics. " +
			"Presented in 'Minds, Brains, and Programs' (Behavioral and Brain Sciences, " +
			"1980), which became the journal's 'most influential target article'. The " +
			"argument is directed against functionalism, computationalism, and what Searle " +
			"calls the 'strong AI hypothesis'. Similar arguments were made by Leibniz " +
			"(1714, the mill), Block (1978, China brain), and Dneprov (1961, 'The Game'), " +
			"but Searle's version is the canonical form. Dramatised in Peter Watts's " +
			"Blindsight (2006) where the alien vessel Rorschach communicates in human " +
			"languages without understanding — the novel's central illustration of the " +
			"Chinese room problem at interstellar scale.",
	}}

	JohnSearle = Person{&Entity{
		ID:    "john-searle",
		Name:  "John Searle",
		Kind:  "person",
		Brief: "American philosopher, professor at UC Berkeley, known for the Chinese " +
			"room argument (1980) and his philosophical position of biological naturalism. " +
			"Argues that consciousness and understanding require specific biological " +
			"machinery — 'brains cause minds' — opposing both functionalism (mind as " +
			"information processing) and Cartesian dualism (mind as non-physical " +
			"substance). Does not deny that machines can be conscious, since 'we are " +
			"precisely such machines', but holds that the specific causal powers of " +
			"biological neural machinery are necessary. The Chinese room thought " +
			"experiment is his most influential contribution to philosophy of mind, " +
			"generating 'an enormous number of commentaries and responses' over four " +
			"decades.",
	}}

	MindsBrainsAndPrograms1980 = Concept{&Entity{
		ID:      "concept-minds-brains-programs-1980",
		Name:    "Minds, Brains, and Programs",
		Kind:    "concept",
		Aliases: []string{"Searle 1980", "Minds Brains and Programs"},
		Brief: "John Searle's 1980 paper in Behavioral and Brain Sciences, presenting " +
			"the Chinese room thought experiment and advancing biological naturalism as " +
			"a philosophical position on consciousness. Became the journal's 'most " +
			"influential target article', described by David Cole as 'probably the most " +
			"widely discussed philosophical argument in cognitive science to appear in " +
			"the past 25 years' and by Varol Akman as 'an exemplar of philosophical " +
			"clarity and purity'. The paper's formal argument proceeds through three " +
			"axioms (programs are syntactic; minds have semantics; syntax alone cannot " +
			"produce semantics) to the conclusion that programs are neither constitutive " +
			"of nor sufficient for minds.",
	}}

	SearleBiologicalNaturalism = Hypothesis{&Entity{
		ID:   "hypothesis-searle-biological-naturalism",
		Name: "Searle's biological naturalism",
		Kind: "hypothesis",
		Brief: "The thesis, advanced by John Searle, that consciousness and understanding " +
			"require specific biological machinery found in brains — 'brains cause minds' " +
			"and 'actual human mental phenomena [are] dependent on actual physical–chemical " +
			"properties of actual human brains'. Searle holds that the brain's 'causal " +
			"powers' (the neural correlates of consciousness) are necessary for " +
			"consciousness, and that no purely computational simulation can produce it. " +
			"Directly opposed to both functionalism and behaviorism. Similar to identity " +
			"theory but with specific technical objections to that position. This is a " +
			"rival TheoryOf Consciousness to Watts's evolutionary-dead-end thesis " +
			"(blindsight.go): where Watts argues consciousness may be an evolutionary " +
			"dead end, Searle argues it is a product of specific biological machinery. " +
			"Both address what consciousness IS; they diverge on substrate (biology- " +
			"specific vs evolution-contingent).",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Four claims, each load-bearing:
//   1. Authored — Searle wrote the paper
//   2. Authored — Searle formulated the thought experiment
//   3. Proposes — Searle advances biological naturalism
//   4. TheoryOf — biological naturalism is a theory of Consciousness
//                  (fires as 7th contested target)
// -----------------------------------------------------------------------------

var (
	SearleAuthoredMindsBrainsPrograms = Authored{
		Subject: JohnSearle,
		Object:  MindsBrainsAndPrograms1980,
		Prov:    chineseRoomSource,
	}

	SearleAuthoredChineseRoom = Authored{
		Subject: JohnSearle,
		Object:  ChineseRoomArgument,
		Prov:    chineseRoomSource,
	}

	SearleProposessBiologicalNaturalism = Proposes{
		Subject: JohnSearle,
		Object:  SearleBiologicalNaturalism,
		Prov:    chineseRoomSource,
	}

	// This claim fires the contested-concept rule on Consciousness.
	// Consciousness was seeded in blindsight.go with
	// WattsConsciousnessTheory as its sole TheoryOf claim (Watts's
	// evolutionary-dead-end thesis). Adding this second TheoryOf from
	// a different file, written in a different session, makes
	// Consciousness the 7th contested target in winze and the second
	// demonstration of the seed-and-wait pattern (first was
	// HumanCognition fired in session 6 by blank_slate.go).
	SearleBiologicalNaturalismTheoryOfConsciousness = TheoryOf{
		Subject: SearleBiologicalNaturalism,
		Object:  Consciousness,
		Prov:    chineseRoomSource,
	}
)
