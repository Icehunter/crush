---
id: S03
parent: M001
milestone: M001
provides:
  - DeriveState(ctx, *db.Queries) → (*State, error) function
  - Action enum with 6 constants for dispatch routing
  - State struct with optional Milestone/Slice/Task pointers
requires:
  - slice: S02
    provides: Domain model types (Milestone, Slice, Task) and MilestoneFromDB/SliceFromDB/TaskFromDB converters
affects:
  - S04
key_files:
  - internal/auto/state.go
  - internal/auto/state_test.go
key_decisions:
  - Dependency check errors treated as 'not met' — skips slice safely without panicking
  - Walk logic separated into findActiveMilestone, deriveSliceState, deriveTaskState helpers
  - Used in-package test (package auto) to access internal helpers directly
patterns_established:
  - setupTestDB + seed helpers pattern for testing auto-mode functions against real SQLite with migrations
  - Action enum as typed string constants — dispatch rules (S04) will switch on these values
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M001/slices/S03/tasks/T01-SUMMARY.md
  - .gsd/milestones/M001/slices/S03/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T18:52:56.039Z
blocker_discovered: false
---

# S03: State Derivation Engine

**DeriveState() walks milestone→slice→task hierarchy with dependency gating and returns the next actionable unit for auto-mode**

## What Happened

T01 created `internal/auto/state.go` with the core state derivation engine. It defines the `Action` type (string) with 6 named constants — `ActionNone`, `ActionPlanMilestone`, `ActionPlanSlice`, `ActionExecuteTask`, `ActionCompleteSlice`, `ActionCompleteMilestone` — and a `State` struct carrying the action plus optional Milestone/Slice/Task pointers. The `DeriveState(ctx, *db.Queries)` function implements the full walk algorithm: find the first active (or pending) milestone, check its phase for planning needs, walk slices in sort_order with dependency checking (errors treated as unmet — skip safely, don't panic), then walk tasks to find the next actionable one. Handles empty DB, planning phases, dependency gates, completion rollups, and blocked states. The implementation was decomposed into `findActiveMilestone`, `deriveSliceState`, and `deriveTaskState` helpers for clarity.

T02 created `internal/auto/state_test.go` with 14 comprehensive test scenarios exercising every branch of DeriveState. Tests use an in-package `setupTestDB` helper (in-memory SQLite + goose migrations) and seed functions for milestone/slice/task. Coverage includes: empty DB → ActionNone, pending/active milestone planning, slice planning, task execution, dependency satisfaction, dependency blocking, missing dependency (non-existent ID), completion roll-up from task→slice and slice→milestone, skip-completed logic, sort order correctness, and all-completed → ActionNone. All tests use `t.Parallel()` and `testify/require`.

## Verification

All slice-level verification passed:
- `go test ./internal/auto/ -v -count=1`: 30/30 PASS (14 new DeriveState tests + 16 existing S02 tests)
- `go test ./internal/db/ -run TestAuto -v -count=1`: 4/4 PASS (no regressions)
- `go build ./internal/auto/...`: exit 0
- `go vet ./internal/auto/...`: exit 0
- grep checks for `func DeriveState`, `type Action string`, `type State struct`: all pass

## Requirements Advanced

- R002 — DeriveState() implemented and tested with 14 scenarios — queries DB and returns active milestone, slice, task, and current phase

## Requirements Validated

- R002 — 14-scenario test suite proves DeriveState returns correct Action for empty DB, planning phases, task execution, dependency gating, completion rollups, and sort order

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

None.

## Known Limitations

DeriveState only checks a single DependsOn value per slice — multiple dependencies would require parsing a comma-separated list or changing the schema. This is sufficient for the current milestone structure but may need extension.

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/state.go` — DeriveState() function, Action type (6 constants), State struct, and helper functions for milestone→slice→task walk with dependency checking
- `internal/auto/state_test.go` — 14-scenario test suite with setupTestDB helper and seed functions covering all DeriveState branches
