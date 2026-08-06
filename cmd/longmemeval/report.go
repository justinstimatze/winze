package main

import (
	"fmt"
	"sort"
	"sync"
)

// resultRow is the outcome of one question, with per-hop nanosecond timings.
type resultRow struct {
	qid        string
	qtype      string
	sessions   int
	facts      int
	retrieved  int
	correct    bool
	answer     string
	extractNS  int64
	buildNS    int64
	syncNS     int64
	retrieveNS int64
	answerNS   int64
	judgeNS    int64
}

// modelUsage accumulates token counts for one model.
type modelUsage struct {
	calls     int
	inputTok  int64
	cachedTok int64
	outputTok int64
}

// usageStats tracks API usage across the run. Concurrency-safe in case the
// loop is parallelised later; today it's called serially.
type usageStats struct {
	mu            sync.Mutex
	perModel      map[string]*modelUsage
	lensCacheHits int
}

func (s *usageStats) record(model string, in, cached, out int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.perModel[model]
	if m == nil {
		m = &modelUsage{}
		s.perModel[model] = m
	}
	m.calls++
	m.inputTok += in
	m.cachedTok += cached
	m.outputTok += out
}

func ms(ns int64) float64 { return float64(ns) / 1e6 }

// report prints the accuracy and the timing breakdown that the perf story
// cares about: the winze-via-defn machinery (build + sync + retrieve) held
// apart from the LLM hops (extract + answer + judge).
//
// errored is the count of questions that raised before producing a row. It
// is reported rather than absorbed, because rows is the SURVIVORS and
// dividing by it alone turns harness failures into a higher score: three
// questions failing to build out of five would print "RESULTS (2
// questions) … accuracy 2/2 = 100%", with nothing on screen saying the
// subset was ever larger. A benchmark number that silently shrinks its own
// denominator is worse than no number, since the first one produced is the
// one that gets quoted.
func report(rows []resultRow, errored int, stats *usageStats) {
	fmt.Printf("\n%s\n", divider())
	fmt.Printf("RESULTS  (%d scored", len(rows))
	if errored > 0 {
		fmt.Printf(", %d errored", errored)
	}
	fmt.Printf(" of %d attempted)\n", len(rows)+errored)
	fmt.Printf("%s\n", divider())

	if len(rows) == 0 {
		fmt.Println("no results")
		return
	}

	correct := 0
	byType := map[string][2]int{} // type -> [correct, total]
	var sumMachine, sumLLM float64
	for _, r := range rows {
		if r.correct {
			correct++
		}
		c := byType[r.qtype]
		if r.correct {
			c[0]++
		}
		c[1]++
		byType[r.qtype] = c
		sumMachine += ms(r.buildNS + r.syncNS + r.retrieveNS)
		sumLLM += ms(r.extractNS + r.answerNS + r.judgeNS)
	}

	fmt.Printf("accuracy: %d/%d = %.0f%%  (of scored)\n", correct, len(rows), 100*float64(correct)/float64(len(rows)))
	if errored > 0 {
		// The honest lower bound. Which of the two is the real number depends
		// on whether the errors are harness bugs or genuine inability to
		// answer, and that is a judgement for whoever reads the stderr lines
		// — so print both and make the spread visible rather than picking.
		att := len(rows) + errored
		fmt.Printf("          %d/%d = %.0f%%  (of attempted — counts the %d errored as wrong)\n",
			correct, att, 100*float64(correct)/float64(att), errored)
	}
	var types []string
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	for _, t := range types {
		c := byType[t]
		fmt.Printf("  %-22s %d/%d\n", t, c[0], c[1])
	}

	fmt.Printf("\nper-question timings (ms):\n")
	fmt.Printf("  %-9s %5s %5s %6s | %8s %6s %8s | %8s %7s %7s\n",
		"qid", "sess", "facts", "retr", "extract", "build", "sync", "retrieve", "answer", "judge")
	for _, r := range rows {
		fmt.Printf("  %-9s %5d %5d %6d | %8.0f %6.0f %8.1f | %8.2f %7.0f %7.0f\n",
			r.qid, r.sessions, r.facts, r.retrieved,
			ms(r.extractNS), ms(r.buildNS), ms(r.syncNS),
			ms(r.retrieveNS), ms(r.answerNS), ms(r.judgeNS))
	}

	n := float64(len(rows))
	fmt.Printf("\nmean winze machinery (build+sync+retrieve): %.1f ms/question\n", sumMachine/n)
	fmt.Printf("mean LLM hops (extract+answer+judge):       %.1f ms/question\n", sumLLM/n)
	fmt.Printf("  (extract dominates and caches to disk — warm re-runs pay only sync+retrieve+answer+judge)\n")

	fmt.Printf("\ntoken usage (lens cache hits: %d sessions served from disk):\n", stats.lensCacheHits)
	var models []string
	for m := range stats.perModel {
		models = append(models, m)
	}
	sort.Strings(models)
	for _, m := range models {
		u := stats.perModel[m]
		fmt.Printf("  %-28s %3d calls  in=%d (cached=%d) out=%d\n",
			m, u.calls, u.inputTok, u.cachedTok, u.outputTok)
	}
}

func divider() string {
	return "════════════════════════════════════════════════════════════════"
}
