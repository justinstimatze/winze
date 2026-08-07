package main

import (
	"fmt"
	"io"
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
// real longmemeval_s haystack is 6.9, which is the difference between a
// benchmark you run and one you talk about running.
//
// `out` receives the per-question log in QUESTION order, not completion order.
// Interleaved lines from concurrent workers are unreadable, and worse, they
// make a log impossible to diff against a serial run — which is the one check
// that says whether concurrency changed any answer. `prog` receives a line per
// finished question in COMPLETION order, so a long run has a heartbeat; it is
// stderr in production, which keeps the heartbeat on the terminal when the
// ordered log is redirected to a file.
//
// `run` is the per-question work, taken as a function rather than reached
// through *runner so this can be tested without an API key. That seam exists
// because the first version of this file shipped with a progress function that
// had never been called: go build, go vet, go test and a race build all passed
// on it, since none of them execute runAll and nothing covered it.
//
// What makes the concurrency safe is that nothing per-question is shared:
// buildStore writes to workDir/store-<qid> and defndb.New opens a .defn
// beneath it, so each worker owns its own store and its own database. The
// three things that were shared are handled at the source — usageStats has a
// mutex and its one unguarded counter now goes through noteCacheHit, and the
// extraction cache writes through a temp file and a rename so two workers
// missing the same session key cannot tear the file.
func runAll(out, prog io.Writer, questions []Question, n int, run func(Question) (resultRow, error)) ([]resultRow, int) {
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
	tick := &ticker{w: prog, total: len(questions)}

	for i, q := range questions {
		wg.Add(1)
		go func(i int, q Question) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var b strings.Builder
			fmt.Fprintf(&b, "\n[%d/%d] %s [%s]\n  Q: %s\n", i+1, len(questions), q.QuestionID, q.QuestionType, q.Question)
			row, err := run(q)
			if err != nil {
				results[i] = outcome{out: b.String(), err: err}
				tick.tick(q.QuestionID, "ERROR")
				return
			}
			mark := "✗"
			if row.correct {
				mark = "✓"
			}
			fmt.Fprintf(&b, "  gold: %q\n  ans:  %q  %s\n", q.Answer, row.answer, mark)
			results[i] = outcome{row: row, out: b.String()}
			tick.tick(q.QuestionID, mark)
		}(i, q)
	}
	wg.Wait()

	var rows []resultRow
	errored := 0
	for _, res := range results {
		fmt.Fprint(out, res.out)
		if res.err != nil {
			fmt.Fprintf(prog, "  ERROR: %v\n", res.err)
			errored++
			continue
		}
		rows = append(rows, res.row)
	}
	return rows, errored
}

// ticker emits one line per finished question, in completion order.
//
// The ordered log stays ordered, which means nothing appears on it until the
// last question lands. That was fine while a run took 25 minutes; on the full
// haystack it is seven hours of an empty file, and the only way to tell a live
// run from a hung one is to count cache entries on disk. Which is what I ended
// up doing.
//
// The mutex covers the counter AND the write as one unit, which matters twice.
// An atomic counter with an unsynchronised Fprintf is a data race on any
// io.Writer that is not itself synchronised — the first version of this shipped
// that way and the race detector caught it the moment a test finally called
// runAll. It also lets [3/8] print before [2/8], since two workers can take
// their numbers and then interleave their writes, which makes the counter
// useless as the liveness signal it exists to be.
type ticker struct {
	mu    sync.Mutex
	w     io.Writer
	done  int
	total int
}

func (t *ticker) tick(qid, mark string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.done++
	fmt.Fprintf(t.w, "  [%d/%d] %s %s\n", t.done, t.total, qid, mark)
}
