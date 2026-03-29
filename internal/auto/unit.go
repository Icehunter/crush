package auto

import "fmt"

// UnitType identifies the kind of work to dispatch next.
type UnitType string

const (
	// UnitResearch dispatches a research agent for a slice.
	UnitResearch UnitType = "research"
	// UnitPlanSlice dispatches a planner agent for a slice.
	UnitPlanSlice UnitType = "plan_slice"
	// UnitExecuteTask dispatches a worker agent for a task.
	UnitExecuteTask UnitType = "execute_task"
	// UnitSummarizeSlice dispatches a summarizer for a completed slice.
	UnitSummarizeSlice UnitType = "summarize_slice"
	// UnitValidateMilestone dispatches validation for a milestone.
	UnitValidateMilestone UnitType = "validate_milestone"
)

// IsValid returns true when the unit type is one of the known constants.
func (u UnitType) IsValid() bool {
	switch u {
	case UnitResearch, UnitPlanSlice, UnitExecuteTask,
		UnitSummarizeSlice, UnitValidateMilestone:
		return true
	}
	return false
}

// Unit describes the next piece of work the engine should dispatch. A
// zero-value Unit (Type == "") signals that nothing is dispatchable —
// either all work is done or the remaining work is blocked.
type Unit struct {
	// Type identifies the kind of work.
	Type UnitType `json:"type"`
	// MilestoneID is always set when Type is non-empty.
	MilestoneID string `json:"milestone_id"`
	// SliceID is set for slice-scoped and task-scoped units.
	SliceID string `json:"slice_id,omitempty"`
	// TaskID is set only for execute_task units.
	TaskID string `json:"task_id,omitempty"`
	// Title is a human-readable description for logs and events.
	Title string `json:"title"`
}

// IsDone returns true when no dispatchable work remains.
func (u Unit) IsDone() bool {
	return u.Type == ""
}

// String returns a compact human-readable representation.
func (u Unit) String() string {
	if u.IsDone() {
		return "done"
	}
	switch u.Type {
	case UnitExecuteTask:
		return fmt.Sprintf("%s %s/%s/%s: %s", u.Type, u.MilestoneID, u.SliceID, u.TaskID, u.Title)
	case UnitResearch, UnitPlanSlice, UnitSummarizeSlice:
		return fmt.Sprintf("%s %s/%s: %s", u.Type, u.MilestoneID, u.SliceID, u.Title)
	default:
		return fmt.Sprintf("%s %s: %s", u.Type, u.MilestoneID, u.Title)
	}
}
