package main

import (
	"testing"
	"time"
)

const testQuestionDate = "2023/04/22 (Sat) 10:00"

func TestDateProximityRankFactsOrdersByDistance(t *testing.T) {
	facts := []Fact{
		{Attribute: "a", Date: "2023/04/10 (Mon) 12:00"}, // 12 days away
		{Attribute: "b", Date: "2023/04/21 (Fri) 12:00"}, // 1 day away
		{Attribute: "c", Date: "not a date"},             // unparseable, must be skipped
		{Attribute: "d", Date: "2023/04/22 (Sat) 12:00"}, // same day
	}
	anchor, err := time.Parse(factDateLayout, testQuestionDate)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}
	rank := dateProximityRankFacts(facts, anchor)
	if _, ok := rank[2]; ok {
		t.Errorf("fact with unparseable date got a rank, want excluded")
	}
	if rank[3] != 1 {
		t.Errorf("same-day fact rank = %d, want 1 (closest)", rank[3])
	}
	if rank[1] != 2 {
		t.Errorf("1-day-away fact rank = %d, want 2", rank[1])
	}
	if rank[0] != 3 {
		t.Errorf("12-days-away fact rank = %d, want 3 (furthest)", rank[0])
	}
}

func TestFuseRankMapsThreeArgAddsSignal(t *testing.T) {
	// term and sem both give fact 0 a slight edge (rank 1 vs 2); date gives
	// fact 1 a large edge (rank 1 vs 5). Hand-verified against rrfK=60: fact0
	// scores 2/61+1/65=0.048172, fact1 scores 2/62+1/61=0.048651 -- the large
	// single-channel edge outweighs two small ones, so fact 1 wins overall.
	term := map[int]int{0: 1, 1: 2}
	sem := map[int]int{0: 1, 1: 2}
	date := map[int]int{0: 5, 1: 1}
	got := fuseRankMaps(term, sem, date)
	if got[0] != 1 {
		t.Errorf("fuseRankMaps(term, sem, date)[0] = %d, want 1 (date channel's large edge outweighs term/sem's small one)", got[0])
	}
}

func TestFuseRankMapsTwoArgUnchanged(t *testing.T) {
	a := map[int]int{0: 1, 1: 2}
	b := map[int]int{0: 2, 1: 1}
	got := fuseRankMaps(a, b)
	want := []int{0, 1} // symmetric RRF scores, tie broken by index
	if len(got) != len(want) {
		t.Fatalf("fuseRankMaps(a, b) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fuseRankMaps(a, b)[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestResolveAnchorDateBadQuestionDate(t *testing.T) {
	if _, ok := resolveAnchorDate("What happened 3 days ago?", "not a date"); ok {
		t.Errorf("resolveAnchorDate resolved an anchor from an unparseable questionDate, want false")
	}
}

func TestResolveAnchorDateExcludesMultiEventShapes(t *testing.T) {
	questions := []string{
		"Which came first, the trip to Paris or the trip to Rome?",
		"What is the order of the concerts I attended, earliest to latest?",
		"How many days passed between the day I bought the racket and the day I received it?",
		"What did I do before my dentist appointment?",
		"What did I do after the museum visit?",
	}
	for _, q := range questions {
		if _, ok := resolveAnchorDate(q, testQuestionDate); ok {
			t.Errorf("resolveAnchorDate(%q) resolved an anchor, want false (multi-event shape)", q)
		}
	}
}

func TestResolveAnchorDateNoPatternMatch(t *testing.T) {
	if _, ok := resolveAnchorDate("What is my favorite color?", testQuestionDate); ok {
		t.Errorf("resolveAnchorDate matched a question with no relative-date language, want false")
	}
}

func TestResolveAnchorDateSingleAnchorPatterns(t *testing.T) {
	cases := []struct {
		name     string
		question string
		want     string // YYYY-MM-DD
	}{
		{"n days ago", "What did I do 3 days ago?", "2023-04-19"},
		{"n weeks ago", "How many weeks ago did I start using the app? I mean what happened 2 weeks ago?", "2023-04-08"},
		{"n months ago", "What happened 1 month ago?", "2023-03-22"},
		{"yesterday", "What did I eat yesterday?", "2023-04-21"},
		{"today", "What am I doing today?", "2023-04-22"},
		{"last weekday", "Who did I see last Wednesday?", "2023-04-19"},
		{"past weekend", "What did I do this past weekend?", "2023-04-22"},
		{"valentines day", "What airline did I fly on Valentine's day?", "2023-02-14"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := resolveAnchorDate(c.question, testQuestionDate)
			if !ok {
				t.Fatalf("resolveAnchorDate(%q) returned false, want an anchor", c.question)
			}
			if got.Format("2006-01-02") != c.want {
				t.Errorf("resolveAnchorDate(%q) = %s, want %s", c.question, got.Format("2006-01-02"), c.want)
			}
		})
	}
}
