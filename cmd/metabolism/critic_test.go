package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseCriticVerdict ------------------------------------------------------

func TestParseCriticVerdict_Accept(t *testing.T) {
	cases := []string{
		"VERDICT: ACCEPT",
		"verdict: accept",
		"VERDICT:ACCEPT",
		"Some preamble.\n\nVERDICT: ACCEPT\n",
	}
	for _, in := range cases {
		v := parseCriticVerdict(in)
		if !v.Accept {
			t.Errorf("parseCriticVerdict(%q) = %+v, want Accept=true", in, v)
		}
		if v.Reason != "" {
			t.Errorf("parseCriticVerdict(%q).Reason = %q, want empty", in, v.Reason)
		}
	}
}

func TestParseCriticVerdict_RejectWithReason(t *testing.T) {
	in := "VERDICT: REJECT\nREASON: subject not named in quote"
	v := parseCriticVerdict(in)
	if v.Accept {
		t.Errorf("parseCriticVerdict reject case returned Accept=true: %+v", v)
	}
	// Reason is sanitized: lowercased, spaces → underscores.
	if v.Reason != "subject_not_named_in_quote" {
		t.Errorf("Reason = %q, want subject_not_named_in_quote", v.Reason)
	}
}

func TestParseCriticVerdict_RejectWithoutReason(t *testing.T) {
	in := "VERDICT: REJECT"
	v := parseCriticVerdict(in)
	if v.Accept {
		t.Error("expected Accept=false")
	}
	if v.Reason != "unspecified" {
		t.Errorf("Reason = %q, want unspecified", v.Reason)
	}
}

func TestParseCriticVerdict_TruncatesLongReason(t *testing.T) {
	long := strings.Repeat("a", 200)
	in := "VERDICT: REJECT\nREASON: " + long
	v := parseCriticVerdict(in)
	if v.Accept {
		t.Error("expected Accept=false")
	}
	if len(v.Reason) > 60 {
		t.Errorf("Reason length = %d, want <= 60", len(v.Reason))
	}
}

func TestParseCriticVerdict_AmbiguousDefaultsToAccept(t *testing.T) {
	cases := []string{
		"",
		"I think this claim is fine.",
		"REJECT this claim",     // missing VERDICT prefix
		"VERDICT: maybe",
	}
	for _, in := range cases {
		v := parseCriticVerdict(in)
		if !v.Accept {
			t.Errorf("ambiguous input %q should default to Accept=true, got %+v", in, v)
		}
	}
}

// --- truncateQuote -----------------------------------------------------------

func TestTruncateQuote(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 100, "short"},
		{"exact length", len("exact length"), "exact length"},
		{"this is too long for the limit", 10, "this is to […]"},
	}
	for _, c := range cases {
		got := truncateQuote(c.in, c.max)
		if got != c.want {
			t.Errorf("truncateQuote(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

// --- formatExemplars ---------------------------------------------------------

func TestFormatExemplars_Empty(t *testing.T) {
	got := formatExemplars(nil)
	if !strings.Contains(got, "no exemplars") {
		t.Errorf("expected 'no exemplars' marker, got %q", got)
	}
}

func TestFormatExemplars_FormatsAllFields(t *testing.T) {
	exemplars := []claimExemplar{
		{Subject: "Chalmers", Predicate: "Proposes", Object: "HardProblem", Quote: "There is no easy reduction…"},
		{Subject: "Bach", Predicate: "Authored", Object: "WTC", Quote: "Bach wrote it as a demonstration."},
	}
	got := formatExemplars(exemplars)
	for _, want := range []string{"Chalmers", "Proposes", "HardProblem", "Bach", "WTC", "Exemplar 1", "Exemplar 2"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatExemplars missing %q in output:\n%s", want, got)
		}
	}
}

// --- buildIngestCriticPrompt -------------------------------------------------

func TestBuildIngestCriticPrompt_ContainsRubricAndCandidate(t *testing.T) {
	candidate := &ingestResult{
		entityName:  "TestPerson",
		entityKind:  "person",
		entityBrief: "test brief",
		quote:       "This is the quote.",
	}
	prompt := buildIngestCriticPrompt(candidate, "Proposes", "SomeHypothesis", nil)
	for _, want := range []string{
		"NAMED-IN-QUOTE-AS-AGENT",
		"PREDICATE-FIT",
		"NOT-A-DUPLICATE-RENAMING",
		"QUALITY-PARITY-WITH-EXEMPLARS",
		"VERDICT: ACCEPT",
		"VERDICT: REJECT",
		"TestPerson",
		"Proposes",
		"SomeHypothesis",
		"This is the quote.",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("buildIngestCriticPrompt missing %q", want)
		}
	}
}

// --- buildTripCriticPrompt ---------------------------------------------------

func TestBuildTripCriticPrompt_ContainsRubricAndCandidate(t *testing.T) {
	conn := TripConnection{
		EntityA:   "EntA",
		EntityB:   "EntB",
		Predicate: "CommentaryOn",
		Rationale: "Both reflect a structural pattern.",
	}
	prompt := buildTripCriticPrompt(conn, nil)
	for _, want := range []string{
		"SUBSTANTIVE-ISOMORPHISM",
		"PREDICATE-PRECONDITIONS",
		"CATEGORY-FIT",
		"VERDICT: ACCEPT",
		"VERDICT: REJECT",
		"EntA",
		"EntB",
		"CommentaryOn",
		"Both reflect a structural pattern.",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("buildTripCriticPrompt missing %q", want)
		}
	}
}

// --- extractClaimIdentField --------------------------------------------------

func TestExtractClaimIdentField(t *testing.T) {
	src := `package p
var X = SomeType{
	Subject: Foo,
	Object:  Bar,
	Prov:    BazProv,
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var cl *ast.CompositeLit
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CompositeLit); ok {
			cl = c
			return false
		}
		return true
	})
	if cl == nil {
		t.Fatal("no composite literal found")
	}
	cases := map[string]string{
		"Subject": "Foo",
		"Object":  "Bar",
		"Prov":    "BazProv",
		"Missing": "",
	}
	for field, want := range cases {
		got := extractClaimIdentField(cl, field)
		if got != want {
			t.Errorf("extractClaimIdentField(%q) = %q, want %q", field, got, want)
		}
	}
}

// --- sampleHighQualityClaims -------------------------------------------------

func TestSampleHighQualityClaims_FiltersThinQuotes(t *testing.T) {
	dir := t.TempDir()
	corpusFile := filepath.Join(dir, "corpus.go")
	src := `package winze

var thinSource = Provenance{
	Origin: "test",
	Quote:  "thin",
}

var richSource = Provenance{
	Origin: "test",
	Quote:  "` + strings.Repeat("X", 250) + `",
}

var ThinClaim = Proposes{
	Subject: SomeAuthor,
	Object:  SomeHypothesis,
	Prov:    thinSource,
}

var RichClaim = Proposes{
	Subject: AnotherAuthor,
	Object:  AnotherHypothesis,
	Prov:    richSource,
}
`
	if err := os.WriteFile(corpusFile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := sampleHighQualityClaims(dir, 10, 200)
	if len(got) != 1 {
		t.Fatalf("want 1 exemplar (thin filtered out), got %d: %+v", len(got), got)
	}
	if got[0].Subject != "AnotherAuthor" {
		t.Errorf("wrong exemplar selected: %+v", got[0])
	}
}

func TestSampleHighQualityClaims_ExcludesMetabolismCycleFiles(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "corpus.go")
	cycleFile := filepath.Join(dir, "metabolism_cycle9.go")
	main := `package winze

var goodSource = Provenance{
	Quote: "` + strings.Repeat("Y", 250) + `",
}

var GoodClaim = Proposes{
	Subject: A,
	Object:  B,
	Prov:    goodSource,
}
`
	cycle := `package winze

var cycleSource = Provenance{
	Quote: "` + strings.Repeat("Z", 250) + `",
}

var CycleClaim = Proposes{
	Subject: A2,
	Object:  B2,
	Prov:    cycleSource,
}
`
	if err := os.WriteFile(mainFile, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cycleFile, []byte(cycle), 0644); err != nil {
		t.Fatal(err)
	}
	got := sampleHighQualityClaims(dir, 10, 200)
	if len(got) != 1 {
		t.Fatalf("want 1 (cycle file excluded), got %d", len(got))
	}
	if got[0].Subject != "A" {
		t.Errorf("wrong subject — cycle file should be excluded: %+v", got[0])
	}
}

func TestSampleHighQualityClaims_ExcludesSpeculativeProvenance(t *testing.T) {
	dir := t.TempDir()
	src := `package winze

var realSource = Provenance{
	Origin: "real source",
	Quote:  "` + strings.Repeat("R", 250) + `",
}

var specSource = Provenance{
	Origin: "winze trip cycle 99 (speculative cross-cluster connection)",
	Quote:  "Both X and Y exhibit speculative cross-cluster connection patterns ` + strings.Repeat(".", 200) + `",
}

var RealClaim = Proposes{
	Subject: Real1,
	Object:  Real2,
	Prov:    realSource,
}

var SpecClaim = CommentaryOn{
	Subject: Spec1,
	Object:  Spec2,
	Prov:    specSource,
}
`
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	got := sampleHighQualityClaims(dir, 10, 200)
	if len(got) != 1 {
		t.Fatalf("want 1 (speculative excluded), got %d: %+v", len(got), got)
	}
	if got[0].Subject != "Real1" {
		t.Errorf("wrong subject: %+v", got[0])
	}
}

func TestSampleHighQualityClaims_RespectsN(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("package winze\n\n")
	for i := 0; i < 10; i++ {
		b.WriteString("var src" + string(rune('a'+i)) + " = Provenance{\n")
		b.WriteString("\tQuote: \"" + strings.Repeat("X", 250) + "\",\n")
		b.WriteString("}\n\n")
		b.WriteString("var Claim" + string(rune('A'+i)) + " = Proposes{\n")
		b.WriteString("\tSubject: S" + string(rune('A'+i)) + ",\n")
		b.WriteString("\tObject: O" + string(rune('A'+i)) + ",\n")
		b.WriteString("\tProv: src" + string(rune('a'+i)) + ",\n")
		b.WriteString("}\n\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}
	got := sampleHighQualityClaims(dir, 3, 200)
	if len(got) != 3 {
		t.Errorf("n=3 should cap result at 3, got %d", len(got))
	}
}

func TestSampleHighQualityClaims_NoCorpusReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	got := sampleHighQualityClaims(dir, 5, 200)
	if len(got) != 0 {
		t.Errorf("empty corpus should yield empty exemplar list, got %d", len(got))
	}
}
