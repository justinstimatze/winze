package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// TestSharedPrefixClearsSonnetMin is the caching guard: it fails if the block
// built from the live corpus falls below Sonnet's 1024-token cache minimum,
// where cache_control silently no-ops. A shrinking vocab is the way this
// regresses, so the guard lives next to the vocab, not in a doc.
func TestSharedPrefixClearsSonnetMin(t *testing.T) {
	root := "../.."
	block := sharedMetabolismPrefix(root)
	if block == "" {
		t.Fatalf("sharedMetabolismPrefix returned empty — vocab extraction failed from %s", root)
	}
	// ~4 chars/token; require a margin over the 1024 floor so ordinary vocab
	// churn doesn't skate under it. 1024 tok ≈ 4096 chars; demand ≥ 4600.
	const minChars = 4600
	if len(block) < minChars {
		toks := len(block) / 4
		t.Errorf("shared prefix is %d chars (~%d tok) — under the Sonnet cache floor with margin (want ≥ %d chars ≈ %d tok). "+
			"Caching will silently no-op. Add real shared content or lower the tier expectation.",
			len(block), toks, minChars, minChars/4)
	}
	t.Logf("shared prefix: %d chars (~%d tok), clears Sonnet 1024-tok min", len(block), len(block)/4)
}

// TestSharedPrefixDeterministic guards the property server-side prefix caching
// keys on: two builds from the same vocab must be byte-identical, or the cache
// never hits across calls/runs.
func TestSharedPrefixDeterministic(t *testing.T) {
	root := "../.."
	a := sharedMetabolismPrefix(root)
	b := sharedMetabolismPrefix(root)
	if a != b {
		t.Fatalf("shared prefix is non-deterministic (%d vs %d chars) — cache will never hit", len(a), len(b))
	}
}

// TestSharedPrefixRolesInSync guards the hand-authored role glosses against
// drift from roles.go: every role type declared in roles.go must appear in the
// gloss block, and vice versa. If someone adds a role, this fails until the
// gloss is written.
func TestSharedPrefixRolesInSync(t *testing.T) {
	root := "../.."
	src, err := os.ReadFile(filepath.Join(root, "roles.go"))
	if err != nil {
		t.Fatalf("read roles.go: %v", err)
	}
	glosses := extractRoleGlosses()
	for _, m := range roleDeclRe.FindAllStringSubmatch(string(src), -1) {
		role := m[1]
		if !strings.Contains(glosses, "- "+role+":") {
			t.Errorf("role %q is declared in roles.go but has no gloss in extractRoleGlosses — add one", role)
		}
	}
}

// TestResolveSharedPrefixDrift is the behavioral drift check. It replays
// historical resolve cases from the log through llmResolve with the shared
// System block OFF (old behavior) and ON (new behavior) and compares the
// classification labels. The shared block is *additive* context, so labels
// should overwhelmingly agree; wide disagreement means the added context
// perturbs the calibrated classifier and the block should be reconsidered.
//
// LLM output is non-deterministic, so a stray single flip is noise, not drift.
// The test fails only on wide disagreement (majority flipped). API-gated;
// costs ~2×N Sonnet calls. Run manually:
//
//	go test ./cmd/metabolism/ -run TestResolveSharedPrefixDrift -v
func TestResolveSharedPrefixDrift(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set — skipping live resolve drift check")
	}
	root := "../.."
	block := sharedMetabolismPrefix(root)
	if block == "" {
		t.Fatalf("empty shared prefix")
	}

	mlog := loadLog(filepath.Join(root, ".metabolism-log.json"))
	var idx []int
	for i, c := range mlog.Cycles {
		if len(c.Papers) > 0 {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		t.Skip("no cycles with papers in log — nothing to replay")
	}
	const nCases = 6
	sample := pickSpacedIndices(idx, nCases)

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	agree, disagree := 0, 0
	for _, ci := range sample {
		c := mlog.Cycles[ci]
		brief := lookupBrief(c.Hypothesis)
		off, err1 := llmResolve(client, "", c.Hypothesis, brief, c.Papers)
		on, err2 := llmResolve(client, block, c.Hypothesis, brief, c.Papers)
		if err1 != nil || err2 != nil {
			t.Logf("cycle %d %s: error off=%v on=%v", ci, c.Hypothesis, err1, err2)
			continue
		}
		if off == on {
			agree++
			t.Logf("cycle %d %-40s OFF=%-12s ON=%-12s  AGREE", ci, c.Hypothesis, off, on)
		} else {
			disagree++
			t.Logf("cycle %d %-40s OFF=%-12s ON=%-12s  *** DIFFER", ci, c.Hypothesis, off, on)
		}
	}
	total := agree + disagree
	if total == 0 {
		t.Skip("all replay calls errored — inconclusive")
	}
	rate := float64(agree) / float64(total) * 100
	fmt.Printf("\n[resolve-drift] %d/%d agree (%.0f%%) between shared-block OFF and ON\n", agree, total, rate)
	// Majority flip => the block materially perturbs the classifier.
	if disagree > agree {
		t.Errorf("resolve drift: %d of %d cases flipped label when the shared block was added — "+
			"the additive context is perturbing the calibrated classifier; reconsider the block or move its content out of resolve's view", disagree, total)
	}
}
