package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/commands"
	"github.com/charmbracelet/crush/internal/ui/dialog"
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

	// Bare /gsd is shorthand for /gsd next.
	if sub == "" {
		return m.gsdNext(sub), true
	}

	// Dispatch known subcommands.
	switch {
	case strings.HasPrefix(sub, "auto"):
		return m.gsdAuto(sub), true
	case sub == "next" || strings.HasPrefix(sub, "next "):
		return m.gsdNext(sub), true
	case sub == "pause":
		return m.gsdPause(), true
	case sub == "stop":
		return m.gsdStop(), true
	case sub == "status":
		return m.gsdStatus(), true
	case sub == "queue":
		return m.gsdQueue(), true
	case sub == "undo":
		return m.gsdUndo(), true
	case sub == "skip" || strings.HasPrefix(sub, "skip "):
		return m.gsdSkip(sub), true
	case strings.HasPrefix(sub, "dispatch"):
		return m.gsdDispatch(sub), true
	case strings.HasPrefix(sub, "steer"):
		return m.gsdSteer(sub), true
	case sub == "history" || strings.HasPrefix(sub, "history "):
		return m.gsdHistory(sub), true
	case strings.HasPrefix(sub, "rate"):
		return m.gsdRate(sub), true
	case sub == "doctor" || sub == "doctor fix":
		return m.gsdDoctor(sub), true
	case strings.HasPrefix(sub, "quick"):
		return m.gsdQuick(sub), true
	case strings.HasPrefix(sub, "init"):
		return m.gsdInit(sub), true
	case strings.HasPrefix(sub, "start"):
		return m.gsdStart(sub), true
	case sub == "park" || strings.HasPrefix(sub, "park "):
		return m.gsdPark(sub), true
	case sub == "unpark" || strings.HasPrefix(sub, "unpark "):
		return m.gsdUnpark(sub), true
	case sub == "rethink":
		return m.gsdRethink(), true
	case sub == "prefs" || strings.HasPrefix(sub, "prefs "):
		return m.gsdPrefs(sub), true
	case sub == "cleanup":
		return m.gsdCleanup(), true
	case sub == "help":
		return gsdHelp(), true
	default:
		cmd := strings.Fields(sub)[0]
		return util.ReportWarn("Unknown /gsd command: " + cmd + ". Type /gsd help for available commands."), true
	}
}

// gsdHelp returns a tea.Cmd that shows the /gsd command help text.
func gsdHelp() tea.Cmd {
	help := strings.Join([]string{
		"/gsd                    — execute next unit then pause (alias for /gsd next)",
		"/gsd next [milestone]   — execute next unit then pause",
		"/gsd auto [milestone]   — start autonomous execution",
		"/gsd pause              — pause after current unit completes",
		"/gsd stop               — stop auto-mode immediately",
		"/gsd status             — show progress dashboard",
		"/gsd queue              — show pending dispatch queue",
		"/gsd undo               — revert the last completed task",
		"/gsd skip <task-id>     — skip a task from auto-dispatch",
		"/gsd dispatch <phase>   — dispatch a specific phase",
		"/gsd steer <text>       — inject guidance into active work",
		"/gsd history [N]        — view execution history (last N units)",
		"/gsd rate <over|ok|under> — rate last unit's model tier",
		"/gsd doctor [fix]       — health checks with optional auto-heal",
		"/gsd quick <task>       — execute a quick task",
		"/gsd init <vision>      — interactive planning from a vision",
		"/gsd start <template>   — start from workflow template",
		"/gsd park [milestone]   — park a milestone",
		"/gsd unpark [milestone] — reactivate a parked milestone",
		"/gsd rethink            — conversational replan",
		"/gsd prefs [key=value]  — view/set preferences",
		"/gsd cleanup            — remove stale worktrees",
		"/gsd help               — show this help",
	}, "\n")
	return util.ReportNotice(help)
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

// gsdStop handles /gsd stop — cancels auto-mode immediately.
func (m *UI) gsdStop() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	return func() tea.Msg {
		if err := m.autoController.StopAuto(); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Auto-mode stopped")
	}
}

// gsdNext handles /gsd next [milestone-id] — executes one unit then pauses.
func (m *UI) gsdNext(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	milestoneID := strings.TrimSpace(strings.TrimPrefix(sub, "next"))
	if milestoneID == "" {
		milestoneID = m.autoMilestoneID
	}
	if milestoneID == "" {
		return util.ReportWarn("No milestone ID provided. Usage: /gsd next <milestone-id>")
	}

	return func() tea.Msg {
		if err := m.autoController.StepAuto(context.Background(), milestoneID); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Executing next unit for " + milestoneID)
	}
}

// gsdQueue handles /gsd queue — shows the pending dispatch queue.
func (m *UI) gsdQueue() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	milestoneID := m.autoMilestoneID
	if milestoneID == "" {
		return util.ReportWarn("No active milestone. Start auto-mode first.")
	}

	return func() tea.Msg {
		queue, err := m.autoController.AutoQueue(context.Background(), milestoneID)
		if err != nil {
			return util.NewErrorMsg(err)
		}
		if len(queue) == 0 {
			return util.NewInfoMsg("No pending units — all work complete")
		}
		header := fmt.Sprintf("Dispatch queue for %s (%d units):", milestoneID, len(queue))
		lines := make([]string, 0, len(queue)+1)
		lines = append(lines, header)
		for i, entry := range queue {
			lines = append(lines, fmt.Sprintf("  %d. %s", i+1, entry))
		}
		return util.SystemNoticeMsg{Text: strings.Join(lines, "\n")}
	}
}

// gsdStatus handles /gsd status — shows a rich progress dashboard.
func (m *UI) gsdStatus() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	status := m.autoController.AutoStatus()
	lines := []string{fmt.Sprintf("Auto-mode: %s", status)}

	if m.autoMilestoneID != "" {
		lines = append(lines, fmt.Sprintf("Milestone: %s", m.autoMilestoneID))
	}

	if m.autoSnapshot != nil {
		if m.autoSnapshot.ActiveUnit != "" {
			lines = append(lines, fmt.Sprintf("Active unit: %s", m.autoSnapshot.ActiveUnit))
		}

		// Slice progress.
		for _, s := range m.autoSnapshot.Slices {
			statusIcon := "○"
			switch {
			case s.TasksDone == s.TasksTotal && s.TasksTotal > 0:
				statusIcon = "●"
			case s.TasksDone > 0:
				statusIcon = "◐"
			}
			lines = append(lines, fmt.Sprintf("  %s %s: %d/%d tasks  [%s]",
				statusIcon, s.Title, s.TasksDone, s.TasksTotal, s.Status))
		}

		if m.autoSnapshot.TotalCost > 0 {
			lines = append(lines, fmt.Sprintf("Cost: $%.4f", m.autoSnapshot.TotalCost))
		}
		if m.autoSnapshot.ElapsedSeconds > 0 {
			elapsed := m.autoSnapshot.ElapsedSeconds
			minutes := int(elapsed) / 60
			seconds := int(elapsed) % 60
			lines = append(lines, fmt.Sprintf("Elapsed: %dm%ds", minutes, seconds))
		}
	}

	return util.ReportNotice(strings.Join(lines, "\n"))
}

// gsdUndo handles /gsd undo — reverts the last completed task.
func (m *UI) gsdUndo() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	milestoneID := m.autoMilestoneID
	if milestoneID == "" {
		return util.ReportWarn("No active milestone. Start auto-mode first.")
	}

	return func() tea.Msg {
		desc, err := m.autoController.UndoLast(context.Background(), milestoneID)
		if err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg(desc)
	}
}

// gsdSkip handles /gsd skip <task-id> — marks a task as skipped.
func (m *UI) gsdSkip(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	taskID := strings.TrimSpace(strings.TrimPrefix(sub, "skip"))
	if taskID == "" {
		return util.ReportWarn("Usage: /gsd skip <task-id>")
	}

	return func() tea.Msg {
		if err := m.autoController.SkipUnit(context.Background(), taskID); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Skipped task " + taskID)
	}
}

// gsdDispatch handles /gsd dispatch <phase> — dispatches a specific phase.
func (m *UI) gsdDispatch(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	phase := strings.TrimSpace(strings.TrimPrefix(sub, "dispatch"))
	if phase == "" {
		return util.ReportWarn("Usage: /gsd dispatch <phase> (research, plan, execute, summarize, validate)")
	}

	milestoneID := m.autoMilestoneID
	if milestoneID == "" {
		return util.ReportWarn("No active milestone. Start auto-mode first.")
	}

	return func() tea.Msg {
		if err := m.autoController.DispatchPhase(context.Background(), milestoneID, phase); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Dispatching " + phase + " for " + milestoneID)
	}
}

// gsdSteer handles /gsd steer <text> — injects guidance into active work.
func (m *UI) gsdSteer(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	guidance := strings.TrimSpace(strings.TrimPrefix(sub, "steer"))
	if guidance == "" {
		return util.ReportWarn("Usage: /gsd steer <guidance text>")
	}

	return func() tea.Msg {
		if err := m.autoController.Steer(context.Background(), guidance); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Steering guidance applied: " + guidance)
	}
}

// gsdHistory handles /gsd history [N] — shows recent execution history.
func (m *UI) gsdHistory(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	count := 10
	arg := strings.TrimSpace(strings.TrimPrefix(sub, "history"))
	if arg != "" {
		if n, err := strconv.Atoi(arg); err == nil && n > 0 {
			count = n
		}
	}

	return func() tea.Msg {
		result, err := m.autoController.History(context.Background(), count)
		if err != nil {
			return util.NewErrorMsg(err)
		}
		return util.SystemNoticeMsg{Text: result}
	}
}

// gsdRate handles /gsd rate <over|ok|under> — rates model tier selection.
func (m *UI) gsdRate(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	rating := strings.TrimSpace(strings.TrimPrefix(sub, "rate"))
	switch rating {
	case "over", "ok", "under":
		// Valid ratings.
	default:
		return util.ReportWarn("Usage: /gsd rate <over|ok|under>")
	}

	return func() tea.Msg {
		if err := m.autoController.RateTier(context.Background(), rating); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Tier rated as: " + rating)
	}
}

// gsdDoctor handles /gsd doctor [fix] — runs health checks.
func (m *UI) gsdDoctor(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	fix := strings.Contains(sub, "fix")

	return func() tea.Msg {
		result, err := m.autoController.RunDoctor(context.Background(), fix)
		if err != nil {
			return util.NewErrorMsg(err)
		}
		return util.SystemNoticeMsg{Text: result}
	}
}

// gsdQuick handles /gsd quick <task> — executes a quick task.
func (m *UI) gsdQuick(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	description := strings.TrimSpace(strings.TrimPrefix(sub, "quick"))
	if description == "" {
		return util.ReportWarn("Usage: /gsd quick <task description>")
	}

	milestoneID := m.autoMilestoneID
	if milestoneID == "" {
		return util.ReportWarn("No active milestone. Start auto-mode first.")
	}

	return func() tea.Msg {
		if err := m.autoController.QuickTask(context.Background(), milestoneID, description); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Quick task dispatched: " + description)
	}
}

// gsdInit handles /gsd init <vision> — runs interactive planning.
func (m *UI) gsdInit(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	vision := strings.TrimSpace(strings.TrimPrefix(sub, "init"))
	if vision == "" {
		return util.ReportWarn("Usage: /gsd init <vision description>")
	}

	// Show immediate feedback, then run init asynchronously.
	return tea.Batch(
		util.ReportInfo("Initializing project — this may take a moment..."),
		func() tea.Msg {
			if err := m.autoController.InitProject(context.Background(), vision); err != nil {
				return util.SystemNoticeMsg{Text: "Init failed: " + err.Error()}
			}
			return util.SystemNoticeMsg{Text: "Project initialized from vision: " + vision}
		},
	)
}

// gsdStart handles /gsd start <template> — starts from a workflow template.
func (m *UI) gsdStart(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	templateID := strings.TrimSpace(strings.TrimPrefix(sub, "start"))
	if templateID == "" {
		return util.ReportWarn("Usage: /gsd start <template> (bugfix, feature, spike, hotfix, refactor)")
	}

	return func() tea.Msg {
		name, err := m.autoController.StartFromTemplate(context.Background(), templateID)
		if err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Template loaded: " + name)
	}
}

// gsdPark handles /gsd park [milestone] — parks a milestone.
func (m *UI) gsdPark(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	milestoneID := strings.TrimSpace(strings.TrimPrefix(sub, "park"))
	if milestoneID == "" {
		milestoneID = m.autoMilestoneID
	}
	if milestoneID == "" {
		return util.ReportWarn("No milestone ID provided. Usage: /gsd park <milestone-id>")
	}

	return func() tea.Msg {
		if err := m.autoController.ParkMilestone(context.Background(), milestoneID); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Milestone parked: " + milestoneID)
	}
}

// gsdUnpark handles /gsd unpark [milestone] — reactivates a parked milestone.
func (m *UI) gsdUnpark(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	milestoneID := strings.TrimSpace(strings.TrimPrefix(sub, "unpark"))
	if milestoneID == "" {
		milestoneID = m.autoMilestoneID
	}
	if milestoneID == "" {
		return util.ReportWarn("No milestone ID provided. Usage: /gsd unpark <milestone-id>")
	}

	return func() tea.Msg {
		if err := m.autoController.UnparkMilestone(context.Background(), milestoneID); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Milestone unparked: " + milestoneID)
	}
}

// gsdRethink handles /gsd rethink — conversational replan.
func (m *UI) gsdRethink() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	milestoneID := m.autoMilestoneID
	if milestoneID == "" {
		return util.ReportWarn("No active milestone. Start auto-mode first.")
	}

	return func() tea.Msg {
		if err := m.autoController.Rethink(context.Background(), milestoneID); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Rethink dispatched for " + milestoneID)
	}
}

// gsdPrefs handles /gsd prefs [key=value] — view or set preferences.
func (m *UI) gsdPrefs(sub string) tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	arg := strings.TrimSpace(strings.TrimPrefix(sub, "prefs"))

	// If no argument, show current preferences.
	if arg == "" {
		return func() tea.Msg {
			result, err := m.autoController.GetPreferences()
			if err != nil {
				return util.NewErrorMsg(err)
			}
			return util.SystemNoticeMsg{Text: result}
		}
	}

	// Parse key=value.
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) != 2 {
		return util.ReportWarn("Usage: /gsd prefs [key=value]")
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	return func() tea.Msg {
		if err := m.autoController.SetPreference(key, value); err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg("Preference set: " + key + "=" + value)
	}
}

// gsdCleanup handles /gsd cleanup — removes stale worktrees.
func (m *UI) gsdCleanup() tea.Cmd {
	if m.autoController == nil {
		return util.ReportWarn("Auto-mode not available")
	}

	return func() tea.Msg {
		result, err := m.autoController.CleanupWorktrees(context.Background())
		if err != nil {
			return util.NewErrorMsg(err)
		}
		return util.NewInfoMsg(result)
	}
}

// openGSDArgDialog opens an Arguments dialog to collect input for a GSD command.
// The cmdTemplate should contain $ARG_ID placeholders (e.g., "init $VISION").
func (m *UI) openGSDArgDialog(title, cmdTemplate string, args ...commands.Argument) {
	argsDialog := dialog.NewArguments(
		m.com,
		title,
		"",
		args,
		dialog.ActionGSDWithArg{Command: cmdTemplate},
	)
	m.dialog.OpenDialog(argsDialog)
}
