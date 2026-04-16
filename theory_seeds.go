package winze

// Theory seeds: competing theories of consciousness and human cognition,
// meta-claims about the KB's own limitations, and falsifiable predictions.
//
// Originally ingested from PKM vault (2026-04-15), then refactored into
// core corpus format with proper entity naming and TheoryOf wiring.

// --- provenance ---

var wattsConsciousnessOverheadSource = Provenance{
	Origin:     "PKM vault / echopraxia.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "Non-conscious AI systems will outperform conscious-AI-aspirant systems on specific benchmark tasks by 2030, supporting Watts's thesis that consciousness is computational overhead.",
}

var finiteOntologySource = Provenance{
	Origin:     "PKM vault / finite_ontology_incompleteness.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "The role system will prove insufficient for classifying at least one entity within the next 50 metabolism ingest cycles, requiring either a new role or a multi-role mechanism.",
}

var fepSubsumesPPSource = Provenance{
	Origin:     "PKM vault / free_energy_principle.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "FEP will be formally shown to subsume predictive processing as a special case, with PP's prediction error minimization derivable from FEP's variational inference framework.",
}

var gwtDissociatesIITSource = Provenance{
	Origin:     "PKM vault / global_workspace_theory.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "GWT's neural correlates (late ignition, P300 wave) will be shown to dissociate from IIT's phi measure in at least one experimental paradigm by 2028.",
}

var iitIntractabilitySource = Provenance{
	Origin:     "PKM vault / integrated_information_theory.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "IIT will remain unfalsified but unmeasurable in biological neural networks through 2030 due to computational intractability of phi calculation in systems with more than ~30 nodes.",
}

var substrateIndependenceSource = Provenance{
	Origin:     "PKM vault / permutation_city.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "The substrate independence debate will remain unresolved through 2035 because no agreed-upon consciousness measure exists to test it empirically.",
}

var reificationRiskSource = Provenance{
	Origin:     "PKM vault / reification_risk.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "Entity merges or splits will be necessary as the KB grows past 200 entities, because at least one concept will prove to span multiple incompatible theoretical frameworks.",
}

var selfAuditEpiphenomenalismSource = Provenance{
	Origin:     "PKM vault / self_audit_epiphenomenalism.md",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze pkm-ingest",
	Quote:      "At least one metabolism decision will be demonstrably altered by a bias auditor finding within the next 10 metabolism cycles.",
}

// --- concepts ---

var Echopraxia = Concept{&Entity{
	ID:    "echopraxia",
	Name:  "Echopraxia",
	Kind:  "concept",
	Brief: "Watts's 2014 novel exploring non-conscious agency. Sequel to Blindsight.",
}}

var FiniteOntologyIncompleteness = Concept{&Entity{
	ID:    "finite-ontology-incompleteness",
	Name:  "Finite Ontology Incompleteness",
	Kind:  "concept",
	Brief: "Meta-hypothesis: any finite type system will eventually encounter phenomena it cannot classify with its existing categories. Linked to Gödel incompleteness applied to knowledge representation.",
}}

var FreeEnergyPrinciple = Concept{&Entity{
	ID:    "free-energy-principle",
	Name:  "Free Energy Principle",
	Kind:  "concept",
	Brief: "Karl Friston's (2006) variational framework: biological agents minimize surprise by updating internal models or acting on the world.",
}}

var GlobalWorkspaceTheory = Concept{&Entity{
	ID:    "global-workspace-theory",
	Name:  "Global Workspace Theory",
	Kind:  "concept",
	Brief: "Bernard Baars's (1988) theory of consciousness as global broadcast. Extended by Stanislas Dehaene with neural correlates (P300 wave, late ignition).",
}}

var IntegratedInformationTheory = Concept{&Entity{
	ID:    "integrated-information-theory",
	Name:  "Integrated Information Theory",
	Kind:  "concept",
	Brief: "Giulio Tononi's (2004) theory quantifying consciousness as integrated information (phi). Mathematically precise but computationally intractable for biological networks.",
}}

var PermutationCityConcept = Concept{&Entity{
	ID:    "permutation-city-concept",
	Name:  "Substrate Independence (Permutation City)",
	Kind:  "concept",
	Brief: "Greg Egan's (1994) exploration of consciousness surviving arbitrary substrate changes. Central to the substrate independence debate.",
}}

var ReificationRisk = Concept{&Entity{
	ID:    "reification-risk",
	Name:  "Reification Risk in Typed Knowledge Bases",
	Kind:  "concept",
	Brief: "Meta-hypothesis: pointing multiple theories at a single Concept entity may commit a reification fallacy — assuming the word refers to a single phenomenon when theories actually address different questions.",
}}

var SelfAuditEpiphenomenalism = Concept{&Entity{
	ID:    "self-audit-epiphenomenalism",
	Name:  "Self-Audit Epiphenomenalism",
	Kind:  "concept",
	Brief: "Meta-hypothesis about winze: does the KB's self-modeling (bias audit, topology, calibration) actually change behavior, or is it epiphenomenal — generating reports that never alter decisions?",
}}

// --- hypotheses (falsifiable predictions) ---

var ConsciousnessAsOverheadHypothesis = Hypothesis{&Entity{
	ID:    "consciousness-as-overhead-hypothesis",
	Name:  "Consciousness as computational overhead",
	Kind:  "hypothesis",
	Brief: "Non-conscious AI systems will outperform conscious-AI-aspirant systems on specific benchmark tasks by 2030, supporting Watts's thesis.",
}}

var FiniteOntologyHypothesis = Hypothesis{&Entity{
	ID:    "finite-ontology-hypothesis",
	Name:  "Finite ontology incompleteness prediction",
	Kind:  "hypothesis",
	Brief: "The role system will prove insufficient for classifying at least one entity within 50 metabolism ingest cycles, requiring a new role or multi-role mechanism.",
}}

var FEPSubsumesPPHypothesis = Hypothesis{&Entity{
	ID:    "fep-subsumes-pp-hypothesis",
	Name:  "FEP subsumes predictive processing",
	Kind:  "hypothesis",
	Brief: "FEP will be formally shown to subsume predictive processing, with PP's prediction error minimization derivable from FEP's variational inference framework.",
}}

var GWTDissociatesIITHypothesis = Hypothesis{&Entity{
	ID:    "gwt-dissociates-iit-hypothesis",
	Name:  "GWT dissociates from IIT",
	Kind:  "hypothesis",
	Brief: "GWT's neural correlates (late ignition, P300 wave) will dissociate from IIT's phi measure in at least one experimental paradigm by 2028.",
}}

var IITIntractabilityHypothesis = Hypothesis{&Entity{
	ID:    "iit-intractability-hypothesis",
	Name:  "IIT measurement intractability",
	Kind:  "hypothesis",
	Brief: "IIT will remain unfalsified but unmeasurable in biological neural networks through 2030 due to computational intractability of phi in systems with >30 nodes.",
}}

var SubstrateIndependenceHypothesis = Hypothesis{&Entity{
	ID:    "substrate-independence-hypothesis",
	Name:  "Substrate independence unresolvable",
	Kind:  "hypothesis",
	Brief: "The substrate independence debate will remain unresolved through 2035 because no agreed-upon consciousness measure exists to test it.",
}}

var ReificationRiskHypothesis = Hypothesis{&Entity{
	ID:    "reification-risk-hypothesis",
	Name:  "Reification risk prediction",
	Kind:  "hypothesis",
	Brief: "Entity merges or splits will be necessary past 200 entities, because at least one concept will span multiple incompatible theoretical frameworks.",
}}

var SelfAuditEffectivenessHypothesis = Hypothesis{&Entity{
	ID:    "self-audit-effectiveness-hypothesis",
	Name:  "Self-audit effectiveness prediction",
	Kind:  "hypothesis",
	Brief: "At least one metabolism decision will be demonstrably altered by a bias auditor finding within 10 metabolism cycles.",
}}

// --- TheoryOf claims ---

var ConsciousnessOverheadTheoryOfEchopraxia = TheoryOf{
	Subject: ConsciousnessAsOverheadHypothesis,
	Object:  Echopraxia,
	Prov:    wattsConsciousnessOverheadSource,
}

var FiniteOntologyTheory = TheoryOf{
	Subject: FiniteOntologyHypothesis,
	Object:  FiniteOntologyIncompleteness,
	Prov:    finiteOntologySource,
}

var FEPTheoryOfHumanCognition = TheoryOf{ //winze:contested
	Subject: FEPSubsumesPPHypothesis,
	Object:  HumanCognition,
	Prov:    fepSubsumesPPSource,
}

var GWTTheoryOfConsciousness = TheoryOf{ //winze:contested
	Subject: GWTDissociatesIITHypothesis,
	Object:  Consciousness,
	Prov:    gwtDissociatesIITSource,
}

var IITTheoryOfConsciousness = TheoryOf{ //winze:contested
	Subject: IITIntractabilityHypothesis,
	Object:  Consciousness,
	Prov:    iitIntractabilitySource,
}

var SubstrateIndependenceTheory = TheoryOf{
	Subject: SubstrateIndependenceHypothesis,
	Object:  PermutationCityConcept,
	Prov:    substrateIndependenceSource,
}

var ReificationRiskTheory = TheoryOf{
	Subject: ReificationRiskHypothesis,
	Object:  ReificationRisk,
	Prov:    reificationRiskSource,
}

var SelfAuditTheory = TheoryOf{
	Subject: SelfAuditEffectivenessHypothesis,
	Object:  SelfAuditEpiphenomenalism,
	Prov:    selfAuditEpiphenomenalismSource,
}
