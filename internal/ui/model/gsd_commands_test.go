package model

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "/gsd help")
	require.Contains(t, notice.Text, "/gsd auto")
	require.Contains(t, notice.Text, "/gsd pause")
	require.Contains(t, notice.Text, "/gsd stop")
	require.Contains(t, notice.Text, "/gsd status")
}

func TestGSD_BareSlashGSD(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	// Bare /gsd routes to gsdNext which needs a controller and milestone.
	// With nil controller, it should warn.
	cmd, ok := m.handleGSDCommand("/gsd")
	require.True(t, ok, "bare /gsd should be handled")
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_BareSlashGSD_WithMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd")
	require.True(t, ok, "bare /gsd should be handled")
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Executing next unit")
	require.True(t, mock.stepCalled)
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
	require.True(t, mock.stopCalled)
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
		Slices: []auto.SliceProgress{
			{ID: "S01", Title: "Auth", Status: "active", TasksDone: 1, TasksTotal: 3},
		},
	}

	cmd, ok := m.handleGSDCommand("/gsd status")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "running")
	require.Contains(t, notice.Text, "M003")
	require.Contains(t, notice.Text, "S01/T02")
	require.Contains(t, notice.Text, "1.23")
	require.Contains(t, notice.Text, "Auth")
	require.Contains(t, notice.Text, "1/3")
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
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "idle")
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

func TestGSD_Next(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd next M001")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Executing next unit for M001")
	require.True(t, mock.stepCalled)
}

func TestGSD_NextFallbackMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M002"

	cmd, ok := m.handleGSDCommand("/gsd next")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Executing next unit for M002")
	require.True(t, mock.stepCalled)
}

func TestGSD_NextNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd next")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No milestone ID provided")
}

func TestGSD_Queue(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{
		statusVal:   "running",
		queueResult: []string{"research S01 — Auth", "plan S01", "execute tasks for S01"},
	}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd queue")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "Dispatch queue")
	require.Contains(t, notice.Text, "research S01")
	require.Contains(t, notice.Text, "3 units")
}

func TestGSD_QueueEmpty(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{
		statusVal:   "idle",
		queueResult: []string{},
	}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd queue")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "all work complete")
}

func TestGSD_QueueNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	// No autoMilestoneID set.

	cmd, ok := m.handleGSDCommand("/gsd queue")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No active milestone")
}

func TestGSD_Undo(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{
		statusVal:  "idle",
		undoResult: "Undid T01: Login",
	}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd undo")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Undid T01")
	require.True(t, mock.undoCalled)
}

func TestGSD_UndoNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd undo")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No active milestone")
}

func TestGSD_UndoNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd undo")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_Skip(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd skip T01")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Skipped task T01")
	require.True(t, mock.skipCalled)
}

func TestGSD_SkipNoTaskID(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd skip")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_Dispatch(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd dispatch research")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Dispatching research")
	require.True(t, mock.dispatchCalled)
}

func TestGSD_DispatchNoPhase(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd dispatch")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_Steer(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "running"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd steer focus on error handling")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "focus on error handling")
	require.True(t, mock.steerCalled)
}

func TestGSD_SteerNoText(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "running"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd steer")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_HelpShowsWave2Commands(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd help")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "/gsd undo")
	require.Contains(t, notice.Text, "/gsd skip")
	require.Contains(t, notice.Text, "/gsd dispatch")
	require.Contains(t, notice.Text, "/gsd steer")
}

// --- Wave 3 tests ---

func TestGSD_History(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", historyResult: "Last 5 units:\n  ✓ [execute_task] Fix bug (200ms, $0.0100)"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd history 5")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "Last 5 units")
	require.True(t, mock.historyCalled)
}

func TestGSD_HistoryDefault(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", historyResult: "No execution history found"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd history")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	_, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.True(t, mock.historyCalled)
}

func TestGSD_HistoryNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd history")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_HistoryError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", historyErr: errors.New("journal corrupt")}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd history")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "journal corrupt")
}

func TestGSD_RateOk(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd rate ok")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "ok")
	require.True(t, mock.rateCalled)
}

func TestGSD_RateOver(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd rate over")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "over")
	require.True(t, mock.rateCalled)
}

func TestGSD_RateUnder(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd rate under")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "under")
	require.True(t, mock.rateCalled)
}

func TestGSD_RateInvalid(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd rate bad")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
	require.False(t, mock.rateCalled)
}

func TestGSD_RateNoArg(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd rate")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_RateError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", rateErr: errors.New("no recent entries")}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd rate ok")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "no recent entries")
}

func TestGSD_Doctor(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", doctorResult: "All checks passed"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd doctor")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "All checks passed")
	require.True(t, mock.doctorCalled)
}

func TestGSD_DoctorFix(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", doctorResult: "Fixed 2 issues"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd doctor fix")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "Fixed 2 issues")
	require.True(t, mock.doctorCalled)
}

func TestGSD_DoctorNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd doctor")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_DoctorError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", doctorErr: errors.New("DB unreachable")}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd doctor")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "DB unreachable")
}

func TestGSD_Quick(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd quick fix the login bug")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "fix the login bug")
	require.True(t, mock.quickCalled)
}

func TestGSD_QuickNoDescription(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd quick")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_QuickNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = ""

	cmd, ok := m.handleGSDCommand("/gsd quick do something")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No active milestone")
}

func TestGSD_QuickNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd quick do something")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_QuickError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", quickErr: errors.New("dispatch failed")}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd quick fix it")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "dispatch failed")
}

func TestGSD_HelpShowsWave3Commands(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd help")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "/gsd history")
	require.Contains(t, notice.Text, "/gsd rate")
	require.Contains(t, notice.Text, "/gsd doctor")
	require.Contains(t, notice.Text, "/gsd quick")
}

// --- Wave 4 tests ---

func TestGSD_Start(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", templateResult: "Bug Fix"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd start bugfix")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Bug Fix")
}

func TestGSD_StartNoTemplate(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd start")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_StartUnknownTemplate(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", templateErr: errors.New("unknown template")}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd start unknown")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "unknown template")
}

func TestGSD_StartNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd start bugfix")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_Park(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd park M001")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "parked")
	require.True(t, mock.parkCalled)
}

func TestGSD_ParkFallbackMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd park")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "M001")
}

func TestGSD_ParkNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = ""

	cmd, ok := m.handleGSDCommand("/gsd park")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No milestone")
}

func TestGSD_ParkError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", parkErr: errors.New("already parked")}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd park M001")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "already parked")
}

func TestGSD_Unpark(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd unpark M001")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "unparked")
	require.True(t, mock.unparkCalled)
}

func TestGSD_UnparkNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = ""

	cmd, ok := m.handleGSDCommand("/gsd unpark")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No milestone")
}

func TestGSD_Rethink(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = "M001"

	cmd, ok := m.handleGSDCommand("/gsd rethink")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "Rethink")
	require.True(t, mock.rethinkCalled)
}

func TestGSD_RethinkNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock
	m.autoMilestoneID = ""

	cmd, ok := m.handleGSDCommand("/gsd rethink")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No active milestone")
}

func TestGSD_RethinkNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd rethink")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_PrefsView(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", prefsResult: "Preferences:\n  Global: /path"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd prefs")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "Preferences")
}

func TestGSD_PrefsSet(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd prefs auto_push=true")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "auto_push=true")
}

func TestGSD_PrefsInvalidFormat(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd prefs badformat")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_Cleanup(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", cleanupResult: "Stale worktrees pruned"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd cleanup")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "pruned")
	require.True(t, mock.cleanupCalled)
}

func TestGSD_CleanupNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd cleanup")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_CleanupError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", cleanupErr: errors.New("git not found")}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd cleanup")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "git not found")
}

func TestGSD_Init(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd init build a task management system")
	require.True(t, ok)
	require.NotNil(t, cmd)

	// tea.Batch returns a BatchMsg containing multiple commands.
	msg := cmd()
	batch, isBatch := msg.(tea.BatchMsg)
	require.True(t, isBatch, "expected tea.BatchMsg, got %T", msg)

	// Find the async result command and run it.
	var foundNotice bool
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		result := batchCmd()
		if notice, ok := result.(util.SystemNoticeMsg); ok {
			require.Contains(t, notice.Text, "initialized")
			foundNotice = true
		}
	}
	require.True(t, foundNotice, "expected SystemNoticeMsg in batch")
	require.True(t, mock.initCalled)
}

func TestGSD_InitNoVision(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle"}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd init")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "Usage")
}

func TestGSD_InitNilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()
	m.autoController = nil

	cmd, ok := m.handleGSDCommand("/gsd init something")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	info, isInfo := msg.(util.InfoMsg)
	require.True(t, isInfo, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestGSD_InitError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{statusVal: "idle", initErr: errors.New("init not configured")}
	m := newTestUIForGSD()
	m.autoController = mock

	cmd, ok := m.handleGSDCommand("/gsd init something")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	batch, isBatch := msg.(tea.BatchMsg)
	require.True(t, isBatch, "expected tea.BatchMsg, got %T", msg)

	var foundNotice bool
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		result := batchCmd()
		if notice, ok := result.(util.SystemNoticeMsg); ok {
			require.Contains(t, notice.Text, "init not configured")
			foundNotice = true
		}
	}
	require.True(t, foundNotice, "expected error SystemNoticeMsg in batch")
}

func TestGSD_HelpShowsInitCommand(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd help")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "/gsd init")
}

func TestGSD_HelpShowsWave4Commands(t *testing.T) {
	t.Parallel()
	m := newTestUIForGSD()

	cmd, ok := m.handleGSDCommand("/gsd help")
	require.True(t, ok)
	require.NotNil(t, cmd)

	msg := cmd()
	notice, isNotice := msg.(util.SystemNoticeMsg)
	require.True(t, isNotice, "expected SystemNoticeMsg, got %T", msg)
	require.Contains(t, notice.Text, "/gsd start")
	require.Contains(t, notice.Text, "/gsd park")
	require.Contains(t, notice.Text, "/gsd unpark")
	require.Contains(t, notice.Text, "/gsd rethink")
	require.Contains(t, notice.Text, "/gsd prefs")
	require.Contains(t, notice.Text, "/gsd cleanup")
}
