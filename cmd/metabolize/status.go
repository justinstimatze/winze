package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// The status read-model joins three sources per instance: the registry (tier,
// budget cap, enabled), the corpus's .metabolism-budget.json (spent this
// month), and systemd (timer active + next/last fire). Both the `status`
// subcommand and the TUI render this same struct, so they never drift.

type InstanceStatus struct {
	Instance
	SpentCents       int       // this month, from .metabolism-budget.json
	Month            string    // budget accounting month (YYYY-MM)
	CacheReadTokens  int64     // input served from prompt cache this month
	FreshInputTokens int64     // uncached input this month
	TimerActive      bool      // systemd timer ActiveState == active
	HasTimer         bool      // systemctl query returned a unit
	NextFire         time.Time // zero if unparseable
	NextFireRaw      string    // raw systemd string (always shown if present)
	LastRun          time.Time
	LastRunRaw       string
}

// gatherStatus builds the full read-model for every registered instance.
func gatherStatus() ([]InstanceStatus, error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	out := make([]InstanceStatus, 0, len(reg.Instances))
	for _, in := range reg.Instances {
		st := InstanceStatus{Instance: in}
		st.SpentCents, st.Month, st.CacheReadTokens, st.FreshInputTokens = readBudget(in.Dir)
		if in.Enabled {
			readTimer(in.Dir, &st)
		}
		out = append(out, st)
	}
	return out, nil
}

func readBudget(dir string) (spent int, month string, cacheRead, freshInput int64) {
	data, err := os.ReadFile(filepath.Join(dir, ".metabolism-budget.json"))
	if err != nil {
		return 0, "", 0, 0
	}
	var b struct {
		SpentCents       int    `json:"spent_cents"`
		Month            string `json:"month"`
		CacheReadTokens  int64  `json:"cache_read_tokens"`
		FreshInputTokens int64  `json:"fresh_input_tokens"`
	}
	if json.Unmarshal(data, &b) != nil {
		return 0, "", 0, 0
	}
	return b.SpentCents, b.Month, b.CacheReadTokens, b.FreshInputTokens
}

// cacheCell renders the prompt-cache hit ratio for the status table, or "—"
// before any LLM call has been billed this month. Read/(read+fresh) — the
// same measure the metabolism cycle line prints.
func cacheCell(cacheRead, freshInput int64) string {
	total := cacheRead + freshInput
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(cacheRead)/float64(total))
}

// readTimer fills the systemd-derived fields. Best-effort: any failure leaves
// HasTimer false and the caller shows "?", never errors the whole view.
func readTimer(dir string, st *InstanceStatus) {
	unit, err := timerUnit(dir)
	if err != nil {
		return
	}
	out, err := exec.Command("systemctl", "--user", "show", unit,
		"--property=ActiveState,NextElapseUSecRealtime,LastTriggerUSec").Output()
	if err != nil {
		return
	}
	st.HasTimer = true
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "ActiveState":
			st.TimerActive = val == "active"
		case "NextElapseUSecRealtime":
			st.NextFireRaw = val
			if t, ok := parseSystemdTime(val); ok {
				st.NextFire = t
			}
		case "LastTriggerUSec":
			st.LastRunRaw = val
			if t, ok := parseSystemdTime(val); ok {
				st.LastRun = t
			}
		}
	}
}

// parseSystemdTime parses systemd's realtime rendering ("Thu 2026-07-23
// 09:00:00 PDT"). Zone abbreviations don't round-trip reliably in Go, so a
// failure is expected and non-fatal — callers fall back to the raw string.
func parseSystemdTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "n/a" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("Mon 2006-01-02 15:04:05 MST", s, time.Local); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// relFire renders a next/last fire as a compact relative string when parseable,
// else the raw systemd string, else "?".
func relFire(t time.Time, raw string) string {
	if !t.IsZero() {
		d := time.Until(t)
		if d >= 0 {
			return "in " + compactDur(d)
		}
		return compactDur(-d) + " ago"
	}
	if raw != "" && raw != "n/a" {
		return raw
	}
	return "?"
}

func compactDur(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

// cmdStatus prints the read-model as a table. `watch winze-metabolize status`
// gives a live view without the interactive TUI.
func cmdStatus(_ []string) error {
	sts, err := gatherStatus()
	if err != nil {
		return err
	}
	if len(sts) == 0 {
		fmt.Println("no instances registered — winze-metabolize add <dir>")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "TIER\tTIMER\tNEXT\tLAST\tBUDGET\tCACHE\tDIR")
	for _, s := range sts {
		timer := "off"
		if s.Enabled {
			timer = "enabled"
			if s.HasTimer && !s.TimerActive {
				timer = "inactive"
			}
		}
		next, last := "—", "—"
		if s.Enabled {
			next = relFire(s.NextFire, s.NextFireRaw)
			last = relFire(s.LastRun, s.LastRunRaw)
		}
		fmt.Fprintf(w, "%d %s\t%s\t%s\t%s\t%d/%d¢\t%s\t%s\n",
			s.Tier, tierName(s.Tier), timer, next, last, s.SpentCents, s.BudgetCents,
			cacheCell(s.CacheReadTokens, s.FreshInputTokens), s.Dir)
	}
	return w.Flush()
}
