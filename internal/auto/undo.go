package auto

import (
	"context"
	"fmt"
)

// StatusReverter abstracts the DB writes needed to undo a unit's completion.
type StatusReverter interface {
	RevertStatus(ctx context.Context, unit Unit) error
}

// UndoQuerier extends StateQuerier with the ability to find the most recently
// completed unit for undo purposes.
type UndoQuerier interface {
	StateQuerier
	// FindLastCompletedTask returns the most recently completed task in a
	// milestone, or an error if none found.
	FindLastCompletedTask(ctx context.Context, milestoneID string) (TaskRow, SliceRow, error)
}

// DBUndoQuerier wraps StateQuerier to implement UndoQuerier by walking
// slices and tasks in reverse order.
type DBUndoQuerier struct {
	q StateQuerier
}

// NewDBUndoQuerier creates a DBUndoQuerier.
func NewDBUndoQuerier(q StateQuerier) *DBUndoQuerier {
	return &DBUndoQuerier{q: q}
}

func (d *DBUndoQuerier) ListMilestones(ctx context.Context) ([]MilestoneRow, error) {
	return d.q.ListMilestones(ctx)
}

func (d *DBUndoQuerier) ListSlicesByMilestone(ctx context.Context, milestoneID string) ([]SliceRow, error) {
	return d.q.ListSlicesByMilestone(ctx, milestoneID)
}

func (d *DBUndoQuerier) ListTasksBySlice(ctx context.Context, sliceID string) ([]TaskRow, error) {
	return d.q.ListTasksBySlice(ctx, sliceID)
}

// FindLastCompletedTask walks slices in reverse sort order, then tasks in
// reverse sort order, returning the first completed task found.
func (d *DBUndoQuerier) FindLastCompletedTask(ctx context.Context, milestoneID string) (TaskRow, SliceRow, error) {
	slices, err := d.q.ListSlicesByMilestone(ctx, milestoneID)
	if err != nil {
		return TaskRow{}, SliceRow{}, fmt.Errorf("list slices: %w", err)
	}

	// Walk slices in reverse sort order.
	for i := len(slices) - 1; i >= 0; i-- {
		s := slices[i]
		tasks, err := d.q.ListTasksBySlice(ctx, s.ID)
		if err != nil {
			continue
		}
		// Walk tasks in reverse sort order.
		for j := len(tasks) - 1; j >= 0; j-- {
			t := tasks[j]
			if Status(t.Status) == StatusCompleted {
				return t, s, nil
			}
		}
	}

	return TaskRow{}, SliceRow{}, fmt.Errorf("no completed tasks found in milestone %s", milestoneID)
}

// UndoLastUnit reverts the most recently completed task back to active status.
// It returns the unit that was undone.
func UndoLastUnit(ctx context.Context, querier UndoQuerier, reverter StatusReverter, milestoneID string) (Unit, error) {
	task, slice, err := querier.FindLastCompletedTask(ctx, milestoneID)
	if err != nil {
		return Unit{}, err
	}

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: milestoneID,
		SliceID:     slice.ID,
		TaskID:      task.ID,
		Title:       task.Title,
	}

	if err := reverter.RevertStatus(ctx, unit); err != nil {
		return Unit{}, fmt.Errorf("revert status: %w", err)
	}

	return unit, nil
}
