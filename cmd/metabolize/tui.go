package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The TUI is the interactive cockpit over the same read-model the `status`
// subcommand prints: one row per instance, keys to promote tier, adjust budget,
// pause/resume the timer, and fire a run now. It mutates the registry and drives
// systemd; it never schedules anything itself.

func cmdTUI(_ []string) error {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type refreshMsg struct {
	rows []InstanceStatus
	err  error
}
type tickMsg time.Time

type model struct {
	rows   []InstanceStatus
	cursor int
	msg    string
	err    error
	w, h   int
}

func newModel() model { return model{msg: "loading…"} }

func (m model) Init() tea.Cmd { return tea.Batch(refreshCmd, tickCmd()) }

func refreshCmd() tea.Msg {
	rows, err := gatherStatus()
	return refreshMsg{rows: rows, err: err}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case refreshMsg:
		m.rows, m.err = msg.rows, msg.err
		if m.cursor >= len(m.rows) {
			m.cursor = max(0, len(m.rows)-1)
		}
		return m, nil
	case tickMsg:
		return m, tea.Batch(refreshCmd, tickCmd())
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "g":
		return m, refreshCmd
	}

	// The rest act on the selected row.
	cur := m.current()
	if cur == nil {
		return m, nil
	}
	switch k.String() {
	case "t": // cycle autonomy tier 1→2→3→1
		next := cur.Tier + 1
		if next > TierPush {
			next = TierSenseOnly
		}
		if err := setTier(cur.Dir, next); err != nil {
			m.msg = "tier: " + err.Error()
		} else {
			m.msg = fmt.Sprintf("%s → tier %d (%s)", short(cur.Dir), next, tierName(next))
		}
		return m, refreshCmd
	case "+", "=":
		m.msg = m.bumpBudget(cur, 50)
		return m, refreshCmd
	case "-", "_":
		m.msg = m.bumpBudget(cur, -50)
		return m, refreshCmd
	case "p", " ": // pause/resume the timer
		if cur.Enabled {
			if err := disableInstance(cur.Dir); err != nil {
				m.msg = "pause: " + err.Error()
			} else {
				m.msg = short(cur.Dir) + " paused"
			}
		} else {
			if err := enableInstance(cur.Dir); err != nil {
				m.msg = "resume: " + err.Error()
			} else {
				m.msg = short(cur.Dir) + " resumed"
			}
		}
		return m, refreshCmd
	case "r": // run one cycle now
		if err := startService(cur.Dir); err != nil {
			m.msg = "run: " + err.Error()
		} else {
			m.msg = short(cur.Dir) + " — run triggered"
		}
		return m, refreshCmd
	}
	return m, nil
}

func (m *model) current() *InstanceStatus {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return &m.rows[m.cursor]
}

func (m model) bumpBudget(cur *InstanceStatus, delta int) string {
	nb := cur.BudgetCents + delta
	if nb < 0 {
		nb = 0
	}
	if err := setBudget(cur.Dir, nb); err != nil {
		return "budget: " + err.Error()
	}
	return fmt.Sprintf("%s budget → %d¢", short(cur.Dir), nb)
}

// ---- styles ----

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	selStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func tierColor(t int) lipgloss.Style {
	switch t {
	case TierSenseOnly:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green: safe
	case TierEvolve:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow: writes
	case TierPush:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red: pushes
	default:
		return dimStyle
	}
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("winze metabolism") + "\n\n")

	if m.err != nil {
		b.WriteString("error: " + m.err.Error() + "\n")
		return b.String()
	}
	if len(m.rows) == 0 {
		b.WriteString(dimStyle.Render("no instances registered — winze-metabolize add <dir>") + "\n")
		b.WriteString("\n" + footerStyle.Render("[q]uit"))
		return b.String()
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-16s%-11s%-11s%-12s%s", "TIER", "TIMER", "NEXT", "BUDGET", "INSTANCE")) + "\n")
	for i, s := range m.rows {
		b.WriteString(renderRow(s, i == m.cursor) + "\n")
	}

	b.WriteString("\n")
	if m.msg != "" {
		b.WriteString(dimStyle.Render(m.msg) + "\n")
	}
	b.WriteString(footerStyle.Render("[↑/↓] move  [r]un  [p]ause/resume  [t]ier  [+/-]budget  [g]refresh  [q]uit"))
	return b.String()
}

// renderRow lays out one instance with display-width-aware cells (lipgloss
// Width counts columns, not bytes, so the unicode glyphs align). A selected row
// is the plain layout under a highlight; unselected rows color the tier by risk.
func renderRow(s InstanceStatus, selected bool) string {
	tierTxt := fmt.Sprintf("%d %s", s.Tier, tierName(s.Tier))
	timer, next := "off", "—"
	if s.Enabled {
		timer = "● on"
		if s.HasTimer && !s.TimerActive {
			timer = "○ inactive"
		}
		next = trunc(relFire(s.NextFire, s.NextFireRaw), 10)
	}
	budget := fmt.Sprintf("%d/%d¢", s.SpentCents, s.BudgetCents)
	inst := short(s.Dir)
	cursor := "  "
	if selected {
		cursor = "▸ "
		row := cursor + padCell(tierTxt, 16) + padCell(timer, 11) + padCell(next, 11) + padCell(budget, 12) + inst
		return selStyle.Render(row)
	}
	return cursor + tierColor(s.Tier).Width(16).Render(tierTxt) +
		padCell(timer, 11) + padCell(next, 11) + padCell(budget, 12) + inst
}

func padCell(s string, n int) string { return lipgloss.NewStyle().Width(n).Render(s) }

// ---- helpers shared with the CLI ----

func setTier(dir string, tier int) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	in := reg.find(dir)
	if in == nil {
		return fmt.Errorf("%s not registered", dir)
	}
	in.Tier = tier
	return reg.save()
}

func setBudget(dir string, cents int) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	in := reg.find(dir)
	if in == nil {
		return fmt.Errorf("%s not registered", dir)
	}
	in.BudgetCents = cents
	return reg.save()
}

// startService fires the instance's oneshot service now (an immediate run,
// independent of the timer). Async: systemctl returns once the job is queued.
func startService(dir string) error {
	out, err := exec.Command("systemd-escape", "--template=winze-metabolize@.service", "-p", dir).Output()
	if err != nil {
		return fmt.Errorf("escape: %w", err)
	}
	unit := strings.TrimSpace(string(out))
	return exec.Command("systemctl", "--user", "start", unit).Run()
}

func short(dir string) string {
	if i := strings.LastIndex(strings.TrimRight(dir, "/"), "/"); i >= 0 {
		return dir[i+1:]
	}
	return dir
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
