# S04: Dispatch Rules Table — UAT

**Milestone:** M001
**Written:** 2026-03-27T19:00:01.672Z

# S04: Dispatch Rules Table — UAT

**Milestone:** M001
**Written:** 2026-03-27

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This slice produces Go source code and tests with no runtime behavior — compilation and test execution prove correctness

## Preconditions

- Go toolchain installed (go 1.24+)
- Working directory is the crush project root
- S01–S03 already in place (DB schema, domain model, state derivation)

## Smoke Test

Run `go test ./internal/auto/ -run TestDispatch -v -count=1` — all 11 dispatch tests should pass.

## Test Cases

### 1. Nil State Returns ActionNone

1. Run `go test ./internal/auto/ -run TestDispatch_NilState -v`
2. **Expected:** PASS — Dispatch(nil) returns ActionNone without panic

### 2. Each Action Constant Dispatches Correctly

1. Run `go test ./internal/auto/ -run "TestDispatch_ExecuteTask|TestDispatch_PlanSlice|TestDispatch_PlanMilestone|TestDispatch_CompleteSlice|TestDispatch_CompleteMilestone|TestDispatch_ActionNone" -v`
2. **Expected:** All 6 PASS — each Action constant maps to the correct rule

### 3. Unknown Action Falls Back to None

1. Run `go test ./internal/auto/ -run TestDispatch_FallbackToNone -v`
2. **Expected:** PASS — unrecognized action string ("unknown") hits catch-all, returns ActionNone

### 4. Empty/Zero-Value State Returns ActionNone

1. Run `go test ./internal/auto/ -run TestDispatch_EmptyState -v`
2. **Expected:** PASS — State{} (zero value, empty Action string) returns ActionNone

### 5. Rules Order and Completeness

1. Run `go test ./internal/auto/ -run TestRules_OrderAndCompleteness -v`
2. **Expected:** PASS — Rules() returns 6 rules, last rule is "none" catch-all, all 6 Action constants appear

### 6. Rules Returns Shallow Copy

1. Run `go test ./internal/auto/ -run TestRules_ShallowCopy -v`
2. **Expected:** PASS — mutating returned slice does not affect internal rules

### 7. No Regressions in Full Auto Package

1. Run `go test ./internal/auto/ -v -count=1`
2. **Expected:** All 42 tests PASS (11 dispatch + 14 state derivation + 17 domain model)

### 8. Build and Vet Clean

1. Run `go build ./internal/auto/...`
2. Run `go vet ./internal/auto/...`
3. **Expected:** Both exit code 0, no output

## Edge Cases

### Nil State Safety

1. Call Dispatch(nil)
2. **Expected:** Returns ActionNone immediately, no nil-pointer dereference

### Zero-Value State

1. Call Dispatch(&State{}) where Action is empty string
2. **Expected:** Falls through to catch-all, returns ActionNone

### Unrecognized Action String

1. Call Dispatch with State{Action: Action("unknown")}
2. **Expected:** No rule matches except catch-all → ActionNone

## Failure Signals

- Any of the 11 dispatch tests fail
- `go build ./internal/auto/...` fails with type errors
- `go vet ./internal/auto/...` reports issues
- Full suite regression: fewer than 42 tests pass

## Not Proven By This UAT

- Runtime integration of Dispatch with the execution loop (S05 scope)
- Performance under high call frequency
- Concurrent access to Dispatch from multiple goroutines (safe since rules is read-only, but untested with -race)
- Integration with CLI commands (M002 scope)

## Notes for Tester

- All tests use t.Parallel() — safe to run with -race flag
- Tests are pure function tests — no DB, no network, no setup needed
- State structs in tests use realistic pointer values to confirm dispatch depends only on the Action field, not pointer contents
