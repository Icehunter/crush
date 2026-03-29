---
estimated_steps: 37
estimated_files: 1
skills_used: []
---

# T03: Integration tests for verify→retry→succeed and verify→retry→fail engine paths

## Description

This task writes the integration tests that prove the slice's demo: the full verify→retry→succeed path (first dispatch fails verification, retry dispatch passes) and the verify→retry→fail path (both dispatches fail verification, engine does not advance). These are the objective stopping condition for the slice.

## Steps

1. Create or extend `internal/auto/engine_integration_test.go` (in the M003 worktree) with two integration tests:

2. `TestIntegration_VerifyRetrySucceed`:
   - Set up a mock querier that returns a single `UnitExecuteTask` unit.
   - Set up a `mockVerifier` that fails on the first call and succeeds on the second call.
   - Set up a mock dispatcher that records all calls (including the retry dispatch with diagnostic prompt).
   - Set up a mock advancer coupled to the querier (per K006/K008).
   - Create engine with the mock verifier, run `engine.Step()`.
   - Assert: dispatcher called exactly twice (original + retry with diagnostic).
   - Assert: advancer called exactly once (status advanced after successful retry).
   - Assert: events published in order: UnitStarted → VerificationStarted → VerificationFailed → VerificationStarted → VerificationPassed → UnitCompleted.
   - Assert: the retry dispatch prompt contains truncated failure output from the first verification.

3. `TestIntegration_VerifyRetryFail`:
   - Same setup, but mockVerifier fails on both calls.
   - Create engine, run `engine.Step()`.
   - Assert: dispatcher called exactly twice (original + retry).
   - Assert: advancer NOT called (status not advanced).
   - Assert: events include VerificationFailed twice, no UnitCompleted.
   - Assert: `engine.Step()` returns an error.

4. `TestIntegration_VerifySkippedForNonTaskUnits`:
   - Set up a mock querier that returns a `UnitResearch` or `UnitPlanSlice` unit.
   - Set up a mock verifier that would fail if called.
   - Assert: verifier is never called for non-task units.
   - Assert: advancer is called (dispatch completes normally).

5. Add a helper `mockVerifier` struct that takes a sequence of `[]VerificationResult` responses and returns them in order.

6. Run the full test suite: `go test ./internal/auto/ -count=1 -v`
7. Format: `gofumpt -w internal/auto/engine_integration_test.go`

## Must-Haves

- [ ] `TestIntegration_VerifyRetrySucceed` passes — proves dispatch→fail→retry→succeed→advance path
- [ ] `TestIntegration_VerifyRetryFail` passes — proves dispatch→fail→retry→fail→no-advance path
- [ ] `TestIntegration_VerifySkippedForNonTaskUnits` passes — proves verification only runs for execute_task
- [ ] All existing M002 engine tests still pass

## Verification

- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run TestIntegration_Verification -count=1 -v`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -count=1`

## Inputs

- ``internal/auto/engine.go` — engine with verification gate wired in`
- ``internal/auto/verify.go` — Verifier interface to mock`
- ``internal/auto/events.go` — event types to assert on`
- ``internal/auto/engine_test.go` — existing mock patterns (mockSessionCreator, mockDispatcher, mockAdvancer)`

## Expected Output

- ``internal/auto/engine_integration_test.go` — integration tests for verify→retry→succeed, verify→retry→fail, and verify-skipped-for-non-task paths`

## Verification

cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run 'TestIntegration_Verify' -count=1 -v && go test ./internal/auto/ -count=1
