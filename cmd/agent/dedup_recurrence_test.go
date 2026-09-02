package main

import (
	"reflect"
	"strings"
	"testing"
)

// TestNovelSpecificsOnTheTwoMeasuredRefusals replays the exact pair of writes
// that TestSelfRecallDecaysWithCorpusGrowth saw refused on 2026-09-02, at
// cosine 0.74 and 0.73. Both are recurrences of an earlier topic weeks later,
// not restatements of it, and both were silently dropped. If this test ever
// goes red, the false-refusal behaviour is back.
func TestNovelSpecificsOnTheTwoMeasuredRefusals(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		prior     string
	}{
		{
			name: "crashed machine vs lost sessions, 35 days apart",
			candidate: "Session 2026-08-06 (5e6d9519): Recover Claude sessions from crashed machine\n\n" +
				"Opened with: my machine hard-crashed and I need the sessions back",
			prior: "Session 2026-07-02 (2d582af5): Resume lost Claude code sessions from transcripts " +
				"Opened with: I just lost all my terminator instances and all my active claude code sessions",
		},
		{
			name: "RAM investigation vs RAM investigation, 59 days apart",
			candidate: "Session 2026-08-21 (a78b3bac): RAM and load investigation\n\n" +
				"Opened with: load is pegged again, find out what is eating memory",
			prior: "Session 2026-06-23 (632fb7af): Investigate RAM and disk space usage " +
				"Opened with: what's running me out of ram and also check mdisk space while we're at it",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			novel := novelSpecifics(c.candidate, c.prior)
			if len(novel) == 0 {
				t.Fatalf("no novel specifics found; this write would still be refused")
			}
			t.Logf("stores with a warning, on: %v", novel)
		})
	}
}

func TestNovelSpecificsTokenising(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		prior     string
		want      []string
	}{
		{
			name:      "a date is one token, not three",
			candidate: "happened on 2026-08-06, commit 9673a19b",
			prior:     "nothing dated here",
			want:      []string{"2026-08-06", "9673a19b"},
		},
		{
			name:      "shared specifics are not novel",
			candidate: "on 2026-08-06 the build broke",
			prior:     "the build broke on 2026-08-06",
			want:      nil,
		},
		{
			name:      "hex identifiers count",
			candidate: "commit 9673a19b reverted it",
			prior:     "some earlier commit reverted it",
			want:      []string{"9673a19b"},
		},
		{
			name:      "single digits are ignored as prose noise",
			candidate: "there were 3 of them and 4 more",
			prior:     "there were none",
			want:      nil,
		},
		{
			name:      "repeats within the candidate collapse",
			candidate: "2026-08-06 and again 2026-08-06, both in run 42",
			prior:     "",
			want:      []string{"2026-08-06", "42"},
		},
		{
			name:      "a pure rewording introduces nothing",
			candidate: "the parser slices by byte offset, so context breaks",
			prior:     "context comes back broken because the parser slices by byte offset",
			want:      nil,
		},
		{
			// The 140-session replay refused nothing because every note carried
			// a fresh timestamp. A date on its own says the day changed, which a
			// rewording written today says just as loudly.
			name:      "a date with nothing else beside it is calendar noise",
			candidate: "on 2026-09-02: the parser slices by byte offset, so context breaks",
			prior:     "context comes back broken because the parser slices by byte offset",
			want:      nil,
		},
		{
			name:      "a bare year is calendar noise too",
			candidate: "as of 2026 the shim is still there",
			prior:     "the shim is still there",
			want:      nil,
		},
		{
			name:      "a date counts once a real particular arrives with it",
			candidate: "on 2026-09-02 it refused 140 of them",
			prior:     "it refused some of them",
			want:      []string{"2026-09-02", "140"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := novelSpecifics(c.candidate, c.prior)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("novelSpecifics = %v, want %v", got, c.want)
			}
		})
	}
}

func TestRecurrenceNoteCapsTheList(t *testing.T) {
	many := []string{"11", "22", "33", "44", "55", "66"}
	got := recurrenceNote(0.7, "Whatever", many)
	if strings.Contains(got, "55") || strings.Contains(got, "66") {
		t.Errorf("advisory should cap at 4 specifics, got:\n%s", got)
	}
	if !strings.Contains(got, "44") {
		t.Errorf("advisory should keep the first 4, got:\n%s", got)
	}
}

// TestRecurrenceNoteNamesTheEvidence pins that the advisory says which
// specifics saved the write. A warning that only says "stored anyway" gives
// the caller nothing to check the judgement against.
func TestRecurrenceNoteNamesTheEvidence(t *testing.T) {
	got := recurrenceNote(0.74, "Session 2026-07-02", []string{"2026-08-06", "5e6d9519"})
	for _, want := range []string{"0.74", "Session 2026-07-02", "2026-08-06", "5e6d9519", "recurrence"} {
		if !strings.Contains(got, want) {
			t.Errorf("advisory missing %q:\n%s", want, got)
		}
	}
}
