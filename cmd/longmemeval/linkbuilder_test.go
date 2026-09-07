package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

// linkSuggestScoreMirror and linkSuggestMaxMirror mirror cmd/agent/mcp.go's
// linkSuggestScore()/linkSuggestMax -- cmd/longmemeval can't import cmd/agent
// (both are package main), so these track the production values by hand, the
// same cross-package mirroring rawDoc (cmd/query/rawfulltext.go) already uses
// for cmd/agent/rawlog.go's rawLogEntry. Update both if the source values ever
// move.
const (
	linkSuggestScoreMirror = 0.45
	linkSuggestMaxMirror   = 3
)

// TestFilterLinkCandidates exercises the pure filtering logic against a
// literal recallHits fixture -- no live store, no ollama, mirroring how
// TestBestRankOfPicksTheBetterRank (claimnote_test.go) tests its own pure
// helper the same way.
func TestFilterLinkCandidates(t *testing.T) {
	hits := recallHits{Hits: []struct {
		VarName string  `json:"var_name"`
		Score   float64 `json:"score"`
	}{
		{VarName: "Self", Score: 1.0},
		{VarName: "StrongMatch", Score: 0.6},
		{VarName: "BorderlineAbove", Score: 0.45},
		{VarName: "BelowFloor", Score: 0.44},
		{VarName: "", Score: 0.9},
		{VarName: "FourthCandidate", Score: 0.5},
		{VarName: "FifthCandidate", Score: 0.5},
	}}
	got := filterLinkCandidates(hits, "Self")
	if len(got) != linkSuggestMaxMirror {
		t.Fatalf("got %d candidates, want the cap of %d: %+v", len(got), linkSuggestMaxMirror, got)
	}
	for _, c := range got {
		if c.VarName == "Self" {
			t.Fatalf("self-hit leaked through: %+v", got)
		}
		if c.VarName == "BelowFloor" {
			t.Fatalf("a hit below the floor leaked through: %+v", got)
		}
		if c.VarName == "" {
			t.Fatalf("an empty var leaked through: %+v", got)
		}
	}
	if got[0].VarName != "StrongMatch" {
		t.Fatalf("expected the strongest match first, got %+v", got)
	}
}

func TestFilterLinkCandidatesEmptyWhenNothingClearsTheFloor(t *testing.T) {
	hits := recallHits{Hits: []struct {
		VarName string  `json:"var_name"`
		Score   float64 `json:"score"`
	}{{VarName: "A", Score: 0.2}, {VarName: "B", Score: 0.1}}}
	if got := filterLinkCandidates(hits, "Self"); len(got) != 0 {
		t.Fatalf("expected no candidates below the floor, got %+v", got)
	}
}

// filterLinkCandidates mirrors cmd/agent/mcp.go's linkCandidates: drop the
// self-hit and anything without a var, keep hits at or above the suggestion
// floor, cap the count. Pure and independently testable -- no store, no
// subprocess, no ollama.
func filterLinkCandidates(hits recallHits, selfVar string) []linkCandidate {
	var out []linkCandidate
	for _, h := range hits.Hits {
		if h.VarName == "" || h.VarName == selfVar || h.Score < linkSuggestScoreMirror {
			continue
		}
		out = append(out, linkCandidate{VarName: h.VarName, Score: h.Score})
		if len(out) == linkSuggestMaxMirror {
			break
		}
	}
	return out
}

// linkRelatedSessions runs one pass over the finished store, ranking each
// session's own note text by cosine similarity via winze-query --semantic --
// the same mechanism production's own link-suggestion path
// (cmd/agent/mcp.go's checkDedup -> nearestMemories) uses, and deliberately
// NOT winze_recall/--hybrid: --hybrid's JSON hits carry rrf/lex_rank/sem_rank,
// never a "score" key, so a first version of this function that queried
// winze_recall came back with every candidate's Score silently at its zero
// value and zero links created across all 150 sessions -- confirmed by
// querying the same store directly with --semantic and seeing real
// candidates well above the link floor. That's a live bug in winze_recall's
// own JSON response (queryHit.Score is always 0 there too, since handleRecall
// decodes the same --hybrid shape), worth its own fix, but out of scope here:
// this function only needs to mirror production's real link-suggestion path
// correctly, which is --semantic, not winze_recall.
//
// Hits that clear production's own link-suggestion threshold become real
// winze_link claim edges. This is the fixture fix docs/raw-evidence-retrieval.md's
// Honest limits section calls for: an entity-graph retrieval channel needs a
// store with real edges to be testable at all, and until now nothing in this
// harness ever called winze_link.
//
// Runs after every session is written, against the complete store, not
// incrementally per write -- a session can end up linked to one written
// later in replay order, which is not how a live incremental session would
// see dd.related. Accepted: this pass exists to produce real claim-graph
// structure to measure an entity-graph channel against, not to simulate live
// session write-time behavior.
//
// The rationale text handed to winze_link is deliberately self-labeled as
// harness-synthetic, so nothing here could later be mistaken for a genuine
// winze-authored Conjecture if this scratch store were ever inspected.
func linkRelatedSessions(t *testing.T, run func(args ...string) (string, error), runQuery func(args ...string) (string, error), store string, picked []*transcriptSession, varSets [][]string, noteSets [][]string) int {
	t.Helper()
	linked := 0
	seen := map[string]bool{}
	for i, vars := range varSets {
		if len(vars) == 0 || len(noteSets[i]) == 0 {
			continue
		}
		selfVar := vars[0]
		out, err := runQuery("--json", "--semantic", noteSets[i][0], store)
		if err != nil {
			t.Logf("linkRelatedSessions: semantic query for %s failed: %v\n%s", selfVar, err, out)
			continue
		}
		var hits recallHits
		if err := json.Unmarshal([]byte(out), &hits); err != nil {
			t.Logf("linkRelatedSessions: unparseable semantic JSON for %s: %v\n%s", selfVar, err, out)
			continue
		}
		for _, c := range filterLinkCandidates(hits, selfVar) {
			pairA, pairB := selfVar+"|"+c.VarName, c.VarName+"|"+selfVar
			if seen[pairA] || seen[pairB] {
				continue
			}
			seen[pairA] = true
			rationale := fmt.Sprintf(
				"harness-generated for entity-graph channel measurement: cosine %.2f between session notes (cmd/longmemeval linkRelatedSessions) — not an editorial judgment",
				c.Score)
			linkPayload := fmt.Sprintf(`{"from":%s,"to":%s,"relation":"RelatesTo","rationale":%s}`,
				mustJSON(selfVar), mustJSON(c.VarName), mustJSON(rationale))
			linkOut, err := run("call", "winze_link", linkPayload)
			if err != nil {
				t.Logf("linkRelatedSessions: link %s->%s failed: %v\n%s", selfVar, c.VarName, err, linkOut)
				continue
			}
			linked++
		}
	}
	return linked
}

// linkCandidate is one recall hit worth turning into a claim edge.
type linkCandidate struct {
	VarName string
	Score   float64
}
