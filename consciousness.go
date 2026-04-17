package winze

// Consciousness is the central contested concept in the KB's epistemology-
// of-minds domain. Five competing TheoryOf claims from four files:
//
//   1. WattsConsciousnessAsDeadEndThesis  (blindsight.go)
//   2. SearleBiologicalNaturalism         (chinese_room.go)
//   3. ChalmersHardProblemThesis           (hard_problem.go)
//   4. GWTDissociatesIITHypothesis        (theory_seeds.go)
//   5. IITIntractabilityHypothesis        (theory_seeds.go)
//
// Originally defined in blindsight.go (where it was first needed for the
// Watts thesis). Moved here because Consciousness is not about Blindsight —
// it is the hub concept that multiple files converge on.

var Consciousness = Concept{&Entity{
	ID:    "concept-consciousness",
	Name:  "Consciousness",
	Kind:  "concept",
	Brief: "The philosophical and scientific question of what consciousness is, its necessity for intelligence, and its evolutionary role. A contested concept with competing theories including Watts's evolutionary-dead-end thesis, Chalmers's hard problem, and predictive-processing accounts.",
}}
