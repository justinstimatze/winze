package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runPipeline executes the full automated quality pipeline:
//
//  1. LLM-assisted ingest (generates metabolism_cycleN.go)
//  2. go build ./... (type system validation)
//  3. go vet ./... (static analysis)
//  4. go run ./cmd/lint . (deterministic lint rules)
//  5. go run ./cmd/lint . --llm --llm-max-calls=N (semantic contradiction check)
//  6. If all pass → git commit. If llm-contradiction finds issues → reject.
//
// Exit codes: 0 = committed, 1 = fatal error, 2 = quality gate rejection.
func runPipeline(dir, zimPath, zimIndex string, llmBudget int) {
	start := time.Now()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  metabolism pipeline: ingest → build → lint → commit")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Step 1: Ingest
	fmt.Println("\n[1/5] LLM-assisted ingest from ZIM cycles with signal...")
	outcome := runIngest(dir, zimPath, zimIndex)
	if outcome.OutPath == "" {
		fmt.Println("\n[pipeline] nothing to ingest — no uningested ZIM cycles with papers")
		return
	}
	fmt.Printf("[1/5] ✓ ingest produced %d claims → %d files\n", outcome.ClaimCount, len(outcome.Backups))
	pipelineBackups = outcome.Backups

	// Step 2: go build (should already pass from ingest, but verify clean)
	fmt.Println("\n[2/5] go build ./...")
	if !runGate(dir, "go", "build", "./...") {
		// Try to salvage: remove bad claims one at a time
		if salvagePipeline(dir, outcome.OutPath) {
			fmt.Println("[2/5] ✓ build passed (after salvage)")
		} else {
			rejectPipeline(outcome.OutPath, "go build failed — type system rejected generated code")
			return
		}
	} else {
		fmt.Println("[2/5] ✓ build passed")
	}

	// Step 3: go vet
	fmt.Println("\n[3/5] go vet ./...")
	if !runGate(dir, "go", "vet", "./...") {
		rejectPipeline(outcome.OutPath, "go vet found issues")
		return
	}
	fmt.Println("[3/5] ✓ vet passed")

	// Step 4: Deterministic lint
	fmt.Println("\n[4/5] deterministic lint rules...")
	lintOut, lintOk := runGateCapture(dir, "go", "run", "./cmd/lint", dir)
	if !lintOk {
		rejectPipeline(outcome.OutPath, "deterministic lint failed")
		return
	}
	// Check for actual failures in lint output (orphans are advisory, not failures)
	if lintHasFailures(lintOut) {
		fmt.Println(lintOut)
		rejectPipeline(outcome.OutPath, "lint found non-advisory failures")
		return
	}
	fmt.Println("[4/5] ✓ lint passed")

	// Step 5: LLM contradiction check
	llmRan := false
	fmt.Printf("\n[5/5] LLM contradiction check (budget: %d calls)...\n", llmBudget)
	llmOut, llmOk := runGateCapture(dir, "go", "run", "./cmd/lint", dir,
		"--llm", fmt.Sprintf("--llm-max-calls=%d", llmBudget))
	if !llmOk {
		// LLM lint failure is non-fatal (API might be unavailable)
		fmt.Fprintf(os.Stderr, "[5/5] ⚠ LLM lint errored — proceeding without contradiction check\n")
	} else if llmHasContradictions(llmOut) {
		fmt.Println(llmOut)
		rejectPipeline(outcome.OutPath,
			"LLM contradiction check found issues — escalate, do not auto-commit")
		return
	} else {
		llmRan = true
		fmt.Println("[5/5] ✓ no contradictions found")
	}

	// All gates passed — commit
	fmt.Println("\n───────────────────────────────────────────────────────────")
	commitPipeline(dir, outcome, llmRan)

	// Mark source cycles as ingested so they won't be re-processed
	if len(outcome.CycleIndices) > 0 {
		logPath := filepath.Join(dir, ".metabolism-log.json")
		mlog := loadLog(logPath)
		for _, idx := range outcome.CycleIndices {
			if idx < len(mlog.Cycles) {
				mlog.Cycles[idx].Ingested = true
				mlog.Cycles[idx].LLMLintSkipped = !llmRan
				mlog.Cycles[idx].PipelineClaims = outcome.PipelineClaims
			}
		}
		if err := saveLog(logPath, mlog); err != nil {
			fmt.Fprintf(os.Stderr, "[pipeline] warning: failed to mark cycles as ingested: %v\n", err)
		} else {
			fmt.Printf("[pipeline] marked %d cycles as ingested\n", len(outcome.CycleIndices))
		}
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("\n[pipeline] completed in %s\n", elapsed)
}

// runGate runs a command and returns true if it exits 0.
func runGate(dir string, name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

// runGateCapture runs a command and captures stdout+stderr. Returns output and success.
func runGateCapture(dir string, name string, args ...string) (string, bool) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err == nil
}

// lintHasFailures checks if lint output contains actual failures (not just advisory info).
// naming-oracle violations and value-conflict are failures.
// orphan-report and contested-concept are advisory.
func lintHasFailures(output string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// naming-oracle FAIL lines
		if strings.Contains(line, "FAIL") && strings.Contains(line, "naming-oracle") {
			return true
		}
		// value-conflict with actual conflicts
		if strings.Contains(line, "value-conflict") && strings.Contains(line, "CONFLICT") {
			return true
		}
	}
	return false
}

// llmHasContradictions checks if the LLM lint output reports contradictions.
func llmHasContradictions(output string) bool {
	return strings.Contains(output, "CONTRADICTION") ||
		strings.Contains(output, "contradiction detected")
}

// salvagePipeline attempts to rescue a generated file that failed to build
// by removing claim sections one at a time from the end. Returns true if
// a buildable subset was found.
func salvagePipeline(dir, outPath string) bool {
	content, err := os.ReadFile(outPath)
	if err != nil {
		return false
	}

	// Split on claim section delimiters
	marker := "// ---------------------------------------------------------------------------"
	parts := strings.Split(string(content), marker)
	if len(parts) <= 2 {
		return false // header + at most one claim, can't salvage
	}

	// parts[0] is the package header, parts[1..N] are claim sections (each preceded by the marker)
	// Try removing sections from the end until it builds
	original := len(parts)
	for len(parts) > 2 { // keep at least header + one claim
		parts = parts[:len(parts)-1]
		candidate := strings.Join(parts, marker)
		if err := os.WriteFile(outPath, []byte(candidate), 0644); err != nil {
			return false
		}
		if runGate(dir, "go", "build", "./...") {
			salvaged := len(parts) - 1 // subtract header
			fmt.Printf("[pipeline] salvaged %d/%d claim sections (removed %d bad)\n",
				salvaged, original-1, original-1-salvaged)
			return true
		}
	}

	// Even single claim didn't build — restore original and fail
	if rbErr := os.WriteFile(outPath, content, 0644); rbErr != nil {
		fmt.Fprintf(os.Stderr, "[pipeline] CRITICAL: restore failed for %s: %v\n", filepath.Base(outPath), rbErr)
	}
	return false
}

// rejectPipeline rolls back modified files using backups and reports why.
// When called from --pipeline (standalone), exits with code 2.
// When called from --evolve (runCycle), just rolls back and returns.
var pipelineStandalone = false
var pipelineBackups map[string][]byte // set by runPipeline from outcome.Backups

func rejectPipeline(outPath, reason string) {
	fmt.Println("\n───────────────────────────────────────────────────────────")
	fmt.Printf("[pipeline] REJECTED: %s\n", reason)

	// Roll back all modified files
	for path, original := range pipelineBackups {
		if original == nil {
			// New file — delete it
			if rbErr := os.Remove(path); rbErr != nil {
				fmt.Fprintf(os.Stderr, "[pipeline] CRITICAL: rollback remove failed for %s: %v\n", filepath.Base(path), rbErr)
			}
			fmt.Printf("[pipeline] removed %s\n", filepath.Base(path))
		} else {
			// Existing file — restore backup
			if rbErr := os.WriteFile(path, original, 0644); rbErr != nil {
				fmt.Fprintf(os.Stderr, "[pipeline] CRITICAL: rollback restore failed for %s: %v\n", filepath.Base(path), rbErr)
			}
			fmt.Printf("[pipeline] restored %s\n", filepath.Base(path))
		}
	}

	if pipelineStandalone {
		os.Exit(2)
	}
}

// commitPipeline stages and commits all modified corpus files.
func commitPipeline(dir string, outcome ingestOutcome, llmRan bool) {
	// git add all modified files
	addFailed := 0
	for path := range outcome.Backups {
		relPath, err := filepath.Rel(dir, path)
		if err != nil || relPath == "" {
			relPath = filepath.Base(path)
		}
		addCmd := exec.Command("git", "add", relPath)
		addCmd.Dir = dir
		addCmd.Stderr = os.Stderr
		if err := addCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[pipeline] git add %s failed: %v\n", relPath, err)
			addFailed++
		}
	}
	if addFailed > 0 {
		fmt.Fprintf(os.Stderr, "[pipeline] WARNING: %d/%d git add operations failed\n", addFailed, len(outcome.Backups))
	}

	// git commit
	llmStatus := "llm-contradiction ✓"
	if !llmRan {
		llmStatus = "llm-contradiction ⊘ (skipped)"
	}

	fileList := make([]string, 0, len(outcome.Backups))
	for path := range outcome.Backups {
		fileList = append(fileList, filepath.Base(path))
	}
	msg := fmt.Sprintf("metabolism: automated ingest — %d claims across %d files\n\nFiles: %s\nQuality gates: build ✓ vet ✓ lint ✓ %s",
		outcome.ClaimCount, len(fileList), strings.Join(fileList, ", "), llmStatus)

	commitCmd := exec.Command("git", "commit", "-m", msg)
	commitCmd.Dir = dir
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[pipeline] git commit failed: %v — changes staged but not committed\n", err)
		return
	}

	fmt.Printf("[pipeline] ✓ committed %d claims across %d corpus files\n", outcome.ClaimCount, len(fileList))
}
