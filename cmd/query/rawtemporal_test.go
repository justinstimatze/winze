package main

import (
	"testing"
	"time"
)

func TestParseTemporalRangeDaysAgo(t *testing.T) {
	start, end, ok := parseTemporalRange("the decision from 2 days ago", fixedNow)
	if !ok {
		t.Fatal("expected a match for \"2 days ago\"")
	}
	wantStart := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got [%v, %v), want [%v, %v)", start, end, wantStart, wantEnd)
	}
}

func TestParseTemporalRangeExplicitISODate(t *testing.T) {
	start, end, ok := parseTemporalRange("what about 2026-08-15", fixedNow)
	if !ok {
		t.Fatal("expected a match for an explicit ISO date")
	}
	wantStart := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantStart.AddDate(0, 0, 1)) {
		t.Fatalf("got [%v, %v), want a one-day window starting %v", start, end, wantStart)
	}
}

func TestParseTemporalRangeNoMatch(t *testing.T) {
	if _, _, ok := parseTemporalRange("what theories compete on consciousness", fixedNow); ok {
		t.Fatal("expected no temporal match for a plain content question")
	}
}

func TestParseTemporalRangeThisAndLastMonth(t *testing.T) {
	start, end, ok := parseTemporalRange("this month's numbers", fixedNow)
	if !ok {
		t.Fatal("expected a match for \"this month\"")
	}
	want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(want) || !end.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got [%v, %v), want [%v, 2026-10-01)", start, end, want)
	}

	start, end, ok = parseTemporalRange("last month's numbers", fixedNow)
	if !ok {
		t.Fatal("expected a match for \"last month\"")
	}
	want = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(want) || !end.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got [%v, %v), want [%v, 2026-09-01)", start, end, want)
	}
}

func TestParseTemporalRangeThisAndLastWeek(t *testing.T) {
	for _, phrase := range []string{"this week", "last week"} {
		start, end, ok := parseTemporalRange("notes from "+phrase, fixedNow)
		if !ok {
			t.Fatalf("expected a match for %q", phrase)
		}
		if start.Weekday() != time.Sunday {
			t.Fatalf("%q: start %v is not a Sunday", phrase, start)
		}
		if end.Sub(start) != 7*24*time.Hour {
			t.Fatalf("%q: window is %v, want 7 days", phrase, end.Sub(start))
		}
	}
	_, thisEnd, _ := parseTemporalRange("this week", fixedNow)
	if !fixedNow.Before(thisEnd) {
		t.Fatalf("\"this week\" should contain fixedNow, ends %v", thisEnd)
	}
	_, lastEnd, _ := parseTemporalRange("last week", fixedNow)
	if !lastEnd.Equal(thisEnd.AddDate(0, 0, -7)) {
		t.Fatalf("\"last week\" should end exactly 7 days before \"this week\" ends: %v vs %v", lastEnd, thisEnd)
	}
}

func TestParseTemporalRangeToday(t *testing.T) {
	start, end, ok := parseTemporalRange("what did we ship today", fixedNow)
	if !ok {
		t.Fatal("expected a match for \"today\"")
	}
	want := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	if !start.Equal(want) || !end.Equal(want.AddDate(0, 0, 1)) {
		t.Fatalf("got [%v, %v), want [%v, %v)", start, end, want, want.AddDate(0, 0, 1))
	}
}

func TestParseTemporalRangeWeeksAgo(t *testing.T) {
	start, end, ok := parseTemporalRange("1 week ago we discussed this", fixedNow)
	if !ok {
		t.Fatal("expected a match for \"1 week ago\"")
	}
	if end.Sub(start) != 7*24*time.Hour {
		t.Fatalf("window is %v, want 7 days", end.Sub(start))
	}
	if !start.Before(fixedNow.AddDate(0, 0, -6)) || !start.After(fixedNow.AddDate(0, 0, -8)) {
		t.Fatalf("start %v not ~1 week before %v", start, fixedNow)
	}
}

func TestParseTemporalRangeYesterday(t *testing.T) {
	start, end, ok := parseTemporalRange("what happened yesterday", fixedNow)
	if !ok {
		t.Fatal("expected a match for \"yesterday\"")
	}
	wantStart := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got [%v, %v), want [%v, %v)", start, end, wantStart, wantEnd)
	}
}

func TestRRFFuseRawEmpty(t *testing.T) {
	if got := rrfFuseRaw(map[int]int{}, map[int]int{}, map[int]int{}); len(got) != 0 {
		t.Fatalf("empty inputs should fuse to nothing, got %+v", got)
	}
}

// A doc favored by all three channels must beat one favored by only two.
func TestRRFFuseRawRewardsAgreementAcrossThreeChannels(t *testing.T) {
	lex := map[int]int{10: 1, 20: 2, 30: 3}
	sem := map[int]int{20: 1, 30: 2, 40: 3}
	tmp := map[int]int{20: 1, 10: 2}
	got := rrfFuseRaw(lex, sem, tmp)

	if got[0].idx != 20 {
		t.Fatalf("expected doc 20 (in all three lists) first, got %d (%+v)", got[0].idx, got)
	}
	if got[0].lex != 2 || got[0].sem != 1 || got[0].tmp != 1 {
		t.Fatalf("doc 20 ranks wrong: %+v", got[0])
	}
}

// A document present in only one channel still surfaces, scored from that channel.
func TestRRFFuseRawSingleListSurvives(t *testing.T) {
	got := rrfFuseRaw(map[int]int{99: 1}, map[int]int{}, map[int]int{})
	if len(got) != 1 || got[0].idx != 99 || got[0].sem != 0 || got[0].tmp != 0 {
		t.Fatalf("single-list doc lost: %+v", got)
	}
	want := 1.0 / float64(rrfK+1)
	if got[0].rrf != want {
		t.Fatalf("rrf = %v, want %v", got[0].rrf, want)
	}
}

func TestTemporalRankWindowsAndOrdersByRecency(t *testing.T) {
	docs := []rawDoc{
		{Time: "2026-09-05T10:00:00Z", Note: "in window, older"},
		{Time: "2026-09-05T18:00:00Z", Note: "in window, newer"},
		{Time: "2026-09-04T10:00:00Z", Note: "before window"},
		{Time: "2026-09-06T10:00:00Z", Note: "after window"},
		{Time: "not-a-time", Note: "unparseable"},
	}
	start := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)

	got := temporalRank(docs, start, end)
	if len(got) != 2 {
		t.Fatalf("expected 2 ranked docs, got %+v", got)
	}
	if got[1] != 1 {
		t.Fatalf("newer in-window doc (idx 1) should rank 1st, got %+v", got)
	}
	if got[0] != 2 {
		t.Fatalf("older in-window doc (idx 0) should rank 2nd, got %+v", got)
	}
	if _, ok := got[2]; ok {
		t.Fatalf("doc before the window should not be ranked: %+v", got)
	}
	if _, ok := got[3]; ok {
		t.Fatalf("doc after the window should not be ranked: %+v", got)
	}
	if _, ok := got[4]; ok {
		t.Fatalf("doc with an unparseable time should not be ranked: %+v", got)
	}
}

var fixedNow = time.Date(2026, 9, 7, 15, 4, 5, 0, time.UTC)
