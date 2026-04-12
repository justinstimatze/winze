# winze — bootstrap handoff

This file exists only until winze can represent itself well enough that a fresh
session can read the state directly via `defn` instead of needing a prose
handoff. It is the one place markdown is allowed to be load-bearing.

A fresh Claude Code session starting in this directory should:

1. Read `bootstrap.go` — it encodes the founding state of the project as typed
   Go values: entities, decisions, failure modes, mitigations, open questions.
   `go build ./...` passes. `defn` and `adit` are both wired up as MCP servers
   and have live graphs over the whole package.
2. Read this file for the narrative the Go code assumes you already know —
   especially the **Status** section below for what has actually been done,
   which by now is ahead of the founding bootstrap narrative.
3. Run `go run ./cmd/lint .` to see current naming-oracle and orphan-report
   state before touching anything.
4. Pick from the remaining blocking open questions in `bootstrap.go` and
   work on one.

## Status — 2026-04-11

### Session 9 umbrella: benchmark v0.1 — typed retrieval vs grep baseline

Session 9's driving question: **does typed Go storage retrieve knowledge
more accurately than keyword search, and on which query classes does
structure win?** Answered decisively: structured queries achieve perfect
recall; unstructured retrieval barely reaches 50%.

**Benchmark design:** 24 questions across 4 categories (lexical,
aggregation, multi-hop, contested), evaluated against 4 retrieval modes:

| Mode | Description | Overall R@1 |
|------|-------------|-------------|
| grep | keyword match over var-block text | 0.501 |
| bm25 | BM25 ranking over var-block text (pure Go implementation) | 0.467 |
| defn | SQL queries against defn's Dolt database (structured, realistic) | 1.000 |
| ast  | hand-written go/ast queries (structured, ceiling) | 1.000 |

**Per-category results:**

| Category | grep R@1 | bm25 R@1 | defn R@1 | ast R@1 |
|----------|---------|---------|---------|---------|
| Lexical (6) | 0.833 | 0.833 | 1.000 | 1.000 |
| Aggregation (6) | 0.367 | 0.333 | 1.000 | 1.000 |
| Multi-hop (6) | 0.306 | 0.306 | 1.000 | 1.000 |
| Contested (6) | 0.500 | 0.396 | 1.000 | 1.000 |

**Key findings:**

1. **defn SQL achieves perfect recall.** The refs table in defn's Dolt
   database captures Go symbol references, and JOIN chains through
   the ref graph can answer every question — including multi-hop
   traversals and contested-concept aggregation. This validates the
   decision to use Go as the storage substrate: defn's existing
   code-graph infrastructure works as a knowledge-graph query layer
   without modification.

2. **Unstructured retrieval fails on structural queries.** Grep and
   BM25 achieve ~0.83 on lexical lookups but collapse to ~0.30–0.37
   on aggregation and multi-hop. They cannot count claims, traverse
   edges, or detect contested patterns. This is the gap that typed
   storage fills.

3. **BM25 does not beat grep on this corpus.** The var-block granularity
   makes BM25's IDF weighting largely irrelevant — most blocks are
   short enough that term frequency ranking doesn't add much over
   simple keyword counting. Expected to diverge at larger scale.

4. **defn = ast.** The realistic query layer (defn SQL) matches the
   hand-written ceiling (go/ast). This means the query infrastructure
   is already sufficient — the bottleneck for retrieval quality is
   question formulation, not query execution.

**Implementation:** `cmd/benchmark/` — 6 files, ~700 lines. Pure Go,
zero external dependencies (BM25 implemented from scratch). Four
retrieval modes: grep (keyword match), bm25 (BM25 ranking), defn (SQL
via CLI), ast (go/ast queries copied from cmd/lint).

**W1 status: ADDRESSED.** Zero retrieval benchmarks → 24-question
corpus with 4-mode comparison. Not fully resolved — v0.2 needs
counterfactual queries, RippleEdits mutation tests, and LLM-agent
mode. But the v0.1 results are already a concrete, reproducible
demonstration that typed storage beats text search on structural
knowledge queries.

**Deferred to v0.2:**
- Counterfactual queries ("if X hadn't disputed Y...")
- Contradiction detection queries (needs oq-contradiction-rule)
- RippleEdits mutation tests
- adit authoring-cost axis
- LLM-agent-with-tools as fifth mode
- Auto-generated question corpus from KB graph walks

**Lint state after session 9:** unchanged from session 8. 16 roles,
192 entities, 200 referenced, 0 orphaned. 8 contested targets.

### Session 8 umbrella: alien corpus pressure test, first 3-way contested target

Session 8's driving question: **does the predicate vocabulary handle
mathematical results (proven theorems, formal logic, axiom systems)
without new predicates, and can the contested-concept rule handle N>2
rivals on a single target?** Both answered affirmatively.

**Two slices, each answering a distinct question:**

1. **Gödel's incompleteness theorems** (session 8 slice 1 —
   `godel_incompleteness.go`). First ingest of formal mathematical
   content — the "alien corpus" pressure test. Every previous ingest
   is narrative-declarative; this article's claims are proven theorems
   with axiom-theorem-proof structure. **Result: zero new predicates,
   zero new role types.** Hypothesis handles mathematical theorems
   cleanly — a theorem IS "a positive claim about X," and the role
   type carries no truth commitment (proven vs speculative lives in
   Brief prose). No Theorem role type needed, same discipline as
   session 7's rejection of ThoughtExperiment.

   **MathematicalFoundations fires as 8th contested target** (Gödel's
   first incompleteness theorem vs Hilbert's program). Hilbert claimed
   complete axiomatization is achievable; Gödel proved it is not. The
   asymmetry (one refutes the other) does not change the structural
   representation — same pattern as Conrad vs Shermer on Apophenia.

   Entities: KurtGodel (Person), DavidHilbert (Person),
   OnFormallyUndecidablePropositions1931 (Concept),
   GodelFirstIncompletenessTheorem (Hypothesis),
   GodelSecondIncompletenessTheorem (Hypothesis), HilbertsProgram
   (Hypothesis), MathematicalFoundations (Concept). Claims: 1×
   Authored, 3× Proposes, 2× TheoryOf.

   Note: only the first incompleteness theorem carries a TheoryOf
   claim, not the second. Both address mathematical foundations, but
   having both as TheoryOf would create a "co-signed plurality" false
   positive — two complementary results by the same author firing as
   rivals. Same pattern identified in udhr.go.

   Deliberate exclusions: proof details (Gödel numbering,
   diagonalization), critics (Finsler, Zermelo, Wittgenstein),
   Church-Turing thesis, Lucas-Penrose argument about minds and
   machines (debate description, not committed thesis — brief-level
   adjacency to Consciousness only).

2. **Hard problem of consciousness / Chalmers** (session 8 slice 2 —
   `hard_problem.go`). Surgical ingest to fire Consciousness as the
   **first 3-way contested target**. **Result: zero new predicates.**
   Consciousness now has three TheoryOf subjects from three files,
   three sessions, zero coordination:

   - Watts: consciousness may be an evolutionary dead end (blindsight.go, session 6)
   - Searle: consciousness requires specific biological machinery (chinese_room.go, session 7)
   - Chalmers: consciousness is irreducible to physical/functional explanation (hard_problem.go, session 8)

   Third demonstration of seed-and-wait. The contested-concept rule
   handles N>2 cleanly (it reports "3 subjects" — informational, not
   gated).

   Entities: DavidChalmers (Person), HardProblemOfConsciousness
   (Concept), ChalmersHardProblemThesis (Hypothesis). Claims: 1×
   Authored, 1× Proposes, 1× TheoryOf.

   Deliberate exclusions: Type-A through Type-F taxonomy of
   philosophical responses, IIT, Global Workspace Theory, all named
   respondents (Dennett, Churchland, Block, Nagel, etc.).

**Cumulative session 8 delta:**
- 2 slices
- 10 new entities (7 in godel_incompleteness.go, 3 in hard_problem.go)
- 9 new claims (6 in godel_incompleteness.go, 3 in hard_problem.go)
- 0 new predicates, 0 new role types
- 1 new contested target fired (MathematicalFoundations, 8th overall)
- Consciousness upgraded from 2-way to 3-way (first N>2 target)
- Git initialized with initial commit (34 files, 9,883 lines)

**Lint state after session 8:** 16 roles, 192 entities, 200 referenced,
0 orphaned. 8 contested targets (two 3-way: Consciousness, Nondualism;
six 2-way). 3 functional predicates, 3 recorded disputes. CommentaryOn
and AuthoredOrg still at 1 claim each.

**Schema accretion rate:** 0 primitives across 12 consecutive slices in
explored-or-alien neighbourhoods (sessions 5–8). The Gödel ingest
confirms that the boundary is corpus SHAPE (tabular vs declarative),
not content domain — formal mathematics fits existing predicates as
cleanly as philosophy, fiction, or cognitive science.

**Vocabulary pressure test summary (sessions 7-8):**

| test | source | result | finding |
|------|--------|--------|---------|
| thought experiment role type | Chinese room | 0 new | Concept handles it |
| dispute predicate (ArguesAgainst) | Chinese room | 0 new | competing TheoryOf suffices |
| mathematical theorem role type | Gödel | 0 new | Hypothesis handles it |
| proven-vs-speculative distinction | Gödel | 0 new | truth-status in prose, not types |
| N>2 contested target | hard problem | handled | lint reports 3 subjects cleanly |

**Medium-term roadmap:**
- Session 9: Benchmark v0.1 (oq-benchmark) — **DONE** (W1 addressed)
- Session 10: Contradiction detection (oq-contradiction-rule) — W3
- Session 11: Scale push toward 300 entities
- Session 12: Gas Town awareness (oq-gastown-awareness)

**Benchmark v0.2 roadmap** (unblocked after sessions 10-11):
- Counterfactual queries ("if X hadn't disputed Y...")
- Contradiction detection queries (blocked on oq-contradiction-rule, session 10)
- RippleEdits mutation tests (claim mutation → re-query → detect downstream effects)
- adit authoring-cost axis (measure cost of writing queries per mode)
- LLM-agent-with-tools as fifth retrieval mode
- Auto-generated question corpus from KB graph walks (scale to 100+)

**Other pending work:**
- CommentaryOn / AuthoredOrg accretion (still at 1 claim each)
- W3C PROV expansion
- Contested-target-ready singletons: PredictiveProcessing, Forecasting,
  Falsifiability, HumanRights, HumanUniversals, Advaita
- Live KB visualization (gource — refs in memory)
- #15 Slimemold study, #20 Human universals accretion, #21 Skeleton
  extraction, #7 Frontiers Neurosci paper

### Session 7 umbrella: dispute representation pressure test, vocabulary boundary holds

Session 7's driving question: **does the system handle philosophical dispute
without new predicates, and can cross-file bridges be surgically wired?**
Both answered affirmatively.

**Two operations, each answering a distinct question:**

1. **Chinese room / Searle** (session 7 slice 1 — `chinese_room.go`).
   Pressure test for dispute representation and vocabulary boundary.
   Source: Wikipedia article on the Chinese room thought experiment.
   **Result: zero new predicates, zero new role types.** The existing
   vocabulary handles the distinction between a thought experiment
   (Concept) and a philosophical thesis (Hypothesis) cleanly. Dispute
   representation works via competing TheoryOf claims — no
   ArguesAgainst predicate needed. Searle's biological naturalism
   fires Consciousness as the **7th contested target** (Searle vs
   Watts from blindsight.go), second demonstration of seed-and-wait.

   Entities: ChineseRoomArgument (Concept), JohnSearle (Person),
   MindsBrainsAndPrograms1980 (Concept), SearleBiologicalNaturalism
   (Hypothesis). Claims: 2× Authored, 1× Proposes, 1× TheoryOf.

   Deliberate exclusions: functionalism/computationalism/strong AI
   not reified (targets of argument, not positive theses), respondents
   (Dennett, Minsky, Block, Boden, Hofstadter) not reified (parasitic
   without own article ingests), Dennett's eliminative materialism
   deferred (one-sentence treatment insufficient for TheoryOf).

2. **Pinker-Brown InfluencedBy bridge** (edit to `human_universals.go`).
   Surgical cross-file claim wiring deferred from session 6.
   `InfluencedBy(StevenPinker, DonaldEBrown)` using the DePaul course
   page provenance ("republished in Pinker 2002 The Blank Slate").
   **Result: InfluencedBy now spans 2 domains** (quantum_thief.go +
   human_universals.go). First cross-file claim edit: Subject from
   blank_slate.go, Object from human_universals.go, Provenance from
   human_universals.go.

**Session 7 also included an adversarial panel review** comparing winze
to 2026 SOTA (LoCoMo, Karpathy markdown wiki, Letta/MemGPT, GraphRAG,
Mem.ai, Zep, Glean, OWL/SHACL). Key findings:
- Genuinely novel: compile-time ontology enforcement, seed-and-wait
  convergence, provenance-survives-source-deletion
- Weaker than SOTA: zero retrieval benchmarks (W1), no temporal
  reasoning (W2), no semantic contradiction detection (W3)
- Reinventing with better UX: RDF domain/range (R1), triple/property
  partition (R2), W3C PROV simplified (R3)
- User reframe on W4: code-shaped is NOT a domain lock — humans
  interact via agent layer (e.g. Gas Town mayor), never touching
  code directly. The code substrate limits who the direct author
  is (agents), and that's the point.
- Orchestrator-agnostic: Gas Town has momentum but any orchestrator
  works. Not a strategic risk.

**Cumulative session 7 delta:**
- 1 slice + 1 surgical edit
- 4 new entities, 5 new claims (4 in chinese_room.go, 1 in human_universals.go)
- 0 new predicates, 0 new role types
- 1 new contested target fired (Consciousness, 7th total)
- 1 lonely predicate accreted (InfluencedBy → 2 domains)

**Lint state after session 7:** 16 roles, 182 entities, 190 referenced,
0 orphaned. 7 contested targets. 3 functional predicates, 3 recorded
disputes. CommentaryOn and AuthoredOrg still at 1 claim each.

**Contested-target-ready singletons (unfired):** HumanRights (udhr.go),
PredictiveProcessing (predictive_processing.go), Forecasting
(forecasting.go), HumanUniversals (human_universals.go), Falsifiability
(demon_haunted.go), ScientificSkepticism (demon_haunted.go), Advaita
(nondualism.go), plus 3 misconception singletons.

**Schema accretion rate:** 0 primitives across 10 consecutive slices in
explored neighbourhoods (sessions 5–7). ~1 predicate per genuinely new
corpus shape, 0 inside explored shapes.

**Predicate churn test (N=30).** 30 random Wikipedia articles (user-
provided via Wikipedia:Random) analyzed for predicate coverage. Result:
27/30 (90%) need zero new predicates. The 3 failures are all sports-
statistics articles with tabular data (rosters, race results, W/L
records). The vocabulary boundary is at the **corpus shape** level
(tabular vs declarative), not the domain level — biographies, places,
organizations, infrastructure projects, crime events, cultural
practices, biological taxonomy, film lists, and philosophical
biographies all fit existing predicates. BelongsTo handles both
human universals and insect taxonomy (Rhachosaundersia →
Tachinidae). InfluencedBy handles a 17th-century philosopher
(Heereboord ← Descartes) with no changes. The Mathlib parallel
holds: autonomous agents would reuse predicates ≥90% of the time.
The churn risk is scope discipline (declining tabular content),
not vocabulary instability.

### Session 6 umbrella: fiction predicates generalise, seed-and-wait fires

Session 6's driving question: **do single-domain predicates generalise,
and does the seed-and-wait pattern fire?** Both answered affirmatively
with clean experimental evidence across two slices.

**Two slices run in sequence, each answering a distinct question:**

1. **Blindsight by Peter Watts** (session 6 slice 1 —
   `blindsight.go`). Second fiction ingest, chosen to test whether
   AppearsIn / IsFictional / IsFictionalWork / Authored generalise to
   a second author (Watts vs Rajaniemi), a second fictional universe
   (Oort-cloud first contact vs post-singularity heist), and a
   structurally different novel (single book, not trilogy). **Result:
   zero new primitives.** All four fiction predicates carry the content
   unchanged. Fiction-neighbourhood vocabulary-fit confirmed.

   Novel finding: **fiction as a vehicle for real philosophical
   theses.** The Wikipedia article's Consciousness section, sourced
   with multiple academic citations (Shaviro, McGrath, Elber-Aviram),
   commits to Watts advancing the thesis that consciousness may be an
   evolutionary dead end — not in-fiction speculation but a genuine
   philosophical position explored through fiction. Wired as
   `Proposes(PeterWatts, WattsConsciousnessAsDeadEndThesis)` +
   `TheoryOf(WattsConsciousnessAsDeadEndThesis, Consciousness)`,
   demonstrating that the fiction/non-fiction boundary in winze is
   about the CLAIMS, not the FILES. Peter Watts carries both
   Authored (fiction authorship) and Proposes (philosophical thesis)
   on the same Person entity in the same file.

   Consciousness seeded as a contested-target-ready Concept with one
   TheoryOf claim. A rival theory (Chalmers, Dennett, Clark/Hohwy)
   would fire it as a seventh contested target.

   Cross-file bridges: none at claim level. Brief-level adjacency to
   predictive_processing.go (Clark) and mattson_pattern_processing.go
   (SPP is thematically adjacent to Scramblers' intelligence-without-
   consciousness, but no claim wired because the source doesn't cite
   Mattson).

2. **Pinker's The Blank Slate** (session 6 slice 2 —
   `blank_slate.go`). Chosen specifically to fire the seed-and-wait
   pattern on HumanCognition. **Result: HumanCognition fires as the
   6th contested target** — Mattson's SPP thesis
   (mattson_pattern_processing.go, session 5) vs Pinker's
   evolutionary-psychology thesis (blank_slate.go, session 6),
   two files, two sessions, zero coordination. First end-to-end
   demonstration that seeding a TheoryOf singleton and waiting for a
   rival to arrive later works as designed.

   Zero new primitives. Authored + Proposes + TheoryOf carry all
   three claims. Eighth consecutive slice in an explored
   source-shape neighbourhood to earn zero schema changes.

   Cross-file bridge: HumanCognition (mattson_pattern_processing.go)
   is the TheoryOf target — clean cross-file entity bridge without
   editing the source file. DonaldEBrown
   (human_universals.go) is Brief-level adjacency only — the Blank
   Slate Wikipedia article's See Also lists Brown and Human
   Universals, but the body text does not commit to the influence
   at InfluencedBy level. The DePaul source (human_universals.go
   provenance) says Pinker republished Brown's list as an appendix;
   a future edit to human_universals.go could wire InfluencedBy
   using that source's commitment.

   Scope discipline: political-philosophy content (fears of
   inequality, determinism, nihilism, totalitarianism) all declined.
   Same discipline as session 5's UDHR normative deferral.

**Session 6 findings:**

| slice | corpus shape | schema | bridges | test |
|---|---|---|---|---|
| 1. Blindsight | fiction (second author) | 0 | 0 claim-level | fiction predicate generalisation |
| 2. Blank Slate | Wikipedia nonfiction | 0 | +1 (HumanCognition) | seed-and-wait fires |

- **Fiction predicates generalise.** AppearsIn, IsFictional,
  IsFictionalWork, and Authored work unchanged on a second author and
  a second fictional universe. The fiction predicate family is no
  longer single-domain — it has two independent validations across
  structurally different novels.
- **Seed-and-wait works.** HumanCognition was seeded in session 5
  slice 2 (Mattson) and fired in session 6 slice 2 (Pinker). The
  pattern requires zero coordination between the seeding and firing
  slices — each slice only knows about its own source and the shared
  Concept entity. The contested-concept pragma + lint rule handle
  the rest.
- **Fiction-as-thesis-vehicle demonstrated.** Watts carries both
  Authored and Proposes on the same Person entity, and the
  Consciousness Concept seeded by his thesis is a real-world
  contested-target-ready entity, not an in-fiction one. This is a
  novel winze pattern with no session-5 precedent.
- **Schema accretion rate holds.** Session 6: 0 primitives across 2
  slices, both in explored neighbourhoods (fiction, Wikipedia
  nonfiction). Cumulative calibration: accretion lands at ~1 predicate
  per genuinely new corpus shape boundary, 0 per slice inside
  explored neighbourhoods. The session-5 calibration is confirmed,
  not revised.

**Session 6 cumulative delta (after two slices):**

- 2 slices, 11 new entities, 15 new claims, 0 new predicates, 1 new
  contested target fired (HumanCognition, 6th overall), 1 new
  contested-target-ready Concept seeded (Consciousness), 0 new role
  types, 0 new pragmas.
- Cross-file entity bridges: 5 total (pre-session-6: 4; session 6
  adds HumanCognition bridge from blank_slate.go to
  mattson_pattern_processing.go).
- Lint state: 16 roles / 178 role-typed entities / 186 referenced /
  0 orphaned / 3 functional / 3 recorded disputes / 6 contested
  targets. Four rules green throughout session 6.
- Contested-target-ready singletons (unfired): Consciousness
  (blindsight.go), HumanRights (udhr.go), PredictiveProcessing
  (predictive_processing.go), Forecasting (forecasting.go),
  HumanUniversals (human_universals.go), Falsifiability
  (demon_haunted.go), ScientificSkepticism (demon_haunted.go),
  Advaita (nondualism.go), plus 3 misconception singletons.
- Fresh-from-session-5 predicates still at 1 claim: CommentaryOn,
  AuthoredOrg. Neither session-6 slice naturally accreted them —
  recorded as pending validation, not failures.

### Session 5 umbrella: two-sided convergence evidence, density threshold validated, accretion rate calibrated

Session 5 opened post-compact with the specific question session 4 had
left standing unanswered: **is winze's predicate vocabulary actually
converging, or is the five-slice zero-new-primitive streak an artefact
of picking Wikipedia articles that happen to fit?** The discipline win
of running a deliberate experiment across **four** adjacent slices is
that the convergence claim now has multi-sided evidence from five
distinct source shapes, the entity-density finding from session 4 is
both validated and refined, and the schema-accretion *rate* is now a
calibrated number rather than an open question.

**Four slices run in sequence, each answering a distinct sub-question:**

1. **White & Shergill 2012 commentary** (session 5 slice 1 —
   `white_shergill_commentary.go`). Chosen for explicit
   source-shopping: the paper is literally titled "A commentary on
   Clark 2013" and was selected because its citation lineage
   guaranteed a cross-file entity bridge. Earned **one new
   predicate** — `CommentaryOn BinaryRelation[Concept, Concept]` —
   breaking the five-slice vocabulary-fit streak exactly at the
   peer-commentary corpus boundary. Third cross-file user-content
   entity bridge in winze: `AndyClark` becomes the Subject of a
   new Authored claim living in the new file, without
   predictive_processing.go being edited. The first-slice result:
   **schema accretes when a corpus shape demands it.**

2. **Mattson 2014 SPP review** (session 5 slice 2 —
   `mattson_pattern_processing.go`). Chosen as a deliberate
   **control group** for the schema-convergence test: a different
   scientific-paper shape (review article, not commentary), picked
   NOT for entity overlap but for the schema question. Pre-ingest
   expectation: zero-primitives (validating convergence) AND
   zero-bridges (Mattson does not cite Clark, Shermer, Tetlock,
   Conrad, or any other winze Person). **Result: zero primitives
   as expected, BUT an unplanned cross-file entity bridge landed
   anyway** because Mattson commits to a specific claim about
   schizophrenia — the Concept introduced one slice earlier in
   white_shergill_commentary.go — as "a pathological dysregulation
   of the imagination and mental time travel categories of SPP."
   Two findings from the single slice:

   - **Schema stays stable when content does not demand accretion.**
     First vocabulary-fit slice to land **after** a schema-forcing
     slice (rather than as part of an unbroken streak), defeating
     the session-4 "maybe we're just picking fitting sources"
     worry.
   - **Density threshold for entity bridges demonstrated.** Session
     4's discipline was "entity density has to be shopped for".
     Session 5 slice 2 refines it: **once the graph crosses a
     density threshold, bridges land accidentally from honest
     ingestion, without deliberate source-shopping**. Updated
     discipline: shop for sources when the graph is sparse; once
     it is dense enough, accidental bridges start contributing as
     much as shopped ones. Schizophrenia is the threshold-finding
     milestone — introduced session 5 slice 1, landed its first
     contested-concept fire session 5 slice 2, carries three
     inbound claims from two files two slices after introduction.

   Bonus finding: contested target count advances from 4 to 5.
   `Schizophrenia` is the fifth target, and its two rival
   TheoryOf subjects live in two different files written in the
   same session — the **first cross-file cross-session-slice
   contested-concept fire**, proving the pragma-driven design
   holds for disagreement assembled across ingests with no
   coordination.

3. **Brown / Pinker human universals list** (session 5 slice 3 —
   `human_universals.go`). Chosen as the second control group: a
   **list-as-content** corpus shape from a DePaul course handout
   (neither Wikipedia nor peer review), picked to test whether
   yet another distinct source shape forces schema. Closes
   roadmap #19 and advances roadmap #20 simultaneously. **Result:
   zero new primitives, zero new bridges, and the zero-bridges
   outcome is itself informative.** The slice surfaced three
   tempting-but-forbidden bridge opportunities:

   - `BelongsTo(MagicalThinking, HumanUniversals)` — killed by
     verbatim checking. Brown's list has "magic" and "divination"
     (behavioural / ritual items) but NOT "magical thinking"
     (Mattson's cognitive-stance definition). Under
     mirror-source-commitments the two are different concepts
     that happen to share vocabulary.
   - `TheoryOf(BrownHumanUniversalsThesis, HumanCognition)` as a
     rival to Mattson's SPP thesis — killed by source weakness.
     The DePaul handout does not commit to the cognitive-
     architecture framing that would honestly support the claim;
     Pinker's actual text in Blank Slate probably does, but the
     handout is not Pinker's text. Future slice opportunity
     preserved.
   - Three specific-item adjacency bridges: "future, attempts to
     predict" ↔ forecasting.go / predictive_processing.go;
     "classification" ↔ cognitive_biases.go / nondualism.go;
     "figurative speech" and "metaphor" ↔ nondualism.go's
     polyvalent-term discussion. All three would need one slice
     of careful disambiguation work against Brown's specific
     item definition; none were wired here.

   The negative-result finding is important: **the density
   threshold from session 5 slice 2 is NOT a license to lower
   the standard for what counts as a bridge.** Slices that
   honestly cannot bridge should stay unbridged. The discipline
   is more valuable than the graph-density metric.

4. **Universal Declaration of Human Rights 1948** (session 5
   slice 4 — `udhr.go`). Chosen specifically to run the "genuinely
   distant corpus" test the first three slices had deliberately
   deferred: a normative legal document, primary source, drafted
   in committee and adopted by an international body, whose every
   substantive article has the form "everyone has the right to X"
   rather than "X is the case." Two independent schema-pressure
   points: (a) descriptive-vs-normative speech acts at the
   Hypothesis level, and (b) institutional-authorship at the
   document level. **Result: one new predicate earned
   (AuthoredOrg), at the institutional-authorship boundary.** The
   normative-speech-act boundary did NOT force a primitive —
   existing ProposesOrg carries both descriptive and normative
   hypotheses honestly, and a future `IsNormativeClaim` tag is
   available but not forced. Two findings from the single slice:

   - **Schema accretes at a second distinct corpus-shape
     boundary.** Session 5's first forcing function was
     CommentaryOn at the peer-commentary shape (slice 1).
     AuthoredOrg is the second forcing function at the
     institutional-authorship shape. Both are cleanly earned,
     both were explicitly anticipated in their respective
     docstrings, and neither could have been pre-planned from
     pre-session-5 state. The forcing-function discovery pattern
     is: identify a claim whose Subject or Object slot wants a
     type the existing predicate cannot represent without
     concept-conflation, split the predicate along the
     real-typing boundary, document the split as a Go-lacks-
     sum-types work-around rather than a design principle. This
     is now the **third occurrence** of the Person/Organization
     split pattern (after ProposesOrg and MonitoredByOrg) and
     load-bearing enough to name as a recurring discipline.

   - **Slot-type discipline catches a real would-be bug at
     compile time.** The first draft of `udhr.go` attempted to
     wire `UDHR1948 TheoryOf HumanRights` directly. TheoryOf is
     `BinaryRelation[Hypothesis, Concept]` and UDHR1948 is a
     Concept, not a Hypothesis — the Go compiler refused the
     claim at compile time. Had to introduce a routing Hypothesis
     (`UDHRAsTheoryOfHumanRights`) and wire TheoryOf through it
     instead. This is the **first live demonstration in session 5**
     of the role-wrapper system's value as compile-time
     enforcement of the concept-vs-claim distinction, not just
     lint-level documentation. The role wrappers have been
     carrying their weight in Briefs for sessions; this slice is
     the first where that weight was visible in actual write-time
     friction. Worth naming: slot-type discipline is one of
     winze's most load-bearing design decisions and it is now
     demonstrated-not-just-asserted.

   Bonus finding: the three UDHR articles COULD have been wired
   as TheoryOf(HumanRights), which would have fired the
   contested-concept rule on HumanRights as the sixth contested
   target. Declined because the three articles are
   **complementary co-signed components** of a single framework,
   not rival theories — firing the rule on co-signatories would
   be a false positive. The slice surfaces the first known
   case of a distinction the current lint rule cannot
   machine-distinguish: *co-signed plurality vs rival plurality*.
   Both shapes land as multiple Hypothesis subjects pointing at
   the same Concept object, and only Brief text or external
   knowledge distinguishes them. A `//winze:co-signed` pragma
   variant is a natural future refinement when a second case
   forces it.

**Five-sided evidence for the convergence hypothesis:**

| slice | corpus shape | schema | bridges | accretion? |
|---|---|---|---|---|
| 1. White & Shergill | peer commentary | **+CommentaryOn** | +1 shopped | **yes — commentary shape** |
| 2. Mattson SPP | peer review article | 0 | +1 accidental (Schizophrenia) | no |
| 3. Brown universals | course handout list | 0 | 0 (honest negative) | no |
| 4. UDHR 1948 | normative legal document | **+AuthoredOrg** | 0 (Brief-level adjacency) | **yes — institutional authorship** |

Session 5 schema-accretion rate: **2 predicates earned across 4
slices (50%)** when sampling across maximally-distant source
shapes. Each accretion landed at a real forcing boundary (peer
commentary + institutional authorship), was explicitly
anticipated in existing docstrings, and could not have been
pre-planned from pre-session-5 state. Each non-accretion also
landed at a real fit boundary (existing vocabulary carried the
content without conflation). The previous session-1-through-4
observed rate (0 primitives over 5 consecutive slices) was NOT
a converged-forever finding but a **source-neighbourhood
artefact**: staying inside encyclopedic/Wikipedia shape was
hiding the accretion rate at a number much closer to session 5's
50%. The calibrated understanding: winze's schema accretion
rate is roughly *0-1 predicates per distinct corpus shape*,
stabilising to 0 per slice inside an already-explored shape
neighbourhood and spiking to 1 per slice when a new shape
boundary is crossed.

**The open question this session does NOT close:** whether
shapes still more distant than the UDHR (recipe corpus,
mathematical proof, source code, primary historical document)
would force *additional* primitives beyond what session 5's
four slices earned. Session 5 does not pick that fight; the
roadmap has the option preserved. The session's calibrated
answer to "is winze converged?" is "yes, for source shapes
roughly within the session 1-5 coverage envelope; open for
shapes genuinely outside it."

**Session 5 cumulative delta (after four slices):**

- 4 slices, 37 new entities, 33 new claims, **2 new predicates**
  (CommentaryOn + AuthoredOrg), 1 new contested target
  (Schizophrenia, fifth overall), 0 new role types, 0 new
  pragmas, 0 new value structs, 2 contested-target-ready
  Concepts seeded and waiting for rivals (HumanCognition,
  HumanRights).
- 4 cross-file user-content entity bridges now (pre-session-5: 2;
  session 5 adds AndyClark→ClarkWhateverNextPaper and
  Schizophrenia reused across white_shergill_commentary.go and
  mattson_pattern_processing.go; UDHR has no claim-level cross-
  file bridges, only Brief-level adjacency to Brown's human
  universal "law (rights and obligations)").
- Lint state: 16 roles / 167 role-typed entities / 175 referenced
  / 0 orphaned / 3 functional / 3 recorded disputes / 5 contested
  targets. Four rules green throughout session 5.
- Roadmap tasks closed or advanced: #7 (Frontiers paper) completed
  via the pivot from Mattson-as-primary to Mattson-as-control
  after the pre-recon found Mattson unsuitable as bridge target
  and the White & Shergill commentary was shopped as a replacement
  that earned the cross-file bridge; #19 (humunivers inspiration
  read) completed; #20 (Wikipedia human universals ingest project)
  first slice landed with six universals wired and designed for
  incremental accretion. UDHR slice was not a pre-queued roadmap
  item but filled the "genuinely distant corpus shape test" slot
  that had been preserved as an open option since mid-session.

**Session 5 disciplines reinforced or discovered:**

- **Pre-ingest recon is load-bearing.** The Mattson-to-White&Shergill
  pivot happened because a pre-ingest fetch revealed Mattson
  earned zero bridges; the pivot was decided before any code
  was written. Recon is cheap, ingest-rewrites are expensive.
  Future slices should run a one-query pre-recon ("does this
  source cite existing winze entities?") before committing.
- **Density threshold is real but does not soften discipline.**
  Slice 2 demonstrated bridges can land accidentally; slice 3
  demonstrated that the standard for what counts as a bridge
  does not drop. Verbatim checking and mirror-source-commitments
  continue to kill bridges that would be tempting under a
  weaker standard.
- **Parasitic reification is a real failure mode.** The
  Pinker/BlankSlate decision in human_universals.go is the
  clearest instance: both entities would have existed only to
  reify each other, carrying no claims beyond their mutual
  Authored edge. The discipline: only reify entities that
  carry at least one claim beyond existence.
- **Brief-level references are a legitimate permanent state, not
  a placeholder for a future promotion.** Several slices this
  session noted cross-ingest adjacencies in Briefs without
  wiring claims — the Clark ↔ Mattson predictive-processing /
  pattern-processing adjacency, the Brown "magic" ↔ Mattson
  "magical thinking" distinction, the Sagan skepticism ↔
  Brown "myths" distinction. These are not failures of
  ingestion; they are honest records of conceptual
  neighbourhoods that do not share structural commitments in
  the available sources.

### Session 5 fourth slice: UDHR, second schema-forcing slice, slot-type discipline catches a real would-be bug

Universal Declaration of Human Rights 1948 ingest via un.org
primary source (see `udhr.go`). First winze ingest of a
**normative legal document** corpus shape — distinct from every
prior session-5 shape on two axes simultaneously: (1) every
substantive article is a *speech act of declaration* rather than
a claim about the world, and (2) the document is *institutionally
authored* by a committee where no single drafter's contribution
is the whole of the text.

Outcome:

- **One new predicate earned: `AuthoredOrg BinaryRelation[Organization,
  Concept]`.** First institutional-authorship case in winze. The
  existing Authored docstring had explicitly anticipated this
  forcing function since session 4 ("a future slice can add
  AuthoredOrg if collaborative or institutional authorship shows
  up"), and the UDHR is exactly that forcing case — Roosevelt /
  Cassin / Humphrey / Chang / Malik / Mehta all made distinct
  load-bearing contributions, attributing the declaration to any
  one of them would be wrong, attributing it to all six as
  simultaneous Authored claims would erase the institutional
  character of the act, and leaving it un-authored drops a
  load-bearing historical claim. AuthoredOrg is the minimum honest
  primitive. Parallel split to Proposes/ProposesOrg and
  MonitoredBy/MonitoredByOrg — **third occurrence** of the
  "Go-lacks-sum-types work-around: split the predicate when the
  Subject slot wants to span Person and Organization" pattern.
  At three occurrences the split-discipline is load-bearing and
  the Agent-interface-promotion refactor mentioned in Authored's
  docstring is still available but not yet forced.

- **Zero other new primitives.** Declined to earn
  `IsNormativeClaim UnaryClaim[Hypothesis]` as a tag distinguishing
  declarative/normative hypotheses from descriptive/factual ones.
  Tempting because every pre-session-5 Hypothesis in winze is
  descriptive and the three UDHR articles wired here are
  normative, but no claim structurally requires the distinction —
  the same ProposesOrg predicate carries both Tunguska's "2020
  Russian team proposes the comet hypothesis" (descriptive) and
  "UN General Assembly proposes Article 3" (normative), and the
  difference lives in Brief text rather than the claim graph.
  Same deferral reasoning as `IsScientificPaper` in Mattson:
  convenient future tag, not strictly forced yet. Available as a
  future earn when a query actually distinguishes the two
  hypothesis types.

- **Slot-type discipline catches a real would-be bug in draft.**
  The first draft attempted to wire `UDHR1948 TheoryOf HumanRights`
  directly from the document-Concept to the meta-Concept. TheoryOf
  is `BinaryRelation[Hypothesis, Concept]` and UDHR1948 is a
  Concept, not a Hypothesis, so the Go compiler refused the claim
  at compile time. Had to introduce a routing Hypothesis —
  `UDHRAsTheoryOfHumanRights`, expressing the interpretive claim
  that "what the UDHR articulates IS what human rights are" —
  and wire the TheoryOf claim through it instead. This is the
  **first live demonstration** in session 5 of the role-wrapper
  system's value: a structural type error at compile time
  prevented a silent graph-level conflation of document-as-concept
  vs document's-interpretation-as-hypothesis. The role wrappers
  have been carrying their weight in Briefs for sessions; this
  slice is the first where the weight was visible in actual
  write-time friction. Worth naming: the slot-type discipline
  isn't just lint-level documentation, it's **compile-time
  enforcement of the concept-vs-claim distinction** and one of
  winze's most load-bearing design decisions.

- **HumanRights seeded as the second contested-target-ready
  Concept.** Parallel pattern to HumanCognition in
  mattson_pattern_processing.go. Single initial TheoryOf claim
  (UDHRAsTheoryOfHumanRights → HumanRights), waiting for future
  rivals: natural-law tradition (Aquinas, Locke), social-contract
  framings, Amartya Sen / Martha Nussbaum capability approach,
  libertarian negative-rights framings, Islamic / Confucian /
  Ubuntu-based non-Western accounts. Contested-concept rule does
  NOT fire on HumanRights from this slice alone — the entity
  exists to let future slices land rivals zero-touch, same
  pattern as HumanCognition. Winze now carries two
  contested-target-ready Concepts that are seeded but unfired.

- **Deliberate non-fire: per-article TheoryOf(HumanRights) false
  positive.** Could have wired all three UDHR articles as
  TheoryOf(HumanRights) claims, which would fire the
  contested-concept rule on HumanRights as the sixth target and
  hit a round number. Declined because the three articles are
  **complementary co-signed components** of a single framework,
  not rival theories of what human rights are. Firing the
  contested-concept rule on co-signatories would be a false
  positive and would walk back the rule's information value for
  every prior use (where the multiple TheoryOf subjects were
  real disagreements). The slice surfaces a new distinction the
  current lint rule cannot machine-distinguish from claim-graph
  data alone: **co-signed plurality vs rival plurality**. Both
  shapes land as multiple Hypothesis subjects pointing at the
  same Concept object, and only Brief text or external
  knowledge distinguishes them. A future `//winze:co-signed`
  variant of the contested pragma, or a `//winze:rival`
  refinement, could carry the distinction — but no slice
  currently forces it, and this slice just names the edge case
  so the first slice that does force it has prior art to cite.

- **Deferred bridge, third honest negative result in session 5.**
  Brown's human universal "law (rights and obligations)" is
  verbatim on the Brown list (confirmed during the
  human_universals.go slice recon) and looks like it should
  bridge to UDHR1948. But the honest shape is
  `Instantiates[Concept, Concept]` or `Exemplifies[Concept,
  Concept]` — UDHR is a specific document in a specific
  historical context; Brown's "law" is a universal-category
  claim about human societies; BelongsTo would miscategorize the
  abstraction-level mismatch. No slice currently forces the
  Instantiates primitive, and forcing it for a single claim
  would walk back the organic-schema discipline. Brief-level
  bridge only, third honest negative in session 5 after the
  human-universals "magical thinking" killed bridge and the
  human-universals "future/classification/figurative speech"
  adjacency cluster.

- **Fourteen new entities, eleven new claims, no contested state
  changes beyond counts.** 16 roles (unchanged) / 167 role-typed
  entities (+14) / 175 referenced (+14) / 0 orphaned / 3
  functional / 3 recorded disputes / **5 contested targets**
  (unchanged — no fabricated HumanRights fire). Four rules green.

- **Session 5 cumulative after four slices:** 37 new entities,
  33 new claims, 2 new predicates (CommentaryOn + AuthoredOrg),
  1 new contested target (Schizophrenia). Schema accretion rate
  ~50% across four slices when sampling across maximally-distant
  source shapes — encyclopedic / peer commentary / peer review /
  course handout / normative legal document. Five corpus shapes
  pressure-tested; schema accreted at two real forcing
  boundaries (peer commentary shape, institutional authorship)
  and stayed stable at the other three. The convergence
  hypothesis is now robust enough that *both* its accretion
  pattern and its stability pattern are calibrated findings,
  not open questions.

### Session 5 third slice: human universals, zero primitives zero bridges, negative result is informative

Brown / Pinker human universals list ingest via DePaul course
handout (see `human_universals.go`). First winze ingest of a
course-handout corpus shape — neither Wikipedia nor peer review,
and specifically list-as-content (~300 alphabetically-ordered
items with no framing text or categorisation).

Outcome:

- **Zero new primitives.** Authored, Proposes, TheoryOf, and
  BelongsTo cover the entire slice. Second consecutive post-
  forcing vocabulary-fit slice (after Mattson), making three-
  sided evidence for convergence — schema accretes
  (CommentaryOn), stays stable on review article (Mattson),
  stays stable on list-as-content (this slice).

- **Zero cross-file entity bridges, and the zero is informative.**
  The verbatim check killed
  `BelongsTo(MagicalThinking, HumanUniversals)` — Brown has
  "magic" / "divination" / "magic to sustain life" as behavioural
  items but not "magical thinking" as Mattson's cognitive-stance
  concept. The discipline holds: session 5's density-threshold
  finding (bridges land accidentally once the graph is dense) is
  not license to soften the standard for what counts as a
  bridge. Slices that honestly cannot bridge should stay
  unbridged.

- **Three future-bridge opportunities surfaced and deferred
  honestly:** "future, attempts to predict" on Brown's list is
  adjacent to Forecasting/Prediction in forecasting.go and
  HierarchicalPredictionMachine in predictive_processing.go;
  "classification" is adjacent to DimaraTaskBasedClassification
  in cognitive_biases.go and the three rival typologies in
  nondualism.go; "figurative speech" and "metaphor" are
  adjacent to nondualism.go's polyvalent-term discussion. Each
  needs one slice of careful disambiguation work before a
  bridge can land — none forced in this slice.

- **Ten new entities, nine new claims, no lint state changes
  beyond counts.** 16 roles / 153 role-typed entities (+10) /
  161 referenced (+10) / 0 orphaned / 3 functional / 3 recorded
  disputes / 5 contested targets. Four rules green.

- **HumanCognition remains contested-fire-ready but unfired.**
  Mattson's `MattsonSPPThesisTheoryOfHumanCognition` is still
  the only TheoryOf(HumanCognition) claim. Brown's thesis was
  wired as TheoryOf(HumanUniversals) — not HumanCognition —
  because the DePaul handout does not commit to the
  cognitive-architecture framing. A future slice reading
  Pinker's actual Blank Slate text (the canonical Brown-to-
  cognition bridge) would honestly wire the second rival and
  fire the contested-concept rule on HumanCognition for the
  first time.

- **Six universals wired for incremental accretion.** Language,
  Music, Marriage, FearOfDeath, Mythology, ToolMaking. Selected
  for canonical status, flat-noun disambiguation against
  existing winze entities, and cross-domain coverage (linguistic,
  aesthetic, social, existential, cultural, technological)
  without being explicitly grouped that way. The slice shape
  is designed so a future accretion just adds Concept +
  BelongsTo pairs with no schema touches.

### Session 5 second slice: convergence hypothesis gets two-sided evidence, bridges start landing accidentally

Mattson 2014 "Superior pattern processing is the essence of the
evolved human brain" (Frontiers in Neuroscience 8:265) was picked
as the deliberate control-group slice set up by the White & Shergill
finding. The experiment: now that `CommentaryOn` is earned, does a
*different* scientific-paper shape earn further schema, or does it
fit back into the vocabulary cleanly? Pre-ingest outlook was
zero-primitives AND zero-bridges — Mattson does not cite Clark,
does not mention apophenia / patternicity, does not engage any
existing winze Person entity by name.

Outcome (see `mattson_pattern_processing.go`):

- **Zero new primitives earned.** Four existing predicates
  (Authored, Proposes, TheoryOf, BelongsTo) carry the entire slice.
  Seven new entities, seven new claims, zero new role types, zero
  new predicates, zero new pragmas, zero new value structs. This
  is the **first vocabulary-fit slice in winze that has landed
  after a schema-forcing slice**. The prior five-slice vocabulary-
  fit streak (Clark, Forecasting, Apophenia, QT trilogy ext,
  Demon-Haunted) could have been dismissed as an artefact of
  staying inside the Wikipedia neighbourhood; this slice defeats
  that dismissal by fitting cleanly *in spite of* a different
  source shape and *after* the commentary slice just proved
  scientific-paper shape can earn primitives when the content
  demands it. The convergence claim now has two-sided evidence:
  schema accretes when a corpus shape demands it (CommentaryOn in
  white_shergill_commentary.go) and stays stable when it does not
  (this slice). That is stronger evidence than any one-sided test
  could provide.

- **Cross-file entity bridge landed WITHOUT source-shopping.** The
  biggest surprise of the slice. Mattson commits to a specific,
  load-bearing claim about schizophrenia: positive symptoms
  represent "a pathological dysregulation of the imagination and
  mental time travel categories of SPP." That is a Hypothesis
  cleanly attributable to Mattson, and its target Concept is
  `Schizophrenia` — the entity introduced one slice earlier in
  `white_shergill_commentary.go`. Honest ingestion of the source
  (which was picked for the schema experiment, not for entity
  overlap) produced `MattsonSchizophreniaFramingTheoryOfSchizophrenia`
  as a cross-file TheoryOf claim pointing at the same Schizophrenia
  Concept the commentary slice had introduced. This updates the
  session-4 discipline:

    > Session 4: entity density has to be shopped for.
    > Session 5: once the graph crosses a density threshold, bridges
    > also land accidentally. Shop for sources when the graph is
    > sparse; once it is dense enough, honest ingestion produces
    > bridges without the shopping.

  Schizophrenia is a live candidate for the threshold-finding
  milestone: introduced session 5 slice 1, landed its first
  contested-concept fire session 5 slice 2, carries three inbound
  claims from two files two slices after its introduction. The
  density threshold is demonstrated, not just claimed.

- **Fifth contested target, first cross-file contested fire where
  rivals were written in the same session.** `Schizophrenia` joins
  Nondualism×3, NondualAwareness×2, CognitiveBias×2, Apophenia×2
  as the fifth target TheoryOf fires on. The two rival subjects
  are `WhiteShergillReducedTopDownFraming`
  (white_shergill_commentary.go:283) and
  `MattsonSchizophreniaSPPDysregulationFraming`
  (mattson_pattern_processing.go:298). The rule fired pragma-only,
  no lint-binary touches, on its **sixth** ingest-domain validation.
  The cross-file-same-session shape is a first: previously every
  contested fire had its rivals live inside a single file
  (nondualism.go's four rival TheoryOf(Nondualism) claims,
  cognitive_biases.go's two rival TheoryOf(CognitiveBias) claims,
  apophenia.go's two rival TheoryOf(Apophenia) claims). The
  cross-file fire is a stronger test of the pragma-driven design
  because it proves disagreement can be assembled across ingests
  without either file being written with knowledge of the other.

- **New entities and claims.** 7 new role-typed entities:
  SuperiorPatternProcessing (Concept), HumanCognition (Concept),
  MagicalThinking (Concept), Mattson2014SPPPaper (Concept),
  MarkMattson (Person), MattsonSPPThesis (Hypothesis),
  MattsonSchizophreniaSPPDysregulationFraming (Hypothesis). 7 new
  claims: MattsonAuthoredSPPPaper, MattsonProposesSPPThesis,
  MattsonSPPThesisTheoryOfHumanCognition, MagicalThinkingBelongsToSPP,
  MattsonProposesSchizophreniaFraming,
  MattsonSchizophreniaFramingTheoryOfSchizophrenia (the cross-file
  bridge), plus HumanCognition is seeded as a future contested-concept
  target with one initial TheoryOf claim waiting for rivals.

- **Lint state post-slice:** 16 roles / 143 role-typed entities (+7) /
  151 referenced (+7) / 0 orphaned / 3 functional / 3 recorded
  disputes / **5 contested targets** (up from 4). Four rules all
  green.

- **Deliberate exclusions.** Creativity, Language, Reasoning,
  Imagination, MentalTimeTravel — the other four "types of SPP" —
  are listed in the paper but Mattson attaches no further claims
  to them beyond list membership. Reifying them as Concepts with
  BelongsTo claims would add graph noise without adding answerable
  queries (the misconceptions-slice discipline). MagicalThinking
  is the only list item reified because it is the only one with
  substantive claims (definition + religion framing + TMS lateral-
  temporal-lobe evidence). Tenenbaum et al. 2011 is mentioned in
  one sentence as a contrasted framework — not reified as a
  Disputes edge because one-sentence engagement is not a scholarly
  engagement at the commitment level that would justify a
  structural claim.

- **Deferred future opportunity: SPP vs HPM as rival theories of
  human cognition.** Mattson's SPP thesis and Clark's Hierarchical
  Prediction Machine occupy adjacent intellectual real estate but
  target different Concepts at TheoryOf level — HPM TheoryOf
  PredictiveProcessing, SPP TheoryOf HumanCognition. A manual
  "these are rivals" edge would require a new
  `Rivals[Hypothesis, Hypothesis]` predicate that no slice
  currently forces. The finding worth surfacing: the contested-
  concept rule only fires on shared Object slots, so a rivalry
  at hypothesis-level that does NOT share a target Concept is
  invisible to the rule today. A potential future
  `contested-hypothesis` lint rule could cover it once enough
  slices land rival hypothesis pairs that don't share objects.

### Session 5 first slice: vocabulary-fit streak breaks at the scientific-paper boundary

Post-compact session 5 opened by confronting a question session 4 had
left standing — is winze's five-slice zero-new-primitive "vocabulary-fit"
streak a property of the schema having converged for this neighbourhood,
or just a property of all five slices having been Wikipedia articles that
happened to fit? The roadmap's queued #7 (Frontiers in Neuroscience 2014)
was supposed to answer it by stressing scientific-paper shape.

Pre-ingest recon on Mattson 2014 "Superior pattern processing" — the
queued target — killed it as the primary slice without touching winze:
Mattson does not cite Clark 2013, does not mention apophenia /
patternicity / pareidolia, and under mirror-source-commitments
cannot honestly carry any cross-file entity bridge. Session-4's
headline finding was that **entity density has to be shopped for**,
and Mattson fails that test. We swapped to a source-shopped target
instead: **White & Shergill 2012 "Using Illusions to Understand
Delusions"** (Frontiers in Psychology 3:407), which is explicitly
a peer commentary on Clark 2013 and therefore structurally committed
to citing an existing winze entity.

Outcome of the White & Shergill slice (see `white_shergill_commentary.go`):

- **Vocabulary-fit streak broken at the commentary shape.** One new
  predicate earned: `CommentaryOn BinaryRelation[Concept, Concept]`,
  the first paper-to-paper structural edge in winze. It is earned
  because a commentary's relationship to its target cannot be
  reduced to Proposes (Person→Hypothesis), InfluencedBy (Person→
  Person, and collapses to author level), Authored (Person→
  Concept), or DerivedFrom (etymological lineage, not scholarly
  response). The breakage is the finding: vocabulary-fit is *not*
  because the schema has fully converged — it is because the
  previous five slices happened to stay inside the encyclopedic-
  Wikipedia neighbourhood where the existing vocabulary fits. A
  genuinely new corpus shape (peer commentary) earned a primitive
  on the first slice that took it seriously.

- **Third cross-file user-content entity bridge landed, deliberately
  shopped for.** `ClarkWhateverNextPaper` is introduced as a paper-
  shaped Concept in `white_shergill_commentary.go`, and the
  `ClarkAuthoredWhateverNext` claim uses `AndyClark` from
  `predictive_processing.go` as its Subject — a claim crossing
  the file boundary with neither file aware of the other at
  write-time. Predictive_processing.go was not edited; AndyClark's
  effective neighbourhood simply thickened by two inbound claims
  (the Authored + the CommentaryOn target). This is the first slice
  to validate the session-4 thesis *prospectively*: session 4 named
  the discipline ("shop for sources that cite existing winze content")
  and session 5's first slice followed the discipline and earned the
  bridge. Three user-content cross-file bridges now: AndyClark (via
  Rajaniemi + via White&Shergill), MichaelShermer (via demon_haunted),
  and now ClarkWhateverNextPaper itself as the paper-level anchor.

- **New entities and claims.** 6 new role-typed entities:
  ClarkWhateverNextPaper (Concept), WhiteShergillUsingIllusionsCommentary
  (Concept), ThomasWhite (Person), SukhiShergill (Person),
  Schizophrenia (Concept), WhiteShergillReducedTopDownFraming
  (Hypothesis). 6 new claims: ClarkAuthoredWhateverNext (cross-file),
  WhiteAuthoredCommentary, ShergillAuthoredCommentary,
  WhiteShergillCommentaryOnClark (cross-file, first CommentaryOn use),
  WhiteProposesReducedTopDownFraming, ReducedTopDownTheoryOfSchizophrenia.

- **Lint state post-slice:** 16 roles / 136 role-typed entities (+6) /
  144 referenced (+6) / 0 orphaned / 3 functional / 3 recorded disputes
  / 4 contested targets (unchanged — commentary did not earn a rival
  TheoryOf on PredictiveProcessing because its framing is an
  application of Clark's framework to Schizophrenia, not a rival
  theory *of* predictive processing itself).

- **Deferred, not earned:** IsScientificPaper UnaryClaim[Concept] —
  the paper-as-Concept pattern is self-identifying via Authored +
  CommentaryOn context, no query needs the tag yet. Paper-to-figure
  / section / citation granularity — two-page commentary does not
  force it. Will be earned the first time a research-article slice
  actually needs figure-level claims. ProposedIn/Advances as a
  Hypothesis→Paper predicate — same reasoning, Brief-level linkage
  currently covers every query that would use it.

- **Deferred future backfill:** ConradApopheniaClinicalFraming in
  `apophenia.go` could TheoryOf(Schizophrenia) pointing at the new
  Schizophrenia Concept entity — would earn a second cross-file
  bridge via schizophrenia rather than via Clark. Explicitly not
  wired here because the current Wikipedia Apophenia Provenance
  does not commit directly enough. The right source to promote it
  is Conrad's 1958 monograph or a secondary source that explicitly
  claims the link; the opportunity is preserved in the header of
  `white_shergill_commentary.go`.

Session 5 implication for the roadmap: the Mattson 2014 target isn't
dead, it just becomes a different kind of slice. Now that CommentaryOn
is earned, Mattson is the **control group** for the convergence
hypothesis — if a pattern-processing review article earns *zero* new
primitives when ingested *after* a schema-forcing slice, it's the
first genuine vocabulary-fit ingest following a schema-accretion event
and measures convergence from the opposite direction. That is the
exact shape of slice the session-4 worry ("are we just picking
fitting sources?") wanted.

### Session 4 umbrella: predicate density is real, entity density was not — until now

Session 4 ran five consecutive ingests (Quantum Thief trilogy, Clark
predictive processing, Forecasting, Apophenia, Demon-Haunted World)
plus an out-of-band query session on the graph itself. The query
session exposed a structural finding sharp enough to change how the
rest of the session picked its moves and worth recording as its own
calibration milestone:

**Winze's cross-ingest density is at the predicate-type level, not
the entity level.**

Run against a 120-entity / 128-claim graph, the query "which entities
are referenced from files other than their own" returned exactly
three results: `Stope` (bootstrap.go → stope_constraints.go,
founding-session scaffolding), `ExternalTerms` (external.go →
cmd/lint/main.go, infrastructure), and `AndyClark`
(predictive_processing.go → quantum_thief.go, the InfluencedBy bridge
session 4 had just wired). In the entire user-content claim graph —
nine ingest slices spanning environmental monitoring, Tunguska,
Nondualism, Tunguska energy, user memory, cognitive biases,
misconceptions, Quantum Thief trilogy, Clark 2013, Forecasting, and
Apophenia — **only one non-infrastructure entity was referenced
across file boundaries before session 4 closed the gap.**

Meanwhile predicate-type reuse was healthy and cross-file: Proposes
(17 claims / 6 files), TheoryOf (15/6), BelongsTo (12/4),
IsCognitiveBias (5/2 after the apophenia bridge), Disputes (5/2),
IsPolyvalentTerm (2/2). Twenty-plus predicates in live cross-file
use, plus the contested-concept rule firing on four targets across
three ingests with pragma-driven generalisation. The "density" the
session had been claiming was real — but it was a **predicate** density,
where the schema vocabulary earned cross-file reuse, not an **entity**
density where individual concepts and people were referenced from
multiple files. Every ingest slice was its own silo, connected only
through the shared predicate types.

This flipped the mid-session plan. Demon-Haunted World was promoted
from "fifth in the queue" to "next, specifically because it can
deliberately shop for existing entities to reference". The slice
paired Sagan's 1995 book with targeted facts from the Michael Shermer
Wikipedia biography article, because Shermer already existed as a
Person entity in apophenia.go and the biography article provided
citable facts about him: his founding of the Skeptics Society in
1991, his authorship of Why People Believe Weird Things in 1997. Two
claims in demon_haunted.go — `ShermerAffiliatedWithSkepticsSociety`
and `ShermerAuthoredWhyPeopleBelieveWeirdThings` — point at the
MichaelShermer entity from apophenia.go. These are the **first real
entity-level cross-file bridges in winze that are not scaffolding
or the single Clark link**. The query repeated after the slice landed
returned three non-infrastructure entities with external references:
Stope, AndyClark, and MichaelShermer. Entity-level density goes from
"one" to "two" in a single deliberate slice — proof that the density
was achievable all along; the earlier slices had simply not been
choosing sources with real citation overlap into winze's existing
content.

### Session 4: five consecutive vocabulary-fit ingests + one schema pivot

Across the session, five slices earned **zero new primitives** in a row:

- **Clark 2013 predictive processing** (predictive_processing.go) — 3
  entities, 2 claims, entirely reusing Proposes + TheoryOf. Paired
  with Forecasting per the user's observation that the two are
  "very related although technically on slightly different time
  scales" — predictive processing at milliseconds in neural
  substrate, explicit forecasting at days to years in human
  practice. The pairing earned **one** new predicate
  (`InfluencedBy BinaryRelation[Person, Person]`), which is defined
  in predicates.go but earns its keep across two slices at once:
  the Rajaniemi→Clark bridge in quantum_thief.go uses AndyClark from
  predictive_processing.go, and the backfilled Rajaniemi→Leblanc
  bridge uses the new MauriceLeblanc Person entity. First cross-
  ingest edge wired as a real claim in winze.

- **Forecasting** (forecasting.go) — 6 entities, 7 claims, zero new
  predicates. The Forecasting Wikipedia article is meta-level and
  asserts no specific dated forecast, so the temporal-prediction
  schema that was theoretically forced by the Clark/Forecasting
  pairing was **not** actually earned. Instead the slice reused
  IsPolyvalentTerm (hydrology vs general usage), TheoryOf
  (Tetlock's calibration framing), and BelongsTo (qualitative vs
  quantitative method taxonomy). A deferred-schema comment in the
  file captures the user's explicit observation that any future
  Predicts predicate family will need wide-range time-scale
  representation: milliseconds (neural) through decades+ (climate)
  in a single TemporalMarker promotion, since the current
  TemporalMarker is a free-text string introduced for geological-
  era dates.

- **Apophenia** (apophenia.go) — 7 entities, 7 claims, zero new
  predicates. The first slice deliberately picked for **graph
  densification** rather than new schema: the user had flagged
  apophenia as "meta-relevant to several other topics". The slice
  wired ClusteringIllusion as a new Concept in apophenia.go but
  tagged it IsCognitiveBias, making IsCognitiveBias the first
  cross-file UnaryClaim tag in winze — five bias claims across two
  files, zero coordination. Two rival framings (Conrad's 1958
  clinical origin, Shermer's 2008 patternicity reframing) landed
  as TheoryOf Apophenia, bringing the contested-concept rule to
  four targets across three ingests.

- **Demon-Haunted World** (demon_haunted.go) — 10 entities, 9
  claims, zero new predicates. Fifth consecutive zero-primitive
  slice, and first to deliberately earn entity-level cross-file
  bridges via the two Shermer claims described above. Carl Sagan
  planted as a hub entity for future skepticism / philosophy-of-
  science ingests; SkepticsSociety, baloney-detection-kit,
  dragon-in-garage, and Why People Believe Weird Things accreted
  as real content. Factual dispute noted but not reified: the
  Apophenia article and the Shermer biography article disagree on
  whether patternicity was coined in 2008 or 1997. The slice
  records the disagreement in Briefs only, refusing to invent a
  CoinedIn[Concept, TemporalMarker] functional predicate for a
  single occurrence.

- **Quantum Thief trilogy extension + predictive_processing bridge**
  — see the dedicated section below. Earned 4 new predicates
  (IsFictionalWork, IsFictional, AppearsIn, Authored) plus the
  InfluencedBy bridge listed above.

### Discipline findings earned in session 4

- **The "vocabulary-fit ingest" pattern now has a name.** A slice
  whose discipline-win is that existing predicates already fit the
  source, so inventing structure would bloat the predicate graph
  without yield. Five consecutive occurrences make the pattern real,
  not a coincidence. The selection-bias question this raises —
  whether the vocabulary is actually complete for the current
  subject neighbourhood or whether session 4's slice picks happened
  to fit the existing vocabulary — is worth answering, and the
  cleanest way is a slice from a deliberately distant domain
  (scientific paper, legal document, recipe corpus) that would
  force a genuinely new shape.

- **The taxonomy-tag UnaryClaim pattern has crossed the "worth
  naming" threshold.** Five distinct predicates now follow the
  `Is<X> UnaryClaim[Role]` shape where the predicate TYPE NAME
  carries the semantic content: IsPolyvalentTerm,
  CorrectsCommonMisconception, IsCognitiveBias, IsFictionalWork,
  IsFictional. Twenty claims total across the five types. The
  cognitive_biases.go inline comment that said "at four
  occurrences it is probably worth naming the pattern explicitly;
  at three, still treat it as coincidence" has been exceeded but
  is not yet promoted to a first-class discipline note in
  predicates.go — a cheap future cleanup.

- **Cross-file bridges only land when the source has real citation
  overlap with existing winze content.** The Clark→Rajaniemi bridge
  earned itself because the Fractal Prince Wikipedia article
  literally named Clark as an acknowledgment influence. The
  Shermer→SkepticsSociety bridge earned itself because the Michael
  Shermer Wikipedia article made biographical claims about an
  entity that already existed in winze. Every slice before session
  4 chose sources by content richness, not by citation overlap,
  and the entity graph stayed siloed. The correction: future
  ingest picks should factor "does this source cite entities
  already in winze" as a first-class criterion, alongside
  schema-forcing and content richness. Predicate density will
  continue to earn itself from any ingest — entity density has to
  be shopped for.

- **The temporal-prediction schema is still deferred.** Session 4
  surfaced the need for it (Clark + Forecasting pairing, user's
  explicit time-scale-range note) but no slice in the session
  forced its invention. The first ingest that reads a specific
  dated forecast with a credence value and a resolution outcome
  will earn Predicts + Credence + ResolvedAs. Until then the
  design constraint sits in forecasting.go's header comment as
  recorded intent, not live schema.

- **Lint state at session 4 close: 16 roles, 130 entities, 138
  referenced, 0 orphaned; 4 new predicates (IsFictionalWork,
  IsFictional, AppearsIn, Authored, InfluencedBy) bringing the
  total accretion to +5 for session 4; 3 functional predicates
  (unchanged); 3 KnownDispute annotations (unchanged); 0
  unresolved conflicts; 3 recorded disputes; 1 contested predicate;
  4 contested targets across 3 ingests (Nondualism,
  NondualAwareness, CognitiveBias, Apophenia); 3 cross-file entity
  references (Stope, AndyClark, MichaelShermer).** Growth since
  the end of the Tunguska session: +38 entities, 6 new source
  files, 5 new predicates, 2 new user-content cross-file bridges.

### Sixth public-corpus ingest: Quantum Thief trilogy (in-fiction schema)

Hannu Rajaniemi's Jean le Flambeur trilogy — Wikipedia articles for
all three novels plus the series concept. Chosen specifically for the
**in-world-fact vs real-world-fact** forcing function: the Wikipedia
articles assert two structurally different kinds of claim in the same
document (Rajaniemi really wrote the book vs Jean le Flambeur is a
thief), and winze's existing predicate vocabulary had no way to keep
them straight before this slice. Encoding them under the ordinary
predicates would have been a category error — a query for "cities"
should not return the Oubliette alongside Moscow.

The slice resolved the split with unary tags rather than a role
explosion, matching the established "predicate name is content"
discipline over role ceremony:

- **`IsFictionalWork UnaryClaim[Concept]`** — tags a concept as a
  creative work of fiction. The work itself exists in the real world;
  the tag is the real-world half of the split.
- **`IsFictional UnaryClaim[Concept]`** — tags a concept as existing
  only within the frame of some fictional work. Diegetic half of the
  split. The two tags are orthogonal: a concept can carry either,
  both (a fictional book inside a fictional book), or neither.
- **`AppearsIn BinaryRelation[Concept, Concept]`** — anchors an
  in-fiction entity back to its work. Explicitly **not** functional.
- **`Authored BinaryRelation[Person, Concept]`** — real-world
  authorship of a creative work. Distinct from `Proposes` because
  fiction is constructed, not asserted-true-of-the-world. Will split
  Authored / AuthoredOrg if collaborative authorship ever shows up.

The trilogy extension stresses `AppearsIn` non-functionality directly:
Jean le Flambeur, Mieli, and the Sobornost each have **three**
AppearsIn claims (one per novel), and Perhonen has two (the Wikipedia
article for TQT does not name Mieli's ship, so honesty demands two
claims rather than three). Seven subjects, fifteen AppearsIn
claims, zero value-conflict noise — proof by successful absence
that the predicate is correctly keyed as non-functional. Had it
been accidentally tagged `//winze:functional` the rule would have
exploded on all seven subjects.

Reuse wins in the trilogy extension:

- **`BelongsTo` reused for series membership.** Each novel BelongsTo
  the `JeanLeFlambeurSeries` concept — the same predicate introduced
  by the cognitive-biases ingest for bias→family membership. Second
  subject domain validating BelongsTo with zero schema change, and
  the "taxonomic membership" naming pattern holds up across
  completely unrelated subject matter (cognitive psychology vs
  science fiction).
- **No new roles.** The book, the series, characters, factions,
  cities, and invented memory substrates are all Concepts. A
  FictionalWork role split would have been premature: no query
  pattern forced it, and the Concept role is permissive enough
  to absorb them all without lying about what they are.

Discipline earned / validated:

- **Mirror source commitments.** Perhonen has only two AppearsIn
  claims because the Wikipedia article for The Quantum Thief does
  not name Mieli's ship, even though she clearly has one in the
  book. A richer ingest that reads the novels directly could add
  the third appearance honestly, but this slice refuses to
  fabricate beyond its source — same discipline the misconceptions
  slice established, applied here to a different shape of gap.
- **No new pragmas.** Unreliable-narrator handling (is the King of
  Mars actually Jean le Flambeur? who compromised the Oubliette's
  cryptography?) is deferred to a future slice that can layer
  contested in-fiction claims through existing Hypothesis + TheoryOf
  + `//winze:contested` machinery with zero new primitives. The
  schema is already expressive enough for the in-fiction contested
  case; only the specific hypotheses are missing.

What landed:

- **`quantum_thief.go`** — ~280 lines. 6 real-world entities (3 books,
  1 series, 1 author; all four creative-work concepts tagged
  IsFictionalWork) and 6 in-fiction Concept entities (Jean le Flambeur,
  Oubliette, Sobornost, Exomemory, Mieli, Perhonen; all tagged
  IsFictional). 28 claims total: 4 IsFictionalWork, 3 Authored, 3
  BelongsTo (book→series), 6 IsFictional, 12 AppearsIn (Jean×3,
  Mieli×3, Sobornost×3, Perhonen×2, Oubliette×1, Exomemory×1).
- **Four new predicates** in `predicates.go` — IsFictionalWork,
  IsFictional, AppearsIn, Authored. No new roles, no new pragmas,
  no new functional predicates.
- **defn v0.7.0 upgrade validated end-to-end on this ingest.** The
  bodies-misalignment and var-type-signature fixes from the
  2026-04-11 defn feedback cycle work correctly on the new file,
  and type-scoped queries (`WHERE signature LIKE 'var % Concept'`)
  return the full set of winze concepts in seconds.

Lint state after this slice: **16 roles, 103 entities, 111
referenced, 0 orphans; 4 new predicates (total predicate count +4);
3 functional predicates (unchanged); 3 KnownDispute annotations
(unchanged); 0 unresolved conflicts; 3 recorded disputes; 1
contested predicate; 3 contested targets across 2 ingests.** Zero
rule regressions on a 33% entity growth and a 4-predicate schema
expansion — the strongest single-slice validation yet that
accreting predicates strictly on ingest need does not destabilise
the existing ruleset.

After the founding session (see next section), a second working pass ran
the first public-corpus ingest: the Wikipedia article on the Tunguska
event. This ingest was chosen because it is unusually claim-dense *and*
unusually contradiction-rich — four rival cause hypotheses (stony
asteroid airburst, cometary airburst, glancing iron returning to orbit,
natural-gas release) and a live Lake Cheko crater dispute between the
2007 Bologna hypothesis and the 2017 Russian soil-varve rejection.

What landed:

- **`tunguska.go`** — ~380 lines, 27 entities, 27 claims. Six Hypothesis
  entities with proposer / disputant / explains-event relations. Real
  attribution for Whipple, Kresák, Sekanina, Chyba, Farinella, Longo,
  Kundt, Kulik. Institutional-level attribution for the 2007 Bologna,
  2017 Russian, and 2020 Russian teams where the source does not name
  individual authors.
- **`Hypothesis` role** added (wikidata Q41719) — first new role since
  the founding session. Role count 14 → 15, all still grounded.
- **Four new predicates** accreted strictly on ingest need: `ProposesOrg`,
  `DisputesOrg`, `FundedBy`, `AffiliatedWith`. The ProposesOrg /
  DisputesOrg split mirrors `MonitoredBy` vs `MonitoredByOrg` from the
  Reggie ingest — both predicates exist because Go has no sum type for
  "agent" and Tunguska's sources mix named-individual and
  institutional-collective proposers in the same debate.
- **defn graph** grew from 46/106/13 (types/vars/funcs) to 58/169/13.
  Build stays clean, naming-oracle stays 15/15 grounded, all
  previously-actionable orphans are wired.

### Concrete finding about contradiction shape

Writing the Tunguska cause hypotheses forced a pivot in how the next
deterministic lint rule should be shaped, which BOOTSTRAP.md's original
projection got wrong.

The four rival cause hypotheses for Tunguska are NOT disjoint-at-type.
They are written as Hypothesis *entities* with attribution relations,
because the authored facts ("Whipple proposed cometary in 1930",
"Sekanina proposed stony asteroid in 1983") are simultaneously true
for all four rivals and non-contradictory. A `//winze:disjoint` pragma
over predicate types — the originally-projected next lint rule — would
not catch the real disagreement here, because the disagreement lives
inside claim *values*, not between claim types.

The Lake Cheko crater dispute makes this even clearer: the contradiction
is semantic ("Lake Cheko is ~100 years old" vs "Lake Cheko is 280+ years
old"), not type-level. Both `BolognaProposesLakeChekoCrater` (2007) and
`Russian2017DisputesLakeCheko` coexist as valid claims about the state
of the debate.

**The revised next lint rule should be a value-conflict rule**: same
subject + same predicate + different objects at overlapping temporal
markers, flagged as a contradiction requiring explicit reconciliation
(a rename, a temporal split, or an AuthorialPolicy saying the conflict
is load-bearing). The predicate-disjointness pragma is still worth
having eventually for the rarer type-level cases (WorksFor vs
FormerlyWorkedFor at same era), but it is not the most valuable next
rule to build.

### Value-conflict lint rule (#3) landed

Immediately after the Tunguska ingest, the value-conflict rule was built
and immediately earned itself on its first run. The loop ran end-to-end
for the first time: ingest surfaced a contradiction shape, the schema
gained a predicate (`FormedAt`) and a pragma (`//winze:functional`) to
represent it, and the lint rule caught it.

- **Rule count: 3** — naming-oracle, orphan-report, value-conflict.
  Two deterministic structural rules plus one deterministic *semantic*
  rule, which is the first of its kind in winze.
- **Pragma machinery** — `//winze:functional` is the first real predicate
  pragma. A predicate type whose doc comment contains `//winze:functional`
  declares that at most one Object may exist per Subject; the lint rule
  then flags any such group with ≥2 distinct objects. The same
  `collectClaims` pass also primes a map of functional types, so
  `//winze:disjoint` and future pragmas will be a ~10-line addition to
  the same infrastructure rather than a new parse pass.
- **First-run calibration.** On the very first unconditional run, the
  rule flagged 8 (predicate, subject) groups — 1 real (Lake Cheko
  formation date) and 7 legitimate one-to-many relations (Reggie
  operates four instruments, Stope has three layers, etc). This
  immediately proved that value-conflict is *not* a universal rule; it
  has to key on a functional declaration. After adding the pragma, the
  rule settled at 1 real conflict, 0 false positives. This calibration
  step is worth remembering: **the first version of any semantic lint
  rule should be over-triggering, because the false positives are the
  most informative training signal about what the rule actually is.**
- **Lake Cheko is the first intentionally-load-bearing contradiction
  in winze** — and the mechanism for recording it now exists. A new
  `KnownDispute` type (in `disputes.go`) is a meta-annotation, not an
  ontological entity: it does NOT embed `*Entity`, so it lives outside
  the naming-oracle's role-type world entirely. A dispute is about the
  graph, not about the world the graph represents.
- **KnownDispute suppression closes the loop.** The value-conflict
  rule now keys each flagged group against a set of KnownDispute
  declarations (by `SubjectRef` identifier + `PredicateType` string) and
  bins conflicts into two buckets: *unresolved* (need action) and
  *recorded disputes* (load-bearing, shown for audit only). After
  adding `LakeChekoFormationDispute`, the first real lint run shows
  **0 unresolved, 1 recorded.** The pipeline's full contradiction-handling
  loop — surface → record → re-audit — ran cleanly on real ingested
  material for the first time.

### Fifth public-corpus ingest: cognitive biases (cross-ingest rule validation)

Small slice from the Wikipedia List of cognitive biases. Motivation
was cross-ingest validation: does the contested-concept lint rule +
`//winze:contested` pragma generalise zero-touch to a completely
different domain, or was it subtly Nondualism-specific?

The answer was: **zero-touch generalization confirmed**. No lint
binary changes, no pragma additions, no new rule wiring — the
machinery already built for Nondualism fired correctly on
CognitiveBias as soon as the two rival TheoryOf claims landed in
`cognitive_biases.go`. The rule now reports three contested targets
across two ingests: Nondualism (3 subjects), NondualAwareness (2
subjects), and CognitiveBias (2 subjects). First real validation
that predicate-type pragmas are a cross-ingest design point, not
just a Nondualism convenience.

- **`cognitive_biases.go`** — 6 Concept entities (CognitiveBias
  umbrella, EstimationBiases family, 4 specific biases:
  Availability heuristic, Anchoring, Dunning–Kruger, Hot-hand
  fallacy), 1 Person (Gigerenzer), 1 Organization (Dimara et al.
  2020), 2 Hypothesis entities (task-based classification and
  rational-deviation reframing). ~14 claims.
- **Two new predicates:**
  - `IsCognitiveBias UnaryClaim[Concept]` — third case of the
    "mark a concept as belonging to a well-defined taxonomy"
    unary pattern, after `IsPolyvalentTerm` (Nondualism) and
    `CorrectsCommonMisconception` (misconceptions). At four
    occurrences it is probably worth naming the pattern
    explicitly; at three, still treat it as coincidence.
  - `BelongsTo BinaryRelation[Concept, Concept]` — taxonomic
    membership, distinct from DerivedFrom (etymology). Not
    functional (overlapping memberships are legal). Cycle
    detection is a candidate for a future lint rule but not
    forced by this ingest.
- **No HypothesisAbout widening.** Both the individual biases and
  the family concepts and the umbrella meta-concept are all
  Concept-typed, so existing TheoryOf handles the
  hypothesis→concept linking cleanly. The Place/Person/Facility-
  shaped misconceptions remain the forcing case for that
  widening, and still waiting for a real ingest to demand it.
- **Contested-concept rule now has cross-ingest evidence.** The
  pragma machinery was designed on Nondualism and validated on
  CognitiveBias with zero touches to the lint binary. Add to the
  non-executable-is-load-bearing tax-savings tally: the
  pragma-driven rule design is the kind of abstraction that pays
  off when applied to a corpus it wasn't designed against,
  exactly as promised. A human-targeted KB would more naturally
  hand-curate per-domain rules; winze's machine-readable discipline
  lets a single pragma definition cover every ingest that ever
  uses the predicate.
- **Meta-level contestation is a recurring pattern.** This is now
  the third ingest where the source article explicitly flags
  meta-level disagreement about how to partition or frame the
  domain (Nondualism's typologies, common-core vs constructionist
  debate, Gigerenzer vs classical-framing). Worth noting as a
  recurring shape: academically-written sources frequently
  *declare* their meta-level controversies in the first few
  paragraphs, and extracting those is extremely high-value for
  winze because they are pre-packaged contested-concept material
  the contested-concept rule can surface immediately.
- Lint state after this slice: **16 roles, 92 entities, 100
  referenced, 0 orphans; 3 functional predicates; 3 KnownDispute
  annotations; 0 unresolved conflicts; 3 recorded disputes; 1
  contested predicate; 3 contested targets across 2 ingests.**

### Fourth public-corpus ingest: common misconceptions (source-discipline test)

A small parallel-work slice from the Wikipedia List of common
misconceptions about science, technology, and mathematics. Motivation
was a generality check: does the value-conflict machinery built for
Tunguska / Nondualism extend to an entirely new claim shape —
"popular belief vs scientific consensus" — without forcing new
structural work?

The answer was: **no, and that is itself the finding**. The source
article explicitly states "each entry is worded as a correction; the
misconceptions themselves are implied rather than stated." A
value-conflict representation — "people believe X but actually ¬X" —
would require winze to fabricate a structured record of X that the
source *deliberately refuses to provide*. Mirroring the source's
discipline is the correct move, and it forces a different encoding:
one Hypothesis per corrected fact, tagged with a new unary
predicate.

- **`misconceptions.go`** — 3 Concept entities (EarthSeasons,
  AtmosphericEntryHeating, ArtificialStructureSpaceVisibility),
  3 corrected-fact Hypothesis entities whose Names *are* the
  corrections, 6 claims (3 TheoryOf + 3 CorrectsCommonMisconception
  tags). Deliberately scoped to Concept-shaped subjects so existing
  TheoryOf handles the linking; Place/Person/Facility-shaped
  misconceptions (Sun colour, Napoleon's height, Great Wall as a
  Place) are deferred pending a role-split decision for the
  HypothesisAbout predicate family.
- **`CorrectsCommonMisconception UnaryClaim[Hypothesis]`** — new
  predicate type. First UnaryClaim in winze whose Subject is a
  Hypothesis rather than a Person (style claims in user.go) or a
  Concept (NondualismIsPolyvalent). The pattern generalizes:
  UnaryClaim is now used as a meta-annotation on reified
  intellectual-position entities, not just on role-typed
  real-world entities. Name-is-content discipline carries over
  cleanly — no value field, no structured misbelief content.
- **Discipline earned: mirror-source-commitments.** When a source
  refuses to structure a claim, winze should refuse too, even
  when the existing schema makes the richer structure cheap. The
  temptation in this ingest was to encode both the misbelief and
  the correction as rival Hypothesis entities with a
  `Corrects[Hypothesis, Hypothesis]` relation and a
  CommonlyBelieved tag on the false one. That would have been
  more "winze-native" in a schema sense, but it would have
  fabricated content — specifically, a structured representation
  of what people *actually* believe, which Wikipedia's editors
  chose not to record. The slice explicitly chose the thinner
  encoding and documented the fact that a future ingest wanting
  structured misbelief content will need its own forcing case.
  **Add to the non-executable-is-load-bearing tax-savings tally**:
  refusing to invent source-internal structure is a discipline
  winze can afford because its readers are agents that can follow
  the Provenance.Quote trail, not humans scanning a Brief.
- **First UnaryClaim on a Hypothesis subject.** Previously
  UnaryClaim was only used on Person (style claims) and Concept
  (polyvalence). Extending to Hypothesis proves the
  predicate-type-name-is-content pattern works at every level of
  the reification stack, not just at the role-entity leaf.
- Lint state after this slice: **16 roles, 82 entities, 90
  referenced, 0 orphans; 3 functional predicates; 3 KnownDispute
  annotations; 0 unresolved conflicts; 3 recorded disputes; 1
  contested predicate; 2 contested targets.** Same shape as after
  the Müller slice plus a clean 6-entity / 6-claim addition;
  no rule regressions.

### Müller translation dispute slice (third functional predicate)

Followed the Nondualism ingest's deferred Müller-advaita-as-Monism
thread to completion as a self-contained filler slice. The source
article is explicit that three rival English renderings of 'advaita'
coexist in print: Müller (1879) 'Monism', standard-usage 'nondualism
/ not-two', and the Hacker/Volker 'that which has no second beside
it'. All three are now recorded as `EnglishTranslationOf` claims
under a `KnownDispute`.

- **`EnglishTranslationOf`** — third `//winze:functional` predicate
  after FormedAt and EnergyEstimate. First functional predicate with
  a Concept subject (previous two were Place and Event). Object slot
  is a new non-entity struct `*EnglishRendering{Value, By}`,
  mirroring TemporalMarker and EnergyReading — the value-with-
  attribution pattern is now used three times and is probably a
  shape worth naming explicitly if a fourth case shows up.
- **Dual-layer dispute representation.** winze now records the
  Müller disagreement at two complementary levels: the Hypothesis /
  attribution layer (Müller Proposes AdvaitaAsMonismTranslation,
  Watts Disputes same — from the earlier Nondualism slice) and the
  value / functional-predicate layer (three rival
  EnglishTranslationOf claims annotated by AdvaitaTranslationDispute
  — this slice). The two layers are NOT redundant: the Hypothesis
  layer carries attribution and disputant structure, the functional
  layer carries the rival values themselves, and each answers
  questions the other cannot. Recording both is the correct move
  when the same real-world disagreement has both a social-
  attribution aspect and a value-conflict aspect. Add to the non-
  executable-is-load-bearing tax-savings tally: a human-oriented
  KB would push the user to pick one representation; winze can
  keep both because the cost is paid by the machine-query layer,
  not by a human scanning a Brief.
- Lint state after this slice: **16 roles, 76 entities, 84
  referenced, 0 orphans; 3 functional predicates; 3 KnownDispute
  annotations; 0 unresolved conflicts; 3 recorded disputes; 1
  contested predicate; 2 contested targets.** First run where
  value-conflict has more than one dispute source (Lake Cheko,
  Tunguska energy, Advaita rendering), which is also the first
  real stress on the per-group sort/print paths — the output is
  alphabetical by predicate name and readable without further
  grouping work.

### Contested-concept lint rule (#4) landed

Immediately after the Nondualism ingest surfaced "multiple theories of
the same concept" as a real structural pattern in the graph, a fourth
lint rule was added to catch it automatically:

- **`//winze:contested` pragma** — second predicate-type pragma after
  `//winze:functional`, applied to `TheoryOf`. Signals that the
  predicate's *object* is the axis on which disagreement lives, not
  the subject. Symmetric-mirror of functional in mechanism but
  opposite in polarity: functional groups by (predicate, subject)
  and flags value divergence; contested groups by (predicate,
  object) and surfaces subject plurality.
- **`contestedConceptRule`** — ~60 lines on the existing
  `collectClaims` scaffolding. Returns 0 always; never fails the
  build. Its job is to surface the landscape of disagreement, not
  to adjudicate. This is the first *advisory-only* lint rule in
  winze — an explicit information-surface rule rather than a
  correctness rule.
- **First run found exactly what the ad-hoc defn reference-graph
  query found during the reflection pass**: `TheoryOf (Nondualism)`
  with 3 distinct subjects (Murti/Loy/Volker typologies), `TheoryOf
  (NondualAwareness)` with 2 distinct subjects (perennialist vs
  constructionist theses). The rule closed the gap between "visible
  via a one-off SQL query against defn" and "surfaced automatically
  at every lint run" — a meaningful workflow improvement, since ad-
  hoc queries are fine for reflection but bad as the primary
  surface for live structural patterns.
- **Rule count: 4** — naming-oracle (deterministic structural),
  orphan-report (deterministic structural), value-conflict
  (deterministic semantic, pragma-gated, failure-surfacing),
  contested-concept (deterministic semantic, pragma-gated,
  advisory-surfacing). Two pragmas now drive the semantic rule
  layer; the machinery generalizes to a third with another ~10
  lines if a future ingest demands it.
- **The broader lesson** — a pattern being visible in a reference
  graph is not the same as being surfaced by the pipeline. Queries
  are fine for reflection; rules are what make patterns live in the
  ingest loop. When an ad-hoc query turns out to be answering a
  question ingest workers will want answered every run, the right
  move is to promote it to a rule.

### Third public-corpus ingest: Wikipedia Nondualism (schema-forcing test)

After the dogfood slice, the Nondualism article was ingested as the
hardest-available schema test on the roadmap. The article opens by
declaring nondualism a "polyvalent term" — the contested-definition
shape is *announced by the source*, not inferred. The test was: can
winze represent "the same word means incompatible things to different
authors, and some of the authors disagree about the structure of that
disagreement itself" without a new sense-disambiguation primitive?

The answer landed as **yes, via reification of each author's typology
as a Hypothesis entity**, one level up from the Tunguska move. No new
primitive for senses, no tradition-scoped term annotation, no
per-Entity alternate-definition field.

- **`nondualism.go`** — 6 Concept entities (Nondualism, Advaita,
  Advaya, Śūnyatā only in Brief form, NondualAwareness, the rest), 6
  Person entities (Murti, Loy, Volker, Müller, Watts, Katz), 6
  Hypothesis entities (three typologies — Murti's advaita/advaya
  binary, Loy's five flavors, Volker's three types — plus
  perennialism, constructionism, and the Müller advaita-as-Monism
  translation hypothesis). ~15 claims wiring it all together.
- **`Concept` role** added (roles.go, grounded to wikidata Q151885 in
  external.go). First role accretion since Hypothesis (Tunguska). The
  role is deliberately permissive: anything from a Sanskrit root to a
  contemporary philosophical position lives here. The disambiguation
  work is done by the claims attached, not by the role itself.
- **Three new predicates, all earned on ingest need:**
  `IsPolyvalentTerm UnaryClaim[Concept]` (corpus-level meta-claim
  that lets winze mark a concept's contested-meaning status as a
  first-class fact rather than leaving it in prose), `TheoryOf
  BinaryRelation[Hypothesis, Concept]` (parallel to
  HypothesisExplains's Event slot — splits for the same
  no-sum-types reason), `DerivedFrom BinaryRelation[Concept, Concept]`
  (etymological lineage, deliberately not functional because
  Nondualism descends from both Advaita and Advaya).
- **Reification-over-schema-extension validated at a higher level
  than before.** The Tunguska ingest already showed reification
  handling competing causal explanations (each cause is a Hypothesis).
  Nondualism pushes this one level up: competing *typologies of a
  concept* — schema-level disagreements — are also just Hypothesis
  entities, attached with TheoryOf. Murti, Loy, and Volker proposing
  incompatible partitions of nondualism is structurally the same as
  Whipple, Sekanina, and Kundt proposing incompatible causes of
  Tunguska. This is a substantial tax saving: a KB that served human
  readers would almost certainly need sense-disambiguation or
  tradition-scoped-term primitives here, because a human reader
  scanning Briefs would mis-resolve "advaita" to Müller's Monism
  translation. Winze gets to skip that entire design space because
  machine readers traverse relations rather than skim definitions.
  Added to the non-executable-is-load-bearing tax-savings tally.
- **Orphan-report earned itself again on ingest discipline.** The
  first pass created stub Concept entities for Brahman, Śūnyatā, and
  Prapañca and a Person entity for Paul Hacker, each mentioned in
  surrounding Briefs but with no honest claim wiring in this slice.
  The orphan rule flagged all four, forcing a decision: delete the
  stubs or wire them dishonestly. Deleted the stubs — a subsequent
  Advaita Vedanta or Madhyamaka slice (for Brahman / Śūnyatā /
  Prapañca) and a `EtymologicalReadingOf[Person, Concept]`
  predicate slice (for Hacker) will reintroduce them with real
  claims. **General discipline surfaced: an entity whose only
  plausible wiring would misrepresent the source should remain
  undeclared rather than stubbed. Brief-level references are fine
  — the orphan rule correctly punishes fake-claim ceremony.**
- **Deliberately deferred: the Müller-advaita-as-Monism translation
  is a second-order value-conflict-shape dispute (Müller asserts one
  translation, other scholars reject it, Watts proposes a distinct
  reading — three parties, a functional-predicate shape). Left for a
  follow-up slice so this ingest stays scoped to the typology layer.
  Strong candidate for a third `//winze:functional` predicate
  (`EnglishTranslationOf BinaryRelation[Concept, Concept]`) and a
  third KnownDispute.
- Lint state after this slice: **16 roles, 76 entities, 81
  referenced, 0 orphans; 2 functional predicates; 0 unresolved
  conflicts; 2 recorded disputes.** Unchanged dispute count because
  the translation dispute was scoped out.

### Second functional-predicate slice: Tunguska energy estimate

Immediately after the dogfood slice, a second functional predicate was
added to stress the value-conflict rule with a three-way (rather than
two-way) recorded dispute.

- **`EnergyEstimate`** — second `//winze:functional` predicate. Object
  slot is a new value-with-attribution struct `*EnergyReading{Value,
  By}`, mirroring `*TemporalMarker`'s pattern: a non-entity struct
  that lives outside the naming-oracle's role-type world. Kept in
  `predicates.go` next to `FormedAt` for discoverability.
- **Three rival readings for the Tunguska airburst energy** — `~20-30
  Mt` (Kulik-era forest-damage estimates), `~10-15 Mt` (Ben-Menahem
  1975 seismic waveform), `~3-5 Mt` (Boslough & Crawford 2007 3D
  airburst simulation). All three remain in print as best-fit numbers
  from different methods; later does not displace earlier in
  citation practice. The source's own "3-50 Mt" bracket is the
  superset, but winze records the three clusters, not the bracket.
- **`TunguskaEnergyDispute`** KnownDispute annotates the group as
  load-bearing.
- **First real three-way dispute in winze.** The value-conflict rule
  now prints two recorded-dispute groups — two-way Lake Cheko and
  three-way Tunguska energy — with no changes needed to the rule
  itself. Validates that the dispute partitioning, sorting, and
  printing all handle arbitrary group sizes cleanly.
- Lint state after this slice: 60 entities / 65 referenced / 0
  orphans, 2 functional predicates, 2 KnownDispute annotations, 0
  unresolved conflicts, 2 recorded disputes.

### Memory dogfood slice (user.go)

The user pointed out that storing session-level working-style knowledge
in an external markdown `memory/` directory is exactly the
load-bearing-prose dodge winze is supposed to replace, and suggested
dogfooding by moving the memory into winze proper.

- **`user.go`** encodes the user as a `Person` entity plus four distinct
  UnaryClaim predicate types — `GrantsBroadAuthorityOverWinze`,
  `PrefersTerseResponses`, `PushesBackOnOverengineering`,
  `PrefersOrganicSchemaGrowth`. Each is a first-class predicate type
  whose *name* is the content, so no string-valued field is needed and
  the dogfood slice follows the same "predicate as type" discipline as
  the ingest predicates.
- **The external memory/ directory now holds only three short
  pointer files** (`user_style.md`, `winze_thesis.md`,
  `defn_adit_ownership.md`, plus `MEMORY.md` as the index). They exist
  as a fallback for sessions that start before winze is read. The
  canonical copies live inside winze.
- **The dogfood test immediately surfaced a real bug in the
  orphan-report rule.** The first user.go run flagged `UserJustin` as
  orphaned, because orphan-report had been collecting references only
  from BinaryRelation claims (Subject+Object composite literals) and
  was blind to UnaryClaim claims (Subject-only literals, which is what
  every style claim is). Fixed by adding a `collectUnarySubjects` pass
  that harvests subject references from unary-shaped literals and
  merges them into the referenced set. That bug had been latent and
  would have surfaced at the first real UnaryClaim ingest; dogfooding
  caught it early. **Rule of thumb: any new shape of claim added to
  winze should be dogfooded against winze itself before being used on
  ingest content, so lint blind spots surface on load-bearing
  self-reference first.**

### Papercuts logged this session

- **`defn query` CLI vs MCP lock collision.** (RESOLVED in this session.)
  The `orphan-report` lint rule originally shelled out to `defn query`,
  which opened the Dolt DB directly and collided with the always-running
  MCP defn server holding a write lock. Rather than fix the collision,
  orphan-report was rewritten as a pure go/ast pass (walk role-typed
  entity vars, subtract those referenced by any Subject or Object field
  in a claim composite literal). The lint binary now has **zero
  dependency on defn** and is fully self-contained. Useful lesson: when a
  tool collision blocks you, check whether the downstream consumer
  actually needed the tool's unique capability — for orphan detection,
  the defn reference graph was not load-bearing; the same analysis falls
  out of the claim parser the lint binary already had.

- **`defn status` reports false staleness after MCP auto-sync.** The
  MCP server auto-syncs on query (verified by probe: a fresh type
  addition showed up in a query without any explicit sync call). But
  `defn status` still reports "database may be stale: X.go is newer
  than DB" because it checks file mtimes against a DB timestamp that
  MCP-side sync does not update. Cosmetic; confusing for agents
  deciding whether to call sync. Both manual `defn sync` CLI calls and
  MCP `op: sync` ops are redundant in an MCP-server workflow.

## Status — 2026-04-11 (end of founding session)

The founding session got through more of the checkpoint list than this file's
original checkpoint prose anticipates. Concretely:

- **Schema v0.2 exists.** `schema.go` has `Provenance`, `TemporalMarker`,
  generic `BinaryRelation[S, O]` / `UnaryClaim[S]` bases, `Scene`. Predicates
  are named distinct types over `BinaryRelation[...]` instantiations — each
  predicate is a first-class type in defn's graph.
- **Role-typed slots enforce ontology sanity at compile time.** `roles.go`
  introduces A-shape roles `Person`, `Organization`, `Place`, `Event`,
  `Facility`, `Substance`, `Instrument`. `design_roles.go` introduces
  B-shape roles `CreativeWork`, `DesignLayer`, `Phase`, `ProtectedLine`,
  `NeverAnswered`, `AuthorialPolicy`, `Reading`. The compiler rejects
  `WorksFor{Subject: ChurchRockSpill, ...}` because an `Event` cannot fill
  a `Person` slot — the Mathlib-flavor predicate-as-type argument has
  real teeth.
- **Provenance is audit-only, no live source link.** Sources are transient
  in the intended workflow — winze is the canonical representation, not a
  mirror of external files. `Provenance = {Origin, IngestedAt, IngestedBy,
  Quote}`. When the source is gone, the extracted `Quote` fragment is the
  audit trail.
- **Two real ingests are encoded.** `reggie.go` hand-encodes claims from
  the Stope corpus's `npc-reggie-tsosie.md` (18 claims across people,
  places, instruments, events). `stope_constraints.go` hand-encodes the
  B-shape structure of `stope-constraints.md` (layers, phases, protected
  lines with layered readings, never-answered questions, authorial
  policies). Both compile and cross-link: `StopeGame = CreativeWork{Stope}`
  joins `stope_constraints.go` back to `bootstrap.go`'s `Stope` entity.
- **External naming oracle exists and is consulted.** `external.go` contains
  `ExternalTerms`, a flat lookup table pulling names from Schema.org,
  WordNet, Wikidata, plus a `winze-native` source for legitimately invented
  vocabulary.
- **Two deterministic lint rules run.** `cmd/lint/main.go` is a separate
  `package main` that imports winze as a library:
    1. **naming-oracle** — parses `.go` files via `go/ast`, finds every
       type whose underlying struct embeds `*Entity`, checks the name
       against `ExternalTerms`, exits non-zero on ungrounded names.
       All 14 current role types pass.
    2. **orphan-report** — shells out to `defn query`, reports definitions
       with zero inbound references. Advisory (does not fail the build)
       because most leaf claims are legitimately orphan-shaped. First
       real run caught 5 entity-orphans — created during ingest but never
       claimed about — which is a concrete win the pipeline earned on
       its first meaningful invocation.
- **Decisions updated.** `dec-prose-is-io` now explicitly says sources are
  transient; `mit-pending` (pending/ subpackage) is marked obsolete in
  favor of `mit-naming-oracle` against a flat root layout — the argument
  was that defn's reference graph already tracks usage count, and a
  physical package split delivers no query or update value defn doesn't
  already provide. `dec-no-upper-ontology` was added: do NOT import
  Cyc/DOLCE/SUMO/Schema.org as a dependency, but DO use them as a
  disambiguation oracle during ingest.

What is NOT done, and is the obvious next session surface:

- **No disjointness pragma infrastructure** — the motivating deterministic
  contradiction-detection lint rule (`mit-predicate-disjoint`). Cheap to
  implement once there is a real `//winze:disjoint` example. See the
  Tunguska status section above for why this is now believed to be the
  *second* most valuable next rule, behind a value-conflict rule.
- **No `defn overview --all` or similar project self-description.** This
  file is still load-bearing; the ambition is to delete it.
- **No Gas Town skill package** (`oq-gastown-awareness` still blocking).
- **No LLM-backed lint rule** (`oq-contradiction-rule` still blocking).
- **No v1 benchmark design** (`oq-benchmark` still blocking).
- **No public demo corpus chosen.** The Stope reference docs are the user's
  private notes for a non-public project. They are fine as seed, but winze
  needs a public, claim-dense, contradiction-rich ingest target before it
  can be demoed to anyone. Wikipedia Tunguska has landed as the first
  public slice (see 2026-04-11 Tunguska status section).

- **Open schema question: `TernaryRelation` (and beyond).** Raised by
  the user 2026-04-11, with a sharpened framing on the follow-up: n-ary
  relations are an "inevitable grind" for a sufficiently complex KB —
  not a hypothetical case but an expected load-bearing future primitive.
  The one genuine 3-ary case so far (`LineHasReadingAtLayer` in Stope
  constraints) is handled by reifying the middle slot (Reading) into
  an entity that carries the Layer attachment. Reification is the
  semantic-web mainstream answer and it works, but waiting indefinitely
  just accumulates reification debt. **Revised policy:** add
  `TernaryRelation[S, O1, O2]` as a first-class primitive at the *first*
  ingest case where reification genuinely resists (an irreducible
  3-ary like "X is between Y and Z", not a decomposable "X did Y to Z"
  with a natural event-middle). Do NOT add it prospectively — still
  wait for the forcing function — but do not require a backlog of
  cases either. Higher-arity primitives (`QuaternaryRelation`, ...)
  remain prospectively forbidden — and will probably stay forbidden
  permanently. The observation that forced this: past 3 slots, natural
  language stops parsing the relation as a single predicate and starts
  parsing it as a reified event or scene with named attributes ("a
  chemical reaction has a catalyst, a solvent, a temperature" is how
  humans think of it, not `Reacts[reactant, catalyst, product, solvent,
  temperature]`). So the arity cap at ternary isn't arbitrary — it
  tracks where the ergonomic failure mode flips from "you're dodging by
  reifying" to "you're being helpful by reifying." Above 3, reification
  is the right answer, not a stopgap.

## Ingest roadmap

Queued for next ingest passes. Each is on the roadmap for two reasons at
once: as a test ingest that stresses a new corpus shape (extending beyond
encyclopedic prose into scientific-paper structure), AND as live material
for open question `oq-gastown-awareness` — the papers below are meant to
inspire what a Gas Town worker fleet running *inside* winze could actually
be doing as a cognitive architecture, not only as an orchestrator.

- **Frontiers in Neuroscience 2014 —** `https://www.frontiersin.org/journals/neuroscience/articles/10.3389/fnins.2014.00265/full`
  Scientific paper. New ingest shape: structured abstract / introduction
  / methods / results / discussion, with cited references forming a
  cross-paper graph that winze does not yet model. Will force decisions
  about `Citation`, `Claim-in-paper`, `Figure`, and `Author` roles.

- **PubMed 23663408 —** `https://pubmed.ncbi.nlm.nih.gov/23663408/`
  Second paper for the same pass; at minimum provides a second node in
  the citation graph so cross-paper reference resolution is tested on
  more than a single source.

- **Wikipedia List of cognitive biases —** `https://en.wikipedia.org/wiki/List_of_cognitive_biases`
  Ingest via local ZIM MCP (not online). Large enumerated-taxonomy
  shape: each bias is a named concept with a short definition and
  often a canonical example. New corpus shape for winze — stresses
  how a taxonomy of ~200 named concepts is represented without
  collapsing each entry into a markdown blob. Also: every bias is a
  candidate *LLM-lint-rule pattern*, so this ingest doubles as a
  backlog for `oq-contradiction-rule`-adjacent detectors.

- **Wikipedia Nondualism —** `https://en.wikipedia.org/wiki/Nondualism`
  Ingest via local ZIM MCP (including subpages as appropriate).
  Contested-definition philosophical prose — Advaita, Mahāyāna,
  Kashmiri Shaiva, Western mysticism all use overlapping terminology
  with incompatible technical meanings. Stresses coreference and
  alias resolution, and forces an honest decision about whether
  winze can represent "a term means X in tradition A and ¬X in
  tradition B" cleanly without collapsing to a single canonical
  sense.

- **Wikipedia List of common misconceptions —** `https://en.wikipedia.org/wiki/List_of_common_misconceptions`
  Ingest via local ZIM MCP (including subpages as appropriate).
  Claim-dense list of popular-belief vs actual-fact pairs — a
  natural substrate for the value-conflict and contradiction lint
  rules. Each entry is almost already shaped as a
  `PopularlyBelieved(X)` vs `ActuallyTrue(¬X)` pair; ingesting this
  should either earn its keep on the existing value-conflict rule
  or surface what second-pragma (`//winze:disjoint` or similar) the
  rule needs to gain to handle it. **First slice landed** — see
  the common-misconceptions status section above. Further slices
  pending a role-split decision for the HypothesisAbout family to
  handle Place/Person/Facility-shaped subjects.

- **Wikipedia The Quantum Thief —** `https://en.wikipedia.org/wiki/The_Quantum_Thief`
  **First slice + trilogy extension landed** — see the Quantum Thief
  status section above. The in-world-fact vs real-world-fact split
  earned four new predicates (IsFictionalWork, IsFictional,
  AppearsIn, Authored) and the trilogy extension stress-tested
  AppearsIn non-functionality with seven multi-object subjects.
  Future slices can add unreliable-narrator handling (contested
  in-fiction claims via existing TheoryOf + `//winze:contested`
  machinery with zero new primitives), further invented vocabulary
  (gevulot, Quiet, cryptarchs, gogols, Zoku), and additional
  characters (Mieli has no sibling in the slice yet — Isidore
  Beautrelet, Tawaddud Gomelez, Joséphine Pellegrini are all
  unwired). Gas Town architectural dual-purpose preserved: the
  typed concepts now give the reasoning-agent layer something to
  point at instead of free text.

- **Wikipedia The Demon-Haunted World —** `https://en.wikipedia.org/wiki/The_Demon-Haunted_World`
  Carl Sagan's 1995 book on pseudoscience and the 'baloney
  detection kit'. Dual purpose: (1) natural partner to the
  existing CorrectsCommonMisconception tag — the book catalogues
  debunkings and names specific rhetorical fallacies, which
  likely forces a rhetorical-fallacy or epistemic-hygiene
  predicate family winze does not yet have; (2) architectural
  inspiration for winze's anti-Cyc discipline — Sagan's instinct
  that most claims deserve provisional suspicion maps directly
  onto winze's "only encode what the source commits to" rule and
  onto the orphan-report's refusal to let ceremonial stubs
  linger. A natural candidate to ingest after a couple more
  predicate-accretion slices have accumulated enough vocabulary
  to avoid gratuitous schema work on first contact.

- **Wikipedia Forecasting —** `https://en.wikipedia.org/wiki/Forecasting`
  Structurally novel because every Hypothesis winze currently
  encodes is about past or timeless facts; a forecast has a
  future resolution date and an accuracy dimension. Likely forces
  a `Predicts[Hypothesis, Event]`-shaped predicate where the
  Event is in the future, plus a resolved-vs-unresolved state and
  possibly a Brier-score-style accuracy annotation. Rival
  forecasts of the same outcome map cleanly onto the existing
  value-conflict machinery (the same shape as rival energy
  estimates or rival English renderings, with time as the new
  axis). Paired with PubMed 23663408 per the user's note —
  "very related although technically on slightly different time
  scales" — so the two should be ingested in the same pass to
  surface whatever shared temporal-prediction shape they want.

## Uncitable references (not ingest targets)

Resources consulted for architectural inspiration but explicitly
*not* ingested into winze. Each is too large, too dense, or too
uncitable to encode as typed claims under the current schema, and
Provenance.Quote discipline is a poor fit for whole-book prose.
These are for the assistant to consult directly when thinking
about a design question — no Provenance lineage into winze
should originate from them.

- **John C. Wright, _The Golden Age_ —** `/home/gas6amus/Downloads/goldenage.html`
  ~840 KB HTML dump of the full novel. Rich fictional model of a
  post-singularity society of interacting minds: Sophotechs as
  paternalistic AI, the Hortators as social enforcement, sestiad
  clone lineages, distributed consensus protocols,
  memory-editing privacy concerns. Consult during
  `oq-gastown-awareness` design work or when thinking about how
  a machine-mind society might model agent-to-agent,
  agent-to-self, or agent-to-substrate relationships. Pair with
  the Quantum Thief ingest: Quantum Thief for citable Wikipedia-
  level encoding, Golden Age for unencodable but useful
  architectural ideas.

- **Mark Fiddler, humunivers.htm —** `https://condor.depaul.edu/~mfiddler/hyphen/humunivers.htm`
  DePaul course page on human universals, paired with goldenage.html
  as inspiration-only reading for the Gas Town reasoning-agent
  layer. Not an ingest target — course material is not canonical
  citable content, and no Provenance lineage from this URL should
  ever land in winze. Consult via Read/WebFetch when thinking
  about `oq-gastown-awareness` and what kinds of claims a
  multi-agent society would actually need to ground. If a specific
  idea turns out useful, refine it through conversation with the
  user and cite the user's decision, not the page.

Both papers are queued specifically because the user wants their content
to inform the design of winze-internal Gas Town worker roles — not only
to be ingested, but to be *used* as architectural inspiration. This
makes the first ingest worth doing slowly and curatorially: the encoding
itself is a design exercise for the cognitive loop winze is trying to
support.

## What winze is

A personal knowledge base where the storage substrate is a compilable-but-never-
executed Go codebase. LLM worker agents (running under Gas Town) author it; no
human reads or writes the code directly; `go build` is the consistency checker
for references and types; a separate lint layer (some rules deterministic, some
LLM-backed) handles the checks the compiler can't.

The thesis, in one sentence: **a personal knowledge base is just a codebase the
agent never compiles for execution, and every hard problem in PKM is a solved
problem in SWE tooling that nobody thought to apply because they assumed
knowledge was for humans.**

## Why Go, specifically

Because `defn` parses Go with `go/types` and gives us a battle-tested type
checker, a reference graph, rename safety, impact analysis, SQL queries over
the graph, and Dolt-level branch/merge on structured data — for free. None of
this has to be built. The trade is: Go's type system is weak (structural
interfaces, no sum types, no dependent types), which is paradoxically the right
level of strictness for a general-knowledge KB. Strong enough to catch
reference drift, weak enough that LLMs can churn the schema without fighting
the compiler.

## The tool stack

- **winze** — the codebase (this directory)
- **defn** — MCP ops (impact, rename, simulate, query, branch/merge) against
  the Dolt-backed reference graph. `/home/justin/Documents/defn`.
- **adit** — cost oracle (file size, grep noise, blast radius, ambiguous
  names). Validated against SWE-bench trajectories. Runs as a Gas Town quality
  gate. `/home/justin/Documents/adit`.
- **Gas Town** — Steve Yegge's orchestrator for fleets of Claude Code
  instances, built on Dolt. Worker roles (Mayor, Polecats, Convoys), merge
  queue, quality gates. Winze runs as a Gas Town project, not a fork. See
  `/home/gas6amus/Documents/psychosis/gastown.html` and `/tmp/wasteland.html`
  for Yegge's posts.
- **Dolt** — SQL + git semantics. Storage substrate for defn's graph and for
  Gas Town's state.

## Locked-in decisions (see bootstrap.go for full rationale)

- Non-executable is load-bearing. Executability is icing. *What this
  decision is actually worth, not just what it rules out:* every place
  winze would otherwise owe a cost to human ergonomics, it instead gets
  to spend that budget on machine-queryability and churn tolerance. The
  arity cap at ternary (see `TernaryRelation` open question) is a
  concrete example — above 3 slots, humans stop parsing a relation as a
  single predicate and start parsing it as a reified event; an
  execution-optimised or human-readable KB would have to either accept
  the unreadability or work around the cap; winze just reifies above 3
  and moves on, because there is no human in the read path to slow
  down. Add more examples here as they surface — this section is a
  running tally of what the non-executable tax *saves*, not a proof it
  is worth paying.
- Code editing and knowledge manipulation are the same operation.
- Humans never read or write winze directly.
- Gas Town is the orchestrator. Do not fork it; write a skill package.
- LLM judgment is one lint rule among many, not a separate stage.
- Prose is input/output, not state. **No //go:embed sidecars.** Ingest workers
  read prose sources and emit typed claims. Query workers render typed claims
  back to prose when asked. The authoritative state is the typed graph.
- adit is the authoring-cost oracle.

## The three failure modes

1. **Ontology churn (severity 2)** — LLM re-types the world every session.
   Manageable with Mathlib's rename-with-deprecation discipline and a churn KPI.
2. **Queryability mirage (severity 2)** — agents default to grep even when
   structured queries exist. Engineerable via forced-briefing and
   grep-hostile substrate.
3. **Consistency is not correctness (severity 3, most dangerous)** — `go build`
   passes on contradictory claims. Requires a separate-prompt separate-context
   LLM judge as a lint rule, running post-commit. Empirically critical because
   SOTA LLMs accept adversarial contradictions ~80% of the time.

## What the first real session should do

In order, stopping to decide at each checkpoint:

1. **Read bootstrap.go and this file.** Run `go build ./...` to confirm it
   compiles in the current environment. Run `go vet ./...`.
2. **Take a first pass at the claim schema** (open question `oq-claim-schema`).
   This is the hardest unsolved piece. What is a `Claim`? A `Scene`? A
   `Relationship`? A `TemporalMarker`? How does a sentence from a Stope essay
   map to graph nodes? How are aliases and coreferences resolved at ingest
   time? Expect the schema to churn — use rename-with-deprecation from day one.
   Do not try to solve this perfectly; produce a v0 that can encode ~10 sample
   claims from a Stope essay and let the failure modes surface.
3. **Sketch the Rule interface and pipeline** (open question `oq-rule-registry`).
   Cheap / medium / expensive tiers. Deterministic vs LLM-backed. Budget
   enforcement. Ship ~5 cheap rule stubs first (cycle detection, blast-radius
   threshold, orphan detection, disjointness-pragma check, hash-stale check
   for provenance source files). LLM rules come later.
4. **Draft the winze Gas Town skill package** (open question
   `oq-gastown-awareness`). This is how Gas Town learns that winze is a KB
   project, not a software project, so the Mayor generates task types like
   "ingest this source" and "audit this neighborhood" instead of "fix this bug"
   and "add this feature." Worker roles: ingest, lint, audit, curator,
   refactor. Start with ingest — point it at one Stope essay and see what it
   produces.
5. **Run the first ingest.** Pick the smallest self-contained slice of Stope
   lore you can find. Feed it to the ingest worker. Commit the result. Run
   `go build ./...`. Run `adit score` on the commit. That is the first real
   datapoint on whether the encoding is tractable or painful.

Checkpoints 2-4 can be attempted in parallel by separate sub-sessions if you
want to test Gas Town-style orchestration from the start, but on a human-driven
session just go sequentially.

## What not to do in the first session

- Do not reintroduce `//go:embed` sidecars for prose. They are the dodge we
  already rejected. If you find yourself wanting one, that is a signal the
  claim schema isn't rich enough yet — fix the schema.
- Do not try to implement the LLM contradiction-detection lint rule before the
  deterministic lint rules and the ingest flow are working. It is the most
  expensive check and the most error-prone; it deserves dedicated thought and
  a benchmark, neither of which exist yet.
- Do not fork Gas Town. Write the skill package as a standalone artifact and
  load it into an unmodified Gas Town instance.
- Do not start the v1 benchmark yet. It is blocking (see open question
  `oq-benchmark`), but it only pays off once there is a real winze project to
  benchmark against. First get an ingest loop working.
- Do not delete this file. Replace it once winze can represent its own
  handoff state via defn queries — until then, this is load-bearing.

## Context pointers for the fresh session

- **This conversation's transcript** (the full founding discussion) is at
  `/home/gas6amus/.claude/projects/-home-gas6amus-Documents-mempalace/fb54580c-4527-4c9d-86b0-14942ca0a2fc.jsonl`.
  Treat it as a source for the first ingest, not as documentation.
- **Prior-art survey results**: two background research agent runs. First
  round: general prior-art survey confirming winze is in an unoccupied niche
  (closest analogs: Mathlib, Karpathy LLM Wiki, Glean, Unison, Lepiter).
  Second round: failure-mode deep dive with empirical data on Cyc/Mathlib
  ontology churn, grep-vs-structured-tool reach in agents, and contradiction
  detection recall rates. Both are in the mempalace session transcript.
- **Stope reference corpus** lives at
  `/home/justin/Documents/lamina/poc/dense/stope/reference/` — ~85 markdown
  essays, ~400k words. Winze's first real ingest target.
- **mempalace's LongMemEval benchmark infra** is reusable for the BM25 floor
  baseline when we get to the v1 winze benchmark. `benchmarks/longmemeval_bench.py`
  and `benchmarks/grepmem_bench.py` in `/home/gas6amus/Documents/mempalace`.

## Reminder about the nature of this file

This is markdown because the alternative (a fresh session with no winze-native
way to read typed Go state) doesn't exist yet. The first thing a mature winze
would do is generate its own handoff from the Dolt graph via a query worker.
Until then, this file is the scaffolding, and every edit to it is an
acknowledgement that winze can't yet describe itself. When it can, delete this
file and replace it with the query.
