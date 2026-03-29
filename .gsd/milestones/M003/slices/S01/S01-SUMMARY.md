---
id: S01
parent: M003
milestone: M003
provides:
  - Verifier interface and ShellVerifier implementation
  - AutoConfig struct with verification_commands, budget_ceiling, stuck_threshold, worktree_mode
  - Engine verification gate with single-retry-on-failure
  - Three verification event types for downstream subscribers
requires:
  []
affects:
  - S02
  - S03
key_files:
  - internal/auto/verify.go
  - internal/auto/verify_test.go
  - internal/auto/events.go
  - internal/auto/engine.go
  - internal/auto/engine_integration_test.go
  - internal/config/config.go
  - internal/config/auto_test.go
  - internal/csync/maps.go
key_decisions:
  - Short-circuit verification on first command failure rather than running all commands
  - Truncate output to last 4096 bytes (tail) for diagnostic prompt relevance
  - Verification only runs for UnitExecuteTask units — research/plan/summarize/validate skip it
  - Fixed pre-existing go vet failure (value receiver on sync.RWMutex) in csync/maps.go
patterns_established:
  - Verifier interface pattern: pluggable verification with mockVerifier for tests
  - Engine gate pattern: post-dispatch hooks that run between dispatch and advance
  - Diagnostic retry pattern: on verification failure, re-dispatch with truncated failure output appended to prompt
observability_surfaces:
  - EventVerificationStarted/Passed/Failed events via pubsub broker
  - slog.Info/Error with command name, exit code, truncated output for each verification command
  - VerificationResult struct captures per-command exit code, stdout, stderr, duration
drill_down_paths:
  - .gsd/milestones/M003/slices/S01/tasks/T01-SUMMARY.md
  - .gsd/milestones/M003/slices/S01/tasks/T02-SUMMARY.md
  - .gsd/milestones/M003/slices/S01/tasks/T03-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-28T05:05:47.871Z
blocker_discovered: false
---

# S01: Auto Config + Verification Gates

**Added AutoConfig to crush.json, implemented Verifier interface with ShellVerifier, wired verification gate into engine step() with single-retry on failure, and proved all paths with integration tests.**

## What Happened

This slice delivered two core capabilities: (1) the `auto` configuration section in crush.json with four fields (verification_commands, budget_ceiling, stuck_threshold, worktree_mode), and (2) a verification gate that runs configured shell commands after each task execution with automatic retry on failure.

T01 copied the M002 auto package (19 Go files + 6 templates) and required DB dependencies into the M003 worktree, then added the AutoConfig struct to the config system with round-trip parsing tests (3 tests).

T02 implemented the Verifier interface and ShellVerifier in internal/auto/verify.go, added three new event types (EventVerificationStarted/Passed/Failed), and wired the verification gate into engine.step() — verification runs only for UnitExecuteTask units, short-circuits on first command failure, and truncates output to 4096 bytes for diagnostic prompts. 7 unit tests cover pass/fail/empty/truncate/context-cancel/nonexistent-command paths.

T03 added three integration tests proving the full engine paths: verify→retry→succeed (mock verifier fails then succeeds, dispatcher called twice, advancer called once), verify→retry→fail (both attempts fail, advancer not called, error returned), and verify-skipped-for-non-task (research/plan units bypass verification entirely).

A pre-existing `go vet` failure in internal/csync/maps.go (value receiver on struct containing sync.RWMutex) was fixed during slice closure by changing JSONSchemaAlias to a pointer receiver.

## Verification

All verification gates pass:
- `go vet ./...` — clean (0 exit)
- `go build ./internal/auto/` — clean (0 exit)
- `go test ./internal/auto/ -run 'TestShellVerifier|TestVerifier|TestFormatFailure|TestIntegration_Verify' -count=1 -v` — 10/10 pass
- `go test ./internal/config/ -run TestAutoConfig -count=1 -v` — 3/3 pass

## Requirements Advanced

- R010 — Verifier interface, ShellVerifier, engine gate with retry, and integration tests proving verify→retry→succeed and verify→retry→fail paths
- R014 — AutoConfig struct with all four fields added to Config, round-trip parsing tests pass

## Requirements Validated

- R010 — Integration tests TestIntegration_VerifyRetrySucceed and TestIntegration_VerifyRetryFail prove full verify→retry→succeed and verify→retry→fail paths. ShellVerifier unit tests prove command execution, short-circuit, truncation.
- R014 — TestAutoConfig, TestAutoConfig_Empty, TestAutoConfig_PartialFields prove all four auto config fields round-trip through crush.json parsing.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

T01 required copying M002 DB artifacts (db.go, models.go, querier.go, 3 sqlc files, 3 SQL sources, 1 migration) beyond what the task plan listed. Fixed pre-existing go vet failure in internal/csync/maps.go during slice closure.

## Known Limitations

ShellVerifier creates a new shell.Shell per command invocation — no shell state reuse across commands. Verification retry is hardcoded to 1 attempt (not configurable). Budget ceiling and stuck threshold fields are defined in config but not yet enforced (S02 and S03 scope).

## Follow-ups

S02 will wire budget_ceiling enforcement. S03 will wire stuck_threshold and context pressure. Verification retry count could be made configurable in a future slice.

## Files Created/Modified

- `internal/auto/verify.go` — New file: Verifier interface, ShellVerifier, VerificationResult, truncateOutput, formatFailureDiagnostic
- `internal/auto/verify_test.go` — New file: 7 unit tests for ShellVerifier and truncation
- `internal/auto/events.go` — Added EventVerificationStarted, EventVerificationPassed, EventVerificationFailed constants
- `internal/auto/engine.go` — Added verifier field, verification gate in step() with retry logic
- `internal/auto/engine_integration_test.go` — Added 3 integration tests: VerifyRetrySucceed, VerifyRetryFail, VerifySkippedForNonTaskUnits
- `internal/config/config.go` — Added AutoConfig struct and Auto field on Config
- `internal/config/auto_test.go` — New file: 3 config parsing tests
- `internal/csync/maps.go` — Fixed value receiver to pointer receiver on JSONSchemaAlias (go vet fix)
