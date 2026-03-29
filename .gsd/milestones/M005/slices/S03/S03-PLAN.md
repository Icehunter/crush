# S03: TUI Integration Wiring

**Goal:** The TUI is wired to the real auto-mode engine: ctrl+a triggers engine start/pause/resume, the engine publishes events with populated AutoSnapshot data, and the sidebar renders live progress from real DB state.
**Demo:** After this: # S03: TUI Integration Wiring — UAT

**Milestone:** M005
**Written:** 2026-03-28T10:50:47.257Z

# S03: TUI Integration Wiring — UAT

**Milestone:** M005
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: All changes are compile-time wiring and data-layer queries — verification is compilation, unit tests, and integration tests. No runtime UI rendering to observe.

## Preconditions

- M005 worktree checked out with S01+S02+S03 changes
- Go toolchain available (go 1.24+)

## Smoke Test

Run `go build ./...` — must exit 0. Then `go test ./internal/auto/... -count=1` — must pass all tests.

## Test Cases

### 1. Full compilation gate

1. Run `go build ./...`
2. **Expected:** Exit code 0, no errors.

### 2. Static analysis gate

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no new warnings.

### 3. BuildSnapshot with populated DB

1. Run `go test ./internal/auto/... -count=1 -run TestBuildSnapshot -v`
2. **Expected:** Test passes — BuildSnapshot returns AutoSnapshot with correct SliceProgress entries, TasksDone/TasksTotal counts matching inserted data.

### 4. BuildSnapshot with empty DB

1. Run `go test ./internal/auto/... -count=1 -run TestBuildSnapshot_Empty -v`
2. **Expected:** Test passes — returns snapshot with empty Slices slice, no errors.

### 5. EngineController satisfies AutoController interface

1. Run `go test ./internal/auto/... -count=1 -run TestEngineController_Interface -v`
2. **Expected:** Compile-time interface satisfaction check passes.

### 6. EngineController AutoStatus reflects engine state

1. Run `go test ./internal/auto/... -count=1 -run TestEngineController_AutoStatus -v`
2. **Expected:** AutoStatus returns "idle" for newly-constructed engine.

### 7. Published events carry snapshots

1. Run `go test ./internal/auto/... -count=1 -run TestPublishWithSnapshot -v`
2. **Expected:** Subscribed event has non-nil Snapshot with correct milestone ID and slice progress.

### 8. Integration — full pipeline with real DB

1. Run `go test ./internal/auto/... -count=1 -run TestControllerIntegration_FullPipeline -v`
2. **Expected:** Engine step publishes event, received event has non-nil Snapshot with milestone data from real SQLite.

### 9. Integration — multiple events carry snapshots

1. Run `go test ./internal/auto/... -count=1 -run TestControllerIntegration_MultipleEvents -v`
2. **Expected:** Multiple engine steps each produce events with non-nil snapshots.

### 10. UI model setter methods exist

1. Run `go test ./internal/ui/model/... -count=1`
2. **Expected:** All model tests pass — SetAutoController and SetAutoMilestoneID don't break existing behavior.

### 11. App.Queries field exposed

1. Run `grep -q 'Queries \*db.Queries' internal/app/app.go`
2. **Expected:** Exit 0 — field is present on App struct.

### 12. Full adapter chain in cmd/root.go

1. Run `grep -c 'auto.New' internal/cmd/root.go`
2. **Expected:** Count ≥ 5 (NewDBStateQuerier, NewDBStatusAdvancer, NewDBTokenQuerier, NewSessionServiceCreator, NewCoordinatorDispatcher or NewDBBudgetChecker, NewEngine, NewEngineController).

### 13. Existing auto test suite regression check

1. Run `go test ./internal/auto/... -count=1`
2. **Expected:** All tests pass — no regressions from S03 changes.

## Edge Cases

### snapshotQuerier is nil

1. Covered by all pre-existing engine tests which pass nil for snapshotQuerier.
2. **Expected:** publish() skips snapshot attachment, no panic.

### App.Queries is nil (no DB available)

1. Covered by nil guard in cmd/root.go — adapter chain is not constructed when Queries is nil.
2. **Expected:** App starts normally without auto-mode wiring.

## Failure Signals

- `go build ./...` fails → import cycle or type mismatch in adapter chain
- TestBuildSnapshot fails → slice/task query logic broken, sidebar will show wrong progress
- TestPublishWithSnapshot fails → events won't carry snapshots, sidebar won't update
- TestControllerIntegration fails → real adapters don't compose with engine correctly
- `go test ./internal/ui/model/...` fails → setter methods broke existing UI model

## Not Proven By This UAT

- Actual TUI sidebar rendering with live data (visual — requires running the app)
- Real CoordinatorDispatcher with LLM sessions (deferred to S05)
- CLI `crush auto start` command triggering the controller (deferred to S04)
- Runtime ctrl+a keybinding behavior (TUI visual test, not automated)


## Tasks
- [x] **T01: Added BuildSnapshot function, EngineController implementing AutoController, and enriched engine.publish() to attach real DB-backed snapshots to all auto-mode events** — Create `internal/auto/controller.go` containing:

1. `BuildSnapshot(ctx, querier StateQuerier, milestoneID, status, activeUnit string, totalCost, elapsed float64) *AutoSnapshot` — queries ListSlicesByMilestone then ListTasksBySlice for each slice to build SliceProgress entries with TasksDone/TasksTotal counts. Gets milestone title from the first result or milestoneID.

2. `EngineController` struct that implements `model.AutoController`:
   - Holds `*Engine`, `StateQuerier`, and milestone tracking fields
   - `StartAuto(ctx, milestoneID)` → stores milestoneID, calls `Engine.Run(ctx, milestoneID)` in a goroutine, returns nil or error from validation
   - `PauseAuto()` → calls `Engine.Pause()`
   - `ResumeAuto(ctx)` → calls `Engine.Run(ctx, storedMilestoneID)` in a goroutine (Run() is re-entrant after pause since pause makes Run() return)
   - `AutoStatus()` → returns `string(Engine.Status().State)`

3. Modify `engine.publish()` to build and attach a snapshot. Add a `snapshotQuerier` field to Engine (type StateQuerier, optional/nil-safe). When non-nil, publish calls `BuildSnapshot` and sets `event.Snapshot` before publishing.

4. Update `NewEngine()` to accept the optional `snapshotQuerier StateQuerier` parameter (add it to the constructor). Update all existing NewEngine call sites in test files to pass nil for the new parameter.

5. Create `internal/auto/controller_test.go` with:
   - `TestBuildSnapshot` — uses setupAdapterTestDB (from adapters_test.go) to insert milestone+slices+tasks, calls BuildSnapshot, asserts correct SliceProgress counts
   - `TestBuildSnapshot_Empty` — empty DB returns snapshot with empty slices
   - `TestEngineController_Interface` — compile-time interface satisfaction check
   - `TestEngineController_AutoStatus` — creates EngineController with a real Engine (idle), verifies AutoStatus returns 'idle'
   - `TestPublishWithSnapshot` — creates engine with snapshotQuerier, publishes an event, subscribes and verifies snapshot is populated
  - Estimate: 1h30m
  - Files: internal/auto/controller.go, internal/auto/controller_test.go, internal/auto/engine.go
  - Verify: go build ./... && go vet ./... && go test ./internal/auto/... -count=1 -run 'TestBuildSnapshot|TestEngineController|TestPublishWithSnapshot' -v
- [x] **T02: Wired production EngineController into TUI via App.Queries, SetAutoController/SetAutoMilestoneID methods, and full adapter chain in cmd/root.go** — Connect the production EngineController to the TUI so ctrl+a triggers the real engine:

1. Expose DB queries on App: add a `Queries *db.Queries` field to `App` struct in `internal/app/app.go`. Set it in `app.New()` from the existing `q := db.New(conn)` line. This gives downstream code access to DB queries for adapter construction.

2. Add a `SetAutoController(c AutoController)` method to `*UI` in `internal/ui/model/ui.go`. It sets `m.autoController = c`. Also add `SetAutoMilestoneID(id string)` to set `m.autoMilestoneID`.

3. Wire in `internal/cmd/root.go` after `model := ui.New(com, sessionID, continueLast)`:
   - Import `auto` and `app` packages
   - Construct adapters: `querier := auto.NewDBStateQuerier(com.App.Queries)`, `sessions := auto.NewSessionServiceCreator(com.App.Sessions)`, `dispatcher := auto.NewCoordinatorDispatcher(com.App.AgentCoordinator)`, `advancer := auto.NewDBStatusAdvancer(com.App.Queries)`, `tokenQuerier := auto.NewDBTokenQuerier(com.App.Queries)`, `budgetChecker := auto.NewDBBudgetChecker(com.App.Queries)` (if it exists, else nil)
   - Construct engine: `engine := auto.NewEngine(querier, sessions, dispatcher, advancer, nil, budgetChecker, cfg.BudgetCeiling, nil, nil, app.AutoBroker(), dataDir, slog.Default(), querier)` where the last param is the snapshotQuerier
   - Construct controller: `ctrl := auto.NewEngineController(engine, querier)`
   - Inject: `model.SetAutoController(ctrl)`
   - If `com.App.Config().Auto != nil`, check for a milestone ID config field; if present, `model.SetAutoMilestoneID(id)` — for now, auto.milestone_id doesn't exist in config, so this is a TODO for S04

4. Create `internal/auto/controller_integration_test.go` with a test that:
   - Sets up a real DB with milestone/slice/task data
   - Constructs a full EngineController with DBStateQuerier as snapshotQuerier
   - Subscribes to the broker
   - Calls `engine.Step()` (which will publish events)
   - Verifies the received event has a non-nil Snapshot with correct milestone data

5. Run full verification: `go build ./...`, `go vet ./...`, `go test ./internal/auto/... -count=1`, `go test ./internal/ui/model/... -count=1`
  - Estimate: 1h
  - Files: internal/app/app.go, internal/ui/model/ui.go, internal/cmd/root.go, internal/auto/controller_integration_test.go
  - Verify: go build ./... && go vet ./... && go test ./internal/auto/... -count=1 -v && go test ./internal/ui/model/... -count=1
