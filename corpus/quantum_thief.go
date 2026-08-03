package winze

// Sixth public-corpus ingest: Wikipedia's article on The Quantum Thief,
// Hannu Rajaniemi's 2010 debut SF novel. Chosen specifically for the
// in-world-fact vs real-world-fact forcing function: the Wikipedia
// article asserts two structurally different kinds of claim in the
// same document, and winze's existing predicate vocabulary had no
// way to keep them straight before this slice.
//
//   - Real-world facts about the book: Rajaniemi authored it, Gollancz
//     published it in 2010, it was inspired by Maurice Leblanc's
//     Arsène Lupin, it was nominated for the 2011 Locus Award. These
//     are ordinary claims about a real-world creative artefact.
//
//   - In-fiction facts from within the book: Jean le Flambeur is a
//     gentleman thief trapped in a Sobornost prison; the Oubliette is
//     a mobile Martian city where time is currency; exomemory is a
//     shared-memory substrate. These are NOT real-world claims.
//     Encoding them with the ordinary predicates (LocatedIn,
//     WorksFor, etc.) would be a category error — a query for
//     "cities" should not return the Oubliette alongside Moscow.
//
// Schema forcing functions earned by this slice:
//
//   - `IsFictionalWork UnaryClaim[Concept]` — the "real-world tag"
//     half of the in-fiction/real-world split. The book as an
//     artefact exists in the real world.
//
//   - `IsFictional UnaryClaim[Concept]` — the "diegetic tag" half.
//     Any claim about an IsFictional subject is to be read "within
//     the fiction".
//
//   - `AppearsIn BinaryRelation[Concept, Concept]` — anchors an
//     in-fiction entity back to the work it exists in. Not
//     functional: sequels come free.
//
//   - `Authored BinaryRelation[Person, Concept]` — real-world
//     authorship. Distinct from Proposes because fiction is
//     constructed, not asserted-true-of-the-world.
//
// No new role types, no new pragmas, no new functional predicates.
// Unreliable-narrator handling is deliberately deferred — a follow-up
// slice can encode contested in-fiction claims (is the King of Mars
// really Jean le Flambeur? who cryptographically compromised the
// Oubliette?) through existing Hypothesis + TheoryOf +
// //winze:contested machinery with zero new primitives.
//
// Slice scope: one book (IsFictionalWork), one real-world author
// (Person), four in-fiction concepts each tagged IsFictional and
// anchored via AppearsIn. Sequels (The Fractal Prince, The Causal
// Angel), additional characters (Mieli, Isidore Beautrelet), and
// more invented vocabulary (Quiet, cryptarchs, gogols, Zoku) are all
// zero-schema follow-ups for future slices.
//
// Dual-purpose note: this ingest is also long-term architectural
// inspiration for the Gas Town reasoning-agent layer. The invented-
// vocabulary concepts dramatise relation shapes (many-copy identity,
// shared-memory substrates, time-as-social-currency) that a
// multi-agent society may eventually need to model — having them
// encoded as typed concepts gives the agent layer something to point
// at instead of free text.

var quantumThiefSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Quantum_Thief",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "The Quantum Thief is the debut science fiction novel by Finnish writer Hannu Rajaniemi and the first novel in a trilogy featuring the character of Jean le Flambeur [...] A warrior from the Oort Cloud [...] successfully retrieves one of the Le Flambeur gogols and uploads it into a real-space body. Acting on behalf of a competing Sobornost authority, this Oortian, Mieli, ferries the thief to the Martian city known as The Oubliette [...] An alliance of powerful gogol copies rule the inner system from computronium megastructures [...] This alliance, the Sobornost, has been in conflict with a community of quantum entangled minds who adhere to the 'no-cloning' principle [...] Among the last remnants of near-baseline humanity exist on the mobile cities of Mars [...] The most notable of these cities is the Oubliette, where time is used as a currency. [...] In the book, the people living in the Oubliette society on Mars have two types of memory; in addition to a traditional, personal memory, there is the exomemory, which can be accessed by other people, from anywhere in the city.",
}

var fractalPrinceSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Fractal_Prince",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "The Fractal Prince is the second science fiction novel by Hannu Rajaniemi and the second novel to feature the post-human gentleman thief Jean le Flambeur. It was published in Britain by Gollancz in September 2012 [...] After the events of The Quantum Thief, Jean le Flambeur and Mieli are on their way to Earth. Jean is trying to open the Schrödinger's Box he retrieved from the memory palace on the Oubliette. After making little progress, he is prodded by the ship Perhonen to talk to Mieli, who turns out to be possessed by the pellegrini again.",
}

var causalAngelSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Causal_Angel",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "The Causal Angel is the third science fiction novel by Hannu Rajaniemi featuring the protagonist Jean le Flambeur. It was published in July 2014 by Gollancz in the UK and by Tor in the US. The novel is the finale of a trilogy. [...] After the events of The Fractal Prince, Jean le Flambeur and Mieli are separated and their sentient spacecraft Perhonen is destroyed. [...] The most powerful factions, the Sobornost and the Zoku, are gathering their forces and making their plays, while simultaneously being torn apart by internal strifes.",
}

// -----------------------------------------------------------------------------
// Real-world entities. The book exists as a Concept tagged IsFictionalWork;
// its author is a Person. No IsFictional tag on the book itself — the book
// is a real creative artefact, the *contents* of the book are what is
// fictional.
// -----------------------------------------------------------------------------

var (
	TheQuantumThief = Concept{&Entity{
		ID:      "concept-the-quantum-thief",
		Name:    "The Quantum Thief",
		Kind:    "concept",
		Aliases: []string{"Quantum Thief"},
		Brief:   "Science fiction debut novel by Hannu Rajaniemi (2010), first in the Jean le Flambeur trilogy. A post-singularity heist story set in the Solar System.",
	}}

	TheFractalPrince = Concept{&Entity{
		ID:      "concept-the-fractal-prince",
		Name:    "The Fractal Prince",
		Kind:    "concept",
		Aliases: []string{"Fractal Prince"},
		Brief:   "Second novel in Rajaniemi's Jean le Flambeur trilogy, following Jean and Mieli's attempt to open Schrödinger's Box on a wildcode-ravaged Earth, with frame narrative inspired by The Arabian Nights.",
	}}

	TheCausalAngel = Concept{&Entity{
		ID:      "concept-the-causal-angel",
		Name:    "The Causal Angel",
		Kind:    "concept",
		Aliases: []string{"Causal Angel"},
		Brief:   "The final novel in Rajaniemi's Jean le Flambeur trilogy. Jean and Mieli are separated as their spacecraft Perhonen is destroyed amid escalating war between the Sobornost and Zoku factions.",
	}}

	JeanLeFlambeurSeries = Concept{&Entity{
		ID:      "concept-jean-le-flambeur-series",
		Name:    "Jean le Flambeur series",
		Kind:    "concept",
		Aliases: []string{"Jean le Flambeur trilogy"},
		Brief:   "Science fiction trilogy by Hannu Rajaniemi following post-human gentleman thief Jean le Flambeur across three novels: The Quantum Thief (2010), The Fractal Prince (2012), and The Causal Angel (2014).",
	}}

	HannuRajaniemi = Person{&Entity{
		ID:    "hannu-rajaniemi",
		Name:  "Hannu Rajaniemi",
		Kind:  "person",
		Brief: "Finnish science fiction writer and mathematician known for the Jean le Flambeur trilogy (2010-2014), which blends heist narratives with complex philosophical ideas about memory and consciousness.",
	}}

	// Real-world inspiration for Jean le Flambeur. Wired as a claim via
	// InfluencedBy rather than left at brief-level reference because the
	// Wikipedia article makes the Lupin → Le Flambeur modelling explicit
	// ("a protagonist modeled on Arsène Lupin, the gentleman thief of
	// Maurice Leblanc"), which is exactly the standard of commitment that
	// earns a Leblanc entity its place in the graph without fabrication.
	MauriceLeblanc = Person{&Entity{
		ID:    "maurice-leblanc",
		Name:  "Maurice Leblanc",
		Kind:  "person",
		Brief: "French novelist (1864–1941) who created the gentleman thief Arsène Lupin, a foundational character in crime fiction.",
	}}
)

// -----------------------------------------------------------------------------
// In-fiction entities. All are Concepts tagged IsFictional; each is anchored
// to TheQuantumThief via an AppearsIn claim. A second slice can add claims
// from the sequels by reusing AppearsIn with new fictional-work objects —
// zero schema work.
// -----------------------------------------------------------------------------

var (
	JeanLeFlambeur = Concept{&Entity{
		ID:      "concept-jean-le-flambeur",
		Name:    "Jean le Flambeur",
		Kind:    "concept",
		Aliases: []string{"le Flambeur"},
		Brief:   "Legendary gentleman thief character in The Quantum Thief, imprisoned in a virtual Sobornost prison where countless copies of him play iterated prisoner's dilemma until learning cooperation.",
	}}

	Oubliette = Concept{&Entity{
		ID:    "concept-oubliette-mars",
		Name:  "The Oubliette",
		Kind:  "concept",
		Brief: "A fictional Martian city where time serves as currency and cryptography protects baseline humans from forced uploading by the Sobornost.",
	}}

	Sobornost = Concept{&Entity{
		ID:    "concept-sobornost",
		Name:  "The Sobornost",
		Kind:  "concept",
		Brief: "In-fiction faction of gogol brain-emulation copies ruling the inner Solar System, devoted to resurrecting the dead through Fedorov's philosophy and in conflict with the Zoku over cloning ethics.",
	}}

	Exomemory = Concept{&Entity{
		ID:    "concept-exomemory",
		Name:  "Exomemory",
		Kind:  "concept",
		Brief: "Shared memory system in the Oubliette accessible citywide with customizable access levels, used for communication and subject to manipulation by the cryptarchs in the central plot conspiracy.",
	}}

	Mieli = Concept{&Entity{
		ID:    "concept-mieli",
		Name:  "Mieli",
		Kind:  "concept",
		Brief: "Oortian warrior in The Quantum Thief trilogy who ferries Jean le Flambeur from Sobornost Dilemma Prison, acting for Joséphine Pellegrini.",
	}}

	Perhonen = Concept{&Entity{
		ID:    "concept-perhonen",
		Name:  "Perhonen",
		Kind:  "concept",
		Brief: "Sentient spacecraft in Jean le Flambeur trilogy, serving as Mieli's ship. Destroyed at the opening of The Causal Angel's finale.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Real-world claims first, then the in-fiction tagging and anchoring.
// -----------------------------------------------------------------------------

var (
	TheQuantumThiefIsFictionalWork = IsFictionalWork{
		Subject: TheQuantumThief,
		Prov:    quantumThiefSource,
	}
	TheFractalPrinceIsFictionalWork = IsFictionalWork{
		Subject: TheFractalPrince,
		Prov:    fractalPrinceSource,
	}
	TheCausalAngelIsFictionalWork = IsFictionalWork{
		Subject: TheCausalAngel,
		Prov:    causalAngelSource,
	}
	JeanLeFlambeurSeriesIsFictionalWork = IsFictionalWork{
		Subject: JeanLeFlambeurSeries,
		Prov:    quantumThiefSource,
	}

	RajaniemiAuthoredTheQuantumThief = Authored{
		Subject: HannuRajaniemi,
		Object:  TheQuantumThief,
		Prov:    quantumThiefSource,
	}
	RajaniemiAuthoredTheFractalPrince = Authored{
		Subject: HannuRajaniemi,
		Object:  TheFractalPrince,
		Prov:    fractalPrinceSource,
	}
	RajaniemiAuthoredTheCausalAngel = Authored{
		Subject: HannuRajaniemi,
		Object:  TheCausalAngel,
		Prov:    causalAngelSource,
	}

	// Each book BelongsTo the series. Reuses the BelongsTo predicate from
	// the cognitive-biases ingest, validating that taxonomic-membership
	// applies cleanly to creative-work series as well as cognitive-bias
	// families with zero schema change.
	TQTBelongsToSeries = BelongsTo{
		Subject: TheQuantumThief,
		Object:  JeanLeFlambeurSeries,
		Prov:    quantumThiefSource,
	}
	TFPBelongsToSeries = BelongsTo{
		Subject: TheFractalPrince,
		Object:  JeanLeFlambeurSeries,
		Prov:    fractalPrinceSource,
	}
	TCABelongsToSeries = BelongsTo{
		Subject: TheCausalAngel,
		Object:  JeanLeFlambeurSeries,
		Prov:    causalAngelSource,
	}

	JeanLeFlambeurIsFictional = IsFictional{
		Subject: JeanLeFlambeur,
		Prov:    quantumThiefSource,
	}
	OublietteIsFictional = IsFictional{
		Subject: Oubliette,
		Prov:    quantumThiefSource,
	}
	SobornostIsFictional = IsFictional{
		Subject: Sobornost,
		Prov:    quantumThiefSource,
	}
	ExomemoryIsFictional = IsFictional{
		Subject: Exomemory,
		Prov:    quantumThiefSource,
	}
	MieliIsFictional = IsFictional{
		Subject: Mieli,
		Prov:    quantumThiefSource,
	}
	PerhonenIsFictional = IsFictional{
		Subject: Perhonen,
		Prov:    fractalPrinceSource,
	}

	// AppearsIn claims. Jean le Flambeur, Mieli, and the Sobornost each
	// appear in all three novels — three AppearsIn claims per subject,
	// distinct objects. Perhonen appears in only the two sequels (the
	// Wikipedia TQT article does not name Mieli's ship, so honesty
	// demands two claims rather than three). This is the primary stress
	// test for AppearsIn non-functionality: if the predicate were
	// accidentally flagged as //winze:functional the value-conflict
	// rule would explode on seven subjects at once.
	JeanLeFlambeurAppearsInTQT = AppearsIn{
		Subject: JeanLeFlambeur,
		Object:  TheQuantumThief,
		Prov:    quantumThiefSource,
	}
	JeanLeFlambeurAppearsInTFP = AppearsIn{
		Subject: JeanLeFlambeur,
		Object:  TheFractalPrince,
		Prov:    fractalPrinceSource,
	}
	JeanLeFlambeurAppearsInTCA = AppearsIn{
		Subject: JeanLeFlambeur,
		Object:  TheCausalAngel,
		Prov:    causalAngelSource,
	}

	MieliAppearsInTQT = AppearsIn{
		Subject: Mieli,
		Object:  TheQuantumThief,
		Prov:    quantumThiefSource,
	}
	MieliAppearsInTFP = AppearsIn{
		Subject: Mieli,
		Object:  TheFractalPrince,
		Prov:    fractalPrinceSource,
	}
	MieliAppearsInTCA = AppearsIn{
		Subject: Mieli,
		Object:  TheCausalAngel,
		Prov:    causalAngelSource,
	}

	PerhonenAppearsInTFP = AppearsIn{
		Subject: Perhonen,
		Object:  TheFractalPrince,
		Prov:    fractalPrinceSource,
	}
	PerhonenAppearsInTCA = AppearsIn{
		Subject: Perhonen,
		Object:  TheCausalAngel,
		Prov:    causalAngelSource,
	}

	SobornostAppearsInTQT = AppearsIn{
		Subject: Sobornost,
		Object:  TheQuantumThief,
		Prov:    quantumThiefSource,
	}
	SobornostAppearsInTFP = AppearsIn{
		Subject: Sobornost,
		Object:  TheFractalPrince,
		Prov:    fractalPrinceSource,
	}
	SobornostAppearsInTCA = AppearsIn{
		Subject: Sobornost,
		Object:  TheCausalAngel,
		Prov:    causalAngelSource,
	}

	// The Oubliette and exomemory are specific to the Mars plot of the
	// first book and are not named in the sequel Wikipedia summaries, so
	// they each have only one AppearsIn. A future slice that reads the
	// novels directly (rather than their Wikipedia stubs) could add
	// sequel appearances honestly — but this ingest is bounded by
	// Wikipedia content and refuses to fabricate beyond it.
	OublietteAppearsInTQT = AppearsIn{
		Subject: Oubliette,
		Object:  TheQuantumThief,
		Prov:    quantumThiefSource,
	}
	ExomemoryAppearsInTQT = AppearsIn{
		Subject: Exomemory,
		Object:  TheQuantumThief,
		Prov:    quantumThiefSource,
	}

	// Real-world intellectual-influence claims on Rajaniemi. The first
	// uses of winze's new InfluencedBy predicate, and the first real
	// cross-ingest bridge: the AndyClark target is defined in
	// predictive_processing.go (the PubMed 23663408 ingest), not in
	// this file. The non-executable discipline lets two slices meet at
	// a shared entity without either having to anticipate the other.
	RajaniemiInfluencedByLeblanc = InfluencedBy{
		Subject: HannuRajaniemi,
		Object:  MauriceLeblanc,
		Prov:    quantumThiefSource,
	}
	RajaniemiInfluencedByClark = InfluencedBy{
		Subject: HannuRajaniemi,
		Object:  AndyClark,
		Prov:    fractalPrinceSource,
	}
)
