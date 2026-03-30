package auto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeReverter struct {
	revertedUnit Unit
	revertErr    error
}

func (f *fakeReverter) RevertStatus(_ context.Context, unit Unit) error {
	f.revertedUnit = unit
	return f.revertErr
}

func TestUndoLastUnit_Success(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "active", Phase: "executing", SortOrder: 1},
			},
		},
		tasks: map[string][]TaskRow{
			"S01": {
				{ID: "T01", Title: "Login", Status: "completed", SortOrder: 1},
				{ID: "T02", Title: "Logout", Status: "active", SortOrder: 2},
			},
		},
	}
	reverter := &fakeReverter{}
	undoQ := NewDBUndoQuerier(q)

	unit, err := UndoLastUnit(context.Background(), undoQ, reverter, "M001")
	require.NoError(t, err)
	require.Equal(t, "T01", unit.TaskID)
	require.Equal(t, "Login", unit.Title)
	require.Equal(t, "T01", reverter.revertedUnit.TaskID)
}

func TestUndoLastUnit_LastTaskInOrder(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "active", Phase: "executing", SortOrder: 1},
			},
		},
		tasks: map[string][]TaskRow{
			"S01": {
				{ID: "T01", Title: "Login", Status: "completed", SortOrder: 1},
				{ID: "T02", Title: "Logout", Status: "completed", SortOrder: 2},
				{ID: "T03", Title: "Token", Status: "active", SortOrder: 3},
			},
		},
	}
	reverter := &fakeReverter{}
	undoQ := NewDBUndoQuerier(q)

	// Should find T02 (last completed in sort order).
	unit, err := UndoLastUnit(context.Background(), undoQ, reverter, "M001")
	require.NoError(t, err)
	require.Equal(t, "T02", unit.TaskID)
}

func TestUndoLastUnit_NoCompleted(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "active", Phase: "executing", SortOrder: 1},
			},
		},
		tasks: map[string][]TaskRow{
			"S01": {
				{ID: "T01", Title: "Login", Status: "active", SortOrder: 1},
			},
		},
	}
	reverter := &fakeReverter{}
	undoQ := NewDBUndoQuerier(q)

	_, err := UndoLastUnit(context.Background(), undoQ, reverter, "M001")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no completed tasks")
}

func TestUndoLastUnit_MultipleSlices(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "completed", Phase: "completed", SortOrder: 1},
				{ID: "S02", Title: "API", Status: "active", Phase: "executing", SortOrder: 2},
			},
		},
		tasks: map[string][]TaskRow{
			"S01": {
				{ID: "T01", Title: "Login", Status: "completed", SortOrder: 1},
			},
			"S02": {
				{ID: "T02", Title: "Endpoint", Status: "completed", SortOrder: 1},
				{ID: "T03", Title: "Handler", Status: "active", SortOrder: 2},
			},
		},
	}
	reverter := &fakeReverter{}
	undoQ := NewDBUndoQuerier(q)

	// Should find T02 in S02 (last slice in reverse order with completed tasks).
	unit, err := UndoLastUnit(context.Background(), undoQ, reverter, "M001")
	require.NoError(t, err)
	require.Equal(t, "T02", unit.TaskID)
}
