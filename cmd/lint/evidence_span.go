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

// evidenceSpanRule checks every named, top-level Provenance var whose
// EvidenceHash is set: the archived evidence/<hash>.txt file exists, its
// content hashes to its own filename (catches corruption/truncation), and
// Quote appears verbatim within it (catches a Quote that drifted from what
// was actually archived after the fact). Mirrors codeRefSpanRule's shape
// and return-code convention (0 clean, 1 rule failure, 2 hard error), but
// simpler: evidence is always local to this repo, so there is no --clients
// resolution axis to gate on.
//
// Only a named var is visible here at all -- ParseCorpusFull's Provenance
// list comes from tryParseProvenance, which only recognizes a top-level
// `var X = Provenance{...}` declaration, not one inlined directly inside a
// claim's own composite literal (renderClaim's default authoring mode, see
// docs/authoring.md). That is a deliberate scope boundary, not a gap this
// rule tries to work around: EvidenceHash is only meaningful on a Provenance
// worth naming and reusing in the first place.
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
		path := filepath.Join(storeRoot, "evidence", p.EvidenceHash+".txt")
		content, err := os.ReadFile(path)
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s: evidence/%s.txt: %v", p.VarName, p.EvidenceHash, err))
			continue
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != p.EvidenceHash {
			bad = append(bad, fmt.Sprintf("%s: evidence/%s.txt content does not hash to its own filename (corrupted?)", p.VarName, p.EvidenceHash))
			continue
		}
		if !strings.Contains(string(content), p.Quote) {
			bad = append(bad, fmt.Sprintf("%s: Quote not found verbatim in evidence/%s.txt (drifted from what was archived)", p.VarName, p.EvidenceHash))
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
