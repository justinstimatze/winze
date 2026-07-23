package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// tierInvocation maps an autonomy tier to the metabolism arguments it runs and
// whether it pushes afterward. This is the whole autonomy policy in one place.
//
//	T1 sense-only  — sensors + cheap phases; no ingest, no commit, ~free
//	T2 evolve/local — full --evolve; commits to local main; never pushes
//	T3 evolve/push  — full --evolve; commits AND pushes (fully autonomous)
func tierInvocation(tier int) (args []string, push bool, err error) {
	switch tier {
	case TierSenseOnly:
		return []string{"--evolve", "--phases=sense,bias,dream,calibrate"}, false, nil
	case TierEvolve:
		return []string{"--evolve"}, false, nil
	case TierPush:
		return []string{"--evolve"}, true, nil
	default:
		return nil, false, fmt.Errorf("invalid tier %d", tier)
	}
}

// metabolismBin resolves the winze-metabolism binary: under WINZE_BIN when set,
// else the bare name via PATH (after `make install`). Matches cmd/mem's convention.
func metabolismBin() string {
	if v := os.Getenv("WINZE_BIN"); v != "" {
		return filepath.Join(v, "winze-metabolism")
	}
	return "winze-metabolism"
}

// runInstance is what the systemd service invokes each tick: look the instance
// up, run its tier's metabolism invocation against its own budget, and (T3)
// push. A disabled or unregistered instance is a no-op exit 0 — a stale timer
// must never fail loudly or spend.
func runInstance(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	in := reg.find(abs)
	if in == nil {
		fmt.Fprintf(os.Stderr, "winze-metabolize: %s not registered — skipping\n", abs)
		return nil
	}
	if !in.Enabled {
		fmt.Fprintf(os.Stderr, "winze-metabolize: %s disabled — skipping\n", abs)
		return nil
	}
	if !validTier(in.Tier) {
		return fmt.Errorf("%s has invalid tier %d", abs, in.Tier)
	}

	args, push, err := tierInvocation(in.Tier)
	if err != nil {
		return err
	}
	args = append(args, abs)

	fmt.Fprintf(os.Stderr, "winze-metabolize: %s tier %d (%s), budget %d¢\n",
		abs, in.Tier, tierName(in.Tier), in.BudgetCents)

	cmd := exec.Command(metabolismBin(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "METABOLISM_BUDGET_CENTS="+strconv.Itoa(in.BudgetCents))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("metabolism failed: %w", err)
	}

	if push {
		return gitPush(abs)
	}
	return nil
}

// gitPush pushes the instance's default branch. Only tier 3 reaches here, and
// only after a clean metabolism run — the corpus is committed by --evolve's
// pipeline gate before this point.
func gitPush(dir string) error {
	cmd := exec.Command("git", "-C", dir, "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push (tier 3): %w", err)
	}
	return nil
}
