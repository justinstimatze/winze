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
		Brief:   "Systematic patterns of deviation from norm and/or rationality in human (and some non-human animal) judgment. The umbrella meta-concept the List of cognitive biases article is about. Whether these patterns are best framed as 'errors in judgment' or as 'rational deviations from logical thought' is itself a contested meta-level question — see Gigerenzer vs the classical framing.",
	}}

	EstimationBiases = Concept{&Entity{
		ID:    "concept-estimation-biases",
		Name:  "Estimation-task biases",
		Kind:  "concept",
		Brief: "One of the six top-level task-based families in the Dimara et al. 2020 classification of cognitive biases. Contains biases that operate when people are asked to assess the value of a quantity. Further sub-classified into five flavors (Association, Baseline, Inertia, Outcome, Self-perspective), though winze currently only records the top-level family membership.",
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
		Brief: "Multi-author team whose 2020 paper proposed the task-based classification of cognitive biases (Estimation, Decision, Hypothesis assessment, Causal attribution, Recall, Opinion reporting) that Wikipedia's List of cognitive biases currently uses as its organising scheme. Modelled as an Organization rather than a Person because 'et al.' signals a collective attribution the ingest cannot honestly resolve to a single author.",
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
		Brief: "A classification of cognitive biases by the task in which they operate: estimation, decision, hypothesis assessment, causal attribution, recall, and opinion reporting. The biases are further sub-classified into five flavors (Association, Baseline, Inertia, Outcome, Self-perspective). This is the organising scheme currently used by the Wikipedia List of cognitive biases article.",
	}}

	GigerenzerRationalDeviationReframing = Hypothesis{&Entity{
		ID:    "hyp-gigerenzer-rational-deviation",
		Name:  "Cognitive biases are rational deviations from logical thought, not errors in judgment",
		Kind:  "hypothesis",
		Brief: "Gigerenzer's meta-level reframing: the patterns classically labelled as 'cognitive biases' should not be framed as errors in judgment at all. They arise from heuristics that are adaptive under real-world information and time constraints, and calling them errors misrepresents the cognitive architecture that produces them. This is a rival to both the classical errors-in-judgment framing and the task-based classification layered on top of it.",
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
