---
id: S05
parent: M001
milestone: M001
provides:
  - Integration proof that DeriveState→Dispatch cycle produces correct action sequences for full milestone lifecycles
requires:
  - slice: S04
    provides: Dispatch rules table
affects:
  []
key_files:
  - internal/auto/integration_test.go
key_decisions:
  - Used table-driven step loop in FullLifecycle test for concise 11-step assertion
patterns_established:
  - advanceState/setMilestone/setSlice/setTask integration test helpers for composing state machine cycles against real SQLite
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M001/slices/S05/tasks/T01-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T19:18:27.593Z
blocker_discovered: false
---

# S05: Integration Proof

**4 integration tests proving DeriveState→Dispatch produces correct action sequences across full milestone lifecycles against real SQLite**

## What Happened

Created `internal/auto/integration_test.go` with four integration tests that compose DeriveState() and Dispatch() against real in-memory SQLite databases. An `advanceState` helper reduces per-step boilerplate by calling DeriveState then Dispatch and returning the action. Three additional helpers (`setMilestone`, `setSlice`, `setTask`) wrap SQLC update queries for concise state advancement.

TestIntegration_FullLifecycle walks a complete milestone through 11 steps: plan_milestone → plan_slice(S01) → execute_task(T01) → execute_task(T02) → complete_slice(S01) → plan_slice(S02) → execute_task(T03) → execute_task(T04) → complete_slice(S02) → complete_milestone → none. Uses a table-driven step loop for concise assertion of the full sequence.

TestIntegration_EmptyDB proves an empty database returns ActionNone. TestIntegration_DependencyGating proves blocked slices are skipped until their dependency completes. TestIntegration_TerminalState proves a fully-completed milestone returns ActionNone.

All 45 tests pass (41 existing + 4 new integration tests). Build, vet, and test all clean.

## Verification

go build ./internal/auto/... — exits 0. go vet ./internal/auto/... — exits 0. go test ./internal/auto/ -v -count=1 -run TestIntegration — all 4 integration tests pass. go test ./internal/auto/ -v -count=1 — full suite passes (45 tests total).

## Requirements Advanced

- R001 — Integration tests prove milestones/slices/tasks work as first-class entities through a full lifecycle
- R003 — Integration tests prove dispatch rules produce correct actions for every state transition in a complete lifecycle

## Requirements Validated

- R003 — TestIntegration_FullLifecycle asserts 11-step action sequence from plan_milestone through none, proving dispatch rules produce correct actions for every state. TestIntegration_DependencyGating proves blocked slices are skipped. TestIntegration_TerminalState proves completed state yields ActionNone.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

Plan estimated 42 existing tests; actual count was 41. No impact on verification.

## Known Limitations

Integration tests cover the DeriveState→Dispatch cycle but do not test actual LLM execution or session creation — those are M002 concerns.

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/integration_test.go` — New file: 4 integration tests with helpers proving DeriveState→Dispatch cycle against real SQLite
