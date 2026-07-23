package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestTierInvocation pins the autonomy policy: tier 1 must never mutate or
// push, tier 2 evolves but never pushes, only tier 3 pushes. A regression here
// is a safety bug (a "watch it breathe" instance silently writing/pushing).
func TestTierInvocation(t *testing.T) {
	t.Run("tier1 sense-only never mutates or pushes", func(t *testing.T) {
		args, push, err := tierInvocation(TierSenseOnly)
		if err != nil {
			t.Fatal(err)
		}
		if push {
			t.Error("tier 1 must not push")
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "ingest") || strings.Contains(joined, "resolve") || strings.Contains(joined, "trip") {
			t.Errorf("tier 1 must exclude mutating phases, got %q", joined)
		}
		if !strings.Contains(joined, "--phases=") {
			t.Errorf("tier 1 must pin an explicit phase subset, got %q", joined)
		}
	})
	t.Run("tier2 evolves but does not push", func(t *testing.T) {
		_, push, err := tierInvocation(TierEvolve)
		if err != nil {
			t.Fatal(err)
		}
		if push {
			t.Error("tier 2 must not push")
		}
	})
	t.Run("tier3 pushes", func(t *testing.T) {
		_, push, err := tierInvocation(TierPush)
		if err != nil {
			t.Fatal(err)
		}
		if !push {
			t.Error("tier 3 must push")
		}
	})
	t.Run("invalid tier errors", func(t *testing.T) {
		if _, _, err := tierInvocation(0); err == nil {
			t.Error("tier 0 should error")
		}
		if _, _, err := tierInvocation(4); err == nil {
			t.Error("tier 4 should error")
		}
	})
}

func TestRegistryRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Instances) != 0 {
		t.Fatalf("fresh registry should be empty, got %d", len(reg.Instances))
	}

	reg.upsert(Instance{Dir: "/a", Tier: 1, BudgetCents: 300, Enabled: false})
	reg.upsert(Instance{Dir: "/b", Tier: 3, BudgetCents: 500, Enabled: true})
	if err := reg.save(); err != nil {
		t.Fatal(err)
	}

	got, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Instances) != 2 {
		t.Fatalf("want 2 instances, got %d", len(got.Instances))
	}
	// upsert edits in place, does not duplicate
	got.upsert(Instance{Dir: "/a", Tier: 2, BudgetCents: 300, Enabled: false})
	if len(got.Instances) != 2 {
		t.Fatalf("upsert of existing dir must not add a row, got %d", len(got.Instances))
	}
	if in := got.find("/a"); in == nil || in.Tier != 2 {
		t.Errorf("upsert should have edited /a to tier 2, got %+v", in)
	}
	if !got.remove("/a") || got.find("/a") != nil {
		t.Error("remove /a failed")
	}
}

func TestConfigPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got, want := configPath(), filepath.FromSlash("/xdg/winze/metabolize.json"); got != want {
		t.Errorf("configPath = %q, want %q", got, want)
	}
}
