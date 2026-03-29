# S01: Branch Reconciliation

**Goal:** Merge M001–M003 engine code into M005 working branch with all conflicts resolved, event systems unified, and full compilation + test suite passing.
**Demo:** After this: # S01: Branch Reconciliation — UAT

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


## Tasks
- [x] **T01: Merged milestone/M003 into M005 with all .gsd/ conflicts resolved, preserving ReasoningEffort in coordinator.go and bringing 12+ auto-mode engine files onto the branch** — Perform `git merge milestone/M003` on the M005 branch. All .go source files merge cleanly (confirmed by merge-tree). Only `.gsd/` metadata files conflict — resolve those by keeping M005's versions (they include M004 completion data). **Critical:** The coordinator.go change from M003 removes the `ReasoningEffort` check in `isAnthropicThinking()` — this is a regression because M005 uses ReasoningEffort extensively. Reject that hunk by reverting coordinator.go to M005's version after merge.

After this task, the tree will NOT compile because both `event.go` (M004) and `events.go` (M003) define `EventUnitStarted` and `EventUnitCompleted` with different types. That's expected — T02 resolves it.

Steps:
1. Run `git merge milestone/M003` from M005 branch
2. For each `.gsd/` conflict: `git checkout --ours .gsd/` to keep M005 versions
3. Verify coordinator.go: `git diff HEAD -- internal/agent/coordinator.go` and revert the `isAnthropicThinking` change with `git checkout HEAD~1 -- internal/agent/coordinator.go` (or manual restore)
4. `git add` all resolved files and commit the merge
5. Verify new M003 files exist: `ls internal/auto/engine.go internal/auto/state.go internal/auto/stuck.go`
  - Estimate: 20m
  - Files: internal/agent/coordinator.go, internal/auto/events.go, internal/auto/engine.go, internal/auto/state.go, internal/config/config.go, internal/db/querier.go, internal/db/sessions.sql.go, .gsd/DECISIONS.md, .gsd/KNOWLEDGE.md, .gsd/REQUIREMENTS.md
  - Verify: git log --oneline -1 shows merge commit. `ls internal/auto/engine.go internal/auto/state.go internal/auto/stuck.go internal/auto/events.go` all exist. `grep -c ReasoningEffort internal/agent/coordinator.go` returns > 0 (regression not applied).
- [x] **T02: Unified M003 engine events and M004 TUI events into a single AutoEvent type, eliminating all duplicate-symbol compilation errors** — Resolve the duplicate-symbol compile error by unifying M003's engine events (`events.go`) and M004's TUI events (`event.go`) into a single event system. The engine is the source of truth — M003's `events.go` has 12 event types vs M004's 8, and uses the correct `pubsub.EventType` type.

Steps:
1. Move `AutoSnapshot` and `SliceProgress` structs from `event.go` into `events.go` (after the `AutoEvent` struct)
2. Add `Snapshot *AutoSnapshot` field to M003's `AutoEvent` struct
3. Add M004's lifecycle event constants that M003 is missing (`EventAutoStarted`, `EventAutoPaused`, `EventAutoResumed`, `EventAutoCompleted`, `EventAutoError`, `EventStateChanged`) as `pubsub.EventType` constants — use the same string values M004 used
4. Delete `event.go` entirely
5. Rewrite `event_test.go` to test the unified struct: test that all event constants are unique, test `AutoSnapshot` construction, test `AutoEvent` with both engine fields (Unit, Error, Timestamp, Message) and TUI fields (Snapshot)
6. Update `internal/app/auto_events_test.go`: change `auto.AutoEvent{Type: auto.EventUnitStarted, ...}` to use the new struct shape (remove `Type` field since events are identified by `pubsub.EventType` not a struct field; use `NewAutoEvent()` where appropriate; keep `Snapshot` field)
7. Update `internal/ui/model/ui.go` line ~631: the handler `case pubsub.Event[auto.AutoEvent]: m.autoSnapshot = msg.Payload.Snapshot` should still work since the unified struct has `Snapshot *AutoSnapshot`
8. Run `go build ./...` to confirm compilation
9. Run `gofumpt -w .` to format
  - Estimate: 45m
  - Files: internal/auto/events.go, internal/auto/event.go, internal/auto/event_test.go, internal/app/auto_events_test.go, internal/ui/model/ui.go
  - Verify: `go build ./...` exits 0. `ls internal/auto/event.go` returns 'No such file' (deleted). `grep -c 'AutoSnapshot' internal/auto/events.go` returns > 0. `grep -c 'Snapshot' internal/auto/events.go` returns > 0.
- [x] **T03: All six verification gates pass: go build, go vet, and four targeted test suites (auto, config, app, db) exit 0 with zero failures** — Run the complete verification suite to confirm the merge and unification are correct. Fix any remaining compilation errors or test failures.

Steps:
1. Run `go build ./...` — must exit 0
2. Run `go vet ./...` — must exit 0 (check for pre-existing issues per K005)
3. Run `go test ./internal/auto/... -v -count=1` — all engine, state, safety, event, worktree tests pass
4. Run `go test ./internal/config/... -run TestAutoConfig -v` — config parsing tests pass
5. Run `go test ./internal/app/... -run TestAutoEvent -v` — broker tests pass
6. Run `go test ./internal/db/... -v -count=1` — DB tests including SumChildSessionCosts pass
7. If any test fails, diagnose and fix. Common issues:
   - M003 engine tests may reference `NewAutoEvent()` — verify it still exists in unified `events.go`
   - Mock types in engine tests may need updated event field references
   - The `internal/db/auto_test.go` file should be deleted by M003's merge — verify it's gone
8. Run `gofumpt -w .` for final formatting
9. Commit all fixes
  - Estimate: 30m
  - Files: internal/auto/events.go, internal/auto/event_test.go, internal/auto/engine.go, internal/auto/engine_test.go, internal/auto/engine_integration_test.go, internal/auto/state_test.go, internal/app/auto_events_test.go
  - Verify: `go build ./...` exits 0 && `go vet ./...` exits 0 && `go test ./internal/auto/... -count=1` exits 0 && `go test ./internal/config/... -run TestAutoConfig` exits 0 && `go test ./internal/app/... -run TestAutoEvent` exits 0 && `go test ./internal/db/... -count=1` exits 0
