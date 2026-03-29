package model

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/util"
)

// toggleAutoMode cycles auto-mode through idle→start, running→pause,
// paused→resume based on the current snapshot status.
func (m *UI) toggleAutoMode() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}
	if m.session == nil {
		return util.ReportWarn("No active session")
	}

	status := "idle"
	if m.autoSnapshot != nil && m.autoSnapshot.Status != "" {
		status = m.autoSnapshot.Status
	}

	switch status {
	case "idle":
		if m.autoMilestoneID == "" {
			return util.ReportWarn("No milestone configured for auto-mode")
		}
		return func() tea.Msg {
			if err := m.autoController.StartAuto(context.Background(), m.autoMilestoneID); err != nil {
				return util.NewErrorMsg(err)
			}
			return util.NewInfoMsg("Auto-mode started")
		}
	case "running":
		return func() tea.Msg {
			if err := m.autoController.PauseAuto(); err != nil {
				return util.NewErrorMsg(err)
			}
			return util.NewInfoMsg("Auto-mode paused")
		}
	case "paused":
		return func() tea.Msg {
			if err := m.autoController.ResumeAuto(context.Background()); err != nil {
				return util.NewErrorMsg(err)
			}
			return util.NewInfoMsg("Auto-mode resumed")
		}
	default:
		return util.ReportWarn("Auto-mode is in state: " + status)
	}
}
