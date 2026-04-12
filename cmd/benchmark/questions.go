package main

// Question corpus for winze benchmark v0.1.
// 24 questions across 4 categories, gold answers verified against the KB.

type Question struct {
	ID       string
	Category string
	Query    string
	Gold     []string
	DefnSQL  string
}

var questions = []Question{
	// -------------------------------------------------------------------------
	// Category A: Lexical (6) — direct entity/fact lookup
	// -------------------------------------------------------------------------
	{
		ID:       "lex-01",
		Category: "lexical",
		Query:    "Who proposed the hard problem of consciousness?",
		Gold:     []string{"DavidChalmers"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var' AND d.source_file = 'hard_problem.go' AND d.name LIKE '%Chalmers%'`,
	},
	{
		ID:       "lex-02",
		Category: "lexical",
		Query:    "Who published the 1931 incompleteness theorems?",
		Gold:     []string{"KurtGodel"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var' AND d.name LIKE '%Godel%'`,
	},
	{
		ID:       "lex-03",
		Category: "lexical",
		Query:    "What is apophenia?",
		Gold:     []string{"Apophenia"},
		DefnSQL:  `SELECT name FROM definitions WHERE kind = 'var' AND name = 'Apophenia'`,
	},
	{
		ID:       "lex-04",
		Category: "lexical",
		Query:    "Who authored The Quantum Thief?",
		Gold:     []string{"HannuRajaniemi"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var' AND d.source_file = 'quantum_thief.go'`,
	},
	{
		ID:       "lex-05",
		Category: "lexical",
		Query:    "Who wrote The Demon-Haunted World?",
		Gold:     []string{"CarlSagan"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var' AND d.source_file = 'demon_haunted.go' AND d.name LIKE '%Sagan%'`,
	},
	{
		ID:       "lex-06",
		Category: "lexical",
		Query:    "Who led the 1927 Tunguska expedition?",
		Gold:     []string{"Kulik"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var' AND d.name = 'Kulik'`,
	},

	// -------------------------------------------------------------------------
	// Category B: Aggregation (6) — counting, grouping, set operations
	// -------------------------------------------------------------------------
	{
		ID:       "agg-01",
		Category: "aggregation",
		Query:    "How many hypotheses explain the Tunguska event?",
		Gold:     []string{"6"},
		DefnSQL:  `SELECT COUNT(*) as cnt FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'HypothesisExplains' AND d.kind = 'var'`,
	},
	{
		ID:       "agg-02",
		Category: "aggregation",
		Query:    "List all cognitive biases in the KB.",
		Gold:     []string{"ClusteringIllusion", "AvailabilityHeuristic", "AnchoringBias", "DunningKrugerEffect", "HotHandFallacy"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'IsCognitiveBias' AND d4.name = 'Concept' AND d3.kind = 'var' AND d3.name != 'IsCognitiveBias'`,
	},
	{
		ID:       "agg-03",
		Category: "aggregation",
		Query:    "How many entities of role type Person exist?",
		Gold:     []string{"41"},
		DefnSQL:  `SELECT COUNT(*) as cnt FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var'`,
	},
	{
		ID:       "agg-04",
		Category: "aggregation",
		Query:    "How many TheoryOf claims exist in the KB?",
		Gold:     []string{"28"},
		DefnSQL:  `SELECT COUNT(*) as cnt FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'TheoryOf' AND d.kind = 'var'`,
	},
	{
		ID:       "agg-05",
		Category: "aggregation",
		Query:    "List all InfluencedBy claims.",
		Gold:     []string{"RajaniemiInfluencedByLeblanc", "RajaniemiInfluencedByClark", "PinkerInfluencedByBrown"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'InfluencedBy' AND d.kind = 'var'`,
	},
	{
		ID:       "agg-06",
		Category: "aggregation",
		Query:    "List all Disputes claims.",
		Gold:     []string{"VolkerDisputesLoyFiveFlavors", "KatzDisputesPerennialism", "WattsDisputesAdvaitaAsMonism", "SekaninaDisputesCometary", "LongoDisputesCometary"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Disputes' AND d.kind = 'var'`,
	},

	// -------------------------------------------------------------------------
	// Category C: Multi-hop (6) — 2+ edge traversals
	// -------------------------------------------------------------------------
	{
		ID:       "hop-01",
		Category: "multi-hop",
		Query:    "What vars reference Consciousness as a TheoryOf target?",
		Gold:     []string{"WattsConsciousnessTheory", "SearleBiologicalNaturalismTheoryOfConsciousness", "ChalmersHardProblemTheoryOfConsciousness"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id WHERE d.kind = 'var' AND d2.name = 'Consciousness' AND d3.name = 'TheoryOf'`,
	},
	{
		ID:       "hop-02",
		Category: "multi-hop",
		Query:    "Who influenced the author of The Quantum Thief?",
		Gold:     []string{"MauriceLeblanc", "AndyClark"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'InfluencedBy' AND d.name LIKE 'Rajaniemi%' AND d4.name = 'Person' AND d3.kind = 'var' AND d3.name != 'HannuRajaniemi'`,
	},
	{
		ID:       "hop-03",
		Category: "multi-hop",
		Query:    "Who disputes a hypothesis that explains the Tunguska event?",
		Gold:     []string{"Sekanina", "Longo"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON r2.from_def = d.id JOIN definitions d3 ON r2.to_def = d3.id WHERE d2.name = 'Disputes' AND d.kind = 'var' AND d3.name IN ('Sekanina', 'Longo')`,
	},
	{
		ID:       "hop-04",
		Category: "multi-hop",
		Query:    "What fictional concepts appear in The Quantum Thief?",
		Gold:     []string{"JeanLeFlambeur", "Oubliette", "Sobornost", "Exomemory", "Mieli", "Perhonen"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'IsFictional' AND d3.kind = 'var' AND d4.name = 'Concept' AND d.source_file = 'quantum_thief.go' AND d3.name != 'IsFictional'`,
	},
	{
		ID:       "hop-05",
		Category: "multi-hop",
		Query:    "What people propose hypotheses about mathematical foundations?",
		Gold:     []string{"KurtGodel", "DavidHilbert"},
		DefnSQL:  `SELECT d.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id WHERE d2.name = 'Person' AND d.kind = 'var' AND d.source_file = 'godel_incompleteness.go'`,
	},
	{
		ID:       "hop-06",
		Category: "multi-hop",
		Query:    "Who influenced Steven Pinker?",
		Gold:     []string{"DonaldEBrown"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'InfluencedBy' AND d.name LIKE 'Pinker%' AND d4.name = 'Person' AND d3.kind = 'var' AND d3.name != 'StevenPinker'`,
	},

	// -------------------------------------------------------------------------
	// Category D: Contested/Dispute (6) — structural meta-queries
	// -------------------------------------------------------------------------
	{
		ID:       "con-01",
		Category: "contested",
		Query:    "What concepts have 3 or more competing TheoryOf subjects?",
		Gold:     []string{"Consciousness", "Nondualism"},
		DefnSQL:  `SELECT target.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions theoryof ON r.to_def = theoryof.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions target ON r2.to_def = target.id JOIN refs r3 ON target.id = r3.from_def JOIN definitions targettype ON r3.to_def = targettype.id WHERE d.kind = 'var' AND theoryof.name = 'TheoryOf' AND targettype.name = 'Concept' AND target.kind = 'var' GROUP BY target.name HAVING COUNT(DISTINCT d.name) >= 3`,
	},
	{
		ID:       "con-02",
		Category: "contested",
		Query:    "List all contested concept targets in the KB.",
		Gold:     []string{"Apophenia", "CognitiveBias", "Consciousness", "HumanCognition", "MathematicalFoundations", "NondualAwareness", "Nondualism", "Schizophrenia"},
		DefnSQL:  `SELECT target.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions theoryof ON r.to_def = theoryof.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions target ON r2.to_def = target.id JOIN refs r3 ON target.id = r3.from_def JOIN definitions targettype ON r3.to_def = targettype.id WHERE d.kind = 'var' AND theoryof.name = 'TheoryOf' AND targettype.name = 'Concept' AND target.kind = 'var' GROUP BY target.name HAVING COUNT(DISTINCT d.name) >= 2`,
	},
	{
		ID:       "con-03",
		Category: "contested",
		Query:    "How many KnownDispute annotations exist?",
		Gold:     []string{"3"},
		DefnSQL:  `SELECT COUNT(*) as cnt FROM definitions WHERE kind = 'var' AND name LIKE '%Dispute' AND exported = 1`,
	},
	{
		ID:       "con-04",
		Category: "contested",
		Query:    "What are the rival theories of Schizophrenia?",
		Gold:     []string{"MattsonSchizophreniaSPPDysregulationFraming", "WhiteShergillReducedTopDownFraming"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'TheoryOf' AND d3.kind = 'var' AND d4.name = 'Hypothesis' AND d.name LIKE '%Schizophrenia%'`,
	},
	{
		ID:       "con-05",
		Category: "contested",
		Query:    "What are the rival theories of Apophenia?",
		Gold:     []string{"ConradApopheniaClinicalFraming", "ShermerPatternicityFraming"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'TheoryOf' AND d3.kind = 'var' AND d4.name = 'Hypothesis' AND d.name LIKE '%Apophenia%'`,
	},
	{
		ID:       "con-06",
		Category: "contested",
		Query:    "What concepts are tagged IsPolyvalentTerm?",
		Gold:     []string{"Forecasting", "Nondualism"},
		DefnSQL:  `SELECT DISTINCT d3.name FROM definitions d JOIN refs r ON r.from_def = d.id JOIN definitions d2 ON r.to_def = d2.id JOIN refs r2 ON d.id = r2.from_def JOIN definitions d3 ON r2.to_def = d3.id JOIN refs r3 ON d3.id = r3.from_def JOIN definitions d4 ON r3.to_def = d4.id WHERE d.kind = 'var' AND d2.name = 'IsPolyvalentTerm' AND d4.name = 'Concept' AND d3.kind = 'var' AND d3.name != 'IsPolyvalentTerm'`,
	},
}
