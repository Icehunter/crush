package auto

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database with all migrations applied
// and returns a *db.Queries. Uses db.New() instead of db.Prepare() to avoid
// prepared-statement errors from schema mismatches in unrelated tables.
func setupTestDB(t *testing.T) *db.Queries {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	goose.SetBaseFS(db.FS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(sqlDB, "migrations"))

	return db.New(sqlDB)
}

// runTool is a test helper that marshals params to JSON and invokes the tool.
func runTool(t *testing.T, tool fantasy.AgentTool, name string, params any) fantasy.ToolResponse {
	t.Helper()

	input, err := json.Marshal(params)
	require.NoError(t, err)

	call := fantasy.ToolCall{
		ID:    "test-call",
		Name:  name,
		Input: string(input),
	}

	resp, err := tool.Run(context.Background(), call)
	require.NoError(t, err)
	return resp
}

// parseResponse extracts the JSON body into a map.
func parseResponse(t *testing.T, resp fantasy.ToolResponse) map[string]string {
	t.Helper()
	var m map[string]string
	require.NoError(t, json.Unmarshal([]byte(resp.Content), &m))
	return m
}

// --- create_milestone tests ---

func TestCreateMilestoneTool_HappyPath(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID:    "M001",
		Title: "Build core feature",
	})

	require.False(t, resp.IsError)
	m := parseResponse(t, resp)
	require.Equal(t, "M001", m["id"])
	require.Equal(t, "Build core feature", m["title"])
	require.Equal(t, "active", m["status"], "first milestone should be active")

	// Verify DB record.
	got, err := q.GetMilestone(context.Background(), "M001")
	require.NoError(t, err)
	require.Equal(t, "M001", got.ID)
	require.Equal(t, string(StatusActive), got.Status)
	require.Equal(t, string(PhasePrePlanning), got.Phase)
}

func TestCreateMilestoneTool_SecondMilestoneIsPending(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	// Create first (active).
	runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "First",
	})

	// Create second (pending).
	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M002", Title: "Second",
	})

	m := parseResponse(t, resp)
	require.Equal(t, "pending", m["status"], "second milestone should be pending")
}

func TestCreateMilestoneTool_EmptyID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "", Title: "No ID",
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "id is required")
}

func TestCreateMilestoneTool_EmptyTitle(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "",
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "title is required")
}

func TestCreateMilestoneTool_WhitespaceOnlyID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "   ", Title: "Some title",
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "id is required")
}

func TestCreateMilestoneTool_DuplicateID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "First",
	})

	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Duplicate",
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "failed to create milestone")
}

// --- create_slice tests ---

func TestCreateSliceTool_HappyPath(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	// Create parent milestone first.
	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})

	tool := NewCreateSliceTool(q)
	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID:          "S01",
		MilestoneID: "M001",
		Title:       "Core slice",
		SortOrder:   1,
	})

	require.False(t, resp.IsError)
	m := parseResponse(t, resp)
	require.Equal(t, "S01", m["id"])
	require.Equal(t, "M001", m["milestone_id"])

	// Verify DB.
	got, err := q.GetSlice(context.Background(), "S01")
	require.NoError(t, err)
	require.Equal(t, string(StatusPending), got.Status)
	require.Equal(t, string(PhasePrePlanning), got.Phase)
	require.Equal(t, int64(1), got.SortOrder)
}

func TestCreateSliceTool_WithDependsOn(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})

	tool := NewCreateSliceTool(q)
	runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "First", SortOrder: 1,
	})

	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID: "S02", MilestoneID: "M001", Title: "Second", SortOrder: 2, DependsOn: "S01",
	})

	require.False(t, resp.IsError)
	got, err := q.GetSlice(context.Background(), "S02")
	require.NoError(t, err)
	require.True(t, got.DependsOn.Valid)
	require.Equal(t, "S01", got.DependsOn.String)
}

func TestCreateSliceTool_EmptyID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateSliceTool(q)

	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		MilestoneID: "M001", Title: "No ID", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "id is required")
}

func TestCreateSliceTool_EmptyMilestoneID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateSliceTool(q)

	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID: "S01", Title: "No parent", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "milestone_id is required")
}

func TestCreateSliceTool_EmptyTitle(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateSliceTool(q)

	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "title is required")
}

func TestCreateSliceTool_InvalidMilestoneID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateSliceTool(q)

	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "NONEXISTENT", Title: "Bad parent", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "failed to create slice")
}

func TestCreateSliceTool_SortOrderZero(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})

	tool := NewCreateSliceTool(q)
	resp := runTool(t, tool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "Zero order", SortOrder: 0,
	})

	require.False(t, resp.IsError)
	got, err := q.GetSlice(context.Background(), "S01")
	require.NoError(t, err)
	require.Equal(t, int64(0), got.SortOrder)
}

// --- create_task tests ---

func TestCreateTaskTool_HappyPath(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	// Create parent milestone and slice.
	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})
	sTool := NewCreateSliceTool(q)
	runTool(t, sTool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "Slice", SortOrder: 1,
	})

	tool := NewCreateTaskTool(q)
	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		ID: "T01", SliceID: "S01", MilestoneID: "M001",
		Title: "Implement feature", Description: "Build the thing", SortOrder: 1,
	})

	require.False(t, resp.IsError)
	m := parseResponse(t, resp)
	require.Equal(t, "T01", m["id"])
	require.Equal(t, "S01", m["slice_id"])
	require.Equal(t, "M001", m["milestone_id"])
	require.Equal(t, "Implement feature", m["title"])

	// Verify DB.
	got, err := q.GetTask(context.Background(), "T01")
	require.NoError(t, err)
	require.Equal(t, string(StatusPending), got.Status)
	require.Equal(t, string(PhasePrePlanning), got.Phase)
	require.Equal(t, int64(1), got.SortOrder)
	require.True(t, got.Description.Valid)
	require.Equal(t, "Build the thing", got.Description.String)
}

func TestCreateTaskTool_EmptyID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateTaskTool(q)

	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		SliceID: "S01", MilestoneID: "M001", Title: "No ID", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "id is required")
}

func TestCreateTaskTool_EmptySliceID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateTaskTool(q)

	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		ID: "T01", MilestoneID: "M001", Title: "No slice", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "slice_id is required")
}

func TestCreateTaskTool_EmptyMilestoneID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateTaskTool(q)

	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		ID: "T01", SliceID: "S01", Title: "No milestone", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "milestone_id is required")
}

func TestCreateTaskTool_EmptyTitle(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateTaskTool(q)

	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		ID: "T01", SliceID: "S01", MilestoneID: "M001", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "title is required")
}

func TestCreateTaskTool_InvalidSliceID(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})

	tool := NewCreateTaskTool(q)
	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		ID: "T01", SliceID: "NONEXISTENT", MilestoneID: "M001",
		Title: "Bad parent", SortOrder: 1,
	})

	m := parseResponse(t, resp)
	require.Contains(t, m["error"], "failed to create task")
}

func TestCreateTaskTool_EmptyDescription(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})
	sTool := NewCreateSliceTool(q)
	runTool(t, sTool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "Slice", SortOrder: 1,
	})

	tool := NewCreateTaskTool(q)
	resp := runTool(t, tool, CreateTaskToolName, createTaskParams{
		ID: "T01", SliceID: "S01", MilestoneID: "M001",
		Title: "No description", SortOrder: 1,
	})

	require.False(t, resp.IsError)
	got, err := q.GetTask(context.Background(), "T01")
	require.NoError(t, err)
	require.False(t, got.Description.Valid, "empty description should be NULL")
}

func TestCreateTaskTool_SortOrderRespected(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)

	mTool := NewCreateMilestoneTool(q)
	runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Milestone",
	})
	sTool := NewCreateSliceTool(q)
	runTool(t, sTool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "Slice", SortOrder: 1,
	})

	tool := NewCreateTaskTool(q)
	for i, id := range []string{"T01", "T02", "T03"} {
		runTool(t, tool, CreateTaskToolName, createTaskParams{
			ID: id, SliceID: "S01", MilestoneID: "M001",
			Title: "Task " + id, SortOrder: int64(i + 1),
		})
	}

	tasks, err := q.ListTasksBySlice(context.Background(), "S01")
	require.NoError(t, err)
	require.Len(t, tasks, 3)
	require.Equal(t, "T01", tasks[0].ID)
	require.Equal(t, "T02", tasks[1].ID)
	require.Equal(t, "T03", tasks[2].ID)
}

func TestCreateMilestoneTool_LongTitle(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	tool := NewCreateMilestoneTool(q)

	longTitle := strings.Repeat("A", 1000)
	resp := runTool(t, tool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: longTitle,
	})

	require.False(t, resp.IsError)
	got, err := q.GetMilestone(context.Background(), "M001")
	require.NoError(t, err)
	require.Equal(t, longTitle, got.Title)
}
