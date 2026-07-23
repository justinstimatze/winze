package winze

// Self-directed learning goals — winze's outward-facing curiosity.
//
// Topology's sensor targets are inward-facing: given a structurally fragile
// hypothesis, find external evidence that bears on it. A LearningGoal is the
// other drive — territory the corpus has chosen to acquire. The metabolism
// sense phase reads the specs here and steers sensing toward each goal's seeds,
// alongside the fragility targets it already generates.
//
// The domain boundary is the fork rule. An in-domain goal (InDomain: true)
// deepens THIS corpus — its findings land as Concepts tagged AdvancesGoal. A
// cross-domain goal (InDomain: false) is a fork seed: the corpus holds the
// LearningGoal node as a pointer but never ingests the off-domain material; a
// dedicated fork does that learning, melded back read-only when cross-
// pollination is wanted. That routing is what keeps a technical field learned
// as a hobby (the classic case: quantum computing) from bloating an
// epistemology-of-minds corpus. The fork+meld path is a later layer; the
// in-domain path is live.

// GoalSpec bundles a LearningGoal entity with the parameters the sense phase
// needs. Seeds are the search terms that steer sensing toward the goal's
// territory. CoverAt is the number of AdvancesGoal-tagged Concepts at which the
// goal is satisfied and stops generating sensor targets — a defined "full", so
// a goal is a loop that closes, not an open faucet.
type GoalSpec struct {
	Goal     LearningGoal
	Seeds    []string
	InDomain bool
	CoverAt  int
}

// GoalPredictiveHallucination deepens a thin, in-domain neighborhood: the
// corpus already touches predictive processing (predictive_processing.go,
// white_shergill_commentary.go, the Mattson SPP framing) but the account of
// perception-as-inference and its failure modes is sparse. Squarely inside the
// domain — it is about how a mind's generative model overrides sensory
// evidence — so it deepens main rather than seeding a fork.
var GoalPredictiveHallucination = LearningGoal{&Entity{
	ID:    "goal-predictive-hallucination",
	Name:  "How predictive processing accounts for hallucination",
	Kind:  "learning_goal",
	Brief: "Deepen the corpus's account of perception-as-inference and its failure modes — where a mind's generative model overrides sensory evidence (aberrant precision, reduced top-down constraint).",
}}

var goalPredictiveHallucinationSpec = GoalSpec{
	Goal:     GoalPredictiveHallucination,
	Seeds:    []string{"predictive coding hallucination", "aberrant precision psychosis", "active inference perception"},
	InDomain: true,
	CoverAt:  8,
}

// ActiveLearningGoals is the registry the metabolism sense phase reads.
// Appending a spec here puts its goal into pursuit on the next cycle.
var ActiveLearningGoals = []GoalSpec{
	goalPredictiveHallucinationSpec,
}
