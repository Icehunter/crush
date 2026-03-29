---
id: M004
title: "M004: TUI Integration + Git Worktrees"
status: complete
completed_at: 2026-03-28T06:53:13.153Z
key_decisions:
  - D012: Single ctrl+a toggle for auto-mode — idle→start, running→pause, paused→resume. Mnemonic, conserves keybind space.
  - D013: Only per-milestone worktree mode in MVP. Per-slice too complex for initial implementation.
key_files:
  - internal/auto/event.go — AutoEventType constants, AutoEvent, AutoSnapshot, SliceProgress structs
  - internal/auto/worktree.go — WorktreeManager with Create/Merge/Remove/Exists/EnsureGit
  - internal/app/auto_events.go — autoBroker, SubscribeAutoEvents, PublishAutoEvent, AutoBroker
  - internal/ui/model/auto_controller.go — AutoController interface (StartAuto/PauseAuto/ResumeAuto/AutoStatus)
  - internal/ui/model/auto_toggle.go — toggleAutoMode() state-based dispatch
  - internal/ui/model/sidebar.go — autoModeInfo() sidebar rendering method
  - internal/ui/model/keys.go — Auto.Toggle keybinding (ctrl+a)
  - internal/ui/model/ui.go — autoSnapshot field, autoController field, event handler
  - internal/config/config.go — WorktreeMode field in Options struct
lessons_learned:
  - Following existing patterns (LSP event wiring) for new features reduces integration risk and code review friction — auto events exactly mirror LSP events.
  - Single-key toggle UIs work well when sidebar already displays current state — user always knows what the toggle will do.
  - Testing git worktree operations against real repos via t.TempDir() catches edge cases (branch-exists fallback, nothing-to-commit variants) that mocks would miss.
  - gofumpt unavailability in worktrees is a recurring friction point — goimports works as fallback but formatter consistency should be addressed.
---

# M004: M004: TUI Integration + Git Worktrees

**Delivered TUI sidebar auto-mode panel, ctrl+a start/pause/resume keybinding, and git worktree isolation for milestones — making auto-mode fully visible and controllable from within the Crush TUI.**

## What Happened

M004 delivered three slices integrating auto-mode into the Crush TUI experience and adding optional git worktree isolation.

**S01 (Event Wiring + Sidebar Panel)** established the event contract: 8 AutoEventType constants, AutoEvent/AutoSnapshot/SliceProgress structs in internal/auto/event.go, a pubsub.Broker wired through App via setupEvents() (following the existing LSP event pattern exactly), and a sidebar rendering method that shows milestone tree with status icons, slice progress fractions, active unit, cost, and elapsed time. The sidebar inserts between header and files, reducing file/LSP/MCP space when active. 3 event tests + 3 broker tests + 5 sidebar rendering tests.

**S02 (TUI Start/Pause/Resume Keybindings)** added ctrl+a as a single toggle keybinding (D012) that cycles idle→start, running→pause, paused→resume. An AutoController interface decouples the TUI from the engine implementation. State derivation uses autoSnapshot.Status to keep Update() side-effect-free per Bubble Tea convention. Guard clauses handle nil controller, nil session, and missing milestone gracefully. 11 tests cover all paths including error propagation.

**S03 (Git Worktree Isolation)** implemented WorktreeManager with Create/Merge/Remove/Exists operations using exec.CommandContext for git commands. Worktrees are created at .crush/worktrees/<MID>/ on auto/<MID> branches. Merge uses squash-merge with idempotent nothing-to-commit handling. Remove is idempotent via isNotFoundError helper. WorktreeMode config field added to Options. 10 tests validate the full lifecycle against real git repos, including branch-exists fallback and cleanup verification.

All 32 tests pass (3 event + 3 broker + 5 sidebar + 11 toggle + 10 worktree). Build and vet are clean.

## Success Criteria Results

- **Sidebar shows live auto-mode progress:** ✅ autoModeInfo() renders milestone tree with status icons (▶/⏸/✓/✗), slice progress fractions, active unit, cost, elapsed time. 5 tests (TestAutoModeInfo_Nil, _Running, _Paused, _Truncation, _EmptySlices) prove all rendering states.
- **ctrl+a toggles auto-mode start/pause/resume:** ✅ Single keybinding cycles through idle→start→pause→resume with guard clauses. 11 tests (TestAutoToggle_*) prove all state transitions and error paths.
- **Git worktree created at .crush/worktrees/<MID>/ with squash-merge back:** ✅ WorktreeManager.Create/Merge/Remove/Exists implemented. TestWorktreeManager_FullLifecycle proves create→commit→merge→verify-on-main→remove→verify-cleanup lifecycle against real git repos.

## Definition of Done Results

- **All slices complete:** ✅ S01, S02, S03 all have SUMMARY.md with verification_result: passed
- **All tests pass:** ✅ 32 tests across 4 packages (auto, app, ui/model) all pass
- **Build clean:** ✅ `go build ./...` exits 0
- **Vet clean:** ✅ `go vet ./...` exits 0 (includes csync/maps.go fix from S01)
- **Cross-slice integration:** ✅ S01's autoSnapshot type consumed by S02's toggleAutoMode for state derivation

## Requirement Outcomes

- **R015** (active → advanced): Sidebar panel renders milestone tree with status icons, slice progress fractions, active unit, cost, and elapsed time. 5 unit tests verify rendering for all states.
- **R016** (active → advanced): ctrl+a keybinding registered in TUI with toggleAutoMode handler dispatching start/pause/resume based on auto-mode state. Interface and wiring complete; awaits production AutoController implementation.
- **R017** (active → advanced): AutoEvent types with 8 EventType constants defined. Broker wired through App via setupEvents(). UI subscribes and stores snapshots. 3 broker tests verify publish/subscribe lifecycle.
- **R018** (active → validated): Full implementation of WorktreeManager with Create/Merge/Remove/Exists at .crush/worktrees/<MID>/ with squash-merge back. 10-test suite proves full lifecycle including error paths.

## Deviations

S02 added 3 extra tests beyond plan (PauseError, ResumeError, UnknownStatus) for complete error coverage. S03 T02 fixed two bugs in worktree.go during test development (stderr in runGit output, dual nothing-to-commit variants). gofumpt not on PATH — goimports used as fallback formatter.

## Follow-ups

Wire production AutoController implementation to UI's autoController field when auto-mode engine merges. Integrate WorktreeManager into engine loop (Create at milestone start, Merge+Remove at completion). Add milestone selection UI or config-based milestone targeting. Add conflict resolution strategy for worktree merges (pause and surface to user).
