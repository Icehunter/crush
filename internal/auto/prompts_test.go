package auto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPrompt_AllUnitTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		unitType UnitType
		ctx      PromptContext
		contains []string
	}{
		{
			unitType: UnitResearch,
			ctx: PromptContext{
				MilestoneID:    "M001",
				MilestoneTitle: "Core Feature",
				SliceID:        "S01",
				SliceTitle:     "Setup Slice",
				SliceGoal:      "Set up the project structure",
				WorkingDir:     "/tmp/test",
				PriorSummaries: "Prior task completed setup.",
			},
			contains: []string{
				"S01", "Setup Slice", "M001", "Core Feature",
				"Set up the project structure", "/tmp/test",
				"Prior task completed setup.",
				"RESEARCH.md",
			},
		},
		{
			unitType: UnitPlanSlice,
			ctx: PromptContext{
				MilestoneID:    "M001",
				MilestoneTitle: "Core Feature",
				SliceID:        "S01",
				SliceTitle:     "Setup Slice",
				SliceGoal:      "Set up the project structure",
				WorkingDir:     "/tmp/test",
			},
			contains: []string{
				"S01", "Setup Slice", "M001", "Core Feature",
				"Set up the project structure", "/tmp/test",
				"PLAN.md",
			},
		},
		{
			unitType: UnitExecuteTask,
			ctx: PromptContext{
				MilestoneID:     "M001",
				MilestoneTitle:  "Core Feature",
				SliceID:         "S01",
				SliceTitle:      "Setup Slice",
				TaskID:          "T01",
				TaskTitle:       "Create files",
				TaskDescription: "Create the initial project files.",
				WorkingDir:      "/tmp/test",
			},
			contains: []string{
				"T01", "Create files", "S01", "Setup Slice",
				"M001", "Core Feature",
				"Create the initial project files.",
				"/tmp/test",
			},
		},
		{
			unitType: UnitSummarizeSlice,
			ctx: PromptContext{
				MilestoneID:    "M001",
				MilestoneTitle: "Core Feature",
				SliceID:        "S01",
				SliceTitle:     "Setup Slice",
				SliceGoal:      "Set up the project structure",
				WorkingDir:     "/tmp/test",
			},
			contains: []string{
				"S01", "Setup Slice", "M001", "Core Feature",
				"Set up the project structure", "/tmp/test",
			},
		},
		{
			unitType: UnitValidateMilestone,
			ctx: PromptContext{
				MilestoneID:    "M001",
				MilestoneTitle: "Core Feature",
				WorkingDir:     "/tmp/test",
			},
			contains: []string{
				"M001", "Core Feature", "/tmp/test",
				"success criteria",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.unitType), func(t *testing.T) {
			t.Parallel()
			result, err := BuildPrompt(tt.unitType, tt.ctx)
			require.NoError(t, err)
			require.NotEmpty(t, result)
			for _, s := range tt.contains {
				require.Contains(t, result, s, "output should contain %q", s)
			}
		})
	}
}

func TestBuildPrompt_InvalidUnitType(t *testing.T) {
	t.Parallel()
	_, err := BuildPrompt("invalid_type", PromptContext{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no template for unit type")
}

func TestBuildPrompt_EmptyOptionalFields(t *testing.T) {
	t.Parallel()

	// Execute task with no task description or prior summaries should
	// still render without error.
	result, err := BuildPrompt(UnitExecuteTask, PromptContext{
		MilestoneID:    "M001",
		MilestoneTitle: "Milestone",
		SliceID:        "S01",
		SliceTitle:     "Slice",
		TaskID:         "T01",
		TaskTitle:      "Task",
		WorkingDir:     "/tmp/test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)
	// Should not contain "Prior Context" header when PriorSummaries is empty.
	require.False(t, strings.Contains(result, "## Prior Context"),
		"empty PriorSummaries should not produce Prior Context section")
	// Should not contain "Task Description" header when TaskDescription is empty.
	require.False(t, strings.Contains(result, "## Task Description"),
		"empty TaskDescription should not produce Task Description section")
}

func TestBuildPrompt_WithPriorSummaries(t *testing.T) {
	t.Parallel()

	result, err := BuildPrompt(UnitResearch, PromptContext{
		MilestoneID:    "M001",
		MilestoneTitle: "Milestone",
		SliceID:        "S01",
		SliceTitle:     "Slice",
		SliceGoal:      "Goal",
		WorkingDir:     "/tmp/test",
		PriorSummaries: "T01 completed: built the foundation.",
	})
	require.NoError(t, err)
	require.Contains(t, result, "## Prior Context")
	require.Contains(t, result, "T01 completed: built the foundation.")
}

func TestBuildPrompt_TemplateNamesComplete(t *testing.T) {
	t.Parallel()

	// Every valid UnitType must have a corresponding template.
	allTypes := []UnitType{
		UnitResearch, UnitPlanSlice, UnitExecuteTask,
		UnitSummarizeSlice, UnitValidateMilestone,
	}
	for _, ut := range allTypes {
		_, ok := templateNames[ut]
		require.True(t, ok, "templateNames missing entry for %q", ut)
	}
}

func TestBuildInitPrompt_HappyPath(t *testing.T) {
	t.Parallel()

	result, err := BuildInitPrompt(InitPromptContext{
		Vision:     "Build a REST API with authentication",
		WorkingDir: "/tmp/project",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Contains(t, result, "Build a REST API with authentication")
	require.Contains(t, result, "/tmp/project")
	require.Contains(t, result, "create_milestone")
	require.Contains(t, result, "create_slice")
	require.Contains(t, result, "create_task")
	require.Contains(t, result, "M001")
}

func TestBuildInitPrompt_EmptyFields(t *testing.T) {
	t.Parallel()

	// Template should still render with empty fields.
	result, err := BuildInitPrompt(InitPromptContext{})
	require.NoError(t, err)
	require.NotEmpty(t, result)
	// The structure should still be present.
	require.Contains(t, result, "project planner")
}
