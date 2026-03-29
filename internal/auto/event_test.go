package auto

import (
	"errors"
	"testing"

	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

func TestEventConstants_Unique(t *testing.T) {
	t.Parallel()

	types := []pubsub.EventType{
		EventUnitStarted,
		EventUnitCompleted,
		EventUnitFailed,
		EventLoopPaused,
		EventLoopStopped,
		EventStateTransition,
		EventVerificationStarted,
		EventVerificationPassed,
		EventVerificationFailed,
		EventBudgetExceeded,
		EventStuckDetected,
		EventContextPressure,
	}

	seen := make(map[pubsub.EventType]bool, len(types))
	for _, et := range types {
		require.NotEmpty(t, string(et), "event type constant must not be empty")
		require.False(t, seen[et], "duplicate event type: %s", et)
		seen[et] = true
	}

	require.Len(t, seen, 12, "expected exactly 12 distinct event types")
}

func TestAutoSnapshot_Construction(t *testing.T) {
	t.Parallel()

	snap := AutoSnapshot{
		MilestoneID:    "M001",
		MilestoneTitle: "Foundation",
		Slices: []SliceProgress{
			{ID: "S01", Title: "Event Wiring", Status: "completed", TasksDone: 3, TasksTotal: 3},
			{ID: "S02", Title: "Sidebar Panel", Status: "active", TasksDone: 1, TasksTotal: 4},
			{ID: "S03", Title: "Integration", Status: "pending", TasksDone: 0, TasksTotal: 2},
		},
		ActiveUnit:     "M001/S02/T02",
		TotalCost:      1.23,
		ElapsedSeconds: 456.7,
		Status:         "running",
	}

	require.Equal(t, "M001", snap.MilestoneID)
	require.Equal(t, "Foundation", snap.MilestoneTitle)
	require.Len(t, snap.Slices, 3)
	require.Equal(t, "completed", snap.Slices[0].Status)
	require.Equal(t, 3, snap.Slices[0].TasksDone)
	require.Equal(t, "M001/S02/T02", snap.ActiveUnit)
	require.InDelta(t, 1.23, snap.TotalCost, 0.001)
	require.Equal(t, "running", snap.Status)
}

func TestNewAutoEvent(t *testing.T) {
	t.Parallel()

	unit := Unit{MilestoneID: "M001", SliceID: "S01", TaskID: "T01"}
	ev := NewAutoEvent(unit, nil, "test message")

	require.Equal(t, unit, ev.Unit)
	require.NoError(t, ev.Error)
	require.Equal(t, "test message", ev.Message)
	require.False(t, ev.Timestamp.IsZero())
	require.Nil(t, ev.Snapshot)
}

func TestAutoEvent_WithError(t *testing.T) {
	t.Parallel()

	unit := Unit{MilestoneID: "M001", SliceID: "S01", TaskID: "T01"}
	testErr := errors.New("dispatch failed")
	ev := NewAutoEvent(unit, testErr, "failure")

	require.ErrorIs(t, ev.Error, testErr)
	require.Equal(t, "failure", ev.Message)
}

func TestAutoEvent_WithSnapshot(t *testing.T) {
	t.Parallel()

	snap := &AutoSnapshot{
		MilestoneID: "M002",
		Status:      "running",
		Slices: []SliceProgress{
			{ID: "S01", Title: "First", Status: "active", TasksDone: 0, TasksTotal: 1},
		},
	}

	ev := AutoEvent{
		Unit:     Unit{MilestoneID: "M002", SliceID: "S01", TaskID: "T01"},
		Snapshot: snap,
	}

	require.NotNil(t, ev.Snapshot)
	require.Equal(t, "running", ev.Snapshot.Status)
	require.Len(t, ev.Snapshot.Slices, 1)
	require.Equal(t, "M002", ev.Unit.MilestoneID)
}
