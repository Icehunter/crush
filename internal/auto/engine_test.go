package auto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// --- Mock implementations ---

type mockSessionCreator struct {
	mu       sync.Mutex
	sessions []string
}

func (m *mockSessionCreator) CreateSession(_ context.Context, title string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("session-%d", len(m.sessions))
	m.sessions = append(m.sessions, id)
	return id, nil
}

func (m *mockSessionCreator) CreateChildSession(_ context.Context, id, parentID, title string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = append(m.sessions, id)
	return id, nil
}

type mockDispatcher struct {
	mu        sync.Mutex
	calls     []dispatchCall
	errOnCall int // Return error on the Nth call (1-based), 0 = never.
}

type dispatchCall struct {
	SessionID string
	Prompt    string
	Tier      config.SelectedModelType
}

func (m *mockDispatcher) RunWithForcedTier(_ context.Context, sessionID, prompt string, tier config.SelectedModelType) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, dispatchCall{SessionID: sessionID, Prompt: prompt, Tier: tier})
	if m.errOnCall > 0 && len(m.calls) == m.errOnCall {
		return errors.New("simulated dispatch error")
	}
	return nil
}

type mockAdvancer struct {
	mu       sync.Mutex
	advanced []Unit
	querier  *fixedSequenceQuerier // If set, advances the querier index on each call.
}

func (m *mockAdvancer) AdvanceStatus(_ context.Context, unit Unit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.advanced = append(m.advanced, unit)
	if m.querier != nil {
		m.querier.Advance(unit)
	}
	return nil
}

// sequenceQuerier returns units from a predefined sequence, one per call.
// After the sequence is exhausted it returns done.
type sequenceQuerier struct {
	mu    sync.Mutex
	units []Unit
	index int
}

func (q *sequenceQuerier) ListMilestones(_ context.Context) ([]MilestoneRow, error) {
	panic("sequenceQuerier: use via DeriveState override")
}

func (q *sequenceQuerier) ListSlicesByMilestone(_ context.Context, _ string) ([]SliceRow, error) {
	panic("sequenceQuerier: use via DeriveState override")
}

func (q *sequenceQuerier) ListTasksBySlice(_ context.Context, _ string) ([]TaskRow, error) {
	panic("sequenceQuerier: use via DeriveState override")
}

func (q *sequenceQuerier) next() Unit {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.index >= len(q.units) {
		return Unit{} // Done.
	}
	u := q.units[q.index]
	q.index++
	return u
}

// We test the engine by providing a stateQuerier that feeds a fixed
// sequence. We do this by replacing the querier with one that delegates
// to sequenceQuerier via the standard interface but only uses
// ListMilestones (the DeriveState function walks the full hierarchy,
// which we don't want in engine tests). Instead, we build a thin
// adapter.

// fixedSequenceQuerier implements StateQuerier by returning a scripted
// sequence of units via DeriveState's call path.
type fixedSequenceQuerier struct {
	mu    sync.Mutex
	units []Unit
	index int
}

func (q *fixedSequenceQuerier) ListMilestones(_ context.Context) ([]MilestoneRow, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.index >= len(q.units) {
		// No active milestones → DeriveState returns done.
		return nil, nil
	}
	// Return a single active milestone so DeriveState enters the slice walk.
	return []MilestoneRow{{ID: "M001", Status: string(StatusActive), Phase: string(PhaseExecuting)}}, nil
}

func (q *fixedSequenceQuerier) ListSlicesByMilestone(_ context.Context, _ string) ([]SliceRow, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.index >= len(q.units) {
		return []SliceRow{{ID: "S01", Status: string(StatusCompleted), Phase: string(PhaseCompleted)}}, nil
	}
	u := q.units[q.index]
	// Craft slice state that makes DeriveState return the scripted unit type.
	switch u.Type {
	case UnitExecuteTask:
		return []SliceRow{{ID: u.SliceID, Status: string(StatusActive), Phase: string(PhaseExecuting), SortOrder: 1}}, nil
	case UnitResearch:
		return []SliceRow{{ID: u.SliceID, Status: string(StatusActive), Phase: string(PhaseResearching), SortOrder: 1}}, nil
	case UnitPlanSlice:
		return []SliceRow{{ID: u.SliceID, Status: string(StatusActive), Phase: string(PhasePlanning), SortOrder: 1}}, nil
	case UnitSummarizeSlice:
		return []SliceRow{{ID: u.SliceID, Status: string(StatusActive), Phase: string(PhaseSummarizing), SortOrder: 1}}, nil
	case UnitValidateMilestone:
		// All slices completed → DeriveState returns validate.
		return []SliceRow{{ID: "S01", Status: string(StatusCompleted), Phase: string(PhaseCompleted)}}, nil
	default:
		return nil, nil
	}
}

func (q *fixedSequenceQuerier) ListTasksBySlice(_ context.Context, _ string) ([]TaskRow, error) {
	q.mu.Lock()
	u := q.units[q.index]
	q.mu.Unlock()
	if u.Type == UnitExecuteTask {
		return []TaskRow{{ID: u.TaskID, Status: string(StatusActive), Phase: string(PhaseExecuting), SortOrder: 1}}, nil
	}
	// For summarize: all tasks complete.
	return []TaskRow{{ID: "T01", Status: string(StatusCompleted), Phase: string(PhaseCompleted), SortOrder: 1}}, nil
}

// Advance moves the querier to the next unit. Called by the advancer
// after a unit completes. This is the single point where the index
// is moved forward so ListSlicesByMilestone sees the next state.
func (q *fixedSequenceQuerier) Advance(_ Unit) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.index < len(q.units) {
		q.index++
	}
}

func newTestEngine(querier StateQuerier, dir string) (*Engine, *mockSessionCreator, *mockDispatcher, *mockAdvancer, *pubsub.Broker[AutoEvent]) {
	sessions := &mockSessionCreator{}
	dispatch := &mockDispatcher{}
	advancer := &mockAdvancer{}
	broker := pubsub.NewBroker[AutoEvent]()
	logger := slog.Default()

	// Couple advancer to the querier so it advances the index after
	// each unit completes.
	if fsq, ok := querier.(*fixedSequenceQuerier); ok {
		advancer.querier = fsq
	}

	eng := NewEngine(querier, sessions, dispatch, advancer, nil, nil, 0, nil, nil, broker, dir, logger, nil)
	return eng, sessions, dispatch, advancer, broker
}

// --- Tests ---

func TestEngine_RunAdvancesThroughSequence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "Execute T01"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T02", Title: "Execute T02"},
	}
	querier := &fixedSequenceQuerier{units: units}
	eng, _, dispatch, advancer, _ := newTestEngine(querier, dir)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 2, "should dispatch two units")
	require.Equal(t, config.SelectedModelTypeMain, dispatch.calls[0].Tier)
	dispatch.mu.Unlock()

	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 2)
	advancer.mu.Unlock()
}

func TestEngine_PauseFinishesCurrentUnit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Use a dispatcher that pauses the engine after the first dispatch.
	pauseDispatcher := &pauseAfterFirstDispatcher{
		gate: make(chan struct{}),
	}

	// Many units — we should only complete one before pausing.
	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T02", Title: "T02"},
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T03", Title: "T03"},
	}
	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	advancer := &mockAdvancer{}
	broker := pubsub.NewBroker[AutoEvent]()
	eng := NewEngine(querier, sessions, pauseDispatcher, advancer, nil, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	// The dispatcher will call eng.Pause() after the first dispatch.
	pauseDispatcher.engine = eng

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to events before Run.
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()
	events := broker.Subscribe(subCtx)

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Should have dispatched exactly 1 unit (pause set during first
	// dispatch, checked after it completes).
	pauseDispatcher.mu.Lock()
	require.Equal(t, 1, pauseDispatcher.callCount)
	pauseDispatcher.mu.Unlock()

	status := eng.Status()
	require.Equal(t, EnginePaused, status.State)

	// Verify LoopPaused event was published.
	timeout := time.After(2 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == EventLoopPaused {
				return // Success.
			}
		case <-timeout:
			t.Fatal("timed out waiting for LoopPaused event")
		}
	}
}

func TestEngine_StepExecutesOneUnit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitResearch, MilestoneID: "M001", SliceID: "S01", Title: "Research S01"},
		{Type: UnitPlanSlice, MilestoneID: "M001", SliceID: "S01", Title: "Plan S01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	eng, _, dispatch, _, _ := newTestEngine(querier, dir)

	ctx := context.Background()
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "Step should dispatch exactly one unit")
	require.Equal(t, config.SelectedModelTypePlanning, dispatch.calls[0].Tier, "Research should use planning tier")
	dispatch.mu.Unlock()
}

func TestEngine_LockPreventsConcrrentRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Use a blocking dispatcher so eng1 holds the lock while eng2 tries.
	blocking := &blockingMockDispatcher{called: make(chan struct{})}

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier1 := &fixedSequenceQuerier{units: units}
	querier2 := &fixedSequenceQuerier{units: units}

	sessions := &mockSessionCreator{}
	advancer := &mockAdvancer{}
	broker1 := pubsub.NewBroker[AutoEvent]()
	broker2 := pubsub.NewBroker[AutoEvent]()

	eng1 := NewEngine(querier1, sessions, blocking, advancer, nil, nil, 0, nil, nil, broker1, dir, slog.Default(), nil)
	eng2 := NewEngine(querier2, sessions, &mockDispatcher{}, advancer, nil, nil, 0, nil, nil, broker2, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start eng1 in background — it will block in the dispatcher.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = eng1.Run(ctx, "M001")
	}()

	// Wait until eng1's dispatcher is called (lock is held).
	select {
	case <-blocking.called:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for eng1 dispatch")
	}

	// eng2 should fail to acquire.
	err := eng2.Run(ctx, "M001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrLockHeld)

	cancel()
	wg.Wait()
}

func TestEngine_ResumeFromDBState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Simulate: first two tasks already completed, only T03 remains.
	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T03", Title: "T03"},
	}
	querier := &fixedSequenceQuerier{units: units}
	eng, _, dispatch, advancer, _ := newTestEngine(querier, dir)

	ctx := context.Background()
	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 1, "should only dispatch remaining unit")
	dispatch.mu.Unlock()

	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 1)
	require.Equal(t, "T03", advancer.advanced[0].TaskID)
	advancer.mu.Unlock()
}

func TestEngine_EventPublishing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	eng, _, _, _, broker := newTestEngine(querier, dir)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()
	events := broker.Subscribe(subCtx)

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Collect events.
	var collected []pubsub.EventType
	timeout := time.After(2 * time.Second)
	for {
		select {
		case ev := <-events:
			collected = append(collected, ev.Type)
			if ev.Type == EventUnitCompleted {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	require.Contains(t, collected, EventUnitStarted)
	require.Contains(t, collected, EventUnitCompleted)
}

func TestEngine_TierSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		unitType UnitType
		expected config.SelectedModelType
	}{
		{UnitResearch, config.SelectedModelTypePlanning},
		{UnitPlanSlice, config.SelectedModelTypePlanning},
		{UnitExecuteTask, config.SelectedModelTypeMain},
		{UnitSummarizeSlice, config.SelectedModelTypeBackground},
		{UnitValidateMilestone, config.SelectedModelTypeBackground},
	}
	for _, tt := range tests {
		require.Equal(t, tt.expected, tierForUnit(tt.unitType), "tier for %s", tt.unitType)
	}
}

func TestEngine_StatusReflectsState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	querier := &fixedSequenceQuerier{}
	eng, _, _, _, _ := newTestEngine(querier, dir)

	status := eng.Status()
	require.Equal(t, EngineIdle, status.State)
}

func TestEngine_StopCancelsContext(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a dispatcher that blocks until context is cancelled.
	blockingDispatcher := &blockingMockDispatcher{called: make(chan struct{})}

	units := []Unit{
		{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "T01"},
	}
	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	advancer := &mockAdvancer{}
	broker := pubsub.NewBroker[AutoEvent]()

	eng := NewEngine(querier, sessions, blockingDispatcher, advancer, nil, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- eng.Run(ctx, "M001")
	}()

	// Wait for the dispatcher to be called.
	select {
	case <-blockingDispatcher.called:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for dispatch")
	}

	eng.Stop()

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Run to return after Stop")
	}
}

type blockingMockDispatcher struct {
	called chan struct{}
	once   sync.Once
}

func (b *blockingMockDispatcher) RunWithForcedTier(ctx context.Context, _, _ string, _ config.SelectedModelType) error {
	b.once.Do(func() { close(b.called) })
	<-ctx.Done()
	return ctx.Err()
}

// pauseAfterFirstDispatcher calls engine.Pause() after the first dispatch
// completes, simulating an external pause signal mid-loop.
type pauseAfterFirstDispatcher struct {
	mu        sync.Mutex
	callCount int
	engine    *Engine
	gate      chan struct{}
}

func (p *pauseAfterFirstDispatcher) RunWithForcedTier(_ context.Context, _, _ string, _ config.SelectedModelType) error {
	p.mu.Lock()
	p.callCount++
	count := p.callCount
	p.mu.Unlock()

	if count == 1 {
		// Signal pause after the first dispatch succeeds.
		p.engine.Pause()
	}
	return nil
}
