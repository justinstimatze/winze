package main

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These exist because the first version of pool.go shipped with a progress
// function that had never once been called. go build, go vet, go test and a
// race build all passed on it: none of them execute runAll, and nothing
// covered it. The gap was found by a person asking whether the fix had been
// verified, which is not a gate.
//
// Everything below runs against a stub instead of the real per-question work,
// so there is no API key, no network and no cost. Run with -race; the point of
// most of them is the concurrency.

func testQuestions(n int) []Question {
	qs := make([]Question, n)
	for i := range qs {
		qs[i] = Question{
			QuestionID:   fmt.Sprintf("q%02d", i),
			QuestionType: "test-type",
			Question:     fmt.Sprintf("question %d?", i),
		}
	}
	return qs
}

// The ordered log is what lets a concurrent run be diffed against a serial
// one, which is the only check that says whether concurrency changed an
// answer. Completion order here is exactly reversed against question order.
func TestRunAllPrintsInQuestionOrderNotCompletionOrder(t *testing.T) {
	qs := testQuestions(8)
	var out, prog strings.Builder
	rows, errored := runAll(&out, &prog, qs, 8, func(q Question) (resultRow, error) {
		// q00 sleeps longest, q07 returns first.
		var i int
		fmt.Sscanf(q.QuestionID, "q%d", &i)
		time.Sleep(time.Duration(len(qs)-i) * 5 * time.Millisecond)
		return resultRow{qid: q.QuestionID, correct: true, answer: "a"}, nil
	})
	if errored != 0 {
		t.Fatalf("errored = %d, want 0", errored)
	}
	if len(rows) != len(qs) {
		t.Fatalf("rows = %d, want %d", len(rows), len(qs))
	}
	for i, r := range rows {
		if r.qid != qs[i].QuestionID {
			t.Errorf("rows[%d].qid = %q, want %q — rows came back in completion order", i, r.qid, qs[i].QuestionID)
		}
	}
	var seen []string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "[") {
			seen = append(seen, line)
		}
	}
	if len(seen) != len(qs) {
		t.Fatalf("printed %d headers, want %d", len(seen), len(qs))
	}
	for i, line := range seen {
		want := fmt.Sprintf("[%d/%d] q%02d", i+1, len(qs), i)
		if !strings.HasPrefix(line, want) {
			t.Errorf("log line %d = %q, want prefix %q — output is in completion order", i, line, want)
		}
	}
	// The heartbeat is the opposite: completion order, so q07 lands first.
	if !strings.HasPrefix(strings.TrimSpace(prog.String()), "[1/8] q07") {
		t.Errorf("first progress line = %q, want q07 first (completion order)",
			strings.SplitN(strings.TrimSpace(prog.String()), "\n", 2)[0])
	}
}

// Every question must produce a heartbeat line, or a long run's counter drifts
// from reality and stops being usable as a liveness signal.
func TestRunAllProgressCoversEveryQuestion(t *testing.T) {
	qs := testQuestions(20)
	var out, prog strings.Builder
	var mu sync.Mutex
	runAll(&out, &prog, qs, 5, func(q Question) (resultRow, error) {
		mu.Lock()
		defer mu.Unlock()
		return resultRow{qid: q.QuestionID, correct: true}, nil
	})
	lines := strings.Split(strings.TrimSpace(prog.String()), "\n")
	if len(lines) != len(qs) {
		t.Fatalf("progress lines = %d, want %d", len(lines), len(qs))
	}
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), fmt.Sprintf("[%d/%d]", i+1, len(qs))) {
			t.Errorf("progress line %d = %q, want counter %d/%d", i, line, i+1, len(qs))
		}
	}
}

// n is a bound, not a suggestion. Exceeding it means the API sees more load
// than asked for, which is how a run starts collecting 429s.
func TestRunAllRespectsConcurrencyLimit(t *testing.T) {
	for _, n := range []int{1, 3, 7} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			qs := testQuestions(30)
			var live, peak int64
			var out, prog strings.Builder
			runAll(&out, &prog, qs, n, func(q Question) (resultRow, error) {
				cur := atomic.AddInt64(&live, 1)
				for {
					p := atomic.LoadInt64(&peak)
					if cur <= p || atomic.CompareAndSwapInt64(&peak, p, cur) {
						break
					}
				}
				time.Sleep(2 * time.Millisecond)
				atomic.AddInt64(&live, -1)
				return resultRow{qid: q.QuestionID}, nil
			})
			if peak > int64(n) {
				t.Errorf("peak concurrency = %d, limit was %d", peak, n)
			}
			if peak < 1 {
				t.Error("nothing ran")
			}
		})
	}
}

// n < 1 must not deadlock on a zero-capacity semaphore or spawn unbounded.
func TestRunAllCoercesBadConcurrency(t *testing.T) {
	for _, n := range []int{0, -1} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			qs := testQuestions(4)
			var out, prog strings.Builder
			done := make(chan struct{})
			go func() {
				defer close(done)
				rows, _ := runAll(&out, &prog, qs, n, func(q Question) (resultRow, error) {
					return resultRow{qid: q.QuestionID}, nil
				})
				if len(rows) != len(qs) {
					t.Errorf("rows = %d, want %d", len(rows), len(qs))
				}
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("runAll deadlocked on a non-positive concurrency")
			}
		})
	}
}

// The error path had never been exercised, even by hand. A failing question
// must not take its neighbours with it, must not appear in rows, and must
// still tick the heartbeat — a run silently dropping questions is exactly when
// a liveness counter matters.
func TestRunAllErrorsDoNotLoseOtherRows(t *testing.T) {
	qs := testQuestions(10)
	var out, prog strings.Builder
	rows, errored := runAll(&out, &prog, qs, 4, func(q Question) (resultRow, error) {
		var i int
		fmt.Sscanf(q.QuestionID, "q%d", &i)
		if i%3 == 0 {
			return resultRow{}, fmt.Errorf("boom on %s", q.QuestionID)
		}
		return resultRow{qid: q.QuestionID, correct: true}, nil
	})
	if want := 4; errored != want { // q00, q03, q06, q09
		t.Errorf("errored = %d, want %d", errored, want)
	}
	if want := len(qs) - 4; len(rows) != want {
		t.Fatalf("rows = %d, want %d", len(rows), want)
	}
	for _, r := range rows {
		var i int
		fmt.Sscanf(r.qid, "q%d", &i)
		if i%3 == 0 {
			t.Errorf("%s errored but is in rows", r.qid)
		}
	}
	// The heartbeat counts every question, errors included. Count only the
	// [n/m] lines: the end-of-run "ERROR: ..." summaries go to the same
	// writer, and a total-line count is satisfied by them one-for-one when the
	// heartbeat is skipped — which is exactly how the first version of this
	// assertion passed against a mutant that dropped the errored tick.
	beats := 0
	for _, line := range strings.Split(prog.String(), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			beats++
		}
	}
	if beats != len(qs) {
		t.Errorf("heartbeat lines = %d, want %d — errored questions skipped the heartbeat", beats, len(qs))
	}
	if !strings.Contains(prog.String(), "ERROR") {
		t.Error("no ERROR marker in the progress stream")
	}
}
