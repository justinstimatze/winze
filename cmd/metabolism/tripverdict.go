package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// tripVerdictsFile is the critic's calibration log: one row per trip
// candidate the critic actually ruled on. It lives alongside the other
// .metabolism-* artifacts in the corpus dir and is gitignore-able.
const tripVerdictsFile = ".metabolism-trip-verdicts.jsonl"

// tripVerdictRow records one critic ruling in full — the candidate's own
// argument next to the verdict passed on it.
//
// Why this exists separately from the promotion-attempt log: that log
// (logTripPromotionAttempts) records THAT a candidate was critic-rejected
// and under which reason code, but not WHAT the candidate argued. The
// reason code is also sanitized and truncated to 60 chars, so multi-clause
// rejections arrive cut mid-word ("shallow_pattern_matching_+_predicate_
// misuse_structurallyanal"). Neither is enough to judge whether a
// rejection was correct, which is exactly what calibrating the shallowness
// bar requires.
//
// Label and LabelNote are deliberately empty on write. The file IS the
// labelling worksheet: fill Label with "accept"/"reject" by hand, then
// compare against Accept to get the critic's agreement rate —
//
//	jq -r 'select(.label != "") |
//	       [(if .accept then "accept" else "reject" end), .label] | @tsv' \
//	  corpus/.metabolism-trip-verdicts.jsonl | sort | uniq -c
//
// One asymmetry to keep in mind while labelling: the critic sees Rationale
// (truncated to 800 chars by buildTripCriticPrompt) and does NOT see
// Connection. Both are recorded here because a human labeller wants the
// whole candidate, but judging the critic on text it never received would
// score it for a fault that belongs to the prompt.
type tripVerdictRow struct {
	Timestamp   string  `json:"timestamp"`
	Subject     string  `json:"subject"`
	Object      string  `json:"object"`
	SubjectRole string  `json:"subject_role,omitempty"`
	ObjectRole  string  `json:"object_role,omitempty"`
	ClusterA    int     `json:"cluster_a"`
	ClusterB    int     `json:"cluster_b"`
	Predicate   string  `json:"predicate"`
	Connection  string  `json:"connection"`
	Rationale   string  `json:"rationale"`
	Score       int     `json:"score"`
	PromptType  string  `json:"prompt_type"`
	Temperature float64 `json:"temperature"`
	Accept      bool    `json:"accept"`
	Reason      string  `json:"reason,omitempty"`     // sanitized code, as the pipeline uses it
	RawReason   string  `json:"raw_reason,omitempty"` // the critic's REASON line verbatim
	CriticModel string  `json:"critic_model"`
	Label       string  `json:"label"`                // hand-label slot: "accept" | "reject" | ""
	LabelNote   string  `json:"label_note,omitempty"` // why, when the label disagrees
}

// appendTripVerdicts appends rows to the critic calibration log. Best
// effort: a write failure is reported and returns the count written so
// far, never aborting a promotion run over a logging problem. Returns the
// number of rows appended.
func appendTripVerdicts(dir string, rows []tripVerdictRow) int {
	if len(rows) == 0 {
		return 0
	}
	path := filepath.Join(dir, tripVerdictsFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[trip-verdict] could not open %s: %v\n", path, err)
		return 0
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	ts := time.Now().UTC().Format(time.RFC3339)
	written := 0
	for _, r := range rows {
		if r.Timestamp == "" {
			r.Timestamp = ts
		}
		if err := enc.Encode(r); err != nil {
			fmt.Fprintf(os.Stderr, "[trip-verdict] write row: %v\n", err)
			return written
		}
		written++
	}
	return written
}

// readTripVerdicts loads the calibration log. Returns what it parsed
// alongside the error when a line is malformed — the file is hand-edited
// during labelling, and one bad edit should not hide the rows before it.
func readTripVerdicts(dir string) ([]tripVerdictRow, error) {
	path := filepath.Join(dir, tripVerdictsFile)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []tripVerdictRow
	dec := json.NewDecoder(f)
	for dec.More() {
		var r tripVerdictRow
		if err := dec.Decode(&r); err != nil {
			return rows, fmt.Errorf("decode %s: %w", tripVerdictsFile, err)
		}
		rows = append(rows, r)
	}
	return rows, nil
}
