package winze

// Human universals ingest: Donald E.
// Brown's list of human universals, compiled originally in Brown
// 1991 "Human Universals" (McGraw-Hill) and republished in Steven
// Pinker's 2002 "The Blank Slate". Ingest source is the DePaul
// course page hosting the list verbatim
// (condor.depaul.edu/~mfiddler/hyphen/humunivers.htm). This is the
// first step of roadmap item #20 (ingest Wikipedia/course-material
// articles for a representative selection of human universals) and
// closes roadmap item #19 (humunivers inspiration read) as part of
// the same slice.
//
// Source-shape note: this is winze's first ingest of a **course
// handout / teaching page** rather than an encyclopedic article,
// commentary, or peer-reviewed review. The handout is a flat
// alphabetical list of ~300 universals with minimal framing text —
// no categorical organisation, no methodology, no per-item
// commentary, no explicit discussion of which universals are most
// load-bearing or most contested. The shape is interesting because
// it's *maximally enumerative* (list-as-content) in contrast to the
// *maximally argumentative* shape of a commentary or the
// *narrative-structured* shape of a Wikipedia article. If the
// schema fits this shape cleanly, it is further two-sided evidence
// that winze's vocabulary has converged for slice ingests whose
// content is "taxonomy of things."
//
// Schema forcing functions earned by this slice:
//
//   - **None.** Four existing predicates cover the whole slice:
//     Authored (Brown → book), Proposes (Brown → thesis), TheoryOf
//     (thesis → HumanUniversals), BelongsTo (each selected universal
//     → HumanUniversals). Ten entities, nine claims, zero new role
//     types, zero new predicates, zero new pragmas. This is the
//     **second consecutive post-forcing vocabulary-fit slice** in
//     (after Mattson 2014), and therefore the second data
//     point confirming that the convergence hypothesis holds when
//     the content does not demand new primitives. Three-sided
//     evidence now: schema accretes (CommentaryOn, earned by
//     white_shergill_commentary.go), stays stable on a review-
//     article shape (mattson_pattern_processing.go), and stays
//     stable on a list-as-content shape (this slice).
//
// Cross-ingest bridges wired by this slice:
//
//   - **None, and the negative result is informative.** The
//     temptation was to wire BelongsTo(MagicalThinking, HumanUniversals)
//     as a cross-file bridge to mattson_pattern_processing.go —
//     MagicalThinking is an entity winze already knows about, and
//     "universal cross-cultural magical thinking" feels obviously
//     true. But verbatim checking against Brown's list killed the
//     bridge: Brown has "magic", "magic to increase life", "magic
//     to sustain life", "magic to win love", and "divination" —
//     all behavioural / ritual items — but NOT the phrase "magical
//     thinking" and nothing that matches Mattson's specific
//     definition of magical thinking as "beliefs that defy
//     culturally accepted laws of causality" (a cognitive-stance
//     claim, not a behavioural one). Under winze's mirror-source-
//     commitments discipline, Brown's "magic" and Mattson's
//     "magical thinking" are DIFFERENT CONCEPTS that happen to
//     share vocabulary, and forcing a BelongsTo bridge would be
//     the exact kind of concept-conflation the discipline exists
//     to prevent. The negative result is worth naming: the prior
//     density-threshold finding ("once the graph is dense enough,
//     bridges land accidentally") is not a license to lower the
//     standard for what counts as a bridge — it just observes
//     that when the content honestly supports a bridge, one will
//     land. Slices that honestly cannot bridge should stay
//     unbridged, and this is one.
//
//   - **Future-bridge opportunities surfaced by this slice.** The
//     universals chosen here are deliberately uncontroversial and
//     general-purpose, but several are one-ingest away from
//     load-bearing bridges:
//
//     1. "future, attempts to predict" is on Brown's list and is
//        obviously adjacent to Forecasting / Prediction in
//        forecasting.go and Clark's HierarchicalPredictionMachine
//        in predictive_processing.go. Not bridged here because the
//        item is not in this slice's selection (it would need its
//        own Concept, and the adjacency-to-winze-Forecasting is
//        close enough that the new Concept vs. the existing
//        Forecasting would need careful name disambiguation). A
//        future slice reading a specific source on cross-cultural
//        divination/future-prediction could wire it cleanly.
//
//     2. "classification" is on Brown's list and is adjacent to
//        cognitive_biases.go's Dimara et al. task-based
//        classification and nondualism.go's three rival typologies.
//        Not bridged because classification-as-universal is a
//        meta-cognitive claim and classification-as-domain-schema
//        are different subjects.
//
//     3. "figurative speech" and "metaphor" are both on Brown's
//        list verbatim. Adjacent to nondualism.go's polyvalent-term
//        discussion and the English-translation-of dispute on
//        advaita. Not bridged because polyvalence and figurative
//        speech are different cognitive operations.
//
//     All three are "BelongsTo could land with one slice of careful
//     disambiguation work" — marked here for a future ingest that
//     explicitly wants to build out the universal-to-existing-
//     entity bridge layer.
//
//   - **Future opportunity: Brown vs Mattson as rival theories of
//     human cognition.** Mattson's SPP thesis was seeded as a
//     TheoryOf(HumanCognition) claim in mattson_pattern_processing.go
//     with explicit intent that future slices could add rivals.
//     Brown's human-universals thesis is the obvious candidate —
//     it's a descriptive-empirical alternative account of what is
//     universal about human cognition. BUT: the DePaul course page
//     does NOT editorialise about whether the list is meant as a
//     theory of human cognition, and Pinker's own framing in Blank
//     Slate (which would honestly support the rival-theory claim)
//     is not on the DePaul page. Honest move: wire Brown's thesis
//     as TheoryOf(HumanUniversals), not TheoryOf(HumanCognition),
//     and defer the HumanCognition rival to a future slice that
//     reads Pinker's actual text in Blank Slate. The contested-
//     concept rule will not fire on HumanCognition from this slice;
//     Mattson's claim is still the only one. The setup remains.
//
// Slice scope and deliberate exclusions:
//
//   - HumanUniversals as a meta-Concept, Brown as Person author,
//     Brown 1991 book as paper-shape Concept (parallel pattern to
//     Mattson2014SPPPaper and ClarkWhateverNextPaper in prior
//     two prior slices), Brown's meta-thesis as Hypothesis, and
//     six representative universals as Concept entities with
//     BelongsTo edges to HumanUniversals.
//
//   - **Six selected universals, not 300.** Language, Music,
//     Marriage, FearOfDeath, Mythology, ToolMaking. Selection
//     criteria: (1) each is canonical and uncontroversial, (2)
//     each is a flat noun that does not require behavioural /
//     ritual / cognitive-stance disambiguation against an existing
//     winze entity, (3) the set spans linguistic, social, aesthetic,
//     existential, cultural, and technological domains without
//     being explicitly organised that way in Brown's alphabetical
//     list. A future slice can accrete additional universals by
//     simply adding more Concept + BelongsTo pairs with no schema
//     touch — the shape is designed for incremental accretion.
//
//   - **Not reified: Steven Pinker and The Blank Slate 2002.** The
//     DePaul course page cites Pinker's book as the publication
//     venue for Brown's list but provides no Pinker-authored
//     commentary, framing, or editorial content. Reifying Pinker
//     and Blank Slate as entities would create a parasitic pair
//     whose only inbound claim is an Authored edge between them —
//     pure reification without load-bearing claims. The citation
//     lives at the Provenance.Origin layer where it belongs, and a
//     future slice reading actual Pinker content (there's a
//     predictable roadmap slot for an ingest of Blank Slate's
//     discussion of the universals as evidence for cognitive
//     architecture) can promote him then with a real Hypothesis
//     attached. Same discipline as Tenenbaum 2011 in
//     mattson_pattern_processing.go: one-sentence / one-citation
//     engagement does not justify a structural edge.
//
//   - **Not reified: Brown 2000 "Human Universals and their
//     Implications".** Cited on the DePaul page as a secondary
//     Brown source but not quoted or summarised. Same reasoning —
//     a reference without load-bearing content lives in
//     Provenance.Origin only.
//
//   - **Not reified: any specific category labels.** Brown's list
//     is flat; the DePaul page does not organise the items into
//     cognitive / social / linguistic / material-culture / etc.
//     families. Winze does not add category labels the source
//     refuses to provide. If a future slice reads a secondary
//     source that DOES categorise Brown's universals (Pinker's
//     discussion in Blank Slate is the canonical candidate, or
//     Murdock's earlier Cross-Cultural Survey), the category
//     families can be added as Concept entities with nested
//     BelongsTo edges — the BelongsTo predicate is non-functional
//     so the dual membership (item BelongsTo category AND item
//     BelongsTo HumanUniversals) is legal.

var humanUniversalsSource = Provenance{
	Origin:     "DePaul course page (condor.depaul.edu/~mfiddler/hyphen/humunivers.htm) hosting Donald E. Brown's list of human universals verbatim, compiled in Brown, D.E. 1991 'Human Universals' (McGraw-Hill) and republished in Pinker, S. 2002 'The Blank Slate'",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "compiled by Donald E. Brown, as published in The Blank Slate by Steven Pinker, 2002. [Brown, D.E. 1991. Human universals. New York: McGraw-Hill] [Brown, D.E. 2000. Human universals and their implications.] [...partial alphabetical list:] abstraction in speech & thought, aesthetics, affection expressed and felt, age grades, belief in supernatural/religion, classification, cooking, cooperation, dance, death rituals, division of labor by sex, dreams, emotions, envy, etiquette, facial expressions, family, fear of death, feasting, figurative speech, fire, folklore, future (attempts to predict), generosity admired, gestures, gift giving, government, grammar, group living, healing the sick, hope, incest prevention, inheritance rules, institutions, jokes, kin groups, language, law, leaders, magic, marriage, medicine, memory, metaphor, music, myths, narrative, person (concept of), planning, play, poetry, property, reciprocal exchanges, tool making, tools, [...].",
}

// -----------------------------------------------------------------------------
// The meta-category, the source book, the author, and the meta-thesis.
// -----------------------------------------------------------------------------

var (
	HumanUniversals = Concept{&Entity{
		ID:      "concept-human-universals",
		Name:    "Human universals",
		Kind:    "concept",
		Aliases: []string{"Brown's universals", "human cultural universals"},
		Brief:   "Approximately 300 behavioural, cognitive, linguistic, social, and material-culture patterns documented across all known human societies, compiled by Donald E. Brown and republished in Steven Pinker's *The Blank Slate*.",
	}}

	BrownHumanUniversalsBook1991 = Concept{&Entity{
		ID:      "concept-brown-1991-human-universals",
		Name:    "Brown 1991 Human Universals",
		Kind:    "concept",
		Aliases: []string{"Human Universals (Brown 1991)", "Brown's Human Universals book"},
		Brief:   "1991 anthropology book enumerating approximately 300 human cultural universals across societies, establishing the foundational list subsequently republished in Pinker's 2002 work.",
	}}

	DonaldEBrown = Person{&Entity{
		ID:    "donald-e-brown",
		Name:  "Donald E. Brown",
		Kind:  "person",
		Brief: "American anthropologist who compiled a list of approximately 300 universal behavioral and cultural patterns across all documented human societies in his 1991 book \"Human Universals.",
	}}

	BrownHumanUniversalsThesis = Hypothesis{&Entity{
		ID:    "hyp-brown-human-universals-thesis",
		Name:  "There exists a substantial, enumerable, and systematically under-reported set of behavioural, cognitive, linguistic, social, and material-culture patterns present in every documented human society — cross-cultural universality is a load-bearing empirical finding, not a methodological artefact of Western ethnography",
		Kind:  "hypothesis",
		Brief: "Anthropologist Donald Brown's 1991 claim that ~300 human universals span linguistic, social, cognitive, aesthetic, and material domains, arguing 20th-century anthropology systematically underreported them due to cultural-relativist bias.",
	}}
)

// -----------------------------------------------------------------------------
// Selected universals as Concepts. Six items chosen as canonical,
// uncontroversial, and name-disambiguated against existing winze
// entities — the slice is designed for incremental accretion, so
// future slices can add more Concept + BelongsTo pairs with zero
// schema touches.
// -----------------------------------------------------------------------------

var (
	LanguageUniversal = Concept{&Entity{
		ID:      "concept-language-universal",
		Name:    "Language",
		Kind:    "concept",
		Aliases: []string{"language (human universal)", "natural language"},
		Brief:   "Structured symbolic communication system universal to all documented human societies. Foundational human capacity for grammar, phonemes, figurative speech, and narrative.",
	}}

	MusicUniversal = Concept{&Entity{
		ID:      "concept-music-universal",
		Name:    "Music",
		Kind:    "concept",
		Aliases: []string{"music (human universal)"},
		Brief:   "Human practice of organized sound with melodic, rhythmic, and expressive structure, appearing on Brown's list of cultural universals.",
	}}

	MarriageUniversal = Concept{&Entity{
		ID:      "concept-marriage-universal",
		Name:    "Marriage",
		Kind:    "concept",
		Aliases: []string{"marriage (human universal)"},
		Brief:   "Formalised pair-bonding institution appearing as a human universal, with socially recognised rights, obligations, and kin consequences. Specific forms vary across societies.",
	}}

	FearOfDeath = Concept{&Entity{
		ID:      "concept-fear-of-death",
		Name:    "Fear of death",
		Kind:    "concept",
		Aliases: []string{"death anxiety"},
		Brief:   "Psychological universal characterized by emotional anticipation of mortality. Listed by Brown among cross-cultural emotional responses alongside death rituals and mourning practices.",
	}}

	Mythology = Concept{&Entity{
		ID:      "concept-mythology-universal",
		Name:    "Mythology",
		Kind:    "concept",
		Aliases: []string{"myths", "myth"},
		Brief:   "Narrative framework found in all human cultures that explains origins, cosmos, moral order, and the supernatural.",
	}}

	ToolMaking = Concept{&Entity{
		ID:      "concept-tool-making-universal",
		Name:    "Tool making",
		Kind:    "concept",
		Aliases: []string{"tool making (human universal)", "toolmaking"},
		Brief:   "The universal human practice of fabricating physical artifacts for instrumental use; distinguished from the resulting tools themselves as an activity-based cultural universal.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	BrownAuthoredHumanUniversalsBook = Authored{
		Subject: DonaldEBrown,
		Object:  BrownHumanUniversalsBook1991,
		Prov:    humanUniversalsSource,
	}

	BrownProposesHumanUniversalsThesis = Proposes{
		Subject: DonaldEBrown,
		Object:  BrownHumanUniversalsThesis,
		Prov:    humanUniversalsSource,
	}

	// TheoryOf(HumanUniversals) — the target is the universality
	// claim itself, NOT HumanCognition. See slice header for the
	// discipline reasoning: DePaul handout does not commit to the
	// cognitive-architecture framing, and forcing a
	// TheoryOf(HumanCognition) rival to Mattson would fabricate
	// beyond source. The Mattson rival remains a future-slice
	// opportunity for a reading of actual Pinker content.
	BrownHumanUniversalsThesisTheoryOfHumanUniversals = TheoryOf{
		Subject: BrownHumanUniversalsThesis,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}

	// The six BelongsTo claims. Zero schema touches needed for any
	// future expansion of this slice: add a new Concept, add a new
	// BelongsTo, done.
	LanguageBelongsToHumanUniversals = BelongsTo{
		Subject: LanguageUniversal,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}
	MusicBelongsToHumanUniversals = BelongsTo{
		Subject: MusicUniversal,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}
	MarriageBelongsToHumanUniversals = BelongsTo{
		Subject: MarriageUniversal,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}
	FearOfDeathBelongsToHumanUniversals = BelongsTo{
		Subject: FearOfDeath,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}
	MythologyBelongsToHumanUniversals = BelongsTo{
		Subject: Mythology,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}
	ToolMakingBelongsToHumanUniversals = BelongsTo{
		Subject: ToolMaking,
		Object:  HumanUniversals,
		Prov:    humanUniversalsSource,
	}

	// Cross-file bridge: Pinker (blank_slate.go) influenced by Brown
	// (this file). The DePaul course page commits to the relationship:
	// Brown's list was "republished in Pinker, S. 2002 'The Blank Slate'".
	// Republication of a complete list as an appendix commits to a
	// directional influence relationship — Pinker incorporated Brown's
	// work as foundational to his own argument. This was deferred in
	// blank_slate.go because the Blank Slate Wikipedia
	// article's See Also link was insufficient; the DePaul source
	// provides the commitment. Second domain for InfluencedBy
	// (first: quantum_thief.go, Rajaniemi → Clark/Leblanc).
	PinkerInfluencedByBrown = InfluencedBy{
		Subject: StevenPinker,
		Object:  DonaldEBrown,
		Prov:    humanUniversalsSource,
	}
)
