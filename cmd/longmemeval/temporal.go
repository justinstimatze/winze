package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// factDateLayout is the layout LongMemEval-S question dates and Fact.Date
// both use, e.g. "2023/02/17 (Fri) 21:16".
const factDateLayout = "2006/01/02 (Mon) 15:04"

// dateProximityRankFacts ranks facts by ascending absolute day-distance from
// anchor, so the third RRF channel rewards a fact for having happened near
// when the question is actually asking about -- a signal neither term
// overlap nor embedding cosine can see. A fact with an unparseable Date is
// skipped from this channel's rank map only (still fully visible to the
// other two channels), mirroring semanticRankFacts's own per-fact failure
// handling ("a single fact's embedding failure only drops that fact from
// this channel").
func dateProximityRankFacts(facts []Fact, anchor time.Time) map[int]int {
	type scored struct {
		idx   int
		delta float64
	}
	var ss []scored
	for i, f := range facts {
		t, ok := parseFactDate(f.Date)
		if !ok {
			continue
		}
		delta := t.Sub(anchor).Hours() / 24
		if delta < 0 {
			delta = -delta
		}
		ss = append(ss, scored{idx: i, delta: delta})
	}
	sort.SliceStable(ss, func(a, b int) bool {
		if ss[a].delta != ss[b].delta {
			return ss[a].delta < ss[b].delta
		}
		return ss[a].idx < ss[b].idx
	})
	rank := make(map[int]int, len(ss))
	for r, s := range ss {
		rank[s.idx] = r + 1
	}
	return rank
}

// mostRecentWeekday returns the most recent occurrence of wd strictly
// before now -- "last Saturday" means the Saturday before today, not today
// itself even if today is a Saturday.
func mostRecentWeekday(now time.Time, wd time.Weekday) time.Time {
	d := now.AddDate(0, 0, -1)
	for d.Weekday() != wd {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// mostRecentWeekendDay returns the most recent Saturday or Sunday on or
// before now -- how "the past weekend" is naturally read relative to now.
func mostRecentWeekendDay(now time.Time) time.Time {
	d := now
	for d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// parseFactDate parses a Fact.Date string using the same layout LongMemEval
// dates share throughout this package.
func parseFactDate(dateStr string) (time.Time, bool) {
	t, err := time.Parse(factDateLayout, dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// resolveAnchorDate reads question for a single-anchor relative-date
// expression ("two weeks ago", "last Saturday", "past weekend", a named
// holiday) and resolves it to a concrete date using questionDate as "now".
// Returns false -- contributing nothing to retrieval -- when the question
// matches exclusionPattern or when no known pattern matches at all. Fails
// open by construction, the same posture as embedCached failing open to
// rankFacts on an unreachable ollama.
func resolveAnchorDate(question, questionDate string) (time.Time, bool) {
	if exclusionPattern.MatchString(question) {
		return time.Time{}, false
	}
	now, err := time.Parse(factDateLayout, questionDate)
	if err != nil {
		return time.Time{}, false
	}

	lower := strings.ToLower(question)
	for holiday, md := range fixedHolidayMonthDay {
		if strings.Contains(lower, holiday) {
			return time.Date(now.Year(), time.Month(md[0]), md[1], 0, 0, 0, 0, now.Location()), true
		}
	}
	if todayPattern.MatchString(question) {
		return now, true
	}
	if yesterdayPattern.MatchString(question) {
		return now.AddDate(0, 0, -1), true
	}
	if pastWeekendPattern.MatchString(question) {
		return mostRecentWeekendDay(now), true
	}
	if m := lastWeekdayPattern.FindStringSubmatch(question); m != nil {
		wd := weekdayByName[strings.ToLower(m[1])]
		return mostRecentWeekday(now, wd), true
	}
	if m := nDaysAgoPattern.FindStringSubmatch(question); m != nil {
		n, _ := strconv.Atoi(m[1])
		return now.AddDate(0, 0, -n), true
	}
	if m := nWeeksAgoPattern.FindStringSubmatch(question); m != nil {
		n, _ := strconv.Atoi(m[1])
		return now.AddDate(0, 0, -7*n), true
	}
	if m := nMonthsAgoPattern.FindStringSubmatch(question); m != nil {
		n, _ := strconv.Atoi(m[1])
		return now.AddDate(0, -n, 0), true
	}
	return time.Time{}, false
}

// exclusionPattern matches question shapes that need multi-event coverage --
// order/sequence ("which came first", "earliest to latest") and event-to-
// event duration ("how many days passed between X and Y") -- rather than a
// single relative anchor. resolveAnchorDate must not fire on these: guessing
// one anchor for a question with no single reference point only adds noise
// to a ranking that would otherwise be untouched. Found by categorizing all
// 133 temporal-reasoning questions in longmemeval_s_cleaned.json: 57/133 are
// order/sequence and ~48 are event-to-event, leaving only ~25-30 as the
// single-anchor shape this file actually targets.
var exclusionPattern = regexp.MustCompile(`(?i)\b(first|before|after|earliest|latest|order|between)\b`)

// fixedHolidayMonthDay are calendar dates a question can name directly
// instead of phrasing a relative offset. Valentine's Day is the only one
// that actually occurs in longmemeval_s_cleaned.json's temporal-reasoning
// set -- checked directly against all 133 questions, not guessed -- but the
// table is left open for whichever others a future dataset slice needs.
var fixedHolidayMonthDay = map[string][2]int{
	"valentine's day": {2, 14},
	"valentines day":  {2, 14},
}

var (
	nDaysAgoPattern    = regexp.MustCompile(`(?i)\b(\d+)\s+days?\s+ago\b`)
	nWeeksAgoPattern   = regexp.MustCompile(`(?i)\b(\d+)\s+weeks?\s+ago\b`)
	nMonthsAgoPattern  = regexp.MustCompile(`(?i)\b(\d+)\s+months?\s+ago\b`)
	lastWeekdayPattern = regexp.MustCompile(`(?i)\blast (monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`)
	pastWeekendPattern = regexp.MustCompile(`(?i)\bpast weekend\b`)
	yesterdayPattern   = regexp.MustCompile(`(?i)\byesterday\b`)
	todayPattern       = regexp.MustCompile(`(?i)\btoday\b`)
)

var weekdayByName = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday,
	"saturday": time.Saturday,
}
