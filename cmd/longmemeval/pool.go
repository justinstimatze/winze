package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// runAll runs every question, up to n at a time, and returns the rows in
// question order along with the count that errored.
//
// The loop was serial and the run is ~99% blocked on the API: a cold sixty
// question pass measured 17.0 s/question, of which extract is 12.5 s, answer
// 2.5 s and judge 1.1 s, against 0.9 s for the whole winze machinery — build
// 128 ms, defn sync 795 ms, retrieve 2.2 ms. Nothing there gets faster by
// being written better. At that rate the full oracle set is 3.9 hours and the
// real longmemeval_s haystack is 67, which is the difference between a
// benchmark you run and one you talk about running.
//
// Output is printed in question order rather than completion order. Interleaved
// progress lines from concurrent workers are unreadable, and worse, they make a
// log impossible to diff against a serial run — which is the one check that
// says whether concurrency changed any answer.
//
// What makes this safe is that nothing per-question is shared: buildStore
// writes to workDir/store-<qid> and defndb.New opens a .defn beneath it, so
// each worker owns its own store and its own database. The three things that
// were shared are handled at the source — usageStats has a mutex and its one
// unguarded counter now goes through noteCacheHit, and the extraction cache
// writes through a temp file and a rename so two workers missing the same
// session key cannot tear the file.
func runAll(r *runner, questions []Question, k, n int) ([]resultRow, int) {
	if n < 1 {
		n = 1
	}
	type outcome struct {
		row resultRow
		out string
		err error
	}
	results := make([]outcome, len(questions))
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup

	for i, q := range questions {
		wg.Add(1)
		go func(i int, q Question) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var b strings.Builder
			fmt.Fprintf(&b, "\n[%d/%d] %s [%s]\n  Q: %s\n", i+1, len(questions), q.QuestionID, q.QuestionType, q.Question)
			row, err := r.runQuestion(q, k)
			if err != nil {
				results[i] = outcome{out: b.String(), err: err}
				return
			}
			mark := "✗"
			if row.correct {
				mark = "✓"
			}
			fmt.Fprintf(&b, "  gold: %q\n  ans:  %q  %s\n", q.Answer, row.answer, mark)
			results[i] = outcome{row: row, out: b.String()}
		}(i, q)
	}
	wg.Wait()

	var rows []resultRow
	errored := 0
	for _, res := range results {
		fmt.Print(res.out)
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", res.err)
			errored++
			continue
		}
		rows = append(rows, res.row)
	}
	return rows, errored
}
