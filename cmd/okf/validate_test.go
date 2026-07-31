package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBundle(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// The bundle the exporter actually produces must pass its own conformance
// check with no warnings — including the link check, which the spec lets
// consumers ignore but a producer has no excuse for failing.
func TestExportedBundleIsConformant(t *testing.T) {
	out := exportFixture(t)
	r, err := validateBundle(out)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ok() {
		t.Errorf("exported bundle is not conformant: %v", r.Errors)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("exported bundle should emit no warnings, got: %v", r.Warnings)
	}
	if r.Concepts == 0 {
		t.Error("no concept documents counted")
	}
}

func TestMissingTypeIsAnError(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"index.md":         "# B\n",
		"concepts/none.md": "---\ntitle: No type\n---\n\nbody\n",
		"concepts/bare.md": "no frontmatter at all\n",
		"concepts/ok.md":   "---\ntype: Concept\n---\n\nbody\n",
	})
	r, err := validateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.ok() {
		t.Fatal("expected non-conformant")
	}
	joined := strings.Join(r.Errors, "\n")
	if !strings.Contains(joined, "no non-empty `type`") {
		t.Errorf("missing type not reported: %v", r.Errors)
	}
	if !strings.Contains(joined, "no YAML frontmatter") {
		t.Errorf("missing frontmatter not reported: %v", r.Errors)
	}
	if len(r.Errors) != 2 {
		t.Errorf("the valid document should not be flagged: %v", r.Errors)
	}
}

func TestUnparseableFrontmatterIsAnError(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"concepts/bad.md": "---\ntype: Concept\ntags: [unclosed\n---\n\nbody\n",
	})
	r, err := validateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.ok() {
		t.Fatal("expected non-conformant")
	}
	if !strings.Contains(strings.Join(r.Errors, "\n"), "does not parse as YAML") {
		t.Errorf("unexpected errors: %v", r.Errors)
	}
}

// okf_version is legal only on the bundle-root index.
func TestOKFVersionOnlyAtRoot(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"index.md":          "---\nokf_version: \"0.2\"\n---\n\n# B\n",
		"concepts/index.md": "---\nokf_version: \"0.2\"\n---\n\n# C\n",
	})
	r, err := validateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.ok() {
		t.Fatal("expected non-conformant")
	}
	if !strings.Contains(strings.Join(r.Errors, "\n"), "only the bundle-root index.md") {
		t.Errorf("unexpected errors: %v", r.Errors)
	}
}

func TestLogStructure(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"log.md": "# Log\n\n## May 22 2026\n\n* **Update**: x\n",
	})
	r, err := validateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.ok() || !strings.Contains(strings.Join(r.Errors, "\n"), "not ISO 8601") {
		t.Errorf("non-ISO date heading should be an error: %v", r.Errors)
	}

	root = writeBundle(t, map[string]string{
		"log.md": "# Log\n\n## 2026-04-13\n\n* **Update**: x\n\n## 2026-05-22\n\n* **Update**: y\n",
	})
	r, err = validateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ok() {
		t.Errorf("ordering is a warning, not an error: %v", r.Errors)
	}
	if !strings.Contains(strings.Join(r.Warnings, "\n"), "newest-first") {
		t.Errorf("out-of-order log not warned: %v", r.Warnings)
	}
}

// Broken links are a warning, not an error: the spec forbids consumers from
// rejecting on them, so a bundle with one is still conformant. Winze reports
// them anyway because emitting one is a producer bug.
func TestBrokenLinksWarnButConform(t *testing.T) {
	root := writeBundle(t, map[string]string{
		"index.md": "---\nokf_version: \"0.2\"\n---\n\n# B\n",
		"concepts/a.md": "---\ntype: Concept\n---\n\n" +
			"* [gone](/concepts/missing.md)\n" +
			"* [here](/concepts/b.md)\n" +
			"* [rel](./b.md)\n" +
			"* [ext](https://example.com/x.md)\n" +
			"* [anchored](/concepts/b.md#section)\n",
		"concepts/b.md": "---\ntype: Concept\n---\n\nbody\n",
	})
	r, err := validateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ok() {
		t.Errorf("broken links must not make a bundle non-conformant: %v", r.Errors)
	}
	warn := strings.Join(r.Warnings, "\n")
	if !strings.Contains(warn, "/concepts/missing.md") {
		t.Errorf("broken link not warned: %v", r.Warnings)
	}
	for _, ok := range []string{"/concepts/b.md\n", "./b.md", "example.com"} {
		if strings.Contains(warn, "broken link to "+ok) {
			t.Errorf("valid link %q flagged: %v", ok, r.Warnings)
		}
	}
	if len(r.Warnings) != 1 {
		t.Errorf("expected exactly one warning, got %v", r.Warnings)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	fm, body, ok := splitFrontmatter("---\ntype: Concept\n---\n\nhello\n")
	if !ok || fm != "type: Concept" || body != "\nhello\n" {
		t.Errorf("got fm=%q body=%q ok=%v", fm, body, ok)
	}
	if _, _, ok := splitFrontmatter("no frontmatter\n"); ok {
		t.Error("expected ok=false with no frontmatter")
	}
	if _, _, ok := splitFrontmatter("---\ntype: Concept\n"); ok {
		t.Error("expected ok=false with an unterminated block")
	}
}
