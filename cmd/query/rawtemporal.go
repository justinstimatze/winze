package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// parseTemporalRange looks for a small, deterministic set of date/time
// expressions in query and converts the match to a [start, end) UTC window.
// ok is false when nothing matches, so the temporal channel can contribute
// nothing to a query rather than guessing -- the same posture lexRank and
// semRank take toward a doc they have no opinion on.
func parseTemporalRange(query string, now time.Time) (start, end time.Time, ok bool) {
	q := strings.ToLower(query)
	now = now.UTC()
	dayStart := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	monthStart := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	today := dayStart(now)

	switch {
	case strings.Contains(q, "today"):
		return today, today.AddDate(0, 0, 1), true
	case strings.Contains(q, "yesterday"):
		return today.AddDate(0, 0, -1), today, true
	case strings.Contains(q, "this week"):
		s := today.AddDate(0, 0, -int(today.Weekday()))
		return s, s.AddDate(0, 0, 7), true
	case strings.Contains(q, "last week"):
		thisWeek := today.AddDate(0, 0, -int(today.Weekday()))
		return thisWeek.AddDate(0, 0, -7), thisWeek, true
	case strings.Contains(q, "this month"):
		s := monthStart(today)
		return s, s.AddDate(0, 1, 0), true
	case strings.Contains(q, "last month"):
		thisMonth := monthStart(today)
		return thisMonth.AddDate(0, -1, 0), thisMonth, true
	}

	if m := agoPattern.FindStringSubmatch(q); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			switch {
			case strings.HasPrefix(m[2], "day"):
				s := today.AddDate(0, 0, -n)
				return s, s.AddDate(0, 0, 1), true
			case strings.HasPrefix(m[2], "week"):
				s := today.AddDate(0, 0, -7*n)
				return s, s.AddDate(0, 0, 7), true
			case strings.HasPrefix(m[2], "month"):
				s := monthStart(today).AddDate(0, -n, 0)
				return s, s.AddDate(0, 1, 0), true
			}
		}
	}

	if m := isoDatePattern.FindString(q); m != "" {
		if t, err := time.ParseInLocation("2006-01-02", m, time.UTC); err == nil {
			return t, t.AddDate(0, 0, 1), true
		}
	}

	return time.Time{}, time.Time{}, false
}

// temporalRank ranks raw docs whose Time falls inside [start, end), most
// recent first. A doc outside the window, or with an unparseable Time, is
// left out of the map entirely -- the same "no signal" convention lexRank
// and semRank already use for a doc neither channel has an opinion on.
func temporalRank(docs []rawDoc, start, end time.Time) map[int]int {
	type cand struct {
		idx int
		t   time.Time
	}
	var cands []cand
	for i, d := range docs {
		t, err := time.Parse(time.RFC3339, d.Time)
		if err != nil {
			continue
		}
		t = t.UTC()
		if t.Before(start) || !t.Before(end) {
			continue
		}
		cands = append(cands, cand{i, t})
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].t.After(cands[j].t) })
	rank := make(map[int]int, len(cands))
	for i, c := range cands {
		rank[c.idx] = i + 1
	}
	return rank
}

var (
	agoPattern     = regexp.MustCompile(`(\d+)\s+(day|days|week|weeks|month|months)\s+ago`)
	isoDatePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
)
