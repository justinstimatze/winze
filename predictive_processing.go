package winze

// Seventh public-corpus ingest: Andy Clark's 2013 target article
// "Whatever next? Predictive brains, situated agents, and the future
// of cognitive science" in Behavioral and Brain Sciences, 36(3),
// 181–204 (PMID 23663408). The canonical journal-length articulation
// of the predictive processing framework — the proposal that the
// brain is fundamentally a hierarchical prediction machine
// constantly minimising prediction error between top-down
// expectations and incoming sensory input.
//
// Motivation for the pairing with the Forecasting ingest (queued):
// both are about "prediction" but at different time scales and by
// different agents. Predictive processing operates at milliseconds-
// to-seconds in neural substrate and is a theory of how minds
// *implement* prediction implicitly; explicit Forecasting is the
// deliberate human practice of making claims about future events
// with resolution dates and accuracy dimensions. Ingesting the
// Clark paper first surfaces the 'prediction as cognitive
// substrate' framing before the Forecasting ingest forces the
// 'prediction as explicit claim with temporal resolution' schema.
// The two slices share zero predicates today but the conceptual
// neighbourhood is tight, and any future unification (a
// PredictionAgent role, a Predicts predicate family, a time-scale
// annotation) should be designed with both ingests in view.
//
// Schema forcing functions earned by this slice:
//
//   - None of its own. All three primary claims reuse existing
//     Proposes + TheoryOf + //winze:contested machinery. This is
//     the first slice since the common-misconceptions ingest to
//     earn zero new primitives — which is itself a finding: when
//     an ingest neatly fits existing vocabulary, resist the
//     temptation to invent structure for it.
//
//   - One cross-ingest bridge earned: the new InfluencedBy
//     predicate (in predicates.go) is defined for this slice but
//     earns its keep across both this ingest and the Quantum Thief
//     slice simultaneously. Rajaniemi acknowledges Clark as an
//     influence on the 'mind as self-loop' idea in The Fractal
//     Prince, and Rajaniemi is also modelled-on-Leblanc's-Lupin in
//     The Quantum Thief — two occurrences in two different slices,
//     which is the earliest accretion the discipline allows. The
//     backfill into quantum_thief.go proves that cross-ingest
//     edges can land cleanly in winze's non-executable model
//     without either slice having to anticipate the other.
//
// Slice scope: Andy Clark as Person, HierarchicalPredictionMachine
// as Hypothesis, PredictiveProcessing as the meta-Concept the
// hypothesis is a TheoryOf. The paper's 30+ peer commentaries are
// noted in Clark's Brief but not reified as entities — the PubMed
// abstract does not list individual commentators, so encoding them
// would fabricate beyond the source. A future slice that reads
// specific commentaries (Hohwy, Friston, Rao & Ballard, etc.)
// could earn a contested-concept surface on PredictiveProcessing
// by adding rival TheoryOf claims — zero schema work, just more
// honest reading of primary sources.

var clarkPredictiveBrainSource = Provenance{
	Origin:     "PubMed 23663408 / Clark, Andy (2013). 'Whatever next? Predictive brains, situated agents, and the future of cognitive science.' Behavioral and Brain Sciences, 36(3), 181-204",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 4 (predictive processing ingest, PubMed 23663408)",
	Quote:      "Clark examines the 'hierarchical prediction machine' hypothesis — the proposition that brains function fundamentally as prediction systems, constantly attempting to match incoming sensory inputs with top-down expectations or predictions through hierarchical generative processing designed to minimise prediction error. [...] Clark argues the predictive processing framework represents 'the best clue yet to the shape of a unified science of mind and action.' [...] The journal published 30+ peer commentaries alongside this target article, indicating substantive scholarly engagement with Clark's central claims.",
}

// -----------------------------------------------------------------------------
// Concepts, persons, and hypotheses.
// -----------------------------------------------------------------------------

var (
	PredictiveProcessing = Concept{&Entity{
		ID:      "concept-predictive-processing",
		Name:    "Predictive processing",
		Kind:    "concept",
		Aliases: []string{"predictive coding", "prediction error minimisation"},
		Brief:   "Cognitive-science paradigm that treats the brain as fundamentally a hierarchical prediction machine: top-down generative models continuously predict incoming sensory input, and learning is driven by the minimisation of prediction error at every level of the hierarchy. Perception, attention, and action are all reframed as instances of the same prediction-error-minimisation loop. The Clark 2013 target article is the canonical journal-length articulation of the framework for a cognitive-science audience, but the underlying mathematical apparatus descends from earlier work on predictive coding in vision (Rao and Ballard 1999) and the free-energy principle (Friston). Left as a Concept rather than reified into its own Hypothesis because it names a research paradigm — a family of overlapping theories — rather than a single testable claim.",
	}}

	AndyClark = Person{&Entity{
		ID:    "andy-clark",
		Name:  "Andy Clark",
		Kind:  "person",
		Brief: "British philosopher of mind and cognitive scientist at the University of Sussex (previously Edinburgh), known for the extended-mind thesis (with David Chalmers, 1998) and for influential work on predictive processing. Author of the 2013 Behavioral and Brain Sciences target article 'Whatever next?' which received 30+ peer commentaries and is the canonical journal articulation of predictive processing for a cross-disciplinary audience. Also cited by Hannu Rajaniemi in The Fractal Prince acknowledgments as an influence on the novel's 'mind as self-loop' treatment — see the InfluencedBy claim in quantum_thief.go for the cross-ingest bridge.",
	}}

	HierarchicalPredictionMachine = Hypothesis{&Entity{
		ID:    "hyp-hierarchical-prediction-machine",
		Name:  "The brain is a hierarchical prediction machine that minimises prediction error",
		Kind:  "hypothesis",
		Brief: "Clark's central thesis in 'Whatever next?' (2013): the brain functions fundamentally as a prediction system that constantly matches incoming sensory inputs against top-down expectations, with learning and action both subserved by the minimisation of prediction error across a hierarchy of generative models. The thesis is advanced as 'the best clue yet to the shape of a unified science of mind and action' — qualified optimism rather than settled consensus, with substantial unresolved challenges acknowledged in the article.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	ClarkProposesHierarchicalPredictionMachine = Proposes{
		Subject: AndyClark,
		Object:  HierarchicalPredictionMachine,
		Prov:    clarkPredictiveBrainSource,
	}

	HierarchicalPredictionMachineTheoryOfPredictiveProcessing = TheoryOf{
		Subject: HierarchicalPredictionMachine,
		Object:  PredictiveProcessing,
		Prov:    clarkPredictiveBrainSource,
	}
)
