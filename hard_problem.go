package winze

// Wikipedia's Hard Problem of Consciousness article — fires Consciousness
// as the first 3-way contested target (Watts evolutionary-dead-end,
// Searle biological naturalism, Chalmers irreducibility). No new predicates.
//
// Scope: Chalmers' positive thesis + Accepts/EarlyFormulationOf claims
// for Kirk, Campbell, Nagel. Excludes: Type-A–F taxonomy of responses,
// IIT/GWT (each deserves own article), named respondents (Dennett,
// Churchland, etc. — parasitic without own ingests).

var hardProblemSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
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
	// Three TheoryOf subjects from three files:
	//   1. WattsConsciousnessAsDeadEndThesis (blindsight.go)
	//   2. SearleBiologicalNaturalism (chinese_room.go)
	//   3. ChalmersHardProblemThesis (this file)
	ChalmersHardProblemTheoryOfConsciousness = TheoryOf{
		Subject: ChalmersHardProblemThesis,
		Object:  Consciousness,
		Prov:    hardProblemSource,
	}
)

// ---------------------------------------------------------------------------
// Hard problem of consciousness: Joseph Levine Accepts ChalmersHardProblemThesis
// Levine's acceptance of the hard problem demonstrates philosophical agreement that minds cannot fully model consciousness through physical mechanisms alone.
// ---------------------------------------------------------------------------

var josephLevineAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness (batch attribution: 6 scholars listed in one sentence)",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been accepted by some philosophers of mind such as Joseph Levine, Colin McGinn, and Ned Block and cognitive neuroscientists such as Francisco Varela, Giulio Tononi, and Christof Koch.\"",
}

var JosephLevine = Person{&Entity{
	ID:    "joseph-levine",
	Name:  "Joseph Levine",
	Kind:  "person",
	Brief: "Philosopher of mind who coined the concept of the 'explanatory gap' between physical processes and subjective experience, and accepted the hard problem of consciousness.",
}}

var JosephLevineAcceptsChalmersHardProblemThesis = Accepts{
	Subject: JosephLevine,
	Object:  ChalmersHardProblemThesis,
	Prov:    josephLevineAcceptsSource,
}

// ---------------------------------------------------------------------------
// Hard problem of consciousness: Colin McGinn Accepts ChalmersHardProblemThesis
// McGinn's acceptance of the hard problem demonstrates philosophical support for the thesis that consciousness poses unique epistemological challenges to physicalist models of mind.
// ---------------------------------------------------------------------------

var colinMcGinnAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness (batch attribution: 6 scholars listed in one sentence)",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been accepted by some philosophers of mind such as Joseph Levine, Colin McGinn, and Ned Block and cognitive neuroscientists such as Francisco Varela, Giulio Tononi, and Christof Koch.\"",
}

var ColinMcGinn = Person{&Entity{
	ID:    "colin-mcginn",
	Name:  "Colin McGinn",
	Kind:  "person",
	Brief: "Philosopher of mind and proponent of cognitive closure who accepted the hard problem of consciousness.",
}}

var ColinMcGinnAcceptsChalmersHardProblemThesis = Accepts{
	Subject: ColinMcGinn,
	Object:  ChalmersHardProblemThesis,
	Prov:    colinMcGinnAcceptsSource,
}

// ---------------------------------------------------------------------------
// Hard problem of consciousness: Ned Block Accepts ChalmersHardProblemThesis
// Block's acceptance of the hard problem shows agreement that current mechanistic approaches fail to explain phenomenal consciousness.
// ---------------------------------------------------------------------------

var nedBlockAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness (batch attribution: 6 scholars listed in one sentence)",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been accepted by some philosophers of mind such as Joseph Levine, Colin McGinn, and Ned Block and cognitive neuroscientists such as Francisco Varela, Giulio Tononi, and Christof Koch.\"",
}

var NedBlock = Person{&Entity{
	ID:    "ned-block",
	Name:  "Ned Block",
	Kind:  "person",
	Brief: "Philosopher of mind who distinguished phenomenal from access consciousness and accepted the hard problem of consciousness.",
}}

var NedBlockAcceptsChalmersHardProblemThesis = Accepts{
	Subject: NedBlock,
	Object:  ChalmersHardProblemThesis,
	Prov:    nedBlockAcceptsSource,
}

// ---------------------------------------------------------------------------
// Hard problem of consciousness: Francisco Varela Accepts ChalmersHardProblemThesis
// Varela's acceptance demonstrates that empirical neuroscience researchers acknowledge limits in explaining subjective experience through functional mechanisms.
// ---------------------------------------------------------------------------

var franciscoVarelaAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness (batch attribution: 6 scholars listed in one sentence)",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been accepted by some philosophers of mind such as Joseph Levine, Colin McGinn, and Ned Block and cognitive neuroscientists such as Francisco Varela, Giulio Tononi, and Christof Koch.\"",
}

var FranciscoVarela = Person{&Entity{
	ID:    "francisco-varela",
	Name:  "Francisco Varela",
	Kind:  "person",
	Brief: "Chilean cognitive neuroscientist and co-author of The Embodied Mind (1991) who accepted the hard problem of consciousness.",
}}

var FranciscoVarelaAcceptsChalmersHardProblemThesis = Accepts{
	Subject: FranciscoVarela,
	Object:  ChalmersHardProblemThesis,
	Prov:    franciscoVarelaAcceptsSource,
}

// ---------------------------------------------------------------------------
// Hard problem of consciousness: Giulio Tononi Accepts ChalmersHardProblemThesis
// Tononi's acceptance shows recognition within neuroscience that consciousness presents
// ---------------------------------------------------------------------------

var giulioTononiAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness (batch attribution: 6 scholars listed in one sentence)",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been accepted by some philosophers of mind such as Joseph Levine, Colin McGinn, and Ned Block and cognitive neuroscientists such as Francisco Varela, Giulio Tononi, and Christof Koch.\"",
}

var GiulioTononi = Person{&Entity{
	ID:    "giulio-tononi",
	Name:  "Giulio Tononi",
	Kind:  "person",
	Brief: "Neuroscientist who developed Integrated Information Theory (IIT) and accepted the hard problem of consciousness.",
}}

var GiulioTononiAcceptsChalmersHardProblemThesis = Accepts{
	Subject: GiulioTononi,
	Object:  ChalmersHardProblemThesis,
	Prov:    giulioTononiAcceptsSource,
}

// ---------------------------------------------------------------------------
// The Conscious Mind: Patricia Churchland Disputes ChalmersHardProblemThesis
// This documents a direct philosophical dispute about whether consciousness fails to supervene on physical facts, challenging Chalmers' epistemological claim about mind-body relations.
// ---------------------------------------------------------------------------

var patriciaChurchlandDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Conscious_Mind",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Patricia and Paul Churchland have criticised Chalmers claim that everything but consciousness logically supervenes on the physical, and that such failures of supervenience mean that materialism must be false.\"",
}

var PatriciaChurchland = Person{&Entity{
	ID:    "patricia-churchland",
	Name:  "Patricia Churchland",
	Kind:  "person",
	Brief: "Philosopher who criticized Chalmers' claims about supervenience and materialism.",
}}

var PatriciaChurchlandDisputesChalmersHardProblemThesis = Disputes{
	Subject: PatriciaChurchland,
	Object:  ChalmersHardProblemThesis,
	Prov:    patriciaChurchlandDisputesSource,
}

// ---------------------------------------------------------------------------
// Mirror-source-commitments correction:
//
// Kirk (1974), Campbell (1970), and Nagel (1970s) were previously wired as
// Proposes ChalmersHardProblemThesis. The sources commit only to these
// thinkers advancing zombie-like arguments that *preceded* Chalmers' 1994
// formulation — they did not propose his thesis. The Proposes claims were
// anachronistic fabrications from metabolism cycle 6 LLM-assisted ingest.
//
// Session 33 (panel R8): 3 instances is a forcing function per schema
// accretion. Added EarlyFormulationOf predicate — captures "advanced an
// earlier version of the argument" without implying direct proposal.
// ---------------------------------------------------------------------------

var robertKirkEarlyFormulationSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"it was philosopher Robert Kirk who first used the term 'zombie' in this context, in 1974.\"",
}

var RobertKirk = Person{&Entity{
	ID:    "robert-kirk",
	Name:  "Robert Kirk",
	Kind:  "person",
	Brief: "Philosopher who first introduced the term \"zombie\" in the philosophical context in 1974.",
}}

var keithCampbellEarlyFormulationSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Before that, Keith Campbell made a similar argument in his 1970 book Body and Mind, using the term 'imitation man'.\"",
}

var KeithCampbell = Person{&Entity{
	ID:    "keith-campbell",
	Name:  "Keith Campbell",
	Kind:  "person",
	Brief: "Philosopher who made an argument similar to the zombie argument in 1970.",
}}

var thomasNagelEarlyFormulationSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Further such arguments were notably advanced in the 1970s by Thomas Nagel (1970; 1974) and Robert Kirk (1974).\"",
}

var ThomasNagel = Person{&Entity{
	ID:    "thomas-nagel",
	Name:  "Thomas Nagel",
	Kind:  "person",
	Brief: "Philosopher who advanced arguments related to consciousness and the limits of physicalism in the 1970s.",
}}

// Era declarations are per-file: each corpus file declares the temporal
// markers it needs. No overlap with tunguska.go's Era* vars. If a future
// file needs the same era, import or hoist to a shared location.
var (
	Era1970 = &TemporalMarker{Era: "1970"}
	Era1974 = &TemporalMarker{Era: "1974"}
)

var RobertKirkEarlyFormulationOfChalmersHardProblemThesis = EarlyFormulationOf{
	Subject: RobertKirk,
	Object:  ChalmersHardProblemThesis,
	When:    Era1974,
	Prov:    robertKirkEarlyFormulationSource,
}

var KeithCampbellEarlyFormulationOfChalmersHardProblemThesis = EarlyFormulationOf{
	Subject: KeithCampbell,
	Object:  ChalmersHardProblemThesis,
	When:    Era1970,
	Prov:    keithCampbellEarlyFormulationSource,
}

var ThomasNagelEarlyFormulationOfChalmersHardProblemThesis = EarlyFormulationOf{
	Subject: ThomasNagel,
	Object:  ChalmersHardProblemThesis,
	When:    Era1974,
	Prov:    thomasNagelEarlyFormulationSource,
}

// ---------------------------------------------------------------------------
// Metabolism cycle 2–5 ingest: disputants and proposers surfaced by
// topology-driven sensor queries against Wikipedia ZIM. All claims below
// were originally in metabolism_cycle{2,3,4,5}.go and consolidated here
// because they all target ChalmersHardProblemThesis or related entities.
// ---------------------------------------------------------------------------

var dennettDisputeSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
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

var pinkerHumanUniversalsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Human_Universals",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 2 (sensor: topology-driven ZIM query " +
		"'Brown Human Universals', corroboration ingest)",
	Quote: "Steven Pinker lists all Brown's universals in the appendix of " +
		"his book The Blank Slate. The list is seen by Brown (and Pinker) " +
		"to be evidence of mental adaptations to communal life in our " +
		"species' evolutionary history.",
}

var DanielDennett = Person{&Entity{
	ID:      "daniel-dennett",
	Name:    "Daniel Dennett",
	Kind:    "person",
	Aliases: []string{"Dennett"},
	Brief:   "American philosopher and cognitive scientist (1942–2024) who argued that consciousness can be fully explained through functional and mechanistic analysis of the brain, rejecting the hard problem of consciousness.",
}}

var DennettDisputesHardProblemThesis = Disputes{
	Subject: DanielDennett,
	Object:  ChalmersHardProblemThesis,
	Prov:    dennettDisputeSource,
}

var PinkerProposesBrownHumanUniversalsThesis = Proposes{
	Subject: StevenPinker,
	Object:  BrownHumanUniversalsThesis,
	Prov:    pinkerHumanUniversalsSource,
}

// Cycle 3: Anil Seth disputes ChalmersHardProblemThesis

var anilSethDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 3 (LLM-assisted ingest from ZIM)",
	Quote:      "\"On the other hand, its existence is rejected by other philosophers of mind, such as Daniel Dennett, Massimo Pigliucci, Thomas Metzinger, Patricia Churchland, and Keith Frankish, and by cognitive neuroscientists such as Stanislas Dehaene, Bernard Baars, Anil Seth, and Antonio Damasio.\"",
}

var AnilSeth = Person{&Entity{
	ID:    "anil-seth",
	Name:  "Anil Seth",
	Kind:  "person",
	Brief: "Cognitive neuroscientist who rejects the existence of the hard problem of consciousness. Professor of Cognitive and Computational Neuroscience at the University of Sussex.",
}}

var AnilSethDisputesChalmersHardProblemThesis = Disputes{
	Subject: AnilSeth,
	Object:  ChalmersHardProblemThesis,
	Prov:    anilSethDisputesSource,
}

// Cycle 3: Christopher Hill disputes via zombie argument

var christopherHillDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 3 (LLM-assisted ingest from ZIM)",
	Quote:      "\"others, such as Christopher Hill, argue that philosophical zombies are coherent but metaphysically impossible.\"",
}

var ChristopherHill = Person{&Entity{
	ID:    "christopher-hill",
	Name:  "Christopher Hill",
	Kind:  "person",
	Brief: "Philosopher who argues that philosophical zombies are coherent but metaphysically impossible.",
}}

var ChristopherHillDisputesChalmersHardProblemThesis = Disputes{
	Subject: ChristopherHill,
	Object:  ChalmersHardProblemThesis,
	Prov:    christopherHillDisputesSource,
}

// Cycle 4: Bernard Baars disputes ChalmersHardProblemThesis

var bernardBaarsDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 4 (LLM-assisted ingest from ZIM)",
	Quote:      "\"On the other hand, its existence is rejected by other philosophers of mind, such as Daniel Dennett, Massimo Pigliucci, Thomas Metzinger, Patricia Churchland, and Keith Frankish, and by cognitive neuroscientists such as Stanislas Dehaene, Bernard Baars, Anil Seth, and Antonio Damasio.\"",
}

var BernardBaars = Person{&Entity{
	ID:    "bernard-baars",
	Name:  "Bernard Baars",
	Kind:  "person",
	Brief: "Cognitive neuroscientist who rejects the existence of the hard problem of consciousness.",
}}

var BernardBaarsDisputesChalmersHardProblemThesis = Disputes{
	Subject: BernardBaars,
	Object:  ChalmersHardProblemThesis,
	Prov:    bernardBaarsDisputesSource,
}

// Cycle 4: Galen Strawson disputes via zombie conceivability

var galenStrawsonDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze metabolism cycle 4 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Galen Strawson argues that it is not possible to establish the conceivability of zombies, so the argument, lacking its first premise, can never get going.\"",
}

var GalenStrawson = Person{&Entity{
	ID:    "galen-strawson",
	Name:  "Galen Strawson",
	Kind:  "person",
	Brief: "A philosopher who argues against the conceivability of philosophical zombies.",
}}

var GalenStrawsonDisputesChalmersHardProblemThesis = Disputes{
	Subject: GalenStrawson,
	Object:  ChalmersHardProblemThesis,
	Prov:    galenStrawsonDisputesSource,
}

// Cycle 5: Massimo Pigliucci disputes ChalmersHardProblemThesis

var massimoPigliucciDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 5 (LLM-assisted ingest from ZIM)",
	Quote:      "\"On the other hand, its existence is rejected by other philosophers of mind, such as Daniel Dennett, Massimo Pigliucci, Thomas Metzinger, Patricia Churchland, and Keith Frankish, and by cognitive neuroscientists such as Stanislas Dehaene, Bernard Baars, Anil Seth, and Antonio Damasio.\"",
}

var MassimoPigliucci = Person{&Entity{
	ID:    "massimo-pigliucci",
	Name:  "Massimo Pigliucci",
	Kind:  "person",
	Brief: "Philosopher of mind who rejects the existence of the hard problem of consciousness.",
}}

var MassimoPigliucciDisputesChalmersHardProblemThesis = Disputes{
	Subject: MassimoPigliucci,
	Object:  ChalmersHardProblemThesis,
	Prov:    massimoPigliucciDisputesSource,
}

// Cycle 5: Paul Churchland disputes via The Conscious Mind

var paulChurchlandDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Conscious_Mind",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 5 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Patricia and Paul Churchland have criticised Chalmers claim that everything but consciousness logically supervenes on the physical, and that such failures of supervenience mean that materialism must be false.\"",
}

var PaulChurchland = Person{&Entity{
	ID:    "paul-churchland",
	Name:  "Paul Churchland",
	Kind:  "person",
	Brief: "Philosopher who has criticized Chalmers' arguments about consciousness and materialism.",
}}

var PaulChurchlandDisputesChalmersHardProblemThesis = Disputes{
	Subject: PaulChurchland,
	Object:  ChalmersHardProblemThesis,
	Prov:    paulChurchlandDisputesSource,
}

// Cycle 5: Nigel J. T. Thomas disputes via zombie conceivability

var nigelJTThomasDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 5 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Critics who primarily argue that zombies are not conceivable include Daniel Dennett, Nigel J. T. Thomas, David Braddon-Mitchell, and Robert Kirk.\"",
}

var NigelJTThomas = Person{&Entity{
	ID:    "nigel-j-t-thomas",
	Name:  "Nigel J. T. Thomas",
	Kind:  "person",
	Brief: "Philosopher who critiques philosophical zombie arguments.",
}}

var NigelJTThomasDisputesChalmersHardProblemThesis = Disputes{
	Subject: NigelJTThomas,
	Object:  ChalmersHardProblemThesis,
	Prov:    nigelJTThomasDisputesSource,
}

// ---------------------------------------------------------------------------
// Hard problem of consciousness: Christof Koch Accepts ChalmersHardProblemThesis
// Koch's acceptance of the hard problem reflects his acknowledgment that subjective experience cannot be fully explained by physical neural mechanisms alone.
// ---------------------------------------------------------------------------

var christofKochAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Hard_problem_of_consciousness",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been accepted by some philosophers of mind such as Joseph Levine, Colin McGinn, and Ned Block and cognitive neuroscientists such as Francisco Varela, Giulio Tononi, and Christof Koch.\"",
}

var ChristofKoch = Person{&Entity{
	ID:    "christof-koch",
	Name:  "Christof Koch",
	Kind:  "person",
	Brief: "Cognitive neuroscientist who accepts the hard problem of consciousness.",
}}

var ChristofKochAcceptsChalmersHardProblemThesis = Accepts{
	Subject: ChristofKoch,
	Object:  ChalmersHardProblemThesis,
	Prov:    christofKochAcceptsSource,
}

// ---------------------------------------------------------------------------
// David Chalmers: Douglas Hofstadter InfluencedBy DavidChalmers
// This documents how a mind (Chalmers) built its model of philosophical reality through the influence of another thinker's work on consciousness and cognition.
// ---------------------------------------------------------------------------

var douglasHofstadterInfluencedBySource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / David_Chalmers",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "When Chalmers was 13, he read Douglas Hofstadter's 1979 book Gödel, Escher, Bach, which awakened an interest in philosophy.",
}

var DouglasHofstadter = Person{&Entity{
	ID:    "douglas-hofstadter",
	Name:  "Douglas Hofstadter",
	Kind:  "person",
	Brief: "Cognitive scientist and author who influenced Chalmers's philosophical interests.",
}}

var DouglasHofstadterInfluencedByDavidChalmers = InfluencedBy{
	Subject: DouglasHofstadter,
	Object:  DavidChalmers,
	Prov:    douglasHofstadterInfluencedBySource,
}

// ---------------------------------------------------------------------------
// The Conscious Mind: David Lewis Accepts ChalmersHardProblemThesis
// Shows recognition that the hard problem thesis successfully models consciousness in ways that challenge standard physical reduction theories.
// ---------------------------------------------------------------------------

var davidLewisAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Conscious_Mind",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Lewis praises Chalmers for his understanding of the issue and for leaving his critics with 'few points to make' that Chalmers 'hasn't made already'. Lewis has characterised The Conscious Mind as 'exceptionally ambitious and exceptionally successful', considering it 'the best book in philosophy of mind for many years.'\"",
}

var DavidLewis = Person{&Entity{
	ID:    "david-lewis",
	Name:  "David Lewis",
	Kind:  "person",
	Brief: "Proponent of materialism who engaged with Chalmers' arguments in philosophy of mind.",
}}

var DavidLewisAcceptsChalmersHardProblemThesis = Accepts{
	Subject: DavidLewis,
	Object:  ChalmersHardProblemThesis,
	Prov:    davidLewisAcceptsSource,
}

// ---------------------------------------------------------------------------
// Philosophical zombie: Daniel Stoljar Accepts ChalmersHardProblemThesis
// Stoljar accepts and elaborates upon the zombie-based argument against physicalism, a key defense of the hard problem thesis regarding consciousness.
// ---------------------------------------------------------------------------

var danielStoljarAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Philosopher Daniel Stoljar points out that zombies need not be utterly without subjective states, and that even a subtle psychological difference between two hypothetically physically identical people, such as how coffee tastes to them, is enough to refute physicalism.\"",
}

var DanielStoljar = Person{&Entity{
	ID:    "daniel-stoljar",
	Name:  "Daniel Stoljar",
	Kind:  "person",
	Brief: "Philosopher who argues for a refined version of the zombie argument.",
}}

var DanielStoljarAcceptsChalmersHardProblemThesis = Accepts{
	Subject: DanielStoljar,
	Object:  ChalmersHardProblemThesis,
	Prov:    danielStoljarAcceptsSource,
}

// ---------------------------------------------------------------------------
// Philosophical zombie: Amy Kind Accepts ChalmersHardProblemThesis
// Kind accepts and provides an analogical framework for understanding how zombie arguments defend the hard problem thesis about consciousness.
// ---------------------------------------------------------------------------

var amyKindAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Philosophical_zombie",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "\"The philosophical zombie argument can also be seen through the counterfeit bill example brought forth by Amy Kind. Kind's example centers around a counterfeit 20-dollar bill made to be exactly like an authentic 20-dollar bill... the zombie argument can be put in this standard form from a dualist point of view.\"",
}

var AmyKind = Person{&Entity{
	ID:    "amy-kind",
	Name:  "Amy Kind",
	Kind:  "person",
	Brief: "Philosopher who provides the counterfeit bill analogy for the zombie argument.",
}}

var AmyKindAcceptsChalmersHardProblemThesis = Accepts{
	Subject: AmyKind,
	Object:  ChalmersHardProblemThesis,
	Prov:    amyKindAcceptsSource,
}
