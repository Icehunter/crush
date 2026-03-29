---
estimated_steps: 39
estimated_files: 7
skills_used: []
---

# T01: Write integration tests proving full derive→dispatch lifecycle

Create `internal/auto/integration_test.go` with integration tests that compose DeriveState() and Dispatch() against a real in-memory SQLite database.

## Steps

1. Read `internal/auto/state_test.go` to understand the existing test helpers: `setupTestDB()`, `seedMilestone()`, `seedSlice()`, `seedTask()`. These are unexported and in-package — reuse them directly.
2. Read `internal/auto/state.go` and `internal/auto/dispatch.go` for function signatures. Key types: `DeriveState(ctx, q) (*State, error)`, `Dispatch(state) Action`, `Action` constants (`ActionPlanMilestone`, `ActionPlanSlice`, `ActionExecuteTask`, `ActionCompleteSlice`, `ActionCompleteMilestone`, `ActionNone`).
3. Read the SQLC update queries in `internal/db/milestones.sql.go`, `internal/db/slices.sql.go`, `internal/db/tasks.sql.go` — specifically `UpdateMilestoneStatus`, `UpdateMilestonePhase`, `UpdateSliceStatus`, `UpdateSlicePhase`, `UpdateTaskStatus`, `UpdateTaskPhase` params.
4. Create `internal/auto/integration_test.go` in `package auto` with:
   - **`advanceState` helper**: calls `DeriveState(ctx, q)` then `Dispatch(state)`, returns the action. Reduces per-step boilerplate.
   - **`TestIntegration_FullLifecycle`**: Seeds a milestone (pending/pre_planning) with 2 slices (S01 no deps, S02 depends on S01), each with 2 tasks. Walks through the entire lifecycle step-by-step:
     1. Derive+Dispatch → `plan_milestone`. Advance: set milestone status=active, phase=planning.
     2. Derive+Dispatch → `plan_slice` (S01). Advance: set S01 status=active, phase=executing.
     3. Derive+Dispatch → `execute_task` (T01). Advance: set T01 status=completed.
     4. Derive+Dispatch → `execute_task` (T02). Advance: set T02 status=completed.
     5. Derive+Dispatch → `complete_slice` (S01). Advance: set S01 status=completed.
     6. Derive+Dispatch → `plan_slice` (S02). Advance: set S02 status=active, phase=executing.
     7. Derive+Dispatch → `execute_task` (T03). Advance: set T03 status=completed.
     8. Derive+Dispatch → `execute_task` (T04). Advance: set T04 status=completed.
     9. Derive+Dispatch → `complete_slice` (S02). Advance: set S02 status=completed.
     10. Derive+Dispatch → `complete_milestone`. Advance: set milestone status=completed.
     11. Derive+Dispatch → `none` (terminal).
     Assert each action matches the expected sequence. Also assert State.Milestone, State.Slice, and State.Task point to the correct entities at each step.
   - **`TestIntegration_EmptyDB`**: Empty DB → DeriveState + Dispatch returns `ActionNone`.
   - **`TestIntegration_DependencyGating`**: Seed milestone with S01 (no deps, pending) and S02 (depends on S01, pending). Set milestone active. Advance S01 to active/executing but don't complete it. Derive state — should target S01's first task, not S02. Complete S01 — now S02 becomes actionable.
   - **`TestIntegration_TerminalState`**: Seed a fully-completed milestone (milestone completed, all slices completed, all tasks completed). DeriveState + Dispatch → `ActionNone`.
5. All tests use `t.Parallel()`, `testify/require`, and follow project conventions (AGENTS.md).
6. Run `go build ./internal/auto/...` and `go vet ./internal/auto/...` — must be clean.
7. Run `go test ./internal/auto/ -v -count=1` — all tests pass (existing 42 + new integration tests).
8. Format with `gofumpt -w internal/auto/integration_test.go` (fall back to `gofmt` if unavailable).

## Must-Haves

- [ ] `TestIntegration_FullLifecycle` asserts exact 11-step action sequence with correct entity targeting
- [ ] `TestIntegration_EmptyDB` proves empty DB returns ActionNone
- [ ] `TestIntegration_DependencyGating` proves blocked slices are skipped until dependency completes
- [ ] `TestIntegration_TerminalState` proves all-completed state returns ActionNone
- [ ] All tests use t.Parallel() and testify/require
- [ ] Existing 42 tests still pass

## Verification

- `go build ./internal/auto/...` exits 0
- `go vet ./internal/auto/...` exits 0
- `go test ./internal/auto/ -v -count=1 -run TestIntegration` — all 4 integration tests pass
- `go test ./internal/auto/ -v -count=1` — full suite passes (42 existing + 4 new = 46+ tests)

## Inputs

- ``internal/auto/state.go` — DeriveState() function to compose`
- ``internal/auto/dispatch.go` — Dispatch() function to compose`
- ``internal/auto/state_test.go` — setupTestDB, seedMilestone, seedSlice, seedTask helpers to reuse`
- ``internal/auto/status.go` — Status/Phase/Action constants`
- ``internal/db/milestones.sql.go` — UpdateMilestoneStatus, UpdateMilestonePhase SQLC queries`
- ``internal/db/slices.sql.go` — UpdateSliceStatus, UpdateSlicePhase SQLC queries`
- ``internal/db/tasks.sql.go` — UpdateTaskStatus, UpdateTaskPhase SQLC queries`

## Expected Output

- ``internal/auto/integration_test.go` — New file with 4 integration tests proving derive→dispatch lifecycle`

## Verification

go build ./internal/auto/... && go vet ./internal/auto/... && go test ./internal/auto/ -v -count=1
