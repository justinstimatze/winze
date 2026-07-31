package corpusparse

import (
	"os"
	"path/filepath"
	"testing"
)

const fixtureAttribution = `package winze

var vaultSource = Provenance{
	Origin:     "PKM vault / free_energy_principle.md",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze pkm-ingest",
	Quote:      "The free energy principle says systems minimise surprise.",
}

var Friston = Person{&Entity{ID: "friston", Name: "Karl Friston", Kind: "person", Brief: "b"}}
var FEP = Hypothesis{&Entity{ID: "fep", Name: "Free energy principle", Kind: "hypothesis", Brief: "b"}}
var Surprise = Concept{&Entity{ID: "surprise", Name: "Surprise", Kind: "concept", Brief: "b"}}

var FristonProposesFEP = Proposes{
	Subject: Friston,
	Object:  FEP,
	Prov:    vaultSource,
}

var InlineSourced = CommentaryOn{
	Subject: FEP,
	Object:  Surprise,
	Prov: Provenance{
		Origin:     "inline origin",
		IngestedAt: "2026-05-01",
		IngestedBy: "winze",
		Quote:      "inline quote",
	},
}

var TripCycle9FEPCommentaryOnSurprise = CommentaryOn{
	Subject: FEP,
	Object:  Surprise,
	Prov: Conjecture{
		GeneratedBy:      "metabolism-trip",
		From:             []*Entity{FEP.Entity, Surprise.Entity},
		CycleN:           9,
		Temperature:      1.0,
		PromptType:       "analogy",
		Score:            4,
		Rationale:        "winze's own reasoning",
		GeneratedAt:      "2026-04-27",
		GeneratedByAgent: "winze-metabolism-trip",
	},
}

var KeyedCarrier = SourceDoc{
	Entity: &Entity{ID: "doc-lock", Name: "Corpus write lock", Kind: "sourcedoc", Brief: "b"},
	Refs: []CodeRef{
		{Symbol: corpuslock.Acquire, Path: "internal/corpuslock.Acquire", Note: "takes the flock"},
		{Symbol: corpuslock.Release, Path: "internal/corpuslock.Release"},
	},
}
`

func parseFixture(t *testing.T, src string) *Corpus {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := ParseCorpusFull(dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func claimByVar(t *testing.T, c *Corpus, name string) Claim {
	t.Helper()
	for _, cl := range c.Claims {
		if cl.VarName == name {
			return cl
		}
	}
	t.Fatalf("claim %s not parsed", name)
	return Claim{}
}

func TestParseProvenanceVars(t *testing.T) {
	c := parseFixture(t, fixtureAttribution)
	p, ok := c.ProvenanceByVar()["vaultSource"]
	if !ok {
		t.Fatal("provenance var not parsed")
	}
	if p.Origin != "PKM vault / free_energy_principle.md" || p.IngestedAt != "2026-04-13" ||
		p.IngestedBy != "winze pkm-ingest" || p.Quote == "" {
		t.Errorf("provenance fields wrong: %+v", p)
	}
	// A provenance var is not a claim and must not be counted as one.
	for _, cl := range c.Claims {
		if cl.VarName == "vaultSource" {
			t.Error("provenance var misparsed as a claim")
		}
	}
}

func TestClaimAttributionIsExactlyOneOf(t *testing.T) {
	c := parseFixture(t, fixtureAttribution)

	shared := claimByVar(t, c, "FristonProposesFEP")
	if shared.ProvVar != "vaultSource" || shared.Conjectural || shared.Conj != nil || shared.ProvInline != nil {
		t.Errorf("shared-var claim: %+v", shared)
	}

	inline := claimByVar(t, c, "InlineSourced")
	if inline.ProvInline == nil || inline.ProvInline.Quote != "inline quote" {
		t.Errorf("inline provenance not captured: %+v", inline)
	}
	if inline.Conjectural || inline.Conj != nil || inline.ProvVar != "" {
		t.Errorf("inline-sourced claim must not look conjectural: %+v", inline)
	}

	conj := claimByVar(t, c, "TripCycle9FEPCommentaryOnSurprise")
	if !conj.Conjectural || conj.Conj == nil {
		t.Fatalf("conjecture not captured: %+v", conj)
	}
	if conj.ProvVar != "" || conj.ProvInline != nil {
		t.Errorf("a conjecture must carry no source attribution: %+v", conj)
	}
	if conj.Conj.GeneratedBy != "metabolism-trip" || conj.Conj.CycleN != 9 || conj.Conj.Score != 4 ||
		conj.Conj.Temperature != 1.0 || conj.Conj.PromptType != "analogy" ||
		conj.Conj.GeneratedAt != "2026-04-27" || conj.Conj.Rationale != "winze's own reasoning" {
		t.Errorf("conjecture fields wrong: %+v", conj.Conj)
	}
	if len(conj.Conj.From) != 2 || conj.Conj.From[0] != "FEP" || conj.Conj.From[1] != "Surprise" {
		t.Errorf("From should resolve to entity var names, got %v", conj.Conj.From)
	}
}

// SourceDoc writes its embedded *Entity keyed rather than positionally. Before
// the walker unwrapped the key, every such carrier vanished from the parse.
func TestKeyedEntityCarrierParses(t *testing.T) {
	c := parseFixture(t, fixtureAttribution)
	e, ok := c.EntityByVar()["KeyedCarrier"]
	if !ok {
		t.Fatal("keyed entity carrier not parsed")
	}
	if e.RoleType != "SourceDoc" || e.Name != "Corpus write lock" {
		t.Errorf("entity fields wrong: %+v", e)
	}
	if len(e.Refs) != 2 {
		t.Fatalf("expected 2 code refs, got %d", len(e.Refs))
	}
	if e.Refs[0].Path != "internal/corpuslock.Acquire" || e.Refs[0].Note != "takes the flock" {
		t.Errorf("code ref 0 wrong: %+v", e.Refs[0])
	}
	if e.Refs[1].Note != "" {
		t.Errorf("a ref with no Note should parse with an empty one: %+v", e.Refs[1])
	}
	// Ordinary role wrappers have no Refs field and must not acquire one.
	if len(c.EntityByVar()["Friston"].Refs) != 0 {
		t.Error("plain role wrapper should have no code refs")
	}
}

// Getting the actor convention backwards would inflate the trust tier of every
// exported claim, so the mapping is asserted rather than assumed.
func TestActorConvention(t *testing.T) {
	cases := map[string]string{
		"winze":                    "process:winze",
		"winze pkm-ingest":         "process:winze pkm-ingest",
		"winze metabolism cycle 6": "process:winze metabolism cycle 6",
		"winze-metabolism-trip":    "process:winze-metabolism-trip",
		"some-sensor":              "process:some-sensor",
		"Ada Lovelace":             "human:Ada Lovelace",
		"":                         "process:winze",
		"   ":                      "process:winze",
	}
	for in, want := range cases {
		if got := Actor(in); got != want {
			t.Errorf("Actor(%q) = %q, want %q", in, got, want)
		}
	}
}

// ParseCorpus must keep returning exactly what it always did now that
// ParseCorpusFull backs it.
func TestParseCorpusDelegates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(fixtureAttribution), 0o644); err != nil {
		t.Fatal(err)
	}
	ents, claims, err := ParseCorpus(dir)
	if err != nil {
		t.Fatal(err)
	}
	full, err := ParseCorpusFull(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != len(full.Entities) || len(claims) != len(full.Claims) {
		t.Errorf("ParseCorpus (%d/%d) diverges from ParseCorpusFull (%d/%d)",
			len(ents), len(claims), len(full.Entities), len(full.Claims))
	}
}
