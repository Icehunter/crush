---
estimated_steps: 34
estimated_files: 3
skills_used: []
---

# T04: CLI commands — crush auto start/pause/stop/status with integration test

Wire the Engine into cobra commands and add an integration test proving the full loop lifecycle. This is the final assembly task that makes auto-mode runnable from the terminal.

## Steps

1. Create `internal/cmd/auto.go` with cobra commands:
   - `crush auto start [milestone-id]` — creates App, constructs Engine, calls Engine.Run(). Handles SIGINT/SIGTERM for graceful shutdown. Prints progress to stderr. Requires `--milestone` or positional arg.
   - `crush auto pause` — sends pause signal (writes to a signal file or uses the lock file). Engine finishes current unit then exits.
   - `crush auto stop` — sends stop signal (kills the running process via PID from lock file).
   - `crush auto status` — reads DB state + lock file, prints structured status: active milestone, current slice/task, phase, progress (N/M tasks done), whether engine is running.
   - All commands use `setupApp(cmd)` pattern from root.go.
   - Register `autoCmd` (parent) with subcommands in root.go's `init()`.

2. Update `internal/cmd/root.go` to register `autoCmd` in the `AddCommand` block.

3. Wire Engine construction in `internal/app/app.go` or directly in the command — keep it simple. The Engine needs: db.Queries, Coordinator, Sessions, Broker, dataDir. All available from App.

4. Create `internal/auto/engine_integration_test.go` with an integration test:
   - Use setupTestDB from db package to create in-memory DB
   - Seed a milestone with one slice and two tasks
   - Create a mock Coordinator that records calls and returns success
   - Run Engine.Run() — verify it dispatches research, plan, execute (×2), summarize, validate in order
   - Verify all tasks end up status=completed
   - Verify events were published in correct order
   - Test Engine.Step() executes exactly one unit
   - Test pause mid-loop

5. Run `go test ./internal/auto/... -count=1`, `go test ./internal/cmd/... -count=1`, `go vet ./...`, and `go build .` to verify everything compiles and passes.

## Must-Haves

- [ ] `crush auto start` launches engine loop for a milestone
- [ ] `crush auto pause` signals finish-then-stop
- [ ] `crush auto stop` terminates the running engine
- [ ] `crush auto status` prints milestone/slice/task progress and engine state
- [ ] Commands registered in root.go and use setupApp pattern
- [ ] Integration test proves full loop lifecycle: seed → run → all tasks completed → events published
- [ ] `go build .` succeeds — entire project compiles

## Verification

- `go test ./internal/auto/... -count=1 -v` — integration test passes
- `go test ./internal/cmd/... -count=1` — no regressions in existing cmd tests
- `go build .` — project compiles
- `go vet ./internal/auto/... ./internal/cmd/...` — no vet errors

## Inputs

- `internal/auto/engine.go`
- `internal/auto/state.go`
- `internal/auto/unit.go`
- `internal/auto/events.go`
- `internal/auto/lock.go`
- `internal/auto/prompts.go`
- `internal/cmd/root.go`
- `internal/cmd/run.go`
- `internal/app/app.go`
- `internal/session/session.go`
- `internal/agent/coordinator.go`

## Expected Output

- `internal/cmd/auto.go`
- `internal/cmd/root.go`
- `internal/auto/engine_integration_test.go`

## Verification

go test ./internal/auto/... -count=1 && go test ./internal/cmd/... -count=1 && go build . && go vet ./internal/auto/... ./internal/cmd/...
