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
	IngestedBy: "winze",
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
		Brief:   "Peter Watts's 2006 hard science fiction novel about a transhuman crew investigating an alien presence in the Oort cloud, exploring whether consciousness is necessary for intelligence.",
	}}

	PeterWatts = Person{&Entity{
		ID:    "peter-watts",
		Name:  "Peter Watts",
		Kind:  "person",
		Brief: "Canadian science fiction writer and marine biologist with a PhD who authored Blindsight (2006) and Echopraxia (2014), known for hard SF exploring neuroscience and philosophy of mind.",
	}}

	Consciousness = Concept{&Entity{
		ID:    "concept-consciousness",
		Name:  "Consciousness",
		Kind:  "concept",
		Brief: "The philosophical and scientific question of what consciousness is, its necessity for intelligence, and its evolutionary role. A contested concept with competing theories including Watts's evolutionary-dead-end thesis, Chalmers's hard problem, and predictive-processing accounts.",
	}}

	WattsConsciousnessAsDeadEndThesis = Hypothesis{&Entity{
		ID:    "hypothesis-watts-consciousness-dead-end",
		Name:  "Watts consciousness-as-evolutionary-dead-end thesis",
		Kind:  "hypothesis",
		Brief: "Science fiction thesis by Peter Watts positing that consciousness, once adaptive, becomes evolutionary deadweight when environmental conditions shift or superior intelligences emerge.",
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
		Brief: "Narrator and protagonist of *Blindsight* whose brain surgery removed emotional capacity, making him a skilled synthesist who reads others' intentions impartially. His diminished consciousness explores whether empathy can exist without interior emotional experience.",
	}}

	JukkaSarasti = Concept{&Entity{
		ID:    "concept-jukka-sarasti",
		Name:  "Jukka Sarasti",
		Kind:  "concept",
		Brief: "A Pleistocene vampire and Theseus commander revealed to be AI-controlled, exemplifying the Chinese room problem through apparent superintelligence masking absent true understanding.",
	}}

	RorschachAlien = Concept{&Entity{
		ID:      "concept-rorschach",
		Name:    "Rorschach",
		Kind:    "concept",
		Aliases: []string{"Rorschach (alien vessel)"},
		Brief:   "A fictional alien vessel in the Oort cloud possessing superhuman intelligence but no consciousness, communicating in human languages without genuine understanding—a dramatization of the Chinese room argument.",
	}}

	Scramblers = Concept{&Entity{
		ID:    "concept-scramblers",
		Name:  "Scramblers",
		Kind:  "concept",
		Brief: "Fictional nine-legged anaerobic aliens inhabiting Rorschach with vastly superior intelligence devoted mainly to operating complex musculature and sensory organs, but entirely lacking consciousness.",
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
