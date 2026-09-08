package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSurfaceFormsCapEnforcement(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{VarName: "A", Name: "A", Brief: "brief a"},
		{VarName: "B", Name: "B", Brief: "brief b"},
		{VarName: "C", Name: "C", Brief: "brief c"},
	}}
	cache := &surfaceFormCache{m: map[string][]string{
		surfaceFormKey("A", "brief a"): {"cached a"},
		surfaceFormKey("B", "brief b"): {"cached b"},
	}}
	calls := 0
	generate := func(e entityRecord) ([]string, error) {
		calls++
		return []string{"generated " + e.Name}, nil
	}

	out := buildSurfaceForms(kb, cache, 0, generate)
	if calls != 0 {
		t.Fatalf("expected 0 generate calls against a zero cap, got %d", calls)
	}
	if len(out) != 2 || out[0][0] != "cached a" || out[1][0] != "cached b" {
		t.Fatalf("expected both cached entities present, got %v", out)
	}

	calls = 0
	out = buildSurfaceForms(kb, cache, 1, generate)
	if calls != 1 {
		t.Fatalf("expected exactly 1 generate call, got %d", calls)
	}
	if len(out) != 3 {
		t.Fatalf("expected all 3 entities present after generation, got %v", out)
	}
}

func TestBuildSurfaceFormsSkipsOnGenerateError(t *testing.T) {
	kb := &kbIndex{Entities: []entityRecord{
		{VarName: "A", Name: "A", Brief: "brief a"},
	}}
	cache := &surfaceFormCache{m: map[string][]string{}}
	generate := func(e entityRecord) ([]string, error) {
		return nil, fmt.Errorf("boom")
	}
	out := buildSurfaceForms(kb, cache, 10, generate)
	if len(out) != 0 {
		t.Fatalf("expected no entry on generate error, got %v", out)
	}
}

func TestParseSurfaceFormsResponseRejectsEmptyArray(t *testing.T) {
	if _, err := parseSurfaceFormsResponse("[]"); err == nil {
		t.Fatal("expected an error for an empty array")
	}
}

func TestParseSurfaceFormsResponseRejectsNonJSON(t *testing.T) {
	if _, err := parseSurfaceFormsResponse("not json at all"); err == nil {
		t.Fatal("expected an error for non-JSON input")
	}
}

func TestParseSurfaceFormsResponseStripsMarkdownFence(t *testing.T) {
	forms, err := parseSurfaceFormsResponse("```json\n[\"what happened\", \"why\"]\n```")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forms) != 2 || forms[0] != "what happened" {
		t.Fatalf("got %v", forms)
	}
}

func TestSurfaceFormCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := loadSurfaceFormCache(dir)
	c.m[surfaceFormKey("Name", "Brief")] = []string{"what happened", "why did this happen"}
	c.dirty = true
	c.save()

	if _, err := os.Stat(filepath.Join(dir, embedCacheDir, surfaceFormsCacheFile)); err != nil {
		t.Fatalf("cache file not written: %v", err)
	}
	reloaded := loadSurfaceFormCache(dir)
	v, ok := reloaded.m[surfaceFormKey("Name", "Brief")]
	if !ok || len(v) != 2 || v[0] != "what happened" {
		t.Fatalf("cache round-trip failed: got %v ok=%v", v, ok)
	}
}

func TestSurfaceFormsEnabled(t *testing.T) {
	if surfaceFormsEnabled() {
		t.Fatal("expected disabled when WINZE_SURFACE_FORMS is unset")
	}
	t.Setenv("WINZE_SURFACE_FORMS", "1")
	if !surfaceFormsEnabled() {
		t.Fatal("expected enabled when WINZE_SURFACE_FORMS is set")
	}
}

func TestSurfaceFormsMaxCallsDefaultAndOverride(t *testing.T) {
	if got := surfaceFormsMaxCalls(); got != 20 {
		t.Fatalf("default: got %d, want 20", got)
	}
	t.Setenv("WINZE_SURFACE_FORMS_MAX_CALLS", "5")
	if got := surfaceFormsMaxCalls(); got != 5 {
		t.Fatalf("override: got %d, want 5", got)
	}
	t.Setenv("WINZE_SURFACE_FORMS_MAX_CALLS", "not-a-number")
	if got := surfaceFormsMaxCalls(); got != 20 {
		t.Fatalf("invalid override: got %d, want fallback 20", got)
	}
	t.Setenv("WINZE_SURFACE_FORMS_MAX_CALLS", "0")
	if got := surfaceFormsMaxCalls(); got != 20 {
		t.Fatalf("zero override: got %d, want fallback 20", got)
	}
}
