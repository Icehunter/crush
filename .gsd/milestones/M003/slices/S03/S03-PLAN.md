# S03: Stuck Detection + Context Pressure

**Goal:** Engine tracks dispatch results in a sliding window for stuck detection and monitors token usage for context pressure, with both gates pausing the engine and publishing events when triggered.
**Demo:** After this: Engine tracks dispatch results in a sliding window. When >50% of recent dispatches fail for the same unit, engine retries with diagnostic then pauses if still stuck. Context monitor tracks token usage against model context window and signals wrap-up at configurable threshold. Tests prove stuck detection escalation and context pressure signaling.

## Tasks
- [x] **T01: Added StuckDetector with per-unit sliding window, wired into engine step() with diagnostic retry and pause-on-failure escalation, plus EventStuckDetected event** — Build the StuckDetector struct with a per-unit sliding window that tracks dispatch pass/fail outcomes. Wire it into Engine.step() to detect when >50% of recent dispatches for the same unit fail, retry with diagnostic, and pause if still stuck. Add EventStuckDetected event constant. Write unit tests for the detector and integration tests for the full engine stuck path.

## Steps

1. Add `EventStuckDetected` constant to `internal/auto/events.go`.
2. Create `internal/auto/stuck.go` with `StuckDetector` struct:
   - Fields: `windowSize int`, `mu sync.Mutex`, `windows map[string][]bool` (circular buffer per unit key)
   - `Record(unitKey string, passed bool)` — appends to the unit's window, evicting oldest if at capacity
   - `IsStuck(unitKey string) bool` — returns true when window is full AND >50% are failures
   - Unit key format: `MilestoneID/SliceID/TaskID` (use `unit.String()` style or `fmt.Sprintf`)
   - Thread-safe via sync.Mutex
   - Zero windowSize or nil detector means disabled
3. Create `internal/auto/stuck_test.go` with unit tests:
   - Window fills up, >50% failures triggers stuck
   - Mixed results (50/50) does not trigger (must be strictly >50%)
   - Window slides — old entries drop off, new results change stuck status
   - Empty window returns not-stuck
   - Single pass in full-failure window clears stuck if it drops below threshold
4. Add `stuckDetector *StuckDetector` field to Engine struct. Update `NewEngine` to accept `stuckDetector *StuckDetector` parameter (nil means disabled). Place it after `budgetCeiling` parameter.
5. Wire stuck detection into `Engine.step()`:
   - After verification gate (or after dispatch for non-task units), call `stuckDetector.Record(unitKey, passed)` where passed=true if dispatch+verification succeeded
   - Before the next dispatch (or at step entry after DeriveState), call `stuckDetector.IsStuck(unitKey)` — if stuck, retry with a diagnostic prompt, if retry also results in stuck, pause and publish EventStuckDetected
   - The stuck gate runs at the step level (tracks across loop iterations), NOT inside runVerificationGate
   - Use same pause pattern as budget exceeded: set paused flag, publish event, return nil
6. Update ALL existing `NewEngine` call sites in test files to pass the new `stuckDetector` parameter (nil for existing tests that don't test stuck detection).
7. Create `internal/auto/engine_stuck_integration_test.go` with integration tests:
   - `TestIntegration_StuckRetrySucceed`: dispatcher fails enough to trigger stuck, then succeeds on retry — advancer called, engine continues
   - `TestIntegration_StuckRetryFail`: dispatcher keeps failing, stuck detected, retry fails, engine pauses and publishes EventStuckDetected
   - `TestIntegration_StuckNotTriggeredBelowThreshold`: fewer failures than threshold, engine proceeds normally
8. Run `go vet ./internal/auto/...`, `go build ./internal/auto/`, `go test ./internal/auto/ -count=1 -v`

## Constraints
- Do NOT change the Dispatcher interface — it returns only error
- Stuck detection is separate from verification retry — verification retries within a single step, stuck detection tracks outcomes across multiple loop iterations
- StuckDetector is a concrete struct, not an interface — no need for mocking since it's pure in-memory
- The `stuck_threshold` config field (AutoConfig.StuckThreshold, default 5) controls window size
- Use `fmt.Sprintf("%s/%s/%s", unit.MilestoneID, unit.SliceID, unit.TaskID)` for unit key to distinguish tasks
  - Estimate: 1h30m
  - Files: internal/auto/stuck.go, internal/auto/stuck_test.go, internal/auto/events.go, internal/auto/engine.go, internal/auto/engine_stuck_integration_test.go, internal/auto/engine_test.go, internal/auto/engine_budget_integration_test.go, internal/auto/engine_verify_integration_test.go
  - Verify: go vet ./internal/auto/... && go build ./internal/auto/ && go test ./internal/auto/ -run 'TestStuck|TestIntegration_Stuck' -count=1 -v && go test ./internal/auto/ -count=1
- [x] **T02: Added ContextMonitor with TokenQuerier interface, wired into engine step() to pause and publish EventContextPressure when session token usage exceeds configurable threshold** — Build the ContextMonitor struct that compares cumulative session token usage against a configurable threshold of the model context window. Define a TokenQuerier interface for reading session token counts. Wire into Engine.step() to pause and publish EventContextPressure when usage exceeds threshold.

## Steps

1. Add `EventContextPressure` constant to `internal/auto/events.go`.
2. Create `internal/auto/context.go`:
   - `TokenQuerier` interface with `GetTokenUsage(ctx context.Context, sessionID string) (promptTokens int64, completionTokens int64, err error)`
   - `ContextMonitor` struct with fields: `threshold float64` (0.0-1.0, default 0.8), `contextWindow int64` (model context window size in tokens), `tokenQuerier TokenQuerier`
   - `NewContextMonitor(threshold float64, contextWindow int64, querier TokenQuerier) *ContextMonitor` constructor — returns nil if contextWindow <= 0 or querier is nil (disabled)
   - `Check(ctx context.Context, sessionID string) (exceeded bool, err error)` method — queries token usage, computes `(prompt + completion) / contextWindow`, returns true if >= threshold
3. Create `internal/auto/context_test.go` with unit tests:
   - Below threshold returns false
   - At threshold returns true
   - Above threshold returns true
   - Zero context window returns false (safety — NewContextMonitor returns nil)
   - Nil querier returns false (safety — NewContextMonitor returns nil)
   - Error from querier propagated
4. Add `contextMonitor *ContextMonitor` field to Engine struct. Update `NewEngine` to accept `contextMonitor *ContextMonitor` parameter (nil means disabled). Place it after `stuckDetector` parameter.
5. Wire context pressure check into `Engine.step()`:
   - After dispatch succeeds (and after verification and stuck recording), check context pressure
   - If exceeded: log, publish EventContextPressure, set paused flag, return nil (same pattern as budget exceeded)
   - Pass the parent session ID to Check() — the monitor queries cumulative tokens for the session
6. Update ALL existing `NewEngine` call sites (including those updated in T01) to pass the new `contextMonitor` parameter (nil for tests that don't test context pressure).
7. Create `internal/auto/engine_context_integration_test.go` with integration tests:
   - `TestIntegration_ContextPressurePauses`: mock TokenQuerier returns high usage, engine pauses and publishes EventContextPressure after first dispatch
   - `TestIntegration_ContextPressureBelowThreshold`: mock TokenQuerier returns low usage, engine completes normally
   - `TestIntegration_ContextPressureNilMonitorSkips`: nil context monitor, engine completes normally
8. Run `go vet ./internal/auto/...`, `go build ./internal/auto/`, `go test ./internal/auto/ -count=1 -v` (all tests including T01's stuck tests)

## Constraints
- Do NOT change the Dispatcher interface
- TokenQuerier is a new interface in internal/auto/context.go — same pattern as BudgetQuerier
- ContextMonitor.Check uses the parent session ID, not child session ID — cumulative usage across all dispatches
- Context pressure runs AFTER dispatch succeeds, not before (unlike budget which runs before)
- The threshold and contextWindow are set at engine creation time, not hot-reloaded
- D009 applies: this is engine-level context pressure, independent of agent-level StopWhen auto-summarization
  - Estimate: 1h30m
  - Files: internal/auto/context.go, internal/auto/context_test.go, internal/auto/events.go, internal/auto/engine.go, internal/auto/engine_context_integration_test.go, internal/auto/engine_test.go, internal/auto/engine_budget_integration_test.go, internal/auto/engine_verify_integration_test.go, internal/auto/engine_stuck_integration_test.go
  - Verify: go vet ./internal/auto/... && go build ./internal/auto/ && go test ./internal/auto/ -run 'TestContextMonitor|TestIntegration_Context' -count=1 -v && go test ./internal/auto/ -count=1
