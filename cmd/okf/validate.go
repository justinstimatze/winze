package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// The OKF v0.2 conformance floor, verbatim from the spec: every non-reserved
// .md file parses its frontmatter, every frontmatter block carries a non-empty
// `type`, and reserved filenames follow their prescribed structure. Failing any
// of these makes a bundle non-conformant.
//
// The spec also tells CONSUMERS what they must tolerate — broken cross-links,
// unknown types, missing index files. That is a rule about reading, not about
// writing, and winze declines the license: a producer that emits a broken link
// has a bug, and "the consumer must tolerate it" is not a reason to ship it.
// Those checks are therefore reported as warnings and, because this bundle is
// generated from a corpus the compiler already keeps referentially honest, any
// warning here is a defect in the exporter rather than a fact about the corpus.
type report struct {
	Root     string
	Files    int
	Concepts int
	Errors   []string
	Warnings []string
}

func (r *report) ok() bool { return len(r.Errors) == 0 }

func (r *report) errf(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *report) warnf(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *report) print(w io.Writer) {
	fmt.Fprintf(w, "OKF v%s conformance: %s\n\n", okfVersion, r.Root)
	fmt.Fprintf(w, "  %d markdown files, %d concept documents\n", r.Files, r.Concepts)
	for _, e := range r.Errors {
		fmt.Fprintf(w, "  ERROR   %s\n", e)
	}
	for _, warn := range r.Warnings {
		fmt.Fprintf(w, "  WARN    %s\n", warn)
	}
	fmt.Fprintln(w)
	if r.ok() {
		fmt.Fprintf(w, "  conformant (%d warnings)\n", len(r.Warnings))
	} else {
		fmt.Fprintf(w, "  NOT conformant: %d errors, %d warnings\n", len(r.Errors), len(r.Warnings))
	}
}

var isoDateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// linkRE matches inline markdown links. Footnote references ([^id]) are not
// links and are excluded by requiring the label not to begin with '^'.
var linkRE = regexp.MustCompile(`\[([^\^\]][^\]]*)\]\(([^)]+)\)`)

func validateBundle(root string) (*report, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	r := &report{Root: root}

	present := map[string]bool{}
	var files []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = "/" + filepath.ToSlash(rel)
		present[rel] = true
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	r.Files = len(files)

	for _, rel := range files {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(rel, "/"))))
		if err != nil {
			return nil, err
		}
		text := string(body)
		switch path.Base(rel) {
		case "index.md":
			validateIndex(r, rel, text)
		case "log.md":
			validateLog(r, rel, text)
		default:
			validateConcept(r, rel, text)
		}
		validateLinks(r, rel, text, present)
	}

	if !present["/index.md"] {
		r.warnf("bundle root has no index.md (optional per spec, but it is the entry point for progressive disclosure)")
	}
	return r, nil
}

// validateConcept enforces the two hard requirements on a concept document:
// parseable frontmatter, and a non-empty `type` inside it.
func validateConcept(r *report, rel, text string) {
	r.Concepts++
	fm, _, ok := splitFrontmatter(text)
	if !ok {
		r.errf("%s: no YAML frontmatter block", rel)
		return
	}
	var m yaml.MapSlice
	if err := yaml.Unmarshal([]byte(fm), &m); err != nil {
		r.errf("%s: frontmatter does not parse as YAML: %v", rel, err)
		return
	}
	typ := ""
	for _, item := range m {
		if k, _ := item.Key.(string); k == "type" {
			typ, _ = item.Value.(string)
		}
	}
	if strings.TrimSpace(typ) == "" {
		r.errf("%s: frontmatter has no non-empty `type`", rel)
	}
	if strings.Contains(text, "[^") && !strings.Contains(text, "\n[^") {
		r.warnf("%s: footnote reference with no definition", rel)
	}
}

// validateIndex checks the reserved index.md structure. Frontmatter is optional
// and only the bundle-root index may declare okf_version.
func validateIndex(r *report, rel, text string) {
	fm, _, ok := splitFrontmatter(text)
	if !ok {
		return
	}
	var m map[string]any
	if err := yaml.Unmarshal([]byte(fm), &m); err != nil {
		r.errf("%s: frontmatter does not parse as YAML: %v", rel, err)
		return
	}
	if _, has := m["okf_version"]; has && rel != "/index.md" {
		r.errf("%s: only the bundle-root index.md may declare okf_version", rel)
	}
}

// validateLog checks the reserved log.md structure: date-grouped entries under
// ISO 8601 headings, newest first.
func validateLog(r *report, rel, text string) {
	var dates []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		d := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if !isoDateRE.MatchString(d) {
			r.errf("%s: date heading %q is not ISO 8601 (YYYY-MM-DD)", rel, d)
			continue
		}
		dates = append(dates, d)
	}
	for i := 1; i < len(dates); i++ {
		if dates[i] > dates[i-1] {
			r.warnf("%s: entries are not newest-first (%s follows %s)", rel, dates[i], dates[i-1])
			break
		}
	}
}

// validateLinks resolves every internal markdown link against the files that
// actually exist. Absolute links are bundle-relative per the spec; relative
// ones resolve against the containing directory. External URLs are skipped —
// this checks the bundle's internal integrity, not the live web.
func validateLinks(r *report, rel, text string, present map[string]bool) {
	for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
		target := m[2]
		if i := strings.IndexAny(target, "#?"); i >= 0 {
			target = target[:i]
		}
		if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		resolved := target
		if !strings.HasPrefix(target, "/") {
			resolved = path.Join(path.Dir(rel), target)
		}
		if !present[resolved] {
			r.warnf("%s: broken link to %s", rel, target)
		}
	}
}

// splitFrontmatter separates a leading `---` delimited YAML block from the
// markdown body. Returns ok=false when the document has no frontmatter at all,
// which is an error for a concept and legal for a reserved file.
func splitFrontmatter(text string) (string, string, bool) {
	if !strings.HasPrefix(text, "---\n") {
		return "", text, false
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", text, false
	}
	body := rest[end+len("\n---"):]
	return rest[:end], strings.TrimPrefix(body, "\n"), true
}
