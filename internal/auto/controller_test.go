package auto

import (
	"context"
	"log/slog"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestBuildSnapshot — verifies SliceProgress counts from real DB data.
// ---------------------------------------------------------------------------

func TestBuildSnapshot(t *testing.T) {
	t.Parallel()
	q := setupAdapterTestDB(t)
	ctx := context.Background()

	// Seed: 1 milestone, 2 slices, 3 tasks (2 completed, 1 pending).
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedSlice(t, q, "S02", "M001", "pending", "pre_planning", 2, "")
	seedTask(t, q, "T01", "S01", "M001", "completed", "completed", 1)
	seedTask(t, q, "T02", "S01", "M001", "completed", "completed", 2)
	seedTask(t, q, "T03", "S01", "M001", "pending", "pre_planning", 3)

	sq := NewDBStateQuerier(q)
	snap := BuildSnapshot(ctx, sq, "M001", "running", "S01/T03", 1.23, 45.0)

	require.NotNil(t, snap)
	require.Equal(t, "M001", snap.MilestoneID)
	require.Equal(t, "running", snap.Status)
	require.Equal(t, "S01/T03", snap.ActiveUnit)
	require.InDelta(t, 1.23, snap.TotalCost, 0.001)
	require.InDelta(t, 45.0, snap.ElapsedSeconds, 0.001)
	require.Len(t, snap.Slices, 2)

	// S01 has 3 tasks, 2 completed.
	require.Equal(t, "S01", snap.Slices[0].ID)
	require.Equal(t, 3, snap.Slices[0].TasksTotal)
	require.Equal(t, 2, snap.Slices[0].TasksDone)

	// S02 has 0 tasks.
	require.Equal(t, "S02", snap.Slices[1].ID)
	require.Equal(t, 0, snap.Slices[1].TasksTotal)
	require.Equal(t, 0, snap.Slices[1].TasksDone)
}

// ---------------------------------------------------------------------------
// TestBuildSnapshot_Empty — empty DB returns snapshot with empty slices.
// ---------------------------------------------------------------------------

func TestBuildSnapshot_Empty(t *testing.T) {
	t.Parallel()
	q := setupAdapterTestDB(t)
	ctx := context.Background()
	sq := NewDBStateQuerier(q)

	snap := BuildSnapshot(ctx, sq, "MXXX", "idle", "", 0, 0)
	require.NotNil(t, snap)
	require.Equal(t, "MXXX", snap.MilestoneID)
	require.Empty(t, snap.Slices)
}

// ---------------------------------------------------------------------------
// TestEngineController_Interface — verifies EngineController satisfies the
// AutoController method set (StartAuto, PauseAuto, ResumeAuto, AutoStatus).
// We cannot reference model.AutoController directly due to an import cycle,
// so we define a local mirror interface for the compile-time check.
// ---------------------------------------------------------------------------

// autoController mirrors model.AutoController to avoid an import cycle.
type autoController interface {
	StartAuto(ctx context.Context, milestoneID string) error
	PauseAuto() error
	ResumeAuto(ctx context.Context) error
	AutoStatus() string
}

var _ autoController = (*EngineController)(nil)

func TestEngineController_Interface(t *testing.T) {
	t.Parallel()
	// The compile-time var _ check above is the real assertion. This
	// function exists so `go test -run TestEngineController_Interface`
	// reports a passing test.
	var ctrl autoController = &EngineController{}
	require.NotNil(t, ctrl)
}

// ---------------------------------------------------------------------------
// TestEngineController_AutoStatus — verifies status delegation.
// ---------------------------------------------------------------------------

func TestEngineController_AutoStatus(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	eng := NewEngine(
		&fakeStateQuerier{},
		&fakeSessionCreator{},
		&fakeDispatcher{},
		&fakeStatusAdvancer{},
		nil, nil, 0, nil, nil,
		broker, dir, slog.Default(), nil,
	)

	ctrl := NewEngineController(eng, nil)
	require.Equal(t, "idle", ctrl.AutoStatus())
}

// ---------------------------------------------------------------------------
// TestPublishWithSnapshot — engine with snapshotQuerier attaches snapshot.
// ---------------------------------------------------------------------------

func TestPublishWithSnapshot(t *testing.T) {
	t.Parallel()

	q := setupAdapterTestDB(t)
	ctx := context.Background()

	// Seed minimal data so the snapshot has content.
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedTask(t, q, "T01", "S01", "M001", "completed", "completed", 1)
	seedTask(t, q, "T02", "S01", "M001", "pending", "pre_planning", 2)

	sq := NewDBStateQuerier(q)
	broker := pubsub.NewBroker[AutoEvent]()
	dir := t.TempDir()

	eng := NewEngine(
		sq,
		&fakeSessionCreator{},
		&fakeDispatcher{},
		&fakeStatusAdvancer{},
		nil, nil, 0, nil, nil,
		broker, dir, slog.Default(), sq,
	)

	// Set milestoneID so publish can use it.
	eng.mu.Lock()
	eng.milestoneID = "M001"
	eng.mu.Unlock()

	// Subscribe before publishing.
	sub := broker.Subscribe(ctx)

	unit := Unit{MilestoneID: "M001", SliceID: "S01", TaskID: "T02", Type: UnitExecuteTask, Title: "Task T02"}
	eng.publish(EventUnitStarted, unit, nil, "test")

	// Read from subscription.
	select {
	case msg := <-sub:
		require.Equal(t, EventUnitStarted, msg.Type)
		require.NotNil(t, msg.Payload.Snapshot, "expected snapshot to be attached")
		require.Equal(t, "M001", msg.Payload.Snapshot.MilestoneID)
		require.Len(t, msg.Payload.Snapshot.Slices, 1)
		require.Equal(t, 2, msg.Payload.Snapshot.Slices[0].TasksTotal)
		require.Equal(t, 1, msg.Payload.Snapshot.Slices[0].TasksDone)
	default:
		t.Fatal("expected event on subscription channel")
	}

	_ = ctx // Suppress unused warning.
}

// ---------------------------------------------------------------------------
// Minimal fakes for EngineController tests.
// ---------------------------------------------------------------------------

type fakeStateQuerier struct{}

func (f *fakeStateQuerier) ListMilestones(context.Context) ([]MilestoneRow, error) {
	return nil, nil
}

func (f *fakeStateQuerier) ListSlicesByMilestone(context.Context, string) ([]SliceRow, error) {
	return nil, nil
}

func (f *fakeStateQuerier) ListTasksBySlice(context.Context, string) ([]TaskRow, error) {
	return nil, nil
}

type fakeSessionCreator struct{}

func (f *fakeSessionCreator) CreateSession(context.Context, string) (string, error) {
	return "sess-1", nil
}

func (f *fakeSessionCreator) CreateChildSession(context.Context, string, string, string) (string, error) {
	return "child-1", nil
}

type fakeDispatcher struct{}

func (f *fakeDispatcher) RunWithForcedTier(context.Context, string, string, config.SelectedModelType) error {
	return nil
}

type fakeStatusAdvancer struct{}

func (f *fakeStatusAdvancer) AdvanceStatus(context.Context, Unit) error {
	return nil
}
