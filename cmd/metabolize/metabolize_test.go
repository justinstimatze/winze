package main

import (
	"os"
	"os/exec"
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
		// reify regenerates tracked predictions.go — a write. A sense-only tier
		// must not carry it, or an unattended timer dirties the tree each tick.
		if strings.Contains(joined, "reify") {
			t.Errorf("tier 1 must exclude the reify phase (writes predictions.go), got %q", joined)
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

// TestNormalizeInstanceDir guards the systemd %f/%I escaping round-trip: a path
// that lost its leading slash (the %I bug) must be restored to absolute, not
// resolved against the service's home working dir (which doubled it).
func TestNormalizeInstanceDir(t *testing.T) {
	// A real absolute dir passes through unchanged.
	abs := t.TempDir()
	if got := normalizeInstanceDir(abs); got != abs {
		t.Errorf("absolute dir changed: got %q want %q", got, abs)
	}
	// The lost-slash form of that same dir (leading slash stripped) is restored.
	stripped := strings.TrimPrefix(abs, "/")
	if got := normalizeInstanceDir(stripped); got != abs {
		t.Errorf("lost-slash %q not restored: got %q want %q", stripped, got, abs)
	}
	// A genuinely relative path that is not a lost-slash absolute is left alone
	// (rooting it at / names nothing), so Abs still resolves it against cwd.
	rel := "definitely/not/a/real/rooted/dir/xyzzy"
	if got := normalizeInstanceDir(rel); got != rel {
		t.Errorf("plain relative path changed: got %q want %q", got, rel)
	}
}

// TestChangedRootGoFiles pins the sweep's discriminating logic: it must pick up
// root-level *.go the cycle changed (predictions.go, metabolism_cycle*.go) and
// never a tooling file under a subdir. Git's `*.go` pathspec matches at any
// depth, so the no-slash filter is the load-bearing part — a regression would
// let commitCycleState sweep cmd/ or internal/ edits into an autonomous commit.
func TestChangedRootGoFiles(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "t@t")
	git("config", "user.name", "t")
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A committed clean baseline.
	write("schema.go", "package winze\n")
	git("add", "-A")
	git("commit", "-qm", "base")

	// The cycle's writes: a modified root file, a new root cycle file, and a
	// tooling edit under cmd/ that must NOT be swept.
	write("schema.go", "package winze\n\n// touched\n")
	write("metabolism_cycle9.go", "package winze\n")
	write("predictions.go", "package winze\n")
	write("cmd/metabolize/run.go", "package main\n")

	got, err := changedRootGoFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"schema.go": true, "metabolism_cycle9.go": true, "predictions.go": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the 3 root files %v", got, want)
	}
	for _, f := range got {
		if !want[f] {
			t.Errorf("swept %q — not an expected root corpus file (a subdir/tooling file leaked in)", f)
		}
	}
}
