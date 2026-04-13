package winze

// Tenth public-corpus ingest and first session-5 slice: Thomas P. White
// and Sukhi S. Shergill's 2012 Frontiers in Psychology commentary
// "Using Illusions to Understand Delusions" (doi:10.3389/fpsyg.2012.00407),
// explicitly a peer commentary on Andy Clark's then-in-press 2013
// Behavioral and Brain Sciences target article "Whatever next? Predictive
// brains, situated agents, and the future of cognitive science" — the
// same paper predictive_processing.go ingested as its source. The slice
// was chosen deliberately after a session-5 pre-ingest recon found that
// Mattson 2014 "Superior pattern processing" (the originally-queued
// scientific-paper stress-test target) earns zero cross-file entity
// bridges: Mattson does not cite Clark 2013, does not mention apophenia
// or patternicity, and under the mirror-source-commitments discipline
// cannot honestly be wired to existing winze content. The White &
// Shergill commentary, by contrast, is structurally committed to
// citing Clark — it is literally a commentary on his paper — which
// makes the cross-file entity bridge load-bearing on real scholarly
// lineage, not invented-for-graph-density.
//
// This slice is the first deliberate test of the session-4 finding
// that **entity density has to be shopped for**. The hypothesis being
// stressed: if you pick a source by whether it cites existing winze
// entities, you can earn cross-file bridges on every ingest, not just
// the occasional lucky one. Outcome of this slice is a new cross-file
// entity (Clark 2013 as a paper-shaped Concept, authored by the
// existing AndyClark Person) plus one paper-to-paper structural
// edge (CommentaryOn).
//
// Schema forcing functions earned by this slice:
//
//   - **CommentaryOn BinaryRelation[Concept, Concept]**. First new
//     predicate since the Quantum Thief trilogy extension — ends a
//     five-slice vocabulary-fit streak (Clark, Forecasting, Apophenia,
//     QT trilogy ext, Demon-Haunted). The predicate is earned because
//     a peer commentary's relationship to its target paper is not
//     reducible to any existing winze primitive: Proposes is
//     Person→Hypothesis, InfluencedBy is Person→Person, Authored is
//     Person→Concept, DerivedFrom is Concept→Concept but tracks
//     etymological/technical lineage rather than scholarly response.
//     The "A is a commentary on B" edge needs its own predicate to
//     survive queries like "what commentaries does winze know about
//     Clark 2013?" — the author-level collapse via InfluencedBy
//     cannot answer that question at paper granularity. See
//     predicates.go for the full docstring including why the slice
//     broke the vocabulary-fit streak exactly where it did and why
//     that breakage is informative, not a regression.
//
// Deferred (available for a future slice but not forced here):
//
//   - **IsScientificPaper UnaryClaim[Concept]** as a taxonomy tag
//     parallel to IsFictionalWork/IsFictional/IsCognitiveBias. Not
//     earned by this slice because both papers it introduces carry
//     enough structural context via Authored + CommentaryOn for a
//     reader to identify them as scientific papers; adding a tag
//     now would be name-is-content decoration without a query that
//     needs it. Will be earned the first time a slice ingests a
//     paper whose Authored author is also the author of a non-paper
//     work in winze, forcing disambiguation.
//
//   - **ProposedIn** or **Advances** as a Hypothesis→Paper predicate.
//     The slice captures White & Shergill's reduced-top-down framing
//     as a Hypothesis, and both Clark's HPM hypothesis (in
//     predictive_processing.go) and this commentary's hypothesis
//     could in principle be wired to the specific paper that
//     advances them. Deferred because the Brief-level linkage is
//     already enough for "which paper contains this hypothesis"
//     queries, and adding a dedicated predicate would partially
//     collapse into Authored (the author authors the paper and the
//     paper advances the hypothesis, so the hypothesis is implicitly
//     associated with the author's paper). Promote when a slice
//     actually has a query that this collapse cannot answer — for
//     example, a paper with multiple authors advancing different
//     hypotheses, or a hypothesis advanced by multiple non-overlapping
//     author sets.
//
//   - **Figure, Section, Citation** as first-class entities. Not
//     earned by this slice because a two-page commentary does not
//     force figure-level or section-level granularity, and the
//     reference list is short enough to capture in the Brief. The
//     session-5 scientific-paper stress test continues: the first
//     slice that ingests a long research article with substantive
//     figure-level claims will earn these.
//
// Cross-ingest bridges wired by this slice:
//
//   - **AndyClark gains a paper-level attribution.** Until this slice
//     Clark was a Person entity linked to winze only via the Proposes
//     claim to HierarchicalPredictionMachine in predictive_processing.go.
//     That was a thesis-level bridge but not a paper-level bridge —
//     winze did not know which paper of Clark's the hypothesis was
//     advanced in. This slice introduces ClarkWhateverNextPaper as a
//     paper-shaped Concept and wires ClarkAuthoredWhateverNext crossing
//     the file boundary. The bridge proves that late-arriving slices
//     can retroactively thicken existing entity neighbourhoods without
//     touching the file that introduced the entity — predictive_processing.go
//     is not edited by this ingest, but AndyClark's effective
//     neighbourhood gains two inbound claims (the Authored and the
//     CommentaryOn target) from this file. Third cross-file user-content
//     entity bridge in winze after AndyClark(←Rajaniemi) and
//     MichaelShermer(←demon_haunted).
//
//   - **Schizophrenia is introduced as a Concept entity and becomes a
//     future-backfill candidate for apophenia.go.** The commentary's
//     reduced-top-down framing is advanced as a TheoryOf(Schizophrenia),
//     which makes Schizophrenia a live Concept in winze for the first
//     time. The apophenia slice already has ConradApopheniaClinicalFraming
//     as a Hypothesis that frames apophenia as a prodromal stage of
//     beginning schizophrenia — a claim TheoryOf(ConradApopheniaClinicalFraming,
//     Schizophrenia) could be added in an apophenia-backfill without
//     any schema work to point Conrad's framing at the same Schizophrenia
//     entity this slice introduces. Not wired here because the apophenia
//     source (Wikipedia / Apophenia) does not name schizophrenia as the
//     Object of Conrad's theory in a way that clearly commits to the
//     same entity; the backfill should come from a slice that reads
//     Conrad's actual 1958 monograph or a secondary source that does
//     explicitly link the two. The opportunity is noted here and left
//     for the right source.
//
//   - **PredictiveProcessing gains its first contested-concept surface.**
//     predictive_processing.go flagged this as a deferred opportunity
//     ("a future slice that reads specific commentaries could earn a
//     contested-concept surface on PredictiveProcessing by adding rival
//     TheoryOf claims"), and this is that slice — almost. The commentary
//     does not propose a rival theory of predictive processing itself;
//     it proposes an application of Clark's framework to schizophrenia
//     and records a specific empirical puzzle (the reduced illusion
//     susceptibility in schizophrenia that Clark's naive bottom-up
//     story does not predict). So the contested-concept rule does not
//     actually fire on PredictiveProcessing via this slice — the
//     commentary's hypothesis is TheoryOf(Schizophrenia), not
//     TheoryOf(PredictiveProcessing). The promotion opportunity is
//     preserved for a future slice that reads a genuine rival framing
//     of the predictive-processing paradigm itself.
//
// Factual notes and discipline:
//
//   - The commentary observes a specific empirical puzzle: under
//     Clark's hierarchical prediction machine, schizophrenia's weakened
//     top-down priors should predict *increased* illusion susceptibility
//     (more reliance on raw sensory input), but the empirical literature
//     instead shows schizophrenic subjects exhibit *reduced* susceptibility
//     to perceptual illusions — specifically the hollow-mask illusion
//     (Schneider et al. 1996, 2002) and the McGurk phenomenon (Pearl et
//     al. 2009). White & Shergill cite Dima et al. 2009's fMRI dynamic
//     causal modelling as evidence that schizophrenia involves
//     "weakening of top-down processes and strengthening of bottom-up
//     processes" — which is the opposite direction from what a naive
//     application of Clark's framework would predict. The commentary's
//     actual contribution is framing this mismatch as a research
//     programme: computational modelling of illusion tasks in
//     schizophrenia to recover the Bayesian priors disrupted by the
//     disorder.
//
//   - The slice does not wire Dima, Pearl, Schneider, Shams, Krabbendam,
//     Ford, Averbeck, or Moritz as entities. None of these researchers
//     are currently in winze, the reference list is short, and the
//     commentary engages with them at citation granularity rather than
//     as intellectual interlocutors. A future slice reading any of the
//     cited original papers could promote them, but this slice would
//     not be any better for reifying citations it only mentions in
//     passing — the over-reification trap the misconceptions slice
//     already surfaced.

var whiteShergillCommentarySource = Provenance{
	Origin:     "Frontiers in Psychology / White TP, Shergill SS (2012). 'Using illusions to understand delusions.' Front. Psychol. 3:407. doi:10.3389/fpsyg.2012.00407",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 5 (white & shergill commentary ingest, scientific-paper shape stress test, cross-file entity bridge)",
	Quote:      "A commentary on 'Whatever next? Predictive brains, situated agents, and the future of cognitive science' by Clark, A. (in press). Behav. Brain Sci. [...] In Bayesian frameworks, mental states arise from integrating encountered data with prior beliefs to form posterior distributions. [...] Clark's synthesis highlights how minimising prediction error can paradoxically generate illusory experiences when stimuli have ambiguous hidden causes. [...] Individuals with schizophrenia show reduced susceptibility to perceptual illusions, including diminished McGurk phenomenon responses (Pearl et al., 2009) and superior detection of hollow-face inversions compared to controls (Schneider et al., 1996, 2002). Dynamic causal modelling of fMRI data suggests this reflects weakening of top-down processes and strengthening of bottom-up processes (Dima et al., 2009). [...] We advocate computational modelling — specifically temporal difference algorithms — to examine Bayesian updating during illusion tasks in schizophrenia.",
}

// -----------------------------------------------------------------------------
// Paper-shaped Concepts. The two papers involved in the commentary
// relationship become Concept entities so CommentaryOn has two endpoints
// to wire. ClarkWhateverNextPaper is the cross-file bridge target:
// predictive_processing.go owns AndyClark, and this file owns the paper
// he authored. No edits to predictive_processing.go are required.
// -----------------------------------------------------------------------------

var (
	ClarkWhateverNextPaper = Concept{&Entity{
		ID:      "concept-clark-2013-whatever-next",
		Name:    "Clark 2013 Whatever next?",
		Kind:    "concept",
		Aliases: []string{"Whatever next? (Clark 2013)", "Clark 2013 target article"},
		Brief:   "Andy Clark's 2013 Behavioral and Brain Sciences target article presenting the predictive processing framework as a unified account of cognition and situating agents in their environments.",
	}}

	WhiteShergillUsingIllusionsCommentary = Concept{&Entity{
		ID:      "concept-white-shergill-2012-using-illusions",
		Name:    "White & Shergill 2012 Using Illusions to Understand Delusions",
		Kind:    "concept",
		Aliases: []string{"White Shergill 2012", "Using Illusions to Understand Delusions"},
		Brief:   "2012 commentary arguing that schizophrenia's reduced susceptibility to visual illusions (hollow-mask, McGurk) contradicts predictive coding predictions of weakened top-down priors, framing this as a computational modelling research programme.",
	}}
)

// -----------------------------------------------------------------------------
// Authors (new Persons) and topic (new Concept).
// -----------------------------------------------------------------------------

var (
	ThomasWhite = Person{&Entity{
		ID:    "thomas-p-white",
		Name:  "Thomas P. White",
		Kind:  "person",
		Brief: "Psychosis researcher at King's College London who authored a 2012 commentary applying perceptual illusions to understand delusions and schizophrenia's computational mechanisms.",
	}}

	SukhiShergill = Person{&Entity{
		ID:    "sukhi-s-shergill",
		Name:  "Sukhi S. Shergill",
		Kind:  "person",
		Brief: "British psychiatrist and neuroscientist at King's College London whose research on sensory attenuation and motor prediction informed the 2012 White & Shergill commentary extending predictive coding theory to self-generated action.",
	}}

	Schizophrenia = Concept{&Entity{
		ID:      "concept-schizophrenia",
		Name:    "Schizophrenia",
		Kind:    "concept",
		Aliases: []string{"schizophrenic disorder"},
		Brief:   "A psychiatric disorder characterized by positive symptoms (hallucinations, delusions), negative symptoms (avolition, flattened affect), and cognitive disorganization.",
	}}
)

// -----------------------------------------------------------------------------
// Hypothesis advanced by the commentary.
// -----------------------------------------------------------------------------

var (
	WhiteShergillReducedTopDownFraming = Hypothesis{&Entity{
		ID:    "hyp-white-shergill-reduced-top-down",
		Name:  "In schizophrenia, top-down predictive processes are weakened and bottom-up sensory processes strengthened — producing reduced susceptibility to perceptual illusions that depend on prior expectation (hollow-mask, McGurk) and implicating disrupted efference-copy prediction of self-generated action as a mechanism for positive symptoms",
		Kind:  "hypothesis",
		Brief: "White & Shergill's 2012 application of Clark's hierarchical prediction framework to schizophrenia, proposing reduced top-down priors relative to bottom-up sensory evidence, predicting diminished susceptibility to illusions and explaining positive symptoms through disrupted motor prediction.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	// Paper-level authorships. ClarkAuthoredWhateverNext is the
	// cross-file entity bridge — its Subject (AndyClark) lives in
	// predictive_processing.go and its Object (ClarkWhateverNextPaper)
	// lives here, without either file being edited by the other.
	ClarkAuthoredWhateverNext = Authored{
		Subject: AndyClark,
		Object:  ClarkWhateverNextPaper,
		Prov:    whiteShergillCommentarySource,
	}
	WhiteAuthoredCommentary = Authored{
		Subject: ThomasWhite,
		Object:  WhiteShergillUsingIllusionsCommentary,
		Prov:    whiteShergillCommentarySource,
	}
	ShergillAuthoredCommentary = Authored{
		Subject: SukhiShergill,
		Object:  WhiteShergillUsingIllusionsCommentary,
		Prov:    whiteShergillCommentarySource,
	}

	// The paper-to-paper structural edge. First CommentaryOn claim in
	// winze; earns the predicate.
	WhiteShergillCommentaryOnClark = CommentaryOn{
		Subject: WhiteShergillUsingIllusionsCommentary,
		Object:  ClarkWhateverNextPaper,
		Prov:    whiteShergillCommentarySource,
	}

	// Hypothesis attribution and theory-of. Attributed to ThomasWhite as
	// first author per winze's Person-level Proposes convention; a
	// multi-author Proposes widening (ProposesCollectively? AuthorsOf
	// widened to a hypothesis slot?) is not forced by this slice because
	// the single-author convention covers the one claim cleanly.
	WhiteProposesReducedTopDownFraming = Proposes{
		Subject: ThomasWhite,
		Object:  WhiteShergillReducedTopDownFraming,
		Prov:    whiteShergillCommentarySource,
	}

	ReducedTopDownTheoryOfSchizophrenia = TheoryOf{
		Subject: WhiteShergillReducedTopDownFraming,
		Object:  Schizophrenia,
		Prov:    whiteShergillCommentarySource,
	}
)
