# S02: Production DB Adapters + Engine Wiring

**Goal:** Production implementations of StateQuerier, StatusAdvancer, SessionCreator, TokenQuerier, and Dispatcher adapters backed by sqlc queries and session.Service, with BudgetChecker already done. All adapters satisfy engine interfaces and pass tests against real SQLite.
**Demo:** After this: # S02: Production DB Adapters + Engine Wiring — UAT

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


## Tasks
- [x] **T01: Added GetSessionTokenUsage sqlc query and 5 production adapters (DBStateQuerier, DBStatusAdvancer, SessionServiceCreator, DBTokenQuerier, CoordinatorDispatcher) satisfying all engine interfaces** — Add the missing sqlc query for token usage, run sqlc generate, then implement all 5 production adapter structs (DBStateQuerier, DBStatusAdvancer, SessionServiceCreator, DBTokenQuerier, CoordinatorDispatcher) in a single new file `internal/auto/adapters.go`.

The adapters are thin wrappers:
- **DBStateQuerier** wraps `*db.Queries`, converts `db.Milestone`→`auto.MilestoneRow`, `db.Slice`→`auto.SliceRow` (handling `sql.NullString` for DependsOn), `db.Task`→`auto.TaskRow` (handling `sql.NullString` for Description)
- **DBStatusAdvancer** wraps `*db.Queries`, switches on unit type to call the right Update methods. Phase transitions MUST match DeriveState expectations:
  - `UnitResearch` → `UpdateSlicePhase(planning)`
  - `UnitPlanSlice` → `UpdateSlicePhase(executing)`
  - `UnitExecuteTask` → `UpdateTaskStatus(completed)` + `UpdateTaskPhase(completed)`
  - `UnitSummarizeSlice` → `UpdateSliceStatus(completed)` + `UpdateSlicePhase(completed)`
  - `UnitValidateMilestone` → `UpdateMilestoneStatus(completed)` + `UpdateMilestonePhase(completed)`
- **SessionServiceCreator** wraps `session.Service`. `CreateSession`→`service.Create()` returning `session.ID`. `CreateChildSession`→`service.CreateTaskSession()` (id=toolCallID, parentID=parentSessionID)
- **DBTokenQuerier** wraps `*db.Queries`, calls the new `GetSessionTokenUsage` query
- **CoordinatorDispatcher** wraps `agent.Coordinator`, calls `RunWithForcedTier()` discarding `*AgentResult`, returning only error (per D015)

Also verify BudgetChecker (`DBBudgetChecker` in budget.go) already satisfies `BudgetChecker` interface — no new code needed.
  - Estimate: 1h
  - Files: internal/db/sql/sessions.sql, internal/db/sessions.sql.go, internal/db/querier.go, internal/db/models.go, internal/auto/adapters.go
  - Verify: go build ./... && go vet ./... && grep -q 'GetSessionTokenUsage' internal/db/querier.go && grep -q 'DBStateQuerier' internal/auto/adapters.go && grep -q 'DBStatusAdvancer' internal/auto/adapters.go && grep -q 'CoordinatorDispatcher' internal/auto/adapters.go
- [x] **T02: Added 5 test functions (14 subtests) for all production adapters against real in-memory SQLite with goose migrations** — Write `internal/auto/adapters_test.go` testing each adapter against real in-memory SQLite with goose migrations, following the K004 pattern (unique DSN per test, `t.Parallel()`).

Tests needed:
1. **TestDBStateQuerier** — seed milestones/slices/tasks, call ListMilestones/ListSlicesByMilestone/ListTasksBySlice, assert correct mapping from db types to auto Row types. Verify sql.NullString→string conversion for Slice.DependsOn and Task.Description (both empty and non-empty).
2. **TestDBStatusAdvancer** — for each unit type, seed the appropriate entity, call AdvanceStatus, read back from DB and verify phase/status. Critical: verify phase transitions match DeriveState expectations exactly:
   - UnitResearch → slice phase becomes 'planning'
   - UnitPlanSlice → slice phase becomes 'executing'
   - UnitExecuteTask → task status+phase become 'completed'
   - UnitSummarizeSlice → slice status+phase become 'completed'
   - UnitValidateMilestone → milestone status+phase become 'completed'
3. **TestDBTokenQuerier** — create sessions with known token counts, call GetTokenUsage, verify sums.
4. **TestSessionServiceCreator** — requires a `session.Service` mock or thin fake. Test CreateSession returns an ID, CreateChildSession returns an ID with parent linkage.
5. **TestCoordinatorDispatcher** — use a mock coordinator that records calls and returns nil/error. Verify RunWithForcedTier passes through correctly and discards AgentResult.

For DB-backed tests (StateQuerier, StatusAdvancer, TokenQuerier), use `setupTestDB` from internal/db/auto_test.go pattern but in the auto package — create a local helper that opens in-memory SQLite, runs goose migrations, and returns `*db.Queries`.

For SessionServiceCreator and CoordinatorDispatcher, define minimal mock types in the test file.
  - Estimate: 1.5h
  - Files: internal/auto/adapters_test.go
  - Verify: go test ./internal/auto/... -count=1 -run 'TestDB|TestSession|TestCoordinator' -v && go vet ./internal/auto/...
- [x] **T03: Added 2 integration tests proving engine.Step() with real DB adapters advances task→slice→milestone through derive→dispatch→advance cycle against real SQLite** — Write an integration test in `internal/auto/adapters_integration_test.go` that proves the full derive→dispatch→advance loop works with real production adapters against real SQLite.

Test scenario:
1. Set up in-memory SQLite with goose migrations
2. Seed a milestone (status=active, phase=executing), a slice (status=active, phase=executing), and a task (status=active, phase=executing)
3. Create all production adapters:
   - `DBStateQuerier` wrapping `*db.Queries`
   - `DBStatusAdvancer` wrapping `*db.Queries`
   - Mock `SessionCreator` (return fixed IDs — real session.Service needs too much setup)
   - Mock `Dispatcher` (record call, return nil)
   - `nil` for BudgetChecker (0 ceiling disables the gate)
   - `nil` for ContextMonitor (disabled)
   - `nil` for StuckDetector (disabled)
   - `nil` for Verifier (disabled for non-task or skip)
4. Construct engine via `NewEngine()` with all adapters
5. Call `engine.Step()` 
6. Assert: the mock dispatcher was called with the expected task's prompt, AND the task's status in DB is now 'completed' and phase is 'completed'
7. Call `engine.Step()` again — with all tasks complete, DeriveState should return UnitSummarizeSlice. Mock dispatcher handles it, then advancer sets slice to completed.

This proves the real adapters work as an assembled system, not just individually. The mock dispatcher is necessary because a real Coordinator requires too many services (config, sessions, messages, permissions, etc.) — that integration is deferred to S05.
  - Estimate: 1h
  - Files: internal/auto/adapters_integration_test.go
  - Verify: go test ./internal/auto/... -count=1 -run TestIntegration -v && go test ./internal/auto/... -count=1 && go test ./internal/db/... -count=1
