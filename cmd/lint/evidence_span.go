package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

func evidenceSpanRule(dir string) int {
	corpus, err := corpusparse.ParseCorpusFull(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[evidence-span] error: %v\n", err)
		return 2
	}

	var archived []corpusparse.Provenance
	for _, p := range corpus.Provenance {
		if p.EvidenceHash != "" {
			archived = append(archived, p)
		}
	}
	if len(archived) == 0 {
		fmt.Println("[evidence-span] no archived-evidence Provenance in this corpus")
		return 0
	}

	// dir is the corpus subdirectory; evidence/ sits at the store root
	// alongside it -- the same filepath.Dir(dir) convention codeRefSpanRule
	// uses for a Client=="" (in-store) CodeRef.
	storeRoot := filepath.Dir(dir)

	var bad []string
	for _, p := range archived {
		if msg, ok := checkArchivedProvenance(storeRoot, p); !ok {
			bad = append(bad, msg)
		}
	}

	fmt.Printf("[evidence-span] %d archived Provenance var(s), %d bad\n", len(archived), len(bad))
	if len(bad) == 0 {
		return 0
	}
	for _, b := range bad {
		fmt.Println("   ", b)
	}
	return 1
}

func checkArchivedProvenance(storeRoot string, p corpusparse.Provenance) (msg string, ok bool) {
	path := filepath.Join(storeRoot, "evidence", p.EvidenceHash+".txt")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("%s: evidence/%s.txt: %v", p.VarName, p.EvidenceHash, err), false
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != p.EvidenceHash {
		return fmt.Sprintf("%s: evidence/%s.txt content does not hash to its own filename (corrupted?)", p.VarName, p.EvidenceHash), false
	}
	if !strings.Contains(string(content), p.Quote) {
		return fmt.Sprintf("%s: Quote not found verbatim in evidence/%s.txt (drifted from what was archived)", p.VarName, p.EvidenceHash), false
	}
	return "", true
}
