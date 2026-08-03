package winze

// B-shape predicates. Slot types are design-role wrappers so the compiler
// keeps the A- and B-shape families disjoint.
//
// B-shape entities are populated in private corpus slices. These predicates
// and roles are defined here so the public schema is complete and the
// compiler catches slot-type errors in private code. Zero corpus vars
// use these predicates in the public repo by design.

// AppliesToWork: a policy or commitment applies to a creative work.
type AppliesToWork BinaryRelation[AuthorialPolicy, CreativeWork]

// WorkHasLayer: a creative work has an interpretive layer.
type WorkHasLayer BinaryRelation[CreativeWork, DesignLayer]

// WorkHasPhase: a creative work has a production phase.
type WorkHasPhase BinaryRelation[CreativeWork, Phase]

// WorkHasProtectedLine: a creative work contains a protected authored line.
type WorkHasProtectedLine BinaryRelation[CreativeWork, ProtectedLine]

// WorkCommitsToNeverAnswering: a creative work commits to never resolving
// a specific question.
type WorkCommitsToNeverAnswering BinaryRelation[CreativeWork, NeverAnswered]

// LineHasReadingAtLayer: a protected line has a specific reading when the
// player is at a given design layer. Reified via the Reading entity so
// the reading text has a stable home.
type LineHasReadingAtLayer BinaryRelation[ProtectedLine, Reading]

// ReadingAtLayer: a reading is situated at a design layer.
type ReadingAtLayer BinaryRelation[Reading, DesignLayer]

// ReadingAtPhase: a reading is situated at a production phase.
type ReadingAtPhase BinaryRelation[Reading, Phase]
