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
	IngestedBy: "winze session 4 (quantum thief first slice)",
	Quote:      "The Quantum Thief is the debut science fiction novel by Finnish writer Hannu Rajaniemi and the first novel in a trilogy featuring the character of Jean le Flambeur [...] A warrior from the Oort Cloud [...] successfully retrieves one of the Le Flambeur gogols and uploads it into a real-space body. Acting on behalf of a competing Sobornost authority, this Oortian, Mieli, ferries the thief to the Martian city known as The Oubliette [...] An alliance of powerful gogol copies rule the inner system from computronium megastructures [...] This alliance, the Sobornost, has been in conflict with a community of quantum entangled minds who adhere to the 'no-cloning' principle [...] Among the last remnants of near-baseline humanity exist on the mobile cities of Mars [...] The most notable of these cities is the Oubliette, where time is used as a currency. [...] In the book, the people living in the Oubliette society on Mars have two types of memory; in addition to a traditional, personal memory, there is the exomemory, which can be accessed by other people, from anywhere in the city.",
}

var fractalPrinceSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Fractal_Prince",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 4 (trilogy extension)",
	Quote:      "The Fractal Prince is the second science fiction novel by Hannu Rajaniemi and the second novel to feature the post-human gentleman thief Jean le Flambeur. It was published in Britain by Gollancz in September 2012 [...] After the events of The Quantum Thief, Jean le Flambeur and Mieli are on their way to Earth. Jean is trying to open the Schrödinger's Box he retrieved from the memory palace on the Oubliette. After making little progress, he is prodded by the ship Perhonen to talk to Mieli, who turns out to be possessed by the pellegrini again.",
}

var causalAngelSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / The_Causal_Angel",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 4 (trilogy extension)",
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
		Brief:   "Hannu Rajaniemi's 2010 debut science fiction novel, first in the Jean le Flambeur trilogy. A heist story set in a post-singularity Solar System, published in the UK by Gollancz and in the US by Tor. The book as a real-world creative work is tagged IsFictionalWork; the in-fiction entities it introduces (Jean le Flambeur, the Oubliette, the Sobornost, exomemory) are tagged IsFictional and anchored back to this book via AppearsIn.",
	}}

	TheFractalPrince = Concept{&Entity{
		ID:      "concept-the-fractal-prince",
		Name:    "The Fractal Prince",
		Kind:    "concept",
		Aliases: []string{"Fractal Prince"},
		Brief:   "Rajaniemi's 2012 second novel in the Jean le Flambeur trilogy, published by Gollancz (UK) and Tor (US). Opens with Jean and Mieli en route to Earth trying to open the Schrödinger's Box retrieved from the Oubliette, and introduces the wildcode-ravaged Earth city of Sirr, the Gomelez family (Tawaddud and Dunyazad), and a frame-story structure explicitly influenced by The Arabian Nights and Jan Potocki's Manuscript Found in Saragossa.",
	}}

	TheCausalAngel = Concept{&Entity{
		ID:      "concept-the-causal-angel",
		Name:    "The Causal Angel",
		Kind:    "concept",
		Aliases: []string{"Causal Angel"},
		Brief:   "Rajaniemi's 2014 third and final novel in the Jean le Flambeur trilogy, published by Gollancz (UK) and Tor (US). Opens with Jean and Mieli separated and the sentient spacecraft Perhonen destroyed, as the Solar System plummets into an all-out war between the Sobornost and the Zoku with each faction simultaneously torn apart by internal strifes.",
	}}

	JeanLeFlambeurSeries = Concept{&Entity{
		ID:      "concept-jean-le-flambeur-series",
		Name:    "Jean le Flambeur series",
		Kind:    "concept",
		Aliases: []string{"Jean le Flambeur trilogy"},
		Brief:   "The trilogy of Rajaniemi novels centred on the post-human gentleman thief Jean le Flambeur: The Quantum Thief (2010), The Fractal Prince (2012), and The Causal Angel (2014). Each book is a distinct IsFictionalWork and also BelongsTo this series concept — the same structural pattern cognitive_biases.go uses for individual biases belonging to task families, validated here for a second subject domain with zero schema change.",
	}}

	HannuRajaniemi = Person{&Entity{
		ID:    "hannu-rajaniemi",
		Name:  "Hannu Rajaniemi",
		Kind:  "person",
		Brief: "Finnish science fiction writer and mathematician, author of the Jean le Flambeur trilogy (The Quantum Thief 2010, The Fractal Prince 2012, The Causal Angel 2014). Stated in interviews that he set out to 'cram every idea' he had into his outline, which expanded into the three books that followed. Acknowledges multiple influences across the trilogy: Maurice Leblanc's Arsène Lupin (the explicit template for Jean le Flambeur), Roger Zelazny, Ian McDonald, Frances A. Yates's 'The Art of Memory' (for the memory-palace device), and in The Fractal Prince acknowledgments specifically Andy Clark and Douglas Hofstadter on the 'mind as self-loop' idea, and The Arabian Nights and Jan Potocki's 'Manuscript Found in Saragossa' on the frame-story structure of Sirr.",
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
		Brief: "French novelist (1864–1941), creator of the gentleman thief Arsène Lupin. Jean le Flambeur in Rajaniemi's trilogy is explicitly modelled on Lupin — 'what intrigued Rajaniemi were the cycles of redemption and relapse Lupin goes through as he tries to go straight, always falling short.' First real-world influence wired as a claim in winze, earning the new InfluencedBy predicate alongside the Clark→Rajaniemi cross-ingest bridge.",
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
		Brief:   "In-fiction character: a legendary gentleman thief modelled on Maurice Leblanc's Arsène Lupin, trapped at the novel's opening in a virtual Sobornost prison in orbit around Neptune where countless copies of him play an iterated prisoner's dilemma until his mind learns to cooperate. Tagged IsFictional because this concept exists only within the frame of The Quantum Thief (and its sequels).",
	}}

	Oubliette = Concept{&Entity{
		ID:    "concept-oubliette-mars",
		Name:  "The Oubliette",
		Kind:  "concept",
		Brief: "In-fiction place: a mobile Martian city among the last remnants of near-baseline humanity, where advanced cryptography and an obsessive privacy culture keep the Sobornost from uploading its citizens' minds. Time is the city's currency; when a citizen's balance reaches zero their mind is transferred into a Quiet (a robotic servitor body) for a fixed term before being returned. Encoded as a Concept rather than a Place role because the Place role carries real-world-location semantics (geolocation, formation dates, monitoring) that do not apply to a fictional city. Tagged IsFictional.",
	}}

	Sobornost = Concept{&Entity{
		ID:    "concept-sobornost",
		Name:  "The Sobornost",
		Kind:  "concept",
		Brief: "In-fiction faction: an alliance of powerful 'gogol' brain-emulation copies ruling the inner Solar System from computronium megastructures housing trillions of virtual minds, labouring to resurrect the dead in religious devotion to the philosophy of Nikolai Fedorov. In conflict with the Zoku, who adhere to the no-cloning principle of quantum information theory and so see the Sobornost's copying project as death rather than resurrection. Tagged IsFictional.",
	}}

	Exomemory = Concept{&Entity{
		ID:    "concept-exomemory",
		Name:  "Exomemory",
		Kind:  "concept",
		Brief: "In-fiction concept: the Oubliette's second, shared type of memory — accessible by other people from anywhere in the city, partitionable with per-person access levels, usable as a form of communication. The central plot conspiracy turns on the Oubliette's hidden rulers (the 'cryptarchs') manipulating and abusing exomemory and the citizens' transitions through Quiet to tamper with traditional memory as well. Tagged IsFictional.",
	}}

	Mieli = Concept{&Entity{
		ID:    "concept-mieli",
		Name:  "Mieli",
		Kind:  "concept",
		Brief: "In-fiction character: an Oortian warrior from the Finnish-colonised Oort Cloud who ferries Jean le Flambeur out of the Sobornost Dilemma Prison and into the Oubliette on Mars at the opening of The Quantum Thief, acting on behalf of the Sobornost Founder Joséphine Pellegrini. Appears in all three novels of the trilogy — her three AppearsIn claims are the primary stress test for the AppearsIn predicate's non-functionality. Tagged IsFictional.",
	}}

	Perhonen = Concept{&Entity{
		ID:    "concept-perhonen",
		Name:  "Perhonen",
		Kind:  "concept",
		Brief: "In-fiction spacecraft: Mieli's sentient ship, explicitly named in the Wikipedia plot summaries of The Fractal Prince (where it prods Jean to talk to a possessed Mieli) and The Causal Angel (where it is destroyed, separating Jean and Mieli at the opening of the finale). Not named in The Quantum Thief's article, so only two AppearsIn claims are wired — a second stress test for AppearsIn non-functionality with a different arity (2 vs 3) than Mieli's. Tagged IsFictional.",
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
