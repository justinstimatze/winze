package main

import "testing"

func TestLintHasFailures(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "clean output",
			output: "[naming-oracle] 16 role types, 16 grounded, 0 ungrounded\n[value-conflict] 4 functional predicates, 0 unresolved conflicts",
			want:   false,
		},
		{
			name:   "naming oracle FAIL",
			output: "[naming-oracle] FAIL: 2 ungrounded role types\n  Widget  some_file.go:10",
			want:   true,
		},
		{
			name:   "value conflict CONFLICT",
			output: "[value-conflict] CONFLICT: FormedAt has 2 values for LakeCheko",
			want:   true,
		},
		{
			name:   "FAIL in unrelated context",
			output: "[brief-check] 1 FAIL: overlong brief",
			want:   false,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lintHasFailures(tc.output)
			if got != tc.want {
				t.Errorf("lintHasFailures(%q) = %v, want %v", tc.output, got, tc.want)
			}
		})
	}
}

func TestLLMHasContradictions(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "no contradiction",
			output: "[llm-contradiction] 0 contradictions found",
			want:   false,
		},
		{
			name:   "CONTRADICTION keyword",
			output: "CONTRADICTION: entity X claims Y but Z claims not-Y",
			want:   true,
		},
		{
			name:   "lowercase contradiction detected",
			output: "contradiction detected in claim pair 3",
			want:   true,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := llmHasContradictions(tc.output)
			if got != tc.want {
				t.Errorf("llmHasContradictions(%q) = %v, want %v", tc.output, got, tc.want)
			}
		})
	}
}
