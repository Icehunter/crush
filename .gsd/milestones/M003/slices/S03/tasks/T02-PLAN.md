---
estimated_steps: 33
estimated_files: 9
skills_used: []
---

# T02: Implement context pressure monitoring with TokenQuerier, engine wiring, and integration tests

Build the ContextMonitor struct that compares cumulative session token usage against a configurable threshold of the model context window. Define a TokenQuerier interface for reading session token counts. Wire into Engine.step() to pause and publish EventContextPressure when usage exceeds threshold.

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

## Inputs

- ``internal/auto/engine.go` — Engine struct with stuckDetector (from T01), step(), NewEngine to extend further`
- ``internal/auto/events.go` — event constants including EventStuckDetected (from T01)`
- ``internal/auto/stuck.go` — StuckDetector struct (from T01, must not break)`
- ``internal/auto/engine_test.go` — mock helpers and NewEngine call sites (updated by T01)`
- ``internal/auto/engine_stuck_integration_test.go` — NewEngine call sites from T01 to update`
- ``internal/auto/budget.go` — BudgetQuerier interface pattern to follow for TokenQuerier`

## Expected Output

- ``internal/auto/context.go` — ContextMonitor struct, TokenQuerier interface, NewContextMonitor, Check method`
- ``internal/auto/context_test.go` — unit tests for ContextMonitor`
- ``internal/auto/events.go` — EventContextPressure constant added`
- ``internal/auto/engine.go` — contextMonitor field, NewEngine parameter, context pressure gate in step()`
- ``internal/auto/engine_context_integration_test.go` — integration tests for context pressure paths`

## Verification

go vet ./internal/auto/... && go build ./internal/auto/ && go test ./internal/auto/ -run 'TestContextMonitor|TestIntegration_Context' -count=1 -v && go test ./internal/auto/ -count=1
