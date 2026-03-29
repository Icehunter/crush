---
id: S03
parent: M003
milestone: M003
provides:
  - StuckDetector struct with per-unit sliding window
  - ContextMonitor struct with TokenQuerier interface
  - EventStuckDetected and EventContextPressure event constants
requires:
  - slice: S01
    provides: Auto config with stuck_threshold field, verification gate pattern, NewEngine signature
affects:
  []
key_files:
  - internal/auto/stuck.go
  - internal/auto/stuck_test.go
  - internal/auto/context.go
  - internal/auto/context_test.go
  - internal/auto/events.go
  - internal/auto/engine.go
  - internal/auto/engine_stuck_integration_test.go
  - internal/auto/engine_context_integration_test.go
key_decisions:
  - Stuck detection uses a ringBuffer circular buffer for O(1) push and constant memory
  - Stuck gate runs before dispatch when IsStuck returns true, separate from the verification gate
  - Context pressure check runs after dispatch succeeds and after stuck recording, before status advance
  - NewContextMonitor returns nil for zero contextWindow or nil querier, making the gate a safe no-op
patterns_established:
  - Safety gates follow a consistent pattern: nil/zero config disables the gate, gate checks at a defined point in step(), exceeded condition pauses engine and publishes a typed event
observability_surfaces:
  - EventStuckDetected published when engine pauses due to stuck unit
  - EventContextPressure published when engine pauses due to context pressure
drill_down_paths:
  - .gsd/milestones/M003/slices/S03/tasks/T01-SUMMARY.md
  - .gsd/milestones/M003/slices/S03/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-28T05:34:11.848Z
blocker_discovered: false
---

# S03: Stuck Detection + Context Pressure

**Stuck detection with per-unit sliding window and diagnostic retry escalation, plus context pressure monitoring that pauses the engine when session token usage approaches the model context window**

## What Happened

Built two new safety rails for the auto-mode engine, completing the M003 safety suite.

**T01 — StuckDetector:** Created `internal/auto/stuck.go` with a `StuckDetector` struct backed by a per-unit circular ring buffer (`ringBuffer`). Each unit key (MilestoneID/SliceID/TaskID) gets its own sliding window tracking pass/fail outcomes. `IsStuck()` returns true when the window is full and >50% are failures. The detector is thread-safe via `sync.Mutex`. Wired into `engine.step()` as a pre-dispatch gate — when `IsStuck` fires, the engine dispatches a diagnostic retry. If the retry also fails (still stuck), the engine pauses and publishes `EventStuckDetected`, following the same pause pattern as budget exceeded. Added 9 unit tests covering window fill, 50/50 boundary, sliding eviction, partial window safety, nil detector, zero window size, multi-unit independence. Added 3 integration tests proving the full engine paths: stuck-retry-succeed, stuck-retry-fail-pause, below-threshold-normal.

**T02 — ContextMonitor:** Created `internal/auto/context.go` with a `TokenQuerier` interface (matching the `BudgetQuerier` pattern) and `ContextMonitor` struct. The monitor compares cumulative session token usage (`prompt + completion`) against a configurable fraction of the model context window. `NewContextMonitor` returns nil for invalid inputs (zero window, nil querier), making the gate a safe no-op when disabled. Wired into `engine.step()` after dispatch success and stuck recording — when token usage exceeds the threshold, the engine pauses and publishes `EventContextPressure`. Added 6 unit tests and 3 integration tests covering threshold comparison, nil/zero safety, error propagation, and full engine paths.

Both gates required updating `NewEngine` across all existing call sites (14 for T01, then 17 for T02 including T01's new tests). Both follow the established pause-and-publish pattern from S02's budget ceiling.

## Verification

All verification passed across both tasks:
- `go vet ./internal/auto/...` — clean (exit 0)
- `go build ./internal/auto/` — clean (exit 0)
- Targeted stuck tests: 12/12 pass (TestStuck* + TestIntegration_Stuck*)
- Targeted context tests: 9/9 pass (TestContextMonitor* + TestIntegration_Context*)
- Full suite: `go test ./internal/auto/ -count=1` — all tests pass including all prior S01/S02 tests, total ~100 tests across the auto package

## Requirements Advanced

None.

## Requirements Validated

- R012 — StuckDetector with per-unit sliding window, 9 unit tests + 3 integration tests prove stuck detection, diagnostic retry, pause escalation, and below-threshold normal operation
- R013 — ContextMonitor with TokenQuerier interface, 6 unit tests + 3 integration tests prove token usage threshold comparison, pause+EventContextPressure, nil/zero safety

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

Stuck gate checks before dispatch (at step entry) rather than after, since it tracks outcomes across loop iterations. This is a design choice within the plan's constraints, not a deviation from intent.

## Known Limitations

None. Both gates are fully functional with configurable thresholds and safe no-op behavior when disabled.

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/stuck.go` — New: StuckDetector struct with per-unit sliding window backed by circular ring buffer
- `internal/auto/stuck_test.go` — New: 9 unit tests for StuckDetector (window fill, threshold, sliding, nil safety, multi-unit)
- `internal/auto/context.go` — New: TokenQuerier interface and ContextMonitor struct comparing session token usage against context window threshold
- `internal/auto/context_test.go` — New: 6 unit tests for ContextMonitor (threshold comparison, nil/zero safety, error propagation)
- `internal/auto/events.go` — Added EventStuckDetected and EventContextPressure constants
- `internal/auto/engine.go` — Added stuckDetector and contextMonitor fields to Engine, wired both gates into step()
- `internal/auto/engine_stuck_integration_test.go` — New: 3 integration tests for stuck detection engine path
- `internal/auto/engine_context_integration_test.go` — New: 3 integration tests for context pressure engine path
- `internal/auto/engine_test.go` — Updated NewEngine calls with stuckDetector and contextMonitor params
- `internal/auto/engine_budget_integration_test.go` — Updated NewEngine calls with new params
- `internal/auto/engine_verify_integration_test.go` — Updated NewEngine calls with new params
