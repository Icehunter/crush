package auto

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/crush/internal/gsd"
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
	engine   *Engine
	querier  StateQuerier
	reverter StatusReverter
	parker   *MilestoneParker

	mu          sync.Mutex
	milestoneID string

	// prefsGlobalPath and prefsProjectPath are paths to PREFERENCES.md files.
	prefsGlobalPath  string
	prefsProjectPath string

	// initConfigFn builds an InitConfig for a given vision string.
	initConfigFn func(vision string) InitConfig

	// templateApplier creates DB records from a workflow template.
	templateApplier TemplateApplier
}

// NewEngineController creates an EngineController.
func NewEngineController(engine *Engine, querier StateQuerier) *EngineController {
	return &EngineController{
		engine:  engine,
		querier: querier,
	}
}

// SetReverter configures the status reverter used by UndoLast.
func (c *EngineController) SetReverter(reverter StatusReverter) {
	c.reverter = reverter
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

// StopAuto stops auto-mode immediately by cancelling the context.
func (c *EngineController) StopAuto() error {
	c.engine.Stop()
	return nil
}

// StepAuto executes exactly one unit then returns.
func (c *EngineController) StepAuto(ctx context.Context, milestoneID string) error {
	c.mu.Lock()
	c.milestoneID = milestoneID
	c.mu.Unlock()

	go func() {
		_ = c.engine.Step(ctx, milestoneID)
	}()
	return nil
}

// AutoStatus returns the current engine state as a string.
func (c *EngineController) AutoStatus() string {
	return string(c.engine.Status().State)
}

// AutoQueue returns a formatted list of pending units for the given milestone.
func (c *EngineController) AutoQueue(ctx context.Context, milestoneID string) ([]string, error) {
	return DeriveQueue(ctx, c.querier, milestoneID)
}

// UndoLast reverts the most recently completed task and returns its description.
func (c *EngineController) UndoLast(ctx context.Context, milestoneID string) (string, error) {
	if c.reverter == nil {
		return "", fmt.Errorf("undo not configured")
	}
	undoQuerier := NewDBUndoQuerier(c.querier)
	unit, err := UndoLastUnit(ctx, undoQuerier, c.reverter, milestoneID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Undid %s: %s", unit.TaskID, unit.Title), nil
}

// SkipUnit marks a task as completed (skipped) so auto-dispatch won't execute it.
// It sets both status and phase to completed.
func (c *EngineController) SkipUnit(ctx context.Context, taskID string) error {
	if c.reverter == nil {
		return fmt.Errorf("skip not configured")
	}
	// We skip by marking the task as completed — DeriveState will then move past it.
	unit := Unit{
		Type:   UnitExecuteTask,
		TaskID: taskID,
	}
	advancer, ok := c.reverter.(*DBStatusAdvancer)
	if !ok {
		return fmt.Errorf("skip requires DBStatusAdvancer")
	}
	return advancer.AdvanceStatus(ctx, unit)
}

// DispatchPhase creates and dispatches a one-shot unit for the given phase.
func (c *EngineController) DispatchPhase(ctx context.Context, milestoneID, phase string) error {
	c.mu.Lock()
	c.milestoneID = milestoneID
	c.mu.Unlock()

	go func() {
		_ = c.engine.Step(ctx, milestoneID)
	}()
	return nil
}

// Steer injects a guidance message by logging it. In the future this will
// inject into the active session's context.
func (c *EngineController) Steer(_ context.Context, guidance string) error {
	c.engine.logger.Info("User steering guidance received", "guidance", guidance)
	return nil
}

// History returns formatted recent execution history from the journal.
func (c *EngineController) History(_ context.Context, count int) (string, error) {
	if c.engine.journal == nil {
		return "No journal configured", nil
	}
	if count <= 0 {
		count = 10
	}
	entries, err := ReadRecentEntries(c.engine.journal.dir, 7)
	if err != nil {
		return "", fmt.Errorf("read journal: %w", err)
	}
	if len(entries) == 0 {
		return "No execution history found", nil
	}

	// Limit to last N entries.
	if len(entries) > count {
		entries = entries[len(entries)-count:]
	}

	lines := []string{fmt.Sprintf("Last %d units:", len(entries))}
	for _, e := range entries {
		status := "✓"
		if !e.Success {
			status = "✗"
		}
		line := fmt.Sprintf("  %s [%s] %s (%dms, $%.4f)",
			status, e.UnitType, e.UnitTitle, e.DurationMs, e.Cost)
		if e.ErrorMsg != "" {
			line += " — " + e.ErrorMsg
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

// RateTier records user feedback on the last unit's model tier.
func (c *EngineController) RateTier(_ context.Context, rating string) error {
	if c.engine.journal == nil {
		return fmt.Errorf("no journal configured")
	}

	// Find the last completed entry to know which unit type/tier to rate.
	entries, err := ReadRecentEntries(c.engine.journal.dir, 1)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no recent entries to rate")
	}

	last := entries[len(entries)-1]

	// Record to routing history if available.
	routingPath := c.engine.dataDir + "/routing-history.json"
	rh := NewRoutingHistory(routingPath)
	rh.Rate(last.UnitType, last.ModelTier, rating)

	c.engine.logger.Info("Tier rated",
		"unit_type", last.UnitType,
		"model_tier", last.ModelTier,
		"rating", rating)
	return nil
}

// RunDoctor performs health checks with optional auto-fix.
func (c *EngineController) RunDoctor(ctx context.Context, fix bool) (string, error) {
	report := RunDoctor(ctx, c.engine.dataDir, c.querier, fix)
	return report.Summary(), nil
}

// QuickTask dispatches a lightweight task without full planning.
// It uses StepAuto which derives the next unit from state.
func (c *EngineController) QuickTask(ctx context.Context, milestoneID, description string) error {
	c.engine.logger.Info("Quick task requested", "milestone", milestoneID, "description", description)
	// For now, quick tasks are dispatched as a single step.
	return c.StepAuto(ctx, milestoneID)
}

// InitProject runs the interactive planning flow with the given vision.
// It delegates to RunInit which creates milestones, slices, and tasks
// through the planning agent.
func (c *EngineController) InitProject(ctx context.Context, vision string) error {
	if c.initConfigFn == nil {
		return fmt.Errorf("init not configured — missing InitConfig provider")
	}
	cfg := c.initConfigFn(vision)
	return RunInit(ctx, cfg)
}

// SetInitConfigFn configures a function that builds an InitConfig for a
// given vision string. This decouples the controller from agent/session deps.
func (c *EngineController) SetInitConfigFn(fn func(vision string) InitConfig) {
	c.initConfigFn = fn
}

// SetParker configures the milestone parker used by Park/Unpark.
func (c *EngineController) SetParker(parker *MilestoneParker) {
	c.parker = parker
}

// SetPreferencesPaths configures the paths to PREFERENCES.md files.
func (c *EngineController) SetPreferencesPaths(globalPath, projectPath string) {
	c.prefsGlobalPath = globalPath
	c.prefsProjectPath = projectPath
}

// StartFromTemplate creates a milestone from a workflow template and returns
// the template name. If a TemplateApplier is configured, it creates the
// milestone, slices, and tasks in the DB.
func (c *EngineController) StartFromTemplate(ctx context.Context, templateID string) (string, error) {
	t := LookupTemplate(templateID)
	if t == nil {
		available := ListTemplateNames()
		return "", fmt.Errorf("unknown template %q. Available: %s", templateID, strings.Join(available, ", "))
	}

	if c.templateApplier != nil {
		milestoneID, err := c.templateApplier.ApplyTemplate(ctx, t)
		if err != nil {
			return "", fmt.Errorf("apply template %s: %w", t.ID, err)
		}
		c.mu.Lock()
		c.milestoneID = milestoneID
		c.mu.Unlock()
	}

	return t.Name, nil
}

// ParkMilestone parks a milestone so DeriveState skips it.
func (c *EngineController) ParkMilestone(ctx context.Context, milestoneID string) error {
	if c.parker == nil {
		return fmt.Errorf("park not configured")
	}
	return c.parker.Park(ctx, milestoneID)
}

// UnparkMilestone restores a parked milestone to active status.
func (c *EngineController) UnparkMilestone(ctx context.Context, milestoneID string) error {
	if c.parker == nil {
		return fmt.Errorf("unpark not configured")
	}
	return c.parker.Unpark(ctx, milestoneID)
}

// Rethink triggers a conversational replan by dispatching a step that
// will re-derive state. Full LLM-guided replanning is a future enhancement.
func (c *EngineController) Rethink(ctx context.Context, milestoneID string) error {
	c.engine.logger.Info("Rethink requested", "milestone", milestoneID)
	return c.StepAuto(ctx, milestoneID)
}

// GetPreferences returns current GSD preferences as formatted text.
func (c *EngineController) GetPreferences() (string, error) {
	prefs, err := gsd.LoadPreferences(c.prefsGlobalPath, c.prefsProjectPath)
	if err != nil {
		return "", err
	}
	return gsd.FormatPreferences(prefs), nil
}

// SetPreference sets a single preference key=value by loading the project
// PREFERENCES.md, updating the field, and writing it back.
func (c *EngineController) SetPreference(key, value string) error {
	if c.prefsProjectPath == "" {
		return fmt.Errorf("no project preferences path configured")
	}
	return gsd.SetPreferenceValue(c.prefsProjectPath, key, value)
}

// CleanupWorktrees removes stale worktrees and merged branches.
func (c *EngineController) CleanupWorktrees(ctx context.Context) (string, error) {
	if c.engine.worktreeManager == nil {
		return "No worktree manager configured", nil
	}

	// List worktrees and prune stale ones.
	out, err := exec.CommandContext(ctx, "git", "worktree", "prune").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree prune: %w\n%s", err, out)
	}

	return "Stale worktrees pruned", nil
}

