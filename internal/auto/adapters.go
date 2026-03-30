package auto

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/session"
)

// ---------------------------------------------------------------------------
// DBStateQuerier implements StateQuerier backed by sqlc Queries.
// ---------------------------------------------------------------------------

// DBStateQuerier wraps *db.Queries to satisfy the StateQuerier interface.
type DBStateQuerier struct {
	q *db.Queries
}

// NewDBStateQuerier creates a DBStateQuerier.
func NewDBStateQuerier(q *db.Queries) *DBStateQuerier {
	return &DBStateQuerier{q: q}
}

// ListMilestones returns all milestones projected to MilestoneRow.
func (d *DBStateQuerier) ListMilestones(ctx context.Context) ([]MilestoneRow, error) {
	rows, err := d.q.ListMilestones(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MilestoneRow, len(rows))
	for i, r := range rows {
		out[i] = MilestoneRow{
			ID:     r.ID,
			Title:  r.Title,
			Status: r.Status,
			Phase:  r.Phase,
		}
	}
	return out, nil
}

// ListSlicesByMilestone returns slices for a milestone projected to SliceRow.
func (d *DBStateQuerier) ListSlicesByMilestone(ctx context.Context, milestoneID string) ([]SliceRow, error) {
	rows, err := d.q.ListSlicesByMilestone(ctx, milestoneID)
	if err != nil {
		return nil, err
	}
	out := make([]SliceRow, len(rows))
	for i, r := range rows {
		dep := ""
		if r.DependsOn.Valid {
			dep = r.DependsOn.String
		}
		out[i] = SliceRow{
			ID:        r.ID,
			Title:     r.Title,
			Status:    r.Status,
			Phase:     r.Phase,
			SortOrder: r.SortOrder,
			DependsOn: dep,
		}
	}
	return out, nil
}

// ListTasksBySlice returns tasks for a slice projected to TaskRow.
func (d *DBStateQuerier) ListTasksBySlice(ctx context.Context, sliceID string) ([]TaskRow, error) {
	rows, err := d.q.ListTasksBySlice(ctx, sliceID)
	if err != nil {
		return nil, err
	}
	out := make([]TaskRow, len(rows))
	for i, r := range rows {
		out[i] = TaskRow{
			ID:        r.ID,
			Title:     r.Title,
			Status:    r.Status,
			Phase:     r.Phase,
			SortOrder: r.SortOrder,
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// DBStatusAdvancer implements StatusAdvancer backed by sqlc Queries.
// ---------------------------------------------------------------------------

// DBStatusAdvancer wraps *db.Queries to satisfy the StatusAdvancer interface.
type DBStatusAdvancer struct {
	q *db.Queries
}

// NewDBStatusAdvancer creates a DBStatusAdvancer.
func NewDBStatusAdvancer(q *db.Queries) *DBStatusAdvancer {
	return &DBStatusAdvancer{q: q}
}

// AdvanceStatus updates the DB to reflect that the given unit's work is
// complete. The phase transitions match what DeriveState expects:
//
//   - UnitResearch      → slice phase = planning
//   - UnitPlanSlice     → slice phase = executing
//   - UnitExecuteTask   → task status = completed, task phase = completed
//   - UnitSummarizeSlice → slice status = completed, slice phase = completed
//   - UnitValidateMilestone → milestone status = completed, milestone phase = completed
func (d *DBStatusAdvancer) AdvanceStatus(ctx context.Context, unit Unit) error {
	switch unit.Type {
	case UnitResearch:
		_, err := d.q.UpdateSlicePhase(ctx, db.UpdateSlicePhaseParams{
			Phase: string(PhasePlanning),
			ID:    unit.SliceID,
		})
		return err

	case UnitPlanSlice:
		_, err := d.q.UpdateSlicePhase(ctx, db.UpdateSlicePhaseParams{
			Phase: string(PhaseExecuting),
			ID:    unit.SliceID,
		})
		return err

	case UnitExecuteTask:
		if _, err := d.q.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
			Status: string(StatusCompleted),
			ID:     unit.TaskID,
		}); err != nil {
			return fmt.Errorf("update task status: %w", err)
		}
		if _, err := d.q.UpdateTaskPhase(ctx, db.UpdateTaskPhaseParams{
			Phase: string(PhaseCompleted),
			ID:    unit.TaskID,
		}); err != nil {
			return fmt.Errorf("update task phase: %w", err)
		}
		return nil

	case UnitSummarizeSlice:
		if _, err := d.q.UpdateSliceStatus(ctx, db.UpdateSliceStatusParams{
			Status: string(StatusCompleted),
			ID:     unit.SliceID,
		}); err != nil {
			return fmt.Errorf("update slice status: %w", err)
		}
		if _, err := d.q.UpdateSlicePhase(ctx, db.UpdateSlicePhaseParams{
			Phase: string(PhaseCompleted),
			ID:    unit.SliceID,
		}); err != nil {
			return fmt.Errorf("update slice phase: %w", err)
		}
		return nil

	case UnitValidateMilestone:
		if _, err := d.q.UpdateMilestoneStatus(ctx, db.UpdateMilestoneStatusParams{
			Status: string(StatusCompleted),
			ID:     unit.MilestoneID,
		}); err != nil {
			return fmt.Errorf("update milestone status: %w", err)
		}
		if _, err := d.q.UpdateMilestonePhase(ctx, db.UpdateMilestonePhaseParams{
			Phase: string(PhaseCompleted),
			ID:    unit.MilestoneID,
		}); err != nil {
			return fmt.Errorf("update milestone phase: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown unit type %q", unit.Type)
	}
}

// SetMilestoneStatus directly sets a milestone's status to the given value.
// Used by park/unpark operations.
func (d *DBStatusAdvancer) SetMilestoneStatus(ctx context.Context, milestoneID, status string) error {
	_, err := d.q.UpdateMilestoneStatus(ctx, db.UpdateMilestoneStatusParams{
		Status: status,
		ID:     milestoneID,
	})
	return err
}

// RevertStatus undoes AdvanceStatus, reverting a unit to its pre-completion
// state. This is the inverse of AdvanceStatus:
//
//   - UnitExecuteTask   → task status = active, task phase = executing
//   - UnitSummarizeSlice → slice status = active, slice phase = executing
//   - UnitValidateMilestone → milestone status = active, milestone phase = executing
func (d *DBStatusAdvancer) RevertStatus(ctx context.Context, unit Unit) error {
	switch unit.Type {
	case UnitExecuteTask:
		if _, err := d.q.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
			Status: string(StatusActive),
			ID:     unit.TaskID,
		}); err != nil {
			return fmt.Errorf("revert task status: %w", err)
		}
		if _, err := d.q.UpdateTaskPhase(ctx, db.UpdateTaskPhaseParams{
			Phase: string(PhaseExecuting),
			ID:    unit.TaskID,
		}); err != nil {
			return fmt.Errorf("revert task phase: %w", err)
		}
		return nil

	case UnitSummarizeSlice:
		if _, err := d.q.UpdateSliceStatus(ctx, db.UpdateSliceStatusParams{
			Status: string(StatusActive),
			ID:     unit.SliceID,
		}); err != nil {
			return fmt.Errorf("revert slice status: %w", err)
		}
		if _, err := d.q.UpdateSlicePhase(ctx, db.UpdateSlicePhaseParams{
			Phase: string(PhaseExecuting),
			ID:    unit.SliceID,
		}); err != nil {
			return fmt.Errorf("revert slice phase: %w", err)
		}
		return nil

	case UnitValidateMilestone:
		if _, err := d.q.UpdateMilestoneStatus(ctx, db.UpdateMilestoneStatusParams{
			Status: string(StatusActive),
			ID:     unit.MilestoneID,
		}); err != nil {
			return fmt.Errorf("revert milestone status: %w", err)
		}
		if _, err := d.q.UpdateMilestonePhase(ctx, db.UpdateMilestonePhaseParams{
			Phase: string(PhaseExecuting),
			ID:    unit.MilestoneID,
		}); err != nil {
			return fmt.Errorf("revert milestone phase: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("cannot revert unit type %q", unit.Type)
	}
}

// ---------------------------------------------------------------------------
// SessionServiceCreator implements SessionCreator backed by session.Service.
// ---------------------------------------------------------------------------

// SessionServiceCreator wraps session.Service to satisfy the SessionCreator
// interface.
type SessionServiceCreator struct {
	svc session.Service
}

// NewSessionServiceCreator creates a SessionServiceCreator.
func NewSessionServiceCreator(svc session.Service) *SessionServiceCreator {
	return &SessionServiceCreator{svc: svc}
}

// CreateSession creates a top-level session and returns its ID.
func (s *SessionServiceCreator) CreateSession(ctx context.Context, title string) (string, error) {
	sess, err := s.svc.Create(ctx, title)
	if err != nil {
		return "", err
	}
	return sess.ID, nil
}

// CreateChildSession creates a child session under parentID and returns its
// ID. It delegates to CreateTaskSession which accepts an explicit ID.
func (s *SessionServiceCreator) CreateChildSession(ctx context.Context, id, parentID, title string) (string, error) {
	sess, err := s.svc.CreateTaskSession(ctx, id, parentID, title)
	if err != nil {
		return "", err
	}
	return sess.ID, nil
}

// ---------------------------------------------------------------------------
// DBTokenQuerier implements TokenQuerier backed by sqlc Queries.
// ---------------------------------------------------------------------------

// DBTokenQuerier wraps *db.Queries to satisfy the TokenQuerier interface.
type DBTokenQuerier struct {
	q *db.Queries
}

// NewDBTokenQuerier creates a DBTokenQuerier.
func NewDBTokenQuerier(q *db.Queries) *DBTokenQuerier {
	return &DBTokenQuerier{q: q}
}

// GetTokenUsage returns the cumulative prompt and completion token counts
// across all child sessions of the given parent session.
func (d *DBTokenQuerier) GetTokenUsage(ctx context.Context, sessionID string) (int64, int64, error) {
	row, err := d.q.GetSessionTokenUsage(ctx, sql.NullString{String: sessionID, Valid: true})
	if err != nil {
		return 0, 0, err
	}
	return row.TotalPromptTokens, row.TotalCompletionTokens, nil
}

// ---------------------------------------------------------------------------
// CoordinatorDispatcher implements Dispatcher backed by agent.Coordinator.
// ---------------------------------------------------------------------------

// CoordinatorDispatcher wraps agent.Coordinator to satisfy the Dispatcher
// interface. Per D015, it discards the *AgentResult and returns only the
// error.
type CoordinatorDispatcher struct {
	coord agent.Coordinator
}

// NewCoordinatorDispatcher creates a CoordinatorDispatcher.
func NewCoordinatorDispatcher(coord agent.Coordinator) *CoordinatorDispatcher {
	return &CoordinatorDispatcher{coord: coord}
}

// RunWithForcedTier dispatches a prompt in the given session with a forced
// model tier. The *AgentResult from the coordinator is discarded; only the
// error is propagated.
func (d *CoordinatorDispatcher) RunWithForcedTier(ctx context.Context, sessionID, prompt string, tier config.SelectedModelType) error {
	_, err := d.coord.RunWithForcedTier(ctx, sessionID, prompt, tier)
	return err
}
