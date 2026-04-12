// Package winze is a non-executable Go codebase used as a personal knowledge base.
//
// This file is the founding bootstrap. It encodes the entities, decisions,
// failure modes, mitigations, and open questions established in the winze
// founding conversation. It is intended to be read by a fresh Claude Code
// session (via defn, once defn is wired up) and extended by autonomous
// knowledge-worker agents running under Gas Town.
//
// The package compiles but is never executed. `go build` is the consistency
// checker: unresolved references, type mismatches, and rename collisions fail
// the build and block the commit.
//
// See the Decision, FailureMode, Mitigation, and OpenQuestion declarations
// below for the architectural narrative.
package winze

// Entity is a named thing the KB tracks. Kind is an open string; current
// values include "tool", "project", "concept", "paper", "person",
// "character", "place", "organization", "event", "instrument". Aliases
// holds surface-form variants an ingest worker resolved to this entity.
type Entity struct {
	ID      string
	Name    string
	Kind    string
	Brief   string
	Aliases []string
}

// Decision is a locked-in architectural choice. Decisions are load-bearing:
// the rest of the KB assumes they hold. Changing one is a branch-level event.
type Decision struct {
	ID        string
	Title     string
	Rationale string
}

// FailureMode is a known way the architecture can break. Severity is 1 (low)
// to 3 (dealbreaker).
type FailureMode struct {
	ID          string
	Title       string
	Severity    int
	Description string
}

// Mitigation is a defense against a FailureMode. If Automated is true the
// mitigation runs as a winze lint rule; otherwise it is a procedural discipline.
type Mitigation struct {
	ID        string
	Addresses *FailureMode
	Rule      string
	Automated bool
}

// OpenQuestion is something not yet resolved. Blocking questions should be
// answered before implementation work depends on their outcome. When resolved,
// set Resolution to describe the outcome and Blocking to false.
type OpenQuestion struct {
	ID         string
	Title      string
	Blocking   bool
	Resolution string // empty = still open; non-empty = resolved
}

// -----------------------------------------------------------------------------
// Tools that winze depends on or coordinates with.
// -----------------------------------------------------------------------------

var (
	Defn = &Entity{
		ID:    "defn",
		Name:  "defn",
		Kind:  "tool",
		Brief: "Parses Go via go/types. Stores reference graph in Dolt. Exposes MCP ops: read, search, impact, explain, rename, simulate, query, branch, merge.",
	}

	Adit = &Entity{
		ID:    "adit",
		Name:  "adit-code",
		Kind:  "tool",
		Brief: "Measures agent-ergonomics of codebases: file size, grep noise, blast radius, ambiguous names, unneeded reads. Validated against SWE-bench trajectories; file size is the dominant cost predictor (median Spearman +0.474 across 49 repos).",
	}

	Dolt = &Entity{
		ID:    "dolt",
		Name:  "Dolt",
		Kind:  "tool",
		Brief: "SQL database with git semantics (branch, merge, diff, blame on structured data). Used by defn for the reference graph and by Gas Town as the state store.",
	}

	GasTown = &Entity{
		ID:    "gastown",
		Name:  "Gas Town",
		Kind:  "tool",
		Brief: "Yegge's Go-based orchestrator for fleets of Claude Code instances. Built on Dolt. Worker roles: Mayor, Polecats, Convoys. Merge queue and quality gates built in. Winze runs as a Gas Town project, not a fork.",
	}

	Wasteland = &Entity{
		ID:    "wasteland",
		Name:  "The Wasteland",
		Kind:  "tool",
		Brief: "Federation layer over Gas Towns via Dolt fork/merge. Shared Wanted Board of work across users. Relevant to winze only if knowledge-sharing across users becomes a goal.",
	}
)

// -----------------------------------------------------------------------------
// Reference projects that anchor the design.
// -----------------------------------------------------------------------------

var (
	Mathlib = &Entity{
		ID:    "mathlib",
		Name:  "Mathlib (Lean 4)",
		Kind:  "project",
		Brief: "Closest existing analog: ~1.9M lines of typed math with a compiler-checked reference graph. Survives ~15 years of schema evolution via rename-with-deprecation, bot-assisted PR review, and auto-migration tooling. The discipline winze should steal.",
	}

	Cyc = &Entity{
		ID:    "cyc",
		Name:  "Cyc",
		Kind:  "project",
		Brief: "Ontology-churn cautionary tale. ~100 person-years spent on a single representation-language migration ~6 years in. Proof that ontology churn is real; Mathlib is proof it is survivable.",
	}

	KarpathyWiki = &Entity{
		ID:    "karpathy-wiki",
		Name:  "Karpathy LLM Wiki",
		Kind:  "project",
		Brief: "Markdown-based LLM-maintained KB. Winze's sibling, not its competitor: same goal (LLM-authored personal KB), different substrate (prose vs. typed code). Dominant 2026 pattern.",
	}

	Glean = &Entity{
		ID:    "glean",
		Name:  "Glean (Meta)",
		Kind:  "project",
		Brief: "Datalog facts extracted from code, queryable. Closest architectural analog to 'defn is a fact database over a codebase.' Describes code; winze uses the same shape to store knowledge.",
	}

	Mempalace = &Entity{
		ID:    "mempalace",
		Name:  "mempalace",
		Kind:  "project",
		Brief: "Prior project. grepmem FTS5 baseline achieved 96.4% R@5 on LongMemEval-s, establishing BM25 over text as the retrieval floor winze must justify against. Benchmark infrastructure is reusable for the v1 winze benchmark.",
	}

	LongMemEval = &Entity{
		ID:    "longmemeval",
		Name:  "LongMemEval",
		Kind:  "paper",
		Brief: "Conversational long-term memory retrieval benchmark. Lexical-recall heavy; BM25 is at ceiling. Insufficient for discriminating structural KB approaches but useful as a control.",
	}

	RippleEdits = &Entity{
		ID:    "rippleedits",
		Name:  "RippleEdits",
		Kind:  "paper",
		Brief: "Knowledge-editing benchmark measuring whether fact edits propagate to logically entailed consequences. Six criteria: logical generalization, compositionality, subject aliasing, relation specificity, preservation, forgetfulness. Directly adaptable as a winze mutation-consistency eval.",
	}
)

// -----------------------------------------------------------------------------
// Decisions locked in during the founding conversation.
// -----------------------------------------------------------------------------

var (
	NonExecutableIsLoadBearing = &Decision{
		ID:        "dec-non-executable",
		Title:     "winze code is compilable but never executed",
		Rationale: "`go build` is the consistency checker. Executability is explicitly NOT a requirement; if it emerges it emerges, but optimizing for it would pull the encoding back into code-shaped constraints (goroutine safety, allocation, I/O). Dropping execution means the encoding only owes the type checker.",
	}

	CodeEqualsKnowledge = &Decision{
		ID:        "dec-same-picture",
		Title:     "Code editing and knowledge manipulation are the same operation",
		Rationale: "SWE-bench agent trajectories and knowledge-authoring trajectories are the same task shape: read, search, trace, edit, validate, commit. The 'domain' label is cosmetic. This is why adit's SWE-bench file-size correlation transfers to winze: the agent doesn't know or care that the file holds a character sheet instead of a router handler.",
	}

	NoHumanAuthors = &Decision{
		ID:        "dec-no-humans",
		Title:     "Humans never read or write winze directly",
		Rationale: "Drops every format assumption that optimizes for eyeballs (markdown, frontmatter, hierarchy prettiness). Ugliness is free. Encoding optimizes for agent operations: short stable symbols, co-located references, aggressive refactor churn allowed because no human has a mental model to invalidate.",
	}

	GasTownIsTheOrchestrator = &Decision{
		ID:        "dec-use-gastown",
		Title:     "Use Gas Town as the worker-agent orchestrator. Do not fork.",
		Rationale: "Gas Town already solves multi-agent orchestration over Dolt with merge queues and quality gates. Winze is a codebase Gas Town points at, not a replacement for it. Winze-specific workers become Gas Town polecat skill packages. Gas Town is a moving target; tracking upstream as a user is cheaper than tracking it as a fork.",
	}

	LLMAsExpensiveLintRule = &Decision{
		ID:        "dec-llm-as-lint",
		Title:     "LLM judgment is one lint rule among many, not a separate architectural stage",
		Rationale: "Uniform Rule interface where some rules happen to call an LLM and cost more than others. Runs opt-in with an explicit token budget. Slots post-commit, not inline. The failure-mode survey established that inline-author-as-judge is unreliable (~80% adversarial-contradiction acceptance rate); a separate-prompt separate-context judge pass is necessary.",
	}

	ProseIsInputOutputNotState = &Decision{
		ID:    "dec-prose-is-io",
		Title: "winze storage is 100% typed code. Prose is an input/output format, not state. Source documents are transient.",
		Rationale: "Embedding prose as //go:embed sidecars was considered and rejected: it preserves essays losslessly but makes every entity mention inside the prose invisible to the graph, breaking rename propagation and contradiction detection at the sidecar boundary. Instead: ingest workers consume prose sources and produce typed claims as their commit. Query workers render typed claims back to prose on demand. Source documents are NOT retained by winze as live ground truth — the KB is the canonical representation of the knowledge, full stop. Each claim's Provenance holds a human hint (Origin), an ingest date, and the specific source fragment (Quote) that supported the extraction; when the source is gone, the quote IS the audit record. Markdown re-export (query workers rendering typed claims back to prose) is icing that can be built later. This makes the encoding harder (sentence/claim schemas must type the prose itself) but keeps the compiler check covering the whole KB.",
	}

	AditIsTheCostOracle = &Decision{
		ID:        "dec-adit-oracle",
		Title:     "adit is the cost oracle for winze authoring ergonomics",
		Rationale: "adit already measures 'how expensive is this artifact for an agent to work with' and is validated against SWE-bench trajectories. The same metrics (file size, grep noise, blast radius, ambiguous names, unneeded reads) apply to winze with no modification, because of dec-same-picture. adit runs as a Gas Town quality gate: bad scores block commits.",
	}

	NoUpperOntologyImport = &Decision{
		ID:    "dec-no-upper-ontology",
		Title: "Do not import an upper ontology. Use existing vocabularies only as a naming oracle during ingest.",
		Rationale: "Importing Cyc/DOLCE/SUMO/BFO/Schema.org as a dependency forces winze's shape to fit theirs, and the Cyc cautionary tale (fm-ontology-churn) warns that upper-ontology migration is the exact 100-person-year sinkhole we want to avoid. The vast majority of a personal KB's claims live below the level any upper ontology operates at. But the naming-churn subclass of ontology churn IS cheaply mitigable: when an ingest worker invents a novel type name, a lint rule grounds it against a lookup table of well-known external terms (Schema.org, WordNet, Wikidata class names) and prefers the established name on promotion from pending/. This is the Mathlib naming-discipline pattern — don't import mathematics, but don't let a worker call something Foo when Monoid is the standard name. Wikidata is kept separately in scope as an entity-linking oracle (distinct from ontology import): real-world entities like Church Rock spill or United Nuclear can be resolved to Q-IDs by an enrichment worker; fictional entities stay winze-native.",
	}
)

// -----------------------------------------------------------------------------
// Failure modes identified by the first-round literature survey.
// -----------------------------------------------------------------------------

var (
	OntologyChurn = &FailureMode{
		ID:       "fm-ontology-churn",
		Title:    "The Cyc curve: ontology churn outpaces fact authoring",
		Severity: 2,
		Description: "LLM worker re-types the world every few sessions. Tokens spent on structural refactoring dominate tokens spent adding knowledge. Proven real by Cyc (~100 person-years on one migration). Proven survivable by Mathlib via rename-with-deprecation grace period + auto-migration + bot-assisted PR review. Mitigation is engineering, not research.",
	}

	QueryabilityMirage = &FailureMode{
		ID:       "fm-queryability-mirage",
		Title:    "Agents default to grep even when structured queries exist",
		Severity: 2,
		Description: "Empirical: Claude Code's own architecture prefers lexical search; Augment's SWE-bench analysis found grep beat embeddings because agents persisted rather than chose better tools; LSP-in-agents studies show agents fall back to grep when structured tools require precision. Engineerable via forced-briefing patterns, grep-hostile substrates, and structured-tool outputs strictly smaller than grep outputs.",
	}

	ConsistencyIsNotCorrectness = &FailureMode{
		ID:       "fm-consistency-vs-correctness",
		Title:    "go build passes even on contradictory claims",
		Severity: 3,
		Description: "The most dangerous failure mode. The compiler catches references and type mismatches; it does not catch 'Betty trusts Quamash' AND 'Betty distrusts Quamash' both being present. SOTA LLMs accept adversarial contradictions ~80% of the time. Best-in-class hybrid logic+LLM contradiction detectors hit only 60% recall. Requires a dedicated contradiction-detection lint rule running post-commit with a separate-prompt separate-context LLM.",
	}
)

// -----------------------------------------------------------------------------
// Mitigations. Each Addresses one FailureMode. If Automated the mitigation
// is (or will be) implemented as a winze lint rule.
// -----------------------------------------------------------------------------

var (
	RenameWithDeprecation = &Mitigation{
		ID:        "mit-rename-deprecation",
		Addresses: OntologyChurn,
		Rule:      "rename operations emit a deprecation alias that still compiles for N commits before deletion; rename log stored in Dolt; churn KPI (renames_per_week / new_facts_per_week) alerts if >0.3 for 3 weeks",
		Automated: true,
	}

	// Obsoleted 2026-04-11: pending/ subpackage as physical boundary is
	// premature ceremony. defn already tracks usage count via the reference
	// graph ("this type has N references, no rename history" is a query,
	// not a package split). Flat root package is easier on author agents and
	// delivers the same promotion-via-reference-count signal without cross-
	// package friction. Retained here as a history entry; see mit-naming-oracle
	// for the replacement mitigation on the ontology-churn failure mode.
	PendingSubpackageObsolete = &Mitigation{
		ID:        "mit-pending-obsolete",
		Addresses: OntologyChurn,
		Rule:      "OBSOLETE: pending/ as physical subpackage. Replaced by mit-naming-oracle + defn-query maturity check against a flat root package.",
		Automated: false,
	}

	NamingOracle = &Mitigation{
		ID:        "mit-naming-oracle",
		Addresses: OntologyChurn,
		Rule:      "every role-type declaration is lint-checked against ExternalTerms (Schema.org, WordNet, Wikidata class names) before promotion. When an author agent invents `Substance` and Wikidata already has it, the lint rule flags: use the external term. Prevents the naming-churn subclass of ontology churn deterministically and cheaply. Implemented in cmd/lint.",
		Automated: true,
	}

	ForcedBriefing = &Mitigation{
		ID:        "mit-forced-briefing",
		Addresses: QueryabilityMirage,
		Rule:      "every worker pre-reads defn.brief(target) before its first write; structured_calls / total_calls KPI target >= 0.7",
		Automated: true,
	}

	GrepHostileSubstrate = &Mitigation{
		ID:        "mit-grep-hostile",
		Addresses: QueryabilityMirage,
		Rule:      "Go source uses short stable symbols. Prose is not stored in the .go files at all — it is typed at ingest into Claim/Scene/Beat values. Grepping the .go files returns structural noise; useful retrieval requires defn queries.",
		Automated: false,
	}

	PredicateDisjointness = &Mitigation{
		ID:        "mit-predicate-disjoint",
		Addresses: ConsistencyIsNotCorrectness,
		Rule:      "//winze:disjoint pragma on function or method pairs; deterministic lint enforces across Dolt reference graph; cheap, catches the motivating 'trusts vs distrusts' example without any LLM",
		Automated: true,
	}

	SeparateJudgePass = &Mitigation{
		ID:        "mit-separate-judge",
		Addresses: ConsistencyIsNotCorrectness,
		Rule:      "post-commit LLM lint rule with different prompt and context from the authoring worker; budgeted token ceiling; runs async on nightly or pre-merge schedule; logs disagreements with the author",
		Automated: true,
	}

	RippleEditsEval = &Mitigation{
		ID:        "mit-rippleedits-eval",
		Addresses: ConsistencyIsNotCorrectness,
		Rule:      "port RippleEdits 6-criterion eval to winze shape; run nightly on held-out seed set; report recall per criterion; acceptance criterion for the SeparateJudgePass rule",
		Automated: true,
	}
)

// -----------------------------------------------------------------------------
// Open questions. Blocking questions should be answered before the
// implementation work that depends on them.
// -----------------------------------------------------------------------------

func DefineWinzeWorkerRoles() *OpenQuestion {
	_ = GasTown
	_ = Defn
	_ = Adit
	return &OpenQuestion{
		ID:         "oq-worker-roles",
		Title:      "Define winze worker roles as Gas Town polecat skill packages: ingest, lint, audit, curator, refactor. What does each prompt look like, what context does it need, what tools does it call?",
		Blocking:   false,
		Resolution: "Resolved session 11+17. mol-curate formula (5-step curation workflow), kb-health patrol plugin, curate skill package, and 40+ Gas Town formulas installed. Polecats run metabolism cycles autonomously.",
	}
}

func DesignRuleRegistry() *OpenQuestion {
	_ = LLMAsExpensiveLintRule
	return &OpenQuestion{
		ID:       "oq-rule-registry",
		Title:    "Design the Rule interface and pipeline. Cheap/medium/expensive tiers. Deterministic vs LLM-backed. Budget enforcement. Ship ~10 cheap rules first; LLM rules later.",
		Blocking: true,
	}
}

func BenchmarkDesign() *OpenQuestion {
	_ = Mempalace
	_ = LongMemEval
	_ = RippleEdits
	return &OpenQuestion{
		ID:       "oq-benchmark",
		Title:    "Construct v1 benchmark: dual-format corpus, four query categories (lexical, multi-hop, counterfactual, contradiction), BM25 baseline from mempalace grepmem infra, RippleEdits-adapted mutation tests, adit authoring-cost axis.",
		Blocking: true,
	}
}

func OntologyChurnUnderLLMAuthors() *OpenQuestion {
	_ = OntologyChurn
	_ = Mathlib
	return &OpenQuestion{
		ID:       "oq-churn-llm",
		Title:    "Does an LLM worker that re-types entities frequently stabilize under Mathlib-style deprecation discipline, or diverge? Measurable via the churn KPI after 1-2 weeks of active authoring.",
		Blocking: false,
	}
}

func AditMetricsOnKnowledgeWorkloads() *OpenQuestion {
	_ = Adit
	_ = CodeEqualsKnowledge
	return &OpenQuestion{
		ID:       "oq-adit-transfer",
		Title:    "Empirically verify adit's cost metrics correlate with agent cost on knowledge-authoring tasks, not just code-editing tasks. Run a task battery against progressively worse winze layouts and confirm the cost curve matches adit's predictions.",
		Blocking: false,
	}
}

func ContradictionDetectionLintRule() *OpenQuestion {
	_ = SeparateJudgePass
	_ = ConsistencyIsNotCorrectness
	return &OpenQuestion{
		ID:         "oq-contradiction-rule",
		Title:      "Implement the first LLM-backed lint rule: contradiction detection on changed neighborhoods. What prompt? What context framing? What budget? Target recall against a small seeded contradiction corpus.",
		Blocking:   false,
		Resolution: "Resolved session 10. llm-contradiction rule in cmd/lint using Anthropic Go SDK. Opt-in via --llm flag with --llm-max-calls budget cap. Checks claim neighborhoods for semantic contradictions.",
	}
}

// -----------------------------------------------------------------------------
// Empirical findings from sessions 1-12. These were discovered during
// implementation and are load-bearing calibrations, not just history.
// -----------------------------------------------------------------------------

var (
	SchemaAccretionRate = &Decision{
		ID:    "dec-schema-accretion-rate",
		Title: "Schema accretes at 0-1 new predicates per distinct corpus shape",
		Rationale: "After 17+ ingests across 8 corpus shapes (Wikipedia articles, journal papers, fiction, legal documents, game design docs, course handouts, commentaries, taxonomy lists), the predicate vocabulary converged. 11 consecutive slices required zero new predicates. New predicates emerge only when a structurally novel corpus shape forces them (CommentaryOn from paper-on-paper, AuthoredOrg from institutional authorship).",
	}

	ValueConflictIsSemantic = &Decision{
		ID:    "dec-value-conflict-semantic",
		Title: "Value-conflict lint operates at the semantic level (same subject+predicate+different objects), not type-level disjointness",
		Rationale: "The original mit-predicate-disjoint proposed compile-time pragma-based disjointness (e.g., Trusts vs Distrusts). Implementation revealed the real pattern is functional-predicate value conflicts: the same (predicate, subject) pair with different object values. This is caught by the //winze:functional pragma on predicate type declarations.",
	}

	ReificationOverSchemaExtension = &Decision{
		ID:    "dec-reification-over-extension",
		Title: "Handle competing theories via Hypothesis entities + TheoryOf, not new role types",
		Rationale: "When multiple theories compete to explain a concept (consciousness, human cognition, mathematical foundations), each theory is a Hypothesis entity with a TheoryOf claim pointing at the contested Concept. The //winze:contested annotation on TheoryOf signals that multiple subjects per object is expected. This avoids schema proliferation (no ConsciousnessTheoryA role type) and lets the contested-concept lint rule surface the landscape automatically.",
	}

	MirrorSourceCommitmentsValidated = &Decision{
		ID:    "dec-mirror-source",
		Title: "Ingest workers refuse to structure claims the source leaves unstructured",
		Rationale: "If a source mentions a concept without committing to a specific relationship, the ingest worker records it in the Brief (free text) but does not fabricate a typed claim. This discipline was validated by the misconceptions slice: the source refused to state the misconceptions themselves, so winze records only the corrections. Provenance.Quote is the audit mechanism.",
	}

	SlotTypeDisciplineValidated = &Decision{
		ID:    "dec-slot-type-validated",
		Title: "Role-typed predicate slots catch real concept/claim category errors at compile time",
		Rationale: "Validated by the UDHR ingest: attempting to use Authored[Person, Concept] for institutional authorship (UN General Assembly) failed to compile, forcing the creation of AuthoredOrg[Organization, Concept]. The compiler caught a category error that prose-based KBs would silently accept.",
	}
)

// ClaimSchemaDesign is the biggest unsolved piece of the architecture and the
// thing the first real session should chew on. Without a good claim schema,
// ingest workers have nowhere to put what they extract, and the whole "typed
// not embedded" commitment (dec-prose-is-io) has no teeth.
func ClaimSchemaDesign() *OpenQuestion {
	_ = ProseIsInputOutputNotState
	_ = OntologyChurn
	return &OpenQuestion{
		ID:         "oq-claim-schema",
		Title:      "Design the claim-level schema: what is a Claim, a Scene, a Relationship, a TemporalMarker, an Assertion? How granular? How does a sentence map to graph nodes? How are aliases and coreferences resolved at ingest time? This is the hard encoding problem we were dodging with //go:embed sidecars. Expect the schema to churn (see fm-ontology-churn) and use rename-with-deprecation from day one.",
		Blocking:   false,
		Resolution: "Resolved sessions 3-8. Claims are typed predicate instances: BinaryRelation[S,O] (two-slot) and UnaryClaim[S] (one-slot). 30+ predicates across 8 families in predicates.go. 16 role types in roles.go grounded in Schema.org/WordNet/Wikidata. Schema accreted organically — 11 consecutive slices required zero new predicates.",
	}
}

// GasTownProjectAwareness captures the point that a naive Gas Town pointed at
// winze will try to "build features" and "fix bugs" — wrong tasks. The winze
// skill package must teach Gas Town what kinds of tasks exist in a KB project
// (ingest, audit, rename-propagation, claim-gap-filling, contradiction-sweep)
// so the Mayor generates the right work for polecats.
func GasTownProjectAwareness() *OpenQuestion {
	_ = GasTown
	_ = GasTownIsTheOrchestrator
	return &OpenQuestion{
		ID:         "oq-gastown-awareness",
		Title:      "Write the winze skill package for Gas Town. It must encode not just worker prompts but project-type awareness: what tasks exist in a winze project, what quality gates apply, what the Mayor's backlog should look like. Without this, Gas Town will generate code-shaped tasks against a KB-shaped repo.",
		Blocking:   false,
		Resolution: "Resolved session 11+17. Skill package (.claude/skills/curate/), mol-curate formula (5-step: load-context, source-analysis, ingest, validate, submit), kb-health patrol plugin (2h cooldown), beads issue tracking integrated. Gas Town rig operational with witness + refinery.",
	}
}
