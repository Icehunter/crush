---
estimated_steps: 46
estimated_files: 4
skills_used: []
---

# T01: Add ctrl+a keybinding, AutoController interface, toggle handler, help text, and notifications

Add the core auto-mode toggle implementation: keybinding registration, controller interface, state-based dispatch, help text, and notification feedback.

## Steps

1. **Add `Auto` group to `KeyMap`** in `internal/ui/model/keys.go`:
   - Add `Auto struct { Toggle key.Binding }` to the `KeyMap` struct (after the `Initialize` group)
   - In `DefaultKeyMap()`, set `km.Auto.Toggle = key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "auto"))`

2. **Define `AutoController` interface** in a new file `internal/ui/model/auto_controller.go`:
   ```go
   type AutoController interface {
       StartAuto(ctx context.Context, milestoneID string) error
       PauseAuto() error
       ResumeAuto(ctx context.Context) error
       AutoStatus() string // "idle", "running", "paused"
   }
   ```
   Add `autoController AutoController` field to the `UI` struct in `ui.go`. Add `autoMilestoneID string` field for MVP milestone targeting.

3. **Implement `toggleAutoMode()` method** on `*UI` in a new file `internal/ui/model/auto_toggle.go`:
   - Return `tea.Cmd`
   - Guard: if `m.autoController == nil`, return `util.ReportWarn("Auto-mode not available")`
   - Guard: if `m.session == nil`, return `util.ReportWarn("No active session")`
   - Derive state from `m.autoSnapshot`: nil or empty Status → "idle"; else use Status value
   - idle: if `m.autoMilestoneID == ""`, return `util.ReportWarn("No milestone configured for auto-mode")`. Else return a `tea.Cmd` that calls `m.autoController.StartAuto(ctx, m.autoMilestoneID)` and returns `util.ReportInfo("Auto-mode started")` on success or `util.ReportError(err)` on failure
   - "running": return a `tea.Cmd` that calls `m.autoController.PauseAuto()` and returns `util.ReportInfo("Auto-mode paused")` on success
   - "paused": return a `tea.Cmd` that calls `m.autoController.ResumeAuto(ctx)` and returns `util.ReportInfo("Auto-mode resumed")` on success
   - All engine calls MUST be inside the `tea.Cmd` closure, never in `Update()` directly

4. **Wire toggle into `handleGlobalKeys`** in `internal/ui/model/ui.go`:
   - Add a case in `handleGlobalKeys` (before the `Suspend` case): `case key.Matches(msg, m.keyMap.Auto.Toggle): cmds = append(cmds, m.toggleAutoMode()); return true`

5. **Add to help text** in `internal/ui/model/ui.go`:
   - In `ShortHelp()`, in the `uiChat` case, append `k.Auto.Toggle` to `binds` after the existing model/command bindings
   - In `FullHelp()`, in the `uiChat` case, append `k.Auto.Toggle` to `mainBinds`

6. **Format** with `goimports` or `gofmt`

## Must-Haves

- [ ] `Auto.Toggle` binding registered as `ctrl+a` with help text "auto"
- [ ] `AutoController` interface with 4 methods in its own file
- [ ] `autoController` and `autoMilestoneID` fields on `UI` struct
- [ ] `toggleAutoMode()` dispatches StartAuto/PauseAuto/ResumeAuto based on snapshot status
- [ ] All engine calls inside `tea.Cmd` closures
- [ ] Nil-safe: no panic when `autoController` is nil or `autoSnapshot` is nil
- [ ] Toggle in `handleGlobalKeys` — works from both editor and chat focus
- [ ] Help text shows ctrl+a in both ShortHelp and FullHelp
- [ ] Notification feedback via `util.ReportInfo`/`util.ReportWarn`

## Verification

- `go build ./...` — compiles clean
- `go vet ./...` — clean
- `grep -q 'ctrl+a' internal/ui/model/keys.go` — binding registered
- `grep -q 'AutoController' internal/ui/model/auto_controller.go` — interface defined
- `grep -q 'toggleAutoMode' internal/ui/model/auto_toggle.go` — handler defined

## Inputs

- ``internal/ui/model/keys.go` — existing KeyMap struct to extend with Auto group`
- ``internal/ui/model/ui.go` — UI struct to add fields and handleGlobalKeys case`
- ``internal/auto/event.go` — AutoSnapshot type with Status field for state derivation`
- ``internal/ui/util/util.go` — ReportInfo/ReportWarn/ReportError for notification feedback`

## Expected Output

- ``internal/ui/model/keys.go` — Auto.Toggle binding added to KeyMap and DefaultKeyMap()`
- ``internal/ui/model/auto_controller.go` — new file with AutoController interface`
- ``internal/ui/model/auto_toggle.go` — new file with toggleAutoMode() method`
- ``internal/ui/model/ui.go` — autoController/autoMilestoneID fields, handleGlobalKeys case, ShortHelp/FullHelp additions`

## Verification

cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M004 && go build ./... && go vet ./... && grep -q 'ctrl+a' internal/ui/model/keys.go && grep -q 'AutoController' internal/ui/model/auto_controller.go && grep -q 'toggleAutoMode' internal/ui/model/auto_toggle.go
