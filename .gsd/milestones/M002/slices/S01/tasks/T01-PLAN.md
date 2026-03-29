---
estimated_steps: 21
estimated_files: 4
skills_used: []
---

# T01: State derivation, auto event types, and dispatch unit model

Build the pure-logic layer that examines DB state and determines what to dispatch next. This is the foundation everything else builds on — the engine calls DeriveState() each iteration to get the next unit.

## Steps

1. Create `internal/auto/unit.go` with UnitType enum (research, plan_slice, execute_task, summarize_slice, validate_milestone) and a Unit struct containing the milestone/slice/task IDs, unit type, and phase context needed for dispatch.

2. Create `internal/auto/events.go` with AutoEvent struct and typed EventType constants: UnitStarted, UnitCompleted, UnitFailed, LoopPaused, LoopStopped, StateTransition. AutoEvent carries UnitType, milestone/slice/task IDs, optional error, and timestamp. This integrates with the existing `pubsub.Broker[AutoEvent]`.

3. Create `internal/auto/state.go` with a `DeriveState(ctx, db.Queries) (Unit, error)` function. Logic:
   - Find the active milestone (status=active). If none, return done.
   - Find slices for that milestone ordered by sort_order. Find the first non-completed slice whose dependencies are met.
   - Check the slice's phase: if pre_planning → return research unit; if planning → return plan_slice unit; if executing → find first non-completed task and return execute_task unit; if summarizing → return summarize unit; if validating → return validate unit.
   - Respect `depends_on` — a slice can only start if all slices it depends on are completed.
   - Return a sentinel `Unit{}` with `UnitType=""` when nothing is dispatchable (all done or blocked).

4. Create `internal/auto/state_test.go` with comprehensive tests: single task dispatch, multi-task ordering, slice dependency blocking, phase progression through research→plan→execute→summarize→validate, all-done detection, blocked detection.

5. Run `go test ./internal/auto/... -count=1` and `go vet ./internal/auto/...` to verify.

## Must-Haves

- [ ] UnitType enum covers all dispatch phases (research, plan_slice, execute_task, summarize_slice, validate_milestone)
- [ ] Unit struct carries enough context for engine dispatch (milestone/slice/task IDs, type, descriptive title)
- [ ] AutoEvent struct with typed EventType constants for pubsub integration
- [ ] DeriveState correctly walks milestone→slice→task hierarchy respecting status, phase, sort_order, and depends_on
- [ ] Tests cover: single dispatch, ordering, dependency blocking, phase progression, all-done, empty DB

## Verification

- `go test ./internal/auto/... -count=1 -v` — all state derivation tests pass
- `go vet ./internal/auto/...` — no vet errors

## Inputs

- `internal/auto/status.go`
- `internal/auto/milestone.go`
- `internal/auto/slice.go`
- `internal/auto/task.go`
- `internal/db/sql/milestones.sql`
- `internal/db/sql/slices.sql`
- `internal/db/sql/tasks.sql`
- `internal/pubsub/events.go`

## Expected Output

- `internal/auto/unit.go`
- `internal/auto/events.go`
- `internal/auto/state.go`
- `internal/auto/state_test.go`

## Verification

go test ./internal/auto/... -count=1 && go vet ./internal/auto/...
