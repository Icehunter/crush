package model

import (
	"context"
	"errors"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/auto"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/util"
	"github.com/stretchr/testify/require"
)

// mockAutoController implements AutoController for testing.
type mockAutoController struct {
	startCalled      bool
	stopCalled       bool
	pauseCalled      bool
	resumeCalled     bool
	stepCalled       bool
	undoCalled       bool
	skipCalled       bool
	dispatchCalled   bool
	steerCalled      bool
	historyCalled    bool
	rateCalled       bool
	doctorCalled     bool
	quickCalled      bool
	parkCalled       bool
	unparkCalled     bool
	rethinkCalled    bool
	cleanupCalled    bool
	initCalled       bool
	statusVal        string
	startErr         error
	stopErr          error
	pauseErr         error
	resumeErr        error
	stepErr          error
	undoResult       string
	undoErr          error
	skipErr          error
	dispatchErr      error
	steerErr         error
	historyResult    string
	historyErr       error
	rateErr          error
	doctorResult     string
	doctorErr        error
	quickErr         error
	queueResult      []string
	queueErr         error
	templateResult   string
	templateErr      error
	parkErr          error
	unparkErr        error
	rethinkErr       error
	prefsResult      string
	prefsErr         error
	setPrefErr       error
	cleanupResult    string
	cleanupErr       error
	initErr          error
}

func (m *mockAutoController) StartAuto(_ context.Context, _ string) error {
	m.startCalled = true
	return m.startErr
}

func (m *mockAutoController) StopAuto() error {
	m.stopCalled = true
	return m.stopErr
}

func (m *mockAutoController) PauseAuto() error {
	m.pauseCalled = true
	return m.pauseErr
}

func (m *mockAutoController) ResumeAuto(_ context.Context) error {
	m.resumeCalled = true
	return m.resumeErr
}

func (m *mockAutoController) StepAuto(_ context.Context, _ string) error {
	m.stepCalled = true
	return m.stepErr
}

func (m *mockAutoController) AutoStatus() string {
	return m.statusVal
}

func (m *mockAutoController) AutoQueue(_ context.Context, _ string) ([]string, error) {
	return m.queueResult, m.queueErr
}

func (m *mockAutoController) UndoLast(_ context.Context, _ string) (string, error) {
	m.undoCalled = true
	return m.undoResult, m.undoErr
}

func (m *mockAutoController) SkipUnit(_ context.Context, _ string) error {
	m.skipCalled = true
	return m.skipErr
}

func (m *mockAutoController) DispatchPhase(_ context.Context, _, _ string) error {
	m.dispatchCalled = true
	return m.dispatchErr
}

func (m *mockAutoController) Steer(_ context.Context, _ string) error {
	m.steerCalled = true
	return m.steerErr
}

func (m *mockAutoController) History(_ context.Context, _ int) (string, error) {
	m.historyCalled = true
	return m.historyResult, m.historyErr
}

func (m *mockAutoController) RateTier(_ context.Context, _ string) error {
	m.rateCalled = true
	return m.rateErr
}

func (m *mockAutoController) RunDoctor(_ context.Context, _ bool) (string, error) {
	m.doctorCalled = true
	return m.doctorResult, m.doctorErr
}

func (m *mockAutoController) QuickTask(_ context.Context, _, _ string) error {
	m.quickCalled = true
	return m.quickErr
}

func (m *mockAutoController) StartFromTemplate(_ context.Context, _ string) (string, error) {
	return m.templateResult, m.templateErr
}

func (m *mockAutoController) ParkMilestone(_ context.Context, _ string) error {
	m.parkCalled = true
	return m.parkErr
}

func (m *mockAutoController) UnparkMilestone(_ context.Context, _ string) error {
	m.unparkCalled = true
	return m.unparkErr
}

func (m *mockAutoController) Rethink(_ context.Context, _ string) error {
	m.rethinkCalled = true
	return m.rethinkErr
}

func (m *mockAutoController) GetPreferences() (string, error) {
	return m.prefsResult, m.prefsErr
}

func (m *mockAutoController) SetPreference(_, _ string) error {
	return m.setPrefErr
}

func (m *mockAutoController) CleanupWorktrees(_ context.Context) (string, error) {
	m.cleanupCalled = true
	return m.cleanupResult, m.cleanupErr
}

func (m *mockAutoController) InitProject(_ context.Context, _ string) error {
	m.initCalled = true
	return m.initErr
}

func newTestUIForToggle() *UI {
	com := common.DefaultCommon(nil)
	return &UI{com: com}
}

func TestAutoToggle_KeyMatches(t *testing.T) {
	t.Parallel()
	km := DefaultKeyMap()
	msg := tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}
	require.True(t, key.Matches(msg, km.Auto.Toggle),
		"ctrl+a should match Auto.Toggle binding")
}

func TestAutoToggle_NilController(t *testing.T) {
	t.Parallel()
	m := newTestUIForToggle()
	m.autoController = nil

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd, "should return a command even when controller is nil")

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "not available")
}

func TestAutoToggle_NoSession(t *testing.T) {
	t.Parallel()
	m := newTestUIForToggle()
	m.autoController = &mockAutoController{}
	m.session = nil

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No active session")
}

func TestAutoToggle_IdleToStart(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = nil
	m.autoMilestoneID = "M001"

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "started")
	require.True(t, mock.startCalled, "StartAuto should have been called")
}

func TestAutoToggle_IdleNoMilestone(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = nil
	m.autoMilestoneID = ""

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "No milestone configured")
	require.False(t, mock.startCalled, "StartAuto should not have been called")
}

func TestAutoToggle_RunningToPause(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = &auto.AutoSnapshot{Status: "running"}

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "paused")
	require.True(t, mock.pauseCalled, "PauseAuto should have been called")
}

func TestAutoToggle_PausedToResume(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = &auto.AutoSnapshot{Status: "paused"}

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeInfo, info.Type)
	require.Contains(t, info.Msg, "resumed")
	require.True(t, mock.resumeCalled, "ResumeAuto should have been called")
}

func TestAutoToggle_StartError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{
		startErr: errors.New("engine exploded"),
	}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = nil
	m.autoMilestoneID = "M001"

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "engine exploded")
	require.True(t, mock.startCalled)
}

func TestAutoToggle_PauseError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{
		pauseErr: errors.New("pause failed"),
	}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = &auto.AutoSnapshot{Status: "running"}

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "pause failed")
}

func TestAutoToggle_ResumeError(t *testing.T) {
	t.Parallel()
	mock := &mockAutoController{
		resumeErr: errors.New("resume failed"),
	}
	m := newTestUIForToggle()
	m.autoController = mock
	m.session = &session.Session{}
	m.autoSnapshot = &auto.AutoSnapshot{Status: "paused"}

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeError, info.Type)
	require.Contains(t, info.Msg, "resume failed")
}

func TestAutoToggle_UnknownStatus(t *testing.T) {
	t.Parallel()
	m := newTestUIForToggle()
	m.autoController = &mockAutoController{}
	m.session = &session.Session{}
	m.autoSnapshot = &auto.AutoSnapshot{Status: "exploding"}

	cmd := m.toggleAutoMode()
	require.NotNil(t, cmd)

	msg := cmd()
	info, ok := msg.(util.InfoMsg)
	require.True(t, ok, "expected InfoMsg, got %T", msg)
	require.Equal(t, util.InfoTypeWarn, info.Type)
	require.Contains(t, info.Msg, "exploding")
}
