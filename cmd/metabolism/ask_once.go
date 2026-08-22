package main

import (
	"fmt"
	"sort"
	"time"
)

// Ask-once memory for sensor queries.
//
// The loop had none, and the cost was most of what it did. Sense queries the
// first 3 topology targets each cycle (main.go, `limit := 3`); the topology
// ranking is deterministic over an unchanged corpus and learning-goal targets
// are prepended unconditionally, so the same three questions went out every
// hour. Nothing recorded that a question had already been asked, so the same
// query hit the same index and got the same answer, indefinitely.
//
// Measured over 624 logged cycles on 2026-08-22: `goal:GoalPredictiveHallucination`
// alone accounted for 180 of them across six distinct queries — thirty repeats
// each. Only 18 of 114 hypotheses ever reached a substantive verdict, and 194
// of the sensor outcomes were `irrelevant`, which is not 194 failed attempts to
// falsify but roughly 18 questions asked roughly 13 times. Every repeat spent
// API budget and added another row inflating the denominator of
// `survivorship_ratio`. No corpus content has come from any cycle after the
// ninth.
//
// The record needed already existed: every cycle logs its hypothesis, query,
// backend, timestamp and resolution. This consults it.

// Considered and rejected: an exponential backoff on consecutive empty
// returns. Replaying this corpus's 624 logged cycles, it avoided 72% of asks
// against this rule's 39% — but it cost 17 of the 38 substantive verdicts the
// log actually contains, against this rule's 1. Sweeping the base (6h-24h),
// the cap (2d-30d) and the streak at which it engaged (3-12) never moved the
// loss below 10: the verdicts it drops are ones that arrived after a query had
// already come back empty several times, which is precisely what a backoff is
// built to stop asking about. Halving the asks to lose a quarter of the
// findings is the same trade the budget cap made when it killed August, and
// the reason to be suspicious of it is the same.
//
// This rule has no such trade. Declining to re-retrieve an answer already
// sitting unread cannot lose information, because the information is already
// in the log.

// queryOutcome summarises what the log remembers about one
// (hypothesis, query, backend) triple.
//
// Both counters are measured backwards from the most recent ask rather than
// over the triple's lifetime, which is the correction a replay of the real log
// forced. A lifetime "has ever produced a verdict" flag reads as permanent
// immunity: a query that corroborated once in April and has returned nothing
// for four months is exactly the one to stop asking, and a boolean cannot say
// so.
type queryOutcome struct {
	Asks int // how many times it has been asked, ever

	// Pending is asks made after the most recent judged result — retrieved
	// answers nobody has read. Replaying the log showed this is the dominant
	// waste: each of the three live goal queries was asked 60 times in 20 days
	// and judged 3 times. The bottleneck is judgment, not retrieval, and
	// asking again while an unread answer sits in the log buys nothing.
	Pending int

	// DeadStreak is consecutive irrelevant/no_signal verdicts since the last
	// substantive one. Resets to zero the moment a query produces something.
	DeadStreak int

	Last time.Time // most recent ask
}

// recallQuery walks the log for prior asks of this exact triple.
//
// Keyed on the query text rather than the hypothesis alone, deliberately.
// A hypothesis whose query has been rewritten is a genuinely new question and
// should be asked; the same hypothesis re-asked with the same words is not.
func recallQuery(mlog MetabolismLog, hypothesis, query, backend string) queryOutcome {
	var mine []Cycle
	for _, c := range mlog.Cycles {
		be := c.Backend
		if be == "" {
			be = "arxiv" // legacy rows predate the field
		}
		if c.Hypothesis == hypothesis && c.Query == query && be == backend {
			mine = append(mine, c)
		}
	}
	if len(mine) == 0 {
		return queryOutcome{}
	}
	sort.Slice(mine, func(i, j int) bool { return mine[i].Timestamp.Before(mine[j].Timestamp) })

	out := queryOutcome{Asks: len(mine), Last: mine[len(mine)-1].Timestamp}
	// Walk backwards from the newest ask. Both counters describe the tail, so
	// the first substantive verdict encountered ends them.
	for i := len(mine) - 1; i >= 0; i-- {
		switch mine[i].Resolution {
		case "corroborated", "challenged":
			return out
		case "irrelevant", "no_signal":
			out.DeadStreak++
		default:
			out.Pending++
		}
	}
	return out
}

// shouldAskQuery decides whether a sensor query is worth spending on now.
//
// Returns a gateDecision so it reads in the journal exactly like the phase
// gates do — the point is that a skip is visible and says why, rather than the
// loop silently doing the same thing forever, which is the failure this
// replaces.
func shouldAskQuery(mlog MetabolismLog, hypothesis, query, backend string) gateDecision {
	prior := recallQuery(mlog, hypothesis, query, backend)

	if prior.Asks == 0 {
		return gateDecision{Fire: true, Reason: "never asked"}
	}
	// Do not ask a question whose last answer nobody has read. Retrieval is
	// cheap to trigger and judgment is gated behind its own phase, so without
	// this the loop outruns itself: 173 of the 624 logged cycles were asks made
	// on top of an answer still sitting unjudged.
	if prior.Pending > 0 {
		return gateDecision{Fire: false, Reason: fmt.Sprintf("%d retrieved answer(s) still unjudged — read those before asking again", prior.Pending)}
	}
	// Every prior ask has been judged, so there is nothing unread to wait for.
	// The global sense gate rate-limits the ordinary cadence from here.
	if prior.DeadStreak > 0 {
		return gateDecision{Fire: true, Reason: fmt.Sprintf("asked %dx, %d empty running but all judged", prior.Asks, prior.DeadStreak)}
	}
	return gateDecision{Fire: true, Reason: fmt.Sprintf("asked %dx, last verdict was substantive", prior.Asks)}
}
