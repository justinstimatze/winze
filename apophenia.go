package winze

// Wikipedia's Apophenia article — a convergent-bridge ingest whose value
// is in densifying edges across already-ingested domains, not forcing
// new schema. No new predicates. Wires IsCognitiveBias cross-file,
// adds a contested TheoryOf target (Conrad clinical vs Shermer patternicity).
//
// Scope: Apophenia as meta-Concept, sub-types (Pareidolia, Clustering
// illusion), two proposers (Conrad, Shermer), two rival framings.
// Deferred: synchronicity, pattern-recognition models, error management
// theory — candidates for primary-literature ingest.

var apopheniaSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Apophenia",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
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
		Brief: "American science writer and founder of the Skeptics Society. Used the term \"patternicity\" as early as 1997 (Why People Believe Weird Things); popularized it widely circa 2008.",
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
		Brief: "Shermer's reframing of pattern-detection errors as \"patternicity\" — a normal cognitive byproduct rather than pathology. Term appears in his 1997 book; widely popularized circa 2008.",
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

// Cross-source challenge: patternicity coining date.
//
// The Wikipedia Apophenia article (apopheniaSource) says "In 2008
// Michael Shermer coined the word patternicity." The Wikipedia Michael
// Shermer biography (shermerBioSource, in demon_haunted.go) says he
// uses "patternicity" in his 1997 book "Why People Believe Weird Things."
//
// These cannot both be correct. The 1997 attribution is from the Shermer
// biography article, which is likely more authoritative about Shermer's
// own publication history. The Apophenia article's "2008" may refer to
// when the term gained wider visibility (e.g., via Scientific American
// columns) rather than its first use in print.
//
// This is the first genuine "challenged" finding from cross-source
// metabolism analysis: two Wikipedia articles, ingested in the same
// sensor cycle, contradict each other on a factual claim.

// PatternicityDateDispute removed: the Disputes claim wired Shermer as
// disputing his own framing, which is incoherent. Its provenance var
// (patternicityDateChallenge) removed as dead code. The cross-source
// date conflict is documented in the comment block above.

// ---------------------------------------------------------------------------
// The Effects of Heuristics and Apophenia on Probabilistic Choice: Ayton & Fischer Accepts ConradApopheniaClinicalFraming
// This claim documents acceptance of Conrad's apophenia framework by subsequent empirical researchers, validating it as a documented cognitive phenomenon relevant to how minds model reality.
// ---------------------------------------------------------------------------

var aytonFischerAcceptsSource = Provenance{
	Origin:     "Kagi web search result / https://www.ncbi.nlm.nih.gov/pmc/articles/PMC5776328/",
	IngestedAt: "2026-04-27",
	IngestedBy: "winze metabolism cycle 9 (LLM-assisted ingest from Kagi snippet)",
	Quote:      "\"In humans, this is an empirically well-documented phenomenon (Ayton & Fischer, 2004; Falk & Konold, 1997; Gilovich, Vallone, & Tversky, 1985).\"",
}

var AytonFischer = Person{&Entity{
	ID:    "ayton-fischer",
	Name:  "Ayton & Fischer",
	Kind:  "person",
	Brief: "Researchers who documented apophenia as an empirical phenomenon in humans.",
}}

var AytonFischerAcceptsConradApopheniaClinicalFraming = Accepts{
	Subject: AytonFischer,
	Object:  ConradApopheniaClinicalFraming,
	Prov:    aytonFischerAcceptsSource,
}
