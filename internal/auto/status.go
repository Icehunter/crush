package auto

// Status represents the lifecycle status of a milestone, slice, or task.
type Status string

const (
	// StatusPending indicates the item has not started.
	StatusPending Status = "pending"
	// StatusActive indicates the item is in progress.
	StatusActive Status = "active"
	// StatusCompleted indicates the item is finished.
	StatusCompleted Status = "completed"
	// StatusBlocked indicates the item cannot proceed.
	StatusBlocked Status = "blocked"
)

// IsValid returns true when the status is one of the known constants.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusActive, StatusCompleted, StatusBlocked:
		return true
	}
	return false
}

// Phase represents the current workflow phase of a milestone, slice, or task.
type Phase string

const (
	// PhasePrePlanning is the initial phase before formal planning begins.
	PhasePrePlanning Phase = "pre_planning"
	// PhasePlanning is the planning phase.
	PhasePlanning Phase = "planning"
	// PhaseResearching is the research phase.
	PhaseResearching Phase = "researching"
	// PhaseExecuting is the execution phase.
	PhaseExecuting Phase = "executing"
	// PhaseSummarizing is the summarization phase.
	PhaseSummarizing Phase = "summarizing"
	// PhaseValidating is the validation phase.
	PhaseValidating Phase = "validating"
	// PhaseCompleted indicates the phase workflow is finished.
	PhaseCompleted Phase = "completed"
)

// IsValid returns true when the phase is one of the known constants.
func (p Phase) IsValid() bool {
	switch p {
	case PhasePrePlanning, PhasePlanning, PhaseResearching, PhaseExecuting,
		PhaseSummarizing, PhaseValidating, PhaseCompleted:
		return true
	}
	return false
}
