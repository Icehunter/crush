---
estimated_steps: 22
estimated_files: 1
skills_used: []
---

# T02: Create dispatch_test.go with comprehensive test coverage for all rules and edge cases

Create `internal/auto/dispatch_test.go` with tests proving every rule fires correctly, ordering is respected, and edge cases are handled safely.

## Steps

1. Create `internal/auto/dispatch_test.go` in the `auto` package (in-package test, not `auto_test`).
2. Import `testing` and `github.com/stretchr/testify/require`.
3. Write `TestDispatch_NilState` — calls `Dispatch(nil)`, asserts result is `ActionNone`. Uses `t.Parallel()`.
4. Write `TestDispatch_ActionNone` — creates `State{Action: ActionNone}`, asserts `Dispatch` returns `ActionNone`.
5. Write `TestDispatch_ExecuteTask` — creates State with `Action: ActionExecuteTask` and populated Milestone/Slice/Task pointers, asserts `Dispatch` returns `ActionExecuteTask`.
6. Write `TestDispatch_PlanSlice` — creates State with `Action: ActionPlanSlice` and Milestone/Slice pointers, asserts `ActionPlanSlice`.
7. Write `TestDispatch_PlanMilestone` — creates State with `Action: ActionPlanMilestone` and Milestone pointer, asserts `ActionPlanMilestone`.
8. Write `TestDispatch_CompleteSlice` — creates State with `Action: ActionCompleteSlice`, asserts `ActionCompleteSlice`.
9. Write `TestDispatch_CompleteMilestone` — creates State with `Action: ActionCompleteMilestone`, asserts `ActionCompleteMilestone`.
10. Write `TestDispatch_FallbackToNone` — creates State with Action set to an unrecognized string (e.g., `Action("unknown")`), asserts `Dispatch` returns `ActionNone` (catch-all fires).
11. Write `TestRules_OrderAndCompleteness` — calls `Rules()`, asserts length is 6, asserts the last rule name is `"none"` (catch-all is last), asserts all 6 Action constants appear in the rules.
12. Write `TestDispatch_EmptyState` — creates `State{}` (zero value, Action is empty string), asserts `Dispatch` returns `ActionNone`.
13. Run `gofumpt -w internal/auto/dispatch_test.go` to format.
14. Run `go test ./internal/auto/ -run TestDispatch -v -count=1` — all tests pass.
15. Run `go test ./internal/auto/ -v -count=1` — all tests pass (no regressions, existing 30 tests + new dispatch tests).

## Constraints
- All tests must use `t.Parallel()` per project convention
- Use `testify/require` for assertions, not `testing.T` methods
- Tests are pure function tests — no DB, no setupTestDB needed
- State structs in tests should have realistic pointer values (use `&Milestone{ID: "M001"}` etc.) to confirm dispatch doesn't depend on pointer contents, only on Action field

## Inputs

- ``internal/auto/dispatch.go` — Rule struct, rules table, Dispatch() and Rules() functions from T01`
- ``internal/auto/state.go` — State struct and Action constants for constructing test inputs`
- ``internal/auto/status.go` — Status/Phase enums for constructing realistic State values`

## Expected Output

- ``internal/auto/dispatch_test.go` — 10+ test functions covering all 6 Action constants, nil state, empty state, unknown action fallback, rules ordering and completeness`

## Verification

go test ./internal/auto/ -run TestDispatch -v -count=1 && go test ./internal/auto/ -v -count=1 && go vet ./internal/auto/...
