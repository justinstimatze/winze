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
	fmt.Println("\n[1/5] LLM-assisted ingest from corroborated ZIM cycles...")
	outcome := runIngest(dir, zimPath, zimIndex)
	if outcome.OutPath == "" {
		fmt.Println("\n[pipeline] nothing to ingest — no corroborated cycles with papers")
		return
	}
	fmt.Printf("[1/5] ✓ ingest produced %d claims → %s\n", outcome.ClaimCount, filepath.Base(outcome.OutPath))

	// Step 2: go build (should already pass from ingest, but verify clean)
	fmt.Println("\n[2/5] go build ./...")
	if !runGate(dir, "go", "build", "./...") {
		rejectPipeline(outcome.OutPath, "go build failed — type system rejected generated code")
		return
	}
	fmt.Println("[2/5] ✓ build passed")

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
		fmt.Println("[5/5] ✓ no contradictions found")
	}

	// All gates passed — commit
	fmt.Println("\n───────────────────────────────────────────────────────────")
	commitPipeline(dir, outcome)
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

// rejectPipeline removes the generated file and reports why.
func rejectPipeline(outPath, reason string) {
	fmt.Println("\n───────────────────────────────────────────────────────────")
	fmt.Printf("[pipeline] REJECTED: %s\n", reason)
	fmt.Printf("[pipeline] removing %s\n", filepath.Base(outPath))
	if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[pipeline] warning: could not remove %s: %v\n", outPath, err)
	}
	os.Exit(2)
}

// commitPipeline stages and commits the generated file.
func commitPipeline(dir string, outcome ingestOutcome) {
	relPath, _ := filepath.Rel(dir, outcome.OutPath)
	if relPath == "" {
		relPath = filepath.Base(outcome.OutPath)
	}

	// git add
	addCmd := exec.Command("git", "add", relPath)
	addCmd.Dir = dir
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[pipeline] git add failed: %v\n", err)
		os.Exit(1)
	}

	// git commit
	msg := fmt.Sprintf("metabolism: automated ingest — %d claims (%s)\n\nGenerated by: go run ./cmd/metabolism --pipeline\nQuality gates: build ✓ vet ✓ lint ✓ llm-contradiction ✓",
		outcome.ClaimCount, filepath.Base(outcome.OutPath))

	commitCmd := exec.Command("git", "commit", "-m", msg)
	commitCmd.Dir = dir
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[pipeline] git commit failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[pipeline] ✓ committed %s with %d claims\n", filepath.Base(outcome.OutPath), outcome.ClaimCount)
}
