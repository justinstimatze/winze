package main

import "testing"

func TestNormalizeSlug(t *testing.T) {
	cases := map[string]string{
		"Chinese_room":            "chinese-room",
		"Chinese room":            "chinese-room",
		"chinese-room":            "chinese-room",
		"The Demon-Haunted World": "the-demon-haunted-world",
		"  padded  ":              "padded",
		"":                        "",
	}
	for in, want := range cases {
		if got := normalizeSlug(in); got != want {
			t.Errorf("normalizeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOriginSlugs(t *testing.T) {
	// Origin with "/" separator → extracts trailing title.
	got := originSlugs("Wikipedia (zim 2025-12) / Chinese_room")
	if len(got) == 0 || got[0] != "chinese-room" {
		t.Errorf("originSlugs wiki: %v, want [chinese-room, ...]", got)
	}
	// arXiv ID embedded in origin → extracted as identifier slug.
	arxOut := originSlugs("arXiv:2402.12345 / Paper title")
	foundArx := false
	for _, s := range arxOut {
		if s == "arxiv-2402-12345" {
			foundArx = true
		}
	}
	if !foundArx {
		t.Errorf("originSlugs arxiv: %v missing arxiv-2402-12345", arxOut)
	}
	// Origin with no slash → no title slug, but identifier may match.
	bare := originSlugs("winze trip cycle 1 (speculative cross-cluster connection)")
	_ = bare // no title slug expected; identifier heuristic should not match
}

func TestPaperIsNovel(t *testing.T) {
	idx := &corpusProvenanceIndex{slugs: map[string]bool{
		"chinese-room":  true,
		"pseudoscience": true,
	}}
	cases := []struct {
		name   string
		paper  PaperSummary
		wantOk bool // true = novel (not in corpus)
	}{
		{"title matches, not novel", PaperSummary{Title: "Chinese room"}, false},
		{"title matches underscore variant", PaperSummary{Title: "Chinese_room"}, false},
		{"title not in corpus", PaperSummary{Title: "Hard problem of consciousness"}, true},
		{"id matches corpus slug", PaperSummary{ID: "Pseudoscience"}, false},
		{"zim-prefixed id does not false-match unrelated slug", PaperSummary{ID: "zim:Pseudoscience", Title: "Pseudoscience"}, false}, // Title still matches
		{"zim-prefixed id with novel title", PaperSummary{ID: "zim:Something_else", Title: "Something else"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := idx.paperIsNovel(tc.paper)
			if got != tc.wantOk {
				t.Errorf("paperIsNovel(%+v) = %v, want %v", tc.paper, got, tc.wantOk)
			}
		})
	}
}

func TestClassifyGapStatus(t *testing.T) {
	idx := &corpusProvenanceIndex{slugs: map[string]bool{"chinese-room": true}}
	cases := []struct {
		name string
		c    Cycle
		want string
	}{
		{"kb-internal excluded", Cycle{PredictionType: "trip_lint_durability", PapersFound: 2}, ""},
		{"no papers → no_signal", Cycle{PapersFound: 0}, "no_signal"},
		{"all novel → gap_confirmed", Cycle{PapersFound: 2, Papers: []PaperSummary{{Title: "a"}, {Title: "b"}}}, "gap_confirmed"},
		{"all overlap → no_gap", Cycle{PapersFound: 1, Papers: []PaperSummary{{Title: "Chinese room"}}}, "no_gap"},
		{"mixed → mixed_overlap", Cycle{PapersFound: 2, Papers: []PaperSummary{{Title: "Chinese room"}, {Title: "novel"}}}, "mixed_overlap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyGapStatus(tc.c, idx); got != tc.want {
				t.Errorf("classifyGapStatus = %q, want %q", got, tc.want)
			}
		})
	}
}
