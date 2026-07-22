package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
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
