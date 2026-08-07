package winze

// Connections recovered from the trip cycle's isolated log.
//
// 596492b added StructurallyAnalogousTo to the trip emit menu. Before it,
// every cross-cluster analogy was forced to predicate=NONE and written to
// .metabolism-trip-isolated.jsonl, where it never met the critic — those rows
// failed at emit, which is upstream of the gate. 48 distinct pairs
// accumulated that way between 2026-04-28 and the fix.
//
// The critic accepted 5 of the first 10 replayed through --replay-isolated.
// Reading all 48 by hand gave a different answer: they are not 48 insights.
// Eighteen of them say "a system claiming completeness meets what it cannot
// reach", with Godel or Hilbert on one side of every one. Eleven say "a
// two-tier architecture where the upper tier fails to override the lower",
// with Kahneman on one side of nearly all. Five say "an error reframed as
// adaptive", all hanging off Gigerenzer. Promoting the critic's accept list
// would have put five near-duplicate override-failure claims in the corpus.
//
// So three are promoted, one per shape, each the instance whose legs share
// actual parts rather than a theme. The rest stay in the log.

var TripReplayDragonInGarageStructurallyAnalogousToGodelIncompleteness = StructurallyAnalogousTo{
	Subject: DragonInGarageArgument.Entity,
	Object:  GodelFirstIncompletenessTheorem.Entity,
	Prov: Conjecture{
		GeneratedBy: "metabolism-trip-replay",
		Rationale:   `Unfalsifiability and unprovability are the same escape. Sagan's dragon is built so no empirical test can reach it; Godel exhibits a statement no proof inside the system can reach. Both legs carry the same three parts: a validating system, a claim inside its declared scope, and a demonstration that the claim permanently evades it. Recovered by --replay-isolated from a generation predating 596492b, when the trip emit menu had no StructurallyAnalogousTo and forced every cross-cluster analogy to predicate=NONE.`,
		GeneratedAt: "2026-08-07",
	},
}

var TripReplayKahnemanStructurallyAnalogousToWhiteShergillReducedTopDown = StructurallyAnalogousTo{
	Subject: KahnemanDualProcessFraming.Entity,
	Object:  WhiteShergillReducedTopDownFraming.Entity,
	Prov: Conjecture{
		GeneratedBy: "metabolism-trip-replay",
		Rationale:   `Both locate the failure at the weighting between two tiers rather than in either tier's output. System 2 not overriding System 1 maps part-for-part onto reduced top-down priors not constraining bottom-up evidence: a prior, a signal, and a weighting that decides which wins. What breaks is the boundary, not the generation. Recovered by --replay-isolated from a pre-596492b generation forced to predicate=NONE by the missing emit-menu entry.`,
		GeneratedAt: "2026-08-07",
	},
}

var TripReplayShermerPatternicityStructurallyAnalogousToGigerenzerRationalDeviation = StructurallyAnalogousTo{
	Subject: ShermerPatternicityFraming.Entity,
	Object:  GigerenzerRationalDeviationReframing.Entity,
	Prov: Conjecture{
		GeneratedBy: "metabolism-trip-replay",
		Rationale:   `The same argumentative operation performed twice in one field: take a phenomenon classified as cognitive error and reframe it as adaptive under real constraints. Shermer does it for patternicity, Gigerenzer for the heuristics-and-biases programme. Not a resemblance between conclusions but a shared move, with the same before and after. Recovered by --replay-isolated from a pre-596492b generation forced to predicate=NONE by the missing emit-menu entry.`,
		GeneratedAt: "2026-08-07",
	},
}
