package winze

// Ninth public-corpus ingest: Wikipedia's Apophenia article.
// Selected at the user's explicit prompt as "meta-relevant to several
// other topics" in the winze roadmap — specifically, apophenia is
// the convergent-bridge ingest the slice discipline had been
// approaching without naming: a concept whose value is not in forcing
// new schema but in densifying the edges of the existing graph
// across multiple already-ingested domains.
//
// Schema forcing functions earned by this slice:
//
//   - None. Fourth consecutive slice to earn zero new primitives,
//     after predictive_processing.go, forecasting.go, and this one.
//     Apophenia is a pure **vocabulary-fit + graph-densification**
//     ingest — the discipline win is that existing predicates were
//     already expressive enough and the slice's value is in the
//     edges it wires, not the predicates it invents.
//
// Cross-ingest bridges wired by this slice:
//
//   - **IsCognitiveBias now crosses file boundaries.** Clustering
//     illusion is introduced as a Concept in this file (apophenia.go)
//     but tagged IsCognitiveBias — the same UnaryClaim predicate
//     cognitive_biases.go uses for Availability heuristic, Anchoring,
//     Dunning–Kruger, and Hot-hand. The tag is now cross-file: a
//     query "all IsCognitiveBias subjects" returns claims from two
//     different slices, proving that winze's unary-tag pattern
//     generalises across ingests without coordination. The tag also
//     sits alongside BelongsTo(ClusteringIllusion, Apophenia) —
//     legitimate dual family membership (the clustering illusion is
//     simultaneously a cognitive bias *and* a sub-type of apophenia),
//     which is exactly the non-functional BelongsTo design point
//     validated for the first time.
//
//   - **TheoryOf contested-concept rule gains a fourth target.**
//     The article names two distinct framings of apophenia: Conrad's
//     1958 clinical description (as a beginning-schizophrenia
//     phenomenon) and Shermer's 2008 "patternicity" renaming (as a
//     broader cognitive-secular phenomenon). Both become Hypothesis
//     entities with TheoryOf(Apophenia) claims, and the contested-
//     concept rule will now fire on Apophenia as the fourth contested
//     target across three ingests (Nondualism, NondualAwareness,
//     CognitiveBias, Apophenia). Third ingest domain validating the
//     pragma-driven rule. The Conrad and Shermer framings are not
//     merely renamings — Conrad's embeds apophenia in a pathological
//     schizophrenia-onset frame, Shermer's in a normal-variation
//     evolutionary-psych frame — so the contested structure is real,
//     not a cosmetic disagreement about nomenclature.
//
//   - **Brief-level references into predictive_processing.go.** The
//     Wikipedia article explains apophenia as arising from pattern-
//     recognition mechanisms (template matching, prototype matching,
//     feature analysis), which is exactly the neural substrate Clark
//     2013 theorises about in a completely different domain. No claim
//     is wired between the two because the apophenia article does not
//     cite Clark or predictive processing, and winze's mirror-source-
//     commitments discipline forbids fabricating citation lineage.
//     The connection is noted in Briefs only — a legitimate Brief-
//     level reference, distinct from a claim, that a future ingest
//     reading a source which *does* cite Clark in an apophenia
//     context (Hohwy's Predictive Mind book, possibly; the Friston
//     free-energy-principle literature) can promote to a real
//     InfluencedBy or TheoryOf claim.
//
// Slice scope: Apophenia as meta-Concept, two sub-type Concepts
// (Pareidolia, Clustering illusion), two proposers (Klaus Conrad,
// Michael Shermer), two rival framings as Hypotheses. Deferred:
// synchronicity (the article's mention is too sketchy to honestly
// wire a BelongsTo claim — the text says "can be considered
// synonymous with correlation" without asserting it is a sub-type
// of apophenia); pattern-recognition mechanism models (template/
// prototype/feature matching — these are explanatory mechanisms,
// not framings-of-apophenia, and encoding them as Hypotheses would
// over-read the source); error management theory (evolutionary-
// psychology explanation mentioned only tangentially). All three
// are candidates for a later slice that reads the primary
// psychology literature rather than the Wikipedia stub.

var apopheniaSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Apophenia",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 4 (apophenia ingest, convergent-bridge slice)",
	Quote:      "Apophenia is the tendency to perceive meaningful connections between unrelated things. The term (German: Apophänie from the Greek verb ἀποφαίνειν, romanized: apophaínein) was coined by psychiatrist Klaus Conrad in his 1958 publication on the beginning stages of schizophrenia. He defined it as 'unmotivated seeing of connections [accompanied by] a specific feeling of abnormal meaningfulness'. [...] Pareidolia is a type of apophenia involving the perception of images or sounds in random stimuli. [...] A clustering illusion is a type of cognitive bias in which a person sees a pattern in a random sequence of numbers or events. [...] In 2008 Michael Shermer coined the word patternicity, defining it as 'the tendency to find meaningful patterns in meaningless noise'. [...] In statistics, apophenia is an example of a type I error — the false identification of patterns in data.",
}

// -----------------------------------------------------------------------------
// Core concept, sub-type concepts.
// -----------------------------------------------------------------------------

var (
	Apophenia = Concept{&Entity{
		ID:      "concept-apophenia",
		Name:    "Apophenia",
		Kind:    "concept",
		Aliases: []string{"apophany", "patternicity"},
		Brief:   "The cognitive tendency to perceive meaningful patterns in random or unrelated data. Originally identified by Klaus Conrad in 1958 as a symptom of early schizophrenia; later reconceptualized as \"patternicity\" by Michael Shermer as a general feature of human pattern recognition.",
	}}

	Pareidolia = Concept{&Entity{
		ID:    "concept-pareidolia",
		Name:  "Pareidolia",
		Kind:  "concept",
		Brief: "A perceptual phenomenon where the brain interprets random stimuli as familiar patterns, typically faces or recognizable images, due to overactivity in face recognition areas of the brain.",
	}}

	ClusteringIllusion = Concept{&Entity{
		ID:      "concept-clustering-illusion",
		Name:    "Clustering illusion",
		Kind:    "concept",
		Aliases: []string{"clustering effect"},
		Brief:   "Cognitive bias in which a person perceives patterns in random sequences of numbers or events. A sub-type of apophenia.",
	}}
)

// -----------------------------------------------------------------------------
// People proposing rival framings.
// -----------------------------------------------------------------------------

var (
	KlausConrad = Person{&Entity{
		ID:    "klaus-conrad",
		Name:  "Klaus Conrad",
		Kind:  "person",
		Brief: "German psychiatrist (1905–1961) who coined the term \"apophänie\" to describe unmotivated seeing of connections with abnormal meaningfulness, framing it as an early symptom of schizophrenia onset.",
	}}

	MichaelShermer = Person{&Entity{
		ID:    "michael-shermer",
		Name:  "Michael Shermer",
		Kind:  "person",
		Brief: "American science writer and founder of the Skeptics Society, known for books and columns on science, skepticism, and pseudoscience. He coined the term \"patternicity\" in 2008 to describe the human tendency to find meaningful patterns in meaningless noise.",
	}}
)

// -----------------------------------------------------------------------------
// Rival framings as Hypotheses. Both TheoryOf Apophenia — the
// contested-concept rule will fire on Apophenia as the fourth
// contested target in winze.
// -----------------------------------------------------------------------------

var (
	ConradApopheniaClinicalFraming = Hypothesis{&Entity{
		ID:    "hyp-conrad-apophenia-clinical",
		Name:  "Apophenia is an early clinical sign of beginning schizophrenia — 'unmotivated seeing of connections' in self-referential over-interpretation of sensory perception",
		Kind:  "hypothesis",
		Brief: "A 1958 psychiatric hypothesis framing apophenia as a prodromal stage of schizophrenia characterized by abnormal meaningfulness attributed to real sensory content, distinguished from hallucinations by over-interpretation rather than fabrication.",
	}}

	ShermerPatternicityFraming = Hypothesis{&Entity{
		ID:    "hyp-shermer-patternicity",
		Name:  "Apophenia (renamed 'patternicity') is normal cognitive variation — the tendency to find meaningful patterns in meaningless noise, understandable as an adaptive cost of pattern-recognition machinery",
		Kind:  "hypothesis",
		Brief: "Michael Shermer's 2008 reframing of pattern-detection errors as \"patternicity\"—a normal cognitive byproduct rather than pathology, compatible with evolutionary accounts of false-positive bias as adaptive.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	// Rival theories of apophenia — fires the contested-concept lint
	// rule with Apophenia as the fourth contested target across three
	// ingests (Nondualism, NondualAwareness, CognitiveBias, Apophenia).
	ConradClinicalTheoryOfApophenia = TheoryOf{
		Subject: ConradApopheniaClinicalFraming,
		Object:  Apophenia,
		Prov:    apopheniaSource,
	}
	ShermerPatternicityTheoryOfApophenia = TheoryOf{
		Subject: ShermerPatternicityFraming,
		Object:  Apophenia,
		Prov:    apopheniaSource,
	}

	// Attributions.
	ConradProposesClinicalFraming = Proposes{
		Subject: KlausConrad,
		Object:  ConradApopheniaClinicalFraming,
		Prov:    apopheniaSource,
	}
	ShermerProposesPatternicity = Proposes{
		Subject: MichaelShermer,
		Object:  ShermerPatternicityFraming,
		Prov:    apopheniaSource,
	}

	// Sub-type memberships. BelongsTo is non-functional, so both
	// Pareidolia and Clustering illusion can belong to Apophenia
	// without forcing a role split.
	PareidoliaBelongsToApophenia = BelongsTo{
		Subject: Pareidolia,
		Object:  Apophenia,
		Prov:    apopheniaSource,
	}
	ClusteringIllusionBelongsToApophenia = BelongsTo{
		Subject: ClusteringIllusion,
		Object:  Apophenia,
		Prov:    apopheniaSource,
	}

	// Cross-ingest bridge. The IsCognitiveBias predicate, introduced
	// in cognitive_biases.go for four Estimation-task biases, now
	// crosses file boundaries for the first time. A query for all
	// IsCognitiveBias subjects will return Availability/Anchoring/
	// Dunning-Kruger/Hot-hand + ClusteringIllusion — five biases
	// across two slices, with zero coordination between the files.
	ClusteringIllusionIsCognitiveBiasTag = IsCognitiveBias{
		Subject: ClusteringIllusion,
		Prov:    apopheniaSource,
	}
)
