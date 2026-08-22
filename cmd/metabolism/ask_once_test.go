package main

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func askCycle(hyp, query, backend, resolution string, at time.Time) Cycle {
	return Cycle{Hypothesis: hyp, Query: query, Backend: backend, Resolution: resolution, Timestamp: at}
}

// TestShouldAskQueryDoesNotStarveProductiveQueries is the guard against
// overcorrecting. Twenty empties followed by a corroboration is a live source,
// not a barren one — the streak resets on the verdict.
func TestShouldAskQueryDoesNotStarveProductiveQueries(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	const h, q, b = "BrownHumanUniversalsThesis", "Brown Human Universals", "arxiv"

	var mlog MetabolismLog
	for i := 0; i < 20; i++ {
		mlog.Cycles = append(mlog.Cycles, askCycle(h, q, b, "irrelevant", now.Add(-time.Duration(i+2)*time.Hour)))
	}
	mlog.Cycles = append(mlog.Cycles, askCycle(h, q, b, "corroborated", now.Add(-time.Hour)))

	got := shouldAskQuery(mlog, h, q, b)
	if !got.Fire {
		t.Errorf("a query whose LAST verdict was substantive must stay live: %q", got.Reason)
	}
}

// TestShouldAskQueryWaitsForUnreadAnswers is the dominant rule. Replaying the
// real log showed 173 of 624 cycles were asks made while an already-retrieved
// answer sat unjudged: each of the three live goal queries was asked 60 times
// in 20 days and judged 3 times. Retrieval outran judgment, and every extra ask
// bought a duplicate nobody read.
func TestShouldAskQueryWaitsForUnreadAnswers(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	const h, q, b = "goal:GoalPredictiveHallucination", "active inference perception", "arxiv"

	var mlog MetabolismLog
	mlog.Cycles = append(mlog.Cycles, askCycle(h, q, b, "corroborated", now.Add(-72*time.Hour)))
	mlog.Cycles = append(mlog.Cycles, askCycle(h, q, b, "", now.Add(-2*time.Hour))) // retrieved, unread

	got := shouldAskQuery(mlog, h, q, b)
	if got.Fire {
		t.Errorf("an unread answer must block the next ask: %q", got.Reason)
	}
	if !strings.Contains(got.Reason, "unjudged") {
		t.Errorf("the skip must name the reason: %q", got.Reason)
	}

	// Once that answer is judged, the query is live again.
	mlog.Cycles = append(mlog.Cycles, askCycle(h, q, b, "corroborated", now.Add(-time.Hour)))
	if got := shouldAskQuery(mlog, h, q, b); !got.Fire {
		t.Errorf("judging the backlog should unblock it: %q", got.Reason)
	}
}

// TestShouldAskQueryKeysOnTheQueryNotTheHypothesis pins the granularity. The
// same hypothesis asked a NEW way is a new question; asked the same way it is
// not. Keying on the hypothesis alone would freeze out rephrasings, which is
// how the loop would actually escape a barren query.
func TestShouldAskQueryKeysOnTheQueryNotTheHypothesis(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	var mlog MetabolismLog
	// One unread answer against the old phrasing — that triple is blocked.
	mlog.Cycles = append(mlog.Cycles, askCycle("H", "old phrasing", "arxiv", "", now.Add(-time.Hour)))

	if got := shouldAskQuery(mlog, "H", "old phrasing", "arxiv"); got.Fire {
		t.Errorf("the triple with an unread answer should be held back: %q", got.Reason)
	}
	if got := shouldAskQuery(mlog, "H", "a different phrasing", "arxiv"); !got.Fire {
		t.Errorf("a rephrased query is a new question and must fire: %q", got.Reason)
	}
	if got := shouldAskQuery(mlog, "H", "old phrasing", "zim"); !got.Fire {
		t.Errorf("a different backend is a different source and must fire: %q", got.Reason)
	}
}

// TestUnjudgedAsksDoNotCountAsBarren keeps the two rules distinct. Ten
// retrieved-but-unread answers block the next ask, but they are not evidence
// the query is barren — the skip must say "unjudged", not invent a dead streak
// and start doubling a backoff on questions nobody has read yet.
func TestUnjudgedAsksDoNotCountAsBarren(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	var mlog MetabolismLog
	for i := 0; i < 10; i++ {
		mlog.Cycles = append(mlog.Cycles, askCycle("H", "q", "arxiv", "", now.Add(-time.Duration(i+1)*time.Hour)))
	}
	got := shouldAskQuery(mlog, "H", "q", "arxiv")
	if got.Fire {
		t.Errorf("ten unread answers should block: %q", got.Reason)
	}
	if strings.Contains(got.Reason, "empty return") {
		t.Errorf("unread is not barren — the reason conflates them: %q", got.Reason)
	}
	if recallQuery(mlog, "H", "q", "arxiv").DeadStreak != 0 {
		t.Error("unjudged asks must not accumulate a dead streak")
	}
}

// TestAskOnceReplayAgainstTheRealLog runs the gate over this corpus's actual
// history, one cycle at a time, and reports how many asks it would have
// avoided. It is the only test here that exercises the real distribution
// rather than a fixture, and it is what caught the first design being wrong:
// a lifetime "ever productive" flag avoided nothing on August data.
//
// Skips when the log is absent, which is every CI run — .metabolism-log.json
// is gitignored. That makes this a local instrument rather than a gate, which
// is the honest status for a measurement whose input is not in the repo.
func TestAskOnceReplayAgainstTheRealLog(t *testing.T) {
	mlog := loadLog(filepath.Join(repoCorpusDir(t), ".metabolism-log.json"))
	if len(mlog.Cycles) == 0 {
		t.Skip("no local metabolism log — nothing to replay")
	}
	cycles := append([]Cycle(nil), mlog.Cycles...)
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].Timestamp.Before(cycles[j].Timestamp) })

	var seen MetabolismLog
	asked, skipped := 0, 0
	for _, c := range cycles {
		be := c.Backend
		if be == "" {
			be = "arxiv"
		}
		if shouldAskQuery(seen, c.Hypothesis, c.Query, be).Fire {
			asked++
			seen.Cycles = append(seen.Cycles, c)
		} else {
			skipped++
		}
	}
	total := asked + skipped
	t.Logf("replayed %d cycles: %d asks kept, %d avoided (%.0f%%)",
		total, asked, skipped, float64(skipped)/float64(total)*100)

	// The measured figure on this log is 39%. Asserting a floor rather than the
	// number keeps the test from failing every time the log grows, while still
	// catching a change that quietly turns the gate into a no-op — which is
	// exactly what the first design was on August data.
	if skipped*100/total < 20 {
		t.Errorf("gate avoided only %d%% of asks — it is close to a no-op", skipped*100/total)
	}
}

// repoCorpusDir locates corpus/ from the package directory.
func repoCorpusDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "corpus")
}

// TestAskOnceKeepsTheAsksThatProducedValue is the safety side of the replay.
// Avoiding 72% of asks is only good if none of the avoided ones are the asks
// that actually returned a substantive verdict — a gate that saves money by
// skipping the productive queries is worse than no gate.
func TestAskOnceKeepsTheAsksThatProducedValue(t *testing.T) {
	mlog := loadLog(filepath.Join(repoCorpusDir(t), ".metabolism-log.json"))
	if len(mlog.Cycles) == 0 {
		t.Skip("no local metabolism log — nothing to replay")
	}
	cycles := append([]Cycle(nil), mlog.Cycles...)
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].Timestamp.Before(cycles[j].Timestamp) })

	var seen MetabolismLog
	keptValue, lostValue := 0, 0
	for _, c := range cycles {
		be := c.Backend
		if be == "" {
			be = "arxiv"
		}
		substantive := c.Resolution == "corroborated" || c.Resolution == "challenged"
		if shouldAskQuery(seen, c.Hypothesis, c.Query, be).Fire {
			seen.Cycles = append(seen.Cycles, c)
			if substantive {
				keptValue++
			}
		} else if substantive {
			lostValue++
		}
	}
	t.Logf("substantive verdicts: %d kept, %d would have been skipped", keptValue, lostValue)
	// Measured at 37 kept / 1 lost. A tenth is a generous ceiling: the whole
	// argument for this rule over a backoff is that it barely costs verdicts,
	// so a change that starts costing them has abandoned the argument.
	if lostValue*10 > keptValue {
		t.Errorf("the gate skipped %d productive asks against %d kept — it is cutting the wrong ones",
			lostValue, keptValue)
	}
}
