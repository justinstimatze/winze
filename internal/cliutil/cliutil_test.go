package cliutil

import (
	"reflect"
	"testing"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		in, want string
		max      int
	}{
		{"short", "short", 20},                          // under max: unchanged
		{"line one\nline two", "line one line two", 40}, // newlines collapse to spaces
		{"abcdefghij", "abcde…", 5},                     // cut + unicode ellipsis
	}
	for _, c := range cases {
		if got := Truncate(c.in, c.max); got != c.want {
			t.Errorf("Truncate(%q,%d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

// Rune-safety: cutting in the middle of a multibyte rune must back off to a
// rune boundary, never emitting invalid UTF-8 (the byte-slicing the old copies
// did could split a rune).
func TestTruncateRuneSafe(t *testing.T) {
	// "aé" — 'é' is 2 bytes (0xC3 0xA9). max=2 would split it; expect back-off.
	got := Truncate("aément", 2)
	if got != "a…" {
		t.Errorf("Truncate rune-split = %q, want %q", got, "a…")
	}
	for i := 0; i < len(got); {
		if got[i] >= 0x80 && got[i] < 0xC0 { // stray continuation byte
			t.Fatalf("Truncate emitted invalid UTF-8: %q", got)
		}
		i++
	}
}

func TestSortedKeys(t *testing.T) {
	got := SortedKeys(map[string]int{"c": 1, "a": 2, "b": 3})
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("SortedKeys = %v, want %v", got, want)
	}
	// generic over value type
	got2 := SortedKeys(map[string]string{"z": "x", "y": "w"})
	if want := []string{"y", "z"}; !reflect.DeepEqual(got2, want) {
		t.Errorf("SortedKeys[string] = %v, want %v", got2, want)
	}
}
