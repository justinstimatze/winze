package winze

// Depth-first curation: third theories for 6 thin contested concepts.
//
// Each concept previously had exactly 2 TheoryOf subjects. Topology
// flagged these as "thin_contested" — structurally fragile because a
// single retraction would leave zero contestation. Adding a third
// perspective per concept brings each above the fragility threshold.
//
// Sources: all Wikipedia 2025-12 ZIM. Each theory is mirror-source-
// committed: only claims the source explicitly makes, with provenance
// quotes.

// ---------------------------------------------------------------------------
// Source: Wikipedia "Error management theory"
// ---------------------------------------------------------------------------

var emtSource = Provenance{
	Origin:     "Wikipedia 2025-12 / Error_management_theory",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze session 22 (depth-first curation)",
	Quote:      "Error management theory (EMT) is an approach to perception and cognition biases originally coined by David Buss and Martie Haselton.",
}

// ---------------------------------------------------------------------------
// Source: Wikipedia "Dual process theory"
// ---------------------------------------------------------------------------

var dualProcessSource = Provenance{
	Origin:     "Wikipedia 2025-12 / Dual_process_theory",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze session 22 (depth-first curation)",
	Quote:      "Daniel Kahneman provided further interpretation by differentiating the two styles of processing more, calling them intuition and reasoning in 2003. Intuition (or system 1), similar to associative reasoning, was determined to be fast and automatic, usually with strong emotional bonds included in the reasoning process. Reasoning (or system 2) was slower and much more volatile, being subject to conscious judgments and attitudes.",
}

// ---------------------------------------------------------------------------
// Source: Wikipedia "Embodied cognition"
// ---------------------------------------------------------------------------

var embodiedCognitionSource = Provenance{
	Origin:     "Wikipedia 2025-12 / Embodied_cognition",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze session 22 (depth-first curation)",
	Quote:      "Embodied cognition represents a diverse group of theories which investigate how cognition is shaped by the bodily state and capacities of the organism. These embodied factors include the motor system, the perceptual system, bodily interactions with the environment (situatedness), and the assumptions about the world that shape the functional structure of the brain and body of the organism.",
}

// ---------------------------------------------------------------------------
// Source: Wikipedia "Intuitionism"
// ---------------------------------------------------------------------------

var intuitionismSource = Provenance{
	Origin:     "Wikipedia 2025-12 / Intuitionism",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze session 22 (depth-first curation)",
	Quote:      "In the philosophy of mathematics, intuitionism, or neointuitionism (opposed to preintuitionism), is an approach where mathematics is considered to be purely the result of the constructive mental activity of humans rather than the discovery of fundamental principles claimed to exist in an objective reality.",
}

// ---------------------------------------------------------------------------
// Source: Wikipedia "Robert K. C. Forman"
// ---------------------------------------------------------------------------

var formanPCESource = Provenance{
	Origin:     "Wikipedia 2025-12 / Robert_K._C._Forman",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze session 22 (depth-first curation)",
	Quote:      "Yaroslav Komarovski (2015) notes that Forman's notion of a 'pure consciousness event' (PCE) has a very limited applicability in Tibetan Buddhism. According to Komarovski, the realization of emptiness as described in the Buddhist Madhyamaka tradition is different from the PCE.",
}

// ---------------------------------------------------------------------------
// Source: Wikipedia "Dopamine hypothesis of schizophrenia"
// ---------------------------------------------------------------------------

var dopamineHypothesisSource = Provenance{
	Origin:     "Wikipedia 2025-12 / Dopamine_hypothesis_of_schizophrenia",
	IngestedAt: "2026-04-12",
	IngestedBy: "winze session 22 (depth-first curation)",
	Quote:      "The dopamine hypothesis of schizophrenia or the dopamine hypothesis of psychosis is a model that attributes the positive symptoms of schizophrenia to a disturbed and hyperactive dopaminergic signal transduction. The model draws evidence from the observation that a large number of antipsychotics have dopamine-receptor antagonistic effects.",
}

// ===========================================================================
// 1. Apophenia — third theory: Error Management Theory framing
//
// Existing: Conrad (clinical/pathological), Shermer (patternicity/normal variation)
// New: Haselton & Buss (adaptive bias from asymmetric error costs)
// ===========================================================================

var HaseltonBuss = Organization{&Entity{
	ID:    "haselton-buss",
	Name:  "Haselton & Buss",
	Kind:  "organization",
	Brief: "Martie Haselton and David Buss, evolutionary psychologists who coined error management theory (EMT), explaining cognitive biases as adaptive responses to asymmetric error costs.",
}}

var ErrorManagementTheoryOfApophenia = Hypothesis{&Entity{
	ID:    "hyp-error-management-apophenia",
	Name:  "Error management theory of apophenia",
	Kind:  "hypothesis",
	Brief: "Apophenia is an adaptive bias, not a flaw: the cost of missing a real pattern (type II error) was higher than the cost of false detection (type I error) across evolutionary time, so cognition is biased toward over-detection.",
}}

var HaseltonBussProposesEMTApophenia = ProposesOrg{
	Subject: HaseltonBuss,
	Object:  ErrorManagementTheoryOfApophenia,
	Prov:    emtSource,
}

var EMTTheoryOfApophenia = TheoryOf{
	Subject: ErrorManagementTheoryOfApophenia,
	Object:  Apophenia,
	Prov:    emtSource,
}

// ===========================================================================
// 2. CognitiveBias — third theory: Kahneman's dual process framing
//
// Existing: Dimara (task-based classification), Gigerenzer (rational deviation)
// New: Kahneman (System 1/2 dual process)
// ===========================================================================

var DanielKahneman = Person{&Entity{
	ID:    "daniel-kahneman",
	Name:  "Daniel Kahneman",
	Kind:  "person",
	Brief: "Israeli-American psychologist and Nobel laureate. Proposed the dual process theory of cognition (System 1 / System 2) explaining cognitive biases as products of fast automatic processing.",
}}

var KahnemanDualProcessFraming = Hypothesis{&Entity{
	ID:    "hyp-kahneman-dual-process-bias",
	Name:  "Kahneman dual process theory of cognitive bias",
	Kind:  "hypothesis",
	Brief: "Cognitive biases arise from the interaction of two processing systems: System 1 (fast, automatic, intuitive) and System 2 (slow, deliberate, analytical). Biases are systematic errors produced when System 1 generates judgments that System 2 fails to correct.",
}}

var KahnemanProposesDualProcess = Proposes{
	Subject: DanielKahneman,
	Object:  KahnemanDualProcessFraming,
	Prov:    dualProcessSource,
}

var KahnemanDualProcessTheoryOfCognitiveBias = TheoryOf{
	Subject: KahnemanDualProcessFraming,
	Object:  CognitiveBias,
	Prov:    dualProcessSource,
}

// ===========================================================================
// 3. HumanCognition — third theory: Embodied cognition (Varela/Thompson/Rosch)
//
// Existing: Mattson (superior pattern processing), Pinker (evolved modules)
// New: Varela/Thompson/Rosch (embodied mind)
// ===========================================================================

var VarelaThompsonRosch = Organization{&Entity{
	ID:    "varela-thompson-rosch",
	Name:  "Varela, Thompson & Rosch",
	Kind:  "organization",
	Brief: "Francisco Varela, Evan Thompson, and Eleanor Rosch, authors of The Embodied Mind (1991). Proposed that cognition arises from bodily sensorimotor interaction with the environment, not from abstract internal representations.",
}}

var EmbodiedMindThesis = Hypothesis{&Entity{
	ID:    "hyp-embodied-mind",
	Name:  "Embodied mind thesis",
	Kind:  "hypothesis",
	Brief: "Cognition depends on the kinds of experience that come from having a body with sensorimotor capacities embedded in a biological, psychological and cultural context. Challenges computationalism and Cartesian dualism by arguing the body is constitutive of cognition, not merely its vehicle.",
}}

var VarelaThompsonRoschProposesEmbodiedMind = ProposesOrg{
	Subject: VarelaThompsonRosch,
	Object:  EmbodiedMindThesis,
	Prov:    embodiedCognitionSource,
}

var EmbodiedMindTheoryOfHumanCognition = TheoryOf{
	Subject: EmbodiedMindThesis,
	Object:  HumanCognition,
	Prov:    embodiedCognitionSource,
}

// ===========================================================================
// 4. MathematicalFoundations — third theory: Brouwer's intuitionism
//
// Existing: Hilbert (complete axiomatization), Gödel (incompleteness)
// New: Brouwer (mathematics as constructive mental activity)
// ===========================================================================

var LEJBrouwer = Person{&Entity{
	ID:    "lej-brouwer",
	Name:  "L. E. J. Brouwer",
	Kind:  "person",
	Brief: "Dutch mathematician (1881–1966). Founded mathematical intuitionism, arguing that mathematics is the constructive mental activity of humans, not the discovery of objective truths. Rejected the law of excluded middle and actual infinity.",
}}

var BrouwerIntuitionism = Hypothesis{&Entity{
	ID:    "hyp-brouwer-intuitionism",
	Name:  "Brouwer's intuitionism",
	Kind:  "hypothesis",
	Brief: "Mathematics is purely the result of constructive mental activity of humans rather than the discovery of fundamental principles in an objective reality. A mathematical object exists only when it can be constructed; proof by contradiction of nonexistence does not suffice. Rejects the law of excluded middle.",
}}

var BrouwerProposesIntuitionism = Proposes{
	Subject: LEJBrouwer,
	Object:  BrouwerIntuitionism,
	Prov:    intuitionismSource,
}

var IntuitionismTheoryOfMathFoundations = TheoryOf{
	Subject: BrouwerIntuitionism,
	Object:  MathematicalFoundations,
	Prov:    intuitionismSource,
}

// ===========================================================================
// 5. NondualAwareness — third theory: Forman's pure consciousness event
//
// Existing: Perennialism (common core), Constructionism (culturally shaped)
// New: Forman (pure consciousness events are pre-conceptual, neither common
//      core nor culturally constructed)
// ===========================================================================

var RobertForman = Person{&Entity{
	ID:    "robert-forman",
	Name:  "Robert K. C. Forman",
	Kind:  "person",
	Brief: "Former professor of religion at City University of New York. Proposed the 'pure consciousness event' (PCE) as a contentless, pre-conceptual mystical state that challenges both perennialist and constructionist accounts.",
}}

var FormanPCEThesis = Hypothesis{&Entity{
	ID:    "hyp-forman-pce",
	Name:  "Forman's pure consciousness event thesis",
	Kind:  "hypothesis",
	Brief: "Some mystical experiences are 'pure consciousness events' (PCE) — wakeful but contentless states that are pre-conceptual and therefore neither a universal common core (perennialism) nor culturally constructed (constructionism). The PCE is a distinct category that the perennialism-constructionism debate fails to accommodate.",
}}

var FormanProposesPCE = Proposes{
	Subject: RobertForman,
	Object:  FormanPCEThesis,
	Prov:    formanPCESource,
}

var FormanPCETheoryOfNondualAwareness = TheoryOf{
	Subject: FormanPCEThesis,
	Object:  NondualAwareness,
	Prov:    formanPCESource,
}

// ===========================================================================
// 6. Schizophrenia — third theory: Dopamine hypothesis
//
// Existing: White/Shergill (reduced top-down prediction), Mattson (SPP dysregulation)
// New: Dopamine hypothesis (hyperactive dopaminergic signaling)
// ===========================================================================

var DopamineHypothesisOfSchizophrenia = Hypothesis{&Entity{
	ID:    "hyp-dopamine-schizophrenia",
	Name:  "Dopamine hypothesis of schizophrenia",
	Kind:  "hypothesis",
	Brief: "Positive symptoms of schizophrenia arise from disturbed and hyperactive dopaminergic signal transduction, specifically overactivation of D2 receptors. Supported by the observation that antipsychotics are dopamine receptor antagonists and that dopamine agonists can induce psychotic symptoms.",
}}

var DopamineTheoryOfSchizophrenia = TheoryOf{
	Subject: DopamineHypothesisOfSchizophrenia,
	Object:  Schizophrenia,
	Prov:    dopamineHypothesisSource,
}
