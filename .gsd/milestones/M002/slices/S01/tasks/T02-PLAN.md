---
estimated_steps: 44
estimated_files: 4
skills_used: []
---

# T02: Engine core — loop, lock file, pause/stop signaling, and session management

Build the Engine struct that runs the auto loop: derive state → create child session → dispatch to Coordinator → advance DB status → publish events. Includes lock file management and pause/stop signaling.

## Failure Modes

| Dependency | On error | On timeout | On malformed response |
|------------|----------|-----------|----------------------|
| Coordinator.RunWithForcedTier | Log error, publish UnitFailed event, do NOT advance status, continue to next iteration | Context cancellation propagates, engine pauses cleanly | Treat as error — log and retry next iteration |
| session.Service.CreateTaskSession | Return error, halt current unit, retry next iteration | Same as error | N/A — typed interface |
| DB queries (GetMilestone, UpdateStatus) | Return error, halt current unit, retry next iteration | Same as error | N/A — typed interface |

## Steps

1. Create `internal/auto/lock.go` with LockFile struct: Acquire(dir string) error, Release() error, IsStale() bool. Lock file at `<dir>/auto.lock` contains JSON `{"pid": <int>, "started_at": <timestamp>}`. Acquire fails if lock exists and PID is alive. Stale detection checks if PID is still running via `os.FindProcess` + signal 0.

2. Create `internal/auto/lock_test.go` testing: acquire/release, double-acquire fails, stale lock detection, concurrent acquire races.

3. Create `internal/auto/engine.go` with Engine struct:
   - Constructor: `NewEngine(db *db.Queries, coordinator agent.Coordinator, sessions session.Service, broker *pubsub.Broker[AutoEvent], dataDir string) *Engine`
   - `Run(ctx context.Context, milestoneID string) error` — acquires lock, enters loop: DeriveState → create child session → build prompt → dispatch via Coordinator.RunWithForcedTier → advance status → publish event. Checks pause/stop signals between iterations. Releases lock on exit.
   - `Step(ctx context.Context, milestoneID string) error` — runs exactly one unit then returns.
   - `Pause()` — sets atomic flag; loop finishes current unit then exits.
   - `Stop()` — cancels context immediately.
   - `Status() EngineStatus` — returns current state (running/paused/idle, active unit, milestone progress).
   - Parent session: on first Run(), create a parent session for the milestone. Child sessions created per unit via session.Service.CreateTaskSession().
   - Model tier selection: research/planning units use planning tier, execution uses main tier, summarize/validate use background tier.
   - Auto-approve permissions for all child sessions.

4. Create `internal/auto/engine_test.go` with tests using mock Coordinator and mock session.Service:
   - Test Run() advances through a seeded task sequence
   - Test Pause() finishes current unit then stops
   - Test Step() executes exactly one unit
   - Test lock file prevents concurrent Run()
   - Test resume from DB state (simulate kill + restart)
   - Test event publishing (subscribe to broker, verify events)

5. Run `go test ./internal/auto/... -count=1` and `go vet ./internal/auto/...`.

## Must-Haves

- [ ] Engine.Run() loops: derive → dispatch → advance → publish until done or paused
- [ ] Engine.Step() executes exactly one unit then returns
- [ ] Engine.Pause() finishes current unit before stopping (D002)
- [ ] Lock file prevents concurrent instances with stale PID detection
- [ ] Each dispatched unit gets a fresh child session under the milestone parent session (R005)
- [ ] Events published for UnitStarted, UnitCompleted, UnitFailed, LoopPaused, LoopStopped (R017)
- [ ] Model tier selection: planning for research/plan, main for execute, background for summarize/validate
- [ ] Tests cover run, pause, step, lock, resume, events

## Verification

- `go test ./internal/auto/... -count=1 -v` — all engine and lock tests pass
- `go vet ./internal/auto/...` — no vet errors

## Observability Impact

- Signals added: AutoEvent published via pubsub.Broker on every state transition, unit start/complete/fail
- How a future agent inspects this: subscribe to Broker[AutoEvent], or call Engine.Status() for snapshot
- Failure state exposed: UnitFailed events carry error details; Engine.Status() reports last error and active unit

## Inputs

- `internal/auto/state.go`
- `internal/auto/unit.go`
- `internal/auto/events.go`
- `internal/auto/status.go`
- `internal/auto/milestone.go`
- `internal/auto/slice.go`
- `internal/auto/task.go`
- `internal/session/session.go`
- `internal/agent/coordinator.go`
- `internal/pubsub/broker.go`
- `internal/db/sql/milestones.sql`
- `internal/db/sql/slices.sql`
- `internal/db/sql/tasks.sql`

## Expected Output

- `internal/auto/engine.go`
- `internal/auto/engine_test.go`
- `internal/auto/lock.go`
- `internal/auto/lock_test.go`

## Verification

go test ./internal/auto/... -count=1 && go vet ./internal/auto/...
