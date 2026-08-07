package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// criticPromptDigest is the sha256 of buildTripCriticPrompt over the fixture
// below, as of the last time TestCriticStability was run against it.
//
// Update it ONLY after running:
//
//	WINZE_CRITIC_STABILITY=1 ANTHROPIC_API_KEY=... go test ./cmd/metabolism/ -run TestCriticStability -v
//
// and confirming every fixture is still unanimous.
//
// Last verified 2026-08-06, 60 trials, all six fixtures unanimous:
// promoted-phrasing / promoted-selfcert / promoted-swapped-dir 10/10 accept,
// rejected-phrasing / rejected+selfcert / rejected+namedterm 10/10 reject.
const criticPromptDigest = "67bea0bbc62e83ec684d38ffc399986ddda645ede5a7bf6c8b373532fe702602"

// TestCriticPromptUnchangedSinceStabilityRun is a tripwire, not a quality
// check. It exists because `go test ./...` reports green while
// TestCriticStability skips itself — that test needs WINZE_CRITIC_STABILITY=1
// and an API key, since it makes 60 billed calls and takes about four minutes.
//
// The failure it prevents is specific and already happened once: a critic
// prompt change shipped on the strength of a green suite that had never
// exercised the critic. A prior session had measured the gate as correctly
// calibrated, and nothing in the run said otherwise because nothing ran.
//
// So this costs nothing, runs always, and fires exactly when the skip starts
// mattering: the moment the prompt text changes. It cannot tell you the new
// prompt is worse — only that nobody has checked yet.
func TestCriticPromptUnchangedSinceStabilityRun(t *testing.T) {
	sum := sha256.Sum256([]byte(buildTripCriticPrompt(
		criticGuardFixtureConn(),
		criticGuardFixtureExemplars(),
		[]string{"AlphaThesis ~ BetaThesis"},
	)))
	got := hex.EncodeToString(sum[:])
	if criticPromptDigest == "" {
		t.Fatalf("criticPromptDigest is unset; run the stability suite and record:\n\t%s", got)
	}
	if got != criticPromptDigest {
		t.Fatalf(`the trip critic prompt changed and its stability has not been re-verified.

  want %s
  got  %s

TestCriticStability is the check that matters here and it skips by default, so
a green suite says nothing about this change. Run it:

  WINZE_CRITIC_STABILITY=1 ANTHROPIC_API_KEY=... \
    go test ./cmd/metabolism/ -run TestCriticStability -v

Every fixture must stay unanimous — the promoted-* ones accepting 10/10 is the
check that a new REJECT rule has not started catching good claims. Then set
criticPromptDigest to the got value above and update the "Last verified" note.`,
			criticPromptDigest, got)
	}
}

// criticGuardFixtureConn is held apart from the stability fixtures on purpose:
// this one must never change, or the digest tracks the fixture instead of the
// prompt and the tripwire stops meaning anything.
func criticGuardFixtureConn() TripConnection {
	return TripConnection{
		EntityA:     "AlphaThesis",
		EntityB:     "GammaThesis",
		ClusterA:    1,
		ClusterB:    2,
		PromptType:  "analogy",
		Temperature: 1.0,
		Predicate:   "StructurallyAnalogousTo",
		Connection:  "Both posit a closed enumeration that meets a case outside it.",
		Rationale:   "Each carries the same three parts: an enumeration claimed complete, a case its members do not cover, and a choice between extending ad hoc or conceding the limit.",
		Score:       4,
	}
}

func criticGuardFixtureExemplars() []claimExemplar {
	return []claimExemplar{{
		Subject:   "DeltaPerson",
		Predicate: "Proposes",
		Object:    "EpsilonHypothesis",
		Quote:     "a fixed exemplar quote, long enough to sit above the substance bar the rubric asks the candidate to match",
	}}
}
