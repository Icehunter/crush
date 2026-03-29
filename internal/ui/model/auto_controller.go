package model

import "context"

// AutoController defines the interface for controlling auto-mode from the TUI.
type AutoController interface {
	// StartAuto begins auto-mode execution for the given milestone.
	StartAuto(ctx context.Context, milestoneID string) error
	// PauseAuto pauses auto-mode after the current unit completes.
	PauseAuto() error
	// ResumeAuto resumes a paused auto-mode session.
	ResumeAuto(ctx context.Context) error
	// AutoStatus returns the current auto-mode state: "idle", "running", or "paused".
	AutoStatus() string
}
