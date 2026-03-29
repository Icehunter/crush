package auto

import (
	"context"
	"sync"
)

// BuildSnapshot queries DB state to produce a point-in-time AutoSnapshot.
// It iterates slices for the given milestone and counts tasks per slice.
func BuildSnapshot(
	ctx context.Context,
	querier StateQuerier,
	milestoneID string,
	status string,
	activeUnit string,
	totalCost float64,
	elapsed float64,
) *AutoSnapshot {
	snap := &AutoSnapshot{
		MilestoneID:    milestoneID,
		MilestoneTitle: milestoneID,
		ActiveUnit:     activeUnit,
		TotalCost:      totalCost,
		ElapsedSeconds: elapsed,
		Status:         status,
	}

	slices, err := querier.ListSlicesByMilestone(ctx, milestoneID)
	if err != nil {
		return snap
	}

	for _, s := range slices {
		sp := SliceProgress{
			ID:     s.ID,
			Title:  s.Title,
			Status: s.Status,
		}

		tasks, err := querier.ListTasksBySlice(ctx, s.ID)
		if err == nil {
			sp.TasksTotal = len(tasks)
			for _, t := range tasks {
				if Status(t.Status) == StatusCompleted {
					sp.TasksDone++
				}
			}
		}

		snap.Slices = append(snap.Slices, sp)
	}

	// Use the first slice's milestone context for a title hint. For a
	// proper title we'd need a GetMilestone query, but slices carry enough
	// context. The ID is a reasonable fallback already set above.

	return snap
}

// EngineController adapts Engine to the model.AutoController interface used
// by the TUI layer.
type EngineController struct {
	engine  *Engine
	querier StateQuerier

	mu          sync.Mutex
	milestoneID string
}

// NewEngineController creates an EngineController.
func NewEngineController(engine *Engine, querier StateQuerier) *EngineController {
	return &EngineController{
		engine:  engine,
		querier: querier,
	}
}

// StartAuto begins auto-mode execution for the given milestone.
func (c *EngineController) StartAuto(ctx context.Context, milestoneID string) error {
	c.mu.Lock()
	c.milestoneID = milestoneID
	c.mu.Unlock()

	go func() {
		_ = c.engine.Run(ctx, milestoneID)
	}()
	return nil
}

// PauseAuto pauses auto-mode after the current unit completes.
func (c *EngineController) PauseAuto() error {
	c.engine.Pause()
	return nil
}

// ResumeAuto resumes a paused auto-mode session by re-running the engine.
func (c *EngineController) ResumeAuto(ctx context.Context) error {
	c.mu.Lock()
	mid := c.milestoneID
	c.mu.Unlock()

	go func() {
		_ = c.engine.Run(ctx, mid)
	}()
	return nil
}

// AutoStatus returns the current engine state as a string.
func (c *EngineController) AutoStatus() string {
	return string(c.engine.Status().State)
}
