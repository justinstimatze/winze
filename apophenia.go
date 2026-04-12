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
		Brief:   "The tendency to perceive meaningful connections between unrelated things. In statistics the phenomenon corresponds to a Type I error — false identification of patterns in data. Originally coined by Klaus Conrad in 1958 in a clinical context (beginning-stage schizophrenia), later renamed 'patternicity' by Michael Shermer in 2008 in a broader cognitive-secular frame. Both framings are recorded as TheoryOf(Apophenia) hypotheses so the contested-concept lint rule surfaces them as a fourth disagreement target. Cross-ingest neighbourhood: the pattern-recognition substrate that apophenia over-activates is the same substrate the Clark 2013 predictive-processing ingest models at the neural-implementation level — see predictive_processing.go — and the clustering illusion sub-type is simultaneously tagged IsCognitiveBias, bridging this slice to cognitive_biases.go.",
	}}

	Pareidolia = Concept{&Entity{
		ID:    "concept-pareidolia",
		Name:  "Pareidolia",
		Kind:  "concept",
		Brief: "A sub-type of apophenia involving the perception of images or sounds in random stimuli — faces in inanimate objects, the 'Man in the Moon', religious figures in toast or wood grain. Explained by overactivity of the fusiform face area, the brain region responsible for face recognition, mistakenly matching face-like features against non-face stimuli. Tagged as BelongsTo Apophenia; the name-is-content discipline carries all definitional content here.",
	}}

	ClusteringIllusion = Concept{&Entity{
		ID:      "concept-clustering-illusion",
		Name:    "Clustering illusion",
		Kind:    "concept",
		Aliases: []string{"clustering effect"},
		Brief:   "A cognitive bias in which a person sees a pattern in a random sequence of numbers or events. Explicitly described by the Wikipedia apophenia article as 'a type of cognitive bias' — which is the assertion that earns the IsCognitiveBias tag on this entity, making clustering illusion the first cross-file use of the IsCognitiveBias predicate (introduced in cognitive_biases.go for Availability heuristic, Anchoring, Dunning–Kruger, and Hot-hand). Simultaneously BelongsTo Apophenia as a sub-type. The dual membership (IsCognitiveBias + BelongsTo Apophenia) exercises BelongsTo's non-functionality — a concept legitimately belongs to multiple overlapping families — and is the slice's sharpest validation of that design point. A famous real-world case cited in the source: the early-2000s Queensland ABC studios breast cancer cluster, where incidence was six times the state rate and an investigation found no site- or lifestyle-related cause.",
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
		Brief: "German psychiatrist (1905–1961) who coined the term 'Apophänie' in his 1958 monograph Die beginnende Schizophrenie (The onset of schizophrenia). Conrad's framing of apophenia is clinical: he described it as 'unmotivated seeing of connections [accompanied by] a specific feeling of abnormal meaningfulness', as an early symptom of delusional thought in schizophrenia onset, distinguished from hallucinations by being a self-referential over-interpretation of actual sensory perceptions rather than a fabrication of sensory content.",
	}}

	MichaelShermer = Person{&Entity{
		ID:    "michael-shermer",
		Name:  "Michael Shermer",
		Kind:  "person",
		Brief: "American science writer, historian of science, and founder of the Skeptics Society, best known for popular-audience books and columns on science, skepticism, and pseudoscience. In 2008 he coined the term 'patternicity', defining it as 'the tendency to find meaningful patterns in meaningless noise' — a secular-cognitive reframing of apophenia intended to carry the same phenomenon out of its psychiatric origin into a broader discussion of normal cognitive variation and evolutionary tradeoffs.",
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
		Brief: "Conrad's 1958 clinical framing. The phenomenon is advanced as a prodromal (pre-psychotic) stage of schizophrenia onset, characterised by the abnormal meaningfulness that attaches to random sensory content in the patient's experience, distinct from hallucinations because the sensory content is real but is being over-interpreted rather than fabricated. Under this framing apophenia is a pathological sign whose appropriate reference frame is psychiatric diagnosis.",
	}}

	ShermerPatternicityFraming = Hypothesis{&Entity{
		ID:    "hyp-shermer-patternicity",
		Name:  "Apophenia (renamed 'patternicity') is normal cognitive variation — the tendency to find meaningful patterns in meaningless noise, understandable as an adaptive cost of pattern-recognition machinery",
		Kind:  "hypothesis",
		Brief: "Shermer's 2008 reframing. 'Patternicity' carries the same underlying phenomenon into a non-clinical frame where it is understood as a side-effect of normally-functioning pattern-recognition cognition rather than a pathology. The framing is compatible with evolutionary-psychology accounts (error management theory) which interpret false-positive pattern detection as adaptive under the asymmetric cost of missing a real pattern versus finding a spurious one. Rival to the Conrad clinical framing not because either is factually wrong but because they embed the same phenomenon in two incompatible explanatory frames.",
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
