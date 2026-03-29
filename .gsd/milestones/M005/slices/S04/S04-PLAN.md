# S04: CLI Commands

**Goal:** Provide `crush auto start|pause|stop|status` and `crush next` CLI subcommands wired to the auto-mode engine, plus wire `AutoConfig.MilestoneID` so TUI ctrl+a can actually start auto-mode from crush.json config.
**Demo:** After this: # S04: CLI Commands — UAT

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


## Tasks
- [x] **T01: Add crush auto start|pause|stop|status and crush next CLI subcommands with shared buildAutoEngine() helper** — Create `internal/cmd/auto.go` containing:

1. `autoCmd` parent command (`crush auto`) with short/long descriptions
2. `autoStartCmd` — takes milestone-id as positional arg (cobra.ExactArgs(1)), calls `setupApp(cmd)`, builds engine via new `buildAutoEngine()` helper, sets up signal context with `signal.NotifyContext(ctx, os.Interrupt)`, calls `engine.Run(ctx, milestoneID)`, prints result/error. Defers `app.Shutdown()`.
3. `autoPauseCmd` — prints message: "Pause is only available in TUI mode (ctrl+a) or send SIGINT to the running process."
4. `autoStopCmd` — prints message: "Stop is only available in TUI mode (ctrl+a) or send SIGINT to the running process."
5. `autoStatusCmd` — reads lock file via `auto.NewLockFile(dataDir)`. Read the lock file JSON directly (os.ReadFile on lockfile.Path()), unmarshal lockPayload, check isProcessAlive(pid). With `--json` flag, outputs JSON `{"running": bool, "pid": int, "started_at": string}`. Without --json, prints human-readable status.
6. `nextCmd` — top-level command (`crush next`), takes milestone-id as positional arg, calls `setupApp(cmd)`, builds engine, calls `engine.Step(ctx, milestoneID)`, prints result. Defers `app.Shutdown()`.
7. `buildAutoEngine(app *appPkg.App) *auto.Engine` — extracted helper that creates all adapters and engine (replicate the pattern from root.go lines 118–145). Reads BudgetCeiling from `app.Config().Auto`. Returns the constructed engine.
8. `init()` function: registers `autoCmd` with `start`, `pause`, `stop`, `status` subcommands, and `nextCmd` on `rootCmd`.

Then modify `internal/config/config.go`:
- Add `MilestoneID string` field to `AutoConfig` struct: `MilestoneID string \`json:"milestone_id,omitempty\" jsonschema:"description=Active milestone ID for auto-mode execution"\``

Then modify `internal/cmd/root.go`:
- Replace the inline engine construction block (lines ~118–145) with a call to `buildAutoEngine(app)` from auto.go. Since buildAutoEngine is in the same package, it's directly callable.
- After `model.SetAutoController(ctrl)`, replace the TODO(S04) comment with: `if cfg.Auto != nil && cfg.Auto.MilestoneID != "" { model.SetAutoMilestoneID(cfg.Auto.MilestoneID) }`

Note on lock file status reading: The lockPayload struct and isProcessAlive function are unexported in auto package. For `autoStatusCmd`, either: (a) export them, (b) add a StatusFromLockFile() function to auto package, or (c) read the file and parse JSON directly in cmd package. Option (b) is cleanest — add a `func StatusFromLockFile(dataDir string) (running bool, pid int, startedAt time.Time, err error)` to `internal/auto/lock.go`.

Follow session.go patterns for subcommand registration and run.go for signal handling. Use `gofumpt -w .` to format.
  - Estimate: 45m
  - Files: internal/cmd/auto.go, internal/cmd/root.go, internal/config/config.go, internal/auto/lock.go
  - Verify: go build ./... && go vet ./...
- [x] **T02: Added 6 parallel test functions verifying cobra command registration, subcommand tree, arg validation, and flag presence for auto and next commands** — Create `internal/cmd/auto_test.go` with tests verifying the cobra command structure:

1. `TestAutoCmd_SubcommandTree` — verify `autoCmd` has exactly 4 subcommands (start, pause, stop, status) by iterating `autoCmd.Commands()` and checking Use fields contain expected names
2. `TestNextCmd_IsRegistered` — verify `nextCmd` is registered on `rootCmd` by finding it via `rootCmd.Find([]string{"next"})` — should not return an error
3. `TestAutoCmd_IsRegistered` — verify `autoCmd` is registered on `rootCmd` via `rootCmd.Find([]string{"auto"})`
4. `TestAutoStartCmd_RequiresArg` — set autoStartCmd.RunE to a no-op, call `autoStartCmd.Execute()` with no args, verify it returns an error
5. `TestNextCmd_RequiresArg` — same pattern as above for nextCmd
6. `TestAutoStatusCmd_HasJSONFlag` — verify `autoStatusCmd.Flags().Lookup("json")` is non-nil

Add a config parsing test (find existing auto config tests — research says they're in `internal/config/`):
7. `TestAutoConfig_MilestoneID` — create a temp crush.json with `{"auto": {"milestone_id": "M005"}}`, parse via config.Init or equivalent, verify `cfg.Auto.MilestoneID == "M005"`

Also add a lock status test in `internal/auto/lock_test.go`:
8. `TestStatusFromLockFile_NoFile` — verify returns running=false when no lock file exists
9. `TestStatusFromLockFile_WithFile` — write a lock payload with current PID, verify returns running=true with correct PID

Use `require` from testify. Use `t.Parallel()`. Format with `gofumpt -w .`.
  - Estimate: 25m
  - Files: internal/cmd/auto_test.go, internal/auto/lock_test.go, internal/config/load_test.go
  - Verify: go test ./internal/cmd/ -run 'TestAuto|TestNext' -v -count=1 && go test ./internal/auto/ -run TestStatusFromLockFile -v -count=1 && go test ./internal/config/ -run TestAutoConfig_MilestoneID -v -count=1
