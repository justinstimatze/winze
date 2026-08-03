package main

import (
	"fmt"
	"strings"
)

// probeTruncation checks, for each selected question, whether the gold-answer-
// bearing turns (has_answer=true) survive renderSession's truncation. If an
// answer turn is cut, the lens never sees it and no prompt change can recover
// the fact — that validates (or refutes) the truncation hypothesis before we
// pay for a full re-extraction run. No API calls.
func probeTruncation(questions []Question) {
	for _, q := range questions {
		var cutSessions, ansTurns, cutAnsTurns int
		for _, sess := range q.HaystackSessions {
			rendered := renderSession(sess)
			truncated := strings.Contains(rendered, "[...session truncated...]")
			sessHasAns := false
			for _, t := range sess {
				if !t.HasAnswer {
					continue
				}
				sessHasAns = true
				ansTurns++
				// Did this answer turn's content make it into the rendered body?
				// Use a distinctive head slice to test membership robustly.
				head := t.Content
				if len(head) > 120 {
					head = head[:120]
				}
				if len(head) > 4000 {
					head = head[:4000]
				}
				if !strings.Contains(rendered, head) {
					cutAnsTurns++
				}
			}
			if truncated && sessHasAns {
				cutSessions++
			}
		}
		flag := ""
		if cutAnsTurns > 0 {
			flag = "  <-- ANSWER TURN CUT"
		}
		fmt.Printf("%-14s [%-24s] ans-turns=%d cut=%d truncated-answer-sessions=%d%s\n",
			q.QuestionID, q.QuestionType, ansTurns, cutAnsTurns, cutSessions, flag)
	}
}
