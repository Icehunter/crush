package model

import "context"

// AutoController defines the interface for controlling auto-mode from the TUI.
type AutoController interface {
	// StartAuto begins auto-mode execution for the given milestone.
	StartAuto(ctx context.Context, milestoneID string) error
	// StopAuto stops auto-mode immediately by cancelling the context.
	StopAuto() error
	// PauseAuto pauses auto-mode after the current unit completes.
	PauseAuto() error
	// ResumeAuto resumes a paused auto-mode session.
	ResumeAuto(ctx context.Context) error
	// StepAuto executes exactly one unit then returns.
	StepAuto(ctx context.Context, milestoneID string) error
	// AutoStatus returns the current auto-mode state: "idle", "running", or "paused".
	AutoStatus() string
	// AutoQueue returns the pending dispatch queue for the given milestone.
	AutoQueue(ctx context.Context, milestoneID string) ([]string, error)
	// UndoLast reverts the most recently completed task and returns its description.
	UndoLast(ctx context.Context, milestoneID string) (string, error)
	// SkipUnit marks a task as skipped so auto-dispatch won't execute it.
	SkipUnit(ctx context.Context, taskID string) error
	// DispatchPhase creates and dispatches a one-shot unit for the given phase.
	DispatchPhase(ctx context.Context, milestoneID, phase string) error
	// Steer injects a guidance message into the current auto session.
	Steer(ctx context.Context, guidance string) error
	// History returns formatted recent execution history.
	History(ctx context.Context, count int) (string, error)
	// RateTier records user feedback on the last unit's model tier.
	RateTier(ctx context.Context, rating string) error
	// RunDoctor performs health checks, optionally auto-fixing issues.
	RunDoctor(ctx context.Context, fix bool) (string, error)
	// QuickTask dispatches a lightweight task without full planning.
	QuickTask(ctx context.Context, milestoneID, description string) error
	// StartFromTemplate creates a milestone from a workflow template.
	StartFromTemplate(ctx context.Context, templateID string) (string, error)
	// ParkMilestone parks a milestone, removing it from auto-dispatch.
	ParkMilestone(ctx context.Context, milestoneID string) error
	// UnparkMilestone restores a parked milestone to active status.
	UnparkMilestone(ctx context.Context, milestoneID string) error
	// Rethink triggers a conversational replan of the current milestone.
	Rethink(ctx context.Context, milestoneID string) error
	// GetPreferences returns current GSD preferences as formatted text.
	GetPreferences() (string, error)
	// SetPreference sets a single preference key=value.
	SetPreference(key, value string) error
	// CleanupWorktrees removes stale worktrees and merged branches.
	CleanupWorktrees(ctx context.Context) (string, error)
	// InitProject runs the interactive planning flow with the given vision.
	InitProject(ctx context.Context, vision string) error
}
