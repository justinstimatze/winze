package winze

// Twelfth public-corpus ingest, third session-5 slice: Donald E.
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
//     session 5 (after Mattson 2014), and therefore the second data
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
//     to prevent. The negative result is worth naming: the session-5
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
//     Mattson2014SPPPaper and ClarkWhateverNextPaper in session 5's
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
	IngestedBy: "winze session 5 (human universals ingest, course-page corpus shape, list-as-content stress test)",
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
		Brief:   "The set of ~300 behavioural, cognitive, linguistic, social, and material-culture patterns compiled by Donald E. Brown as allegedly present in every documented human society. Originally enumerated in Brown 1991 'Human Universals' (McGraw-Hill) and republished as an appendix in Steven Pinker's 2002 'The Blank Slate', the list spans items as varied as language, music, marriage, grammar, fear of death, mythology, tool making, divination, kin-term taxonomy, and belief in the supernatural. The DePaul course handout that serves as winze's ingest source hosts the list verbatim in alphabetical order with no categorical organisation and no per-item commentary. Treated as a standalone Concept rather than folded into HumanCognition because Brown's claim is specifically about cross-cultural regularity, not about cognitive substrate — a future slice reading Pinker's actual framing in Blank Slate could honestly wire a second TheoryOf(HumanCognition) claim to fire the contested-concept rule against Mattson's SPP thesis (seeded for that purpose in mattson_pattern_processing.go), but the DePaul page alone does not commit to that framing and this slice does not fabricate it. The non-functional BelongsTo predicate lets individual universals belong simultaneously to this meta-category and to future categorical groupings (linguistic / social / cognitive / ritual families) without a schema touch.",
	}}

	BrownHumanUniversalsBook1991 = Concept{&Entity{
		ID:      "concept-brown-1991-human-universals",
		Name:    "Brown 1991 Human Universals",
		Kind:    "concept",
		Aliases: []string{"Human Universals (Brown 1991)", "Brown's Human Universals book"},
		Brief:   "Donald E. Brown's 1991 book 'Human Universals' (McGraw-Hill), the primary source for the enumerated list of ~300 cross-cultural universals wired into winze via this slice. Republished as an appendix in Steven Pinker's 2002 'The Blank Slate', which is where most subsequent readers encounter the list; the DePaul course handout winze ingests is itself a verbatim rehosting of the Pinker appendix. Represented as a paper-shape Concept following the session-5 pattern established by ClarkWhateverNextPaper and Mattson2014SPPPaper — books and papers both fit the Concept-with-Authored shape without earning a dedicated Paper role. Pinker and Blank Slate are deliberately not reified as entities in this slice because the DePaul source provides no Pinker-authored content that would justify a structural claim beyond the citation lineage already captured in Provenance.Origin.",
	}}

	DonaldEBrown = Person{&Entity{
		ID:    "donald-e-brown",
		Name:  "Donald E. Brown",
		Kind:  "person",
		Brief: "American anthropologist, author of the 1991 book 'Human Universals' which compiled the list of ~300 behavioural, cognitive, social, and cultural patterns allegedly present in every documented human society. At the time of publication Brown was affiliated with the Department of Anthropology at the University of California, Santa Barbara. Brown's central thesis — that there exists a substantial and enumerable set of human universals that empirical anthropology has systematically under-reported — positions him in explicit tension with the strong-social-constructionist traditions dominant in mid-20th-century anthropology, though the DePaul course handout that serves as winze's ingest source is reticent about this framing and the slice does not reify the scholarly context beyond a Brief mention.",
	}}

	BrownHumanUniversalsThesis = Hypothesis{&Entity{
		ID:    "hyp-brown-human-universals-thesis",
		Name:  "There exists a substantial, enumerable, and systematically under-reported set of behavioural, cognitive, linguistic, social, and material-culture patterns present in every documented human society — cross-cultural universality is a load-bearing empirical finding, not a methodological artefact of Western ethnography",
		Kind:  "hypothesis",
		Brief: "Brown's central claim in the 1991 book. The thesis has two linked parts: (1) an empirical enumeration — there are ~300 such universals and they span linguistic (grammar, phonemes, figurative speech), social (marriage, kin groups, law, division of labor), cognitive (classification, memory, planning, attempts to predict the future), aesthetic (music, dance, body adornment, poetry), and material-culture (cooking, tool making, fire) domains; (2) a meta-methodological claim that mainstream 20th-century anthropology systematically under-reported such universals because its theoretical commitments favored cultural-relative frameworks, and that a honest survey recovers a substantive human nature the constructionist tradition had buried. This slice wires the thesis as TheoryOf(HumanUniversals) — the target Concept is the universality-claim itself rather than human cognition broadly, because the DePaul course handout commits to the former but not explicitly to the latter. A future slice reading Pinker's actual Blank Slate framing (which does extend Brown's list into an argument about cognitive architecture) could honestly add a TheoryOf(HumanCognition) rival to Mattson's SPP thesis and fire the contested-concept rule on HumanCognition for the first time.",
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
		Brief:   "The capacity for structured symbolic communication — a universal human trait present in every documented human society. Appears verbatim on Brown's list ('language') alongside the related items 'grammar', 'phonemes', 'figurative speech', 'metaphor', 'poetry', and 'narrative' — five of which are plausible future-accretion targets for this slice without any schema work. Language is treated as a standalone Concept rather than bridged to any existing winze entity because winze does not currently have a language-specific entity; Mattson 2014 names language as one of the five SPP sub-types in mattson_pattern_processing.go but does not reify it as its own Concept, and the Brief-level cross-ingest reference to Mattson's SPP list is noted here rather than forced as a BelongsTo edge that neither source directly commits to.",
	}}

	MusicUniversal = Concept{&Entity{
		ID:      "concept-music-universal",
		Name:    "Music",
		Kind:    "concept",
		Aliases: []string{"music (human universal)"},
		Brief:   "The universal human practice of organised sound with melodic, rhythmic, and expressive structure. Appears verbatim on Brown's list as 'music'; related items on the list include 'melody', 'dance', 'poetry', 'rhythm', and 'aesthetics' — cluster candidates for a future slice building out aesthetic-domain universals. Music is listed by Brown without further per-item commentary on the DePaul course page, so the Brief carries no additional mechanistic, evolutionary, or cross-cultural claim beyond 'Brown's list includes music'.",
	}}

	MarriageUniversal = Concept{&Entity{
		ID:      "concept-marriage-universal",
		Name:    "Marriage",
		Kind:    "concept",
		Aliases: []string{"marriage (human universal)"},
		Brief:   "The universal human institution of formalised pair-bonding with socially recognised rights, obligations, and kin-term consequences. Appears verbatim on Brown's list alongside related items 'family', 'kin groups', 'incest prevention or avoidance', 'inheritance rules', and 'division of labor by sex' — a social-domain cluster that a future slice can build out with additional Concept + BelongsTo pairs. The specific form marriage takes varies across societies (the DePaul page does not commit to a universal structure beyond 'marriage exists'), and winze follows the source's reticence: this slice reifies marriage-as-universal-category without claiming anything about what marriage-in-general looks like.",
	}}

	FearOfDeath = Concept{&Entity{
		ID:      "concept-fear-of-death",
		Name:    "Fear of death",
		Kind:    "concept",
		Aliases: []string{"death anxiety"},
		Brief:   "The universal human emotional response to anticipated mortality. Appears verbatim on Brown's list as 'fear of death', adjacent to 'death rituals' and 'mourning' as a cluster of mortality-related universals. Notably distinct from the cognitive universals also on Brown's list (memory, classification, planning) because its commitment is affective rather than computational — whereas the SPP framework in mattson_pattern_processing.go centres on cognitive-computational machinery, Brown's list is openly pluralistic about which domains of human experience count as universal. The distinction is Brief-level only; no structural claim linking the two framings is wired here because neither source commits.",
	}}

	Mythology = Concept{&Entity{
		ID:      "concept-mythology-universal",
		Name:    "Mythology",
		Kind:    "concept",
		Aliases: []string{"myths", "myth"},
		Brief:   "The universal human practice of narrative frameworks explaining origins, cosmos, moral order, and the supernatural. Appears verbatim on Brown's list as 'myths'; related items include 'belief in supernatural/religion', 'folklore', 'narrative', and 'magic'. Adjacent to the demon_haunted.go neighbourhood where Carl Sagan's DragonInGarageArgument and the BaloneyDetectionKit engage skeptically with supernatural beliefs — but mirror-source-commitments forbids a structural bridge because Brown's list is purely descriptive (myths exist universally) while Sagan's framing is normative (myths about dragons should be falsifiable). The two live in the same conceptual neighbourhood without sharing a direct claim-level connection.",
	}}

	ToolMaking = Concept{&Entity{
		ID:      "concept-tool-making-universal",
		Name:    "Tool making",
		Kind:    "concept",
		Aliases: []string{"tool making (human universal)", "toolmaking"},
		Brief:   "The universal human practice of fabricating physical artefacts for instrumental use. Both 'tool making' and 'tools' appear verbatim on Brown's list as distinct items — Brown distinguishes the activity from the resulting artefacts — and the DePaul page preserves the distinction. Winze reifies only tool making (the activity) here to avoid an entity-parasitic pair whose only claim is a redundant tool-artefact-vs-tool-activity relation. Material-culture-domain universal, companion to 'fire' and 'cooking' on the list; a future slice can build out the material-culture cluster incrementally.",
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
	// session 6 (blank_slate.go) because the Blank Slate Wikipedia
	// article's See Also link was insufficient; the DePaul source
	// provides the commitment. Second domain for InfluencedBy
	// (first: quantum_thief.go, Rajaniemi → Clark/Leblanc).
	PinkerInfluencedByBrown = InfluencedBy{
		Subject: StevenPinker,
		Object:  DonaldEBrown,
		Prov:    humanUniversalsSource,
	}
)
