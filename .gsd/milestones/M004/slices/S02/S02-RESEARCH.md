# S02: TUI Start/Pause/Resume Keybindings — Research

**Researched:** 2026-03-28
**Status:** Complete
**Depth:** Light — established patterns, straightforward wiring

## Summary

S02 adds a `ctrl+a` keybinding to toggle auto-mode (idle→start, running→pause, paused→resume) from the TUI. S01 already delivered all event wiring, sidebar rendering, and the `autoSnapshot` field on UI. The remaining work is: (1) add an `Auto` group to `KeyMap` with a single toggle binding, (2) handle the keypress in `Update()` to call engine control methods, (3) expose engine access through `App`, and (4) update sidebar status to reflect the toggle state.

The engine (`internal/auto/engine.go` with `Run`/`Pause`/`Stop`/`Status`) does **not exist in this worktree** — it's being built in M002/M003 branches. The keybinding implementation must define a minimal interface or stub for what it calls, so the TUI can compile and be tested independently. The actual engine integration happens when branches merge.

## Requirement Targets

- **R016** (primary) — Keybinding in TUI to start, pause, and resume auto-mode from an active session.
- **D012** — Single `ctrl+a` toggle: idle→start, running→pause, paused→resume. Mnemonic for "auto".

## Implementation Landscape

### KeyMap Addition

**File:** `internal/ui/model/keys.go` (271 lines)

Add an `Auto` struct group to `KeyMap`:
```go
Auto struct {
    Toggle key.Binding
}
```
And in `DefaultKeyMap()`:
```go
km.Auto.Toggle = key.NewBinding(
    key.WithKeys("ctrl+a"),
    key.WithHelp("ctrl+a", "auto"),
)
```

`ctrl+a` is confirmed free — not used by any existing binding in the file. The pattern follows existing groups (`Editor`, `Chat`, `Initialize`).

### Key Handling in Update

**File:** `internal/ui/model/ui.go`

The toggle should be handled in `handleKeyPressMsg()` inside the `handleGlobalKeys` closure (lines ~1632-1695), since it should work regardless of focus state (editor or chat). Insert a new case:
```go
case key.Matches(msg, m.keyMap.Auto.Toggle):
    cmds = append(cmds, m.toggleAutoMode())
    return true
```

The `toggleAutoMode()` method inspects `m.autoSnapshot.Status` (or checks if nil for idle) and dispatches accordingly:
- **nil / no snapshot**: Start auto-mode → `tea.Cmd` that calls engine.Run()
- **"running"**: Pause → `tea.Cmd` that calls engine.Pause()
- **"paused"**: Resume → `tea.Cmd` that calls engine.Run()

All engine calls MUST be in `tea.Cmd` (never block in Update). This is a hard constraint from the TUI architecture.

### Engine Access

The engine is not in this worktree. Two approaches:

1. **Interface on App** — Define an `AutoController` interface in `internal/ui/model/` or `internal/app/`:
   ```go
   type AutoController interface {
       StartAuto(ctx context.Context, milestoneID string) error
       PauseAuto() error
       ResumeAuto(ctx context.Context) error
       AutoStatus() string // "idle", "running", "paused"
   }
   ```
   Add a field `autoController AutoController` to `App` (nil when engine not wired). The TUI nil-checks before calling.

2. **Stub methods on App** — Add `StartAuto`/`PauseAuto`/`ResumeAuto`/`AutoStatus` methods directly on `App` that return not-implemented errors until the engine merges.

Approach 1 (interface) is cleaner — it decouples the TUI from the engine package and allows testing with mocks. The interface should be minimal.

### State Derivation for Toggle

The toggle needs to know the current auto-mode state. Two sources:
- `m.autoSnapshot` — already on the UI struct from S01. Status field is "running", "paused", "completed", "error".
- If `autoSnapshot` is nil → idle state (auto-mode not started).

No additional state field needed. The existing `autoSnapshot` is sufficient.

### Milestone ID for Start

Starting auto-mode requires a milestone ID. Options:
- Prompt user via dialog (complex, deferred).
- Use a pre-configured milestone from config or command argument.
- For MVP: if the user hasn't specified a milestone, show an error notification. The actual milestone selection UX can be a follow-up.

The simplest MVP: `toggleAutoMode()` checks for a `m.autoMilestoneID` field (set via command or config). If empty on start, show a warning notification. This avoids blocking the keybinding implementation on milestone selection UX.

### Help Text

The keybinding should appear in the help bar. The existing `m.status.ToggleHelp()` and help rendering uses `key.Binding` help text. Adding `Auto.Toggle` to the help group display is needed.

**File:** Look for where help bindings are collected — likely in `keys.go` or a help-related method. The `key.WithHelp("ctrl+a", "auto")` on the binding handles the label.

### Sidebar Feedback

S01 already renders status with icons (▶ Running, ⏸ Paused, etc.) based on `autoSnapshot.Status`. When the toggle fires and the engine updates state, it publishes an `AutoEvent` with a new snapshot. The sidebar automatically re-renders. No additional sidebar work needed.

## Key Constraints

1. **Never block in Update()** — engine calls must be `tea.Cmd`.
2. **Engine doesn't exist yet** — define interface, test with mocks.
3. **`ctrl+a` is free** — confirmed by audit of keys.go.
4. **Global key** — toggle works regardless of `uiFocusEditor` vs `uiFocusMain`.
5. **Session required** — sidebar only renders when `m.session != nil`. Toggle should also require a session.
6. **Compact mode** — sidebar not visible, but keybinding should still work. Status is invisible but functional.

## Natural Task Boundaries

1. **T01: KeyMap + toggle handler** — Add `Auto.Toggle` to `KeyMap`, add `toggleAutoMode()` method on `UI` with state-based dispatch, handle in `handleGlobalKeys`. Define `AutoController` interface. Wire into `App` as nil field. Tests: key matching, state transitions (nil→start, running→pause, paused→resume).

2. **T02: Engine stub + integration test** — Implement a mock `AutoController` for testing. Write integration-style tests that simulate the full cycle: press ctrl+a → auto starts (snapshot updates to running) → press ctrl+a → pauses → press ctrl+a → resumes. Verify sidebar reflects each state.

3. **T03: Help text + notifications** — Add auto toggle to help display. Add notification feedback when toggle fires ("Auto-mode started", "Auto-mode paused", "Auto-mode resumed"). Add error notification when no milestone configured.

## Verification Strategy

- `go test ./internal/ui/model/... -run TestAutoToggle` — keybinding tests
- `go build ./...` — full compile clean
- `go vet ./...` — clean
- Manual: inspect that ctrl+a appears in help text

## Skill Discovery

No external skills needed. All work is Go + Bubble Tea keybindings, fully established in this codebase.
