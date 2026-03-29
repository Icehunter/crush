# M004: TUI Integration + Git Worktrees — Research

**Researched:** 2026-03-27
**Status:** Complete

## 1. Codebase Findings

### 1.1 Sidebar Architecture (R015)

**`internal/ui/model/sidebar.go`** (184 lines) — `drawSidebar()` is a method on `UI`. It renders: logo, session title, working directory, model info (provider, model name, reasoning, context/cost), then dynamic-height sections for files, LSPs, MCPs. Uses `uv.NewStyledString(...).Draw(scr, area)` — the Ultraviolet screen buffer pattern.

Key constraints:
- **Fixed 30-char width** (`sidebarWidth := 30` in `generateLayout()`). Hardcoded, not configurable. Auto-mode content must fit within this.
- **Dynamic height allocation** via `getDynamicHeightLimits()` — distributes remaining vertical space across files/LSPs/MCPs with priority ordering. Auto-mode section needs a similar budget scheme.
- **Compact mode hides sidebar entirely** — when `isCompact` is true (terminal height < 30 or user toggle), no sidebar is rendered. Auto-mode sidebar is unavailable in compact mode.
- Sidebar only renders when `m.session != nil`.

The natural insertion point for auto-mode content is between model info and files section, or replacing some of the files/LSPs/MCPs sections when auto-mode is active. The `getDynamicHeightLimits()` pattern should be extended to account for auto-mode section height.

### 1.2 TUI Event Subscription Pattern (R015, R016, R017)

**`internal/app/app.go`** lines 482-498 — `setupEvents()` uses a generic `setupSubscriber[T]()` function that:
1. Takes a `func(context.Context) <-chan pubsub.Event[T]` subscriber
2. Reads from the subscription channel in a goroutine
3. Forwards events as `tea.Msg` to a shared `events chan tea.Msg` (buffered, capacity 100)
4. `app.Subscribe(program)` pumps this channel into the Bubble Tea program via `program.Send(msg)`

This is the established pattern for all service events (sessions, messages, permissions, history, notifications, MCP, LSP). **Auto-mode events must follow this exact pattern**: add a `pubsub.Broker[AutoEvent]` to `App`, subscribe in `setupEvents()`, handle `pubsub.Event[auto.AutoEvent]` in `UI.Update()`.

The TUI's `Update()` (ui.go ~line 490+) has an extensive `switch msg.(type)` block handling every `pubsub.Event[T]` variant. Adding `pubsub.Event[auto.AutoEvent]` is a straightforward addition.

### 1.3 Keybinding System (R016)

**`internal/ui/model/keys.go`** (271 lines) — `KeyMap` struct with nested groups: `Editor`, `Chat`, `Initialize`, plus global bindings. Constructed by `DefaultKeyMap()`. Key bindings use `charm.land/bubbles/v2/key`.

Available ctrl keys already in use: ctrl+c (quit), ctrl+g (help), ctrl+p (commands), ctrl+m/ctrl+l (models), ctrl+z (suspend), ctrl+s (sessions), ctrl+n (new session), ctrl+f (add image/attachment), ctrl+v (paste), ctrl+r (delete attachment), ctrl+o (open editor), ctrl+d (details), ctrl+t (toggle tasks), ctrl+j (newline). Tab, shift+tab also used.

**Available keys for auto-mode**: ctrl+a (likely candidate — not taken), ctrl+e (not taken), ctrl+q (not taken). The keybinding should likely be context-sensitive — only active in `uiFocusMain` state in `uiChat` mode.

Focus routing (ui.go line ~1651): key events route to either editor or chat based on `m.focus`. Auto-mode keybindings should be in a new `Auto` group within `KeyMap`.

### 1.4 Auto-Mode Engine Interface (R016, R017)

**From M003 worktree** (`internal/auto/engine.go`):
- `Engine` struct with `Run(ctx, milestoneID)`, `Step(ctx, milestoneID)`, `Pause()`, `Stop()`, `Status() EngineStatus`
- Engine holds a `*pubsub.Broker[AutoEvent]` for event publishing
- `EngineState`: `EngineIdle`, `EngineRunning`, `EnginePaused`
- `EngineStatus`: State, MilestoneID, ActiveUnit, LastError

**From M003** (`internal/auto/events.go`):
- 12 event types: `EventUnitStarted`, `EventUnitCompleted`, `EventUnitFailed`, `EventLoopPaused`, `EventLoopStopped`, `EventStateTransition`, `EventVerificationStarted/Passed/Failed`, `EventBudgetExceeded`, `EventStuckDetected`, `EventContextPressure`
- `AutoEvent` payload: Unit, Error, Timestamp, Message

The Engine is currently constructed in isolation (no wiring in `app.App`). M004 must add Engine wiring to `App` so the TUI can access it. This is the core integration challenge.

### 1.5 Config (R014, R018)

**From M003** (`internal/config/config.go` line ~402):
```go
type AutoConfig struct {
    VerificationCommands []string `json:"verification_commands,omitempty"`
    BudgetCeiling        float64  `json:"budget_ceiling,omitempty"`
    StuckThreshold       int      `json:"stuck_threshold,omitempty"`
    WorktreeMode         string   `json:"worktree_mode,omitempty"`
}
```
Config parsing for `worktree_mode` already exists and is tested (3 tests). The string value needs semantic interpretation (e.g., "per-milestone", "per-slice", or empty/disabled).

### 1.6 State Querier / DB Layer

`StateQuerier` interface in `state.go` provides: `ListMilestones`, `ListSlicesByMilestone`, `ListTasksBySlice`. The sidebar needs to call these (or cache their results from events) to render the milestone tree.

Two approaches:
1. **Query on each render** — simplest, but `drawSidebar()` is called every frame. SQLite queries per frame is expensive.
2. **Cache state from events** — maintain an in-memory representation of the milestone tree updated by `AutoEvent` messages. This is the correct approach: the event stream provides enough data to reconstruct the tree without DB queries in the render loop.

### 1.7 UI Struct — No Sub-Model Pattern

Per `internal/ui/AGENTS.md`: "Sub-components expose methods, not Update()." The auto-mode sidebar is NOT a separate Bubble Tea model. It should be:
- Fields on `UI` struct holding auto-mode state (milestone tree, active unit, cost, time, engine state)
- A `drawAutoMode()` method on `UI` called from within `drawSidebar()`
- State updated in `Update()` when `pubsub.Event[auto.AutoEvent]` arrives

### 1.8 Git Worktree (R018)

No existing `exec.Command("git", ...)` usage anywhere in `internal/`. The worktree module will be the first shell-out to git in the codebase. This is pure new code.

Key design points:
- Path: `.crush/worktrees/<MID>/` (e.g., `.crush/worktrees/M001/`)
- Create: `git worktree add .crush/worktrees/M001 -b auto/M001`
- Remove: `git worktree remove .crush/worktrees/M001` (after merge)
- Merge: `git merge --squash auto/M001` on the integration branch
- **SQLite sharing concern**: `.crush/crush.db` lives in the main worktree. If the engine runs in a worktree, it still needs the shared DB. Git worktrees share `.git/` so the worktree's `.crush/` would be independent. The DB path must be resolved to the main worktree, not the current working directory.

## 2. Technology Assessment

### 2.1 Key Technologies

All technologies are already in use in the codebase:
- **Bubble Tea v2** — TUI framework (already used extensively)
- **Ultraviolet** — screen buffer rendering (already used in sidebar)
- **pubsub.Broker[T]** — event system (already used for 7+ event types)
- **Git** — worktree commands (new shell-out, but git is a standard dependency)

No external library lookups needed. No skill installations recommended — this is Go TUI work using established project patterns.

## 3. Risk Analysis

### 3.1 High Risk: Sidebar Space Budget (R015)

30 characters is extremely tight for a milestone tree. Consider a milestone with 3 slices, each with 3 tasks — that's potentially 13 lines of tree content plus cost/time stats. Aggressive truncation and abbreviation required.

**Mitigation**: Design the sidebar section with a fixed max-height budget. Show only the active milestone's active slice with task status indicators. Collapsed view by default, expandable on demand.

### 3.2 Medium Risk: Engine Wiring to App (R016, R017)

The Engine currently lives in `internal/auto/` with no connection to `app.App`. Wiring requires:
1. Adding Engine (or an interface) to `App` struct
2. Adding `pubsub.Broker[AutoEvent]` to `App` and subscribing in `setupEvents()`
3. Exposing engine control methods (start/pause/stop) to the TUI
4. The Engine needs heavy dependencies (StateQuerier, SessionCreator, Dispatcher, StatusAdvancer, etc.) — constructing it requires DB access, coordinator, etc.

This is the highest-integration-risk slice. It touches `app.App` (core wiring), `ui.go` (massive file), and `engine.go`.

### 3.3 Medium Risk: SQLite + Worktrees (R018)

If auto-mode runs in a worktree, the working directory changes but the DB path must remain stable. The DB is at `.crush/crush.db` relative to the project root. In a worktree at `.crush/worktrees/M001/`, the DB path resolution must point back to the main tree's `.crush/crush.db`.

**Mitigation**: Resolve DB path to an absolute path at startup (before any worktree cd). Pass absolute DB path to the engine.

### 3.4 Low Risk: Merge Conflicts on ui.go

`ui.go` is 3667 lines and the main edit target for many features. M004 touches it for event handling and sidebar rendering. If other features land first, merge conflicts are likely but manageable since additions are in distinct switch-case branches.

## 4. Natural Slice Boundaries

### Slice 1: Pub/Sub Event Wiring + Auto-Mode State in UI (foundation)
- Add `pubsub.Broker[AutoEvent]` to `App`
- Subscribe in `setupEvents()`
- Add auto-mode state fields to `UI` struct
- Handle `pubsub.Event[auto.AutoEvent]` in `Update()` to maintain in-memory state
- **Risk**: Medium (touches app.go and ui.go core wiring)
- **Proves**: Events flow from engine to TUI

### Slice 2: Sidebar Rendering (visible output)
- Depends on S01
- Add `drawAutoMode()` method to render milestone tree, active unit, cost, time
- Integrate into `drawSidebar()` with height budget
- **Risk**: Medium (space constraints at 30 chars)
- **Proves**: User sees auto-mode progress in sidebar

### Slice 3: TUI Keybindings for Start/Pause/Resume (control)
- Depends on S01
- Add `Auto` group to `KeyMap`
- Handle key events in `Update()` to call engine Start/Pause/Stop
- Engine construction/access through `App`
- **Risk**: Medium (engine construction requires dependency wiring)
- **Proves**: User controls auto-mode from TUI

### Slice 4: Git Worktree Isolation (independent)
- No dependency on S01-S03 (TUI slices)
- New `internal/auto/worktree.go` with Create/Remove/Merge
- Config interpretation for `worktree_mode`
- Absolute DB path resolution
- **Risk**: Medium (git shell-outs, merge semantics, DB path resolution)
- **Proves**: Auto-mode runs in isolated worktree

## 5. Strategic Recommendations

### What Should Be Proven First

**S01 (event wiring)** must come first — it's the foundation for both sidebar rendering and TUI control. Without events flowing, nothing else works.

**S04 (worktree)** is independent and can be developed in parallel with S02/S03.

### Existing Patterns to Reuse

1. **`setupSubscriber[T]()`** — exact same generic subscriber pattern for auto events
2. **`drawSidebar()` + `getDynamicHeightLimits()`** — extend, don't replace
3. **`KeyMap` groups** — add `Auto` group alongside `Editor`, `Chat`, `Initialize`
4. **`pubsub.Event[T]` switch cases in `Update()`** — add one more case
5. **`EngineStatus`** — already designed for status reporting, use it directly

### Boundary Contracts

- **Engine → TUI**: `pubsub.Broker[AutoEvent]` (events) + `Engine.Status()` (polling)
- **TUI → Engine**: `Engine.Run()` / `Engine.Pause()` / `Engine.Stop()` (control)
- **App → Engine**: `App` holds or constructs Engine, exposes it to TUI via `common.Common.App`
- **Worktree → Engine**: Engine receives working directory; worktree module manages create/remove/merge lifecycle

### Codebase Constraints

1. Never block in `Update()` — engine control must be via `tea.Cmd`
2. Sidebar renders every frame — no DB queries in render path
3. Imperative component pattern — no sub-model `Update()` for auto-mode
4. 30-char sidebar width is hardcoded — design for this constraint
5. Compact mode has no sidebar — auto-mode visibility is lost in compact mode (acceptable for MVP)

## 6. Requirement Analysis

### Table Stakes (must have)
- **R015** (sidebar panel) — core visibility feature, the primary deliverable
- **R016** (TUI start/pause/resume) — core control feature
- **R017** (pub/sub consumption) — infrastructure for R015 and R016
- **R018** (worktree isolation) — explicitly scoped, opt-in feature

### Observations
- **R015 lacks detail on compact mode behavior** — sidebar doesn't exist in compact mode. The requirement should note that auto-mode visibility is only available in non-compact layout. Consider a minimal compact-mode indicator (e.g., in the header) as a candidate enhancement.
- **R017 is partially validated** — M002 already publishes events; M004 is the consuming side. The requirement status should track this dual nature.
- **R018 "per-slice" vs "per-milestone"** — the test fixture uses `"per-slice"` as worktree_mode but the context doc says "per milestone." Clarify whether worktree granularity is per-milestone only (simpler) or also per-slice (complex — many worktrees).

### Candidate Requirements (advisory, not auto-binding)
- **Auto-mode status in compact mode header** — users on small terminals lose all auto-mode visibility. A one-line indicator in the compact header would help.
- **Engine construction lifecycle** — R016 says "start from TUI" but doesn't specify how the Engine is constructed (lazy on first start? eager on app boot?). This is an implementation detail but worth deciding early.
- **Worktree cleanup on crash** — if auto-mode crashes with an active worktree, the worktree persists. R009 (crash recovery) should extend to worktree cleanup.

## 7. Skill Discovery

No external skills needed. All work is Go + Bubble Tea + Git CLI, all already deeply established in this codebase. The `available_skills` list has no Go/TUI-specific skills that would be relevant.
