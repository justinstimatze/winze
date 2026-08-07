package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// Extraction through the Message Batches API, at half price.
//
// Extraction is 97% of a run's spend and every call is independent, which is
// exactly what the batch endpoint is for: it trades latency (asynchronous, up
// to 24h, usually far less) for a 50% discount. A benchmark has no latency
// requirement at all, so this is a straight halving with no quality effect —
// the model, the prompts and the sampling parameters are identical.
//
// The design keeps the rest of the pipeline untouched. This runs as a PRE-PASS
// that fills the extraction cache, then the normal per-question flow executes
// against a fully warm cache. Nothing in runQuestion, runAll or the report
// knows a batch happened. That matters because the cache is content-keyed:
// warming it out of band is indistinguishable from having run before.
//
// On the longmemeval_s haystack the arithmetic is 18,464 distinct sessions,
// roughly $138 of extraction at list price and roughly $69 through here.

// batchJob is one session awaiting extraction, tagged with the cache key its
// result will be written under.
type batchJob struct {
	key    string // sha256(lensVersion + model + body) — the cache filename
	body   string // the rendered session
	system string // lensSystem on the first round, lensRetrySystem on the second
}

// collectBatchJobs walks every question's sessions and returns the ones the
// cache does not already hold, deduplicated by key.
//
// Deduplication is the reason this is worth doing as a pre-pass rather than
// per question. Haystack sessions are shared: 23,867 session slots across the
// 500 questions collapse to 18,464 distinct bodies, so a naive per-question
// submission would pay for 23% more extraction than exists.
func collectBatchJobs(cacheDir string, questions []Question) []batchJob {
	seen := map[string]bool{}
	var jobs []batchJob
	for _, q := range questions {
		for _, sess := range q.HaystackSessions {
			body := renderSession(sess)
			h := sha256.Sum256([]byte(lensVersion + "\x00" + string(anthropic.ModelClaudeHaiku4_5) + "\x00" + body))
			key := hex.EncodeToString(h[:])
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, err := os.Stat(filepath.Join(cacheDir, key+".json")); err == nil {
				continue // already extracted, by this run or an earlier one
			}
			jobs = append(jobs, batchJob{key: key, body: body, system: lensSystem})
		}
	}
	return jobs
}

// runBatch submits jobs, waits for the batch to end, and writes each result to
// the extraction cache. Returns the keys whose extraction came back empty,
// which is what the retry round needs.
func runBatch(ctx context.Context, client anthropic.Client, cacheDir string, jobs []batchJob, label string) ([]batchJob, error) {
	if len(jobs) == 0 {
		return nil, nil
	}
	byKey := make(map[string]batchJob, len(jobs))
	reqs := make([]anthropic.MessageBatchNewParamsRequest, 0, len(jobs))
	for _, j := range jobs {
		byKey[j.key] = j
		reqs = append(reqs, anthropic.MessageBatchNewParamsRequest{
			// custom_id round-trips the cache key, so a result needs no
			// positional correspondence with the request list — results come
			// back in arbitrary order and this is what re-associates them.
			CustomID: j.key,
			Params: anthropic.MessageBatchNewParamsRequestParams{
				Model:       anthropic.ModelClaudeHaiku4_5,
				MaxTokens:   4096,
				Temperature: anthropic.Float(0),
				System: []anthropic.TextBlockParam{{
					Text:         j.system,
					CacheControl: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h},
				}},
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock("<session>\n" + j.body + "\n</session>")),
				},
			},
		})
	}

	fmt.Fprintf(os.Stderr, "  [batch] %s: submitting %d extractions\n", label, len(reqs))
	batch, err := client.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{Requests: reqs})
	if err != nil {
		return nil, fmt.Errorf("batch submit: %w", err)
	}

	// Poll rather than stream: the batch endpoint has no push, and a benchmark
	// run has nothing better to do while it waits.
	for {
		b, err := client.Messages.Batches.Get(ctx, batch.ID)
		if err != nil {
			return nil, fmt.Errorf("batch poll: %w", err)
		}
		c := b.RequestCounts
		fmt.Fprintf(os.Stderr, "  [batch] %s %s: %d succeeded, %d errored, %d processing\n",
			label, b.ProcessingStatus, c.Succeeded, c.Errored, c.Processing)
		if b.ProcessingStatus == anthropic.MessageBatchProcessingStatusEnded {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(30 * time.Second):
		}
	}

	var empty []batchJob
	stream := client.Messages.Batches.ResultsStreaming(ctx, batch.ID)
	for stream.Next() {
		res := stream.Current()
		job, ok := byKey[res.CustomID]
		if !ok {
			continue
		}
		succeeded := res.Result.AsSucceeded()
		var text string
		for _, block := range succeeded.Message.Content {
			if block.Type == "text" {
				text = block.Text
				break
			}
		}
		out := lensResult{
			Facts:     parseFacts(text),
			Truncated: succeeded.Message.StopReason == "max_tokens",
			Retried:   job.system == lensRetrySystem,
		}
		// An empty first round is not written to the cache: the retry round
		// gets to replace it, and caching the empty would make that
		// impossible on any later run.
		if len(out.Facts) == 0 && !out.Truncated && job.system == lensSystem {
			empty = append(empty, batchJob{key: job.key, body: job.body, system: lensRetrySystem})
			continue
		}
		writeCacheEntry(cacheDir, job.key, out)
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("batch results: %w", err)
	}
	return empty, nil
}

// writeCacheEntry stores one extraction under its content key, through a temp
// file and a rename so a concurrent reader never sees a partial file.
func writeCacheEntry(cacheDir, key string, res lensResult) {
	b, err := json.Marshal(res)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(cacheDir, key+".*.tmp")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr == nil && cerr == nil {
		_ = os.Rename(tmp.Name(), filepath.Join(cacheDir, key+".json"))
		return
	}
	_ = os.Remove(tmp.Name())
}

// warmCacheByBatch extracts every uncached session through the batch API, in
// two rounds so the empty-first-pass retry still happens.
//
// After this returns, the ordinary pipeline runs with every session served
// from disk — which is also why a batch run and a live run are comparable:
// they produce the same cache entries from the same prompts, and only the
// billing differs.
func warmCacheByBatch(ctx context.Context, client anthropic.Client, cacheDir string, questions []Question) error {
	jobs := collectBatchJobs(cacheDir, questions)
	if len(jobs) == 0 {
		fmt.Fprintln(os.Stderr, "  [batch] every session already cached, nothing to submit")
		return nil
	}
	empty, err := runBatch(ctx, client, cacheDir, jobs, "pass 1")
	if err != nil {
		return err
	}
	if len(empty) == 0 {
		return nil
	}
	fmt.Fprintf(os.Stderr, "  [batch] %d sessions returned NO_FACTS, retrying with the second framing\n", len(empty))
	leftover, err := runBatch(ctx, client, cacheDir, empty, "pass 2")
	if err != nil {
		return err
	}
	// A session both passes decline is a genuine empty extraction. Cache it so
	// the next run does not pay to rediscover that.
	for _, j := range leftover {
		writeCacheEntry(cacheDir, j.key, lensResult{})
	}
	return nil
}
