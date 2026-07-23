package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed units/winze-metabolize@.service units/winze-metabolize@.timer
var unitFS embed.FS

// systemd --user is the cadence layer: it fires the timer whether or not
// anyone is logged in (with `loginctl enable-linger`), on a laptop or a
// headless EC2 box alike. winze-metabolize never schedules anything itself; it
// only installs the units and toggles them.

const userUnitService = "winze-metabolize@.service"
const userUnitTimer = "winze-metabolize@.timer"

func userUnitDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "systemd", "user")
}

// installUnits writes the two template units into the user systemd dir with the
// runner and WINZE_BIN paths resolved to this binary's own location, then
// reloads. Idempotent — safe to re-run after a `make install`.
func installUnits() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, _ = filepath.EvalSymlinks(self)
	binDir := filepath.Dir(self)

	dir := userUnitDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	svc, err := unitFS.ReadFile("units/" + userUnitService)
	if err != nil {
		return err
	}
	rendered := strings.NewReplacer("__RUNNER__", self, "__WINZE_BIN__", binDir).Replace(string(svc))
	if err := os.WriteFile(filepath.Join(dir, userUnitService), []byte(rendered), 0o644); err != nil {
		return err
	}
	timer, err := unitFS.ReadFile("units/" + userUnitTimer)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, userUnitTimer), timer, 0o644); err != nil {
		return err
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	fmt.Printf("installed units to %s (runner: %s)\n", dir, self)
	fmt.Println("units use winze-metabolism from:", binDir)
	fmt.Println("\nfor unattended running on a headless box: loginctl enable-linger $USER")
	return nil
}

// timerUnit returns the full instance timer unit name for a corpus dir, using
// systemd-escape so the path becomes a valid instance token.
func timerUnit(dir string) (string, error) {
	out, err := exec.Command("systemd-escape", "--template=winze-metabolize@.timer", "-p", dir).Output()
	if err != nil {
		return "", fmt.Errorf("systemd-escape %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func systemctl(args ...string) error {
	full := append([]string{"--user"}, args...)
	cmd := exec.Command("systemctl", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// enableInstance flips the instance on: mark it enabled in the registry and
// start+enable its timer. The corpus must already be registered (`add`).
func enableInstance(dir string) error {
	abs, err := absCorpusDir(dir)
	if err != nil {
		return err
	}
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	in := reg.find(abs)
	if in == nil {
		return fmt.Errorf("%s not registered — run: winze-metabolize add %s", abs, abs)
	}
	unit, err := timerUnit(abs)
	if err != nil {
		return err
	}
	if err := systemctl("enable", "--now", unit); err != nil {
		return fmt.Errorf("enable %s: %w", unit, err)
	}
	in.Enabled = true
	if err := reg.save(); err != nil {
		return err
	}
	fmt.Printf("enabled %s (tier %d, %s) — timer %s active\n", abs, in.Tier, tierName(in.Tier), unit)
	return nil
}

// disableInstance stops the timer and marks the instance disabled. The runner
// also no-ops on a disabled instance, so this is belt-and-suspenders.
func disableInstance(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	unit, err := timerUnit(abs)
	if err != nil {
		return err
	}
	if err := systemctl("disable", "--now", unit); err != nil {
		return fmt.Errorf("disable %s: %w", unit, err)
	}
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	if in := reg.find(abs); in != nil {
		in.Enabled = false
		if err := reg.save(); err != nil {
			return err
		}
	}
	fmt.Printf("disabled %s — timer %s stopped\n", abs, unit)
	return nil
}
