package main

import (
	"math"
	"testing"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func TestComputeSetRecall(t *testing.T) {
	cases := []struct {
		name      string
		retrieved []string
		gold      []string
		want      float64
	}{
		{"empty gold", []string{"a"}, nil, 1.0},
		{"perfect recall", []string{"a", "b"}, []string{"a", "b"}, 1.0},
		{"partial recall", []string{"a", "c"}, []string{"a", "b"}, 0.5},
		{"zero recall", []string{"c", "d"}, []string{"a", "b"}, 0.0},
		{"empty retrieved", nil, []string{"a"}, 0.0},
		{"superset retrieved", []string{"a", "b", "c"}, []string{"a", "b"}, 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeSetRecall(tc.retrieved, tc.gold)
			if !approxEqual(got, tc.want) {
				t.Errorf("computeSetRecall = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestComputePrecision(t *testing.T) {
	cases := []struct {
		name      string
		retrieved []string
		gold      []string
		want      float64
	}{
		{"empty retrieved", nil, []string{"a"}, 0.0},
		{"perfect precision", []string{"a", "b"}, []string{"a", "b"}, 1.0},
		{"partial precision", []string{"a", "c"}, []string{"a", "b"}, 0.5},
		{"zero precision", []string{"c", "d"}, []string{"a", "b"}, 0.0},
		{"empty gold", []string{"a"}, nil, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computePrecision(tc.retrieved, tc.gold)
			if !approxEqual(got, tc.want) {
				t.Errorf("computePrecision = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestNormalizeResult(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want int
	}{
		{"nil input", nil, 0},
		{"empty strings filtered", []string{"", "  ", "a"}, 1},
		{"whitespace trimmed", []string{" hello ", "  world  "}, 2},
		{"all empty", []string{"", "", ""}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeResult(tc.in)
			if len(got) != tc.want {
				t.Errorf("normalizeResult returned %d items, want %d", len(got), tc.want)
			}
		})
	}
}

func TestMatchGold(t *testing.T) {
	cases := []struct {
		name      string
		retrieved []string
		gold      []string
		want      float64
	}{
		{"single numeric exact match", []string{"42", "10"}, []string{"42"}, 1.0},
		{"single numeric no match", []string{"10", "20"}, []string{"42"}, 0.0},
		{"single non-numeric uses recall", []string{"a", "b"}, []string{"a"}, 1.0},
		{"multi gold uses recall", []string{"a"}, []string{"a", "b"}, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchGold(tc.retrieved, tc.gold)
			if !approxEqual(got, tc.want) {
				t.Errorf("matchGold = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestIsNumericGold(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"42", true},
		{"0", true},
		{"123456", true},
		{"12a3", false},
		{"abc", false},
		{"", true}, // empty string has no non-digit chars
		{" 42", false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := isNumericGold(tc.input)
			if got != tc.want {
				t.Errorf("isNumericGold(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
