package main

import (
	"fmt"
	"regexp"
	"strings"
)

// novelSpecifics returns the concrete tokens present in candidate and absent
// from prior, deduplicated, in order of first appearance.
//
// This is the signal that separates a recurrence from a rewording, which
// cosine alone cannot do. Measured 2026-09-02 by
// TestSelfRecallDecaysWithCorpusGrowth: replaying 20 real sessions, two writes
// were refused at cosine 0.74 and 0.73 -- "Recover Claude sessions from
// crashed machine" blocked by a note 35 days older about losing sessions, and
// "RAM and load investigation" blocked by one 59 days older about RAM. Same
// topic, different incident, and the second one silently vanished while the
// first stood as though it were the only occurrence.
//
// A reworded duplicate restates a fact and introduces no new particulars. A
// recurrence arrives with its own date and its own identifiers. That
// asymmetry is deterministic, costs no embedder call, and is what this reads.
func novelSpecifics(candidate, prior string) []string {
	seen := make(map[string]bool)
	for _, m := range specificsPattern.FindAllString(strings.ToLower(prior), -1) {
		seen[m] = true
	}
	var novel []string
	for _, m := range specificsPattern.FindAllString(strings.ToLower(candidate), -1) {
		if seen[m] {
			continue
		}
		seen[m] = true // also collapses repeats within the candidate
		novel = append(novel, m)
	}
	return novel
}

// recurrenceNote renders the advisory that replaces a refusal when a note
// carries specifics its nearest neighbour lacks.
//
// It stores and warns rather than storing silently: cosine was high for a
// reason, and the two notes probably do belong to one thread even though they
// are not the same fact. Naming the specifics is what lets the caller check
// the call rather than take it on trust.
func recurrenceNote(score float64, name string, novel []string) string {
	shown := novel
	if len(shown) > 4 {
		shown = shown[:4]
	}
	return fmt.Sprintf(
		"\n\n⚠ stored despite cosine %.2f against %q: this note carries specifics that one does not (%s), "+
			"so it reads as a recurrence rather than a rewording. If they are the same thread, link them.",
		score, name, strings.Join(shown, ", "))
}

// specificsPattern matches the tokens that make a note about a particular
// occurrence rather than about a topic: ISO dates, hex-ish identifiers (commit
// shas, session ids), and multi-digit numbers.
//
// The date alternative leads on purpose. Go's regexp is leftmost-first, so at
// the "2026" in "2026-08-06" the date branch is tried first and consumes the
// whole date; with the number branch first it would match "2026" and leave
// "08" and "06" as two more tokens, making every date look like three
// specifics instead of one.
//
// Single digits are excluded: they appear in ordinary prose constantly and
// would make almost any pair of notes look distinct.
var specificsPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}|\b[0-9a-f]{7,}\b|\b\d{2,}\b`)
