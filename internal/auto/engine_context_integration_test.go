package auto

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// TestIntegration_ContextPressurePauses verifies that when token usage
// exceeds the threshold the engine pauses and publishes
// EventContextPressure.
func TestIntegration_ContextPressurePauses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Token usage is 9000/10000 = 0.9, above 0.8 threshold.
	tokenQ := &mockTokenQuerier{prompt: 6000, completion: 3000}
	cm := NewContextMonitor(0.8, 10000, tokenQ)
	require.NotNil(t, cm)

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, nil, 0, nil, cm, broker, dir, slog.Default(), nil)

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err, "engine should return nil on pause")

	// Engine should be paused.
	status := eng.Status()
	require.Equal(t, EnginePaused, status.State, "engine should pause on context pressure")

	// Should have received EventContextPressure.
	var gotPressure bool
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == EventContextPressure {
				gotPressure = true
				goto done
			}
			if ev.Type == EventLoopPaused {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	require.True(t, gotPressure, "should receive EventContextPressure")
}

// TestIntegration_ContextPressureBelowThreshold verifies that when token
// usage is below the threshold the engine completes normally.
func TestIntegration_ContextPressureBelowThreshold(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Token usage is 2000/10000 = 0.2, well below 0.8 threshold.
	tokenQ := &mockTokenQuerier{prompt: 1000, completion: 1000}
	cm := NewContextMonitor(0.8, 10000, tokenQ)
	require.NotNil(t, cm)

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, nil, 0, nil, cm, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Engine completed normally — not paused.
	status := eng.Status()
	require.Equal(t, EngineIdle, status.State, "engine should complete normally below threshold")

	// Advancer should have been called.
	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 1, "should advance normally")
	advancer.mu.Unlock()
}

// TestIntegration_ContextPressureNilMonitorSkips verifies that a nil
// context monitor does not interfere with normal engine operation.
func TestIntegration_ContextPressureNilMonitorSkips(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// nil context monitor — should be a no-op.
	eng := NewEngine(querier, sessions, dispatch, advancer, nil, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	status := eng.Status()
	require.Equal(t, EngineIdle, status.State, "engine should complete normally with nil monitor")

	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 1, "should advance normally")
	advancer.mu.Unlock()
}
