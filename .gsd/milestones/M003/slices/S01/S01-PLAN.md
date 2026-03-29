# S01: Auto Config + Verification Gates

**Goal:** Configure auto.verification_commands in crush.json. Engine runs them after each task dispatch. On failure, engine re-dispatches with a diagnostic prompt containing truncated failure output. Tests prove the full verify→retry→succeed and verify→retry→fail paths.
**Demo:** After this: Configure auto.verification_commands in crush.json. Engine runs them after each task dispatch. On failure, engine re-dispatches with a diagnostic prompt containing truncated failure output. Tests prove the full verify→retry→succeed and verify→retry→fail paths.

## Tasks
- [x] **T01: Copied M002 auto package (19 Go files + 6 templates) and DB dependencies into M003 worktree; added AutoConfig struct to Config with four fields and round-trip parsing tests** — ## Description

This task brings the M002 engine code into the M003 worktree and adds the `AutoConfig` struct to the config system. Without the auto package files, nothing in this slice compiles. The config addition delivers R014.

## Steps

1. Copy all `internal/auto/` files from the M002 worktree (`/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/`) into the M003 worktree at `internal/auto/`. This includes: `engine.go`, `engine_test.go`, `engine_integration_test.go`, `events.go`, `init.go`, `init_test.go`, `init_tools.go`, `init_tools_test.go`, `lock.go`, `lock_test.go`, `milestone.go`, `prompts.go`, `prompts_test.go`, `slice.go`, `state.go`, `state_test.go`, `status.go`, `task.go`, `unit.go`, and the `templates/` directory with all `.md.tpl` files.
2. Run `go build ./internal/auto/` to verify the consolidated files compile in the M003 worktree. Fix any import issues.
3. Add an `AutoConfig` struct to `internal/config/config.go`:
   ```go
   type AutoConfig struct {
       VerificationCommands []string `json:"verification_commands,omitempty" jsonschema:"description=Shell commands to run after each task execution for verification"`
       BudgetCeiling        float64  `json:"budget_ceiling,omitempty" jsonschema:"description=Maximum dollar cost before auto-mode pauses"`
       StuckThreshold       int      `json:"stuck_threshold,omitempty" jsonschema:"description=Number of consecutive failures before stuck detection triggers,default=5"`
       WorktreeMode         string   `json:"worktree_mode,omitempty" jsonschema:"description=Git worktree isolation mode for auto-mode execution"`
   }
   ```
4. Add an `Auto *AutoConfig` field to the `Config` struct: `Auto *AutoConfig \`json:"auto,omitempty\"\``
5. Add a test in `internal/config/load_test.go` (or a new `internal/config/auto_test.go`) that round-trips a crush.json with an `auto` section through `loadFromBytes` and asserts the `VerificationCommands` field is populated.
6. Run `go vet ./internal/auto/ ./internal/config/` and `go test ./internal/config/ -run TestAutoConfig -count=1`.
7. Format with `gofumpt -w internal/config/config.go internal/config/auto_test.go`.

## Must-Haves

- [ ] All M002 auto package files exist in M003 worktree at `internal/auto/`
- [ ] `go build ./internal/auto/` succeeds
- [ ] `AutoConfig` struct with all four fields exists on `Config`
- [ ] Config parsing test passes for `auto.verification_commands`

## Verification

- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go build ./internal/auto/`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/config/ -run TestAutoConfig -count=1 -v`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go vet ./internal/auto/ ./internal/config/`
  - Estimate: 45m
  - Files: internal/auto/*.go, internal/auto/templates/*.md.tpl, internal/config/config.go, internal/config/auto_test.go
  - Verify: cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go build ./internal/auto/ && go test ./internal/config/ -run TestAutoConfig -count=1 -v && go vet ./internal/auto/ ./internal/config/
- [x] **T02: Added Verifier interface, ShellVerifier implementation, and verification gate in engine step() with single-retry on failure** — ## Description

This task implements the verification gate: a `Verifier` interface, a shell-based implementation, and the engine modifications to run verification after task dispatch with retry-on-failure. This delivers R010.

## Failure Modes

| Dependency | On error | On timeout | On malformed response |
|------------|----------|-----------|----------------------|
| Shell.Exec (verification command) | Treat as verification failure — capture exit code + stderr | Context cancellation propagates — treat as failure | N/A — output is opaque string |

## Negative Tests

- **Malformed inputs**: Empty verification commands list (should skip verification), command that doesn't exist (should fail gracefully with error message)
- **Error paths**: Command returns non-zero exit code (verification failure), command times out via context cancellation
- **Boundary conditions**: Single command vs multiple commands, very long output (must be truncated for diagnostic prompt)

## Steps

1. Create `internal/auto/verify.go` with:
   - `VerificationResult` struct: `Command string`, `ExitCode int`, `Stdout string`, `Stderr string`, `Duration time.Duration`, `Passed bool`
   - `Verifier` interface: `RunVerification(ctx context.Context, workingDir string) ([]VerificationResult, error)`
   - `ShellVerifier` struct implementing `Verifier` that takes `[]string` commands, creates a `shell.Shell` per invocation, runs each command via `Shell.Exec()`, and collects results. Stop on first failure (short-circuit).
   - `truncateOutput(s string, maxLen int) string` helper — truncate to last `maxLen` bytes (default 4096) to keep diagnostic prompts reasonable.
2. Add new event type constants to `internal/auto/events.go`: `EventVerificationStarted`, `EventVerificationPassed`, `EventVerificationFailed`.
3. Add a `verifier Verifier` field to the `Engine` struct and a corresponding parameter in `NewEngine()`. When `verifier` is nil (no verification commands configured), skip verification entirely.
4. Modify `engine.step()` to insert verification after dispatch succeeds but before `AdvanceStatus()`:
   - Only run verification for `UnitExecuteTask` units (not research, planning, summarize, validate).
   - Publish `EventVerificationStarted`.
   - Call `e.verifier.RunVerification(ctx, e.dataDir)`.
   - If all pass: publish `EventVerificationPassed`, proceed to `AdvanceStatus()`.
   - If any fail: publish `EventVerificationFailed` with truncated output in message, then re-dispatch with a diagnostic prompt containing the failure output. Run verification again after retry.
   - If retry also fails: publish `EventVerificationFailed` again, return error (do NOT advance status). The engine's existing error-handling loop will apply backoff.
5. Create `internal/auto/verify_test.go` with unit tests:
   - `TestShellVerifier_AllPass` — two commands that succeed
   - `TestShellVerifier_FirstFails` — first command fails, second is not run
   - `TestShellVerifier_EmptyCommands` — no commands configured, returns empty results
   - `TestVerifier_TruncateOutput` — output longer than limit is truncated
6. Run `go vet ./internal/auto/` and format with `gofumpt -w internal/auto/`.

## Must-Haves

- [ ] `Verifier` interface and `ShellVerifier` implementation in `internal/auto/verify.go`
- [ ] `VerificationResult` struct with Command, ExitCode, Stdout, Stderr, Duration, Passed fields
- [ ] Engine `step()` calls verification after task dispatch, before advance
- [ ] On failure, engine re-dispatches with diagnostic prompt containing truncated failure output
- [ ] On retry failure, engine does not advance status
- [ ] New event types: EventVerificationStarted, EventVerificationPassed, EventVerificationFailed
- [ ] Unit tests for ShellVerifier pass/fail/empty/truncate

## Verification

- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run TestShellVerifier -count=1 -v`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run TestVerifier -count=1 -v`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go vet ./internal/auto/`

## Observability Impact

- Signals added: EventVerificationStarted/Passed/Failed events via pubsub; slog.Info/Error with command name, exit code, truncated output
- How a future agent inspects this: subscribe to pubsub broker for verification events; check Engine.Status().LastError
- Failure state exposed: VerificationResult captures per-command exit code, stdout, stderr, duration
  - Estimate: 1h30m
  - Files: internal/auto/verify.go, internal/auto/verify_test.go, internal/auto/events.go, internal/auto/engine.go
  - Verify: cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run 'TestShellVerifier|TestVerifier' -count=1 -v && go vet ./internal/auto/
- [x] **T03: Added three integration tests proving verify→retry→succeed, verify→retry→fail, and verify-skipped-for-non-task engine paths** — ## Description

This task writes the integration tests that prove the slice's demo: the full verify→retry→succeed path (first dispatch fails verification, retry dispatch passes) and the verify→retry→fail path (both dispatches fail verification, engine does not advance). These are the objective stopping condition for the slice.

## Steps

1. Create or extend `internal/auto/engine_integration_test.go` (in the M003 worktree) with two integration tests:

2. `TestIntegration_VerifyRetrySucceed`:
   - Set up a mock querier that returns a single `UnitExecuteTask` unit.
   - Set up a `mockVerifier` that fails on the first call and succeeds on the second call.
   - Set up a mock dispatcher that records all calls (including the retry dispatch with diagnostic prompt).
   - Set up a mock advancer coupled to the querier (per K006/K008).
   - Create engine with the mock verifier, run `engine.Step()`.
   - Assert: dispatcher called exactly twice (original + retry with diagnostic).
   - Assert: advancer called exactly once (status advanced after successful retry).
   - Assert: events published in order: UnitStarted → VerificationStarted → VerificationFailed → VerificationStarted → VerificationPassed → UnitCompleted.
   - Assert: the retry dispatch prompt contains truncated failure output from the first verification.

3. `TestIntegration_VerifyRetryFail`:
   - Same setup, but mockVerifier fails on both calls.
   - Create engine, run `engine.Step()`.
   - Assert: dispatcher called exactly twice (original + retry).
   - Assert: advancer NOT called (status not advanced).
   - Assert: events include VerificationFailed twice, no UnitCompleted.
   - Assert: `engine.Step()` returns an error.

4. `TestIntegration_VerifySkippedForNonTaskUnits`:
   - Set up a mock querier that returns a `UnitResearch` or `UnitPlanSlice` unit.
   - Set up a mock verifier that would fail if called.
   - Assert: verifier is never called for non-task units.
   - Assert: advancer is called (dispatch completes normally).

5. Add a helper `mockVerifier` struct that takes a sequence of `[]VerificationResult` responses and returns them in order.

6. Run the full test suite: `go test ./internal/auto/ -count=1 -v`
7. Format: `gofumpt -w internal/auto/engine_integration_test.go`

## Must-Haves

- [ ] `TestIntegration_VerifyRetrySucceed` passes — proves dispatch→fail→retry→succeed→advance path
- [ ] `TestIntegration_VerifyRetryFail` passes — proves dispatch→fail→retry→fail→no-advance path
- [ ] `TestIntegration_VerifySkippedForNonTaskUnits` passes — proves verification only runs for execute_task
- [ ] All existing M002 engine tests still pass

## Verification

- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run TestIntegration_Verification -count=1 -v`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -count=1`
  - Estimate: 1h
  - Files: internal/auto/engine_integration_test.go
  - Verify: cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run 'TestIntegration_Verify' -count=1 -v && go test ./internal/auto/ -count=1
