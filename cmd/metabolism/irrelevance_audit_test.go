package main

import "testing"

func TestPickSpacedIndices(t *testing.T) {
	cases := []struct {
		name string
		idx  []int
		n    int
		want []int
	}{
		{"n >= len returns all", []int{1, 2, 3}, 10, []int{1, 2, 3}},
		{"n == len returns all", []int{1, 2, 3}, 3, []int{1, 2, 3}},
		{"evenly spaced from 10", []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 5, []int{0, 2, 4, 6, 8}},
		{"empty input", []int{}, 5, []int{}},
		{"zero n returns empty", []int{1, 2, 3}, 0, []int{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pickSpacedIndices(tc.idx, tc.n)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %d, want %d (full: %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

func TestPickSpacedIndices_Deterministic(t *testing.T) {
	idx := []int{5, 10, 15, 20, 25, 30, 35, 40}
	a := pickSpacedIndices(idx, 3)
	b := pickSpacedIndices(idx, 3)
	if len(a) != len(b) {
		t.Fatalf("non-deterministic length: %v vs %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("non-deterministic at [%d]: %d vs %d", i, a[i], b[i])
		}
	}
}
