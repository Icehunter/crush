---
estimated_steps: 34
estimated_files: 8
skills_used: []
---

# T01: Implement stuck detection with sliding window, engine wiring, and integration tests

Build the StuckDetector struct with a per-unit sliding window that tracks dispatch pass/fail outcomes. Wire it into Engine.step() to detect when >50% of recent dispatches for the same unit fail, retry with diagnostic, and pause if still stuck. Add EventStuckDetected event constant. Write unit tests for the detector and integration tests for the full engine stuck path.

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

## Inputs

- ``internal/auto/engine.go` — Engine struct, step(), NewEngine constructor to extend`
- ``internal/auto/events.go` — existing event constants to add EventStuckDetected`
- ``internal/auto/engine_test.go` — mock helpers (fixedSequenceQuerier, mockAdvancer, etc.) and existing NewEngine call sites`
- ``internal/auto/engine_budget_integration_test.go` — existing NewEngine call sites to update`
- ``internal/auto/engine_verify_integration_test.go` — existing NewEngine call sites to update`
- ``internal/auto/unit.go` — Unit struct for key generation`

## Expected Output

- ``internal/auto/stuck.go` — StuckDetector struct with Record, IsStuck methods and sliding window`
- ``internal/auto/stuck_test.go` — unit tests for StuckDetector`
- ``internal/auto/events.go` — EventStuckDetected constant added`
- ``internal/auto/engine.go` — stuckDetector field, NewEngine parameter, stuck gate in step()`
- ``internal/auto/engine_stuck_integration_test.go` — integration tests for stuck→retry→succeed and stuck→retry→fail→pause paths`

## Verification

go vet ./internal/auto/... && go build ./internal/auto/ && go test ./internal/auto/ -run 'TestStuck|TestIntegration_Stuck' -count=1 -v && go test ./internal/auto/ -count=1
