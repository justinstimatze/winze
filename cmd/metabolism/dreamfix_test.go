package main

import (
	"testing"
)

func TestEscapeGoString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"double quotes", `He said "hello"`, `He said \"hello\"`},
		{"newlines", "line1\nline2", `line1\nline2`},
		{"tabs", "col1\tcol2", `col1\tcol2`},
		{"combined", "He said \"hi\"\nand\tleft", `He said \"hi\"\nand\tleft`},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := escapeGoString(tc.input)
			if got != tc.want {
				t.Errorf("escapeGoString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCollectBriefTargets(t *testing.T) {
	root := repoRoot(t)

	// Test with overlong=true: may find 0 if all Briefs are within threshold.
	// Verify structure of any returned targets.
	targets := collectBriefTargets(root, true)
	for _, target := range targets {
		if target.file == "" {
			t.Error("target has empty file path")
		}
		if target.entity == "" {
			t.Error("target has empty entity name")
		}
		if target.brief == "" && !target.isMissing {
			t.Error("target has empty brief but is not marked missing")
		}
		if target.briefStart >= target.briefEnd {
			t.Errorf("target %s: briefStart (%d) >= briefEnd (%d)",
				target.entity, target.briefStart, target.briefEnd)
		}
	}
	t.Logf("collectBriefTargets found %d overlong targets", len(targets))

	// Test with overlong=false: should find targets (missing Briefs, etc.)
	allTargets := collectBriefTargets(root, false)
	t.Logf("collectBriefTargets found %d total targets (overlong=false)", len(allTargets))
}
