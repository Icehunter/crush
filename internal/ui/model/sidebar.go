package model

import (
	"cmp"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/logo"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
	"github.com/charmbracelet/x/ansi"
)

// modelInfo renders the current model information including reasoning
// settings and context usage/cost for the sidebar.
func (m *UI) modelInfo(width int) string {
	// Use the last assistant message's model if we have one (reflects actual
	// tier used), otherwise fall back to the configured main model.
	displayModel := m.selectedLargeModel()
	reasoningInfo := ""
	providerName := ""
	modelName := ""

	cfg := m.com.Config()

	if m.lastAssistantMsgModel != "" {
		// Show the model that actually answered the last prompt.
		providerName = m.lastAssistantMsgProvider
		if pc, ok := cfg.Providers.Get(m.lastAssistantMsgProvider); ok {
			providerName = pc.Name
		}
		modelName = m.lastAssistantMsgModel
		if catwalkModel := cfg.GetModel(m.lastAssistantMsgProvider, m.lastAssistantMsgModel); catwalkModel != nil {
			modelName = cmp.Or(catwalkModel.Name, m.lastAssistantMsgModel)
		}
	} else if displayModel != nil {
		providerConfig, ok := cfg.Providers.Get(displayModel.ModelCfg.Provider)
		if ok {
			providerName = providerConfig.Name

			// Only check reasoning if model can reason
			if displayModel.CatwalkCfg.CanReason {
				if len(displayModel.CatwalkCfg.ReasoningLevels) == 0 {
					if displayModel.ModelCfg.Think {
						reasoningInfo = "Thinking On"
					} else {
						reasoningInfo = "Thinking Off"
					}
				} else {
					reasoningEffort := cmp.Or(displayModel.ModelCfg.ReasoningEffort, displayModel.CatwalkCfg.DefaultReasoningEffort)
					reasoningInfo = fmt.Sprintf("Reasoning %s", common.FormatReasoningEffort(reasoningEffort))
				}
			}
		}
		modelName = displayModel.CatwalkCfg.Name
	}

	var modelContext *common.ModelContextInfo
	if m.session != nil {
		var contextWindow int64
		if displayModel != nil {
			contextWindow = displayModel.CatwalkCfg.ContextWindow
		}
		// Use the last-used model's context window for the percentage if available.
		if m.lastAssistantMsgModel != "" {
			if catwalkModel := cfg.GetModel(m.lastAssistantMsgProvider, m.lastAssistantMsgModel); catwalkModel != nil {
				contextWindow = catwalkModel.ContextWindow
			}
		}
		modelContext = &common.ModelContextInfo{
			ContextUsed:  m.session.CompletionTokens + m.session.PromptTokens,
			Cost:         m.session.Cost,
			ModelContext: contextWindow,
		}
	}
	return common.ModelInfo(m.com.Styles, modelName, providerName, reasoningInfo, modelContext, width)
}

// getDynamicHeightLimits will give us the num of items to show in each section based on the hight
// some items are more important than others.
func getDynamicHeightLimits(availableHeight int) (maxFiles, maxLSPs, maxMCPs int) {
	const (
		minItemsPerSection      = 2
		defaultMaxFilesShown    = 10
		defaultMaxLSPsShown     = 8
		defaultMaxMCPsShown     = 8
		minAvailableHeightLimit = 10
	)

	// If we have very little space, use minimum values
	if availableHeight < minAvailableHeightLimit {
		return minItemsPerSection, minItemsPerSection, minItemsPerSection
	}

	// Distribute available height among the three sections
	// Give priority to files, then LSPs, then MCPs
	totalSections := 3
	heightPerSection := availableHeight / totalSections

	// Calculate limits for each section, ensuring minimums
	maxFiles = max(minItemsPerSection, min(defaultMaxFilesShown, heightPerSection))
	maxLSPs = max(minItemsPerSection, min(defaultMaxLSPsShown, heightPerSection))
	maxMCPs = max(minItemsPerSection, min(defaultMaxMCPsShown, heightPerSection))

	// If we have extra space, give it to files first
	remainingHeight := availableHeight - (maxFiles + maxLSPs + maxMCPs)
	if remainingHeight > 0 {
		extraForFiles := min(remainingHeight, defaultMaxFilesShown-maxFiles)
		maxFiles += extraForFiles
		remainingHeight -= extraForFiles

		if remainingHeight > 0 {
			extraForLSPs := min(remainingHeight, defaultMaxLSPsShown-maxLSPs)
			maxLSPs += extraForLSPs
			remainingHeight -= extraForLSPs

			if remainingHeight > 0 {
				maxMCPs += min(remainingHeight, defaultMaxMCPsShown-maxMCPs)
			}
		}
	}

	return maxFiles, maxLSPs, maxMCPs
}

// autoModeInfo renders the auto-mode progress section for the sidebar. It
// returns an empty string when autoSnapshot is nil (auto-mode inactive).
func (m *UI) autoModeInfo(width int) string {
	snap := m.autoSnapshot
	if snap == nil {
		return ""
	}

	t := m.com.Styles
	var lines []string

	// Header: status icon + "Auto Mode" + status text.
	var statusIcon, statusText string
	var statusStyle lipgloss.Style
	switch snap.Status {
	case "running":
		statusIcon = "▶"
		statusText = "Running"
		statusStyle = lipgloss.NewStyle().Foreground(t.GreenDark)
	case "paused":
		statusIcon = "⏸"
		statusText = "Paused"
		statusStyle = t.Muted
	case "completed":
		statusIcon = "✓"
		statusText = "Done"
		statusStyle = lipgloss.NewStyle().Foreground(t.GreenDark)
	case "error":
		statusIcon = "✗"
		statusText = "Error"
		statusStyle = lipgloss.NewStyle().Foreground(t.Error)
	default:
		statusIcon = "○"
		statusText = cmp.Or(snap.Status, "Unknown")
		statusStyle = t.Muted
	}

	title := t.ResourceGroupTitle.Render("Auto Mode")
	header := common.Section(t, title, width, statusStyle.Render(statusIcon+" "+statusText))
	lines = append(lines, header)

	// Milestone title.
	if snap.MilestoneTitle != "" {
		mt := ansi.Truncate(snap.MilestoneTitle, width, "…")
		lines = append(lines, t.Muted.Render(mt))
	}

	// Slice tree.
	for _, sl := range snap.Slices {
		var icon string
		switch sl.Status {
		case "completed":
			icon = "✓"
		case "active":
			icon = "▶"
		case "blocked":
			icon = "✗"
		default:
			icon = "○"
		}
		progress := fmt.Sprintf("%d/%d", sl.TasksDone, sl.TasksTotal)
		// icon + space + title + space + progress = icon(1) + 1 + title + 1 + progress.
		titleWidth := width - 2 - lipgloss.Width(progress) - 1
		slTitle := sl.Title
		if titleWidth > 0 && lipgloss.Width(slTitle) > titleWidth {
			slTitle = ansi.Truncate(slTitle, titleWidth, "…")
		}
		line := fmt.Sprintf("%s %s %s", icon, slTitle, t.Muted.Render(progress))
		lines = append(lines, line)
	}

	// Active unit.
	if snap.ActiveUnit != "" {
		au := ansi.Truncate("→ "+snap.ActiveUnit, width, "…")
		lines = append(lines, au)
	}

	// Cost.
	costLine := fmt.Sprintf("Cost: $%.2f", snap.TotalCost)
	lines = append(lines, t.Muted.Render(costLine))

	// Elapsed time.
	elapsed := snap.ElapsedSeconds
	var timeLine string
	switch {
	case elapsed < 60:
		timeLine = fmt.Sprintf("Time: %.0fs", elapsed)
	case elapsed < 3600:
		timeLine = fmt.Sprintf("Time: %dm %ds", int(elapsed)/60, int(elapsed)%60)
	default:
		timeLine = fmt.Sprintf("Time: %dh %dm", int(elapsed)/3600, (int(elapsed)%3600)/60)
	}
	lines = append(lines, t.Muted.Render(timeLine))

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

// sidebar renders the chat sidebar containing session title, working
// directory, model info, file list, LSP status, and MCP status.
func (m *UI) drawSidebar(scr uv.Screen, area uv.Rectangle) {
	if m.session == nil {
		return
	}

	const logoHeightBreakpoint = 30

	t := m.com.Styles
	width := area.Dx()
	height := area.Dy()

	title := t.Muted.Width(width).MaxHeight(2).Render(m.session.Title)
	cwd := common.PrettyPath(t, m.com.Store().WorkingDir(), width)
	sidebarLogo := m.sidebarLogo
	if height < logoHeightBreakpoint {
		sidebarLogo = logo.SmallRender(m.com.Styles, width)
	}
	blocks := []string{
		sidebarLogo,
		title,
		"",
		cwd,
		"",
		m.modelInfo(width),
		"",
	}

	sidebarHeader := lipgloss.JoinVertical(
		lipgloss.Left,
		blocks...,
	)

	// Auto-mode progress section, inserted between model info and files.
	autoSection := m.autoModeInfo(width)
	if autoSection != "" {
		sidebarHeader = lipgloss.JoinVertical(
			lipgloss.Left,
			sidebarHeader,
			autoSection,
			"",
		)
	}

	_, remainingHeightArea := layout.SplitVertical(m.layout.sidebar, layout.Fixed(lipgloss.Height(sidebarHeader)))
	remainingHeight := remainingHeightArea.Dy() - 10
	maxFiles, maxLSPs, maxMCPs := getDynamicHeightLimits(remainingHeight)

	lspSection := m.lspInfo(width, maxLSPs, true)
	mcpSection := m.mcpInfo(width, maxMCPs, true)
	filesSection := m.filesInfo(m.com.Store().WorkingDir(), width, maxFiles, true)

	uv.NewStyledString(
		lipgloss.NewStyle().
			MaxWidth(width).
			MaxHeight(height).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					sidebarHeader,
					filesSection,
					"",
					lspSection,
					"",
					mcpSection,
				),
			),
	).Draw(scr, area)
}
