package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// writeBaseline emits one JSON object per scored question, in run order.
// Errored questions are absent rather than recorded as wrong: a harness
// failure is not a memory failure, and a diff that shows one is misleading.
// selectSubset is deterministic (first N of each type in file order), so the
// same quota yields the same qids and the rows line up across runs.
func writeBaseline(path string, questions []Question, rows []resultRow) error {
	byID := make(map[string]Question, len(questions))
	for _, q := range questions {
		byID[q.QuestionID] = q
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, r := range rows {
		q, ok := byID[r.qid]
		if !ok {
			return fmt.Errorf("baseline: no question for qid %q", r.qid)
		}
		if err := enc.Encode(baselineRow{
			QID:       r.qid,
			Type:      r.qtype,
			Question:  q.Question,
			Gold:      string(q.Answer),
			Answer:    r.answer,
			Correct:   r.correct,
			Sessions:  r.sessions,
			Facts:     r.facts,
			Retrieved: r.retrieved,
			Truncated: r.truncated,
		}); err != nil {
			return err
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// A baseline is the per-question outcome of one configuration, written so the
// next configuration can be diffed question by question instead of score
// against score. The aggregate barely supports that: two byte-identical runs
// scored 43/60 and 44/60, so a two-question move is inside the noise. Which
// specific questions flipped is not — that names a mechanism, and it reads
// even when the totals hardly move.
//
// Deliberately absent: timings. They differ on every run and would rewrite all
// sixty rows each time, burying the handful of lines that carry a finding.
// They stay in the run log, where docs/benchmark.md already records that
// numbers off a loaded box are ceilings rather than measurements.
type baselineRow struct {
	QID       string `json:"qid"`
	Type      string `json:"type"`
	Question  string `json:"question"`
	Gold      string `json:"gold"`
	Answer    string `json:"answer"`
	Correct   bool   `json:"correct"`
	Sessions  int    `json:"sessions"`
	Facts     int    `json:"facts"`
	Retrieved int    `json:"retrieved"`
	// Truncated belongs in the baseline for the same reason Facts does: it is
	// a property of the extraction that a later diff needs in order to read a
	// flip. A question that goes from wrong to right while Truncated goes 1
	// to 0 has an explanation; the same flip with Truncated flat does not.
	Truncated int `json:"truncated"`
}
