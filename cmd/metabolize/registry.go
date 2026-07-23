package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// The registry is the per-instance opt-in surface: which winze corpora
// auto-metabolize, at what autonomy tier, under what budget. One JSON file the
// runner reads, the CLI edits, and (slice 3) the TUI drives. Cadence itself
// lives in systemd, never here — this only records intent and tier.

// Tier is the autonomy level of a metabolizing instance. It is promoted
// per-instance as trust in the loop grows; there is no global mode.
const (
	TierSenseOnly = 1 // sense + cheap phases; never mutates the corpus (~free)
	TierEvolve    = 2 // full --evolve; commits locally; never pushes
	TierPush      = 3 // full --evolve; commits AND pushes (fully autonomous)
)

const defaultBudgetCents = 300

// Instance is one registered winze corpus.
type Instance struct {
	Dir         string `json:"dir"`          // absolute path to the corpus
	Tier        int    `json:"tier"`         // 1..3, see Tier* constants
	BudgetCents int    `json:"budget_cents"` // METABOLISM_BUDGET_CENTS for this instance
	Enabled     bool   `json:"enabled"`      // mirrors whether the systemd timer is active
}

// Registry is the whole opt-in set, persisted to configPath.
type Registry struct {
	Instances []Instance `json:"instances"`
}

// configPath is $XDG_CONFIG_HOME/winze/metabolize.json (or ~/.config/...).
func configPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "winze", "metabolize.json")
}

// loadRegistry reads the registry, returning an empty one if the file is absent.
func loadRegistry() (*Registry, error) {
	data, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("corrupt registry %s: %w", configPath(), err)
	}
	return &r, nil
}

func (r *Registry) save() error {
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o755); err != nil {
		return err
	}
	sort.Slice(r.Instances, func(i, j int) bool { return r.Instances[i].Dir < r.Instances[j].Dir })
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), append(data, '\n'), 0o644)
}

// find returns a pointer to the instance with the given absolute dir, or nil.
func (r *Registry) find(dir string) *Instance {
	for i := range r.Instances {
		if r.Instances[i].Dir == dir {
			return &r.Instances[i]
		}
	}
	return nil
}

// upsert adds or replaces the instance for dir and returns the stored copy.
func (r *Registry) upsert(in Instance) {
	if existing := r.find(in.Dir); existing != nil {
		*existing = in
		return
	}
	r.Instances = append(r.Instances, in)
}

func (r *Registry) remove(dir string) bool {
	for i := range r.Instances {
		if r.Instances[i].Dir == dir {
			r.Instances = append(r.Instances[:i], r.Instances[i+1:]...)
			return true
		}
	}
	return false
}

// absCorpusDir resolves dir to an absolute path and checks it looks like a
// winze corpus (a directory holding at least one .go file). The build gate is
// the real validator downstream; this just catches obvious typos early.
func absCorpusDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	matches, _ := filepath.Glob(filepath.Join(abs, "*.go"))
	if len(matches) == 0 {
		return "", fmt.Errorf("%s has no .go files — not a winze corpus", abs)
	}
	return abs, nil
}

func tierName(t int) string {
	switch t {
	case TierSenseOnly:
		return "sense-only"
	case TierEvolve:
		return "evolve/local"
	case TierPush:
		return "evolve/push"
	default:
		return fmt.Sprintf("tier?%d", t)
	}
}

func validTier(t int) bool { return t >= TierSenseOnly && t <= TierPush }
