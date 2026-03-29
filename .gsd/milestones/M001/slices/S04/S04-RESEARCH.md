# S04: Dispatch Rules Table — Research

**Date:** 2026-03-27
**Depth:** Light — well-understood pattern, all inputs exist from S03

## Summary

S04 builds a declarative dispatch rules table: a Go slice of `Rule` structs evaluated top-down, where each rule has a condition function (`func(*State) bool`) and an associated action or handler. Given any `*State` returned by `DeriveState()`, the rules table returns the matching dispatch instruction. This is a textbook table-driven dispatch pattern with no external dependencies, no new DB work, and no unfamiliar technology.

The `Action` enum (6 constants) and `State` struct from S03 provide the complete input surface. The rules table simply maps `State` patterns to dispatch decisions. The primary risk is rule ordering — wrong order means silent wrong behavior — which is addressed through comprehensive test coverage.

## Recommendation

Implement as a single new file `internal/auto/dispatch.go` with:
1. A `Rule` struct: `Name string`, `Match func(*State) bool`, `Action Action`
2. An exported `Rules` variable (or function returning `[]Rule`) with rules ordered from most-specific to least-specific
3. A `Dispatch(state *State) Action` function that walks rules top-down and returns the first match
4. Comprehensive tests in `internal/auto/dispatch_test.go` covering every Action constant and edge cases (nil state, no-match fallback)

Keep it simple — the rules table is a pure function of `*State`, no DB access needed. The `Dispatch` function is the public API; `Rules` can be exported for introspection/testing but the primary interface is `Dispatch`.

## Implementation Landscape

### Key Files

- `internal/auto/state.go` — Defines `Action` (6 constants), `State` struct with `Action`, `Milestone`, `Slice`, `Task` fields. This is the input to dispatch. **Read-only for S04.**
- `internal/auto/status.go` — `Status` and `Phase` enums used in condition matching. **Read-only for S04.**
- `internal/auto/dispatch.go` — **New file.** Rule struct, rules table, `Dispatch()` function.
- `internal/auto/dispatch_test.go` — **New file.** Tests for every rule, ordering, edge cases.

### Build Order

1. **T01: Create `dispatch.go`** — Define `Rule` struct and `Dispatch()` function with the rules table. Rules map each `State.Action` value to the dispatch result. Since `DeriveState` already computes the `Action` field, the simplest correct dispatch is to route on `State.Action`. However, R003 specifies "condition func → action" — so rules should use `Match` functions that inspect `State` fields (Action, Phase, Status of nested structs) rather than just forwarding `State.Action`. This makes the rules table extensible for future conditions (e.g., "if task is in validating phase, dispatch to validator instead of executor").

   The rules table ordering (most-specific first):
   - `ActionExecuteTask` — task ready for execution
   - `ActionPlanSlice` — slice needs planning  
   - `ActionPlanMilestone` — milestone needs planning
   - `ActionCompleteSlice` — all tasks done, slice needs wrap-up
   - `ActionCompleteMilestone` — all slices done, milestone needs wrap-up
   - `ActionNone` — nothing to do (fallback)

2. **T02: Create `dispatch_test.go`** — Test each rule individually by constructing a `State` that triggers it. Test ordering by constructing ambiguous states. Test edge cases: nil state, State with ActionNone, State with unknown action. Reuse no DB setup — this is pure function testing on `State` values.

### Verification Approach

```bash
# Build compiles
go build ./internal/auto/...

# Vet passes
go vet ./internal/auto/...

# All dispatch tests pass
go test ./internal/auto/ -run TestDispatch -v -count=1

# No regressions in existing tests
go test ./internal/auto/ -v -count=1

# Confirm Dispatch function exists
grep 'func Dispatch' internal/auto/dispatch.go
```

## Constraints

- `CGO_ENABLED=0` — no C dependencies (already satisfied, pure Go)
- Must follow existing `internal/auto/` package patterns: typed string constants, `t.Parallel()` in tests, `testify/require`
- Rules table must be declarative (R003: "Go slice of rule structs") not a switch statement

## Common Pitfalls

- **Rule ordering producing silent wrong behavior** — mitigate with tests that assert specific dispatch results for every Action constant, plus a test that verifies rules are evaluated top-down by putting a catch-all rule last
- **Forgetting the nil-State edge case** — `Dispatch(nil)` should return `ActionNone` safely, not panic
