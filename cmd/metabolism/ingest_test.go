package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot returns the winze repo root for tests that need real corpus files.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// thisFile is .../cmd/metabolism/ingest_test.go → go up 3 levels
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

func TestParseIngestResponse_WellFormed(t *testing.T) {
	response := claimDelimiter + `
ENTITY_NAME: Daniel Dennett
ENTITY_ID: daniel-dennett
ENTITY_KIND: person
ENTITY_BRIEF: American philosopher and cognitive scientist
PREDICATE: Disputes
TARGET: ChalmersHardProblemThesis
QUOTE: Dennett argues that consciousness is not a special phenomenon requiring new science.
EXPLANATION: Directly relevant to epistemology of minds
` + claimDelimiter + `
ENTITY_NAME: David Chalmers
ENTITY_ID: david-chalmers
ENTITY_KIND: person
ENTITY_BRIEF: Australian philosopher of mind
PREDICATE: Proposes
TARGET: HardProblemOfConsciousness
QUOTE: Chalmers coined the term hard problem of consciousness in 1995.
EXPLANATION: Foundational to the hard problem debate
`
	results := parseIngestResponse(response)
	if len(results) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(results))
	}

	r := results[0]
	if r.entityName != "Daniel Dennett" {
		t.Errorf("entity name = %q, want %q", r.entityName, "Daniel Dennett")
	}
	if r.entityID != "daniel-dennett" {
		t.Errorf("entity ID = %q, want %q", r.entityID, "daniel-dennett")
	}
	if r.entityKind != "person" {
		t.Errorf("entity kind = %q, want %q", r.entityKind, "person")
	}
	if r.predicate != "Disputes" {
		t.Errorf("predicate = %q, want %q", r.predicate, "Disputes")
	}
	if r.target != "ChalmersHardProblemThesis" {
		t.Errorf("target = %q, want %q", r.target, "ChalmersHardProblemThesis")
	}
	if !strings.Contains(r.quote, "Dennett argues") {
		t.Errorf("quote = %q, want to contain %q", r.quote, "Dennett argues")
	}

	r2 := results[1]
	if r2.entityName != "David Chalmers" {
		t.Errorf("claim 2 entity name = %q, want %q", r2.entityName, "David Chalmers")
	}
	if r2.predicate != "Proposes" {
		t.Errorf("claim 2 predicate = %q, want %q", r2.predicate, "Proposes")
	}
}

func TestParseIngestResponse_NoClaims(t *testing.T) {
	for _, input := range []string{"NO_CLAIMS", "NO_CLAIM", "NO_CLAIMS\n"} {
		results := parseIngestResponse(input)
		if results != nil {
			t.Errorf("parseIngestResponse(%q) = %d claims, want nil", input, len(results))
		}
	}
}

func TestParseIngestResponse_Malformed(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{
			name: "missing entity name",
			input: claimDelimiter + `
ENTITY_KIND: person
QUOTE: Some quote here
`,
			want: 0, // skipped: entityName required
		},
		{
			name: "missing quote",
			input: claimDelimiter + `
ENTITY_NAME: Someone
ENTITY_KIND: person
`,
			want: 0, // skipped: quote required
		},
		{
			name: "one good one bad",
			input: claimDelimiter + `
ENTITY_NAME: Good Person
QUOTE: Has a quote
` + claimDelimiter + `
ENTITY_KIND: person
`,
			want: 1,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := parseIngestResponse(tc.input)
			if len(results) != tc.want {
				t.Errorf("got %d claims, want %d", len(results), tc.want)
			}
		})
	}
}

func TestParseIngestResponse_Defaults(t *testing.T) {
	response := claimDelimiter + `
ENTITY_NAME: Jane Smith
QUOTE: Some research finding about consciousness
`
	results := parseIngestResponse(response)
	if len(results) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(results))
	}
	r := results[0]

	// Default entityID: kebab-case of name
	if r.entityID != "jane-smith" {
		t.Errorf("default entityID = %q, want %q", r.entityID, "jane-smith")
	}
	// Default entityKind: person
	if r.entityKind != "person" {
		t.Errorf("default entityKind = %q, want %q", r.entityKind, "person")
	}
	// Default predicate: Proposes
	if r.predicate != "Proposes" {
		t.Errorf("default predicate = %q, want %q", r.predicate, "Proposes")
	}
}

func TestParseIngestResponse_DelimiterInjection(t *testing.T) {
	// Simulate article text that contains the old delimiter — should not split
	response := claimDelimiter + `
ENTITY_NAME: Real Person
QUOTE: The study found ---CLAIM--- patterns in the data that suggest consciousness
`
	results := parseIngestResponse(response)
	if len(results) != 1 {
		t.Fatalf("expected 1 claim (old delimiter in quote should not split), got %d", len(results))
	}
	if !strings.Contains(results[0].quote, "---CLAIM---") {
		t.Logf("quote preserved old delimiter text: %q", results[0].quote)
	}
}

func TestToPascalCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"daniel-dennett", "Danieldennett"},   // hyphen stripped, single word (split on spaces only)
		{"Word Learning", "WordLearning"},
		{"consciousness", "Consciousness"},
		{"already-PascalCase", "AlreadyPascalCase"}, // hyphen stripped, P preserved
		{"123-start", "X123start"},                  // digit prefix gets X, hyphen stripped
		{"", ""},
		{"a b c", "ABC"},
		{"multi   space", "MultiSpace"},
		{"Daniel Dennett", "DanielDennett"},  // space-separated → proper PascalCase
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := toPascalCase(tc.input)
			if got != tc.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsValidGoIdent(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"Foo", true},
		{"_bar", true},
		{"x123", true},
		{"", false},
		{"123start", false},
		{"has-hyphen", false},
		{"has space", false},
		{"CamelCase", true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := isValidGoIdent(tc.input)
			if got != tc.valid {
				t.Errorf("isValidGoIdent(%q) = %v, want %v", tc.input, got, tc.valid)
			}
		})
	}
}

func TestKindToRole(t *testing.T) {
	cases := []struct {
		kind string
		role string
	}{
		{"person", "Person"},
		{"Person", "Person"},
		{"organization", "Organization"},
		{"concept", "Concept"},
		{"hypothesis", "Hypothesis"},
		{"event", "Event"},
		{"place", "Place"},
		{"widget", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			got := kindToRole(tc.kind)
			if got != tc.role {
				t.Errorf("kindToRole(%q) = %q, want %q", tc.kind, got, tc.role)
			}
		})
	}
}

func TestIsRelevantArticle(t *testing.T) {
	cases := []struct {
		name     string
		snippet  string
		title    string
		hypBrief string
		want     bool
	}{
		{"domain term in snippet", "Research on consciousness and perception", "Some Article", "", true},
		{"domain term in title", "Generic snippet text here", "Cognitive Bias Review", "", true},
		{"tennis article", "Rafael Nadal won the ATP Bologna tournament", "ATP Tennis", "", false},
		{"museum article", "The Maidstone Museum houses a collection of Egyptian artifacts", "Maidstone Museum", "", false},
		{"empty snippet", "", "Anything", "", true}, // empty snippet → assume relevant
		{"hypothesis word match", "Research on attention mechanisms in the brain", "Neural Study", "attention bottleneck hypothesis", true},
		{"short hyp word ignored", "The cat sat on the mat", "Cat Article", "a is b", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			article := PaperSummary{
				Title:   tc.title,
				Snippet: tc.snippet,
			}
			got := isRelevantArticle(article, tc.hypBrief)
			if got != tc.want {
				t.Errorf("isRelevantArticle(%q, %q) = %v, want %v", tc.snippet, tc.hypBrief, got, tc.want)
			}
		})
	}
}

func TestCollectPredicateSlots(t *testing.T) {
	root := repoRoot(t)
	slots := collectPredicateSlots(root)

	expected := map[string][2]string{
		"Proposes":    {"Person", "Hypothesis"},
		"Disputes":    {"Person", "Hypothesis"},
		"ProposesOrg": {"Organization", "Hypothesis"},
		"DisputesOrg": {"Organization", "Hypothesis"},
		"TheoryOf":    {"Hypothesis", "Concept"},
		"LocatedIn":   {"Place", "Place"},
	}

	for pred, want := range expected {
		got, ok := slots[pred]
		if !ok {
			t.Errorf("predicate %s not found in slots", pred)
			continue
		}
		if got != want {
			t.Errorf("slots[%s] = %v, want %v", pred, got, want)
		}
	}

	if len(slots) == 0 {
		t.Fatal("no predicate slots found — collectPredicateSlots may be broken")
	}
	t.Logf("found %d predicate slot definitions", len(slots))
}

func TestCollectKBMetadata(t *testing.T) {
	root := repoRoot(t)
	meta := collectKBMetadata(root)

	// Should find known entities
	if _, ok := meta.Vars["ChalmersHardProblemThesis"]; !ok {
		t.Error("ChalmersHardProblemThesis not found in Vars")
	}

	// Should have briefs
	if brief, ok := meta.Briefs["ChalmersHardProblemThesis"]; !ok || brief == "" {
		t.Error("ChalmersHardProblemThesis has empty or missing Brief")
	}

	// Should have claims context
	if claims, ok := meta.Claims["ChalmersHardProblemThesis"]; !ok || len(claims) == 0 {
		t.Error("ChalmersHardProblemThesis has no claim context")
	}

	// Sanity: should have a reasonable number of vars
	if len(meta.Vars) < 200 {
		t.Errorf("only %d vars found, expected 200+", len(meta.Vars))
	}
	t.Logf("collectKBMetadata: %d vars, %d briefs, %d claim targets",
		len(meta.Vars), len(meta.Briefs), len(meta.Claims))
}

func TestVerifyQuote(t *testing.T) {
	source := `David Chalmers coined the term "hard problem of consciousness"
in his 1995 paper "Facing Up to the Problem of Consciousness" and argues
that phenomenal experience cannot be reduced to physical processes.`

	cases := []struct {
		name  string
		quote string
		want  bool
	}{
		{
			name:  "exact substring",
			quote: `coined the term "hard problem of consciousness"`,
			want:  true,
		},
		{
			name:  "whitespace normalized match",
			quote: "David  Chalmers   coined the term",
			want:  true,
		},
		{
			name:  "case insensitive match",
			quote: `DAVID CHALMERS COINED THE TERM "HARD PROBLEM OF CONSCIOUSNESS"`,
			want:  true,
		},
		{
			name:  "LLM-added surrounding quotes stripped",
			quote: `"phenomenal experience cannot be reduced to physical processes"`,
			want:  true,
		},
		{
			name:  "fragment match (first 40 chars present in source)",
			quote: "phenomenal experience cannot be reduced to physical processes according to Chalmers and many others who follow his line of reasoning",
			want:  true,
		},
		{
			name:  "too short rejection",
			quote: "hard problem",
			want:  false,
		},
		{
			name:  "completely fabricated",
			quote: "Einstein said consciousness is quantum mechanics in action",
			want:  false,
		},
		{
			name:  "empty quote",
			quote: "",
			want:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := verifyQuote(source, tc.quote)
			if got != tc.want {
				t.Errorf("verifyQuote(source, %q) = %v, want %v", tc.quote, got, tc.want)
			}
		})
	}
}

func TestCleanLLMString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escaped quotes",
			input: `He said \"hello\"`,
			want:  `He said "hello"`,
		},
		{
			name:  "escaped newlines",
			input: `line1\nline2`,
			want:  "line1 line2",
		},
		{
			name:  "smart double quotes",
			input: "He said \u201chello\u201d",
			want:  `He said "hello"`,
		},
		{
			name:  "smart single quotes",
			input: "it\u2019s a \u2018test\u2019",
			want:  "it's a 'test'",
		},
		{
			name:  "leading/trailing whitespace",
			input: "  trimmed  ",
			want:  "trimmed",
		},
		{
			name:  "combined artifacts",
			input: "  \u201cHe said \\\"no\\\"\u201d\\n  ",
			want:  `"He said "no""`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanLLMString(tc.input)
			if got != tc.want {
				t.Errorf("cleanLLMString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGenerateClaimCode(t *testing.T) {
	article := PaperSummary{
		Title: "Test Article",
		ID:    "zim:Test_Article",
	}

	t.Run("well formed person proposes", func(t *testing.T) {
		result := &ingestResult{
			entityName:  "John Smith",
			entityID:    "john-smith",
			entityKind:  "person",
			entityBrief: "A philosopher",
			predicate:   "Proposes",
			target:      "SomeHypothesis",
			quote:       "Smith proposed the theory",
		}
		known := map[string]bool{}
		code := generateClaimCode("SomeHypothesis", "Proposes", article, result, 1, known)

		if !strings.Contains(code, "var JohnSmith = Person{") {
			t.Error("missing entity declaration")
		}
		if !strings.Contains(code, "var johnSmithProposesSource = Provenance{") {
			t.Error("missing provenance declaration")
		}
		if !strings.Contains(code, "Subject: JohnSmith,") {
			t.Error("missing Subject field")
		}
		if !strings.Contains(code, "Object:  SomeHypothesis,") {
			t.Error("missing Object field")
		}
	})

	t.Run("organization auto-converts to ProposesOrg", func(t *testing.T) {
		result := &ingestResult{
			entityName: "ACME Corp",
			entityID:   "acme-corp",
			entityKind: "organization",
			predicate:  "Proposes",
			target:     "SomeHypothesis",
			quote:      "ACME proposed it",
		}
		known := map[string]bool{}
		code := generateClaimCode("SomeHypothesis", "Proposes", article, result, 1, known)

		if !strings.Contains(code, "= ProposesOrg{") {
			t.Errorf("expected ProposesOrg for organization, got:\n%s", code)
		}
		if !strings.Contains(code, "Organization{") {
			t.Error("expected Organization role type for organization entity")
		}
	})

	t.Run("unary predicate has no Object", func(t *testing.T) {
		result := &ingestResult{
			entityName: "Test Bias",
			entityID:   "test-bias",
			entityKind: "concept",
			predicate:  "IsCognitiveBias",
			target:     "",
			quote:      "It is a bias",
		}
		known := map[string]bool{}
		code := generateClaimCode("", "IsCognitiveBias", article, result, 1, known)

		if strings.Contains(code, "Object:") {
			t.Error("unary predicate should not have Object field")
		}
		if !strings.Contains(code, "Subject: TestBias,") {
			t.Error("missing Subject field")
		}
	})

	t.Run("known entity skips entity declaration", func(t *testing.T) {
		result := &ingestResult{
			entityName: "Existing Entity",
			entityID:   "existing",
			entityKind: "person",
			predicate:  "Proposes",
			target:     "SomeTarget",
			quote:      "Some quote",
		}
		known := map[string]bool{
			"ExistingEntity": true, // entity already in KB
		}
		code := generateClaimCode("SomeTarget", "Proposes", article, result, 1, known)

		if strings.Contains(code, "var ExistingEntity = Person{") {
			t.Error("should not redeclare existing entity")
		}
		// But should still emit the claim
		if !strings.Contains(code, "Subject: ExistingEntity,") {
			t.Error("should still emit claim referencing existing entity")
		}
	})
}
