---
id: S02
parent: M004
milestone: M004
provides:
  - AutoController interface for auto-mode engine to implement
  - ctrl+a keybinding wired into handleGlobalKeys
  - toggleAutoMode() dispatch logic consuming autoSnapshot.Status
requires:
  - slice: S01
    provides: autoSnapshot field and AutoModeSnapshot type on UI struct
affects:
  - S03
key_files:
  - internal/ui/model/keys.go
  - internal/ui/model/auto_controller.go
  - internal/ui/model/auto_toggle.go
  - internal/ui/model/ui.go
  - internal/ui/model/auto_toggle_test.go
key_decisions:
  - State derived from autoSnapshot.Status field rather than calling AutoStatus() to keep Update() side-effect-free
  - All auto-mode engine calls wrapped in tea.Cmd closures per Bubble Tea convention
  - Added extra tests beyond plan (PauseError, ResumeError, UnknownStatus) for full error path coverage
patterns_established:
  - AutoController interface pattern: TUI depends on a 4-method interface for auto-mode control, enabling mock testing and decoupled wiring
  - Toggle keybinding pattern: single key cycles through idle→start→pause→resume states with guard clauses and notification feedback
observability_surfaces:
  - Notification feedback via util.ReportInfo/ReportWarn surfaces auto-mode state changes to the user
drill_down_paths:
  - .gsd/milestones/M004/slices/S02/tasks/T01-SUMMARY.md
  - .gsd/milestones/M004/slices/S02/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-28T06:22:01.450Z
blocker_discovered: false
---

# S02: TUI Start/Pause/Resume Keybindings

**ctrl+a keybinding toggles auto-mode through idle→start, running→pause, paused→resume with notification feedback, backed by 11 tests**

## What Happened

T01 added the core infrastructure: Auto.Toggle keybinding (ctrl+a) in KeyMap, an AutoController interface (StartAuto/PauseAuto/ResumeAuto/AutoStatus) in its own file, a toggleAutoMode() method that derives state from autoSnapshot.Status and dispatches engine calls inside tea.Cmd closures, and wiring into handleGlobalKeys plus ShortHelp/FullHelp. State derivation uses the snapshot Status field rather than calling AutoStatus() to keep Update() side-effect-free per Bubble Tea convention. Guard clauses handle nil controller, nil session, and missing milestone ID gracefully via util.ReportWarn.

T02 added comprehensive test coverage: a mockAutoController implementing the interface, and 11 parallel tests covering key matching, nil controller guard, no session guard, idle→start, no-milestone-configured, running→pause, paused→resume, start/pause/resume error propagation, and unknown status handling. Three tests beyond the plan (PauseError, ResumeError, UnknownStatus) were added for complete error path coverage. All S01 sidebar regression tests continue to pass.

## Verification

go build ./... — exit 0, compiles clean. go vet ./... — exit 0. grep confirms ctrl+a in keys.go, AutoController in auto_controller.go, toggleAutoMode in auto_toggle.go. 11 TestAutoToggle_* tests pass. 5 TestAutoModeInfo_* S01 regression tests pass.

## Requirements Advanced

- R016 — ctrl+a keybinding registered in TUI with toggleAutoMode handler dispatching start/pause/resume based on auto-mode state. Interface and wiring complete; awaits production AutoController implementation.

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

T02 added 3 extra tests (PauseError, ResumeError, UnknownStatus) beyond the plan for complete error path coverage. No other deviations.

## Known Limitations

AutoController interface is defined but no production implementation exists yet — it will be wired when the auto-mode engine is connected to the TUI. autoMilestoneID must be set externally; no UI for selecting a milestone.

## Follow-ups

Wire a real AutoController implementation (the auto-mode engine) to the UI's autoController field. Add milestone selection UI or config-based milestone targeting.

## Files Created/Modified

- `internal/ui/model/keys.go` — Added Auto.Toggle keybinding (ctrl+a) to KeyMap struct and DefaultKeyMap()
- `internal/ui/model/auto_controller.go` — New file: AutoController interface with StartAuto, PauseAuto, ResumeAuto, AutoStatus
- `internal/ui/model/auto_toggle.go` — New file: toggleAutoMode() method with state-based dispatch and guard clauses
- `internal/ui/model/ui.go` — Added autoController/autoMilestoneID fields, wired toggle into handleGlobalKeys and help text
- `internal/ui/model/auto_toggle_test.go` — New file: 11 parallel tests with mockAutoController covering all toggle paths
