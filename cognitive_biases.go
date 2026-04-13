package winze

// Fifth public-corpus ingest: Wikipedia's List of cognitive biases.
// Continues the taxonomy-shape work begun by the common-misconceptions
// slice but stresses different structure:
//
//   - Cognitive biases are *named cognitive phenomena*, not corrections
//     of false beliefs. Each entry is a thing-that-exists, not a fact
//     about what people get wrong. The name-is-content discipline is
//     unchanged but the annotation predicate is different.
//
//   - The source organises biases under a two-level hierarchy (task >
//     sub-flavor > bias) per Dimara et al. 2020. The hierarchy itself
//     is a first-class claim — who proposed this classification, and
//     how does it relate to rival classifications?
//
//   - The source article explicitly flags contested meta-level
//     structure: "there are often controversies about how to classify
//     these biases or how to explain them." Gerd Gigerenzer is named
//     as a critic who reframes biases as "rational deviations from
//     logical thought" rather than errors in judgment. This is a
//     genuine contested-concept case about the meta-concept
//     CognitiveBias itself, parallel in shape to the Nondualism
//     typology disagreement but in a completely different domain.
//
// Schema forcing functions earned by this slice:
//
//   - `IsCognitiveBias UnaryClaim[Concept]` — new unary tag, parallel
//     to IsPolyvalentTerm and CorrectsCommonMisconception. Third case
//     of the "mark a concept as belonging to a well-defined
//     taxonomy" pattern, which is probably worth naming explicitly
//     at four occurrences.
//
//   - `BelongsTo BinaryRelation[Concept, Concept]` — new predicate
//     for taxonomic membership, distinct from DerivedFrom (etymology).
//     Not functional (a concept can belong to multiple overlapping
//     families); eventually wants a cycle-detection lint rule but
//     not forced by this slice.
//
//   - No new role types. No new functional predicates. No new contested
//     pragma usages — `TheoryOf` already carries the contested pragma
//     from the Nondualism ingest, and the rule fires on this new
//     subject with zero additional code. This is the cleanest
//     validation so far that the pragma machinery generalises across
//     ingests without bespoke per-corpus wiring.
//
// Slice scope: four biases from the Estimation task family (Availability
// heuristic, Anchoring, Dunning–Kruger, Hot-hand fallacy), plus the
// EstimationBiases family concept and the CognitiveBias meta-concept,
// plus two rival meta-theories (Dimara 2020 task-based classification,
// Gigerenzer rational-deviation reframing). Biases in other task
// families can be added by future slices with zero schema work.

var cognitiveBiasesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / List_of_cognitive_biases",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 3 (cognitive biases first slice)",
	Quote:      "In psychology and cognitive science, cognitive biases are systematic patterns of deviation from norm and/or rationality in judgment. [...] Although the reality of these biases is confirmed by reproducible research, there are often controversies about how to classify these biases or how to explain them. [...] Gerd Gigerenzer has criticized the framing of cognitive biases as errors in judgment, and favors interpreting them as arising from rational deviations from logical thought. [...] This list is organized based on the task-based classification proposed by Dimara et al. (2020).",
}

// -----------------------------------------------------------------------------
// Meta-concepts and family concepts.
// -----------------------------------------------------------------------------

var (
	CognitiveBias = Concept{&Entity{
		ID:      "concept-cognitive-bias",
		Name:    "Cognitive bias",
		Kind:    "concept",
		Aliases: []string{"cognitive biases"},
		Brief:   "Systematic patterns of deviation from rationality in human judgment and decision-making. The meta-concept encompassing documented cognitive biases and whether they constitute errors or rational adaptations.",
	}}

	EstimationBiases = Concept{&Entity{
		ID:    "concept-estimation-biases",
		Name:  "Estimation-task biases",
		Kind:  "concept",
		Brief: "A top-level category of cognitive biases affecting quantity assessment, containing five sub-types: Association, Baseline, Inertia, Outcome, and Self-perspective.",
	}}
)

// -----------------------------------------------------------------------------
// Individual biases. Each is a Concept whose Brief summarises the
// Wikipedia definition; the IsCognitiveBias claim is the machine-queryable
// tag and the BelongsTo claim puts it in the Estimation family.
// -----------------------------------------------------------------------------

var (
	AvailabilityHeuristic = Concept{&Entity{
		ID:      "concept-availability-heuristic",
		Name:    "Availability heuristic",
		Kind:    "concept",
		Aliases: []string{"availability bias"},
		Brief:   "The tendency to overestimate the likelihood of events with greater 'availability' in memory. Recency, proximity, unusualness, or emotional charge make examples more available, and available examples are imputed greater importance than less available ones.",
	}}

	AnchoringBias = Concept{&Entity{
		ID:      "concept-anchoring-bias",
		Name:    "Anchoring bias",
		Kind:    "concept",
		Aliases: []string{"focalism", "anchoring"},
		Brief:   "The tendency to rely too heavily — to 'anchor' — on one trait or piece of information when making decisions, usually the first piece of information acquired on that subject.",
	}}

	DunningKrugerEffect = Concept{&Entity{
		ID:    "concept-dunning-kruger-effect",
		Name:  "Dunning–Kruger effect",
		Kind:  "concept",
		Brief: "The tendency for unskilled individuals to overestimate their own ability and the tendency for experts to underestimate their own ability.",
	}}

	HotHandFallacy = Concept{&Entity{
		ID:      "concept-hot-hand-fallacy",
		Name:    "Hot-hand fallacy",
		Kind:    "concept",
		Aliases: []string{"hot hand phenomenon", "hot hand"},
		Brief:   "The belief that a person who has experienced success with a random event has a greater chance of further success in additional attempts. Contrast the Gambler's fallacy, which runs the opposite direction.",
	}}

	ConfirmationBias = Concept{&Entity{
		ID:      "concept-confirmation-bias",
		Name:    "Confirmation bias",
		Kind:    "concept",
		Aliases: []string{"confirmatory bias", "myside bias"},
		Brief:   "The tendency to search for, interpret, favor, and recall information in a way that confirms or supports one's prior beliefs or values.",
	}}

	SurvivorshipBias = Concept{&Entity{
		ID:      "concept-survivorship-bias",
		Name:    "Survivorship bias",
		Kind:    "concept",
		Aliases: []string{"survival bias"},
		Brief:   "The logical error of concentrating on entities that passed a selection process while overlooking those that did not, leading to incorrect conclusions about what caused success.",
	}}

	FramingEffect = Concept{&Entity{
		ID:      "concept-framing-effect",
		Name:    "Framing effect",
		Kind:    "concept",
		Aliases: []string{"framing bias"},
		Brief:   "The tendency to draw different conclusions from the same information depending on how it is presented, such as whether outcomes are framed as gains or losses.",
	}}

	BaseRateNeglect = Concept{&Entity{
		ID:      "concept-base-rate-neglect",
		Name:    "Base rate neglect",
		Kind:    "concept",
		Aliases: []string{"base rate fallacy", "base rate bias"},
		Brief:   "The tendency to ignore general statistical information (base rates) in favor of specific case information when making probability judgments.",
	}}
)

// -----------------------------------------------------------------------------
// People and organisations proposing rival meta-theories about the
// CognitiveBias concept itself.
// -----------------------------------------------------------------------------

var (
	GerdGigerenzer = Person{&Entity{
		ID:    "gerd-gigerenzer",
		Name:  "Gerd Gigerenzer",
		Kind:  "person",
		Brief: "German psychologist known for arguing that cognitive biases should not be framed as errors in judgment but as rational deviations from logical thought that arise from fast-and-frugal heuristics adapted to real-world information ecologies.",
	}}

	DimaraEtAl2020 = Organization{&Entity{
		ID:    "dimara-et-al-2020",
		Name:  "Dimara et al. 2020",
		Kind:  "organization",
		Brief: "Research team that proposed the task-based classification system for cognitive biases adopted by Wikipedia's List of cognitive biases.",
	}}
)

// -----------------------------------------------------------------------------
// Rival meta-theories. Both are TheoryOf(CognitiveBias). Wiring both
// claims triggers the contested-concept lint rule on a second subject,
// validating that the rule generalises across ingests.
// -----------------------------------------------------------------------------

var (
	DimaraTaskBasedClassification = Hypothesis{&Entity{
		ID:    "hyp-dimara-task-based-classification",
		Name:  "Dimara et al. 2020 task-based classification of cognitive biases",
		Kind:  "hypothesis",
		Brief: "A cognitive bias classification system organized by task type (estimation, decision, hypothesis assessment, causal attribution, recall, opinion reporting) and five behavioral flavors (Association, Baseline, Inertia, Outcome, Self-perspective).",
	}}

	GigerenzerRationalDeviationReframing = Hypothesis{&Entity{
		ID:    "hyp-gigerenzer-rational-deviation",
		Name:  "Cognitive biases are rational deviations from logical thought, not errors in judgment",
		Kind:  "hypothesis",
		Brief: "A hypothesis proposing that so-called cognitive biases reflect adaptive heuristics suited to real-world constraints rather than judgment errors, challenging the classical error-based framing of these phenomena.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	AvailabilityHeuristicIsBias = IsCognitiveBias{
		Subject: AvailabilityHeuristic,
		Prov:    cognitiveBiasesSource,
	}
	AnchoringBiasIsBias = IsCognitiveBias{
		Subject: AnchoringBias,
		Prov:    cognitiveBiasesSource,
	}
	DunningKrugerIsBias = IsCognitiveBias{
		Subject: DunningKrugerEffect,
		Prov:    cognitiveBiasesSource,
	}
	HotHandFallacyIsBias = IsCognitiveBias{
		Subject: HotHandFallacy,
		Prov:    cognitiveBiasesSource,
	}

	ConfirmationBiasIsBias = IsCognitiveBias{
		Subject: ConfirmationBias,
		Prov:    cognitiveBiasesSource,
	}
	SurvivorshipBiasIsBias = IsCognitiveBias{
		Subject: SurvivorshipBias,
		Prov:    cognitiveBiasesSource,
	}
	FramingEffectIsBias = IsCognitiveBias{
		Subject: FramingEffect,
		Prov:    cognitiveBiasesSource,
	}
	BaseRateNeglectIsBias = IsCognitiveBias{
		Subject: BaseRateNeglect,
		Prov:    cognitiveBiasesSource,
	}

	// New biases belong directly to CognitiveBias umbrella (different
	// task families than the Estimation family above).
	ConfirmationBelongsToCognitiveBias = BelongsTo{
		Subject: ConfirmationBias,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}
	SurvivorshipBelongsToCognitiveBias = BelongsTo{
		Subject: SurvivorshipBias,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}
	FramingBelongsToCognitiveBias = BelongsTo{
		Subject: FramingEffect,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}
	BaseRateNeglectBelongsToCognitiveBias = BelongsTo{
		Subject: BaseRateNeglect,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}

	AvailabilityBelongsToEstimation = BelongsTo{
		Subject: AvailabilityHeuristic,
		Object:  EstimationBiases,
		Prov:    cognitiveBiasesSource,
	}
	AnchoringBelongsToEstimation = BelongsTo{
		Subject: AnchoringBias,
		Object:  EstimationBiases,
		Prov:    cognitiveBiasesSource,
	}
	DunningKrugerBelongsToEstimation = BelongsTo{
		Subject: DunningKrugerEffect,
		Object:  EstimationBiases,
		Prov:    cognitiveBiasesSource,
	}
	HotHandBelongsToEstimation = BelongsTo{
		Subject: HotHandFallacy,
		Object:  EstimationBiases,
		Prov:    cognitiveBiasesSource,
	}

	// The Estimation family belongs to the CognitiveBias umbrella —
	// taxonomy hierarchy spans two levels.
	EstimationBelongsToCognitiveBias = BelongsTo{
		Subject: EstimationBiases,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}

	// Rival meta-theories of the CognitiveBias concept. Both TheoryOf
	// claims target the same Object (CognitiveBias), so the
	// contested-concept lint rule will emit a group with 2 distinct
	// subjects — proving the rule is not Nondualism-specific.
	DimaraAboutCognitiveBias = TheoryOf{
		Subject: DimaraTaskBasedClassification,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}
	GigerenzerAboutCognitiveBias = TheoryOf{
		Subject: GigerenzerRationalDeviationReframing,
		Object:  CognitiveBias,
		Prov:    cognitiveBiasesSource,
	}

	// Attributions. Dimara is an Organization (et al.) so ProposesOrg;
	// Gigerenzer is a Person so Proposes.
	DimaraProposesClassification = ProposesOrg{
		Subject: DimaraEtAl2020,
		Object:  DimaraTaskBasedClassification,
		Prov:    cognitiveBiasesSource,
	}
	GigerenzerProposesReframing = Proposes{
		Subject: GerdGigerenzer,
		Object:  GigerenzerRationalDeviationReframing,
		Prov:    cognitiveBiasesSource,
	}
)
