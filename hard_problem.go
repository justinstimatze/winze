package winze

// Seventeenth public-corpus ingest, session 8 slice 2: Wikipedia's article
// on the hard problem of consciousness. Surgical ingest to fire
// Consciousness as the first 3-way contested target in winze.
//
// Schema forcing functions earned by this slice: NONE. Twelfth consecutive
// slice with zero new primitives. Proposes, Authored, and TheoryOf carry
// all claims.
//
// This slice completes the Consciousness rivalry chain:
//   - Session 6 (blindsight.go): Watts seeds Consciousness with
//     evolutionary-dead-end thesis. TheoryOf #1.
//   - Session 7 (chinese_room.go): Searle adds biological naturalism.
//     TheoryOf #2. Consciousness fires as 7th contested target (2-way).
//   - Session 8 (this file): Chalmers adds hard problem irreducibility.
//     TheoryOf #3. Consciousness becomes first 3-way contested target.
//   Three files, three sessions, zero coordination. Seed-and-wait
//   pattern's third demonstration.
//
// The three rivals genuinely differ:
//   - Watts: consciousness may be an evolutionary dead end — maladaptive,
//     selected against in intelligent species.
//   - Searle: consciousness requires specific biological machinery —
//     brains cause minds, the causal powers of neural biology are
//     necessary.
//   - Chalmers: consciousness is irreducible to physical/functional
//     explanation — no amount of mechanistic analysis can explain WHY
//     neural processing is accompanied by subjective experience.
//
// Cross-file bridges:
//   - Consciousness (blindsight.go:105) — TheoryOf target.
//
// Brief-level adjacency (no claim-level bridges):
//   - StevenPinker (blank_slate.go) — mentioned praising Chalmers for
//     "impeccable clarity" and commenting on the hard problem. One-
//     sentence engagement, not InfluencedBy level.
//   - ChineseRoomArgument (chinese_room.go) — thematically related
//     (both concern limits of mechanistic explanation of mind). The
//     article does not mention Searle's Chinese room directly.
//   - DanielDennett — mentioned as rejecting the hard problem (with
//     Churchland, Frankish, Metzinger). Not reified — each would need
//     their own ingest.
//   - ThomasNagel — Chalmers uses Nagel's definition of consciousness
//     ("the feeling of what it is like to be something"). Mention-level,
//     not InfluencedBy level from this article.
//
// Deliberate exclusions:
//   - Type-A through Type-F taxonomy of philosophical responses —
//     rich content but belongs in a dedicated philosophy-of-mind ingest.
//   - IIT (Tononi), Global Workspace Theory (Baars/Dehaene) — each
//     deserves its own article. Either would be a 4th TheoryOf rival.
//   - All named respondents (Dennett, Churchland, Block, Nagel, Pinker,
//     McGinn, Frankish, Koch, Tononi) — same discipline as chinese_room.go
//     respondents. Parasitic without own article ingests.
//   - The Conscious Mind (1996) — Chalmers' book-length treatment. Could
//     be reified as Concept following MindsBrainsAndPrograms1980 pattern,
//     but the Wikipedia article treats the 1995 paper as the primary
//     source, not the book.
//   - "Harder Problem" (Block) and "Even Harder Problem" — refinements,
//     not independent theses.
//   - Meta-problem of consciousness — Chalmers' later work (2018), not
//     load-bearing here.
//
// Scope discipline: the article's extensive discussion of philosophical
// responses (six types of materialism/dualism/monism plus mysterianism)
// is declined. This ingest extracts Chalmers' POSITIVE THESIS and its
// structural relationship to Consciousness. The taxonomy of responses
// is not winze's domain from this source.

var hardProblemSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 8 (hard problem slice, Consciousness 3-way rivalry)",
	Quote: "In the philosophy of mind, the 'hard problem' of consciousness is to " +
		"explain why and how humans (and other organisms) have qualia, phenomenal " +
		"consciousness, or subjective experience. It is contrasted with the 'easy " +
		"problems' of explaining why and how physical systems give a human being " +
		"the ability to discriminate, to integrate information, and to perform " +
		"behavioural functions. [...] The terms 'hard problem' and 'easy problems' " +
		"were coined by the philosopher David Chalmers in a 1994 talk given at The " +
		"Science of Consciousness conference held in Tucson, Arizona. [...] " +
		"Chalmers argues that it is conceivable that the relevant behaviours " +
		"associated with hunger, or any other feeling, could occur even in the " +
		"absence of that feeling. This suggests that experience is irreducible to " +
		"physical systems such as the brain. [...] Chalmers's idea contradicts " +
		"physicalism [...] Though Chalmers rejects physicalism, he is still a " +
		"naturalist. [...] According to a 2020 PhilPapers survey, a majority " +
		"(62.42%) of the philosophers surveyed said they believed that the hard " +
		"problem is a genuine problem.",
}

// -----------------------------------------------------------------------------
// Entities. Three entities: Chalmers (Person), the hard problem as
// intellectual artifact (Concept), and his positive thesis (Hypothesis).
//
// The hard problem is a Concept — it's the intellectual artifact
// (the problem, the question), not the answer. Same pattern as
// ChineseRoomArgument: the thought experiment is a Concept, the thesis
// (biological naturalism) is a Hypothesis. Here: the hard problem is a
// Concept, Chalmers' answer (consciousness is irreducible) is a
// Hypothesis.
// -----------------------------------------------------------------------------

var (
	DavidChalmers = Person{&Entity{
		ID:      "david-chalmers",
		Name:    "David Chalmers",
		Kind:    "person",
		Aliases: []string{"Chalmers"},
		Brief:   "Australian philosopher who formulated the \"hard problem\" of consciousness, arguing that subjective experience cannot be explained by physical mechanisms alone, distinguishing it from solvable \"easy problems\" of consciousness.",
	}}

	HardProblemOfConsciousness = Concept{&Entity{
		ID:      "concept-hard-problem-of-consciousness",
		Name:    "Hard problem of consciousness",
		Kind:    "concept",
		Aliases: []string{"hard problem", "the hard problem"},
		Brief:   "Philosophical problem formulated by David Chalmers (1994) asking why physical brain processes generate subjective experience (qualia), distinct from explaining consciousness's functional and behavioral aspects.",
	}}

	ChalmersHardProblemThesis = Hypothesis{&Entity{
		ID:    "hypothesis-chalmers-hard-problem",
		Name:  "Chalmers' hard problem thesis",
		Kind:  "hypothesis",
		Brief: "Philosophical thesis by David Chalmers positing that subjective conscious experience cannot be explained by physical or functional mechanisms alone, even after solving all mechanistic problems of cognition and behavior.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Three claims:
//   1. Authored — Chalmers formulated the hard problem
//   2. Proposes — Chalmers advances the irreducibility thesis
//   3. TheoryOf — Chalmers' thesis is a theory of Consciousness
//                  (fires 3-way rivalry: Watts, Searle, Chalmers)
// -----------------------------------------------------------------------------

var (
	ChalmersAuthoredHardProblem = Authored{
		Subject: DavidChalmers,
		Object:  HardProblemOfConsciousness,
		Prov:    hardProblemSource,
	}

	ChalmersProposesHardProblemThesis = Proposes{
		Subject: DavidChalmers,
		Object:  ChalmersHardProblemThesis,
		Prov:    hardProblemSource,
	}

	// This claim fires Consciousness as the first 3-way contested target.
	// Three TheoryOf subjects from three files, three sessions:
	//   1. WattsConsciousnessAsDeadEndThesis (blindsight.go, session 6)
	//   2. SearleBiologicalNaturalism (chinese_room.go, session 7)
	//   3. ChalmersHardProblemThesis (this file, session 8)
	ChalmersHardProblemTheoryOfConsciousness = TheoryOf{
		Subject: ChalmersHardProblemThesis,
		Object:  Consciousness,
		Prov:    hardProblemSource,
	}
)
