package winze

// Mattson pattern processing ingest: Mark P.
// Mattson's 2014 Frontiers in Neuroscience review article "Superior
// pattern processing is the essence of the evolved human brain"
// (doi:10.3389/fnins.2014.00265, PMCID PMC4141622). The paper
// proposes superior pattern processing (SPP) as a unifying
// framework for human cognitive uniqueness — intelligence,
// language, imagination, invention, and the belief in imaginary
// entities such as ghosts and gods — and locates the substrate in
// expansion of the prefrontal and visual cortices through human
// evolution.
//
// This slice is a **deliberate control-group ingest**. Its
// purpose is to answer the exact question the White & Shergill
// slice left open: now that CommentaryOn has been earned (breaking
// a five-slice vocabulary-fit streak at the scientific-paper
// boundary), does a *different* scientific-paper-shaped source
// earn more schema, or does it fit back into the existing
// vocabulary cleanly? If Mattson earns zero new primitives it is
// the first genuine vocabulary-fit slice **following** a
// schema-accretion event, and measures convergence from the
// opposite direction. That is the slice shape the prior
// "are we just picking fitting sources?" worry wanted.
//
// Pre-ingest recon accepted the slice knowing Mattson does **not**
// cite Andy Clark, does not mention apophenia / patternicity /
// pareidolia, and does not engage Shermer, Tetlock, or any other
// winze Person entity by name. The slice was chosen despite the
// apparent zero-bridge outlook precisely because the schema
// question is what it is stress-testing. The surprise the slice
// delivered during writing is that Mattson does in fact commit
// — explicitly and with an attributable hypothesis — to a claim
// about schizophrenia, the Concept entity introduced one slice
// earlier in white_shergill_commentary.go. That is a cross-file
// entity bridge landed WITHOUT deliberate source-shopping, which
// is a stronger finding than the prior "entity density has to
// be shopped for" thesis: the graph is now dense enough that a
// non-shopped slice accidentally lands a bridge.
//
// Schema forcing functions earned by this slice:
//
//   - **None.** Six new entities, seven new claims, four existing
//     predicates used (Authored, Proposes, TheoryOf, BelongsTo).
//     Zero new role types, zero new predicates, zero new pragmas,
//     zero new value structs. This is the first vocabulary-fit
//     slice in winze that has landed **after** a schema-forcing
//     slice (CommentaryOn in white_shergill_commentary.go) rather
//     than as part of an unbroken streak. The previous five-slice
//     vocabulary-fit streak (Clark, Forecasting, Apophenia, QT
//     trilogy ext, Demon-Haunted) could have been dismissed as an
//     artefact of staying inside the Wikipedia neighbourhood; this
//     slice defeats that dismissal by fitting cleanly *in spite of*
//     being a different source shape than the streak, and *after*
//     the commentary slice just proved scientific-paper shape can
//     earn new primitives when the content demands it. The
//     convergence claim now has two-sided evidence: schema accretes
//     when a corpus shape demands it (CommentaryOn) and stays
//     stable when a corpus shape does not (this slice).
//
// Cross-ingest bridges wired by this slice:
//
//   - **Schizophrenia gets its second theory, crossing file
//     boundaries unexpectedly.** white_shergill_commentary.go
//     introduced Schizophrenia as a Concept one slice ago, attached
//     a single TheoryOf claim from the White & Shergill reduced-
//     top-down hypothesis, and noted in its header that a future
//     backfill to apophenia.go could add a rival TheoryOf from
//     Conrad's framing. Before that backfill had a chance to
//     happen, this Mattson slice lands a **second, honestly
//     source-committed** TheoryOf(Schizophrenia) claim — Mattson
//     frames schizophrenia as "a pathological dysregulation of the
//     imagination and mental time travel categories of SPP", which
//     is a Hypothesis clearly attributable to Mattson and clearly
//     targeting the same Schizophrenia entity. This does three
//     things simultaneously:
//
//     1. Fires the //winze:contested lint rule on Schizophrenia as
//        the FIFTH contested target across FOUR ingests (previously:
//        Nondualism×3, NondualAwareness×2, CognitiveBias×2, Apophenia×2;
//        now also Schizophrenia×2 after this slice + white_shergill).
//        First cross-ingest contested-concept fire where the two
//        rival subjects come from two different slices in two
//        different files written independently. The rule
//        continues to fire on pragma alone with zero touches to the
//        lint binary — sixth ingest domain validating the pragma-
//        driven contested-concept design.
//
//     2. Lands a cross-file entity bridge WITHOUT source-shopping.
//        Session 4's discipline win was "entity density has to be
//        shopped for". This slice updates that: once the
//        neighbourhood is dense enough, bridges land accidentally.
//        The slice was chosen for its schema-convergence test, not
//        for its entity overlap — and yet the schizophrenia claim
//        is load-bearing enough in Mattson's source that honest
//        ingestion produces the bridge. The discipline update: shop
//        for sources early when the graph is sparse, and once the
//        graph crosses the density threshold, accidental bridges
//        start contributing as much as shopped ones.
//
//     3. Validates Schizophrenia's decision to be a first-class
//        Concept rather than a buried Brief reference. One slice
//        ago that decision looked borderline — the entity carried
//        exactly one inbound claim. Two slices in, it carries
//        three inbound claims from two files (plus the latent
//        backfill from apophenia.go), which is the density
//        threshold a Concept needs to justify its role.
//
//   - **Tacit neighbourhood bridge to predictive_processing.go and
//     apophenia.go (Brief-level only).** Mattson's SPP thesis
//     operates in the same conceptual neighbourhood as Clark's
//     hierarchical prediction machine and the pattern-recognition
//     substrate that apophenia over-activates, but Mattson does not
//     cite either and mirror-source-commitments forbids fabricating
//     citation lineage. The connection is noted in Briefs only —
//     the same Brief-level reference pattern apophenia.go used for
//     its predictive-processing link. A future slice reading a
//     source that explicitly citation-links Mattson to Clark
//     (Hohwy's Predictive Mind, or a Friston-lineage review citing
//     both) could promote the Brief reference to a real structural
//     claim, but not this slice.
//
// Slice scope and deliberate exclusions:
//
//   - SPP as a core Concept, Mattson as its Person proposer,
//     Mattson2014Paper as the paper-shape Concept (parallel to
//     ClarkWhateverNextPaper in white_shergill_commentary.go),
//     MattsonSPPFraming as the Hypothesis that TheoryOf's
//     HumanCognition, and MagicalThinking as a BelongsTo sub-type
//     of SPP. Schizophrenia is reused across file boundaries rather
//     than redeclared, with the MattsonSchizophreniaFraming as a
//     second Hypothesis Mattson also proposes.
//
//   - **Not reified: Creativity, Language, Reasoning, Imagination,
//     MentalTimeTravel.** The paper explicitly lists these as the
//     other four "types of SPP occurring robustly, if not uniquely,
//     in the human brain," but Mattson's claims about them are
//     list-level rather than load-bearing — he does not define them
//     further, does not attach mechanistic claims to each, and does
//     not distinguish them from each other at any level beyond the
//     list. Reifying them as Concept entities with BelongsTo claims
//     would add graph noise without adding answerable queries.
//     Following the discipline established in misconceptions.go:
//     don't over-reify enumerated list items whose only commitment
//     is that they are on the list. MagicalThinking is the only
//     item in the list that earns entity status, because it is the
//     only one with substantive additional claims (definition,
//     religion framing, TMS lateral-temporal-lobe evidence).
//
//   - **Not reified: Tenenbaum et al 2011.** The paper mentions this
//     work in a single sentence to contrast Mattson's "advanced
//     pattern processing" answer with Tenenbaum et al's framework.
//     One-sentence engagement is not a Disputes claim — reifying
//     Tenenbaum as a Person entity and wiring a Disputes claim
//     would fabricate a scholarly engagement that does not exist
//     at the commitment level the single citation provides.
//     Recorded in this Brief as the one place the paper comes
//     closest to a rival-theory structural edge.
//
//   - **Not wired: Mattson as a rival theorist of predictive
//     processing.** Mattson's SPP thesis and Clark's HPM thesis
//     occupy roughly the same intellectual real estate
//     (fundamental framework for human cognition) but target
//     different Concepts at TheoryOf level — HPM TheoryOf
//     PredictiveProcessing, SPP TheoryOf HumanCognition. Wiring a
//     manual "these are rivals" edge would require a new
//     `Rivals[Hypothesis, Hypothesis]` predicate that no slice
//     currently forces. Deferred to the first slice that reads a
//     source explicitly arguing that SPP and HPM are rivals. The
//     worth-surfacing finding is that the contested-
//     concept rule only fires on shared Object slots, so a rivalry
//     at hypothesis-level that does NOT share a target concept is
//     invisible to the rule today — potentially a future
//     `contested-hypothesis` lint rule if enough slices land
//     rival hypothesis pairs that don't share objects.

var mattsonSPPSource = Provenance{
	Origin:     "Frontiers in Neuroscience / Mattson, Mark P. (2014). 'Superior pattern processing is the essence of the evolved human brain.' Front. Neurosci. 8:265. doi:10.3389/fnins.2014.00265, PMCID PMC4141622, PMID 25202234",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "This article considers superior pattern processing (SPP) as the fundamental basis of most, if not all, unique features of the human brain including intelligence, language, imagination, invention, and the belief in imaginary entities such as ghosts and gods. SPP involves the electrochemical, neuronal network-based, encoding, integration, and transfer to other individuals of perceived or mentally-fabricated patterns. [...] I define pattern processing as the encoding and integration of perceived or mentally-fabricated patterns which can then be used for decision-making and for transfer of the patterns to other individuals. [...] During human evolution, pattern processing capabilities became increasingly sophisticated as the result of expansion of the cerebral cortex, particularly the prefrontal cortex and regions involved in processing of images. [...] Magical thinking can be defined as 'beliefs that defy culturally accepted laws of causality.' [...] In general, psychiatric disorders result from an abnormal skewing of SPP in ways that dissolve the neural circuit-based boundaries between reality and imagination, between the realms of possibilities and probabilities. A key feature of schizophrenia is a blurring of the lines between external reality and internal imagination, between patterns that are real and those that are mentally fabricated. [...] hallucinations and paranoia that occur [in] schizophrenia patients [represent] a pathological dysregulation of the imagination and mental time travel categories of SPP.",
}

// -----------------------------------------------------------------------------
// Concepts, persons, and the paper.
// -----------------------------------------------------------------------------

var (
	SuperiorPatternProcessing = Concept{&Entity{
		ID:      "concept-superior-pattern-processing",
		Name:    "Superior pattern processing",
		Kind:    "concept",
		Aliases: []string{"SPP", "superior pattern processing (SPP)"},
		Brief:   "Cognitive framework positing pattern encoding, integration, and transfer as the fundamental basis of human intelligence, language, imagination, and invention, localized to expanded prefrontal and parietal-temporal-occipital regions.",
	}}

	HumanCognition = Concept{&Entity{
		ID:      "concept-human-cognition",
		Name:    "Human cognition",
		Kind:    "concept",
		Aliases: []string{"the evolved human brain", "human cognitive uniqueness"},
		Brief:   "Cognitive capacities distinguishing humans from other anthropoids, including reasoning, language, abstract thought, and invention. Positioned as a standalone target for competing theoretical frameworks.",
	}}

	MagicalThinking = Concept{&Entity{
		ID:      "concept-magical-thinking",
		Name:    "Magical thinking",
		Kind:    "concept",
		Aliases: []string{"magical thinking / fantasy"},
		Brief:   "Cognitive tendency to perceive causal relationships that violate culturally accepted laws of causality, such as beliefs in clairvoyance, astrology, or spirit influences. Associated with pattern-processing activity in the left lateral temporal lobe.",
	}}

	Mattson2014SPPPaper = Concept{&Entity{
		ID:      "concept-mattson-2014-spp",
		Name:    "Mattson 2014 Superior Pattern Processing",
		Kind:    "concept",
		Aliases: []string{"Mattson 2014", "Mattson SPP paper"},
		Brief:   "2014 Frontiers in Neuroscience review article by Mark P. Mattson arguing that superior pattern processing is the core mechanism of human brain evolution and proposing it as a unified framework for cognitive neuroscience, evolutionary biology, and neural pathology.",
	}}

	MarkMattson = Person{&Entity{
		ID:    "mark-p-mattson",
		Name:  "Mark P. Mattson",
		Kind:  "person",
		Brief: "American neuroscientist at the National Institute on Aging studying cellular neuroscience, neurodegeneration, and neuroprotective mechanisms of exercise and dietary restriction.",
	}}
)

// -----------------------------------------------------------------------------
// Hypotheses Mattson proposes in the paper.
// -----------------------------------------------------------------------------

var (
	MattsonSPPThesis = Hypothesis{&Entity{
		ID:    "hyp-mattson-spp-essence",
		Name:  "Superior pattern processing is the fundamental basis of most, if not all, unique features of the human brain — intelligence, language, imagination, invention, and the belief in imaginary entities",
		Kind:  "hypothesis",
		Brief: "A foundational theory of human cognition proposing that sophisticated pattern perception, integration, and social transfer evolved through prefrontal and visual-cortex expansion.",
	}}

	MattsonSchizophreniaSPPDysregulationFraming = Hypothesis{&Entity{
		ID:    "hyp-mattson-schizophrenia-spp-dysregulation",
		Name:  "Schizophrenia's positive symptoms represent a pathological dysregulation of the imagination and mental time travel categories of superior pattern processing — blurring the neural-circuit boundary between perceived and mentally-fabricated patterns",
		Kind:  "hypothesis",
		Brief: "Hypothesis proposing that schizophrenia's positive symptoms arise from dysregulation in the Self-Pattern-Perception system, causing dissolution of the neural distinction between real and mentally fabricated patterns, particularly in imagination and mental time travel.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	MattsonAuthoredSPPPaper = Authored{
		Subject: MarkMattson,
		Object:  Mattson2014SPPPaper,
		Prov:    mattsonSPPSource,
	}

	MattsonProposesSPPThesis = Proposes{
		Subject: MarkMattson,
		Object:  MattsonSPPThesis,
		Prov:    mattsonSPPSource,
	}

	// The central TheoryOf. Does not fire contested-concept on
	// HumanCognition from this slice alone — Mattson's is the only
	// current rival — but the entity exists to let future slices
	// land rivals zero-touch.
	MattsonSPPThesisTheoryOfHumanCognition = TheoryOf{
		Subject: MattsonSPPThesis,
		Object:  HumanCognition,
		Prov:    mattsonSPPSource,
	}

	// Magical thinking as an SPP sub-type. Only reified list item
	// because it is the only one Mattson attaches substantive
	// additional claims to in the paper.
	MagicalThinkingBelongsToSPP = BelongsTo{
		Subject: MagicalThinking,
		Object:  SuperiorPatternProcessing,
		Prov:    mattsonSPPSource,
	}

	// The schizophrenia framing. Second hypothesis Mattson proposes
	// in the same paper (legitimate — Proposes is not functional);
	// plus a TheoryOf that lands the cross-file entity bridge to
	// Schizophrenia from white_shergill_commentary.go and fires the
	// contested-concept rule as the fifth contested target in winze.
	MattsonProposesSchizophreniaFraming = Proposes{
		Subject: MarkMattson,
		Object:  MattsonSchizophreniaSPPDysregulationFraming,
		Prov:    mattsonSPPSource,
	}

	MattsonSchizophreniaFramingTheoryOfSchizophrenia = TheoryOf{
		Subject: MattsonSchizophreniaSPPDysregulationFraming,
		Object:  Schizophrenia,
		Prov:    mattsonSPPSource,
	}
)
