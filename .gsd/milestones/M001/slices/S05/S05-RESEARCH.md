# S05: Integration Proof — Research

**Date:** 2026-03-27

## Summary

S05 proves the full derive → dispatch cycle works end-to-end with real DB state. All building blocks exist and are individually tested: `DeriveState()` (14 tests in state_test.go), `Dispatch()` (11 tests in dispatch_test.go), domain model (17 tests in model_test.go/status_test.go). What's missing is a single integration test that seeds a multi-slice milestone, then steps through the entire lifecycle calling `DeriveState()` → `Dispatch()` at each stage and asserting the action sequence matches expectations.

This is light, low-risk work — no new production code, just one test file that composes existing functions against a real in-memory SQLite database.

## Recommendation

Write a single `TestDeriveDispatchCycle` integration test (plus a few targeted scenario tests) in a new `internal/auto/integration_test.go` file. The main test seeds a milestone with 2 slices (S01 depends on nothing, S02 depends on S01), each with 2 tasks. It then walks the lifecycle: plan milestone → plan S01 → execute T01 → execute T02 → complete S01 → plan S02 → execute T03 → execute T04 → complete S02 → complete milestone. At each step, call `DeriveState()` then `Dispatch()`, assert the action, then mutate DB state to advance. This proves the full cycle produces the correct action sequence.

Additional scenario tests should cover: empty DB cycle (derive+dispatch returns none), dependency gating mid-cycle (blocked slice skipped until dependency completes), and the all-completed terminal state.

## Implementation Landscape

### Key Files

- `internal/auto/state.go` — `DeriveState(ctx, q)` returns `*State` with Action, Milestone, Slice, Task. No changes needed.
- `internal/auto/dispatch.go` — `Dispatch(state)` returns `Action`. No changes needed.
- `internal/auto/state_test.go` — Contains `setupTestDB()`, `seedMilestone()`, `seedSlice()`, `seedTask()` helpers. These are reused directly by the integration test.
- `internal/auto/status.go` — Status/Phase constants (StatusPending, StatusActive, StatusCompleted, PhasePrePlanning, PhaseExecuting, etc.). No changes needed.
- `internal/auto/integration_test.go` — **New file.** All integration tests live here.

### Build Order

1. Write the integration test file with the full lifecycle test first — it's the primary deliverable and the riskiest (most complex seeding/assertion logic).
2. Add edge-case scenario tests (empty DB, dependency gating, terminal state) — these are simpler and reinforce coverage.
3. Verify: `go test ./internal/auto/ -v -count=1 -run TestIntegration` passes, then full suite still passes.

### Verification Approach

- `go build ./internal/auto/...` — compiles cleanly
- `go vet ./internal/auto/...` — no issues
- `go test ./internal/auto/ -v -count=1` — all tests pass (existing 42 + new integration tests)
- The lifecycle test explicitly asserts the action sequence: `[plan_milestone, plan_slice, execute_task, execute_task, complete_slice, plan_slice, execute_task, execute_task, complete_slice, complete_milestone]`

## Constraints

- Test helpers (`setupTestDB`, `seedMilestone`, `seedSlice`, `seedTask`) are unexported — integration test must be in `package auto` (same package), which it will be.
- DB state mutations between steps use the existing SQLC update queries (e.g., `q.UpdateMilestone`, `q.UpdateSlice`, `q.UpdateTask`). Need to confirm these exist.
- `CGO_ENABLED=0` — already working with modernc.org/sqlite in existing tests.

## Common Pitfalls

- **Forgetting to advance DB state between derive calls** — each step must mutate the DB (e.g., set task status to completed) before the next `DeriveState()` call, otherwise the same action repeats forever.
- **Slice dependency ordering** — S02 depends on S01. The test must verify S02 is not actionable until S01 is completed. The existing `seedSlice` helper supports `dependsOn` parameter.
