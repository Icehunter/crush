# S01: Auto Config + Verification Gates — UAT

**Milestone:** M003
**Written:** 2026-03-28T05:05:47.871Z

# S01 UAT: Auto Config + Verification Gates

## Preconditions
- M003 worktree checked out at `/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003`
- Go toolchain available (go 1.24+)

## Test Case 1: AutoConfig Parsing
**Steps:**
1. Run `go test ./internal/config/ -run TestAutoConfig -count=1 -v`
**Expected:** 3/3 tests pass — full round-trip, empty config, partial fields all parse correctly.

## Test Case 2: ShellVerifier Unit Tests
**Steps:**
1. Run `go test ./internal/auto/ -run 'TestShellVerifier' -count=1 -v`
**Expected:** 5/5 tests pass:
- AllPass: two `echo` commands both succeed, 2 results returned with Passed=true
- FirstFails: `false` command fails, second command not executed (short-circuit)
- EmptyCommands: no commands → empty results, no error
- NonexistentCommand: bogus command → Passed=false, stderr contains "not found"
- ContextCancelled: cancelled context → error returned

## Test Case 3: Truncation and Diagnostics
**Steps:**
1. Run `go test ./internal/auto/ -run 'TestVerifier_TruncateOutput|TestFormatFailure' -count=1 -v`
**Expected:** 5/5 tests pass — short/exact/long/zero-maxLen strings truncated correctly; diagnostic format includes command, exit code, and truncated output.

## Test Case 4: Integration — Verify → Retry → Succeed
**Steps:**
1. Run `go test ./internal/auto/ -run TestIntegration_VerifyRetrySucceed -count=1 -v`
**Expected:** Test passes. Dispatcher called exactly twice (original + retry with diagnostic). Advancer called once. Events in order: UnitStarted → VerificationStarted → VerificationFailed → VerificationStarted → VerificationPassed → UnitCompleted.

## Test Case 5: Integration — Verify → Retry → Fail
**Steps:**
1. Run `go test ./internal/auto/ -run TestIntegration_VerifyRetryFail -count=1 -v`
**Expected:** Test passes. Dispatcher called twice. Advancer NOT called. Two VerificationFailed events, no UnitCompleted. Step returns error.

## Test Case 6: Integration — Verification Skipped for Non-Task Units
**Steps:**
1. Run `go test ./internal/auto/ -run TestIntegration_VerifySkippedForNonTaskUnits -count=1 -v`
**Expected:** Test passes. Verifier never called. Advancer called (dispatch completes normally).

## Test Case 7: Full Build and Vet
**Steps:**
1. Run `go build ./internal/auto/`
2. Run `go vet ./...`
**Expected:** Both exit 0 with no output.

## Edge Cases
- **No verification commands configured**: Engine creates nil verifier, verification step is skipped entirely — tested by existing M002 engine tests continuing to pass.
- **Pre-existing vet failures**: The csync/maps.go value-receiver issue was fixed as part of this slice (pointer receiver on JSONSchemaAlias).
