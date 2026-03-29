package auto

import (
	"time"

	"github.com/charmbracelet/crush/internal/pubsub"
)

// Event type constants for the auto loop lifecycle. These are used as the
// pubsub.EventType when publishing through a pubsub.Broker[AutoEvent].
const (
	EventUnitStarted         pubsub.EventType = "unit_started"
	EventUnitCompleted       pubsub.EventType = "unit_completed"
	EventUnitFailed          pubsub.EventType = "unit_failed"
	EventLoopPaused          pubsub.EventType = "loop_paused"
	EventLoopStopped         pubsub.EventType = "loop_stopped"
	EventStateTransition     pubsub.EventType = "state_transition"
	EventVerificationStarted pubsub.EventType = "verification_started"
	EventVerificationPassed  pubsub.EventType = "verification_passed"
	EventVerificationFailed  pubsub.EventType = "verification_failed"
	EventBudgetExceeded      pubsub.EventType = "budget_exceeded"
	EventStuckDetected       pubsub.EventType = "stuck_detected"
	EventContextPressure     pubsub.EventType = "context_pressure"
)

// AutoEvent is the payload published through the auto loop's event broker.
// It carries enough context for subscribers to understand what happened
// without needing to query the database.
type AutoEvent struct {
	// Unit describes the work item this event relates to. Zero-value for
	// loop-level events (paused, stopped).
	Unit Unit `json:"unit"`
	// Error is non-nil when the event signals a failure.
	Error error `json:"error,omitempty"`
	// Timestamp records when the event was created.
	Timestamp time.Time `json:"timestamp"`
	// Message provides optional human-readable context.
	Message string `json:"message,omitempty"`
	// Snapshot is an optional point-in-time snapshot of auto-mode progress,
	// attached to events consumed by the TUI.
	Snapshot *AutoSnapshot `json:"snapshot,omitempty"`
}

// NewAutoEvent creates an AutoEvent with the current timestamp.
func NewAutoEvent(unit Unit, err error, message string) AutoEvent {
	return AutoEvent{
		Unit:      unit,
		Error:     err,
		Timestamp: time.Now(),
		Message:   message,
	}
}

// AutoSnapshot is a point-in-time snapshot of auto-mode progress.
type AutoSnapshot struct {
	MilestoneID    string
	MilestoneTitle string
	Slices         []SliceProgress
	ActiveUnit     string
	TotalCost      float64
	ElapsedSeconds float64
	Status         string // running, paused, completed, error
}

// SliceProgress tracks the progress of a single slice.
type SliceProgress struct {
	ID         string
	Title      string
	Status     string // pending, active, completed, blocked
	TasksDone  int
	TasksTotal int
}
