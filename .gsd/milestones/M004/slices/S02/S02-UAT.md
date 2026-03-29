# S02: TUI Start/Pause/Resume Keybindings — UAT

**Milestone:** M004
**Written:** 2026-03-28T06:22:01.450Z

# S02: TUI Start/Pause/Resume Keybindings — UAT

**Milestone:** M004
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: The keybinding, interface, dispatch logic, and all state transitions are fully exercised by 11 automated tests. No runtime auto-mode engine exists yet to test live, so artifact verification is the appropriate mode.

## Preconditions

- Crush codebase builds cleanly (`go build ./...`)
- Working directory is the M004 worktree

## Smoke Test

Run `go test ./internal/ui/model/... -run TestAutoToggle_IdleToStart -count=1` — should pass, confirming the basic toggle path works.

## Test Cases

### 1. ctrl+a key matching

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_KeyMatches -v -count=1`
2. **Expected:** Test passes — `key.Matches(tea.KeyPressMsg{}, keyMap.Auto.Toggle)` matches ctrl+a

### 2. Nil controller guard

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_NilController -v -count=1`
2. **Expected:** Test passes — toggleAutoMode returns a warn cmd without panicking when autoController is nil

### 3. No session guard

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_NoSession -v -count=1`
2. **Expected:** Test passes — toggleAutoMode returns a warn cmd when session is nil

### 4. Idle to start transition

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_IdleToStart -v -count=1`
2. **Expected:** Test passes — with nil autoSnapshot and a milestone ID set, StartAuto is called on the controller

### 5. No milestone configured

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_IdleNoMilestone -v -count=1`
2. **Expected:** Test passes — toggleAutoMode returns a warn without calling StartAuto

### 6. Running to pause transition

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_RunningToPause -v -count=1`
2. **Expected:** Test passes — with autoSnapshot.Status="running", PauseAuto is called

### 7. Paused to resume transition

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_PausedToResume -v -count=1`
2. **Expected:** Test passes — with autoSnapshot.Status="paused", ResumeAuto is called

### 8. Error propagation

1. Run `go test ./internal/ui/model/... -run 'TestAutoToggle_StartError|TestAutoToggle_PauseError|TestAutoToggle_ResumeError' -v -count=1`
2. **Expected:** All 3 tests pass — errors from controller methods are surfaced as error notifications

### 9. S01 sidebar regression

1. Run `go test ./internal/ui/model/... -run TestAutoModeInfo -v -count=1`
2. **Expected:** All 5 TestAutoModeInfo_* tests pass — sidebar rendering unaffected by S02 changes

## Edge Cases

### Unknown status value

1. Run `go test ./internal/ui/model/... -run TestAutoToggle_UnknownStatus -v -count=1`
2. **Expected:** Test passes — toggleAutoMode returns a warn notification for unrecognized status strings

## Failure Signals

- Any TestAutoToggle_* test failing indicates broken toggle logic
- Any TestAutoModeInfo_* test failing indicates S02 regressed S01 sidebar rendering
- `go build ./...` failure indicates compilation errors in new files
- `go vet ./...` failure indicates code quality issues

## Not Proven By This UAT

- Live auto-mode engine integration (no production AutoController implementation exists yet)
- Visual rendering of ctrl+a help text in the TUI
- End-to-end flow of pressing ctrl+a and seeing sidebar update

## Notes for Tester

The AutoController interface is defined but not yet wired to a real implementation. All tests use a mockAutoController. Live testing will be possible once the engine adapter is built.
