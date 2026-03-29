package auto

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/db"
)

// Tool name constants for the init planning tools.
const (
	CreateMilestoneToolName = "create_milestone"
	CreateSliceToolName     = "create_slice"
	CreateTaskToolName      = "create_task"
)

// --- create_milestone ---

type createMilestoneParams struct {
	ID    string `json:"id" description:"Unique milestone identifier, e.g. M001"`
	Title string `json:"title" description:"Short descriptive title for the milestone"`
}

// NewCreateMilestoneTool returns a fantasy.AgentTool that creates a milestone
// in the database. The first milestone created is set to active; subsequent
// milestones are pending.
func NewCreateMilestoneTool(q *db.Queries) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		CreateMilestoneToolName,
		"Create a milestone in the plan. The first milestone will be set to active automatically.",
		func(ctx context.Context, params createMilestoneParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if strings.TrimSpace(params.ID) == "" {
				return fantasy.NewTextResponse(`{"error":"id is required"}`), nil
			}
			if strings.TrimSpace(params.Title) == "" {
				return fantasy.NewTextResponse(`{"error":"title is required"}`), nil
			}

			// Determine status: first milestone is active, rest are pending.
			status := StatusPending
			milestones, err := q.ListMilestones(ctx)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("list milestones: %w", err)
			}
			if len(milestones) == 0 {
				status = StatusActive
			}

			m := Milestone{
				ID:     params.ID,
				Title:  params.Title,
				Status: status,
				Phase:  PhasePrePlanning,
			}

			created, err := q.CreateMilestone(ctx, m.ToDBCreate())
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf(`{"error":"failed to create milestone: %s"}`, err)), nil
			}

			resp, _ := json.Marshal(map[string]string{
				"id":     created.ID,
				"title":  created.Title,
				"status": created.Status,
			})
			return fantasy.NewTextResponse(string(resp)), nil
		},
	)
}

// --- create_slice ---

type createSliceParams struct {
	ID          string `json:"id" description:"Unique slice identifier, e.g. S01"`
	MilestoneID string `json:"milestone_id" description:"Parent milestone ID"`
	Title       string `json:"title" description:"Short descriptive title for the slice"`
	SortOrder   int64  `json:"sort_order" description:"Execution order within the milestone (1-based)"`
	DependsOn   string `json:"depends_on,omitempty" description:"Comma-separated list of slice IDs this slice depends on"`
}

// NewCreateSliceTool returns a fantasy.AgentTool that creates a slice in the
// database with status=pending and phase=pre_planning.
func NewCreateSliceTool(q *db.Queries) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		CreateSliceToolName,
		"Create a slice (vertical feature slice) within a milestone.",
		func(ctx context.Context, params createSliceParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if strings.TrimSpace(params.ID) == "" {
				return fantasy.NewTextResponse(`{"error":"id is required"}`), nil
			}
			if strings.TrimSpace(params.MilestoneID) == "" {
				return fantasy.NewTextResponse(`{"error":"milestone_id is required"}`), nil
			}
			if strings.TrimSpace(params.Title) == "" {
				return fantasy.NewTextResponse(`{"error":"title is required"}`), nil
			}

			s := Slice{
				ID:          params.ID,
				MilestoneID: params.MilestoneID,
				Title:       params.Title,
				Status:      StatusPending,
				Phase:       PhasePrePlanning,
				SortOrder:   params.SortOrder,
				DependsOn:   params.DependsOn,
			}

			created, err := q.CreateSlice(ctx, s.ToDBCreate())
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf(`{"error":"failed to create slice: %s"}`, err)), nil
			}

			resp, _ := json.Marshal(map[string]string{
				"id":           created.ID,
				"milestone_id": created.MilestoneID,
				"title":        created.Title,
			})
			return fantasy.NewTextResponse(string(resp)), nil
		},
	)
}

// --- create_task ---

type createTaskParams struct {
	ID          string `json:"id" description:"Unique task identifier, e.g. T01"`
	SliceID     string `json:"slice_id" description:"Parent slice ID"`
	MilestoneID string `json:"milestone_id" description:"Parent milestone ID"`
	Title       string `json:"title" description:"Short descriptive title for the task"`
	Description string `json:"description,omitempty" description:"Detailed description of what the task involves"`
	SortOrder   int64  `json:"sort_order" description:"Execution order within the slice (1-based)"`
}

// NewCreateTaskTool returns a fantasy.AgentTool that creates a task in the
// database with status=pending and phase=pre_planning.
func NewCreateTaskTool(q *db.Queries) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		CreateTaskToolName,
		"Create a task within a slice. Tasks are the smallest unit of work.",
		func(ctx context.Context, params createTaskParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if strings.TrimSpace(params.ID) == "" {
				return fantasy.NewTextResponse(`{"error":"id is required"}`), nil
			}
			if strings.TrimSpace(params.SliceID) == "" {
				return fantasy.NewTextResponse(`{"error":"slice_id is required"}`), nil
			}
			if strings.TrimSpace(params.MilestoneID) == "" {
				return fantasy.NewTextResponse(`{"error":"milestone_id is required"}`), nil
			}
			if strings.TrimSpace(params.Title) == "" {
				return fantasy.NewTextResponse(`{"error":"title is required"}`), nil
			}

			t := Task{
				ID:          params.ID,
				SliceID:     params.SliceID,
				MilestoneID: params.MilestoneID,
				Title:       params.Title,
				Status:      StatusPending,
				Phase:       PhasePrePlanning,
				SortOrder:   params.SortOrder,
				Description: params.Description,
			}

			created, err := q.CreateTask(ctx, t.ToDBCreate())
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf(`{"error":"failed to create task: %s"}`, err)), nil
			}

			resp, _ := json.Marshal(map[string]string{
				"id":           created.ID,
				"slice_id":     created.SliceID,
				"milestone_id": created.MilestoneID,
				"title":        created.Title,
			})
			return fantasy.NewTextResponse(string(resp)), nil
		},
	)
}
