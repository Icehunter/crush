---
estimated_steps: 30
estimated_files: 1
skills_used: []
---

# T02: Write toggle tests covering all state transitions, guards, and error paths

Write comprehensive tests for the auto-mode toggle keybinding covering key matching, nil guards, all state transitions, error paths, and sidebar regression.

## Steps

1. **Create test file** `internal/ui/model/auto_toggle_test.go` with the following tests:

2. **Mock AutoController**: Define a `mockAutoController` struct in the test file implementing `AutoController`. Track calls via fields: `startCalled bool`, `pauseCalled bool`, `resumeCalled bool`, `statusVal string`, `startErr error`, `pauseErr error`, `resumeErr error`.

3. **Test: `TestAutoToggle_KeyMatches`** — Verify `key.Matches(tea.KeyPressMsg{...}, keyMap.Auto.Toggle)` matches `ctrl+a`.

4. **Test: `TestAutoToggle_NilController`** — Set `autoController = nil`, call `toggleAutoMode()`, verify it returns a non-nil cmd (warn notification) and doesn't panic.

5. **Test: `TestAutoToggle_NoSession`** — Set controller but `session = nil`, call `toggleAutoMode()`, verify it returns a warn notification.

6. **Test: `TestAutoToggle_IdleToStart`** — Set `autoSnapshot = nil` (idle), `autoMilestoneID = "M001"`, mock controller. Call `toggleAutoMode()`, execute the returned cmd. Verify `startCalled == true`.

7. **Test: `TestAutoToggle_IdleNoMilestone`** — Set `autoSnapshot = nil`, `autoMilestoneID = ""`. Call `toggleAutoMode()`, verify it returns a warn notification without calling StartAuto.

8. **Test: `TestAutoToggle_RunningToPause`** — Set `autoSnapshot.Status = "running"`. Call `toggleAutoMode()`, execute cmd. Verify `pauseCalled == true`.

9. **Test: `TestAutoToggle_PausedToResume`** — Set `autoSnapshot.Status = "paused"`. Call `toggleAutoMode()`, execute cmd. Verify `resumeCalled == true`.

10. **Test: `TestAutoToggle_StartError`** — Set mock to return error from `StartAuto`. Call and execute. Verify error is surfaced.

11. **Run S01 regression**: `go test ./internal/ui/model/... -run TestAutoModeInfo` to confirm sidebar tests still pass.

12. **Format** with `goimports` or `gofmt`.

## Must-Haves

- [ ] mockAutoController implements AutoController interface
- [ ] Test for ctrl+a key matching
- [ ] Test for nil controller guard (no panic)
- [ ] Test for no-session guard
- [ ] Test for idle→start transition
- [ ] Test for no-milestone-configured error
- [ ] Test for running→pause transition
- [ ] Test for paused→resume transition
- [ ] Test for start error propagation
- [ ] All tests use t.Parallel()
- [ ] S01 sidebar tests still pass (regression)

## Verification

- `go test ./internal/ui/model/... -v -run TestAutoToggle` — all toggle tests pass (8+ tests)
- `go test ./internal/ui/model/... -v -run TestAutoModeInfo` — S01 sidebar tests still pass
- `go vet ./...` — clean

## Inputs

- ``internal/ui/model/auto_controller.go` — AutoController interface to mock`
- ``internal/ui/model/auto_toggle.go` — toggleAutoMode() method to test`
- ``internal/ui/model/keys.go` — Auto.Toggle binding to verify key matching`
- ``internal/ui/model/ui.go` — UI struct fields (autoController, autoSnapshot, autoMilestoneID, session)`
- ``internal/ui/model/sidebar_auto_test.go` — existing test patterns and newTestUIForAuto helper`

## Expected Output

- ``internal/ui/model/auto_toggle_test.go` — new test file with 8+ tests covering all toggle states and guards`

## Verification

cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M004 && go test ./internal/ui/model/... -v -run 'TestAutoToggle|TestAutoModeInfo' -count=1 2>&1 | tail -20
