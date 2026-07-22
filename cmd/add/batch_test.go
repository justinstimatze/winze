package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadBatch_SkipsBlankLinesAndParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "b.jsonl")
	content := `{"to":"a.go","name":"C1","predicate":"Proposes","subject":"S","object":"O","quote":"q","origin":"src"}

{"to":"b.go","name":"C2","predicate":"IsCognitiveBias","subject":"S2","quote":"q2","origin":"src2","unary":true}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, err := readBatch(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("got %d specs, want 2 (blank line skipped)", len(specs))
	}
	if specs[0].Name != "C1" || specs[1].Name != "C2" {
		t.Errorf("unexpected names: %q, %q", specs[0].Name, specs[1].Name)
	}
	if !specs[1].Unary {
		t.Errorf("C2 should be unary")
	}
}

func TestReadBatch_MalformedLineErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "b.jsonl")
	if err := os.WriteFile(path, []byte("{\"name\":\"ok\"}\nnot json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readBatch(path)
	if err == nil {
		t.Fatal("expected error on malformed line")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should name the offending line, got %q", err.Error())
	}
}

func TestIngestedByOrDefault(t *testing.T) {
	if got := (claimSpec{}).ingestedByOrDefault(); got != "winze-add" {
		t.Errorf("empty IngestedBy should default to winze-add, got %q", got)
	}
	if got := (claimSpec{IngestedBy: "sensor"}).ingestedByOrDefault(); got != "sensor" {
		t.Errorf("explicit IngestedBy should pass through, got %q", got)
	}
}

// TestRunBatch_RevertsAllOnBadRecord verifies the all-or-nothing property:
// a batch containing one invalid record must not modify any file. Validation
// runs before the first write, so both target files stay byte-identical.
func TestRunBatch_RevertsAllOnBadRecord(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.go")
	bPath := filepath.Join(dir, "b.go")
	orig := "package winze\n"
	for _, p := range []string{aPath, bPath} {
		if err := os.WriteFile(p, []byte(orig), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	batchPath := filepath.Join(dir, "batch.jsonl")
	// Second record is invalid (missing --object on a binary predicate). The
	// batch must fail during validation, before either file is appended to.
	content := `{"to":"a.go","name":"C1","predicate":"Proposes","subject":"S","object":"O","quote":"q","origin":"src"}
{"to":"b.go","name":"C2","predicate":"Proposes","subject":"S2","quote":"q2","origin":"src2"}
`
	if err := os.WriteFile(batchPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	code := runBatch(batchPath, dir, false)
	if code == 0 {
		t.Fatal("expected non-zero exit on invalid record")
	}
	for _, p := range []string{aPath, bPath} {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != orig {
			t.Errorf("%s was modified despite batch failure:\n%s", filepath.Base(p), got)
		}
	}
}

// TestRunBatch_DryRunWritesNothing confirms --dry-run renders without touching
// files or invoking the build gate (so it needs no real corpus).
func TestRunBatch_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.go")
	orig := "package winze\n"
	if err := os.WriteFile(aPath, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	batchPath := filepath.Join(dir, "batch.jsonl")
	content := `{"to":"a.go","name":"C1","predicate":"Proposes","subject":"S","object":"O","quote":"q","origin":"src"}
`
	if err := os.WriteFile(batchPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runBatch(batchPath, dir, true); code != 0 {
		t.Fatalf("dry-run should succeed, got exit %d", code)
	}
	got, err := os.ReadFile(aPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != orig {
		t.Errorf("dry-run modified the file:\n%s", got)
	}
}
