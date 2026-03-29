package model

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/util"
)

// handleGSDCommand checks if value is a /gsd command and dispatches it.
// Returns (cmd, true) if handled, (nil, false) if not a /gsd command.
func (m *UI) handleGSDCommand(value string) (tea.Cmd, bool) {
	if !strings.HasPrefix(value, "/gsd") {
		return nil, false
	}
	// Must be exactly "/gsd" or followed by a space.
	if len(value) > 4 && value[4] != ' ' {
		return nil, false
	}

	sub := strings.TrimSpace(strings.TrimPrefix(value, "/gsd"))

	// Bare /gsd or explicit help.
	if sub == "" || sub == "help" {
		return gsdHelp(), true
	}

	// Dispatch known subcommands.
	switch {
	case strings.HasPrefix(sub, "auto"):
		return m.gsdAuto(sub), true
	case sub == "pause":
		return m.gsdPause(), true
	case sub == "stop":
		return m.gsdStop(), true
	case sub == "status":
		return m.gsdStatus(), true
	default:
		cmd := strings.Fields(sub)[0]
		return util.ReportWarn("Unknown /gsd command: " + cmd + ". Type /gsd help for available commands."), true
	}
}

// gsdHelp returns a tea.Cmd that shows the /gsd command help text.
func gsdHelp() tea.Cmd {
	help := strings.Join([]string{
		"/gsd help               — show this help",
		"/gsd auto <milestone>   — start auto-mode for a milestone",
		"/gsd pause              — pause auto-mode",
		"/gsd stop               — stop auto-mode",
		"/gsd status             — show auto-mode status",
	}, "\n")
	return util.ReportInfo(help)
}

// gsdAuto handles /gsd auto [milestone-id].
func (m *UI) gsdAuto(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	// Extract milestone ID from "auto M001" or "auto".
	milestoneID := strings.TrimSpace(strings.TrimPrefix(sub, "auto"))
	if milestoneID == "" {
		milestoneID = m.autoMilestoneID
	}
	if milestoneID == "" {
		return util.ReportWarn("No milestone ID provided. Usage: /gsd auto <milestone-id>")
	}

	return func() tea.Msg {
		if err := m.autoController.StartAuto(context.Background(), milestoneID); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Auto-mode started for " + milestoneID)
	}
}

// gsdPause handles /gsd pause.
func (m *UI) gsdPause() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	return func() tea.Msg {
		if err := m.autoController.PauseAuto(); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Auto-mode paused")
	}
}

// gsdStop handles /gsd stop. Maps to PauseAuto for now.
func (m *UI) gsdStop() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	return func() tea.Msg {
		if err := m.autoController.PauseAuto(); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Auto-mode stopped")
	}
}

// gsdStatus handles /gsd status.
func (m *UI) gsdStatus() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	status := m.autoController.AutoStatus()
	lines := []string{fmt.Sprintf("Auto-mode status: %s", status)}

	if m.autoMilestoneID != "" {
		lines = append(lines, fmt.Sprintf("Milestone: %s", m.autoMilestoneID))
	}

	if m.autoSnapshot != nil {
		if m.autoSnapshot.ActiveUnit != "" {
			lines = append(lines, fmt.Sprintf("Active unit: %s", m.autoSnapshot.ActiveUnit))
		}
		if m.autoSnapshot.TotalCost > 0 {
			lines = append(lines, fmt.Sprintf("Total cost: $%.2f", m.autoSnapshot.TotalCost))
		}
	}

	return util.ReportInfo(strings.Join(lines, "\n"))
}
