package auto

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// mockBudgetChecker implements BudgetChecker for testing.
type mockBudgetChecker struct {
	cost float64
	err  error
}

func (m *mockBudgetChecker) CheckBudget(_ context.Context, _ string) (float64, error) {
	return m.cost, m.err
}

// TestIntegration_BudgetExceededPausesEngine verifies that when the budget
// checker reports a cost >= ceiling the engine pauses, publishes
// EventBudgetExceeded, and does NOT dispatch.
func TestIntegration_BudgetExceededPausesEngine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	checker := &mockBudgetChecker{cost: 0.75}
	ceiling := 0.50

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, checker, ceiling, nil, nil, broker, dir, slog.Default(), nil)

	// Subscribe to capture events.
	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Engine should be paused.
	status := eng.Status()
	require.Equal(t, EnginePaused, status.State, "engine should be paused after budget exceeded")

	// Dispatcher should NOT have been called.
	dispatch.mu.Lock()
	require.Empty(t, dispatch.calls, "dispatcher should not be called when budget exceeded")
	dispatch.mu.Unlock()

	// Should have received EventBudgetExceeded.
	var gotBudgetExceeded bool
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == EventBudgetExceeded {
				gotBudgetExceeded = true
				require.Contains(t, ev.Payload.Message, "total cost")
				require.Contains(t, ev.Payload.Message, "ceiling")
			}
			if ev.Type == EventLoopPaused || gotBudgetExceeded {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	require.True(t, gotBudgetExceeded, "should have received EventBudgetExceeded")
}

// TestIntegration_BudgetUnderCeilingDispatches verifies that when the budget
// checker reports a cost below the ceiling, dispatch proceeds normally.
func TestIntegration_BudgetUnderCeilingDispatches(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	checker := &mockBudgetChecker{cost: 0.10}
	ceiling := 0.50

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, checker, ceiling, nil, nil, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Engine should be idle (completed all work).
	status := eng.Status()
	require.Equal(t, EngineIdle, status.State, "engine should be idle after completing work")

	// Dispatcher should have been called.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "dispatcher should be called once")
	dispatch.mu.Unlock()
}

// TestIntegration_BudgetZeroCeilingSkipsCheck verifies that a zero ceiling
// disables budget enforcement entirely and dispatch proceeds.
func TestIntegration_BudgetZeroCeilingSkipsCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Checker with high cost — but ceiling is 0, so check should be skipped.
	checker := &mockBudgetChecker{cost: 999.0}
	ceiling := 0.0

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, checker, ceiling, nil, nil, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	status := eng.Status()
	require.Equal(t, EngineIdle, status.State, "engine should complete normally with zero ceiling")

	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "dispatcher should be called when ceiling is zero")
	dispatch.mu.Unlock()
}

// TestIntegration_BudgetNilCheckerSkipsCheck verifies that a nil budget
// checker disables budget enforcement and dispatch proceeds normally.
func TestIntegration_BudgetNilCheckerSkipsCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Ceiling is set but checker is nil — should skip check.
	eng := NewEngine(querier, sessions, dispatch, advancer, nil, nil, 0.50, nil, nil, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	status := eng.Status()
	require.Equal(t, EngineIdle, status.State, "engine should complete normally with nil checker")

	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "dispatcher should be called when checker is nil")
	dispatch.mu.Unlock()
}
