---
depends_on: [M003]
---

# M004: TUI Integration + Git Worktrees — Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

## Project Description

User-facing polish: TUI sidebar panel showing auto-mode progress, TUI-initiated auto-mode (start/pause/resume from within the TUI), pub/sub event rendering, and optional git worktree isolation per milestone. After this milestone, auto-mode is fully integrated into the Crush experience.

## Why This Milestone

M002-M003 deliver a working, safe auto-mode via CLI. M004 brings it into the TUI — users see progress, control auto-mode, and optionally isolate work in git worktrees, all without leaving the Crush interface.

## User-Visible Outcome

### When this milestone is complete, the user can:

- See an auto-mode section in the TUI sidebar showing: milestone progress, active slice/task, cost, elapsed time
- Start, pause, and resume auto-mode from within the TUI via keybindings
- See real-time updates as auto-mode advances through slices and tasks
- Optionally enable git worktree isolation per milestone via `auto.worktree_mode` in crush.json

### Entry point / environment

- Entry point: Crush TUI with auto-mode active
- Environment: local dev terminal
- Live dependencies involved: configured LLM provider, git (for worktrees)

## Completion Class

- Contract complete means: TUI renders auto-mode state correctly, keybindings work, worktree create/merge works
- Integration complete means: starting auto-mode from TUI dispatches to the real auto loop, sidebar updates live
- Operational complete means: worktree isolation creates real git worktrees, squash merges back on completion

## Final Integrated Acceptance

To call this milestone complete, we must prove:

- Starting auto-mode from TUI keybinding begins the auto loop and the sidebar shows live progress
- Pausing from TUI keybinding stops dispatching after the current unit
- Sidebar shows milestone tree with completed/active/pending status for each slice
- Git worktree mode creates a worktree, auto-mode runs there, and squash merges back on milestone completion

## Implementation Decisions

- **Sidebar panel, not overlay** (D006). Auto-mode section lives within the existing 30-char wide sidebar rendered by `drawSidebar()`.
- **Imperative component pattern** — no sub-model `Update()`. Auto-mode sidebar data exposed as methods on `UI`, rendered in `drawSidebar()`. Follows the pattern documented in `internal/ui/AGENTS.md`.
- **Pub/sub via tea.Cmd** — TUI subscribes to `pubsub.Broker[AutoEvent]` using a `tea.Cmd` that waits on the subscription channel. Events arrive as `tea.Msg` in the main `Update()` loop. Never block in Update.
- **Conditional sidebar content** — when auto-mode is active, the sidebar shows auto-mode progress section (milestone tree, current unit, cost, time) alongside or replacing some of the standard sections (files, LSPs, MCPs). When inactive, sidebar looks normal.
- **Keybindings** — registered in `model/keys.go`. Likely: a key to start auto, a key to pause/resume, a key to show auto status detail.
- **Git worktrees via shell** — new `internal/auto/worktree.go` that shells out to `git worktree add` / `git worktree remove`. No Go git library — plain shell commands. Worktree at `.crush/worktrees/<MID>/`.
- **Worktree lifecycle:** On milestone start → `git worktree add .crush/worktrees/M001 -b auto/M001`. On milestone complete → squash merge to integration branch, remove worktree.
- **Worktree is opt-in** — default is branch-based isolation (or no isolation). Configured via `auto.worktree_mode` in crush.json.

## Risks and Unknowns

- **Sidebar space** — 30 chars is tight. Auto-mode progress, milestone tree, cost, and time need to fit. May need to truncate aggressively or use abbreviated formats.
- **TUI codebase may change** — `ui.go` is 3667 lines and actively developed. Merge conflicts are likely if M004 ships much later than M003.
- **Worktree + SQLite** — the DB is in `.crush/crush.db`. Worktrees need to either share the DB or have their own. Sharing means concurrent SQLite access. Own DB means state divergence.

## Existing Codebase / Prior Art

- `internal/ui/model/ui.go` — 3667 lines. `UI` struct is sole Bubble Tea model. `drawSidebar()` renders the sidebar.
- `internal/ui/model/sidebar.go` — sidebar rendering: logo, session title, model info, files, LSPs, MCPs. 184 lines.
- `internal/ui/model/keys.go` — `KeyMap` struct with keybinding groups. 271 lines.
- `internal/ui/AGENTS.md` — TUI architectural guidelines. Must read before implementing.
- `internal/pubsub/` — `Broker[T]`, `Event[T]`. Subscribe returns `<-chan Event[T]`.

## Relevant Requirements

- R015 — TUI sidebar panel (primary)
- R016 — Auto-mode startable from TUI (primary)
- R017 — Pub/sub events — consuming side (primary)
- R018 — Git worktree isolation (primary)

## Scope

### In Scope

- Auto-mode section in sidebar: milestone tree, active unit, cost, time
- Keybindings for start/pause/resume auto-mode from TUI
- Pub/sub event consumption: subscribe to AutoEvent broker, render state changes
- Git worktree create/remove/merge lifecycle
- Config: `auto.worktree_mode` parsing (config struct already added in M003)

### Out of Scope

- Full overlay dashboard (using sidebar only)
- User-customizable prompts (deferred, R021)
- Token-based budget (deferred, R022)
