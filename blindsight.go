package winze

// Second fiction ingest: Wikipedia's article on Blindsight, Peter Watts's
// 2006 hard SF novel. Chosen specifically to test whether the fiction
// predicates (IsFictionalWork, IsFictional, AppearsIn) earned by the
// Quantum Thief slice generalise to a second author, a second fictional
// universe, and a structurally different novel (single book, no trilogy;
// alien-contact plot, not heist; consciousness-as-theme, not memory-as-
// theme).
//
// Schema forcing functions earned by this slice: NONE. All four fiction
// predicates (IsFictionalWork, IsFictional, AppearsIn, Authored) plus
// the thesis predicates (Proposes, TheoryOf) carry the content without
// conflation. This is the first fiction-neighbourhood vocabulary-fit
// result: the predicates designed for The Quantum Thief work unchanged
// on Blindsight.
//
// Novel finding: fiction as a vehicle for real philosophical theses.
// The Wikipedia article's Consciousness section, sourced with multiple
// academic citations, commits to Watts advancing the thesis that
// consciousness may be an evolutionary dead end — not as in-fiction
// speculation but as a genuine philosophical position explored through
// fiction. This earns a real-world Hypothesis and a TheoryOf claim
// targeting a new Consciousness Concept, demonstrating that the
// fiction/non-fiction boundary in winze is about the CLAIMS, not the
// FILES. Peter Watts carries both Authored (fiction authorship) and
// Proposes (philosophical thesis) in the same file on the same Person
// entity.
//
// Cross-file bridges: none at claim level. Brief-level adjacency to
// predictive_processing.go (Clark's predictive processing is a theory
// of cognition/consciousness) and mattson_pattern_processing.go
// (Mattson's "superior pattern processing" is thematically adjacent to
// the Scramblers' "orders of magnitude more brainpower" without
// consciousness). No bridge is wired because the Wikipedia article
// does not cite Clark, Mattson, Shermer, or any other existing winze
// Person.
//
// Consciousness is seeded as a contested-target-ready Concept with one
// TheoryOf claim, parallel to HumanCognition and HumanRights. A rival
// theory (Chalmers hard problem, Dennett eliminativism, Clark's
// predictive-processing account) would fire contested-concept as the
// seventh contested target.

var blindsightSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Blindsight_(Watts_novel)",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 6 (blindsight first slice)",
	Quote: "Blindsight is a hard science fiction novel by Canadian writer Peter Watts, published by Tor Books in 2006. " +
		"The story follows a crew of astronauts sent to investigate a trans-Neptunian comet [...] " +
		"The novel explores themes of identity, consciousness, free will, artificial intelligence, neurology, and game theory as well as evolution and biology. " +
		"[...] The exploration of consciousness is the central thematic element of Blindsight. " +
		"The title of the novel refers to the condition blindsight, in which vision is non-functional in the conscious brain but remains useful to non-conscious action. " +
		"[...] the possibility is raised that consciousness is, for humanity, an evolutionary dead end. " +
		"That is, consciousness may have been naturally selected as a solution for the challenges of a specific place in space and time, " +
		"but will become a limitation as conditions change or competing intelligences are encountered. " +
		"[...] The alien creatures encountered by the crew of the Theseus themselves lack consciousness. " +
		"[...] they are more akin to something like white blood cells in a human body. " +
		"They are dependent on the radiation and EM fields of Rorschach for basic biological functions and seem to completely lack consciousness. " +
		"[...] Philosopher John Searle's Chinese room thought experiment is used as a metaphor to illustrate " +
		"the tension between the notions of consciousness as an interior experience of understanding, " +
		"as contrasted with consciousness as the emergent result of merely functional non-introspective components. " +
		"[...] Blindsight contributes to this debate by implying that some aspects of consciousness are empirically detectable. " +
		"Specifically, the novel supposes that consciousness is necessary for both aesthetic appreciation and for effective communication.",
}

// -----------------------------------------------------------------------------
// Real-world entities. The book is a Concept tagged IsFictionalWork; the
// author is a Person who carries BOTH Authored (fiction authorship) and
// Proposes (philosophical thesis advancement) — demonstrating that these
// predicates coexist cleanly on a single Person entity. Consciousness is
// a real-world Concept seeded as a contested-target-ready TheoryOf object.
// -----------------------------------------------------------------------------

var (
	BlindsightNovel = Concept{&Entity{
		ID:      "concept-blindsight-novel",
		Name:    "Blindsight",
		Kind:    "concept",
		Aliases: []string{"Blindsight (novel)"},
		Brief: "Peter Watts's 2006 hard science fiction novel, published by Tor Books. " +
			"A first-contact story in which a crew of transhuman specialists investigates " +
			"an alien presence in the Oort cloud and discovers that consciousness may not " +
			"be necessary for — or even compatible with — intelligence. Won the Seiun Award, " +
			"nominated for the Hugo and Locus Awards. Sequel/sidequel Echopraxia (2014) " +
			"continues the Firefall series but is not ingested in this slice. The book as " +
			"a real-world creative work is tagged IsFictionalWork; the in-fiction entities " +
			"it introduces are tagged IsFictional and anchored via AppearsIn.",
	}}

	PeterWatts = Person{&Entity{
		ID:    "peter-watts",
		Name:  "Peter Watts",
		Kind:  "person",
		Brief: "Canadian science fiction writer and marine biologist, author of Blindsight (2006) " +
			"and its sequel Echopraxia (2014). Holds a PhD in marine biology. Known for " +
			"rigorously researched hard SF that engages directly with neuroscience and " +
			"philosophy of mind — the Wikipedia article's Major Themes section, sourced " +
			"with multiple academic citations, treats his consciousness thesis as a genuine " +
			"philosophical position explored through fiction rather than pure in-world " +
			"speculation. This makes him the first winze Person entity to carry both " +
			"Authored (fiction) and Proposes (philosophical thesis) simultaneously.",
	}}

	Consciousness = Concept{&Entity{
		ID:   "concept-consciousness",
		Name: "Consciousness",
		Kind: "concept",
		Brief: "The philosophical and scientific question of what consciousness is, whether " +
			"it is necessary for intelligence, and what role it plays in evolution and " +
			"cognition. Seeded as a contested-target-ready Concept by the Blindsight slice " +
			"with one TheoryOf claim (Watts's evolutionary-dead-end thesis). A rival theory " +
			"(Chalmers hard problem, Dennett eliminativism, Clark/Hohwy predictive-processing " +
			"account, IIT) would fire contested-concept. Adjacent to but distinct from " +
			"HumanCognition in mattson_pattern_processing.go — HumanCognition is about " +
			"what makes human cognition special; Consciousness is about what consciousness " +
			"itself is and whether it is adaptive.",
	}}

	WattsConsciousnessAsDeadEndThesis = Hypothesis{&Entity{
		ID:   "hypothesis-watts-consciousness-dead-end",
		Name: "Watts consciousness-as-evolutionary-dead-end thesis",
		Kind: "hypothesis",
		Brief: "The thesis, advanced by Peter Watts through Blindsight, that consciousness " +
			"may have been naturally selected as a solution for the challenges of a specific " +
			"place in space and time, but will become a limitation as conditions change or " +
			"competing intelligences are encountered. The Wikipedia article commits to this " +
			"at the source level via multiple academic citations (Shaviro, McGrath, Elber-Aviram, " +
			"Science Fiction First podcast) confirming it as the novel's central thesis. Not " +
			"an in-fiction hypothesis — it is a real philosophical position that Watts explores " +
			"through fictional scenario rather than asserting in a journal paper.",
	}}
)

// -----------------------------------------------------------------------------
// In-fiction entities. All are Concepts tagged IsFictional; each is anchored
// to BlindsightNovel via AppearsIn. Scope is tight: four entities that are
// thematically central to the consciousness thesis. Characters who serve
// primarily as plot apparatus (Amanda Bates, Isaac Szpindel, Robert
// Cunningham, the Captain AI) are omitted — not parasitic (they would carry
// IsFictional + AppearsIn) but not thematically load-bearing for the claims
// this slice wires.
// -----------------------------------------------------------------------------

var (
	SiriKeeton = Concept{&Entity{
		ID:    "concept-siri-keeton",
		Name:  "Siri Keeton",
		Kind:  "concept",
		Brief: "In-fiction character: the narrator and protagonist of Blindsight. " +
			"Debilitating brain surgery for medical purposes has cut him off from his " +
			"own emotional life and made him a talented 'synthesist', adept at reading " +
			"others' intentions impartially with the aid of cybernetics. His diminished " +
			"consciousness is the novel's primary vehicle for exploring the question of " +
			"whether empathic behaviour suffices without interior emotional experience. " +
			"Tagged IsFictional.",
	}}

	JukkaSarasti = Concept{&Entity{
		ID:    "concept-jukka-sarasti",
		Name:  "Jukka Sarasti",
		Kind:  "concept",
		Brief: "In-fiction character: a genetically reincarnated Pleistocene vampire " +
			"serving as the crew's nominal commander aboard Theseus. Alleged to be far " +
			"smarter than baseline humans, with 'diminished sentience presented as " +
			"comparable to high-functional autism' and multiple simultaneous parallel " +
			"thoughts. Near the climax he is revealed to have been controlled by the " +
			"ship's AI for the entirety of the mission — a plot twist that dramatises " +
			"the Chinese room problem at character scale. Tagged IsFictional.",
	}}

	RorschachAlien = Concept{&Entity{
		ID:      "concept-rorschach",
		Name:    "Rorschach",
		Kind:    "concept",
		Aliases: []string{"Rorschach (alien vessel)"},
		Brief: "In-fiction entity: a giant concealed alien vessel or organism in low " +
			"orbit around a sub-brown dwarf in the Oort cloud. Possesses superhuman " +
			"intelligence but gradually revealed to completely lack consciousness or " +
			"self-awareness. Communicates in human languages learned by eavesdropping " +
			"on radio, but the linguist determines it 'does not really understand what " +
			"either party is actually saying' — the novel's central dramatisation of " +
			"the Chinese room. Tagged IsFictional.",
	}}

	Scramblers = Concept{&Entity{
		ID:    "concept-scramblers",
		Name:  "Scramblers",
		Kind:  "concept",
		Brief: "In-fiction organisms: nine-legged anaerobic aliens inhabiting Rorschach. " +
			"Possess 'orders of magnitude more brainpower than human beings' but use " +
			"most of it to operate their fantastically complex musculature and sensory " +
			"organs. Completely lack consciousness — thematically adjacent to Mattson's " +
			"'superior pattern processing' without consciousness, though no claim-level " +
			"bridge is wired because the Wikipedia source does not cite Mattson. " +
			"Tagged IsFictional.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Real-world claims first (authorship, thesis, theory), then
// in-fiction tagging, then in-fiction anchoring.
// -----------------------------------------------------------------------------

var (
	// Real-world claims.

	WattsAuthoredBlindsight = Authored{
		Subject: PeterWatts,
		Object:  BlindsightNovel,
		Prov:    blindsightSource,
	}

	WattsProposes = Proposes{
		Subject: PeterWatts,
		Object:  WattsConsciousnessAsDeadEndThesis,
		Prov:    blindsightSource,
	}

	WattsConsciousnessTheory = TheoryOf{
		Subject: WattsConsciousnessAsDeadEndThesis,
		Object:  Consciousness,
		Prov:    blindsightSource,
	}

	// In-fiction tags.

	BlindsightIsFictionalWork = IsFictionalWork{
		Subject: BlindsightNovel,
		Prov:    blindsightSource,
	}

	SiriKeetonIsFictional = IsFictional{
		Subject: SiriKeeton,
		Prov:    blindsightSource,
	}
	JukkaSarastiIsFictional = IsFictional{
		Subject: JukkaSarasti,
		Prov:    blindsightSource,
	}
	RorschachIsFictional = IsFictional{
		Subject: RorschachAlien,
		Prov:    blindsightSource,
	}
	ScramblersIsFictional = IsFictional{
		Subject: Scramblers,
		Prov:    blindsightSource,
	}

	// In-fiction anchoring via AppearsIn.

	SiriKeetonAppearsInBlindsight = AppearsIn{
		Subject: SiriKeeton,
		Object:  BlindsightNovel,
		Prov:    blindsightSource,
	}
	JukkaSarastiAppearsInBlindsight = AppearsIn{
		Subject: JukkaSarasti,
		Object:  BlindsightNovel,
		Prov:    blindsightSource,
	}
	RorschachAppearsInBlindsight = AppearsIn{
		Subject: RorschachAlien,
		Object:  BlindsightNovel,
		Prov:    blindsightSource,
	}
	ScramblersAppearsInBlindsight = AppearsIn{
		Subject: Scramblers,
		Object:  BlindsightNovel,
		Prov:    blindsightSource,
	}
)
