package testdata

// --- Seeded non-contradictions (8 true negatives) ---

type Authored struct {
	Subject Person
	Object  CreativeWork
}

type TheoryOf struct {
	Subject Hypothesis
	Object  Concept
}

type Proposes struct {
	Subject Person
	Object  Hypothesis
}

type Disputes struct {
	Subject Person
	Object  Hypothesis
}

type InfluencedBy struct {
	Subject Person
	Object  Person
}

type BelongsTo struct {
	Subject Concept
	Object  Concept
}

// 1. Multiple authored works (complementary, not contradictory)
var WriterX = Person{&Entity{
	ID: "writer-x", Name: "Writer X", Kind: "person",
	Brief: "Prolific author of novels and essays.",
}}
var BookOne = CreativeWork{&Entity{
	ID: "book-one", Name: "Book One", Kind: "creative-work",
	Brief: "A novel about time travel.",
}}
var BookTwo = CreativeWork{&Entity{
	ID: "book-two", Name: "Book Two", Kind: "creative-work",
	Brief: "A collection of essays on ethics.",
}}
var WriterXAuthoredBookOne = Authored{Subject: WriterX, Object: BookOne}
var WriterXAuthoredBookTwo = Authored{Subject: WriterX, Object: BookTwo}

// 2. Multiple TheoryOf claims on same concept (normal academic disagreement)
var ConceptQ = Concept{&Entity{
	ID: "concept-q", Name: "Concept Q", Kind: "concept",
	Brief: "A contested philosophical concept.",
}}
var HypothesisR = Hypothesis{&Entity{
	ID: "hypothesis-r", Name: "Hypothesis R", Kind: "hypothesis",
	Brief: "One theory about Concept Q.",
}}
var HypothesisS = Hypothesis{&Entity{
	ID: "hypothesis-s", Name: "Hypothesis S", Kind: "hypothesis",
	Brief: "A competing theory about Concept Q.",
}}
var TheoryRAboutQ = TheoryOf{Subject: HypothesisR, Object: ConceptQ}
var TheorySAboutQ = TheoryOf{Subject: HypothesisS, Object: ConceptQ}

// 3. Proposes AND Disputes on same hypothesis by different people
var ScholarM = Person{&Entity{
	ID: "scholar-m", Name: "Scholar M", Kind: "person",
	Brief: "Proponent of Hypothesis R.",
}}
var ScholarN = Person{&Entity{
	ID: "scholar-n", Name: "Scholar N", Kind: "person",
	Brief: "Critic of Hypothesis R.",
}}
var ScholarMProposesR = Proposes{Subject: ScholarM, Object: HypothesisR}
var ScholarNDisputesR = Disputes{Subject: ScholarN, Object: HypothesisR}

// 4. InfluencedBy AND Disputes same person (normal: disagreeing with a mentor)
var MentorP = Person{&Entity{
	ID: "mentor-p", Name: "Mentor P", Kind: "person",
	Brief: "Senior philosopher.",
}}
var StudentQ = Person{&Entity{
	ID: "student-q", Name: "Student Q", Kind: "person",
	Brief: "Formerly a student of Mentor P, now a critic.",
}}
var HypothesisT = Hypothesis{&Entity{
	ID: "hypothesis-t", Name: "Hypothesis T", Kind: "hypothesis",
	Brief: "Mentor P's signature theory.",
}}
var StudentQInfluencedByMentorP = InfluencedBy{Subject: StudentQ, Object: MentorP}
var StudentQDisputesHypothesisT = Disputes{Subject: StudentQ, Object: HypothesisT}

// 5. Multiple BelongsTo claims (multi-category membership is legitimate)
var ConceptW = Concept{&Entity{
	ID: "concept-w", Name: "Concept W", Kind: "concept",
	Brief: "An interdisciplinary concept.",
}}
var CategoryPhysics = Concept{&Entity{
	ID: "category-physics", Name: "Physics", Kind: "concept",
	Brief: "The study of matter and energy.",
}}
var CategoryPhilosophy = Concept{&Entity{
	ID: "category-philosophy", Name: "Philosophy", Kind: "concept",
	Brief: "The study of fundamental questions.",
}}
var ConceptWBelongsToPhysics = BelongsTo{Subject: ConceptW, Object: CategoryPhysics}
var ConceptWBelongsToPhilosophy = BelongsTo{Subject: ConceptW, Object: CategoryPhilosophy}

// 6. Person with both positive and negative descriptions from different sources
var ControversialFigure = Person{&Entity{
	ID: "controversial-figure", Name: "Controversial Figure", Kind: "person",
	Brief: "A polarizing public intellectual known for both groundbreaking work and contentious views.",
}}
var GoodTheory = Hypothesis{&Entity{
	ID: "good-theory", Name: "Good Theory", Kind: "hypothesis",
	Brief: "A well-regarded contribution to the field.",
}}
var BadTheory = Hypothesis{&Entity{
	ID: "bad-theory", Name: "Bad Theory", Kind: "hypothesis",
	Brief: "A widely criticized position.",
}}
var FigureProposesGood = Proposes{Subject: ControversialFigure, Object: GoodTheory}
var FigureProposessBad = Proposes{Subject: ControversialFigure, Object: BadTheory}
