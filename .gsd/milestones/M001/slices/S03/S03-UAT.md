# S03: State Derivation Engine — UAT

**Milestone:** M001
**Written:** 2026-03-27T18:52:56.039Z

# S03: State Derivation Engine — UAT

**Milestone:** M001
**Written:** 2026-03-27

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This slice produces Go source code and tests with no runtime behavior — compilation and test execution prove correctness

## Preconditions

- Go toolchain installed (go 1.24+)
- Working directory is the crush project root
- S01 DB schema and S02 domain model already in place

## Smoke Test

Run `go test ./internal/auto/ -run TestDeriveState -v -count=1` — all 14 DeriveState tests should pass.

## Test Cases

### 1. Empty Database Returns ActionNone

1. Run `go test ./internal/auto/ -run TestDeriveState_EmptyDB -v`
2. **Expected:** PASS — DeriveState returns ActionNone with all pointers nil

### 2. Pending Milestone Triggers Planning

1. Run `go test ./internal/auto/ -run TestDeriveState_PendingMilestone -v`
2. **Expected:** PASS — single pending milestone → ActionPlanMilestone with Milestone populated

### 3. Active Milestone in Planning Phase

1. Run `go test ./internal/auto/ -run TestDeriveState_ActiveMilestoneInPlanning -v`
2. **Expected:** PASS — active milestone in planning phase → ActionPlanMilestone

### 4. Slice Planning Detected

1. Run `go test ./internal/auto/ -run TestDeriveState_ActiveMilestoneWithPendingSlice -v`
2. **Expected:** PASS — active milestone with pending slice in pre_planning → ActionPlanSlice

### 5. Task Execution Selected

1. Run `go test ./internal/auto/ -run TestDeriveState_ActiveMilestoneWithActiveTask -v`
2. **Expected:** PASS — active milestone, active slice, active task → ActionExecuteTask with Task populated

### 6. Pending Task Selected When No Active Task

1. Run `go test ./internal/auto/ -run TestDeriveState_PendingTaskSelected -v`
2. **Expected:** PASS — first pending task selected when no active task exists

### 7. Dependency Satisfied Allows Slice

1. Run `go test ./internal/auto/ -run TestDeriveState_SliceDependencySatisfied -v`
2. **Expected:** PASS — S01 completed, S02 depends on S01 → S02 is actionable

### 8. Dependency Not Met Skips Slice

1. Run `go test ./internal/auto/ -run TestDeriveState_SliceDependencyNotMet -v`
2. **Expected:** PASS — S01 still active, S02 depends on S01 → S02 skipped

### 9. Missing Dependency Treated as Unmet

1. Run `go test ./internal/auto/ -run TestDeriveState_SliceDependencyMissing -v`
2. **Expected:** PASS — dependency references non-existent slice ID → skipped without panic

### 10. Task Completion Rolls Up to Slice

1. Run `go test ./internal/auto/ -run TestDeriveState_AllTasksCompleted -v`
2. **Expected:** PASS — all tasks completed → ActionCompleteSlice

### 11. Slice Completion Rolls Up to Milestone

1. Run `go test ./internal/auto/ -run TestDeriveState_AllSlicesCompleted -v`
2. **Expected:** PASS — all slices completed → ActionCompleteMilestone

### 12. All Milestones Completed

1. Run `go test ./internal/auto/ -run TestDeriveState_AllCompleted -v`
2. **Expected:** PASS — all milestones completed → ActionNone

### 13. Completed Slices Skipped

1. Run `go test ./internal/auto/ -run TestDeriveState_SkipsCompletedSlices -v`
2. **Expected:** PASS — first slice completed, second pending → picks second

### 14. Sort Order Respected

1. Run `go test ./internal/auto/ -run TestDeriveState_RespectsSliceSortOrder -v`
2. **Expected:** PASS — out-of-order insertion still derives correct slice by sort_order

## Edge Cases

### Missing Dependency ID

1. Create slice with DependsOn pointing to non-existent ID
2. Call DeriveState
3. **Expected:** Slice skipped safely, no panic, no error propagation

### Empty Milestone (No Slices)

1. Active milestone with executing phase but no slices
2. **Expected:** ActionNone (blocked — no work to do)

### All Completed Hierarchy

1. All milestones marked completed
2. **Expected:** ActionNone with all pointers nil

## Failure Signals

- Any of the 14 DeriveState tests fail
- `go build ./internal/auto/...` fails with type errors
- `go vet ./internal/auto/...` reports issues
- DB regression tests in `internal/db/` fail after changes

## Not Proven By This UAT

- Runtime integration with the dispatch loop (S04 scope)
- Performance under large hierarchies (hundreds of milestones/slices)
- Concurrent access to DeriveState from multiple goroutines
- Integration with CLI commands (M002 scope)

## Notes for Tester

- All 14 DeriveState tests use t.Parallel() — safe to run with -race flag
- Each test creates its own in-memory SQLite database via setupTestDB — fully isolated
- The setupTestDB pattern established here is reused by future slices testing auto-mode functions
- Run the full suite with `go test ./internal/auto/ -v -count=1` to also verify S02 domain model tests (30 total)
