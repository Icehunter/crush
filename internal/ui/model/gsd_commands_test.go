package model

import (
	"testing"

	"github.com/charmbracelet/crush/internal/auto"
	"github.com/charmbracelet/crush/internal/ui/util"
	"github.com/stretchr/testify/require"
)

func newTestUIForGSD() *UI {
	return newTestUIForToggle()
}

func TestGSD_Help(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd help")
	require.True(t, ok, "/gsd help should be handled")
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "/gsd help")
	require.Contains(t, info.Msg, "/gsd auto")
	require.Contains(t, info.Msg, "/gsd pause")
	require.Contains(t, info.Msg, "/gsd stop")
	require.Contains(t, info.Msg, "/gsd status")
}

func TestGSD_BareSlashGSD(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd")
	require.True(t, ok, "bare /gsd should be handled")
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "/gsd help")
}

func TestGSD_UnknownSubcommand(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd foobar")
	require.True(t, ok, "/gsd foobar should be handled")
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Unknown /gsd command: foobar")
	require.Contains(t, info.Msg, "/gsd help")
}

func TestGSD_MainNotIntercepted(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	_, ok := m.handleGSDCommand("/main hello")
	require.False(t, ok, "/main should NOT be intercepted as /gsd")
}

func TestGSD_SlashGSDPrefixNotIntercepted(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	// "/gsdx" should not match — only "/gsd" or "/gsd " prefix.
	_, ok := m.handleGSDCommand("/gsdx something")
	require.False(t, ok, "/gsdx should NOT be intercepted as /gsd")
}

func TestGSD_AutoStart(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd auto M001")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Auto-mode started for M001")
	require.True(t, mock.startCalled)
}

func TestGSD_AutoFallbackMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M002"

	cmd, ok := m.handleGSDCommand("/gsd auto")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Auto-mode started for M002")
	require.True(t, mock.startCalled)
}

func TestGSD_AutoNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	// No autoMilestoneID set.

	cmd, ok := m.handleGSDCommand("/gsd auto")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No milestone ID provided")
}

func TestGSD_AutoNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	// autoController is nil.

	cmd, ok := m.handleGSDCommand("/gsd auto M001")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_Pause(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "running"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd pause")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "paused")
	require.True(t, mock.pauseCalled)
}

func TestGSD_PauseNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd pause")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_Stop(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "running"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd stop")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "stopped")
	require.True(t, mock.pauseCalled) // stop maps to PauseAuto.
}

func TestGSD_StopNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd stop")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_Status(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "running"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M003"
	m.autoSnapshot = &auto.AutoSnapshot{
		ActiveUnit: "S01/T02",
		TotalCost:  1.23,
	}

	cmd, ok := m.handleGSDCommand("/gsd status")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "running")
	require.Contains(t, info.Msg, "M003")
	require.Contains(t, info.Msg, "S01/T02")
	require.Contains(t, info.Msg, "$1.23")
}

func TestGSD_StatusMinimal(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd status")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "idle")
}

func TestGSD_StatusNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd status")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}
