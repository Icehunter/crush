package auto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/google/uuid"
)

// SessionCreator abstracts session creation so the engine is testable without
// a real database.
type SessionCreator interface {
	// CreateSession creates a top-level session and returns its ID.
	CreateSession(ctx context.Context, title string) (string, error)
	// CreateChildSession creates a child session under parentID and returns
	// its ID.
	CreateChildSession(ctx context.Context, id, parentID, title string) (string, error)
}

// Dispatcher abstracts the agent coordinator. It runs a prompt in a session
// with a forced model tier and returns when the agent finishes.
type Dispatcher interface {
	RunWithForcedTier(ctx context.Context, sessionID, prompt string, tier config.SelectedModelType) error
}

// StatusAdvancer abstracts the DB writes that advance milestone/slice/task
// status after a unit completes. This keeps the engine decoupled from sqlc.
type StatusAdvancer interface {
	AdvanceStatus(ctx context.Context, unit Unit) error
}

// EngineState represents the engine's current operational state.
type EngineState string

const (
	EngineIdle    EngineState = "idle"
	EngineRunning EngineState = "running"
	EnginePaused  EngineState = "paused"
)

// EngineStatus is a snapshot of the engine's current state.
type EngineStatus struct {
	State       EngineState `json:"state"`
	MilestoneID string      `json:"milestone_id,omitempty"`
	ActiveUnit  *Unit       `json:"active_unit,omitempty"`
	LastError   string      `json:"last_error,omitempty"`
}

// Engine drives the auto-mode loop: derive state → create session →
// dispatch → advance → publish events. It is safe for concurrent use
// but only one Run() may be active at a time (enforced by LockFile).
type Engine struct {
	querier        StateQuerier
	sessions       SessionCreator
	dispatch       Dispatcher
	advancer       StatusAdvancer
	verifier       Verifier
	budgetChecker  BudgetChecker
	budgetCeiling  float64
	stuckDetector  *StuckDetector
	contextMonitor *ContextMonitor
	broker         *pubsub.Broker[AutoEvent]
	dataDir        string
	logger         *slog.Logger

	// Mutable state guarded by mu.
	mu          sync.Mutex
	state       EngineState
	milestoneID string
	activeUnit  *Unit
	lastErr     string
	cancel      context.CancelFunc

	// Pause is signaled by setting this flag; the loop checks it
	// between iterations.
	paused atomic.Bool

	// snapshotQuerier is used by publish() to build an AutoSnapshot
	// attached to events. Nil-safe — when nil, no snapshot is attached.
	snapshotQuerier StateQuerier

	// worktreeManager manages git worktree lifecycle per milestone.
	// Nil-safe — when nil, no worktree isolation is used.
	worktreeManager *WorktreeManager
	worktreeMode    string

	// autoPush controls whether completed milestones are pushed to remote.
	autoPush bool
	// pushRemote is the git remote to push to (default: "origin").
	pushRemote string

	// phaseSkips controls which phases are automatically skipped.
	phaseSkips PhaseSkipConfig
	// budgetEnforcement controls what happens when budget is exceeded:
	// "warn" (log and continue), "pause" (default), "halt" (non-recoverable).
	budgetEnforcement string

	// retryState tracks exponential backoff across consecutive failures.
	retryState *RetryState

	// journal records unit dispatch outcomes to daily JSONL files.
	// Nil-safe — when nil, no journaling occurs.
	journal *Journal

	// metrics records unit dispatch metrics for aggregation.
	// Nil-safe — when nil, no metrics are recorded.
	metrics *MetricsLedger

	// runStart tracks when the current Run() began, for elapsed time.
	runStart time.Time
}

// NewEngine creates an engine wired to the given dependencies.
func NewEngine(
	querier StateQuerier,
	sessions SessionCreator,
	dispatch Dispatcher,
	advancer StatusAdvancer,
	verifier Verifier,
	budgetChecker BudgetChecker,
	budgetCeiling float64,
	stuckDetector *StuckDetector,
	contextMonitor *ContextMonitor,
	broker *pubsub.Broker[AutoEvent],
	dataDir string,
	logger *slog.Logger,
	snapshotQuerier StateQuerier,
) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		querier:         querier,
		sessions:        sessions,
		dispatch:        dispatch,
		advancer:        advancer,
		verifier:        verifier,
		budgetChecker:   budgetChecker,
		budgetCeiling:   budgetCeiling,
		stuckDetector:   stuckDetector,
		contextMonitor:  contextMonitor,
		broker:          broker,
		dataDir:         dataDir,
		logger:          logger,
		state:           EngineIdle,
		snapshotQuerier: snapshotQuerier,
		retryState:      NewRetryState(),
	}
}

// SetWorktreeManager configures worktree isolation for the engine. Call
// this after NewEngine to avoid adding parameters to the constructor
// (see K012). mode should be "per-milestone" to enable lifecycle
// management; any other value (or nil wm) disables worktree isolation.
func (e *Engine) SetWorktreeManager(wm *WorktreeManager, mode string) {
	e.worktreeManager = wm
	e.worktreeMode = mode
}

// SetJournalAndMetrics configures journal and metrics recording. Call
// after NewEngine.
func (e *Engine) SetJournalAndMetrics(journal *Journal, metrics *MetricsLedger) {
	e.journal = journal
	e.metrics = metrics
}

// SetPushConfig configures automatic git push on milestone completion.
func (e *Engine) SetPushConfig(autoPush bool, remote string) {
	e.autoPush = autoPush
	e.pushRemote = remote
}

// SetPhaseSkips configures which phases are automatically skipped.
func (e *Engine) SetPhaseSkips(skips PhaseSkipConfig) {
	e.phaseSkips = skips
}

// SetBudgetEnforcement configures the budget enforcement mode.
// Valid values: "warn", "pause" (default), "halt".
func (e *Engine) SetBudgetEnforcement(mode string) {
	e.budgetEnforcement = mode
}

// Run acquires the lock file and enters the loop until all work is done,
// the context is cancelled, or Pause/Stop is called.
func (e *Engine) Run(ctx context.Context, milestoneID string) error {
	lock := NewLockFile(e.dataDir)
	if err := lock.Acquire(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			e.logger.Error("Failed to release lock", "error", err)
		}
	}()

	// Check for crash recovery before starting.
	recoveryInfo, recoveryErr := RecoverFromCrash(ctx, lock.Path(), e.querier)
	if recoveryErr != nil {
		e.logger.Error("Crash recovery check failed", "error", recoveryErr)
	} else if recoveryInfo.Action != RecoveryNone {
		e.logger.Info("Crash recovery detected",
			"action", recoveryInfo.Action,
			"crashed_at", recoveryInfo.CrashedAt,
			"unit_completed", recoveryInfo.UnitCompleted)
		e.publish(EventCrashRecovery, recoveryInfo.CrashedUnit, nil,
			fmt.Sprintf("Recovered from crash: %s (unit completed: %v)", recoveryInfo.Action, recoveryInfo.UnitCompleted))
	}

	ctx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.state = EngineRunning
	e.milestoneID = milestoneID
	e.cancel = cancel
	e.paused.Store(false)
	e.mu.Unlock()

	defer func() {
		cancel()
		e.mu.Lock()
		if e.paused.Load() {
			e.state = EnginePaused
		} else {
			e.state = EngineIdle
		}
		e.activeUnit = nil
		e.cancel = nil
		e.mu.Unlock()
	}()

	// Reset retry state and start timer for a fresh run.
	e.retryState.RecordSuccess()
	e.runStart = time.Now()

	// Create a parent session for this milestone run.
	parentSessionID, err := e.sessions.CreateSession(ctx, fmt.Sprintf("auto: %s", milestoneID))
	if err != nil {
		return fmt.Errorf("create parent session: %w", err)
	}

	// Worktree lifecycle: ensure a worktree exists before the main loop.
	if e.worktreeManager != nil && e.worktreeMode == "per-milestone" {
		if e.worktreeManager.Exists(milestoneID) {
			e.logger.Info("Resuming existing worktree", "milestone", milestoneID)
		} else {
			if wtErr := e.worktreeManager.Create(ctx, milestoneID); wtErr != nil {
				return fmt.Errorf("create worktree for %s: %w", milestoneID, wtErr)
			}
		}
	}

	for {
		if e.paused.Load() {
			e.publish(EventLoopPaused, Unit{}, nil, "Loop paused by user")
			return nil
		}

		if ctx.Err() != nil {
			e.publish(EventLoopStopped, Unit{}, nil, "Loop stopped")
			return ctx.Err()
		}

		err := e.step(ctx, milestoneID, parentSessionID)
		if err != nil {
			if errors.Is(err, errDone) {
				// All units complete — merge and clean up worktree.
				e.cleanupWorktree(ctx, milestoneID)
				return nil
			}

			e.mu.Lock()
			e.lastErr = err.Error()
			e.mu.Unlock()

			// Classify the error and determine retry strategy.
			errClass := ClassifyError(err)
			delay, retryable := e.retryState.RecordFailure(err)

			e.publish(EventProviderError, Unit{}, err,
				fmt.Sprintf("Error class: %s, retryable: %v, delay: %s, attempt: %d",
					errClass, retryable, delay, e.retryState.Consecutives()))

			if !retryable {
				// Permanent error or retries exhausted — pause engine.
				e.logger.Error("Non-retryable error, pausing engine",
					"error", err, "class", errClass,
					"consecutive_failures", e.retryState.Consecutives())
				e.paused.Store(true)
				e.mu.Lock()
				e.state = EnginePaused
				e.mu.Unlock()
				return nil
			}

			e.logger.Error("Unit dispatch failed, retrying with backoff",
				"error", err, "class", errClass,
				"delay", delay, "attempt", e.retryState.Consecutives())

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		} else {
			// Success — reset retry state.
			e.retryState.RecordSuccess()
		}
	}
}

// errDone is a sentinel used internally to signal the loop that all units
// are complete.
var errDone = errors.New("all units done")

// Step runs exactly one unit then returns.
func (e *Engine) Step(ctx context.Context, milestoneID string) error {
	lock := NewLockFile(e.dataDir)
	if err := lock.Acquire(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	parentSessionID, err := e.sessions.CreateSession(ctx, fmt.Sprintf("auto-step: %s", milestoneID))
	if err != nil {
		return fmt.Errorf("create parent session: %w", err)
	}

	err = e.step(ctx, milestoneID, parentSessionID)
	if errors.Is(err, errDone) {
		return nil
	}
	return err
}

// step derives the next unit, dispatches it, and advances status. Returns
// errDone when no work remains.
func (e *Engine) step(ctx context.Context, milestoneID, parentSessionID string) error {
	unit, err := DeriveStateWithSkips(ctx, e.querier, e.phaseSkips)
	if err != nil {
		return fmt.Errorf("derive state: %w", err)
	}
	if unit.IsDone() {
		return errDone
	}

	// Budget gate: check cumulative child session costs before dispatch.
	if e.budgetCeiling > 0 && e.budgetChecker != nil {
		totalCost, err := e.budgetChecker.CheckBudget(ctx, parentSessionID)
		if err != nil {
			return fmt.Errorf("check budget: %w", err)
		}
		if totalCost >= e.budgetCeiling {
			msg := fmt.Sprintf("total cost $%.4f >= ceiling $%.4f", totalCost, e.budgetCeiling)
			e.publish(EventBudgetExceeded, unit, nil, msg)

			switch e.budgetEnforcement {
			case "warn":
				// Log and continue — do not pause.
				e.logger.Warn("Budget ceiling reached (warn mode)", "total_cost", totalCost, "ceiling", e.budgetCeiling)
			case "halt":
				// Non-recoverable stop.
				e.logger.Error("Budget ceiling reached (halt mode)", "total_cost", totalCost, "ceiling", e.budgetCeiling)
				e.mu.Lock()
				e.state = EnginePaused
				e.mu.Unlock()
				e.paused.Store(true)
				return fmt.Errorf("budget ceiling exceeded: %s", msg)
			default:
				// "pause" mode (default) — pause engine, user can resume.
				e.logger.Info("Budget ceiling reached (pause mode)", "total_cost", totalCost, "ceiling", e.budgetCeiling)
				e.mu.Lock()
				e.state = EnginePaused
				e.mu.Unlock()
				e.paused.Store(true)
				return nil
			}
		}
	}

	// Track active unit.
	e.mu.Lock()
	e.activeUnit = &unit
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.activeUnit = nil
		e.mu.Unlock()
	}()

	// Create child session for this unit.
	childID := uuid.New().String()
	childTitle := fmt.Sprintf("auto: %s", unit.Title)
	sessionID, err := e.sessions.CreateChildSession(ctx, childID, parentSessionID, childTitle)
	if err != nil {
		e.publish(EventUnitFailed, unit, err, "Failed to create child session")
		return fmt.Errorf("create child session: %w", err)
	}

	// Select model tier based on unit type.
	tier := tierForUnit(unit.Type)

	// Update lock file with active unit for crash recovery.
	lockPath := filepath.Join(e.dataDir, lockFileName)
	if updateErr := UpdateLockUnit(lockPath, unit); updateErr != nil {
		e.logger.Warn("Failed to update lock with unit info", "error", updateErr)
	}

	e.publish(EventUnitStarted, unit, nil, "")
	e.logger.Info("Dispatching unit", "unit", unit.String(), "session", sessionID, "tier", tier)

	// Build prior summaries from journal if available.
	var priorSummaries string
	if e.journal != nil {
		entries, readErr := ReadRecentEntries(e.journal.dir, 3)
		if readErr == nil {
			priorSummaries = BuildPriorSummaries(entries)
		}
	}

	// Build the system prompt from templates.
	promptCtx := PromptContext{
		MilestoneID:    unit.MilestoneID,
		MilestoneTitle: unit.Title,
		SliceID:        unit.SliceID,
		TaskID:         unit.TaskID,
		PriorSummaries: priorSummaries,
		WorkingDir:     e.dataDir,
	}
	prompt, promptErr := BuildPrompt(unit.Type, promptCtx)
	if promptErr != nil {
		e.publish(EventUnitFailed, unit, promptErr, "Failed to build prompt")
		return fmt.Errorf("build prompt for %s: %w", unit.String(), promptErr)
	}

	unitKey := UnitKey(unit)

	// Stuck gate: check if this unit is stuck before dispatching.
	if e.stuckDetector != nil && e.stuckDetector.IsStuck(unitKey) {
		e.logger.Warn("Unit stuck, retrying with diagnostic", "unit", unit.String())
		diagPrompt := fmt.Sprintf(
			"**STUCK DETECTION — DIAGNOSTIC RETRY**\n\n"+
				"This unit has failed repeatedly (>50%% of recent attempts). "+
				"Investigate the root cause before retrying.\n\n"+
				"Unit: %s", unit.String())
		if retryErr := e.dispatch.RunWithForcedTier(ctx, sessionID, diagPrompt, tier); retryErr != nil {
			// Diagnostic retry dispatch itself failed — record and pause.
			e.stuckDetector.Record(unitKey, false)
			e.logger.Error("Stuck diagnostic retry dispatch failed", "unit", unit.String(), "error", retryErr)
			e.publish(EventStuckDetected, unit, retryErr, "Stuck: diagnostic retry dispatch failed")
			e.mu.Lock()
			e.state = EnginePaused
			e.mu.Unlock()
			e.paused.Store(true)
			return nil
		}

		// Check if the retry resolved the situation by running verification.
		if unit.Type == UnitExecuteTask && e.verifier != nil {
			results, verifyErr := e.verifier.RunVerification(ctx, e.dataDir)
			if verifyErr != nil || !allPassed(results) {
				// Still stuck after retry — pause.
				e.stuckDetector.Record(unitKey, false)
				e.logger.Error("Unit still stuck after diagnostic retry", "unit", unit.String())
				e.publish(EventStuckDetected, unit, nil, "Stuck: still failing after diagnostic retry")
				e.mu.Lock()
				e.state = EnginePaused
				e.mu.Unlock()
				e.paused.Store(true)
				return nil
			}
			// Diagnostic retry succeeded.
			e.stuckDetector.Record(unitKey, true)
		} else {
			// Non-task units: if dispatch succeeded, record pass.
			e.stuckDetector.Record(unitKey, true)
		}

		// Advance and return — the diagnostic retry handled this step.
		if advErr := e.advancer.AdvanceStatus(ctx, unit); advErr != nil {
			e.publish(EventUnitFailed, unit, advErr, "Status advance failed")
			return fmt.Errorf("advance status for %s: %w", unit.String(), advErr)
		}
		e.publish(EventUnitCompleted, unit, nil, "Recovered from stuck via diagnostic retry")
		e.logger.Info("Unit recovered from stuck", "unit", unit.String())
		return nil
	}

	dispatchStart := time.Now()

	if dispatchErr := e.dispatch.RunWithForcedTier(ctx, sessionID, prompt, tier); dispatchErr != nil {
		dispatchDuration := time.Since(dispatchStart)
		e.publish(EventUnitFailed, unit, dispatchErr, "Dispatch failed")
		e.mu.Lock()
		e.lastErr = dispatchErr.Error()
		e.mu.Unlock()
		if e.stuckDetector != nil {
			e.stuckDetector.Record(unitKey, false)
		}
		e.recordUnitOutcome(unit, false, dispatchDuration, dispatchErr, string(tier), sessionID)
		return fmt.Errorf("dispatch unit %s: %w", unit.String(), dispatchErr)
	}

	// Run verification gate for task execution units only.
	if unit.Type == UnitExecuteTask && e.verifier != nil {
		if err := e.runVerificationGate(ctx, unit, sessionID, tier); err != nil {
			dispatchDuration := time.Since(dispatchStart)
			if e.stuckDetector != nil {
				e.stuckDetector.Record(unitKey, false)
			}
			e.recordUnitOutcome(unit, false, dispatchDuration, err, string(tier), sessionID)
			return err
		}
	}

	dispatchDuration := time.Since(dispatchStart)

	// Record success in stuck detector.
	if e.stuckDetector != nil {
		e.stuckDetector.Record(unitKey, true)
	}

	// Context pressure gate: check cumulative token usage after dispatch.
	if e.contextMonitor != nil {
		exceeded, cpErr := e.contextMonitor.Check(ctx, parentSessionID)
		if cpErr != nil {
			e.logger.Error("Context pressure check failed", "error", cpErr)
		} else if exceeded {
			e.logger.Info("Context pressure threshold reached")
			e.publish(EventContextPressure, unit, nil, "Context pressure threshold exceeded")
			e.mu.Lock()
			e.state = EnginePaused
			e.mu.Unlock()
			e.paused.Store(true)
			return nil
		}
	}

	// Advance status in DB.
	if advErr := e.advancer.AdvanceStatus(ctx, unit); advErr != nil {
		e.publish(EventUnitFailed, unit, advErr, "Status advance failed")
		return fmt.Errorf("advance status for %s: %w", unit.String(), advErr)
	}

	e.recordUnitOutcome(unit, true, dispatchDuration, nil, string(tier), sessionID)
	e.publish(EventUnitCompleted, unit, nil, "")
	e.logger.Info("Unit completed", "unit", unit.String())
	return nil
}

// recordUnitOutcome writes a journal entry and metrics record for a
// completed (or failed) unit dispatch.
func (e *Engine) recordUnitOutcome(unit Unit, success bool, duration time.Duration, dispatchErr error, modelTier, sessionID string) {
	durationMs := duration.Milliseconds()

	if e.journal != nil {
		entry := JournalEntry{
			MilestoneID: unit.MilestoneID,
			SliceID:     unit.SliceID,
			TaskID:      unit.TaskID,
			UnitType:    string(unit.Type),
			UnitTitle:   unit.Title,
			Success:     success,
			DurationMs:  durationMs,
			ModelTier:   modelTier,
			SessionID:   sessionID,
		}
		if dispatchErr != nil {
			entry.ErrorMsg = dispatchErr.Error()
			entry.ErrorClass = ClassifyError(dispatchErr).String()
		}
		e.journal.Record(entry)
	}

	if e.metrics != nil {
		e.metrics.Record(UnitMetrics{
			MilestoneID: unit.MilestoneID,
			UnitType:    string(unit.Type),
			Success:     success,
			DurationMs:  durationMs,
			ModelTier:   modelTier,
		})
	}
}

// Pause signals the engine to stop after the current unit finishes.
func (e *Engine) Pause() {
	e.paused.Store(true)
}

// Stop cancels the engine context immediately.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

// Status returns a snapshot of the engine's current state.
func (e *Engine) Status() EngineStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	return EngineStatus{
		State:       e.state,
		MilestoneID: e.milestoneID,
		ActiveUnit:  e.activeUnit,
		LastError:   e.lastErr,
	}
}

// publish sends an AutoEvent through the broker. When snapshotQuerier is
// set, it builds and attaches an AutoSnapshot to the event.
func (e *Engine) publish(eventType pubsub.EventType, unit Unit, err error, message string) {
	event := NewAutoEvent(unit, err, message)

	if e.snapshotQuerier != nil {
		status := string(e.state)
		activeUnit := unit.String()

		var totalCost float64
		if e.metrics != nil {
			totalCost = e.metrics.TotalCost(e.milestoneID)
		}
		var elapsed float64
		if !e.runStart.IsZero() {
			elapsed = time.Since(e.runStart).Seconds()
		}

		snap := BuildSnapshot(
			context.Background(),
			e.snapshotQuerier,
			e.milestoneID,
			status,
			activeUnit,
			totalCost,
			elapsed,
		)
		event.Snapshot = snap
	}

	e.broker.Publish(eventType, event)
}

// runVerificationGate runs verification commands after a task dispatch. On
// failure it re-dispatches with a diagnostic prompt. If the retry also fails
// it returns an error without advancing status.
func (e *Engine) runVerificationGate(ctx context.Context, unit Unit, sessionID string, tier config.SelectedModelType) error {
	e.publish(EventVerificationStarted, unit, nil, "")
	e.logger.Info("Running verification", "unit", unit.String())

	results, err := e.verifier.RunVerification(ctx, e.dataDir)
	if err != nil {
		e.publish(EventVerificationFailed, unit, err, "Verification system error")
		return fmt.Errorf("verification for %s: %w", unit.String(), err)
	}

	if allPassed(results) {
		e.publish(EventVerificationPassed, unit, nil, "")
		e.logger.Info("Verification passed", "unit", unit.String())
		return nil
	}

	// First failure — build diagnostic and re-dispatch.
	diagnostic := FormatFailureDiagnostic(results)
	e.publish(EventVerificationFailed, unit, nil, diagnostic)
	e.logger.Error("Verification failed, retrying with diagnostic",
		"unit", unit.String(),
	)

	retryPrompt := fmt.Sprintf(
		"**VERIFICATION FAILED — AUTO-FIX ATTEMPT 1**\n\n"+
			"The verification gate ran after your previous attempt and found failures. "+
			"Fix these issues before completing the task.\n\n%s",
		diagnostic,
	)
	if retryErr := e.dispatch.RunWithForcedTier(ctx, sessionID, retryPrompt, tier); retryErr != nil {
		e.publish(EventUnitFailed, unit, retryErr, "Retry dispatch failed")
		return fmt.Errorf("retry dispatch for %s: %w", unit.String(), retryErr)
	}

	// Verify again after retry.
	e.publish(EventVerificationStarted, unit, nil, "Retry verification")
	retryResults, retryErr := e.verifier.RunVerification(ctx, e.dataDir)
	if retryErr != nil {
		e.publish(EventVerificationFailed, unit, retryErr, "Retry verification system error")
		return fmt.Errorf("retry verification for %s: %w", unit.String(), retryErr)
	}
	if allPassed(retryResults) {
		e.publish(EventVerificationPassed, unit, nil, "Passed after retry")
		e.logger.Info("Verification passed after retry", "unit", unit.String())
		return nil
	}

	// Retry also failed — do not advance.
	retryDiag := FormatFailureDiagnostic(retryResults)
	e.publish(EventVerificationFailed, unit, nil, retryDiag)
	e.mu.Lock()
	e.lastErr = "verification failed after retry"
	e.mu.Unlock()
	return fmt.Errorf("verification failed after retry for %s", unit.String())
}

// cleanupWorktree merges and removes the worktree for a completed
// milestone. Errors are logged but not returned — the work is done, so
// cleanup is best-effort.
func (e *Engine) cleanupWorktree(ctx context.Context, milestoneID string) {
	if e.worktreeManager == nil || e.worktreeMode != "per-milestone" {
		return
	}
	e.worktreeManager.Cleanup(ctx, milestoneID, e.autoPush, e.pushRemote)
}

// allPassed returns true when every result passed or the list is empty.
func allPassed(results []VerificationResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// tierForUnit maps unit types to model tiers. Research and planning use the
// planning tier, execution uses main, summarize and validate use background.
func tierForUnit(ut UnitType) config.SelectedModelType {
	switch ut {
	case UnitResearch, UnitPlanSlice:
		return config.SelectedModelTypePlanning
	case UnitExecuteTask:
		return config.SelectedModelTypeMain
	case UnitSummarizeSlice, UnitValidateMilestone:
		return config.SelectedModelTypeBackground
	default:
		return config.SelectedModelTypeMain
	}
}
