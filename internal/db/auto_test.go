package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupTestDB opens an in-memory SQLite database, runs all migrations, and
// returns both the raw DB and a Queries instance.
func setupTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()

	params := url.Values{}
	params.Add("_pragma", "foreign_keys(ON)")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&%s", t.Name(), params.Encode())

	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })

	goose.SetBaseFS(FS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db, "migrations"))

	return db, New(db)
}

func TestAutoMilestones(t *testing.T) {
	t.Parallel()
	_, q := setupTestDB(t)
	ctx := context.Background()

	// Create.
	m, err := q.CreateMilestone(ctx, CreateMilestoneParams{
		ID:     "M001",
		Title:  "First milestone",
		Status: "pending",
		Phase:  "pre_planning",
	})
	require.NoError(t, err)
	require.Equal(t, "M001", m.ID)
	require.Equal(t, "First milestone", m.Title)
	require.Equal(t, "pending", m.Status)
	require.Equal(t, "pre_planning", m.Phase)

	// Get.
	got, err := q.GetMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Equal(t, m, got)

	// List.
	list, err := q.ListMilestones(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// Update status.
	updated, err := q.UpdateMilestoneStatus(ctx, UpdateMilestoneStatusParams{
		Status: "active",
		ID:     "M001",
	})
	require.NoError(t, err)
	require.Equal(t, "active", updated.Status)

	// Update phase.
	updated, err = q.UpdateMilestonePhase(ctx, UpdateMilestonePhaseParams{
		Phase: "planning",
		ID:    "M001",
	})
	require.NoError(t, err)
	require.Equal(t, "planning", updated.Phase)

	// Delete.
	require.NoError(t, q.DeleteMilestone(ctx, "M001"))
	list, err = q.ListMilestones(ctx)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestAutoSlices(t *testing.T) {
	t.Parallel()
	_, q := setupTestDB(t)
	ctx := context.Background()

	// Prerequisite milestone.
	_, err := q.CreateMilestone(ctx, CreateMilestoneParams{
		ID: "M001", Title: "M", Status: "active", Phase: "planning",
	})
	require.NoError(t, err)

	// Create two slices with different sort_order.
	s1, err := q.CreateSlice(ctx, CreateSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "Slice A",
		Status: "pending", Phase: "pre_planning", SortOrder: 2,
	})
	require.NoError(t, err)

	s2, err := q.CreateSlice(ctx, CreateSliceParams{
		ID: "S02", MilestoneID: "M001", Title: "Slice B",
		Status: "pending", Phase: "pre_planning", SortOrder: 1,
	})
	require.NoError(t, err)

	// Get by ID.
	got, err := q.GetSlice(ctx, "S01")
	require.NoError(t, err)
	require.Equal(t, s1, got)

	// List by milestone — ordered by sort_order ASC.
	list, err := q.ListSlicesByMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, s2.ID, list[0].ID) // sort_order 1 first
	require.Equal(t, s1.ID, list[1].ID) // sort_order 2 second

	// Update status.
	updated, err := q.UpdateSliceStatus(ctx, UpdateSliceStatusParams{
		Status: "active", ID: "S01",
	})
	require.NoError(t, err)
	require.Equal(t, "active", updated.Status)

	// Update phase.
	updated, err = q.UpdateSlicePhase(ctx, UpdateSlicePhaseParams{
		Phase: "executing", ID: "S01",
	})
	require.NoError(t, err)
	require.Equal(t, "executing", updated.Phase)

	// Delete one slice.
	require.NoError(t, q.DeleteSlice(ctx, "S02"))
	list, err = q.ListSlicesByMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestAutoTasks(t *testing.T) {
	t.Parallel()
	_, q := setupTestDB(t)
	ctx := context.Background()

	// Prerequisite milestone and slice.
	_, err := q.CreateMilestone(ctx, CreateMilestoneParams{
		ID: "M001", Title: "M", Status: "active", Phase: "planning",
	})
	require.NoError(t, err)
	_, err = q.CreateSlice(ctx, CreateSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "S",
		Status: "active", Phase: "executing", SortOrder: 1,
	})
	require.NoError(t, err)

	// Create two tasks with different sort_order.
	t1, err := q.CreateTask(ctx, CreateTaskParams{
		ID: "T01", SliceID: "S01", MilestoneID: "M001", Title: "Task A",
		Status: "pending", Phase: "pre_planning", SortOrder: 2,
	})
	require.NoError(t, err)

	t2, err := q.CreateTask(ctx, CreateTaskParams{
		ID: "T02", SliceID: "S01", MilestoneID: "M001", Title: "Task B",
		Status: "pending", Phase: "pre_planning", SortOrder: 1,
	})
	require.NoError(t, err)

	// Get by ID.
	got, err := q.GetTask(ctx, "T01")
	require.NoError(t, err)
	require.Equal(t, t1, got)

	// List by slice — ordered by sort_order ASC.
	bySlice, err := q.ListTasksBySlice(ctx, "S01")
	require.NoError(t, err)
	require.Len(t, bySlice, 2)
	require.Equal(t, t2.ID, bySlice[0].ID)
	require.Equal(t, t1.ID, bySlice[1].ID)

	// List by milestone.
	byMilestone, err := q.ListTasksByMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Len(t, byMilestone, 2)

	// Update status.
	updated, err := q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
		Status: "active", ID: "T01",
	})
	require.NoError(t, err)
	require.Equal(t, "active", updated.Status)

	// Update phase.
	updated, err = q.UpdateTaskPhase(ctx, UpdateTaskPhaseParams{
		Phase: "executing", ID: "T01",
	})
	require.NoError(t, err)
	require.Equal(t, "executing", updated.Phase)

	// Delete a task.
	require.NoError(t, q.DeleteTask(ctx, "T02"))
	bySlice, err = q.ListTasksBySlice(ctx, "S01")
	require.NoError(t, err)
	require.Len(t, bySlice, 1)
}

func TestAutoCascadeDelete(t *testing.T) {
	t.Parallel()
	_, q := setupTestDB(t)
	ctx := context.Background()

	// Create milestone → slice → task.
	_, err := q.CreateMilestone(ctx, CreateMilestoneParams{
		ID: "M001", Title: "M", Status: "active", Phase: "planning",
	})
	require.NoError(t, err)
	_, err = q.CreateSlice(ctx, CreateSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "S",
		Status: "active", Phase: "executing", SortOrder: 1,
	})
	require.NoError(t, err)
	_, err = q.CreateTask(ctx, CreateTaskParams{
		ID: "T01", SliceID: "S01", MilestoneID: "M001", Title: "T",
		Status: "pending", Phase: "pre_planning", SortOrder: 1,
	})
	require.NoError(t, err)

	// Delete the milestone — cascade should remove slice and task.
	require.NoError(t, q.DeleteMilestone(ctx, "M001"))

	_, err = q.GetSlice(ctx, "S01")
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = q.GetTask(ctx, "T01")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
