# M005: M005:

## Vision
M005:

## Slice Overview
| ID | Slice | Risk | Depends | Done | After this |
|----|-------|------|---------|------|------------|
| S01 | Branch Reconciliation | high | — | ✅ | # S01: Branch Reconciliation — UAT

**Milestone:** M005
**Written:** 2026-03-28T09:33:42.476Z

# S01: Branch Reconciliation — UAT

**Milestone:** M005
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This is a code merge/reconciliation slice — all verification is compilation and test execution, no runtime behavior to observe.

## Preconditions

- M005 worktree checked out at the post-merge commit
- Go toolchain available (go 1.24+)

## Smoke Test

Run `go build ./...` — must exit 0 with no errors. This confirms all M001–M003 code compiles together on the M005 branch.

## Test Cases

### 1. Full compilation gate

1. Run `go build ./...`
2. **Expected:** Exit code 0, no compilation errors.

### 2. Static analysis gate

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no vet warnings.

### 3. Engine test suite

1. Run `go test ./internal/auto/... -count=1`
2. **Expected:** All tests pass (100+ tests covering engine loop, state machine, safety rails, events, worktree).

### 4. Config parsing tests

1. Run `go test ./internal/config/... -run TestAutoConfig`
2. **Expected:** 3 tests pass (TestAutoConfig, TestAutoConfig_Empty, TestAutoConfig_PartialFields).

### 5. Event broker tests

1. Run `go test ./internal/app/... -run TestAutoEvent`
2. **Expected:** 3 tests pass confirming pubsub event delivery.

### 6. Database tests

1. Run `go test ./internal/db/... -count=1`
2. **Expected:** All DB tests pass including SumChildSessionCosts.

### 7. ReasoningEffort preservation

1. Run `grep -c ReasoningEffort internal/agent/coordinator.go`
2. **Expected:** Count > 0 (confirms M003 regression was not applied).

### 8. Duplicate event file deleted

1. Run `ls internal/auto/event.go`
2. **Expected:** "No such file or directory" — the duplicate file was removed.

### 9. M003 engine files present

1. Run `ls internal/auto/engine.go internal/auto/state.go internal/auto/stuck.go internal/auto/events.go`
2. **Expected:** All four files listed without error.

### 10. Unified event struct has TUI fields

1. Run `grep 'Snapshot \*AutoSnapshot' internal/auto/events.go`
2. **Expected:** Match found — the unified AutoEvent struct includes the TUI Snapshot field.

## Edge Cases

### Empty auto config

1. Run `go test ./internal/config/... -run TestAutoConfig_Empty`
2. **Expected:** Test passes — crush.json with no `auto` section produces zero-value config without errors.

## Failure Signals

- `go build ./...` fails with duplicate symbol errors → event unification incomplete
- `grep ReasoningEffort internal/agent/coordinator.go` returns 0 → M003 regression was applied
- `ls internal/auto/event.go` succeeds → duplicate file not deleted, will cause compile errors
- Any test suite exits non-zero → merge introduced regressions

## Not Proven By This UAT

- Runtime behavior of the engine loop (no actual LLM calls or auto-mode execution)
- Production DB adapter wiring (deferred to S02)
- TUI sidebar rendering with live auto-mode data (deferred to S03)
- CLI command integration (deferred to S04)

## Notes for Tester

- The `go vet` check may surface pre-existing warnings from main (see K005). As long as no NEW warnings were introduced by the merge, the gate passes.
- `internal/db/auto_test.go` exists and passes — the plan speculated it should be deleted but it contains valid tests.
 |
| S02 | Production DB Adapters + Engine Wiring | high | S01 | ✅ | # S02: Production DB Adapters + Engine Wiring — UAT

**Milestone:** M005
**Written:** 2026-03-28T10:25:08.898Z

# S02: Production DB Adapters + Engine Wiring — UAT

**Milestone:** M005
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: All adapters are pure data-layer wrappers — verification is compilation, unit tests, and integration tests against real SQLite. No runtime UI or network behavior to observe.

## Preconditions

- M005 worktree checked out with S01+S02 changes
- Go toolchain available (go 1.24+)
- SQLite support (CGO_ENABLED=0 with modernc driver)

## Smoke Test

Run `go build ./...` — must exit 0. Then `go test ./internal/auto/... -count=1` — must pass all tests including adapter and integration suites.

## Test Cases

### 1. Full compilation gate

1. Run `go build ./...`
2. **Expected:** Exit code 0, no errors.

### 2. Static analysis gate

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no new warnings.

### 3. Adapter unit tests

1. Run `go test ./internal/auto/... -count=1 -run 'TestDB|TestSession|TestCoordinator' -v`
2. **Expected:** 5 test functions pass with 14 subtests:
   - TestDBStateQuerier (5 subtests: ListMilestones, ListSlicesByMilestone, ListTasksBySlice, empty variants)
   - TestDBStatusAdvancer (5 subtests: one per unit type verifying phase transitions)
   - TestDBTokenQuerier (1 subtest: token sum verification)
   - TestSessionServiceCreator (2 subtests: create + create child)
   - TestCoordinatorDispatcher (1 subtest: call passthrough + error propagation)

### 4. DBStatusAdvancer phase transitions match DeriveState

1. Run `go test ./internal/auto/... -count=1 -run TestDBStatusAdvancer -v`
2. **Expected:** Each unit type produces the correct phase transition:
   - UnitResearch → slice phase = 'planning'
   - UnitPlanSlice → slice phase = 'executing'
   - UnitExecuteTask → task status+phase = 'completed'
   - UnitSummarizeSlice → slice status+phase = 'completed'
   - UnitValidateMilestone → milestone status+phase = 'completed'

### 5. Integration test — single task lifecycle

1. Run `go test ./internal/auto/... -count=1 -run TestIntegration_StepAdvancesTaskThroughRealDB -v`
2. **Expected:** 4 engine steps advance state through execute_task → summarize_slice → validate_milestone → done, all against real SQLite.

### 6. Integration test — multi-task sequential processing

1. Run `go test ./internal/auto/... -count=1 -run TestIntegration_StepMultipleTasksInSlice -v`
2. **Expected:** 5 engine steps process T01, T02, T03 sequentially then summarize and validate.

### 7. DB tests still pass

1. Run `go test ./internal/db/... -count=1`
2. **Expected:** All DB tests pass including SumChildSessionCosts and the new GetSessionTokenUsage.

### 8. GetSessionTokenUsage query exists

1. Run `grep -q 'GetSessionTokenUsage' internal/db/querier.go`
2. **Expected:** Exit 0 — query is in the generated interface.

### 9. All 5 adapter types present

1. Run `grep -c 'type DB\|type Session\|type Coordinator' internal/auto/adapters.go`
2. **Expected:** Count ≥ 5 (DBStateQuerier, DBStatusAdvancer, DBTokenQuerier, SessionServiceCreator, CoordinatorDispatcher).

### 10. BudgetChecker already present

1. Run `grep -q 'DBBudgetChecker' internal/auto/budget.go`
2. **Expected:** Exit 0 — confirms BudgetChecker was already implemented in S01/M003.

## Edge Cases

### Empty result sets

1. Run `go test ./internal/auto/... -run 'TestDBStateQuerier/.*Empty' -v`
2. **Expected:** ListSlicesByMilestone_Empty and ListTasksBySlice_Empty return empty slices without error.

### NullString conversion

1. Covered by TestDBStateQuerier subtests — SliceRow.DependsOn handles both empty and non-empty sql.NullString values.

## Failure Signals

- `go build ./...` fails → adapter imports or type mismatches
- TestDBStatusAdvancer fails → phase transitions don't match DeriveState expectations (critical — engine loop will break)
- TestIntegration fails → real adapters don't compose correctly with engine.Step()
- TestDBTokenQuerier fails → token aggregation query broken

## Not Proven By This UAT

- Real Coordinator dispatch (mocked — requires full service graph, deferred to S05)
- Real session.Service integration (mocked — deferred to S05)
- TUI rendering of auto-mode state (deferred to S03)
- CLI command integration (deferred to S04)

## Notes for Tester

- Integration tests use mock Dispatcher that records calls and returns nil. This is intentional — real Coordinator wiring is S05's scope.
- The setupAdapterTestDB helper uses shared-cache named DSN (not plain :memory:) to support parallel subtests.
 |
| S03 | TUI Integration Wiring | medium | S02 | ✅ | # S03: TUI Integration Wiring — UAT

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
 |
| S04 | CLI Commands | medium | S02 | ✅ | # S04: CLI Commands — UAT

**Milestone:** M005
**Written:** 2026-03-28T18:00:23.230Z

# S04: CLI Commands — UAT

**Milestone:** M005
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: CLI commands are structural — verification is compilation, command registration, and arg validation. No runtime LLM calls needed.

## Preconditions

- M005 worktree checked out with S01+S02+S03+S04 changes
- Go toolchain available (go 1.24+)

## Smoke Test

Run `go build ./...` — must exit 0. Then `go test ./internal/cmd/ -run 'TestAuto|TestNext' -v -count=1` — all 6 tests must pass.

## Test Cases

### 1. Full compilation gate

1. Run `go build ./...`
2. **Expected:** Exit code 0, no errors.

### 2. Static analysis gate

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no new warnings.

### 3. Auto subcommand tree

1. Run `go test ./internal/cmd/ -run TestAutoCmd_SubcommandTree -v -count=1`
2. **Expected:** Verifies autoCmd has exactly 4 subcommands (start, pause, stop, status).

### 4. Auto command registered on root

1. Run `go test ./internal/cmd/ -run TestAutoCmd_IsRegistered -v -count=1`
2. **Expected:** `rootCmd.Find([]string{"auto"})` succeeds.

### 5. Next command registered on root

1. Run `go test ./internal/cmd/ -run TestNextCmd_IsRegistered -v -count=1`
2. **Expected:** `rootCmd.Find([]string{"next"})` succeeds.

### 6. Auto start requires milestone-id arg

1. Run `go test ./internal/cmd/ -run TestAutoStartCmd_RequiresArg -v -count=1`
2. **Expected:** Executing autoStartCmd with no args returns error (ExactArgs(1) enforced).

### 7. Next requires milestone-id arg

1. Run `go test ./internal/cmd/ -run TestNextCmd_RequiresArg -v -count=1`
2. **Expected:** Executing nextCmd with no args returns error.

### 8. Status command has --json flag

1. Run `go test ./internal/cmd/ -run TestAutoStatusCmd_HasJSONFlag -v -count=1`
2. **Expected:** `autoStatusCmd.Flags().Lookup("json")` is non-nil.

### 9. buildAutoEngine shared between CLI and TUI

1. Run `grep -c 'buildAutoEngine' internal/cmd/auto.go internal/cmd/root.go`
2. **Expected:** auto.go has definition + 2 call sites, root.go has 1 call site.

### 10. MilestoneID field in AutoConfig

1. Run `grep 'MilestoneID' internal/config/config.go`
2. **Expected:** Field present with json tag "milestone_id".

### 11. MilestoneID wired in root.go

1. Run `grep 'SetAutoMilestoneID' internal/cmd/root.go`
2. **Expected:** Match found — config value wired to TUI model.

### 12. No regressions in auto package

1. Run `go test ./internal/auto/... -count=1`
2. **Expected:** All tests pass.

### 13. No regressions in config package

1. Run `go test ./internal/config/... -count=1`
2. **Expected:** All tests pass.

## Edge Cases

### No database available

1. `buildAutoEngine()` returns error when `app.Queries` is nil.
2. MilestoneID wiring in root.go is outside the DB guard — config value still reaches TUI model.

### No lock file exists

1. `autoStatusCmd` reports "not running" when lock file doesn't exist.
2. With `--json`, outputs `{"running": false}`.

### Malformed lock file

1. `autoStatusCmd` reports "not running (lock file unreadable)" when JSON unmarshal fails.

## Failure Signals

- `go build ./...` fails → type mismatch or import error in new files
- TestAutoCmd_SubcommandTree fails → subcommand registration broken
- TestAutoStartCmd_RequiresArg passes (no error) → ExactArgs not set
- `grep MilestoneID internal/config/config.go` returns nothing → field not added

## Not Proven By This UAT

- Actual engine.Run() or engine.Step() execution with real LLM sessions (deferred to S05)
- Lock file acquisition/release during CLI execution (requires real run)
- Signal handling (SIGINT) actually stopping the engine (requires real run)
 |
| S05 | End-to-End Proof + Worktree Lifecycle | low | S03, S04 | ✅ | # S05: End-to-End Proof + Worktree Lifecycle — UAT

**Milestone:** M005
**Written:** 2026-03-28T19:38:26.429Z

# S05: End-to-End Proof + Worktree Lifecycle — UAT

**Milestone:** M005
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: All changes are wiring and integration tests — verification is compilation, test execution, and structural grep checks. No runtime UI or LLM behavior to observe.

## Preconditions

- M005 worktree checked out with S01-S05 changes
- Go toolchain available (go 1.24+)
- Git available (for worktree lifecycle tests)

## Smoke Test

Run `go build ./...` — must exit 0. Then `go test ./internal/auto/... -count=1 -run TestE2E -v` — all 6 E2E tests must pass.

## Test Cases

### 1. Full compilation gate

1. Run `go build ./...`
2. **Expected:** Exit code 0, no errors.

### 2. Static analysis gate

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no new warnings.

### 3. E2E — ShellVerifier pass path

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_FullAssemblyWithVerifier -v`
2. **Expected:** Test passes — engine dispatches unit, runs verification command `true`, verification passes, unit completes.

### 4. E2E — ShellVerifier fail path

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_FullAssemblyWithFailingVerifier -v`
2. **Expected:** Test passes — engine dispatches unit, runs verification command `false`, verification fails, diagnostic retry fires.

### 5. E2E — StuckDetector triggers

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_FullAssemblyWithStuckDetector -v`
2. **Expected:** Test passes — stuck detector recognizes repeated failures, triggers diagnostic retry with "stuck" warning.

### 6. E2E — Worktree lifecycle (create/merge/remove)

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_WorktreeLifecycle -v`
2. **Expected:** Test passes — worktree created before dispatch, merged after completion, directory removed. Logs show "Creating worktree", "Merging worktree", "Removing worktree".

### 7. E2E — Worktree resume existing

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_WorktreeResumeExisting -v`
2. **Expected:** Test passes — pre-existing worktree detected, logs "Resuming existing worktree", run completes without duplicate creation error.

### 8. E2E — Full adapter composition with real DB

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_BuildAutoEngineComposition -v`
2. **Expected:** Test passes — real DB adapters compose with engine, steps through execute_task → summarize_slice → validate_milestone → done.

### 9. SetWorktreeManager present in engine

1. Run `grep -c 'SetWorktreeManager' internal/auto/engine.go`
2. **Expected:** Count ≥ 2 (field + method).

### 10. buildAutoEngine wires all three components

1. Run `grep -q 'ShellVerifier' internal/cmd/auto.go && grep -q 'NewStuckDetector' internal/cmd/auto.go && grep -q 'NewWorktreeManager' internal/cmd/auto.go && echo "OK"`
2. **Expected:** Prints "OK".

### 11. Full auto test suite regression check

1. Run `go test ./internal/auto/... -count=1`
2. **Expected:** All 100+ tests pass.

### 12. CLI test regression check

1. Run `go test ./internal/cmd/ -count=1`
2. **Expected:** All CLI tests pass.

## Edge Cases

### No verification commands configured

1. `buildAutoEngine` passes nil ShellVerifier when `cfg.Auto.VerificationCommands` is empty.
2. Engine runs without verification gate — covered by existing non-E2E tests.

### StuckThreshold is 0

1. `buildAutoEngine` passes nil StuckDetector when `cfg.Auto.StuckThreshold` is 0.
2. Engine runs without stuck detection — covered by existing tests.

### WorktreeMode not "per-milestone"

1. `buildAutoEngine` skips WorktreeManager construction for any mode other than "per-milestone".
2. Engine.Run skips worktree lifecycle when worktreeManager is nil.

### Worktree merge with no changes

1. Covered by TestE2E_WorktreeLifecycle — mock dispatcher doesn't create files, so merge encounters "nothing to commit".
2. **Expected:** Merge logs "No changes to merge", continues to remove.

## Failure Signals

- `go build ./...` fails → type mismatch in wiring
- TestE2E_WorktreeLifecycle fails → worktree create/merge/remove lifecycle broken
- TestE2E_BuildAutoEngineComposition fails → adapter composition doesn't work with real DB
- TestE2E_FullAssemblyWithVerifier fails → ShellVerifier integration broken
- Any existing test fails → S05 changes introduced regression

## Not Proven By This UAT

- Real LLM dispatch (Dispatcher is mocked — requires API keys and live models)
- File content changes within worktree during dispatch (mock dispatcher is a no-op)
- Production crush.json parsing of WorktreeMode/VerificationCommands/StuckThreshold (config parsing tested in S01)
 |
