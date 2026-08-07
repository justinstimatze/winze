package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// collectBatchJobs is where a batch run either saves money or wastes it, and
// it is pure — no client, no network. The two things it must get right are
// deduplication and cache-awareness, and both are invisible at the call site:
// getting either wrong still produces a working run, just a more expensive one.
func TestCollectBatchJobsDeduplicatesSharedSessions(t *testing.T) {
	dir := t.TempDir()
	shared := []Turn{{Role: "user", Content: "shared session content"}}
	a := []Turn{{Role: "user", Content: "unique to question A"}}
	b := []Turn{{Role: "user", Content: "unique to question B"}}

	questions := []Question{
		{QuestionID: "qa", HaystackSessions: [][]Turn{shared, a}},
		{QuestionID: "qb", HaystackSessions: [][]Turn{shared, b}},
		{QuestionID: "qc", HaystackSessions: [][]Turn{shared}},
	}
	jobs := collectBatchJobs(dir, questions)
	// Five session slots, three distinct bodies. On the real haystack this
	// ratio is 23,867 slots to 18,464 bodies, so submitting per slot would buy
	// 23% more extraction than exists.
	if len(jobs) != 3 {
		t.Fatalf("jobs = %d, want 3 — shared sessions were submitted more than once", len(jobs))
	}
	seen := map[string]bool{}
	for _, j := range jobs {
		if seen[j.key] {
			t.Errorf("duplicate key %s in the job list", j.key)
		}
		seen[j.key] = true
		if j.system != lensSystem {
			t.Errorf("round-one job carries the retry prompt")
		}
	}
}

func TestCollectBatchJobsSkipsCachedSessions(t *testing.T) {
	dir := t.TempDir()
	questions := []Question{{
		QuestionID: "q",
		HaystackSessions: [][]Turn{
			{{Role: "user", Content: "session one"}},
			{{Role: "user", Content: "session two"}},
		},
	}}

	all := collectBatchJobs(dir, questions)
	if len(all) != 2 {
		t.Fatalf("cold jobs = %d, want 2", len(all))
	}

	// Cache one of them the way the live path would, then re-collect.
	writeCacheEntry(dir, all[0].key, lensResult{Facts: []ExtractedFact{{Attribute: "a", Value: "b"}}})
	warm := collectBatchJobs(dir, questions)
	if len(warm) != 1 {
		t.Fatalf("warm jobs = %d, want 1 — a cached session was resubmitted", len(warm))
	}
	if warm[0].key == all[0].key {
		t.Error("the cached session is the one that got resubmitted")
	}

	// A cached EMPTY extraction must also count as cached. Both lens passes
	// declined that session; resubmitting it every run pays repeatedly to
	// rediscover the same nothing.
	writeCacheEntry(dir, warm[0].key, lensResult{})
	if got := collectBatchJobs(dir, questions); len(got) != 0 {
		t.Errorf("jobs = %d, want 0 — a cached empty extraction was resubmitted", len(got))
	}
}

// The batch path and the live path write the same cache, so an entry from one
// must be readable by the other. If they diverge, a batch run silently
// re-extracts everything on the next live run and the discount is cancelled.
func TestBatchCacheEntryIsReadableByTheLivePath(t *testing.T) {
	dir := t.TempDir()
	key := "deadbeef"
	want := lensResult{
		Facts:     []ExtractedFact{{Attribute: "wfh_job_7", Value: "Transcriptionist", Kind: "assistant_stated", Quote: "q"}},
		Truncated: true,
		Retried:   true,
	}
	writeCacheEntry(dir, key, want)

	b, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		t.Fatalf("batch wrote no cache file: %v", err)
	}
	var got lensResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("the live path cannot decode a batch-written entry: %v", err)
	}
	if len(got.Facts) != 1 || got.Facts[0].Kind != "assistant_stated" {
		t.Errorf("facts round-tripped wrong: %+v", got.Facts)
	}
	if !got.Truncated || !got.Retried {
		t.Errorf("Truncated/Retried lost through the batch cache write: %+v", got)
	}
}

// No .tmp files may survive a write, or the cache directory fills with debris
// across a run that submits tens of thousands of sessions.
func TestWriteCacheEntryLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		writeCacheEntry(dir, filepath.Base(t.Name())+string(rune('a'+i)), lensResult{})
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}
	if len(entries) != 20 {
		t.Errorf("wrote %d files, want 20", len(entries))
	}
}
