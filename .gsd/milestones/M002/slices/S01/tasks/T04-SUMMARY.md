---
id: T04
parent: S01
milestone: M002
provides:
  - CLI commands (crush auto start/pause/stop/status) registered in root.go
  - Integration tests proving full loop lifecycle (research → plan → execute × 2 → summarize → validate)
  - Fixed flaky TestLockFile_ConcurrentAcquire race condition
  - Fixed fixedSequenceQuerier to properly advance through non-task unit types
key_files:
  - internal/cmd/auto.go
  - internal/cmd/root.go
  - internal/auto/engine_integration_test.go
  - internal/auto/engine_test.go
  - internal/auto/lock.go
key_decisions:
  - "CLI adapters use placeholder implementations for StateQuerier/SessionCreator/Dispatcher/StatusAdvancer — full DB wiring deferred to later slice when auto-mode schema lands"
  - "Pause uses a signal file (.pause) alongside the lock file rather than IPC, keeping the mechanism simple and filesystem-observable"
  - "Stop reads PID from lock file and sends SIGTERM directly"
  - "Fixed fixedSequenceQuerier by coupling advancer to querier via Advance() method — index now advances in AdvanceStatus after each unit completes, not during DeriveState calls"
patterns_established:
  - "Test querier + advancer coupling pattern: advancer.Advance() moves the fixedSequenceQuerier index forward so DeriveState sees fresh state on next iteration"
observability_surfaces:
  - "stderr progress events in auto start: ▶ (started), ✓ (completed), ✗ (failed), ⏸ (paused), ⏹ (stopped)"
  - "crush auto status reads lock file + DeriveState to report engine PID and next unit"
duration: 30m
verification_result: passed
completed_at: 2026-03-27
blocker_discovered: false
---

# T04: CLI commands — crush auto start/pause/stop/status with integration test

**Added crush auto start/pause/stop/status CLI commands with placeholder adapters and 4 integration tests proving full loop lifecycle, step execution, pause mid-loop, and child session creation**

## What Happened

Created `internal/cmd/auto.go` with four cobra subcommands registered under `crush auto`:
- **start [milestone-id]**: Constructs Engine with placeholder adapters, subscribes to broker events printing progress to stderr, handles SIGINT/SIGTERM for graceful shutdown.
- **pause**: Writes a `.pause` signal file next to the lock file.
- **stop**: Reads PID from lock file and sends SIGTERM.
- **status**: Checks lock file for running engine PID (via signal 0), calls DeriveState to show next dispatchable unit.

Registered `autoCmd` in root.go's `AddCommand` block.

Fixed two pre-existing test infrastructure bugs:
1. **Flaky `TestLockFile_ConcurrentAcquire`**: The race was between `OpenFile(O_EXCL)` and `Write(payload)` — another goroutine could read the empty file, fail unmarshal, treat it as stale, delete it, and acquire. Fixed by treating unmarshal failures as "lock held" rather than "stale".
2. **Infinite loop in `fixedSequenceQuerier`**: For non-task units (research, plan, summarize, validate), `DeriveState` returns without calling `ListTasksBySlice`, so the index never advanced. Fixed by introducing `Advance()` on the querier, called by the mock advancer's `AdvanceStatus()` after each unit completes.

Created 4 integration tests in `engine_integration_test.go`:
- `TestIntegration_FullLoopLifecycle`: 6-unit sequence (research → plan → execute × 2 → summarize → validate), verifies dispatch order, tier assignments, advancer calls, event publishing (6 started + 6 completed), and idle state after completion.
- `TestIntegration_StepExecutesSingleUnit`: Verifies Step() dispatches exactly one unit.
- `TestIntegration_PauseMidLoop`: Verifies pause stops after current unit, engine enters EnginePaused state.
- `TestIntegration_ChildSessionsCreated`: Verifies 1 parent + 2 child sessions for 2 units.

## Verification

All verification commands pass:
- `go test ./internal/auto/... -count=1` — 36 tests pass (18 state + 14 engine + 4 integration)
- `go test ./internal/cmd/... -count=1` — no regressions
- `go build .` — project compiles
- `go vet ./internal/auto/... ./internal/cmd/...` — clean

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go test ./internal/auto/... -count=1` | 0 | ✅ pass | 0.3s |
| 2 | `go test ./internal/cmd/... -count=1` | 0 | ✅ pass | 0.7s |
| 3 | `go build .` | 0 | ✅ pass | 0.5s |
| 4 | `go vet ./internal/auto/... ./internal/cmd/...` | 0 | ✅ pass | 0.3s |

## Diagnostics

- `crush auto status` reports engine running state and next dispatchable unit.
- Lock file at `{dataDir}/auto.lock` contains PID + start time as JSON.
- Pause signal file at `{dataDir}/auto.lock.pause`.

## Deviations

- CLI adapters are placeholder stubs (return "not wired yet" errors) rather than wired to real App services — the auto-mode DB schema doesn't exist yet so full wiring is deferred to a later slice. The commands compile and the structure is correct.
- Added `Advance()` method to `fixedSequenceQuerier` and coupled it via `mockAdvancer.querier` field — this was necessary to fix the infinite loop bug that existed in the test infrastructure for non-task unit types.

## Known Issues

- CLI adapters (`cmdStateQuerier`, `cmdSessionCreator`, `cmdDispatcher`, `cmdStatusAdvancer`) return stub data — `crush auto start` will fail at runtime until DB wiring is complete in a later slice.

## Files Created/Modified

- `internal/cmd/auto.go` — New file: cobra commands for crush auto start/pause/stop/status with placeholder adapters
- `internal/cmd/root.go` — Added `autoCmd` to root command registration
- `internal/auto/engine_integration_test.go` — New file: 4 integration tests for full loop lifecycle
- `internal/auto/engine_test.go` — Fixed fixedSequenceQuerier index advancement, added coupled advancer pattern
- `internal/auto/lock.go` — Fixed race: treat unmarshal failures as "lock held" not "stale"
