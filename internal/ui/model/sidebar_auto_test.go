package model

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/auto"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func newTestUIForAuto() *UI {
	com := common.DefaultCommon(nil)
	return &UI{com: com}
}

func TestAutoModeInfo_Nil(t *testing.T) {
	t.Parallel()
	m := newTestUIForAuto()
	got := m.autoModeInfo(30)
	require.Empty(t, got)
}

func TestAutoModeInfo_Running(t *testing.T) {
	t.Parallel()
	m := newTestUIForAuto()
	m.autoSnapshot = &auto.AutoSnapshot{
		MilestoneID:    "M001",
		MilestoneTitle: "First Milestone",
		Slices: []auto.SliceProgress{
			{ID: "S01", Title: "Slice One", Status: "completed", TasksDone: 3, TasksTotal: 3},
			{ID: "S02", Title: "Slice Two", Status: "active", TasksDone: 1, TasksTotal: 4},
			{ID: "S03", Title: "Slice Three", Status: "pending", TasksDone: 0, TasksTotal: 2},
		},
		ActiveUnit:     "M001/S02/T02",
		TotalCost:      1.23,
		ElapsedSeconds: 125,
		Status:         "running",
	}

	got := m.autoModeInfo(30)
	plain := ansi.Strip(got)

	require.Contains(t, plain, "Auto Mode")
	require.Contains(t, plain, "Running")
	require.Contains(t, plain, "▶")
	require.Contains(t, plain, "First Milestone")
	require.Contains(t, plain, "✓")              // Completed slice icon.
	require.Contains(t, plain, "3/3")            // Completed slice progress.
	require.Contains(t, plain, "1/4")            // Active slice progress.
	require.Contains(t, plain, "→ M001/S02/T02") // Active unit.
	require.Contains(t, plain, "Cost: $1.23")
	require.Contains(t, plain, "Time: 2m 5s")
}

func TestAutoModeInfo_Paused(t *testing.T) {
	t.Parallel()
	m := newTestUIForAuto()
	m.autoSnapshot = &auto.AutoSnapshot{
		Status:         "paused",
		MilestoneTitle: "Paused Mile",
		ElapsedSeconds: 60,
	}

	got := m.autoModeInfo(30)
	plain := ansi.Strip(got)

	require.Contains(t, plain, "⏸")
	require.Contains(t, plain, "Paused")
	require.Contains(t, plain, "Cost: $0.00")
	require.Contains(t, plain, "Time: 1m 0s")
}

func TestAutoModeInfo_Truncation(t *testing.T) {
	t.Parallel()
	m := newTestUIForAuto()
	m.autoSnapshot = &auto.AutoSnapshot{
		Status:         "running",
		MilestoneTitle: "This is an extremely long milestone title that should be truncated to fit",
		Slices: []auto.SliceProgress{
			{ID: "S01", Title: "A very long slice title that definitely exceeds thirty characters", Status: "active", TasksDone: 1, TasksTotal: 10},
		},
		ActiveUnit:     "M999/S99/T99/some-extra-long-unit-id",
		TotalCost:      0,
		ElapsedSeconds: 7200,
	}

	got := m.autoModeInfo(30)
	// Every line must fit within 30 visible characters.
	for i, line := range strings.Split(got, "\n") {
		w := lipgloss.Width(line)
		require.LessOrEqual(t, w, 30, "line %d exceeds 30 chars (got %d): %q", i, w, line)
	}
}

func TestAutoModeInfo_EmptySlices(t *testing.T) {
	t.Parallel()
	m := newTestUIForAuto()
	m.autoSnapshot = &auto.AutoSnapshot{
		Status:         "completed",
		MilestoneTitle: "",
		Slices:         nil,
		ActiveUnit:     "",
		TotalCost:      0,
		ElapsedSeconds: 0,
	}

	got := m.autoModeInfo(30)
	plain := ansi.Strip(got)

	require.Contains(t, plain, "Auto Mode")
	require.Contains(t, plain, "Done")
	require.Contains(t, plain, "Cost: $0.00")
	require.Contains(t, plain, "Time: 0s")
}
