package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceSpanRuleFlagsHashMismatch(t *testing.T) {
	storeRoot := t.TempDir()
	original := "the text the hash was computed from"
	hash := sha256Hex(original)
	if err := os.MkdirAll(filepath.Join(storeRoot, "evidence"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The file is named after `original`'s hash but its actual content has
	// since changed -- self-consistency check must catch this independent
	// of whether Quote happens to still match.
	if err := os.WriteFile(filepath.Join(storeRoot, "evidence", hash+".txt"), []byte("corrupted content"), 0o644); err != nil {
		t.Fatal(err)
	}
	corpusDir := filepath.Join(storeRoot, "corpus")
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package winze

var archived = Provenance{
	Origin:       "test",
	Quote:        "` + original + `",
	EvidenceHash: "` + hash + `",
}
`
	if err := os.WriteFile(filepath.Join(corpusDir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := evidenceSpanRule(corpusDir); rc != 1 {
		t.Errorf("a file whose content does not hash to its own filename must be flagged (rc=1), got rc=%d", rc)
	}
}

func TestEvidenceSpanRuleFlagsMissingFile(t *testing.T) {
	storeRoot := t.TempDir()
	corpusDir := filepath.Join(storeRoot, "corpus")
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package winze

var archived = Provenance{
	Origin:       "test",
	Quote:        "never archived",
	EvidenceHash: "` + sha256Hex("never archived") + `",
}
`
	if err := os.WriteFile(filepath.Join(corpusDir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := evidenceSpanRule(corpusDir); rc != 1 {
		t.Errorf("a missing evidence/<hash>.txt must be flagged (rc=1), got rc=%d", rc)
	}
}

func TestEvidenceSpanRuleFlagsQuoteNotFound(t *testing.T) {
	storeRoot := t.TempDir()
	archivedText := "the text that actually got archived"
	hash := sha256Hex(archivedText)
	if err := os.MkdirAll(filepath.Join(storeRoot, "evidence"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeRoot, "evidence", hash+".txt"), []byte(archivedText), 0o644); err != nil {
		t.Fatal(err)
	}
	corpusDir := filepath.Join(storeRoot, "corpus")
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Self-consistent archive (content really does hash to its own filename),
	// but Quote drifted to something the archive never contained.
	src := `package winze

var archived = Provenance{
	Origin:       "test",
	Quote:        "a Quote that drifted away from the archive",
	EvidenceHash: "` + hash + `",
}
`
	if err := os.WriteFile(filepath.Join(corpusDir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := evidenceSpanRule(corpusDir); rc != 1 {
		t.Errorf("a Quote not found verbatim in its own archive must be flagged (rc=1), got rc=%d", rc)
	}
}

// TestEvidenceSpanRulePassesOnCleanArchive mirrors
// TestCodeRefSpanRuleChecksInStoreRefWithoutClients's storeRoot/corpus
// layout: evidence/ sits beside corpus/, not inside it.
func TestEvidenceSpanRulePassesOnCleanArchive(t *testing.T) {
	storeRoot := t.TempDir()
	quote := "a short evidence fragment"
	hash := sha256Hex(quote)
	if err := os.MkdirAll(filepath.Join(storeRoot, "evidence"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeRoot, "evidence", hash+".txt"), []byte(quote), 0o644); err != nil {
		t.Fatal(err)
	}
	corpusDir := filepath.Join(storeRoot, "corpus")
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package winze

var archived = Provenance{
	Origin:       "test",
	Quote:        "` + quote + `",
	EvidenceHash: "` + hash + `",
}
`
	if err := os.WriteFile(filepath.Join(corpusDir, "e.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := evidenceSpanRule(corpusDir); rc != 0 {
		t.Errorf("a clean archive matching its own hash and containing Quote must pass (rc=0), got rc=%d", rc)
	}
}

func TestEvidenceSpanRuleSkipsWhenEvidenceHashEmpty(t *testing.T) {
	corpusDir := writeLintFixture(t, `package winze

var unarchived = Provenance{
	Origin: "test",
	Quote:  "no evidence hash set",
}
`)
	if rc := evidenceSpanRule(corpusDir); rc != 0 {
		t.Errorf("a Provenance with no EvidenceHash must be a no-op (rc=0), got rc=%d", rc)
	}
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
