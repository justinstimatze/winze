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
		Brief:   "Cognitive-science paradigm treating the brain as a hierarchical prediction machine that minimizes prediction error across sensory, attentional, and motor processes through top-down generative models.",
	}}

	AndyClark = Person{&Entity{
		ID:    "andy-clark",
		Name:  "Andy Clark",
		Kind:  "person",
		Brief: "British philosopher of mind known for the extended-mind thesis and influential work on predictive processing, including the 2013 target article \"Whatever next?",
	}}

	HierarchicalPredictionMachine = Hypothesis{&Entity{
		ID:    "hyp-hierarchical-prediction-machine",
		Name:  "The brain is a hierarchical prediction machine that minimises prediction error",
		Kind:  "hypothesis",
		Brief: "A hypothesis that the brain operates as a hierarchical prediction system minimizing prediction error through top-down expectations matched against sensory input, subserving both learning and action.",
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
