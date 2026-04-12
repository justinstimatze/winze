package winze

// Second ingest target: stope-constraints.md from the Stope corpus. A
// deliberately different shape from reggie.go — 90% rules and commitments
// rather than entity-relation claims — used to stress the v0.2 schema in
// a direction it wasn't designed for.
//
// Cross-links to bootstrap.go's Stope var (which is a plain *Entity of
// Kind="project"), re-wrapped here as a CreativeWork to make the role-
// type promise to the B-shape predicate family.

var stopeConstraintsSource = Provenance{
	Origin:     "Stope reference corpus / stope-constraints.md",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze founding session",
	Quote:      "Apply before adding any new content (character, flashback, location, fact, dialogue branch, mechanic). The Check. The Never-Answered List. The Player/Character Prose Gradient. Protected Lines.",
}

// -----------------------------------------------------------------------------
// The creative work the constraints apply to. Cross-ref to bootstrap.go.
// -----------------------------------------------------------------------------

var StopeGame = CreativeWork{Stope}

// -----------------------------------------------------------------------------
// Design layers and phases mentioned in the doc.
// -----------------------------------------------------------------------------

var (
	Layer1 = DesignLayer{&Entity{
		ID:    "stope-layer-1",
		Name:  "Layer 1",
		Kind:  "layer",
		Brief: "Surface reading of the game. Must be a coherent response to real strangeness, not stupid.",
	}}

	Layer2 = DesignLayer{&Entity{
		ID:    "stope-layer-2",
		Name:  "Layer 2",
		Kind:  "layer",
		Brief: "Middle interpretive layer. Declared complete — no future content may imply it is incomplete.",
	}}

	Layer3 = DesignLayer{&Entity{
		ID:    "stope-layer-3",
		Name:  "Layer 3",
		Kind:  "layer",
		Brief: "Deepest interpretive layer. Must remain a reading, never explicitly confirmable. No fourth layer beneath it.",
	}}

	PhaseD = Phase{&Entity{
		ID:    "stope-phase-d",
		Name:  "Phase D",
		Kind:  "phase",
		Brief: "Stranger-traces phase. 'The some' is revealed to be plural — someone was here before the player.",
	}}

	PhaseG = Phase{&Entity{
		ID:    "stope-phase-g",
		Name:  "Phase G",
		Kind:  "phase",
		Brief: "Aftermath phase. 'The some' were the chain participants all along.",
	}}
)

// -----------------------------------------------------------------------------
// A protected line with layered readings.
// -----------------------------------------------------------------------------

var (
	LinePopulationThreeHundred = ProtectedLine{
		Entity: &Entity{
			ID:    "line-pop-300-some",
			Name:  "Population three hundred and some, protected line",
			Kind:  "protected-line",
			Brief: "Arrival paragraph line. Warm on first read, increasingly load-bearing on reread.",
		},
		SourceRef: "engine.py:1076",
		Text:      "Population three hundred and some. You're the some, now.",
	}

	ReadingPop300Layer1 = Reading{
		Entity: &Entity{
			ID:    "reading-pop300-layer1",
			Name:  "Pop-300 Layer 1 reading",
			Kind:  "reading",
		},
		Text: "warm small-town welcome",
	}

	ReadingPop300Layer2 = Reading{
		Entity: &Entity{
			ID:    "reading-pop300-layer2",
			Name:  "Pop-300 Layer 2 reading",
			Kind:  "reading",
		},
		Text: "you're complicit now, you're part of this place",
	}

	ReadingPop300PhaseD = Reading{
		Entity: &Entity{
			ID:    "reading-pop300-phased",
			Name:  "Pop-300 Phase D reading",
			Kind:  "reading",
		},
		Text: "'the some' is plural — someone was here before you. The town's conspicuous indifference about them is the tell, not mysterious absence.",
	}

	ReadingPop300PhaseG = Reading{
		Entity: &Entity{
			ID:    "reading-pop300-phaseg",
			Name:  "Pop-300 Phase G reading",
			Kind:  "reading",
		},
		Text: "'the some' were the chain participants all along",
	}
)

// -----------------------------------------------------------------------------
// Never-answered questions the work commits to never resolving.
// -----------------------------------------------------------------------------

var (
	NAEarlsFinalLine = NeverAnswered{
		Entity: &Entity{
			ID:    "na-earls-final-line",
			Name:  "Earl's final line reception",
			Kind:  "never-answered",
		},
		Question: "Whether Earl heard or received anything before writing his final line.",
	}

	NAWhatTCMeasures = NeverAnswered{
		Entity: &Entity{
			ID:    "na-tc-measures",
			Name:  "What TC measures",
			Kind:  "never-answered",
		},
		Question: "What TC actually measures at a mechanism level — why it rises, what happens at zero.",
	}

	NALayer3True = NeverAnswered{
		Entity: &Entity{
			ID:    "na-layer3-true",
			Name:  "Whether Layer 3 is true",
			Kind:  "never-answered",
		},
		Question: "Whether Layer 3 is true.",
	}
)

// -----------------------------------------------------------------------------
// A couple of authorial policies with their rationales.
// -----------------------------------------------------------------------------

var (
	PolicyNoQuantumFraming = AuthorialPolicy{
		Entity: &Entity{
			ID:    "policy-no-quantum-framing",
			Name:  "Puzzles may not use physics/quantum/geology framing",
			Kind:  "authorial-policy",
		},
		Action:    "cut",
		Rationale: "Puzzles are framed as professional craft. Any framing that invokes time travel, retrocausality, quantum entanglement, or 'science' vocabulary collapses the doubt the game holds open.",
	}

	PolicyNoUIFlagExposureEnding = AuthorialPolicy{
		Entity: &Entity{
			ID:    "policy-no-ui-flag-exposure",
			Name:  "Exposure ending must not be UI-announced",
			Kind:  "authorial-policy",
		},
		Action:    "cut",
		Rationale: "The player learns by noticing something changed on the next run. A UI flag destroys the noticing.",
	}
)

// -----------------------------------------------------------------------------
// Claims wiring everything together.
// -----------------------------------------------------------------------------

var (
	StopeHasLayer1 = WorkHasLayer{Subject: StopeGame, Object: Layer1, Prov: stopeConstraintsSource}
	StopeHasLayer2 = WorkHasLayer{Subject: StopeGame, Object: Layer2, Prov: stopeConstraintsSource}
	StopeHasLayer3 = WorkHasLayer{Subject: StopeGame, Object: Layer3, Prov: stopeConstraintsSource}

	StopeHasPhaseD = WorkHasPhase{Subject: StopeGame, Object: PhaseD, Prov: stopeConstraintsSource}
	StopeHasPhaseG = WorkHasPhase{Subject: StopeGame, Object: PhaseG, Prov: stopeConstraintsSource}

	StopeHasLinePop300 = WorkHasProtectedLine{
		Subject: StopeGame,
		Object:  LinePopulationThreeHundred,
		Prov:    stopeConstraintsSource,
	}

	Pop300ReadsAsLayer1 = LineHasReadingAtLayer{
		Subject: LinePopulationThreeHundred,
		Object:  ReadingPop300Layer1,
		Prov:    stopeConstraintsSource,
	}
	Pop300Layer1SituatedAt = ReadingAtLayer{
		Subject: ReadingPop300Layer1,
		Object:  Layer1,
		Prov:    stopeConstraintsSource,
	}

	Pop300ReadsAsLayer2 = LineHasReadingAtLayer{
		Subject: LinePopulationThreeHundred,
		Object:  ReadingPop300Layer2,
		Prov:    stopeConstraintsSource,
	}
	Pop300Layer2SituatedAt = ReadingAtLayer{
		Subject: ReadingPop300Layer2,
		Object:  Layer2,
		Prov:    stopeConstraintsSource,
	}

	Pop300ReadsAsPhaseD = LineHasReadingAtLayer{
		Subject: LinePopulationThreeHundred,
		Object:  ReadingPop300PhaseD,
		Prov:    stopeConstraintsSource,
	}
	Pop300PhaseDSituatedAt = ReadingAtPhase{
		Subject: ReadingPop300PhaseD,
		Object:  PhaseD,
		Prov:    stopeConstraintsSource,
	}

	Pop300ReadsAsPhaseG = LineHasReadingAtLayer{
		Subject: LinePopulationThreeHundred,
		Object:  ReadingPop300PhaseG,
		Prov:    stopeConstraintsSource,
	}
	Pop300PhaseGSituatedAt = ReadingAtPhase{
		Subject: ReadingPop300PhaseG,
		Object:  PhaseG,
		Prov:    stopeConstraintsSource,
	}

	StopeNeverAnswersEarl   = WorkCommitsToNeverAnswering{Subject: StopeGame, Object: NAEarlsFinalLine, Prov: stopeConstraintsSource}
	StopeNeverAnswersTC     = WorkCommitsToNeverAnswering{Subject: StopeGame, Object: NAWhatTCMeasures, Prov: stopeConstraintsSource}
	StopeNeverAnswersLayer3 = WorkCommitsToNeverAnswering{Subject: StopeGame, Object: NALayer3True, Prov: stopeConstraintsSource}

	StopePolicyNoQuantum       = AppliesToWork{Subject: PolicyNoQuantumFraming, Object: StopeGame, Prov: stopeConstraintsSource}
	StopePolicyNoUIFlagEnding  = AppliesToWork{Subject: PolicyNoUIFlagExposureEnding, Object: StopeGame, Prov: stopeConstraintsSource}
)
