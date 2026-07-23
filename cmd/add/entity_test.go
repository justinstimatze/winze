package main

import "testing"

func TestDeriveNames(t *testing.T) {
	cases := []struct {
		brief       string
		wantVar     string
		wantDisplay string
	}{
		{
			// The motivating case: leading "The" dropped, stops at the colon
			// (never reaches "winze-mem" → the old "Winzemem" fragment).
			brief:       "The winze-memory agentic interface: winze-mem binary exposes an MCP.",
			wantVar:     "WinzememoryAgenticInterface",
			wantDisplay: "Winze-memory agentic interface",
		},
		{
			// Colon deep in the sentence: title segment is everything before it,
			// capped at four content words.
			brief:       "Anthropic cache_control silently no-ops below the minimum: sizes matter.",
			wantVar:     "AnthropicCachecontrolSilentlyNoops",
			wantDisplay: "Anthropic cache_control silently no-ops",
		},
		{
			// No boundary punctuation: first four content words, stopwords out.
			brief:       "Justin prefers terse responses and pushes back on overengineering",
			wantVar:     "JustinPrefersTerseResponses",
			wantDisplay: "Justin prefers terse responses",
		},
		{
			// Nothing usable → timestamped fallback (var starts with Note).
			brief:       "…",
			wantVar:     "", // checked by prefix below
			wantDisplay: "",
		},
	}
	for _, c := range cases {
		gotVar, gotDisplay := deriveNames(c.brief)
		if c.wantVar == "" {
			if len(gotVar) < 4 || gotVar[:4] != "Note" {
				t.Errorf("deriveNames(%q) fallback var = %q, want Note* prefix", c.brief, gotVar)
			}
			continue
		}
		if gotVar != c.wantVar {
			t.Errorf("deriveNames(%q) var = %q, want %q", c.brief, gotVar, c.wantVar)
		}
		if gotDisplay != c.wantDisplay {
			t.Errorf("deriveNames(%q) display = %q, want %q", c.brief, gotDisplay, c.wantDisplay)
		}
	}
}
