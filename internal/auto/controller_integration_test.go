package auto

import (
	"context"
	"log/slog"
	"testing"

	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// TestControllerIntegration_FullPipeline sets up a real DB with milestone,
// slice, and task data, constructs a full EngineController with
// DBStateQuerier as snapshotQuerier, publishes an event via the engine, and
// verifies the received event carries a non-nil Snapshot with correct
// milestone data.
func TestControllerIntegration_FullPipeline(t *testing.T) {
	t.Parallel()

	q := setupAdapterTestDB(t)
	ctx := context.Background()

	// Seed: milestone M001 with 2 slices and 4 tasks.
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedSlice(t, q, "S02", "M001", "pending", "pre_planning", 2, "")
	seedTask(t, q, "T01", "S01", "M001", "completed", "completed", 1)
	seedTask(t, q, "T02", "S01", "M001", "completed", "completed", 2)
	seedTask(t, q, "T03", "S01", "M001", "pending", "pre_planning", 3)
	seedTask(t, q, "T04", "S02", "M001", "pending", "pre_planning", 1)

	// Construct real adapters from DB queries.
	querier := NewDBStateQuerier(q)
	advancer := NewDBStatusAdvancer(q)
	budgetChecker := NewDBBudgetChecker(q)
	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	// Build the engine with querier as snapshotQuerier.
	engine := NewEngine(
		querier,
		&fakeSessionCreator{},
		&fakeDispatcher{},
		advancer,
		nil, // verifier
		budgetChecker,
		100.0, // budgetCeiling
		nil,   // stuckDetector
		nil,   // contextMonitor
		broker,
		dir,
		slog.Default(),
		querier, // snapshotQuerier
	)

	// Create the controller — this is the real production wiring.
	ctrl := NewEngineController(engine, querier)

	// Verify initial status.
	require.Equal(t, "idle", ctrl.AutoStatus())

	// Set milestoneID so publish builds the right snapshot.
	engine.mu.Lock()
	engine.milestoneID = "M001"
	engine.mu.Unlock()

	// Subscribe to the broker before publishing.
	sub := broker.Subscribe(ctx)

	// Publish an event and verify the snapshot is populated from real DB data.
	unit := Unit{
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T03",
		Type:        UnitExecuteTask,
		Title:       "Execute T03",
	}
	engine.publish(EventUnitStarted, unit, nil, "integration-test")

	// Read from subscription.
	select {
	case msg := <-sub:
		require.Equal(t, EventUnitStarted, msg.Type)
		require.NotNil(t, msg.Payload.Snapshot, "expected snapshot to be attached")

		snap := msg.Payload.Snapshot
		require.Equal(t, "M001", snap.MilestoneID)
		require.Equal(t, "idle", snap.Status) // Engine not started, so state is idle.
		require.Contains(t, snap.ActiveUnit, "S01/T03")

		// Verify slice-level progress from real DB.
		require.Len(t, snap.Slices, 2)

		// S01: 3 tasks, 2 completed.
		require.Equal(t, "S01", snap.Slices[0].ID)
		require.Equal(t, 3, snap.Slices[0].TasksTotal)
		require.Equal(t, 2, snap.Slices[0].TasksDone)

		// S02: 1 task, 0 completed.
		require.Equal(t, "S02", snap.Slices[1].ID)
		require.Equal(t, 1, snap.Slices[1].TasksTotal)
		require.Equal(t, 0, snap.Slices[1].TasksDone)
	default:
		t.Fatal("expected event on subscription channel")
	}
}

// TestControllerIntegration_MultipleEvents verifies that multiple events
// each carry independently-built snapshots reflecting current DB state.
func TestControllerIntegration_MultipleEvents(t *testing.T) {
	t.Parallel()

	q := setupAdapterTestDB(t)
	ctx := context.Background()

	seedMilestone(t, q, "M002", "active", "executing")
	seedSlice(t, q, "S10", "M002", "active", "executing", 1, "")
	seedTask(t, q, "T10", "S10", "M002", "pending", "pre_planning", 1)

	querier := NewDBStateQuerier(q)
	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	engine := NewEngine(
		querier,
		&fakeSessionCreator{},
		&fakeDispatcher{},
		&fakeStatusAdvancer{},
		nil, nil, 0, nil, nil,
		broker, dir, slog.Default(), querier,
	)

	ctrl := NewEngineController(engine, querier)
	require.Equal(t, "idle", ctrl.AutoStatus())

	engine.mu.Lock()
	engine.milestoneID = "M002"
	engine.mu.Unlock()

	sub := broker.Subscribe(ctx)

	unit := Unit{MilestoneID: "M002", SliceID: "S10", TaskID: "T10", Type: UnitExecuteTask, Title: "Task T10"}

	// First event: task pending.
	engine.publish(EventUnitStarted, unit, nil, "test")
	msg1 := <-sub
	require.NotNil(t, msg1.Payload.Snapshot)
	require.Equal(t, 0, msg1.Payload.Snapshot.Slices[0].TasksDone)

	// Mark task completed in DB, then publish another event.
	seedUpdateTaskStatus(t, q, "T10", "completed")
	engine.publish(EventUnitCompleted, unit, nil, "test")
	msg2 := <-sub
	require.NotNil(t, msg2.Payload.Snapshot)
	require.Equal(t, 1, msg2.Payload.Snapshot.Slices[0].TasksDone)
}
