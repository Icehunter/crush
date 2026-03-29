---
estimated_steps: 47
estimated_files: 4
skills_used: []
---

# T02: Implement verification runner and wire into engine dispatch flow with retry

## Description

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

## Inputs

- ``internal/auto/engine.go` — engine step() to modify with verification gate`
- ``internal/auto/events.go` — event types to extend`
- ``internal/shell/shell.go` — Shell.Exec() for running verification commands`
- ``internal/config/config.go` — AutoConfig.VerificationCommands for command list`

## Expected Output

- ``internal/auto/verify.go` — Verifier interface, ShellVerifier implementation, VerificationResult struct, truncateOutput helper`
- ``internal/auto/verify_test.go` — unit tests for ShellVerifier (pass, fail, empty, truncate)`
- ``internal/auto/events.go` — extended with EventVerificationStarted/Passed/Failed constants`
- ``internal/auto/engine.go` — modified step() with verification gate and retry logic`

## Verification

cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/auto/ -run 'TestShellVerifier|TestVerifier' -count=1 -v && go vet ./internal/auto/
