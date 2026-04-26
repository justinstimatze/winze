package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// reresolve — reclassify historical "irrelevant" sensor cycles under the
// current production prompt (main.go:1064), updating the log where the
// verdict flips. This is the retroactive companion to the 2026-04-18
// prompt tune: the tune affects fresh cycles immediately, but the 114
// existing "irrelevant" cycles stay stuck until we re-evaluate them.
//
// The --irrelevance-audit diagnostic measured 40% flip rate on a N=10
// sample under the tuned prompt (cycles 222, 238, 246, 254). Scaled,
// that's ~45 historical corroborations waiting to surface — the first
// organic non-100% Path A signal the sprint gate has been asking for.
//
// Mutates the log: updates Resolution, Evidence (marker), and ResolvedAt
// on re-resolved cycles. Evidence preserves the original verdict so a
// downstream reader can always see what changed.

// countEligibleIrrelevant returns the number of cycles in the log that
// match the reresolve eligibility filter (irrelevant sensor cycle with
// papers + optional snippet requirement + not already reresolved).
func countEligibleIrrelevant(cycles []Cycle, requireSnippet bool) int {
	n := 0
	for _, c := range cycles {
		if c.Resolution != "irrelevant" {
			continue
		}
		if c.PredictionType != "" && c.PredictionType != "structural_fragility" {
			continue
		}
		if len(c.Papers) == 0 {
			continue
		}
		if requireSnippet && !anyPaperHasSnippet(c.Papers) {
			continue
		}
		if strings.Contains(c.Evidence, "reresolved") {
			continue
		}
		n++
	}
	return n
}

// runReresolveIrrelevant filters the log for "irrelevant" sensor cycles
// whose papers carry at least one snippet (unless requireSnippet=false,
// which accepts title-only cases too), samples up to n of them (0=all),
// and reclassifies each through llmResolve. Verdicts that flip are
// written back with an evidence marker so the change is self-describing.
func runReresolveIrrelevant(dir string, n int, requireSnippet, dryRun, jsonOut bool) {
	ensureBudgetGuard(dir) // record actual LLM spend even outside --evolve
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	var targetIdx []int
	for i, c := range mlog.Cycles {
		if c.Resolution != "irrelevant" {
			continue
		}
		if c.PredictionType != "" && c.PredictionType != "structural_fragility" {
			continue
		}
		if len(c.Papers) == 0 {
			continue
		}
		if requireSnippet && !anyPaperHasSnippet(c.Papers) {
			continue
		}
		// Idempotency: skip cycles already reresolved in a prior run.
		if strings.Contains(c.Evidence, "reresolved") {
			continue
		}
		targetIdx = append(targetIdx, i)
	}
	if len(targetIdx) == 0 {
		fmt.Println("[reresolve] no eligible irrelevant cycles to re-evaluate")
		return
	}

	if n > 0 && n < len(targetIdx) {
		targetIdx = pickSpacedIndices(targetIdx, n)
	}

	if dryRun {
		fmt.Printf("[reresolve] dry-run: would reclassify %d of %d eligible cycles (log has %d cycles total)\n", len(targetIdx), countEligibleIrrelevant(mlog.Cycles, requireSnippet), len(mlog.Cycles))
		for _, i := range targetIdx {
			c := mlog.Cycles[i]
			fmt.Printf("  cycle %d: %s (%d papers)\n", i, c.Hypothesis, len(c.Papers))
		}
		return
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "[reresolve] ANTHROPIC_API_KEY not set — cannot reclassify")
		os.Exit(1)
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	today := time.Now().Format("2006-01-02")
	flipped := map[string]int{}
	kept := 0
	errors := 0

	type flipEntry struct {
		CycleIndex int    `json:"cycle_index"`
		Hypothesis string `json:"hypothesis"`
		From       string `json:"from"`
		To         string `json:"to"`
	}
	var flips []flipEntry

	fmt.Printf("[reresolve] reclassifying %d irrelevant cycles under current production prompt\n", len(targetIdx))
	for _, i := range targetIdx {
		c := mlog.Cycles[i]
		brief := lookupBrief(c.Hypothesis)
		verdict, err := llmResolve(client, c.Hypothesis, brief, c.Papers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  cycle %d %s: error: %v\n", i, c.Hypothesis, err)
			errors++
			continue
		}
		if verdict == "irrelevant" {
			kept++
			continue
		}
		fmt.Printf("  FLIP cycle %d [%s]: irrelevant → %s\n", i, c.Hypothesis, verdict)
		mlog.Cycles[i].Resolution = verdict
		mlog.Cycles[i].Evidence = fmt.Sprintf("reresolved %s under updated prompt (was: irrelevant)", today)
		mlog.Cycles[i].ResolvedAt = today
		flipped[verdict]++
		flips = append(flips, flipEntry{CycleIndex: i, Hypothesis: c.Hypothesis, From: "irrelevant", To: verdict})
	}

	if len(flips) > 0 {
		if err := saveLog(logPath, mlog); err != nil {
			fmt.Fprintf(os.Stderr, "[reresolve] save log: %v\n", err)
			os.Exit(1)
		}
	}

	if jsonOut {
		type Report struct {
			Sampled       int         `json:"sampled"`
			Flipped       int         `json:"flipped"`
			Kept          int         `json:"kept_irrelevant"`
			Errors        int         `json:"errors"`
			FlipBreakdown map[string]int `json:"flip_breakdown"`
			Flips         []flipEntry `json:"flips"`
		}
		report := Report{
			Sampled:       len(targetIdx),
			Flipped:       len(flips),
			Kept:          kept,
			Errors:        errors,
			FlipBreakdown: flipped,
			Flips:         flips,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}

	fmt.Println()
	fmt.Printf("[reresolve] %d flipped, %d stayed irrelevant, %d errors\n", len(flips), kept, errors)
	if len(flipped) > 0 {
		fmt.Print("[reresolve] flipped verdicts: ")
		parts := make([]string, 0, len(flipped))
		for k, v := range flipped {
			parts = append(parts, fmt.Sprintf("%s=%d", k, v))
		}
		fmt.Println(strings.Join(parts, ", "))
	}
	if len(flips) > 0 {
		fmt.Printf("[reresolve] log updated at %s — next --calibrate will show the new verdicts\n", filepath.Join(dir, ".metabolism-log.json"))
	}
}
