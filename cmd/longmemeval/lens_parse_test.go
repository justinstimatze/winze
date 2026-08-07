package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The KIND whitelist in parseFacts is load-bearing and was silently wrong for
// three lens versions. v3 added assistant_stated to the prompt as the label
// separating what winze said from what the user said; the whitelist was not
// widened to match, so every assistant fact arrived as stated_fact and the
// answerer was told the user had said things the assistant said — on exactly
// the question type that asks which was which. The symptom was a flat 5/10 on
// single-session-assistant across k=15, 30 and 60, which reads like a missing
// fact and was a mislabelled one.
//
// Nothing about that failure is visible from the outside, so it gets a test.
func TestParseFactsKeepsAssistantStated(t *testing.T) {
	in := "wfh_job_7\tTranscriptionist\tassistant_stated\tI suggested transcriptionist as the seventh option."
	got := parseFacts(in)
	if len(got) != 1 {
		t.Fatalf("want 1 fact, got %d", len(got))
	}
	if got[0].Kind != "assistant_stated" {
		t.Errorf("Kind = %q, want assistant_stated — the whitelist dropped it and the fact now reads as the user's own words", got[0].Kind)
	}
	if got[0].Attribute != "wfh_job_7" || got[0].Value != "Transcriptionist" {
		t.Errorf("got %+v", got[0])
	}
}

func TestParseFactsKindHandling(t *testing.T) {
	for _, tc := range []struct {
		name, kind, want string
	}{
		{"stated_fact survives", "stated_fact", "stated_fact"},
		{"preference survives", "preference", "preference"},
		{"update survives", "update", "update"},
		{"assistant_stated survives", "assistant_stated", "assistant_stated"},
		// An unknown label is coerced rather than dropped: losing the fact is
		// worse than mislabelling it, since the answerer can still use the
		// value. This is a deliberate choice, not an oversight, so pin it.
		{"unknown coerces to stated_fact", "invented_kind", "stated_fact"},
		{"empty coerces to stated_fact", "", "stated_fact"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseFacts("attr\tvalue\t" + tc.kind + "\tquote")
			if len(got) != 1 {
				t.Fatalf("want 1 fact, got %d", len(got))
			}
			if got[0].Kind != tc.want {
				t.Errorf("Kind = %q, want %q", got[0].Kind, tc.want)
			}
		})
	}
}

// A quote containing a tab must not truncate the quote. parseFacts joins
// parts[3:] rather than taking parts[3], and that join is the only thing
// standing between a tabbed source sentence and a quote cut at the tab —
// which would then sit in the store presented as verbatim.
func TestParseFactsQuoteSurvivesEmbeddedTabs(t *testing.T) {
	got := parseFacts("attr\tvalue\tstated_fact\tfirst half\tsecond half")
	if len(got) != 1 {
		t.Fatalf("want 1 fact, got %d", len(got))
	}
	if got[0].Quote != "first half second half" {
		t.Errorf("Quote = %q, want the whole quote — a tab inside the source text truncated it", got[0].Quote)
	}
}

func TestParseFactsRejectsIncompleteLines(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"NO_FACTS", "NO_FACTS"},
		{"NO_FACTS with trailing prose", "NO_FACTS — the session was small talk"},
		{"empty completion", ""},
		{"too few fields", "attr\tvalue\tstated_fact"},
		{"empty attribute", "\tvalue\tstated_fact\tquote"},
		{"empty value", "attr\t\tstated_fact\tquote"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseFacts(tc.in); len(got) != 0 {
				t.Errorf("want no facts, got %d: %+v", len(got), got)
			}
		})
	}
}

// The extraction cache widened from a bare []ExtractedFact to a lensResult
// struct in v7. Two things have to hold for that not to corrupt a warm run:
// a pre-v7 array must fail to decode (so it falls through to a fresh call
// rather than silently reporting Truncated=false), and a legitimately empty
// extraction must still round-trip — otherwise every starved session
// re-extracts on every warm run and the cache quietly stops working for
// exactly the sessions that cost the most to diagnose.
func TestLensResultCacheFormat(t *testing.T) {
	dir := t.TempDir()

	t.Run("pre-v7 array does not decode as lensResult", func(t *testing.T) {
		old, err := json.Marshal([]ExtractedFact{{Attribute: "a", Value: "b"}})
		if err != nil {
			t.Fatal(err)
		}
		var res lensResult
		if err := json.Unmarshal(old, &res); err == nil {
			t.Error("a bare fact array decoded into lensResult; a stale cache entry would be served as untruncated")
		}
	})

	t.Run("empty extraction round-trips", func(t *testing.T) {
		path := filepath.Join(dir, "empty.json")
		b, err := json.Marshal(lensResult{Facts: nil, Retried: true})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var res lensResult
		if err := json.Unmarshal(raw, &res); err != nil {
			t.Fatalf("a NO_FACTS result must survive the cache, got %v", err)
		}
		if !res.Retried {
			t.Error("Retried was lost through the cache")
		}
	})
}
