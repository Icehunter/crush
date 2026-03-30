package auto

import (
	"context"
	"fmt"
	"strings"
)

// MilestoneRow is the minimal projection DeriveState needs from a milestone
// record. Defined here so the auto package owns its query interface.
type MilestoneRow struct {
	ID     string
	Title  string
	Status string
	Phase  string
}

// SliceRow is the minimal projection DeriveState needs from a slice record.
type SliceRow struct {
	ID        string
	Title     string
	Status    string
	Phase     string
	SortOrder int64
	DependsOn string // Comma-separated slice IDs, or empty.
}

// TaskRow is the minimal projection DeriveState needs from a task record.
type TaskRow struct {
	ID        string
	Title     string
	Status    string
	Phase     string
	SortOrder int64
}

// StateQuerier is the query surface DeriveState depends on. The real
// implementation wraps db.Queries; tests supply an in-memory fake.
type StateQuerier interface {
	ListMilestones(ctx context.Context) ([]MilestoneRow, error)
	ListSlicesByMilestone(ctx context.Context, milestoneID string) ([]SliceRow, error)
	ListTasksBySlice(ctx context.Context, sliceID string) ([]TaskRow, error)
}

// PhaseSkipConfig controls which phases are automatically skipped
// during state derivation.
type PhaseSkipConfig struct {
	SkipResearch      bool
	SkipSliceResearch bool
}

// DeriveState examines the current DB state and returns the next Unit the
// engine should dispatch. It returns a zero-value Unit when all work is
// complete or when remaining work is blocked on unmet dependencies.
func DeriveState(ctx context.Context, q StateQuerier) (Unit, error) {
	return DeriveStateWithSkips(ctx, q, PhaseSkipConfig{})
}

// DeriveStateWithSkips is like DeriveState but respects phase-skip preferences.
func DeriveStateWithSkips(ctx context.Context, q StateQuerier, skips PhaseSkipConfig) (Unit, error) {
	milestones, err := q.ListMilestones(ctx)
	if err != nil {
		return Unit{}, fmt.Errorf("list milestones: %w", err)
	}

	// Find the first active milestone, skipping parked ones.
	var active *MilestoneRow
	for i := range milestones {
		if Status(milestones[i].Status) == StatusParked {
			continue
		}
		if Status(milestones[i].Status) == StatusActive {
			active = &milestones[i]
			break
		}
	}
	if active == nil {
		return Unit{}, nil // Nothing active → done.
	}

	slices, err := q.ListSlicesByMilestone(ctx, active.ID)
	if err != nil {
		return Unit{}, fmt.Errorf("list slices for milestone %s: %w", active.ID, err)
	}

	// Build a set of completed slice IDs for dependency checking.
	completed := make(map[string]bool, len(slices))
	for _, s := range slices {
		if Status(s.Status) == StatusCompleted {
			completed[s.ID] = true
		}
	}

	// Walk slices in sort_order. Return the first dispatchable unit.
	for _, s := range slices {
		if Status(s.Status) == StatusCompleted {
			continue
		}

		// Check dependency constraint.
		if !depsMetFor(s.DependsOn, completed) {
			continue // Blocked — skip to the next slice.
		}

		unit, err := unitForSlice(ctx, q, active, &s, skips)
		if err != nil {
			return Unit{}, err
		}
		if !unit.IsDone() {
			return unit, nil
		}
		// Slice returned done (all tasks complete but slice not yet
		// marked completed) — fall through to next slice.
	}

	// If the active milestone has slices and all are completed, derive
	// a validate_milestone unit if the milestone phase warrants it.
	if allSlicesCompleted(slices) && len(slices) > 0 {
		p := Phase(active.Phase)
		if p != PhaseCompleted && p != PhaseValidating {
			return Unit{
				Type:        UnitValidateMilestone,
				MilestoneID: active.ID,
				Title:       fmt.Sprintf("Validate milestone %s", active.Title),
			}, nil
		}
	}

	return Unit{}, nil // All done or fully blocked.
}

// unitForSlice returns the next unit for a non-completed slice whose
// dependencies are met. The skips config controls which phases are
// automatically skipped.
func unitForSlice(ctx context.Context, q StateQuerier, m *MilestoneRow, s *SliceRow, skips PhaseSkipConfig) (Unit, error) {
	phase := Phase(s.Phase)

	switch phase {
	case PhasePrePlanning, PhaseResearching:
		// If research is skipped, jump directly to planning.
		if skips.SkipResearch || skips.SkipSliceResearch {
			return Unit{
				Type:        UnitPlanSlice,
				MilestoneID: m.ID,
				SliceID:     s.ID,
				Title:       fmt.Sprintf("Plan slice %s (research skipped)", s.Title),
			}, nil
		}
		return Unit{
			Type:        UnitResearch,
			MilestoneID: m.ID,
			SliceID:     s.ID,
			Title:       fmt.Sprintf("Research slice %s", s.Title),
		}, nil

	case PhasePlanning:
		return Unit{
			Type:        UnitPlanSlice,
			MilestoneID: m.ID,
			SliceID:     s.ID,
			Title:       fmt.Sprintf("Plan slice %s", s.Title),
		}, nil

	case PhaseExecuting:
		tasks, err := q.ListTasksBySlice(ctx, s.ID)
		if err != nil {
			return Unit{}, fmt.Errorf("list tasks for slice %s: %w", s.ID, err)
		}
		for _, t := range tasks {
			if Status(t.Status) != StatusCompleted {
				return Unit{
					Type:        UnitExecuteTask,
					MilestoneID: m.ID,
					SliceID:     s.ID,
					TaskID:      t.ID,
					Title:       fmt.Sprintf("Execute task %s", t.Title),
				}, nil
			}
		}
		// All tasks done but slice still in executing phase — signal
		// that the slice needs summarizing. Return a summarize unit.
		return Unit{
			Type:        UnitSummarizeSlice,
			MilestoneID: m.ID,
			SliceID:     s.ID,
			Title:       fmt.Sprintf("Summarize slice %s", s.Title),
		}, nil

	case PhaseSummarizing:
		return Unit{
			Type:        UnitSummarizeSlice,
			MilestoneID: m.ID,
			SliceID:     s.ID,
			Title:       fmt.Sprintf("Summarize slice %s", s.Title),
		}, nil

	case PhaseValidating:
		return Unit{
			Type:        UnitValidateMilestone,
			MilestoneID: m.ID,
			Title:       fmt.Sprintf("Validate milestone %s (via slice %s)", m.Title, s.ID),
		}, nil

	case PhaseCompleted:
		return Unit{}, nil // Slice complete, nothing to dispatch.

	default:
		return Unit{}, fmt.Errorf("unknown phase %q for slice %s", s.Phase, s.ID)
	}
}

// depsMetFor checks whether all dependencies in a comma-separated list are
// present in the completed set. An empty depends string means no
// dependencies, so it always returns true.
func depsMetFor(depends string, completed map[string]bool) bool {
	if depends == "" {
		return true
	}
	for _, dep := range strings.Split(depends, ",") {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		if !completed[dep] {
			return false
		}
	}
	return true
}

// allSlicesCompleted returns true when every slice in the list has status
// completed.
func allSlicesCompleted(slices []SliceRow) bool {
	for _, s := range slices {
		if Status(s.Status) != StatusCompleted {
			return false
		}
	}
	return true
}
