package winze

// Sixteenth public-corpus ingest, session 8 slice 1: Wikipedia's article
// on Gödel's incompleteness theorems. Chosen as the "alien corpus" pressure
// test — the first ingest of formal mathematical content. Every previous
// ingest is narrative-declarative ("person X proposes thesis Y about concept
// Z"). This article's claims are PROVEN THEOREMS: axiom-theorem-proof
// structure, not author-thesis-evidence structure.
//
// Schema forcing functions earned by this slice: NONE. Hypothesis handles
// mathematical theorems cleanly — a theorem IS "a positive claim about X,"
// and the role type carries no truth commitment (proven vs speculative lives
// in Brief prose, not in the type system). This is the eleventh consecutive
// slice in an explored-or-alien source-shape neighbourhood to earn zero new
// primitives.
//
// Pressure test results:
//
//   1. Hypothesis handles mathematical theorems. GodelFirst and
//      GodelSecond are both Hypotheses. The name "Hypothesis" might
//      suggest speculation, but the structural role is identical: a
//      positive claim about a concept, wirable as Subject of TheoryOf
//      and Object of Proposes. The truth-status difference (proven vs
//      speculative) is carried by Brief text, not by the type system.
//      No Theorem role type needed — same discipline as session 7's
//      decision that no ThoughtExperiment role type is needed for the
//      Chinese room.
//
//   2. Hilbert-Gödel dispute fires MathematicalFoundations as 8th
//      contested target. Hilbert's program (positive thesis: all of
//      mathematics can have finitary consistency proofs) and Gödel's
//      first incompleteness theorem (proven result: any sufficiently
//      powerful consistent formal system is incomplete) are both
//      TheoryOf(MathematicalFoundations). The asymmetry — Gödel's
//      result DISPROVES Hilbert's program — does not change the
//      structural representation. Same pattern as Conrad vs Shermer
//      on Apophenia where one framing supersedes another.
//
//   3. Minds and machines section: brief-level adjacency to
//      Consciousness (blindsight.go / chinese_room.go), NOT a
//      TheoryOf-level bridge. Lucas-Penrose and Hofstadter debate
//      what incompleteness implies about human intelligence, but
//      the article presents these as ongoing debates, not committed
//      positions. Following chinese_room.go discipline: do not wire
//      bridges from debate descriptions.
//
// Cross-file bridges:
//   - MathematicalFoundations is a NEW Concept declared in this file.
//     No existing entity to bridge to. Future ingests about formalism,
//     intuitionism, or constructivism would add TheoryOf claims to this
//     target.
//
// Brief-level adjacency (no claim-level bridges):
//   - Consciousness (blindsight.go:105) — the Minds and machines section
//     discusses Lucas-Penrose argument that Gödel's theorem implies minds
//     transcend formal systems. Adjacent to Searle's Chinese room argument
//     (both concern limits of formal symbol manipulation). Not wired
//     because the article presents debate positions, not committed theses.
//   - Hofstadter's "strange loop" thesis about consciousness is mentioned
//     in the Minds and machines section. Would be a TheoryOf(Consciousness)
//     rival if Gödel, Escher, Bach were ingested — but not from this
//     article.
//   - Church-Turing thesis, halting problem, Entscheidungsproblem mentioned
//     in the Relationship with computability section. Not reified — each
//     would need its own article ingest.
//
// Deliberate exclusions:
//   - Proof details (Gödel numbering, diagonalization, arithmetization,
//     Rosser's trick) — technical machinery, not claim-bearing.
//   - Individual critics (Finsler, Zermelo, Wittgenstein) — same
//     discipline as Chinese room respondents. Each would need their own
//     ingest to carry load-bearing claims.
//   - Rosser's improvement, omega-consistency — refinements of the first
//     theorem, not independent theses.
//   - Church-Turing thesis, Turing's halting problem — tempting
//     connections but each needs its own article.
//   - Paraconsistent logic discussion — technical context, not claim-bearing
//     at the level this article commits to.
//   - Lucas-Penrose argument about minds — debate description, not
//     committed thesis.
//   - Hofstadter's strange loop theory — mentioned but not committed to
//     at TheoryOf level.
//
// Scope discipline: the article's technical content (proof sketches,
// examples of undecidable statements, computability relationships) is
// all excluded. The ingest extracts the THESES (what the theorems claim,
// what Hilbert's program proposed) and their structural relationships,
// not the mathematical machinery.

var godelIncompletenessSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Gödel's_incompleteness_theorems",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 8 (Gödel slice, alien corpus pressure test)",
	Quote: "Gödel's incompleteness theorems are two theorems of mathematical logic " +
		"that are concerned with the limits of provability in formal axiomatic " +
		"theories. These results, published by Kurt Gödel in 1931, are important " +
		"both in mathematical logic and in the philosophy of mathematics. The " +
		"theorems are interpreted as showing that Hilbert's program to find a " +
		"complete and consistent set of axioms for all mathematics is impossible. " +
		"[...] The first incompleteness theorem states that no consistent system " +
		"of axioms whose theorems can be listed by an effective procedure (i.e. an " +
		"algorithm) is capable of proving all truths about the arithmetic of " +
		"natural numbers. For any such consistent formal system, there will always " +
		"be statements about natural numbers that are true, but that are unprovable " +
		"within the system. [...] The second incompleteness theorem, an extension " +
		"of the first, shows that the system cannot demonstrate its own " +
		"consistency. [...] Many logicians believe that Gödel's incompleteness " +
		"theorems struck a fatal blow to David Hilbert's second problem, which " +
		"asked for a finitary consistency proof for mathematics. The second " +
		"incompleteness theorem, in particular, is often viewed as making the " +
		"problem impossible.",
}

// -----------------------------------------------------------------------------
// Entities. Seven entities: two persons, the paper (Concept), two theorems
// (Hypothesis — the pressure-test site), one rival thesis (Hypothesis),
// and the TheoryOf target (Concept).
//
// Key vocabulary decision: a mathematical theorem is a Hypothesis.
// The alternative (a new Theorem role type) was rejected for the same
// reason ThoughtExperiment was rejected in session 7: the structural
// role is identical to Hypothesis (positive claim, wirable as Subject
// of TheoryOf), and the distinction (proven vs speculative) lives in
// prose, not in the type system.
// -----------------------------------------------------------------------------

var (
	KurtGodel = Person{&Entity{
		ID:      "kurt-godel",
		Name:    "Kurt Gödel",
		Kind:    "person",
		Aliases: []string{"Gödel", "Kurt Goedel"},
		Brief: "Austrian-American logician and mathematician (1906–1978). Published " +
			"the incompleteness theorems in 1931 in the paper 'Über formal " +
			"unentscheidbare Sätze der Principia Mathematica und verwandter " +
			"Systeme I' (On Formally Undecidable Propositions of Principia " +
			"Mathematica and Related Systems I). The theorems demonstrated " +
			"fundamental limits to provability in formal axiomatic systems, " +
			"showing that Hilbert's program to find a complete and consistent " +
			"set of axioms for all mathematics is impossible. Among the most " +
			"significant results in mathematical logic and the philosophy of " +
			"mathematics.",
	}}

	DavidHilbert = Person{&Entity{
		ID:      "david-hilbert",
		Name:    "David Hilbert",
		Kind:    "person",
		Aliases: []string{"Hilbert"},
		Brief: "German mathematician (1862–1943), one of the most influential " +
			"mathematicians of the late 19th and early 20th centuries. Proposed " +
			"Hilbert's program: the thesis that all of mathematics could be " +
			"given a secure foundation through finitary consistency proofs — " +
			"that a complete, consistent set of axioms for all mathematics " +
			"could be found. His second problem (1900) asked specifically for " +
			"a finitary proof of the consistency of arithmetic. Gödel's " +
			"incompleteness theorems (1931) are widely interpreted as showing " +
			"that this program is impossible, though the precise status of " +
			"Hilbert's second problem remains debated.",
	}}

	OnFormallyUndecidablePropositions1931 = Concept{&Entity{
		ID:      "concept-uber-formal-unentscheidbare-satze-1931",
		Name:    "Über formal unentscheidbare Sätze der Principia Mathematica und verwandter Systeme I",
		Kind:    "concept",
		Aliases: []string{"On Formally Undecidable Propositions", "Gödel 1931"},
		Brief: "Kurt Gödel's 1931 paper, published in Monatshefte für Mathematik, " +
			"presenting the incompleteness theorems. The paper demonstrated that " +
			"any consistent formal system capable of expressing basic arithmetic " +
			"contains statements that are true but unprovable within the system " +
			"(first incompleteness theorem), and that such a system cannot prove " +
			"its own consistency (second incompleteness theorem). The results " +
			"employed a diagonal argument and introduced Gödel numbering — an " +
			"encoding of formal expressions as natural numbers that allows a " +
			"formal system to reason about its own statements. The first " +
			"incompleteness theorem appeared as 'Theorem VI' and the second as " +
			"'Theorem XI' in the paper.",
	}}

	GodelFirstIncompletenessTheorem = Hypothesis{&Entity{
		ID:      "hypothesis-godel-first-incompleteness",
		Name:    "Gödel's first incompleteness theorem",
		Kind:    "hypothesis",
		Aliases: []string{"first incompleteness theorem", "Theorem VI"},
		Brief: "Any consistent formal system within which a certain amount of " +
			"elementary arithmetic can be carried out is incomplete: there are " +
			"statements of the language of the system which can neither be " +
			"proved nor disproved within it. Equivalently, for any such system " +
			"there will always be statements about natural numbers that are " +
			"true but unprovable. Published by Gödel in 1931 as 'Theorem VI'. " +
			"Originally required omega-consistency; strengthened by J. Barkley " +
			"Rosser (1936) to require only consistency. Shows that no " +
			"effectively axiomatized, consistent extension of basic arithmetic " +
			"can be complete — directly undermining Hilbert's program to " +
			"axiomatize all mathematics. This is a PROVEN MATHEMATICAL RESULT, " +
			"not a speculative thesis; it is represented as Hypothesis because " +
			"the role type captures 'positive claim about X' structurally, with " +
			"truth-status carried by prose.",
	}}

	GodelSecondIncompletenessTheorem = Hypothesis{&Entity{
		ID:      "hypothesis-godel-second-incompleteness",
		Name:    "Gödel's second incompleteness theorem",
		Kind:    "hypothesis",
		Aliases: []string{"second incompleteness theorem", "Theorem XI"},
		Brief: "An extension of the first incompleteness theorem: for any " +
			"consistent formal system F within which a certain amount of " +
			"elementary arithmetic can be carried out, the consistency of F " +
			"cannot be proved within F itself. Published by Gödel in 1931 as " +
			"'Theorem XI'. This result is the more direct refutation of " +
			"Hilbert's program, which specifically sought finitary consistency " +
			"proofs — proofs of a system's consistency conducted entirely " +
			"within the system. Gödel's second theorem shows that any such " +
			"proof is impossible for sufficiently powerful consistent systems. " +
			"Together with the first theorem, establishes fundamental limits " +
			"to what formal axiomatic systems can achieve.",
	}}

	HilbertsProgram = Hypothesis{&Entity{
		ID:      "hypothesis-hilberts-program",
		Name:    "Hilbert's program",
		Kind:    "hypothesis",
		Aliases: []string{"Hilbert's second problem", "Hilbert program"},
		Brief: "The thesis, proposed by David Hilbert, that all of mathematics " +
			"can be given a secure foundation through finitary methods: that a " +
			"complete, consistent set of axioms for all mathematics can be " +
			"found, and that the consistency of these axioms can be proved by " +
			"finitary means. Hilbert's second problem (1900) specifically asked " +
			"for a proof of the consistency of arithmetic. This program was " +
			"widely considered dealt a fatal blow by Gödel's incompleteness " +
			"theorems (1931), particularly the second theorem which shows that " +
			"sufficiently powerful consistent systems cannot prove their own " +
			"consistency. A rival TheoryOf MathematicalFoundations to Gödel's " +
			"first incompleteness theorem: where Hilbert claimed complete " +
			"axiomatization is achievable, Gödel proved it is not.",
	}}

	MathematicalFoundations = Concept{&Entity{
		ID:   "concept-mathematical-foundations",
		Name: "Mathematical foundations",
		Kind: "concept",
		Brief: "The study of the logical and philosophical basis of mathematics — " +
			"what axioms mathematics rests on, whether those axioms are complete " +
			"and consistent, and whether mathematical truth can be captured by " +
			"formal systems. Central contested question: can all mathematical " +
			"truth be derived from a finite, consistent set of axioms? Hilbert " +
			"answered yes (Hilbert's program); Gödel proved no (incompleteness " +
			"theorems). Future contested target for ingests about formalism, " +
			"intuitionism (Brouwer), constructivism, or category-theoretic " +
			"foundations. Structurally analogous to Consciousness " +
			"(blindsight.go) and HumanCognition (mattson_pattern_processing.go) " +
			"as a concept that attracts rival theories.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Six claims:
//   1. Authored — Gödel wrote the paper
//   2. Proposes — Gödel advances the first incompleteness theorem
//   3. Proposes — Gödel advances the second incompleteness theorem
//   4. Proposes — Hilbert advances Hilbert's program
//   5. TheoryOf — first incompleteness theorem is a theory of
//                  MathematicalFoundations (one side of contested target)
//   6. TheoryOf — Hilbert's program is a theory of MathematicalFoundations
//                  (other side, fires 8th contested target)
//
// Note: only the first incompleteness theorem carries a TheoryOf claim,
// not the second. Both theorems address mathematical foundations, but
// having both as TheoryOf would create a "co-signed plurality" false
// positive — two complementary results by the same author firing as if
// they were rivals. Same pattern identified in udhr.go (UDHR articles
// are complementary co-signed components, not rival theories). The
// second theorem's relationship to foundations is documented in its Brief.
// -----------------------------------------------------------------------------

var (
	GodelAuthoredOnFormallyUndecidable = Authored{
		Subject: KurtGodel,
		Object:  OnFormallyUndecidablePropositions1931,
		Prov:    godelIncompletenessSource,
	}

	GodelProposesFirstIncompleteness = Proposes{
		Subject: KurtGodel,
		Object:  GodelFirstIncompletenessTheorem,
		Prov:    godelIncompletenessSource,
	}

	GodelProposesSecondIncompleteness = Proposes{
		Subject: KurtGodel,
		Object:  GodelSecondIncompletenessTheorem,
		Prov:    godelIncompletenessSource,
	}

	HilbertProposesHilbertsProgram = Proposes{
		Subject: DavidHilbert,
		Object:  HilbertsProgram,
		Prov:    godelIncompletenessSource,
	}

	GodelFirstIncompletenessTheoryOfMathFoundations = TheoryOf{
		Subject: GodelFirstIncompletenessTheorem,
		Object:  MathematicalFoundations,
		Prov:    godelIncompletenessSource,
	}

	HilbertsProgramTheoryOfMathFoundations = TheoryOf{
		Subject: HilbertsProgram,
		Object:  MathematicalFoundations,
		Prov:    godelIncompletenessSource,
	}
)
