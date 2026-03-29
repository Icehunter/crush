# M005: Production Integration

**Gathered:** 2026-03-28
**Status:** Ready for planning

## Project Description

Wire all isolated auto-mode components from M001–M004 into Crush's production infrastructure so auto-mode actually works end-to-end. The engine, state machine, safety rails, TUI integration, and worktree manager were all built and tested against mock interfaces — this milestone connects them to real SQLite, real session.Service, real Coordinator, and real crush.json config.

## Why This Milestone

M001–M004 built auto-mode as a collection of well-tested isolated components. But none of them are connected to Crush's real runtime. The engine calls `SessionCreator` but nobody implements it against `session.Service`. The TUI calls `AutoController` but nobody implements it against the engine. The CLI stubs exist but don't connect to anything. Without this milestone, auto-mode is a comprehensive test suite with no production path.

## User-Visible Outcome

### When this milestone is complete, the user can:

- Run `crush auto start <milestone-id>` and watch the engine derive state from SQLite, dispatch units to the LLM, verify results, and advance automatically
- Press ctrl+a in the TUI and see the real engine start/pause/resume with live sidebar updates
- Run `crush next` to manually step through one unit at a time
- See real dollar costs tracked per session, with auto-pause when budget ceiling is reached
- Have auto-mode create a git worktree at `.crush/worktrees/<MID>/` when `worktree_mode: "per-milestone"` is configured

### Entry point / environment

- Entry point: `crush auto start`, `crush next`, ctrl+a in TUI
- Environment: local dev terminal
- Live dependencies involved: SQLite database, LLM provider (via fantasy), git (for worktrees)

## Completion Class

- Contract complete means: production adapters compile, unit tests pass, integration tests prove real DB round-trips
- Integration complete means: engine instantiated with all production deps, CLI commands bootstrap full dependency graph, TUI AutoController backed by real engine
- Operational complete means: `crush auto start` on a milestone with tasks actually executes units through the LLM

## Final Integrated Acceptance

To call this milestone complete, we must prove:

- `crush auto start` on a seeded milestone executes at least one unit through the real engine loop (derive → dispatch → advance)
- The TUI sidebar shows real progress when auto-mode is running
- `crush auto status` reports the engine's actual state
- Safety rails (budget, stuck, verification) are wired with real config values

## Risks and Unknowns

- Branch reconciliation complexity — M001–M003 branches were never merged into claudecode. M003 includes M001+M002 code. Event types diverge between M003 (pubsub.EventType constants) and M004 (AutoEventType string type). Conflicts expected in internal/auto/ and internal/config/.
- SumChildSessionCosts query doesn't exist — needs new sqlc query + code regeneration
- Dispatcher interface bridge — engine's `Dispatcher.RunWithForcedTier()` has a different signature than `coordinator.Run()`. Adapter must handle the model tier mapping and result translation.
- Prompt template data alignment — templates reference context types that must align between engine Unit struct and template data structs

## Existing Codebase / Prior Art

- `internal/auto/` — domain models (milestone.go, slice.go, task.go, status.go), event types (event.go), worktree manager (worktree.go). Engine, state machine, safety rails exist on milestone/M003 branch only.
- `internal/agent/coordinator.go` — `Run(ctx, sessionID, prompt, attachments)` is the real dispatch entry point
- `internal/app/app.go` — `RunNonInteractive()` for headless execution, `setupEvents()` for broker wiring
- `internal/session/session.go` — `Service` interface with `Create()`, `CreateTaskSession()`, cost tracking
- `internal/db/sql/` — sqlc queries for milestones, slices, tasks CRUD. Sessions table has parent_session_id.
- `internal/config/config.go` — `Options.WorktreeMode` field already exists
- `internal/app/auto_events.go` — autoBroker, SubscribeAutoEvents, PublishAutoEvent already wired
- `internal/ui/model/auto_controller.go` — AutoController interface (StartAuto/PauseAuto/ResumeAuto/AutoStatus)
- `internal/ui/model/auto_toggle.go` — toggleAutoMode() state dispatch
- `internal/cmd/root.go` — cobra command tree, no auto subcommands yet

> See `.gsd/DECISIONS.md` for all architectural and pattern decisions — it is an append-only register; read it during planning, append to it during execution.

## Relevant Requirements

- R004 — Engine loop with real DB wiring (primary)
- R005 — Child sessions per unit via session.Service
- R006 — Prompt templates wired into dispatcher
- R007 — crush auto CLI commands
- R008 — crush next command
- R009 — Lock file + crash recovery
- R015 — Sidebar with real engine events
- R016 — ctrl+a triggers real engine
- R017 — Real event publishing
- R018 — Worktree lifecycle in engine
- R025 — Branch reconciliation
- R026 — Production DB adapters
- R027 — End-to-end proof

## Scope

### In Scope

- Merge M001–M003 branch code onto claudecode with conflict resolution
- Reconcile event type divergence (M003 engine events vs M004 TUI events)
- Production implementations of StateQuerier, StatusAdvancer, SessionCreator, Dispatcher, BudgetChecker, TokenQuerier
- New sqlc queries (SumChildSessionCosts, session token usage)
- cobra CLI commands: `crush auto start/pause/stop/status`, `crush next`
- Real AutoController implementation backed by engine
- WorktreeManager integration at engine lifecycle boundaries
- Integration tests proving real DB round-trips

### Out of Scope / Non-Goals

- Custom prompt templates per project (R021 — deferred)
- Token-based budget ceiling (R022 — deferred)
- Multi-milestone orchestration (auto-mode runs one milestone at a time)
- LLM-driven plan decomposition from natural language (R019 — the planning tools exist but testing them against real LLMs is not in scope for this milestone)

## Technical Constraints

- CGO_ENABLED=0 — no CGO in SQLite driver
- Must use sqlc for any new queries — no raw SQL outside internal/db/sql/
- Must follow existing Crush patterns: cobra for CLI, pubsub.Broker for events, config.Options for configuration
- All new code must pass go vet and existing linting

## Integration Points

- SQLite (via sqlc) — auto tables, session tables for cost/token queries
- session.Service — child session creation, cost tracking
- agent.Coordinator — LLM dispatch
- pubsub.Broker — event publishing for TUI
- config.Options — auto config (verification_commands, budget_ceiling, stuck_threshold, worktree_mode)
- git — worktree create/merge/remove

## Open Questions

- Whether to unify M003's event constants (pubsub.EventType) with M004's event types (AutoEventType) or keep both with a translation layer — leaning toward unifying on M003's pattern since the engine owns event semantics
- Whether the Dispatcher adapter should call coordinator.Run() directly or go through App.RunNonInteractive() — leaning toward coordinator.Run() since we need session-level control
