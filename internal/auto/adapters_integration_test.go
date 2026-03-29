package auto

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Integration-test mocks: thin stubs for SessionCreator and Dispatcher only.
// Everything else (StateQuerier, StatusAdvancer) is real DB-backed.
// ---------------------------------------------------------------------------

// integrationSessionCreator returns predictable session IDs.
type integrationSessionCreator struct {
	mu       sync.Mutex
	sessions []string
}

func (s *integrationSessionCreator) CreateSession(_ context.Context, title string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := "parent-session"
	s.sessions = append(s.sessions, id)
	return id, nil
}

func (s *integrationSessionCreator) CreateChildSession(_ context.Context, id, _, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = append(s.sessions, id)
	return id, nil
}

// integrationDispatcher records all dispatched prompts and returns nil.
type integrationDispatcher struct {
	mu    sync.Mutex
	calls []integrationDispatchCall
}

type integrationDispatchCall struct {
	SessionID string
	Prompt    string
	Tier      config.SelectedModelType
}

func (d *integrationDispatcher) RunWithForcedTier(_ context.Context, sessionID, prompt string, tier config.SelectedModelType) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, integrationDispatchCall{
		SessionID: sessionID,
		Prompt:    prompt,
		Tier:      tier,
	})
	return nil
}

// ---------------------------------------------------------------------------
// TestIntegration_StepAdvancesTaskThroughRealDB proves the full
// derive→dispatch→advance loop with real production adapters against
// a real in-memory SQLite database.
// ---------------------------------------------------------------------------

func TestIntegration_StepAdvancesTaskThroughRealDB(t *testing.T) {
	t.Parallel()

	// 1. Set up real DB with migrations.
	q := setupAdapterTestDB(t)
	ctx := context.Background()

	// 2. Seed: one milestone, one slice, one task — all active/executing.
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedTask(t, q, "T01", "S01", "M001", "active", "executing", 1)

	// 3. Create real adapters + mocks for session/dispatch.
	querier := NewDBStateQuerier(q)
	advancer := NewDBStatusAdvancer(q)
	sessions := &integrationSessionCreator{}
	dispatcher := &integrationDispatcher{}
	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	eng := NewEngine(
		querier,
		sessions,
		dispatcher,
		advancer,
		nil, // verifier — disabled
		nil, // budgetChecker — disabled
		0,   // budgetCeiling — disabled
		nil, // stuckDetector — disabled
		nil, // contextMonitor — disabled
		broker,
		dir,
		slog.Default(),
		nil, // snapshotQuerier — disabled
	)

	// 4. Step 1: derive should find T01 (execute_task), dispatch, advance.
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	// Assert dispatcher was called once with execute_task tier.
	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 1, "Step 1 should dispatch exactly one unit")
	require.Equal(t, config.SelectedModelTypeMain, dispatcher.calls[0].Tier)
	require.Contains(t, dispatcher.calls[0].Prompt, "T01", "prompt should reference the task")
	dispatcher.mu.Unlock()

	// Assert DB: task is now completed.
	task, err := q.GetTask(ctx, "T01")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), task.Status)
	require.Equal(t, string(PhaseCompleted), task.Phase)

	// 5. Step 2: all tasks done, slice still in executing phase →
	//    DeriveState should return UnitSummarizeSlice. Dispatch +
	//    advance sets slice to completed.
	err = eng.Step(ctx, "M001")
	require.NoError(t, err)

	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 2, "Step 2 should dispatch the summarize unit")
	require.Equal(t, config.SelectedModelTypeBackground, dispatcher.calls[1].Tier,
		"summarize uses background tier")
	dispatcher.mu.Unlock()

	// Assert DB: slice is now completed.
	slice, err := q.GetSlice(ctx, "S01")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), slice.Status)
	require.Equal(t, string(PhaseCompleted), slice.Phase)

	// 6. Step 3: all slices completed → DeriveState returns
	//    UnitValidateMilestone. Dispatch + advance sets milestone completed.
	err = eng.Step(ctx, "M001")
	require.NoError(t, err)

	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 3, "Step 3 should dispatch the validate unit")
	require.Equal(t, config.SelectedModelTypeBackground, dispatcher.calls[2].Tier,
		"validate uses background tier")
	dispatcher.mu.Unlock()

	// Assert DB: milestone is now completed.
	milestone, err := q.GetMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), milestone.Status)
	require.Equal(t, string(PhaseCompleted), milestone.Phase)

	// 7. Step 4: nothing left → should return nil (errDone unwrapped).
	err = eng.Step(ctx, "M001")
	require.NoError(t, err, "Step after all done should return nil")

	// Dispatcher should not have been called again.
	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 3, "no additional dispatch after done")
	dispatcher.mu.Unlock()
}

// TestIntegration_StepMultipleTasksInSlice proves the engine processes
// multiple tasks sequentially before summarizing the slice.
func TestIntegration_StepMultipleTasksInSlice(t *testing.T) {
	t.Parallel()

	q := setupAdapterTestDB(t)
	ctx := context.Background()

	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedTask(t, q, "T01", "S01", "M001", "active", "executing", 1)
	seedTask(t, q, "T02", "S01", "M001", "active", "executing", 2)
	seedTask(t, q, "T03", "S01", "M001", "active", "executing", 3)

	querier := NewDBStateQuerier(q)
	advancer := NewDBStatusAdvancer(q)
	sessions := &integrationSessionCreator{}
	dispatcher := &integrationDispatcher{}
	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	eng := NewEngine(
		querier, sessions, dispatcher, advancer,
		nil, nil, 0, nil, nil, broker, dir, slog.Default(),
		nil, // snapshotQuerier — disabled
	)

	// Steps 1-3: execute each task.
	for i := 1; i <= 3; i++ {
		err := eng.Step(ctx, "M001")
		require.NoError(t, err, "step %d", i)
	}

	// All three tasks should be completed.
	for _, id := range []string{"T01", "T02", "T03"} {
		task, err := q.GetTask(ctx, id)
		require.NoError(t, err)
		require.Equal(t, string(StatusCompleted), task.Status, "task %s status", id)
	}

	dispatcher.mu.Lock()
	require.Len(t, dispatcher.calls, 3, "three task dispatches")
	// All three should be main tier (execute_task).
	for i := 0; i < 3; i++ {
		require.Equal(t, config.SelectedModelTypeMain, dispatcher.calls[i].Tier)
	}
	dispatcher.mu.Unlock()

	// Step 4: summarize slice.
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	slice, err := q.GetSlice(ctx, "S01")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), slice.Status)

	// Step 5: validate milestone.
	err = eng.Step(ctx, "M001")
	require.NoError(t, err)

	milestone, err := q.GetMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), milestone.Status)
}
