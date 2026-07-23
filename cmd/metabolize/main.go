// Command winze-metabolize is the activation and control layer for autonomous
// winze metabolism: a per-instance registry of which corpora auto-evolve, at
// what autonomy tier, under what budget, plus thin wrappers over the systemd
// --user timers that provide the cadence.
//
//	winze-metabolize install                 # install the systemd template units
//	winze-metabolize add <dir> [--tier N]    # register a corpus (default tier 1)
//	winze-metabolize enable <dir>            # start its hourly timer
//	winze-metabolize list [--json]           # show registered instances
//	winze-metabolize disable <dir>           # stop its timer
//	winze-metabolize remove <dir>            # forget it
//	winze-metabolize run <dir>               # one run now (what the timer calls)
//
// Cadence lives in systemd; tier policy lives in run.go; this binary only
// records intent and drives the timers. Slice 3 (a Bubble Tea TUI) wraps these
// same operations in an interactive cockpit.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "install":
		err = installUnits()
	case "add":
		err = cmdAdd(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "status":
		err = cmdStatus(os.Args[2:])
	case "tui":
		err = cmdTUI(os.Args[2:])
	case "remove", "rm":
		err = cmdRemove(os.Args[2:])
	case "enable":
		err = cmdOneDir(os.Args[2:], enableInstance)
	case "disable":
		err = cmdOneDir(os.Args[2:], disableInstance)
	case "run":
		err = cmdOneDir(os.Args[2:], runInstance)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "winze-metabolize: unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "winze-metabolize:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `winze-metabolize — activation + control for autonomous winze metabolism

  install                     install systemd --user template units
  add <dir> [--tier N]        register a corpus (tier 1-3, default 1); [--budget ¢]
  enable <dir>                start the hourly timer for a registered corpus
  disable <dir>               stop its timer
  list [--json]               show registered instances
  status                      live table: timer state, next/last fire, budget
  tui                         interactive cockpit (run-now, pause, tier, budget)
  remove <dir>                forget a corpus
  run <dir>                   run one metabolism cycle now (the timer's ExecStart)

tiers:  1 sense-only (no writes, ~free)   2 evolve/local (commits)   3 evolve/push
`)
}

func cmdAdd(args []string) error {
	// dir leads, flags follow: the stdlib flag parser stops at the first
	// positional, so `add <dir> --tier N` is the only order that parses cleanly.
	if len(args) == 0 || (args[0] != "" && args[0][0] == '-') {
		return fmt.Errorf("usage: winze-metabolize add <dir> [--tier N] [--budget ¢]")
	}
	dir := args[0]
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	tier := fs.Int("tier", TierSenseOnly, "autonomy tier 1-3")
	budget := fs.Int("budget", defaultBudgetCents, "monthly budget cap in cents")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected extra args after <dir>: %v", fs.Args())
	}
	if !validTier(*tier) {
		return fmt.Errorf("tier must be 1, 2, or 3 (got %d)", *tier)
	}
	abs, err := absCorpusDir(dir)
	if err != nil {
		return err
	}
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	// A fresh add is registered but not timer-enabled until `enable`. On a
	// re-add (a tier/budget edit) preserve the running timer's enabled state.
	enabled := false
	if existing := reg.find(abs); existing != nil {
		enabled = existing.Enabled
	}
	reg.upsert(Instance{Dir: abs, Tier: *tier, BudgetCents: *budget, Enabled: enabled})
	if err := reg.save(); err != nil {
		return err
	}
	fmt.Printf("registered %s — tier %d (%s), budget %d¢\n", abs, *tier, tierName(*tier), *budget)
	if !enabled {
		fmt.Printf("enable it:  winze-metabolize enable %s\n", abs)
	}
	return nil
}

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "emit JSON (for the TUI / scripts)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	if *asJSON {
		out, _ := json.MarshalIndent(reg, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	if len(reg.Instances) == 0 {
		fmt.Println("no instances registered — winze-metabolize add <dir>")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "TIER\tSTATE\tBUDGET\tDIR")
	for _, in := range reg.Instances {
		state := "off"
		if in.Enabled {
			state = "on"
		}
		fmt.Fprintf(w, "%d %s\t%s\t%d¢\t%s\n", in.Tier, tierName(in.Tier), state, in.BudgetCents, in.Dir)
	}
	return w.Flush()
}

func cmdRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("remove needs exactly one <dir>")
	}
	abs, err := absCorpusDirLoose(args[0])
	if err != nil {
		return err
	}
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	if !reg.remove(abs) {
		return fmt.Errorf("%s not registered", abs)
	}
	if err := reg.save(); err != nil {
		return err
	}
	fmt.Printf("removed %s (disable its timer separately if still running)\n", abs)
	return nil
}

// cmdOneDir adapts a `func(dir string) error` to a subcommand taking one arg.
func cmdOneDir(args []string, fn func(string) error) error {
	if len(args) != 1 {
		return fmt.Errorf("needs exactly one <dir>")
	}
	return fn(args[0])
}

// absCorpusDirLoose resolves to abs without requiring .go files present, so a
// moved/deleted corpus can still be removed from the registry.
func absCorpusDirLoose(dir string) (string, error) {
	if abs, err := absCorpusDir(dir); err == nil {
		return abs, nil
	}
	return filepath.Abs(dir) // vanished corpus — plain abs is enough to remove it
}
