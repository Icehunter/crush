---
estimated_steps: 45
estimated_files: 1
skills_used: []
---

# T02: Add comprehensive DeriveState test suite with in-package DB helper

## Description

Create `internal/auto/state_test.go` with a test DB helper and comprehensive scenario-based tests proving DeriveState() works correctly across all edge cases. This is the primary verification artifact for R002.

## Negative Tests

- **Malformed inputs**: DependsOn referencing non-existent slice ID → treated as unmet dependency, no panic
- **Boundary conditions**: empty DB returns ActionNone; single milestone with no slices; slice with no tasks; all entities completed

## Steps

1. Create `internal/auto/state_test.go` in package `auto`.
2. Implement `setupTestDB(t *testing.T) (*sql.DB, *db.Queries)` helper following the exact pattern from `internal/db/auto_test.go`:
   - In-memory SQLite with `_pragma=foreign_keys(ON)` and `cache=shared`
   - Use `goose.SetBaseFS(db.FS)` and `goose.Up(sqliteDB, "migrations")`
   - `t.Cleanup` to close DB
   - Import `modernc.org/sqlite`, `github.com/pressly/goose/v3`, `github.com/charmbracelet/crush/internal/db`
3. Add helper `seedMilestone(t, q, id, status, phase)`, `seedSlice(t, q, id, milestoneID, title, status, phase, sortOrder, dependsOn)`, `seedTask(t, q, id, sliceID, milestoneID, title, status, phase, sortOrder)` that call the SQLC Create functions.
4. Write test functions (all using `t.Parallel()` and `testify/require`):
   - `TestDeriveState_EmptyDB` — returns ActionNone, all pointers nil
   - `TestDeriveState_PendingMilestone` — single pending milestone → ActionPlanMilestone
   - `TestDeriveState_ActiveMilestoneInPlanning` — active milestone in planning phase → ActionPlanMilestone
   - `TestDeriveState_ActiveMilestoneWithPendingSlice` — active milestone, one pending slice in pre_planning → ActionPlanSlice
   - `TestDeriveState_ActiveMilestoneWithActiveTask` — active milestone, active slice, active task → ActionExecuteTask
   - `TestDeriveState_PendingTaskSelected` — active milestone, active slice, first task pending → ActionExecuteTask with that task
   - `TestDeriveState_SliceDependencySatisfied` — S01 completed, S02 depends on S01 → S02 is actionable
   - `TestDeriveState_SliceDependencyNotMet` — S01 active, S02 depends on S01 → S02 skipped
   - `TestDeriveState_SliceDependencyMissing` — S02 depends on non-existent ID → S02 skipped
   - `TestDeriveState_AllTasksCompleted` — all tasks in slice completed → ActionCompleteSlice
   - `TestDeriveState_AllSlicesCompleted` — all slices completed → ActionCompleteMilestone
   - `TestDeriveState_AllCompleted` — all milestones completed → ActionNone
   - `TestDeriveState_SkipsCompletedSlices` — first slice completed, second slice pending → picks second slice
   - `TestDeriveState_RespectsSliceSortOrder` — out-of-order insertion, derivation follows sort_order
5. Run `gofumpt -w internal/auto/state_test.go`.
6. Run `go test ./internal/auto/ -v -count=1` to verify all tests pass.
7. Run `go test ./internal/db/ -run TestAuto -v -count=1` to confirm no regressions.

## Must-Haves

- [ ] setupTestDB helper in auto package using goose + in-memory SQLite
- [ ] Seed helpers for milestone, slice, task
- [ ] At least 12 test scenarios covering happy paths and edge cases
- [ ] All tests use t.Parallel() and testify/require
- [ ] Empty DB → ActionNone verified
- [ ] Dependency satisfaction and missing dependency verified
- [ ] Sort order respected verified
- [ ] Completion roll-up (task→slice, slice→milestone) verified
- [ ] No regressions in existing S01/S02 tests

## Verification

- `go test ./internal/auto/ -v -count=1` — all tests pass (S02 existing + S03 new)
- `go test ./internal/db/ -run TestAuto -v -count=1` — 4/4 pass, no regressions
- `go vet ./internal/auto/...` — exit 0

## Inputs

- ``internal/auto/state.go` — State struct, Action enum, and DeriveState() from T01`
- ``internal/auto/status.go` — Status/Phase enums and constants`
- ``internal/auto/milestone.go` — Milestone domain struct`
- ``internal/auto/slice.go` — Slice domain struct`
- ``internal/auto/task.go` — Task domain struct`
- ``internal/db/auto_test.go` — Reference for setupTestDB pattern (goose + in-memory SQLite + foreign_keys ON)`
- ``internal/db/embed.go` — db.FS embedded filesystem for migrations`
- ``internal/db/milestones.sql.go` — CreateMilestone, CreateMilestoneParams`
- ``internal/db/slices.sql.go` — CreateSlice, CreateSliceParams`
- ``internal/db/tasks.sql.go` — CreateTask, CreateTaskParams`

## Expected Output

- ``internal/auto/state_test.go` — Comprehensive test suite with 12+ scenarios and DB helper`

## Verification

go test ./internal/auto/ -v -count=1 && go test ./internal/db/ -run TestAuto -v -count=1
