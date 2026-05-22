package main

import (
	"os"
	"path/filepath"
	"testing"
)

const fixtureEntities = `package winze

var (
	Apophenia = Concept{&Entity{
		ID:    "concept-apophenia",
		Name:  "Apophenia",
		Kind:  "concept",
		Aliases: []string{"apophany", "patternicity"},
		Brief: "The cognitive tendency to perceive meaningful patterns in random data.",
	}}

	KlausConrad = Person{&Entity{
		ID:    "klaus-conrad",
		Name:  "Klaus Conrad",
		Kind:  "person",
		Brief: "German psychiatrist who coined the term apophänie.",
	}}
)
`

const fixtureClaims = `package winze

var (
	ConradFraming = Hypothesis{&Entity{
		ID: "h", Name: "Conrad framing", Kind: "hypothesis", Brief: "Clinical framing.",
	}}

	ConradProposesFraming = Proposes{
		Subject: KlausConrad,
		Object:  ConradFraming,
		Prov:    apopheniaSource,
	}

	ConradTheoryOf = TheoryOf{
		Subject: ConradFraming,
		Object:  Apophenia,
		Prov:    apopheniaSource,
	}

	ApopheniaTag = IsCognitiveBias{
		Subject: Apophenia,
		Prov:    apopheniaSource,
	}
)
`

// shouldnotparse is shaped like an entity declaration but lacks Brief, so
// parseCorpus should silently skip it (the prov skip-rule rejects entity
// vars with missing core fields).
const fixtureNoise = `package winze

var (
	apopheniaSource = Provenance{Origin: "test", Quote: "q", IngestedAt: "2026-05-21", IngestedBy: "test"}

	NoBrief = Concept{&Entity{ID: "x", Name: "X", Kind: "concept"}}
)
`

func writeFixture(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return dir
}

func writeFixtures(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestParseEntities(t *testing.T) {
	dir := writeFixture(t, "ents.go", fixtureEntities)
	ents, _, err := parseCorpus(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ents) != 2 {
		t.Fatalf("want 2 entities, got %d: %+v", len(ents), ents)
	}
	byName := map[string]entity{}
	for _, e := range ents {
		byName[e.varName] = e
	}
	apo, ok := byName["Apophenia"]
	if !ok {
		t.Fatal("Apophenia missing")
	}
	if apo.roleType != "Concept" {
		t.Errorf("Apophenia role: %s", apo.roleType)
	}
	if apo.brief == "" {
		t.Error("Apophenia brief missing")
	}
	if len(apo.aliases) != 2 {
		t.Errorf("Apophenia aliases: %v", apo.aliases)
	}
}

func TestParseClaims(t *testing.T) {
	dir := writeFixtures(t, map[string]string{
		"ents.go":   fixtureEntities,
		"claims.go": fixtureClaims,
		"noise.go":  fixtureNoise,
	})
	ents, claims, err := parseCorpus(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ents) < 3 {
		t.Errorf("want >=3 entities, got %d", len(ents))
	}
	if len(claims) != 3 {
		t.Fatalf("want 3 claims, got %d: %+v", len(claims), claims)
	}

	byName := map[string]claim{}
	for _, c := range claims {
		byName[c.varName] = c
	}
	prop := byName["ConradProposesFraming"]
	if prop.predicateType != "Proposes" {
		t.Errorf("Proposes predicate: %s", prop.predicateType)
	}
	if prop.subjectVar != "KlausConrad" || prop.objectVar != "ConradFraming" {
		t.Errorf("Proposes subj/obj: %s / %s", prop.subjectVar, prop.objectVar)
	}

	unary := byName["ApopheniaTag"]
	if unary.subjectVar != "Apophenia" {
		t.Errorf("unary subject: %s", unary.subjectVar)
	}
	if unary.objectVar != "" {
		t.Errorf("unary should have empty object, got %s", unary.objectVar)
	}
}

func TestNeighborhoodAssembly(t *testing.T) {
	dir := writeFixtures(t, map[string]string{
		"ents.go":   fixtureEntities,
		"claims.go": fixtureClaims,
		"noise.go":  fixtureNoise,
	})
	ents, claims, err := parseCorpus(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	hoods := buildNeighborhoods(ents, claims)

	var apoHood *neighborhood
	for i := range hoods {
		if hoods[i].ent.varName == "Apophenia" {
			apoHood = &hoods[i]
			break
		}
	}
	if apoHood == nil {
		t.Fatal("Apophenia neighborhood missing")
	}
	// Apophenia appears as object of TheoryOf and subject of IsCognitiveBias.
	if len(apoHood.asObj) != 1 {
		t.Errorf("Apophenia as object: want 1 (TheoryOf), got %d", len(apoHood.asObj))
	}
	if len(apoHood.asSubj) != 1 {
		t.Errorf("Apophenia as subject: want 1 (IsCognitiveBias), got %d", len(apoHood.asSubj))
	}
}

func TestFilterConnected(t *testing.T) {
	hoods := []neighborhood{
		{ent: entity{varName: "lonely"}},
		{ent: entity{varName: "connected"}, asSubj: []claim{{varName: "c1"}}},
	}
	got := filterConnected(hoods)
	if len(got) != 1 || got[0].ent.varName != "connected" {
		t.Errorf("filterConnected: %+v", got)
	}
}

func TestSampleDeterministic(t *testing.T) {
	hoods := make([]neighborhood, 100)
	for i := range hoods {
		hoods[i] = neighborhood{
			ent:    entity{varName: stringN(i)},
			asSubj: []claim{{varName: "c"}},
		}
	}
	a := sample(hoods, 5, 42)
	b := sample(hoods, 5, 42)
	if len(a) != 5 || len(b) != 5 {
		t.Fatalf("sample size: %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ent.varName != b[i].ent.varName {
			t.Errorf("same seed gave different samples at %d: %s vs %s", i, a[i].ent.varName, b[i].ent.varName)
		}
	}
}

func TestSampleAllWhenNGEN(t *testing.T) {
	hoods := []neighborhood{
		{ent: entity{varName: "a"}, asSubj: []claim{{varName: "c"}}},
		{ent: entity{varName: "b"}, asSubj: []claim{{varName: "c"}}},
	}
	got := sample(hoods, 100, 1)
	if len(got) != 2 {
		t.Errorf("want all 2 returned, got %d", len(got))
	}
}

func stringN(i int) string {
	return string(rune('a'+(i%26))) + string(rune('0'+(i/26)))
}
