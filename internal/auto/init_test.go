package auto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunInit_CreatesStructuredPlan simulates the LLM tool-call sequence that
// RunInit would produce and verifies milestones, slices, and tasks are
// persisted with correct status, phase, sort_order, and relationships.
func TestRunInit_CreatesStructuredPlan(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	ctx := context.Background()

	mTool := NewCreateMilestoneTool(q)
	sTool := NewCreateSliceTool(q)
	tTool := NewCreateTaskTool(q)

	// Simulate LLM creating a structured plan.
	resp := runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "Build core API",
	})
	require.False(t, resp.IsError)

	resp = runTool(t, sTool, CreateSliceToolName, createSliceParams{
		ID: "S01", MilestoneID: "M001", Title: "Auth module", SortOrder: 1,
	})
	require.False(t, resp.IsError)

	resp = runTool(t, sTool, CreateSliceToolName, createSliceParams{
		ID: "S02", MilestoneID: "M001", Title: "Data layer", SortOrder: 2, DependsOn: "S01",
	})
	require.False(t, resp.IsError)

	resp = runTool(t, tTool, CreateTaskToolName, createTaskParams{
		ID: "T01", SliceID: "S01", MilestoneID: "M001",
		Title: "Set up JWT middleware", SortOrder: 1,
	})
	require.False(t, resp.IsError)

	resp = runTool(t, tTool, CreateTaskToolName, createTaskParams{
		ID: "T02", SliceID: "S01", MilestoneID: "M001",
		Title: "Add login endpoint", SortOrder: 2,
	})
	require.False(t, resp.IsError)

	resp = runTool(t, tTool, CreateTaskToolName, createTaskParams{
		ID: "T03", SliceID: "S02", MilestoneID: "M001",
		Title: "Create DB schema", Description: "Design tables", SortOrder: 1,
	})
	require.False(t, resp.IsError)

	// --- Verify milestones ---
	milestones, err := q.ListMilestones(ctx)
	require.NoError(t, err)
	require.Len(t, milestones, 1)
	require.Equal(t, "M001", milestones[0].ID)
	require.Equal(t, string(StatusActive), milestones[0].Status)
	require.Equal(t, string(PhasePrePlanning), milestones[0].Phase)

	// --- Verify slices ---
	slices, err := q.ListSlicesByMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Len(t, slices, 2)
	require.Equal(t, "S01", slices[0].ID)
	require.Equal(t, int64(1), slices[0].SortOrder)
	require.Equal(t, string(StatusPending), slices[0].Status)
	require.Equal(t, "S02", slices[1].ID)
	require.Equal(t, int64(2), slices[1].SortOrder)
	require.True(t, slices[1].DependsOn.Valid)
	require.Equal(t, "S01", slices[1].DependsOn.String)

	// --- Verify tasks ---
	tasksS01, err := q.ListTasksBySlice(ctx, "S01")
	require.NoError(t, err)
	require.Len(t, tasksS01, 2)
	require.Equal(t, "T01", tasksS01[0].ID)
	require.Equal(t, int64(1), tasksS01[0].SortOrder)
	require.Equal(t, "M001", tasksS01[0].MilestoneID)
	require.Equal(t, "T02", tasksS01[1].ID)
	require.Equal(t, int64(2), tasksS01[1].SortOrder)

	tasksS02, err := q.ListTasksBySlice(ctx, "S02")
	require.NoError(t, err)
	require.Len(t, tasksS02, 1)
	require.Equal(t, "T03", tasksS02[0].ID)
	require.Equal(t, "S02", tasksS02[0].SliceID)
	require.True(t, tasksS02[0].Description.Valid)
	require.Equal(t, "Design tables", tasksS02[0].Description.String)

	// Verify all tasks are pending.
	allTasks, err := q.ListTasksByMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Len(t, allTasks, 3)
	for _, task := range allTasks {
		require.Equal(t, string(StatusPending), task.Status)
		require.Equal(t, string(PhasePrePlanning), task.Phase)
	}
}

// TestRunInit_FirstMilestoneIsActive creates two milestones via tools and
// verifies the first is active while the second is pending.
func TestRunInit_FirstMilestoneIsActive(t *testing.T) {
	t.Parallel()
	q := setupTestDB(t)
	ctx := context.Background()

	mTool := NewCreateMilestoneTool(q)

	resp := runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M001", Title: "First milestone",
	})
	require.False(t, resp.IsError)
	m := parseResponse(t, resp)
	require.Equal(t, "active", m["status"])

	resp = runTool(t, mTool, CreateMilestoneToolName, createMilestoneParams{
		ID: "M002", Title: "Second milestone",
	})
	require.False(t, resp.IsError)
	m = parseResponse(t, resp)
	require.Equal(t, "pending", m["status"])

	// Double-check from DB.
	first, err := q.GetMilestone(ctx, "M001")
	require.NoError(t, err)
	require.Equal(t, string(StatusActive), first.Status)

	second, err := q.GetMilestone(ctx, "M002")
	require.NoError(t, err)
	require.Equal(t, string(StatusPending), second.Status)
}

// TestBuildInitPrompt_RendersVision verifies the init prompt template renders
// with the vision text and contains tool usage instructions.
func TestBuildInitPrompt_RendersVision(t *testing.T) {
	t.Parallel()

	result, err := BuildInitPrompt(InitPromptContext{
		Vision:     "Build a real-time chat application with WebSocket support",
		WorkingDir: "/home/user/project",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// Vision is embedded.
	require.Contains(t, result, "Build a real-time chat application with WebSocket support")
	// Working directory is embedded.
	require.Contains(t, result, "/home/user/project")
	// Tool names are mentioned.
	require.Contains(t, result, "create_milestone")
	require.Contains(t, result, "create_slice")
	require.Contains(t, result, "create_task")
	// Milestone conventions are present.
	require.Contains(t, result, "M001")
}
