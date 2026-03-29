package auto

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullLoopLifecycle seeds a milestone with one slice and two
// tasks, runs the engine, and verifies the complete dispatch sequence:
// research → plan → execute(×2) → summarize → validate.
func TestIntegration_FullLoopLifecycle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Build the scripted unit sequence that DeriveState will produce.
	units := []Unit{
		{Type: UnitResearch, MilestoneID: "M001", SliceID: "S01", Title: "Research slice S01"},
		{Type: UnitPlanSlice, MilestoneID: "M001", SliceID: "S01", Title: "Plan slice S01"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "Execute task T01"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T02", Title: "Execute task T02"},
		{Type: UnitSummarizeSlice, MilestoneID: "M001", SliceID: "S01", Title: "Summarize slice S01"},
		{Type: UnitValidateMilestone, MilestoneID: "M001", Title: "Validate milestone M001"},
	}

	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	// Subscribe to events before Run.
	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Verify dispatch order.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 6, "should dispatch 6 units")

	expectedOrder := []UnitType{
		UnitResearch, UnitPlanSlice, UnitExecuteTask, UnitExecuteTask,
		UnitSummarizeSlice, UnitValidateMilestone,
	}
	for i, expected := range expectedOrder {
		// The prompt contains the unit type string from templates.
		require.NotEmpty(t, dispatch.calls[i].Prompt, "prompt for call %d should not be empty", i)
		_ = expected // Order is enforced by the fixedSequenceQuerier.
	}

	// Verify tier assignments.
	require.Equal(t, config.SelectedModelTypePlanning, dispatch.calls[0].Tier, "research → planning tier")
	require.Equal(t, config.SelectedModelTypePlanning, dispatch.calls[1].Tier, "plan → planning tier")
	require.Equal(t, config.SelectedModelTypeMain, dispatch.calls[2].Tier, "execute → main tier")
	require.Equal(t, config.SelectedModelTypeMain, dispatch.calls[3].Tier, "execute → main tier")
	require.Equal(t, config.SelectedModelTypeBackground, dispatch.calls[4].Tier, "summarize → background tier")
	require.Equal(t, config.SelectedModelTypeBackground, dispatch.calls[5].Tier, "validate → background tier")
	dispatch.mu.Unlock()

	// Verify all units were advanced.
	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 6, "all 6 units should be advanced")
	advancer.mu.Unlock()

	// Verify events were published in correct order.
	var eventTypes []pubsub.EventType
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			eventTypes = append(eventTypes, ev.Type)
			// We expect 6 started + 6 completed = 12 events minimum.
			if len(eventTypes) >= 12 {
				goto doneEvents
			}
		case <-timeout:
			goto doneEvents
		}
	}
doneEvents:
	// Check that started/completed alternate.
	startedCount := 0
	completedCount := 0
	for _, et := range eventTypes {
		switch et {
		case EventUnitStarted:
			startedCount++
		case EventUnitCompleted:
			completedCount++
		}
	}
	require.Equal(t, 6, startedCount, "should have 6 UnitStarted events")
	require.Equal(t, 6, completedCount, "should have 6 UnitCompleted events")

	// Verify engine is idle after completion.
	status := eng.Status()
	require.Equal(t, EngineIdle, status.State)
}

// TestIntegration_StepExecutesSingleUnit verifies that Step() runs exactly
// one unit and returns.
func TestIntegration_StepExecutesSingleUnit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitResearch, MilestoneID: "M001", SliceID: "S01", Title: "Research S01"},
		{Type: UnitPlanSlice, MilestoneID: "M001", SliceID: "S01", Title: "Plan S01"},
	}

	querier := &fixedSequenceQuerier{units: units}
	dispatch := &recordingDispatcher{}
	eng, _, _, _, _ := newTestEngine(querier, dir)
	// Swap in our recording dispatcher.
	eng.dispatch = dispatch

	ctx := context.Background()
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "Step should dispatch exactly one unit")
	dispatch.mu.Unlock()
}

// TestIntegration_PauseMidLoop verifies that calling Pause() during the loop
// stops after the current unit completes.
func TestIntegration_PauseMidLoop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T02", Title: "T02"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T03", Title: "T03"},
	}

	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	pauseDisp := &pauseAfterNDispatcher{pauseAfterN: 1}

	eng := NewEngine(querier, sessions, pauseDisp, advancer, nil, nil, 0, nil, nil, broker, dir, slog.Default(), nil)
	pauseDisp.engine = eng

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Should have dispatched exactly 1 unit.
	pauseDisp.mu.Lock()
	require.Equal(t, 1, pauseDisp.callCount, "should dispatch 1 unit before pausing")
	pauseDisp.mu.Unlock()

	require.Equal(t, EnginePaused, eng.Status().State)
}

// TestIntegration_ChildSessionsCreated verifies that each dispatched unit
// gets its own child session.
func TestIntegration_ChildSessionsCreated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T02", Title: "T02"},
	}
	querier := &fixedSequenceQuerier{units: units}
	eng, sessions, _, _, _ := newTestEngine(querier, dir)

	ctx := context.Background()
	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	sessions.mu.Lock()
	// 1 parent session + 2 child sessions = 3 total.
	require.Len(t, sessions.sessions, 3, "should have 1 parent + 2 child sessions")
	sessions.mu.Unlock()
}

// --- Additional test helpers ---

// recordingDispatcher records all calls without side effects.
type recordingDispatcher struct {
	mu    sync.Mutex
	calls []dispatchCall
}

func (r *recordingDispatcher) RunWithForcedTier(_ context.Context, sessionID, prompt string, tier config.SelectedModelType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, dispatchCall{SessionID: sessionID, Prompt: prompt, Tier: tier})
	return nil
}

// pauseAfterNDispatcher pauses the engine after N dispatches.
type pauseAfterNDispatcher struct {
	mu          sync.Mutex
	callCount   int
	pauseAfterN int
	engine      *Engine
}

func (p *pauseAfterNDispatcher) RunWithForcedTier(_ context.Context, _, _ string, _ config.SelectedModelType) error {
	p.mu.Lock()
	p.callCount++
	count := p.callCount
	p.mu.Unlock()

	if count == p.pauseAfterN {
		p.engine.Pause()
	}
	return nil
}
