package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// okfVersion is the spec revision this exporter targets. It is written to the
// bundle-root index.md, the only place the spec permits it.
const okfVersion = "0.2"

// manifestName marks a directory as a generated bundle. Its presence is what
// lets a re-export overwrite without --force: winze wrote it, so winze may
// replace it. A directory without the marker is someone else's, and gets a
// refusal instead of a silent clobber.
const manifestName = ".winze-okf.json"

type manifest struct {
	OKFVersion string `json:"okf_version"`
	Corpus     string `json:"corpus"`
	Concepts   int    `json:"concepts"`
	Claims     int    `json:"claims"`
	Sources    int    `json:"sources"`
}

// sectionDir maps a winze role to its bundle directory. Roles are grouped by
// directory purely for progressive disclosure — OKF puts no meaning in the
// path, so the role also survives verbatim in each document's `type`.
var sectionDir = map[string]string{
	"Person":       "people",
	"Organization": "organizations",
	"Place":        "places",
	"Event":        "events",
	"Facility":     "facilities",
	"Substance":    "substances",
	"Instrument":   "instruments",
	"Hypothesis":   "hypotheses",
	"LearningGoal": "learning-goals",
	"Concept":      "concepts",
	"SourceDoc":    "source-docs",
}

func dirForRole(role string) string {
	if d, ok := sectionDir[role]; ok {
		return d
	}
	return slug(role) + "s"
}

// doc is one emitted concept document, resolved before any markdown is written
// so links can point at final paths.
type doc struct {
	entity   corpusparse.Entity
	path     string // bundle-absolute, e.g. "/hypotheses/lake-cheko.md"
	outgoing []corpusparse.Claim
	incoming []corpusparse.Claim
}

type exporter struct {
	corpus   *corpusparse.Corpus
	provByV  map[string]corpusparse.Provenance
	docByVar map[string]*doc
	docs     []*doc
}

type summary struct {
	Out      string
	Sections map[string]int
	Concepts int
	Claims   int
	Sourced  int
	Conjectl int
	Sources  int
}

func (s summary) print(w io.Writer) {
	fmt.Fprintf(w, "OKF v%s bundle: %s\n\n", okfVersion, s.Out)
	fmt.Fprintf(w, "  %d concepts, %d claims (%d sourced, %d conjectural), %d sources\n\n",
		s.Concepts, s.Claims, s.Sourced, s.Conjectl, s.Sources)
	names := make([]string, 0, len(s.Sections))
	for k := range s.Sections {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(w, "    %-16s %d\n", n, s.Sections[n])
	}
}

func export(corpusDir, outDir string, force bool) (summary, error) {
	c, err := corpusparse.ParseCorpusFull(corpusDir)
	if err != nil {
		return summary{}, err
	}
	if len(c.Entities) == 0 {
		return summary{}, fmt.Errorf("no entities parsed from %s — not a winze corpus?", corpusDir)
	}

	e := &exporter{corpus: c, provByV: c.ProvenanceByVar(), docByVar: map[string]*doc{}}
	e.plan()

	if err := prepareOutDir(outDir, force); err != nil {
		return summary{}, err
	}
	if err := e.write(outDir); err != nil {
		return summary{}, err
	}

	s := summary{Out: outDir, Sections: map[string]int{}, Concepts: len(e.docs), Sources: len(c.Provenance)}
	for _, d := range e.docs {
		s.Sections[path.Dir(d.path)[1:]]++
	}
	for _, cl := range c.Claims {
		if _, ok := e.docByVar[cl.SubjectVar]; !ok {
			continue
		}
		s.Claims++
		if cl.Conjectural {
			s.Conjectl++
		} else {
			s.Sourced++
		}
	}

	m := manifest{OKFVersion: okfVersion, Corpus: corpusDir, Concepts: s.Concepts, Claims: s.Claims, Sources: s.Sources}
	blob, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return summary{}, err
	}
	if err := os.WriteFile(filepath.Join(outDir, manifestName), append(blob, '\n'), 0o644); err != nil {
		return summary{}, err
	}
	return s, nil
}

// plan resolves every entity to a bundle path and buckets claims onto their
// subject and object documents. Paths are assigned in sorted var order so a
// slug collision resolves the same way on every run.
func (e *exporter) plan() {
	ents := append([]corpusparse.Entity(nil), e.corpus.Entities...)
	sort.Slice(ents, func(i, j int) bool { return ents[i].VarName < ents[j].VarName })

	taken := map[string]bool{}
	for _, ent := range ents {
		dir := dirForRole(ent.RoleType)
		base := slug(firstNonEmpty(ent.ID, ent.Name, ent.VarName))
		if base == "" {
			base = slug(ent.VarName)
		}
		p := "/" + dir + "/" + base + ".md"
		for n := 2; taken[p]; n++ {
			p = fmt.Sprintf("/%s/%s-%d.md", dir, base, n)
		}
		taken[p] = true
		d := &doc{entity: ent, path: p}
		e.docs = append(e.docs, d)
		e.docByVar[ent.VarName] = d
	}

	claims := append([]corpusparse.Claim(nil), e.corpus.Claims...)
	sort.Slice(claims, func(i, j int) bool { return claims[i].VarName < claims[j].VarName })
	for _, cl := range claims {
		if d, ok := e.docByVar[cl.SubjectVar]; ok {
			d.outgoing = append(d.outgoing, cl)
		}
		if cl.ObjectVar == "" || cl.ObjectVar == cl.SubjectVar {
			continue
		}
		if d, ok := e.docByVar[cl.ObjectVar]; ok {
			d.incoming = append(d.incoming, cl)
		}
	}
}

func (e *exporter) write(outDir string) error {
	bySection := map[string][]*doc{}
	for _, d := range e.docs {
		body, err := e.renderDoc(d)
		if err != nil {
			return fmt.Errorf("%s: %w", d.entity.VarName, err)
		}
		full := filepath.Join(outDir, filepath.FromSlash(strings.TrimPrefix(d.path, "/")))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			return err
		}
		bySection[path.Dir(d.path)[1:]] = append(bySection[path.Dir(d.path)[1:]], d)
	}

	for section, docs := range bySection {
		idx := renderSectionIndex(section, docs)
		if err := os.WriteFile(filepath.Join(outDir, section, "index.md"), []byte(idx), 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(outDir, "index.md"), []byte(renderRootIndex(bySection)), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "log.md"), []byte(e.renderLog()), 0o644)
}

// --- concept documents ---

// renderDoc emits one concept document. The two halves of the Attribution sum
// are kept apart throughout: sourced claims accumulate into `srcIDs` (which
// becomes `sources:` and the footnote definitions) while conjectural claims are
// routed to their own section and never touch it.
func (e *exporter) renderDoc(d *doc) (string, error) {
	var sourced, conjectural []corpusparse.Claim
	for _, cl := range d.outgoing {
		if cl.Conjectural {
			conjectural = append(conjectural, cl)
		} else {
			sourced = append(sourced, cl)
		}
	}

	// Resolve the provenance backing each sourced claim, keyed by footnote id.
	provs := map[string]corpusparse.Provenance{}
	footnote := map[string]string{} // claim var -> source id
	for _, cl := range sourced {
		p, id, ok := e.provFor(cl)
		if !ok {
			continue
		}
		provs[id] = p
		footnote[cl.VarName] = id
	}
	ids := sortedKeys(provs)

	fm := yaml.MapSlice{
		{Key: "type", Value: d.entity.RoleType},
		{Key: "title", Value: firstNonEmpty(d.entity.Name, d.entity.VarName)},
	}
	if desc := firstSentence(d.entity.Brief); desc != "" {
		fm = append(fm, yaml.MapItem{Key: "description", Value: desc})
	}
	if tags := tagsFor(d.entity); len(tags) > 0 {
		fm = append(fm, yaml.MapItem{Key: "tags", Value: tags})
	}

	// Trust tiers. A document with at least one Quote-bearing sourced claim has
	// been through the build gate against a real source, which is OKF's
	// machine-confirmed tier. One backed only by conjecture gets no `verified`
	// entry at all — unverified, and marked draft — because that is what it is.
	status := "stable"
	if len(ids) == 0 && len(conjectural) > 0 {
		status = "draft"
	}
	fm = append(fm, yaml.MapItem{Key: "status", Value: status})

	if len(ids) > 0 {
		var srcs []yaml.MapSlice
		for _, id := range ids {
			srcs = append(srcs, sourceEntry(id, provs[id]))
		}
		fm = append(fm, yaml.MapItem{Key: "sources", Value: srcs})
	}
	if g := generatedEntry(provs, conjectural); g != nil {
		fm = append(fm, yaml.MapItem{Key: "generated", Value: g})
	}
	if v := verifiedEntries(provs); len(v) > 0 {
		fm = append(fm, yaml.MapItem{Key: "verified", Value: v})
	}
	// Extra keys are conformant and consumers must tolerate them. The corpus var
	// name is the join key back to the Go source a claim actually lives in.
	fm = append(fm, yaml.MapItem{Key: "winze_var", Value: d.entity.VarName})
	fm = append(fm, yaml.MapItem{Key: "winze_file", Value: d.entity.File})

	head, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(head)
	b.WriteString("---\n\n")
	b.WriteString("# " + firstNonEmpty(d.entity.Name, d.entity.VarName) + "\n\n")
	if d.entity.Brief != "" {
		b.WriteString(d.entity.Brief + "\n\n")
	}
	if len(d.entity.Aliases) > 0 {
		b.WriteString("Also known as: " + strings.Join(d.entity.Aliases, ", ") + "\n\n")
	}

	writeCodeRefs(&b, d.entity.Refs)
	e.writeClaims(&b, sourced, footnote)
	e.writeConjectures(&b, conjectural)
	e.writeReferencedBy(&b, d.incoming)

	if len(ids) > 0 {
		b.WriteString("# Sources\n\n")
		for _, id := range ids {
			p := provs[id]
			// The Quote is quoted inline rather than blockquoted: a `>` on the
			// footnote-definition line renders as a literal angle bracket, not
			// a quotation. The continuation line is indented so it stays part
			// of the same footnote.
			line := "[^" + id + "]: "
			if p.Quote != "" {
				line += "“" + collapse(p.Quote) + "”\n    "
			}
			line += collapse(p.Origin)
			if p.IngestedAt != "" {
				line += fmt.Sprintf(" (ingested %s by %s)", p.IngestedAt, corpusparse.Actor(p.IngestedBy))
			}
			b.WriteString(line + "\n\n")
		}
	}
	return b.String(), nil
}

// writeCodeRefs emits a SourceDoc's typed code citations. In the corpus these
// are compile-checked by value, so a renamed symbol breaks the build; in a
// markdown bundle they are prose again and that guarantee does not travel. The
// section says so rather than presenting the reference as if it still held —
// exporting a guarantee you cannot keep is how a bundle starts lying.
func writeCodeRefs(b *strings.Builder, refs []corpusparse.CodeRef) {
	if len(refs) == 0 {
		return
	}
	b.WriteString("# Code references\n\n")
	b.WriteString("Typed citations to live code. In the corpus these are checked by the compiler; in this bundle they are labels, and can go stale without anything noticing.\n\n")
	for _, r := range refs {
		line := "* `" + r.Path + "`"
		if r.Note != "" {
			line += " — " + collapse(r.Note)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
}

// writeClaims emits sourced claims grouped by predicate. The heading IS the
// edge type — the piece OKF's bare markdown links drop.
func (e *exporter) writeClaims(b *strings.Builder, claims []corpusparse.Claim, footnote map[string]string) {
	if len(claims) == 0 {
		return
	}
	b.WriteString("# Claims\n\n")
	for _, pred := range groupByPredicate(claims) {
		b.WriteString("## " + pred.name + "\n\n")
		for _, cl := range pred.claims {
			mark := ""
			if id, ok := footnote[cl.VarName]; ok {
				mark = "[^" + id + "]"
			}
			b.WriteString("* " + e.linkOrLiteral(cl) + mark + "\n")
		}
		b.WriteString("\n")
	}
}

// writeConjectures emits winze's own generations. Separate section, no
// footnote markers, and nothing here ever reached the `sources:` list — the
// export-side expression of the guarantee schema.go makes with a missing field.
func (e *exporter) writeConjectures(b *strings.Builder, claims []corpusparse.Claim) {
	if len(claims) == 0 {
		return
	}
	b.WriteString("# Conjectures\n\n")
	b.WriteString("Generated by winze, not drawn from a source. These carry no source attribution by construction.\n\n")
	for _, pred := range groupByPredicate(claims) {
		b.WriteString("## " + pred.name + "\n\n")
		for _, cl := range pred.claims {
			b.WriteString("* " + e.linkOrLiteral(cl))
			if cl.Conj != nil {
				b.WriteString(" — " + conjectureLabel(cl.Conj))
			}
			b.WriteString("\n")
			if cl.Conj != nil && cl.Conj.Rationale != "" {
				b.WriteString("\n  > " + collapse(cl.Conj.Rationale) + "\n")
			}
		}
		b.WriteString("\n")
	}
}

func (e *exporter) writeReferencedBy(b *strings.Builder, claims []corpusparse.Claim) {
	if len(claims) == 0 {
		return
	}
	b.WriteString("# Referenced by\n\n")
	for _, pred := range groupByPredicate(claims) {
		b.WriteString("## " + pred.name + "\n\n")
		for _, cl := range pred.claims {
			d, ok := e.docByVar[cl.SubjectVar]
			if !ok {
				continue
			}
			suffix := ""
			if cl.Conjectural {
				suffix = " *(conjecture)*"
			}
			b.WriteString("* " + link(d) + suffix + "\n")
		}
		b.WriteString("\n")
	}
}

// linkOrLiteral renders a claim's object as a link when it resolves to a
// document, and the predicate name alone for unary claims (which have no
// object at all — the claim IS the tag).
func (e *exporter) linkOrLiteral(cl corpusparse.Claim) string {
	if cl.ObjectVar == "" {
		return "**" + spaceCamel(cl.PredicateType) + "**"
	}
	if d, ok := e.docByVar[cl.ObjectVar]; ok {
		return link(d)
	}
	return "`" + cl.ObjectVar + "`"
}

func link(d *doc) string {
	return "[" + firstNonEmpty(d.entity.Name, d.entity.VarName) + "](" + d.path + ")"
}

// provFor resolves a sourced claim's backing provenance and the footnote id it
// should carry. A shared provenance var keys on its own name so every document
// citing it agrees; an inline literal keys on the claim.
func (e *exporter) provFor(cl corpusparse.Claim) (corpusparse.Provenance, string, bool) {
	if cl.ProvInline != nil {
		return *cl.ProvInline, slug(cl.VarName) + "-source", true
	}
	if cl.ProvVar == "" {
		return corpusparse.Provenance{}, "", false
	}
	p, ok := e.provByV[cl.ProvVar]
	if !ok {
		return corpusparse.Provenance{}, "", false
	}
	return p, cl.ProvVar, true
}

// --- frontmatter families ---

func sourceEntry(id string, p corpusparse.Provenance) yaml.MapSlice {
	entry := yaml.MapSlice{
		{Key: "id", Value: id},
		{Key: "resource", Value: resourceURI(id, p)},
	}
	if p.Origin != "" {
		entry = append(entry, yaml.MapItem{Key: "title", Value: collapse(p.Origin)})
	}
	if p.IngestedAt != "" {
		entry = append(entry, yaml.MapItem{Key: "last_modified", Value: p.IngestedAt})
	}
	return entry
}

var urlRE = regexp.MustCompile(`https?://[^\s,)]+`)

// resourceURI gives OKF the URI it wants for a source. Winze's Origin is a
// free-form human hint that is never required to resolve — the Quote is the
// audit record — so a URL is used when the Origin happens to contain one, and
// otherwise the id becomes a `winze:` scope descriptor, which the spec permits
// alongside URLs and bundle paths. Inventing a URL here would be exactly the
// fabrication the type system exists to prevent.
func resourceURI(id string, p corpusparse.Provenance) string {
	if u := urlRE.FindString(p.Origin); u != "" {
		return strings.TrimRight(u, ".,;")
	}
	return "winze:source/" + id
}

// generatedEntry records who produced the document's content. Sourced material
// is attributed to its ingester, conjectural material to the generating winze
// process; both are processes today, and Actor keeps that honest.
func generatedEntry(provs map[string]corpusparse.Provenance, conj []corpusparse.Claim) yaml.MapSlice {
	by, at := "", ""
	for _, id := range sortedKeys(provs) {
		p := provs[id]
		if p.IngestedBy != "" && (at == "" || p.IngestedAt > at) {
			by, at = corpusparse.Actor(p.IngestedBy), p.IngestedAt
		}
	}
	if by == "" {
		for _, cl := range conj {
			if cl.Conj == nil {
				continue
			}
			agent := firstNonEmpty(cl.Conj.GeneratedByAgent, cl.Conj.GeneratedBy)
			if agent != "" && (at == "" || cl.Conj.GeneratedAt > at) {
				by, at = corpusparse.Actor(agent), cl.Conj.GeneratedAt
			}
		}
	}
	if by == "" {
		return nil
	}
	entry := yaml.MapSlice{{Key: "by", Value: by}}
	if at != "" {
		entry = append(entry, yaml.MapItem{Key: "at", Value: at})
	}
	return entry
}

// verifiedEntries emits one machine-verification event per ingest date on which
// this document acquired a sourced claim. The actor is the build gate rather
// than the ingester: what was verified is that a Quote-bearing provenance
// type-checked and passed lint, which is a machine check, not a human review.
// Emitting `human:` here would silently promote the whole corpus a trust tier.
func verifiedEntries(provs map[string]corpusparse.Provenance) []yaml.MapSlice {
	dates := map[string]bool{}
	for _, p := range provs {
		if p.Quote != "" && p.IngestedAt != "" {
			dates[p.IngestedAt] = true
		}
	}
	out := make([]yaml.MapSlice, 0, len(dates))
	for _, at := range sortedBoolKeys(dates) {
		out = append(out, yaml.MapSlice{
			{Key: "by", Value: "process:winze-build-gate"},
			{Key: "at", Value: at},
		})
	}
	return out
}

// --- index and log ---

func renderRootIndex(bySection map[string][]*doc) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("okf_version: \"" + okfVersion + "\"\n")
	b.WriteString("---\n\n")
	b.WriteString("# winze\n\n")
	b.WriteString("A typed epistemic corpus on how minds — human and artificial — build, validate, and fail at modeling reality, projected into OKF v" + okfVersion + ".\n\n")
	b.WriteString("Claims appear under `# Claims` grouped by predicate: the heading is the relationship type. Generated content is quarantined under `# Conjectures` and never carries a source.\n\n")
	b.WriteString("## Sections\n\n")
	for _, section := range sortedDocKeys(bySection) {
		b.WriteString(fmt.Sprintf("* [%s](/%s/index.md) - %d concepts\n", titleize(section), section, len(bySection[section])))
	}
	b.WriteString("\n* [Log](/log.md) - chronological ingest and generation history\n")
	return b.String()
}

func renderSectionIndex(section string, docs []*doc) string {
	ordered := append([]*doc(nil), docs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].path < ordered[j].path })

	var b strings.Builder
	b.WriteString("# " + titleize(section) + "\n\n")
	for _, d := range ordered {
		line := fmt.Sprintf("* [%s](%s)", firstNonEmpty(d.entity.Name, d.entity.VarName), d.path)
		if desc := firstSentence(d.entity.Brief); desc != "" {
			line += " - " + desc
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// renderLog reconstructs the bundle's history from what the corpus actually
// records: provenance ingest dates and conjecture generation dates. Newest
// first, ISO 8601 headings, per the reserved-filename structure.
func (e *exporter) renderLog() string {
	type event struct{ kind, text string }
	byDate := map[string][]event{}

	for _, p := range e.corpus.Provenance {
		if p.IngestedAt == "" {
			continue
		}
		byDate[p.IngestedAt] = append(byDate[p.IngestedAt], event{"Creation",
			fmt.Sprintf("ingested `%s` — %s", p.VarName, collapse(truncate(p.Origin, 160)))})
	}
	generated := map[string]map[string]int{}
	for _, cl := range e.corpus.Claims {
		if cl.Conj == nil || cl.Conj.GeneratedAt == "" {
			continue
		}
		label := firstNonEmpty(cl.Conj.GeneratedBy, "winze")
		if cl.Conj.CycleN > 0 {
			label = fmt.Sprintf("%s (cycle %d)", label, cl.Conj.CycleN)
		}
		if generated[cl.Conj.GeneratedAt] == nil {
			generated[cl.Conj.GeneratedAt] = map[string]int{}
		}
		generated[cl.Conj.GeneratedAt][label]++
	}
	for date, labels := range generated {
		for _, label := range sortedIntKeys(labels) {
			byDate[date] = append(byDate[date], event{"Update",
				fmt.Sprintf("%d conjectural claim(s) generated by %s", labels[label], label)})
		}
	}

	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	var b strings.Builder
	b.WriteString("# Log\n\n")
	for _, date := range dates {
		evs := byDate[date]
		sort.Slice(evs, func(i, j int) bool {
			if evs[i].kind != evs[j].kind {
				return evs[i].kind < evs[j].kind
			}
			return evs[i].text < evs[j].text
		})
		b.WriteString("## " + date + "\n\n")
		for _, ev := range evs {
			b.WriteString(fmt.Sprintf("* **%s**: %s\n", ev.kind, ev.text))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// --- output directory ---

func prepareOutDir(outDir string, force bool) error {
	info, err := os.Stat(outDir)
	if os.IsNotExist(err) {
		return os.MkdirAll(outDir, 0o755)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s exists and is not a directory", outDir)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if _, err := os.Stat(filepath.Join(outDir, manifestName)); err != nil && !force {
		return fmt.Errorf("%s is not empty and carries no %s marker; pass --force to overwrite", outDir, manifestName)
	}
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	return os.MkdirAll(outDir, 0o755)
}
