package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// The embed() call itself needs a running ollama daemon, so it is exercised
// manually / in integration, not here. These cover the deterministic pieces:
// vector math, the content-addressed cache key, and cache persistence.

func TestNormalizeUnitLength(t *testing.T) {
	v := normalize([]float32{3, 4}) // |{3,4}| = 5 -> {0.6, 0.8}
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	if math.Abs(s-1) > 1e-6 {
		t.Fatalf("normalize not unit length: %v (sum of squares %v)", v, s)
	}
}

func TestDotOfNormalizedIsCosine(t *testing.T) {
	got := dot(normalize([]float32{1, 0}), normalize([]float32{1, 1})) // cos 45°
	if math.Abs(got-math.Sqrt2/2) > 1e-6 {
		t.Fatalf("dot of normalized = %v, want ~0.707", got)
	}
}

func TestEmbedKeyDeterministicAndDistinct(t *testing.T) {
	if embedKey("hello") != embedKey("hello") {
		t.Fatal("embedKey not deterministic")
	}
	if embedKey("hello") == embedKey("world") {
		t.Fatal("embedKey collision on distinct text")
	}
}

func TestEmbedText(t *testing.T) {
	if got := embedText(entityRecord{Name: "Apophenia", Brief: "pattern in noise"}); got != "Apophenia. pattern in noise" {
		t.Fatalf("embedText = %q", got)
	}
	if got := embedText(entityRecord{}); got != "." {
		t.Fatalf("empty entity embedText = %q, want \".\" (runSemantic skips it)", got)
	}
}

func TestVecCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := loadVecCache(dir)
	c.m[embedKey("x")] = []float32{0.1, 0.2}
	c.dirty = true
	c.save()

	if _, err := os.Stat(filepath.Join(dir, embedCacheDir, embedModel+".gob")); err != nil {
		t.Fatalf("cache file not written: %v", err)
	}
	reloaded := loadVecCache(dir)
	v, ok := reloaded.m[embedKey("x")]
	if !ok || len(v) != 2 || v[0] != 0.1 {
		t.Fatalf("cache round-trip failed: got %v ok=%v", v, ok)
	}
}

// --- segmenting ------------------------------------------------------------
//
// These exist because maxEmbedChars silently truncated: everything past the
// first 512 bytes of a Brief or a docs section was unreachable semantically
// while BM25 went on matching it, so the two halves of --hybrid answered
// different questions and neither said so. Measured 2026-08-07: 0 of 403 demo
// corpus briefs exceed the cap (which is why the old comment claiming tail
// loss was immaterial held), against 44 of 90 memory-store briefs and 89 of
// 120 docs sections.

func TestSegmentForEmbedShortTextIsOneUnchangedSegment(t *testing.T) {
	// Load-bearing for the cache, not just for tidiness. Entries at or under
	// the cap must keep the exact key they had before segmenting existed, or
	// every short entry in every store silently re-embeds.
	const short = "Apophenia. seeing pattern in noise"
	got := segmentForEmbed(short)
	if len(got) != 1 || got[0] != short {
		t.Fatalf("segmentForEmbed(short) = %q, want exactly [%q] so embedKey is unchanged", got, short)
	}
}

func TestSegmentForEmbedRespectsTheCap(t *testing.T) {
	text := strings.Repeat("alpha beta gamma delta ", 400) // ~9200 bytes
	segs := segmentForEmbed(text)
	if len(segs) < 2 {
		t.Fatalf("segments = %d, want the long text split", len(segs))
	}
	for i, s := range segs {
		if len(s) > maxEmbedChars {
			t.Errorf("segment %d is %d bytes, over the %d cap — ollama 500s past its window", i, len(s), maxEmbedChars)
		}
	}
}

func TestSegmentForEmbedLosesNoContent(t *testing.T) {
	// The whole point is reachability, so a segmenter that drops the tail is
	// worse than the truncation it replaces: it would look fixed.
	text := strings.Repeat("the quick brown fox jumps over the lazy dog ", 60)
	strip := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	if got, want := strip(strings.Join(segmentForEmbed(text), " ")), strip(text); got != want {
		t.Errorf("content lost or reordered: %d bytes out, %d in", len(got), len(want))
	}
}

func TestSegmentForEmbedKeepsRunesIntact(t *testing.T) {
	text := strings.Repeat("héllo wörld — naïve café ", 100) // multibyte throughout
	for i, s := range segmentForEmbed(text) {
		if !utf8.ValidString(s) {
			t.Errorf("segment %d is not valid UTF-8 — a rune was split", i)
		}
	}
}

func TestSegmentForEmbedTerminatesOnUnbrokenText(t *testing.T) {
	// No whitespace to back off to. The boundary search must still make
	// progress rather than loop forever looking for a break that is not there.
	done := make(chan []string, 1)
	go func() { done <- segmentForEmbed(strings.Repeat("x", maxEmbedChars*4)) }()
	select {
	case segs := <-done:
		if len(segs) < 4 {
			t.Errorf("segments = %d, want >= 4 for 4x the cap", len(segs))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("segmentForEmbed did not terminate on text with no whitespace")
	}
}

func TestBestCosineTakesTheMaxNotTheMean(t *testing.T) {
	// Max-pooling is the measured choice, not a convenience. A document whose
	// tail answers the query must not be dragged under by the head segments
	// that do not — averaging is exactly the dilution that made whole-document
	// embedding lose to head-only on head queries.
	q := normalize([]float32{1, 0})
	segs := [][]float32{
		normalize([]float32{0, 1}), // orthogonal
		normalize([]float32{0, 1}), // orthogonal
		normalize([]float32{1, 0}), // the tail segment that actually matches
	}
	got := bestCosine(q, segs)
	if math.Abs(got-1) > 1e-6 {
		t.Fatalf("bestCosine = %v, want ~1 (the matching segment); a mean would give ~0.33", got)
	}
	if bestCosine(q, nil) != 0 {
		t.Error("bestCosine(nil) must be 0, not a panic — an unembeddable entity is skipped, not ranked")
	}
}
