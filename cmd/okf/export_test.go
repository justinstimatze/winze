package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture is a miniature corpus exercising both halves of the Attribution sum,
// a unary claim, an alias, and a keyed SourceDoc carrier.
const fixture = `package winze

var tunguskaSource = Provenance{
	Origin:     "Wikipedia 2025-12 / Tunguska_event https://en.wikipedia.org/wiki/Tunguska_event",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze",
	Quote:      "The Tunguska event was a large explosion of between 3 and 50 megatons.",
}

var TunguskaEvent = Event{&Entity{
	ID:      "tunguska-event",
	Name:    "Tunguska event",
	Kind:    "event",
	Brief:   "A large explosion over Siberia in 1908. Widely attributed to an air burst.",
	Aliases: []string{"Tunguska blast"},
}}

var LakeCheko = Place{&Entity{
	ID:    "lake-cheko",
	Name:  "Lake Cheko",
	Kind:  "place",
	Brief: "A lake near the Tunguska epicentre.",
}}

var SurvivorshipBias = Concept{&Entity{
	ID:    "survivorship-bias",
	Name:  "Survivorship bias",
	Kind:  "concept",
	Brief: "Reasoning from the cases that made it through.",
}}

var ChekoNearTunguska = LocatedIn{
	Subject: LakeCheko,
	Object:  TunguskaEvent,
	Prov:    tunguskaSource,
}

var BiasIsBias = IsCognitiveBias{
	Subject: SurvivorshipBias,
	Prov:    tunguskaSource,
}

var TripCycle25BiasCommentaryOnTunguska = CommentaryOn{
	Subject: SurvivorshipBias,
	Object:  TunguskaEvent,
	Prov: Conjecture{
		GeneratedBy:      "metabolism-trip",
		From:             []*Entity{SurvivorshipBias.Entity, TunguskaEvent.Entity},
		CycleN:           25,
		Temperature:      1.0,
		PromptType:       "analogy",
		Score:            4,
		Rationale:        "Both concern how a record of an event survives the event.",
		GeneratedAt:      "2026-04-27",
		GeneratedByAgent: "winze-metabolism-trip",
	},
}

var CorpusLockDoc = SourceDoc{
	Entity: &Entity{
		ID:    "doc-corpuslock",
		Name:  "Corpus write lock",
		Kind:  "sourcedoc",
		Brief: "Write paths take a corpus-wide advisory flock.",
	},
	Refs: []CodeRef{{
		Symbol: corpuslock.Acquire,
		Path:   "internal/corpuslock.Acquire",
		Note:   "acquires the .winze.lock flock",
	}},
}
`

func exportFixture(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "corpus.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "bundle")
	if _, err := export(src, out, false); err != nil {
		t.Fatalf("export: %v", err)
	}
	return out
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// The load-bearing invariant. schema.go makes a fabricated source
// unrepresentable by giving Conjecture no Quote field; the exporter has to
// carry that guarantee across the projection, because a bundle that listed a
// generated claim's "source" would reintroduce exactly the failure mode the
// type system closed.
func TestConjectureNeverReachesSources(t *testing.T) {
	out := exportFixture(t)
	doc := read(t, out, "/concepts/survivorship-bias.md")

	fm, body, ok := splitFrontmatter(doc)
	if !ok {
		t.Fatal("no frontmatter")
	}
	if strings.Contains(fm, "sources:") {
		// The only claim on this entity with an object is conjectural; the
		// unary claim's provenance is legitimate, so a sources list here must
		// contain nothing traceable to the conjecture.
		if strings.Contains(fm, "metabolism-trip") || strings.Contains(fm, "Both concern how") {
			t.Errorf("conjecture leaked into sources:\n%s", fm)
		}
	}
	if !strings.Contains(body, "# Conjectures") {
		t.Error("conjectural claim not emitted under # Conjectures")
	}
	// The conjecture's link must carry no footnote marker: footnote ids are
	// source ids, and a conjecture has no source.
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "Tunguska event](/events/") && strings.Contains(line, "metabolism-trip") {
			if strings.Contains(line, "[^") {
				t.Errorf("conjectural claim carries a source footnote: %s", line)
			}
		}
	}
}

// A document backed only by conjecture is unverified and draft; one with a
// Quote-bearing sourced claim is machine-confirmed. Getting this backwards
// would inflate the trust tier of everything winze generates.
func TestTrustTiers(t *testing.T) {
	out := exportFixture(t)

	sourced, _, _ := splitFrontmatter(read(t, out, "/places/lake-cheko.md"))
	if !strings.Contains(sourced, "status: stable") {
		t.Errorf("sourced doc should be stable:\n%s", sourced)
	}
	if !strings.Contains(sourced, "by: process:winze-build-gate") {
		t.Errorf("sourced doc should carry a machine verification event:\n%s", sourced)
	}
	if !strings.Contains(sourced, "id: tunguskaSource") {
		t.Errorf("sourced doc should list its provenance var as a source id:\n%s", sourced)
	}
	// The Origin carried a real URL, so resource must be that URL rather than
	// a synthesized one.
	if !strings.Contains(sourced, "https://en.wikipedia.org/wiki/Tunguska_event") {
		t.Errorf("resource should use the URL present in Origin:\n%s", sourced)
	}
	// Actors are processes, never humans — the corpus is machine-ingested.
	if strings.Contains(sourced, "human:") {
		t.Errorf("winze ingest must not be attributed to a human actor:\n%s", sourced)
	}
}

// Predicate headings are the whole reason the export is more than a folder of
// notes: OKF links carry no relationship kind, so the edge type has to live in
// the structure around them.
func TestPredicateHeadingsPreserveEdgeTypes(t *testing.T) {
	out := exportFixture(t)
	doc := read(t, out, "/places/lake-cheko.md")
	if !strings.Contains(doc, "## Located In") {
		t.Errorf("predicate heading missing:\n%s", doc)
	}
	if !strings.Contains(doc, "](/events/tunguska-event.md)") {
		t.Errorf("claim object not linked to its document:\n%s", doc)
	}

	// Unary claims have no object; the predicate itself is the statement.
	concept := read(t, out, "/concepts/survivorship-bias.md")
	if !strings.Contains(concept, "**Is Cognitive Bias**") {
		t.Errorf("unary claim not rendered as a bare predicate:\n%s", concept)
	}

	// The inverse edge appears on the object's document.
	event := read(t, out, "/events/tunguska-event.md")
	if !strings.Contains(event, "# Referenced by") || !strings.Contains(event, "](/places/lake-cheko.md)") {
		t.Errorf("inverse edge missing from object document:\n%s", event)
	}
}

// A keyed carrier (SourceDoc{Entity: ..., Refs: ...}) must parse, and its typed
// code citations must be labelled as no longer compiler-checked once exported.
func TestSourceDocAndCodeRefs(t *testing.T) {
	out := exportFixture(t)
	doc := read(t, out, "/source-docs/doc-corpuslock.md")
	if !strings.Contains(doc, "# Code references") {
		t.Errorf("code references section missing:\n%s", doc)
	}
	if !strings.Contains(doc, "internal/corpuslock.Acquire") {
		t.Errorf("code ref path missing:\n%s", doc)
	}
	if !strings.Contains(doc, "can go stale") {
		t.Errorf("exported code refs must disclose that the compile-time guarantee does not travel:\n%s", doc)
	}
}

func TestBundleStructure(t *testing.T) {
	out := exportFixture(t)

	root := read(t, out, "/index.md")
	if !strings.Contains(root, `okf_version: "`+okfVersion+`"`) {
		t.Errorf("root index must declare okf_version:\n%s", root)
	}
	if !strings.Contains(root, "(/events/index.md)") {
		t.Errorf("root index must link section indexes:\n%s", root)
	}

	section := read(t, out, "/events/index.md")
	if !strings.Contains(section, "* [Tunguska event](/events/tunguska-event.md) - ") {
		t.Errorf("section index entry malformed:\n%s", section)
	}

	log := read(t, out, "/log.md")
	if !strings.Contains(log, "## 2026-04-27") || !strings.Contains(log, "## 2026-04-13") {
		t.Errorf("log must group by ingest and generation dates:\n%s", log)
	}
	if strings.Index(log, "## 2026-04-27") > strings.Index(log, "## 2026-04-13") {
		t.Errorf("log must be newest-first:\n%s", log)
	}

	// Aliases survive as tags and as prose.
	event := read(t, out, "/events/tunguska-event.md")
	if !strings.Contains(event, "tunguska-blast") {
		t.Errorf("aliases should become tags:\n%s", event)
	}
	// description is one sentence; the full Brief stays in the body.
	if !strings.Contains(event, "description: A large explosion over Siberia in 1908.") {
		t.Errorf("description should be the first sentence only:\n%s", event)
	}
	if !strings.Contains(event, "Widely attributed to an air burst.") {
		t.Errorf("full brief should remain in the body:\n%s", event)
	}
}

// Re-exporting an unchanged corpus must produce a byte-identical bundle, so a
// committed bundle diffs against the corpus change that caused it and nothing
// else.
func TestExportIsDeterministic(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "corpus.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "bundle")
	for i := 0; i < 2; i++ {
		if _, err := export(src, out, false); err != nil {
			t.Fatalf("export %d: %v", i, err)
		}
	}
	first := snapshot(t, out)

	out2 := filepath.Join(t.TempDir(), "bundle")
	if _, err := export(src, out2, false); err != nil {
		t.Fatal(err)
	}
	second := snapshot(t, out2)

	if len(first) != len(second) {
		t.Fatalf("file count differs: %d vs %d", len(first), len(second))
	}
	for rel, body := range first {
		if second[rel] != body {
			t.Errorf("%s differs between runs", rel)
		}
	}
}

func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A directory winze did not generate is never clobbered without --force.
func TestRefusesToOverwriteForeignDirectory(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "corpus.go"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	keep := filepath.Join(out, "important.txt")
	if err := os.WriteFile(keep, []byte("do not delete"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := export(src, out, false); err == nil {
		t.Fatal("expected refusal on a non-empty foreign directory")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("refused export must not touch existing files: %v", err)
	}
	if _, err := export(src, out, true); err != nil {
		t.Fatalf("--force should overwrite: %v", err)
	}
	// A regenerated bundle carries the marker, so the next export needs no flag.
	if _, err := export(src, out, false); err != nil {
		t.Fatalf("re-export of a generated bundle should not need --force: %v", err)
	}
}

func TestExportRejectsNonCorpus(t *testing.T) {
	if _, err := export(t.TempDir(), filepath.Join(t.TempDir(), "b"), false); err == nil {
		t.Fatal("expected an error exporting a directory with no entities")
	}
}
