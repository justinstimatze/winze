package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestCreatedVar(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want string
	}{
		{
			name: "winze-add success line",
			out:  "created entity DontFakeThings (Concept) in memory.go (build gate passed)\n",
			want: "DontFakeThings",
		},
		{
			name: "line preceded by other output",
			out:  "gofmt ok\ncreated entity ReverieMode (Concept) in memory.go (build gate passed)",
			want: "ReverieMode",
		},
		{
			name: "unrecognised output yields no var",
			out:  "added 6 claims across 1 files (one build gate)",
			want: "",
		},
		{name: "empty", out: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := createdVar(tc.out); got != tc.want {
				t.Errorf("createdVar(%q) = %q, want %q", tc.out, got, tc.want)
			}
		})
	}
}

func TestLinkCandidates(t *testing.T) {
	// Scores are floor-relative so the test states the rule, not the current
	// calibration — retuning WINZE_LINK_SUGGEST shouldn't break it.
	f := linkSuggestScore()
	hits := []queryHit{
		{Name: "A", VarName: "AVar", Score: f + 0.20},
		{Name: "B", VarName: "BVar", Score: f + 0.05},
		{Name: "C", VarName: "CVar", Score: f},        // exactly at the floor: kept
		{Name: "D", VarName: "DVar", Score: f - 0.01}, // below: dropped
	}
	got := linkCandidates(hits)
	if len(got) != 3 {
		t.Fatalf("want 3 candidates (capped at linkSuggestMax), got %d", len(got))
	}
	if got[0].VarName != "AVar" || got[2].VarName != "CVar" {
		t.Errorf("wrong candidates: %+v", got)
	}
	for _, h := range got {
		if h.Score < linkSuggestScore() {
			t.Errorf("%s scored %.2f, below the suggest floor", h.VarName, h.Score)
		}
	}

	// A hit with no var name can't be named in a winze_link call.
	if c := linkCandidates([]queryHit{{Name: "E", Score: 0.9}}); len(c) != 0 {
		t.Errorf("hit without VarName should be dropped, got %+v", c)
	}
	if c := linkCandidates(nil); len(c) != 0 {
		t.Errorf("nil hits should yield no candidates, got %+v", c)
	}
}

func TestSuggestLinks(t *testing.T) {
	related := []queryHit{{Name: "Dedup guard", VarName: "WinzeRememberDedupGuard", Brief: "checks cosine", Score: linkSuggestScore() + 0.01}}
	out := suggestLinks("NewMemory", related)
	for _, want := range []string{
		"winze_link(",
		`from="NewMemory"`,
		`to="WinzeRememberDedupGuard"`,
		fmt.Sprintf("%.2f", linkSuggestScore()+0.01),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("suggestion missing %q:\n%s", want, out)
		}
	}

	// No subject var, or nothing related: suggest nothing rather than emit a
	// call the caller can't issue.
	if got := suggestLinks("", related); got != "" {
		t.Errorf("no var name should suppress the suggestion, got %q", got)
	}
	if got := suggestLinks("NewMemory", nil); got != "" {
		t.Errorf("no candidates should suppress the suggestion, got %q", got)
	}
}
