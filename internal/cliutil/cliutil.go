// Package cliutil holds small, type-agnostic helpers the winze command tools
// were each re-implementing (a calque scan found truncate copied four ways —
// three ASCII, one newline-collapsing with a unicode ellipsis — plus sortedKeys
// and isFlagSet duplicated across cmd/*). One definition each; consumers
// delegate, so the behavior can't drift between tools.
package cliutil

import (
	"flag"
	"sort"
	"unicode/utf8"
)

// Truncate shortens s to at most max bytes for display: newlines collapse to
// spaces and a unicode ellipsis marks the cut. Rune-safe — it backs off to a
// rune boundary so it never emits invalid UTF-8 (the byte-slicing the old
// copies did could split a multibyte rune). This is the superset of the four
// prior variants.
func Truncate(s string, max int) string {
	// collapse newlines so multi-line Briefs render on one line
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			out = append(out, ' ')
		} else {
			out = append(out, s[i])
		}
	}
	s = string(out)
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// SortedKeys returns the keys of m in ascending order. Generic over the value
// type — the prior copies were fixed to map[string]int.
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// IsFlagSet reports whether the named flag was explicitly set on the default
// command line (flag.CommandLine). Tools that use a custom FlagSet must check
// that set directly.
func IsFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
