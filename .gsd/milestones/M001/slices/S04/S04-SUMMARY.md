---
id: S04
parent: M001
milestone: M001
provides:
  - Dispatch(state *State) Action — maps any derived State to the correct next action
  - Rule struct and Rules() — introspectable dispatch table for debugging and testing
requires:
  - slice: S03
    provides: State struct and Action constants consumed by dispatch rules
affects:
  - S05
key_files:
  - internal/auto/dispatch.go
  - internal/auto/dispatch_test.go
key_decisions:
  - Match closures read State.Action directly, keeping rules extensible for future multi-field conditions
  - Rules stored as var (not const) — Go slice of structs cannot be const
  - Added TestRules_ShallowCopy beyond plan to verify Rules() shallow-copy contract
patterns_established:
  - Declarative rules table pattern: slice of Rule structs with Match/Action, evaluated top-down with catch-all last
  - Rules() returns shallow copy for safe introspection without exposing mutable internal state
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M001/slices/S04/tasks/T01-SUMMARY.md
  - .gsd/milestones/M001/slices/S04/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T19:00:01.671Z
blocker_discovered: false
---

# S04: Dispatch Rules Table

**Declarative dispatch rules table mapping any State to the correct next action via top-down Rule evaluation, with full test coverage including edge cases**

## What Happened

T01 created internal/auto/dispatch.go with the Rule struct (Name string, Match func(*State) bool, Action), an ordered rules slice of 6 entries (execute-task, plan-slice, plan-milestone, complete-slice, complete-milestone, none catch-all), a nil-safe Dispatch(state *State) Action function that walks rules top-down and returns the first match, and a Rules() function returning a shallow copy for introspection. Match closures read state.Action directly, keeping them extensible for future multi-field conditions (e.g., checking Phase).

T02 created internal/auto/dispatch_test.go with 11 parallel test functions: 7 per-action tests, 2 edge-case tests (nil state, zero-value state), 1 fallback test (unknown action string hits catch-all), 1 rules introspection test (order, completeness, all 6 actions present), and 1 shallow-copy safety test beyond the original plan. All use t.Parallel() and testify/require per project convention.

The full auto package now has 42 tests spanning schema (S01), domain model (S02), state derivation (S03), and dispatch (S04) — all pass with clean build and vet.

## Verification

go build ./internal/auto/... — exit 0. go vet ./internal/auto/... — exit 0. go test ./internal/auto/ -v -count=1 — 42/42 PASS (11 dispatch + 14 state derivation + 17 domain model). grep confirms Dispatch, Rule struct, and Rules function present in dispatch.go.

## Requirements Advanced

- R003 — Implemented as Go slice of Rule structs with condition func → action, evaluated top-down by Dispatch()

## Requirements Validated

- R003 — 6 ordered rules cover all State→Action mappings. 11 test functions prove every rule fires correctly, edge cases handled, ordering respected, and Rules() returns safe copy.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

T01: gofumpt not on PATH, used gofmt as fallback (no functional difference). T02: Added TestRules_ShallowCopy beyond plan to verify Rules() returns an independent copy.

## Known Limitations

Dispatch currently only inspects State.Action field. Future rules that need multi-field conditions (e.g., phase-aware routing) will need Match closures updated, but the Rule struct already supports this without structural changes.

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/dispatch.go` — New file: Rule struct, 6 ordered rules, Dispatch() function, Rules() introspection helper
- `internal/auto/dispatch_test.go` — New file: 11 parallel test functions covering all rules, edge cases, ordering, and shallow-copy safety
