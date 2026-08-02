package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Turn is one message in a haystack session. HasAnswer marks a gold-evidence
// turn (LongMemEval annotates the turns that actually contain the answer);
// we use it only for retrieval-recall reporting, never as an extraction hint.
type Turn struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	HasAnswer bool   `json:"has_answer"`
}

// flexString unmarshals a JSON value that may be a string or a number (some
// LongMemEval gold answers are bare numbers) into a string.
type flexString string

func (s *flexString) UnmarshalJSON(b []byte) error {
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) > 0 && b[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*s = flexString(str)
		return nil
	}
	*s = flexString(strings.Trim(string(b), `"`))
	return nil
}

// Question is one LongMemEval-S record. The haystack is the memory the system
// is allowed to consult; question/answer is the probe and gold.
type Question struct {
	QuestionID         string     `json:"question_id"`
	QuestionType       string     `json:"question_type"`
	Question           string     `json:"question"`
	Answer             flexString `json:"answer"`
	QuestionDate       string     `json:"question_date"`
	HaystackSessionIDs []string   `json:"haystack_session_ids"`
	HaystackDates      []string   `json:"haystack_dates"`
	HaystackSessions   [][]Turn   `json:"haystack_sessions"`
	AnswerSessionIDs   []string   `json:"answer_session_ids"`
}

// selectSubset streams the dataset array and returns the first records that
// satisfy the per-type quota. Streaming (rather than one big Unmarshal) keeps
// the 265MB file off the heap — we stop as soon as every quota is filled.
func selectSubset(path string, quota map[string]int) ([]Question, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	// consume the opening '['
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("read opening token: %w", err)
	}

	remaining := 0
	for _, n := range quota {
		remaining += n
	}
	got := map[string]int{}
	var out []Question
	for dec.More() && remaining > 0 {
		var q Question
		if err := dec.Decode(&q); err != nil {
			return nil, fmt.Errorf("decode record: %w", err)
		}
		want := quota[q.QuestionType]
		if got[q.QuestionType] >= want {
			continue
		}
		got[q.QuestionType]++
		remaining--
		out = append(out, q)
	}
	return out, nil
}
