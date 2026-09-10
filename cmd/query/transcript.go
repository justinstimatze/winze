package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justinstimatze/winze/internal/cliutil"
	"github.com/justinstimatze/winze/internal/transcript"
)

// buildFTIndexForTranscript indexes each substantial assistant turn in a
// session transcript as its own document, reusing ftDoc/ftIndex/search
// unchanged against a new population: transcript turns instead of corpus
// entities/provenance. The BM25 scoring loop in (*ftIndex).search only ever
// reads terms/len, never kind -- confirmed by reading it directly, not
// assumed -- so this needed no changes to the shared struct or algorithm.
func buildFTIndexForTranscript(turns []transcript.Turn) *ftIndex {
	fi := &ftIndex{df: map[string]int{}}
	total := 0
	for i, t := range turns {
		toks := tokenize(t.Text)
		if len(toks) == 0 {
			continue
		}
		tf := make(map[string]int, len(toks))
		for _, tok := range toks {
			tf[tok]++
		}
		fi.docs = append(fi.docs, ftDoc{kind: "turn", ref: i, terms: tf, len: len(toks)})
		for tok := range tf {
			fi.df[tok]++
		}
		total += len(toks)
	}
	fi.n = len(fi.docs)
	if fi.n > 0 {
		fi.avgLen = float64(total) / float64(fi.n)
	}
	return fi
}

// reportTranscriptError prints a --transcript failure (missing session,
// unreadable file) in whichever shape the caller asked for and exits --
// there is no partial result to fall back to once the transcript itself
// can't be read.
func reportTranscriptError(err error, jsonOut bool) {
	if jsonOut {
		printJSON(map[string]any{"error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "transcript: %v\n", err)
	}
	os.Exit(1)
}

// resolveSessionTranscript finds the transcript file for a session-id
// across every Claude Code project directory. Session ids are globally
// unique UUIDs, so a plain glob resolves unambiguously regardless of the
// ~/.claude/projects/<encoded-cwd> directory-naming scheme being lossy
// (both "/" and "." in a real cwd collapse to "-", so two different real
// project paths can share an encoded directory name) -- that ambiguity
// only matters for reconstructing a project name from its encoding, which
// this never needs to do.
func resolveSessionTranscript(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return resolveSessionTranscriptUnder(filepath.Join(home, ".claude", "projects"), sessionID)
}

// runTranscriptSearch runs the --transcript mode: BM25 search over one
// session's own turns, returning exact quotes -- the tier-2 lookup behind
// a session-capture entry's coarse locator. Not the retired raw tier
// (docs/raw-evidence-retrieval.md): that indexed the same short note text
// already sitting in the typed store; this indexes the full transcript,
// content no existing winze mechanism searches at all. Semantic search is
// deliberately deferred (see the implementation plan) -- BM25 only for v1,
// mirroring the retired raw tier's own phased build history.
func runTranscriptSearch(sessionID, query string, jsonOut bool) {
	if strings.TrimSpace(query) == "" {
		reportTranscriptError(fmt.Errorf("--transcript requires --transcript-query"), jsonOut)
	}
	path, err := resolveSessionTranscript(sessionID)
	if err != nil {
		reportTranscriptError(err, jsonOut)
	}
	turns, err := transcript.AllSubstantial(path)
	if err != nil {
		reportTranscriptError(err, jsonOut)
	}

	fi := buildFTIndexForTranscript(turns)
	hits := fi.search(query, 5)

	if jsonOut {
		out := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			out = append(out, map[string]any{"score": h.score, "quote": turns[h.ref].Text, "line": turns[h.ref].Index})
		}
		printJSON(map[string]any{"session_id": sessionID, "query": query, "count": len(hits), "hits": out})
		return
	}

	if len(hits) == 0 {
		fmt.Printf("No matches for %q in session %s\n", query, sessionID)
		return
	}
	fmt.Printf("Transcript matches for %q in session %s (%d):\n\n", query, sessionID, len(hits))
	for _, h := range hits {
		fmt.Printf("  [%.2f] line %d\n        %s\n", h.score, turns[h.ref].Index, cliutil.Truncate(turns[h.ref].Text, 300))
	}
}

// resolveSessionTranscriptUnder is resolveSessionTranscript's testable
// core: the glob root is a parameter instead of always ~/.claude/projects,
// so a test can point it at a temp directory instead of the real one.
func resolveSessionTranscriptUnder(projectsDir, sessionID string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(projectsDir, "*", sessionID+".jsonl"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no transcript found for session %s", sessionID)
	}
	return matches[0], nil
}
