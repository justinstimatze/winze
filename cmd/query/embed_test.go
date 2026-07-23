package main

import (
	"testing"
	"unicode/utf8"
)

func TestTruncateRunes(t *testing.T) {
	// ASCII: exact byte cut.
	if got := truncateRunes("hello world", 5); got != "hello" {
		t.Errorf("ascii = %q, want %q", got, "hello")
	}
	// Under cap: unchanged.
	if got := truncateRunes("short", 100); got != "short" {
		t.Errorf("under-cap = %q, want %q", got, "short")
	}
	// Multibyte: a cut inside the 3-byte '→' must back off to a rune boundary,
	// never emitting invalid UTF-8.
	s := "a→b→c" // '→' is 3 bytes each
	for n := 1; n <= len(s); n++ {
		got := truncateRunes(s, n)
		if !utf8.ValidString(got) {
			t.Errorf("truncateRunes(%q,%d) = %q is invalid UTF-8", s, n, got)
		}
		if len(got) > n {
			t.Errorf("truncateRunes(%q,%d) len %d exceeds cap", s, n, len(got))
		}
	}
}
