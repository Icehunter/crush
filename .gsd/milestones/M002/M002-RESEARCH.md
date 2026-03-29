# M002: Auto Loop + Session Management — Research

**Date:** 2026-03-27
**Status:** Complete

## Summary

M002 builds the core autonomous execution engine on top of M001's data model and state machine. The codebase is well-structured with clear patterns to follow: `Coordinator.RunWithForcedTier()` dispatches prompts with model tier selection, `session.Service.CreateTaskSession()` creates child sessions, and `app.RunNonInteractive()` demonstrates the headless execution pattern with auto-approved permissions. The auto-mode loop itself is a straightforward derive→dispatch→execute→advance cycle where each iteration calls `DeriveState()` to determine the next action, builds a unit-type-specific prompt, dispatches it via the Coordinator, then updates DB state.

The primary technical risk is **injecting custom system prompts into the Coordinator dispatch path**. Currently, `NewCoordinator()` builds a single `SessionAgent` with the coder system prompt hardcoded via `coderPrompt()`. Auto-mode needs per-unit-type prompts (research, plan-slice, execute-task, complete-slice, validate-milestone). The cleanest approach is to bypass `NewCoordinator()` entirely for auto-mode and construct `SessionAgent` instances directly using `agent.NewSessionAgent()` + `SetSystemPrompt()` — this is already supported by the `SessionAgent` interface and avoids modifying the existing coordinator contract.

The secondary risk is **M001 code not yet merged to the milestone/M002 branch**. The M001 worktree contains `state.go`, `dispatch.go`, and their tests, but these files don't exist in the M002 worktree. The first slice must consolidate M001's work into the M002 branch before building on it.

## Recommendation

**Build the auto-mode engine as `internal/auto/engine.go`** — a new `Engine` struct that owns the loop lifecycle, prompt template rendering, session management, and pub/sub event publishing. The engine should directly construct `agent.SessionAgent` instances (not go through `Coordinator`) to control system prompts per unit type. This avoids modifying the existing `Coordinator` interface while reusing all the underlying infrastructure (model building, tool construction, provider setup).

**Slice ordering should be: (1) engine core + loop lifecycle, (2) prompt templates + init flow, (3) CLI commands + lock file, (4) `crush next` standalone runner.** The engine is the foundation everything depends on. Prompt templates and init are the next dependency (the engine needs prompts to dispatch). CLI commands wire the engine to user entry points. `crush next` is a thin wrapper over a single engine iteration.

## Implementation Landscape

### Key Files

- `internal/auto/state.go` (M001) — `DeriveState()` returns `*State` with `Action`, `Milestone`, `Slice`, `Task`. The engine calls this each iteration.
- `internal/auto/dispatch.go` (M001) — `Dispatch(*State) Action` evaluates rules. Used by engine to determine what to do next.
- `internal/auto/status.go` — `Status` and `Phase` enums. Engine updates these after each unit completes.
- `internal/auto/milestone.go`, `slice.go`, `task.go` — Domain models with `ToDBCreate()` and `FromDB()` converters.
- `internal/agent/coordinator.go` — `buildAgent()` constructs `SessionAgent` with system prompt and tools. `buildAgentModels()` resolves model tiers. `runSubAgent()` demonstrates the child-session pattern. **Auto-mode needs access to `buildAgentModels()` and `buildTools()` without going through the full `NewCoordinator()` flow.**
- `internal/agent/agent.go` — `SessionAgent` interface with `Run()`, `SetSystemPrompt()`, `SetTools()`, `SetModels()`. `NewSessionAgent()` constructs instances directly. `SessionAgentCall` has `NonInteractive: bool` field for auto-approve behavior.
- `internal/agent/prompts.go` — `coderPrompt()`, `taskPrompt()`, `InitializePrompt()` — pattern for Go template prompts with `//go:embed` and `prompt.NewPrompt()`.
- `internal/agent/prompt/prompt.go` — `Prompt.Build(ctx, provider, model, cfg)` renders Go templates with `PromptDat` (provider, model, config, working dir, git status, context files, skills).
- `internal/app/app.go` — `RunNonInteractive()` demonstrates: create session → `AutoApproveSession()` → `AgentCoordinator.Run()` → stream message events. `App` struct exposes `Sessions`, `Messages`, `Permissions`, `AgentCoordinator`, `config`.
- `internal/cmd/run.go` — Cobra command pattern: `setupApp(cmd)` → `app.RunNonInteractive()`. Model for `crush auto` and `crush next`.
- `internal/cmd/root.go` — `rootCmd.AddCommand()` registers subcommands. `setupApp()` handles config, DB, app initialization.
- `internal/session/session.go` — `CreateTaskSession(toolCallID, parentSessionID, title)` creates child sessions. `Session.Cost` for budget tracking. `Session.ParentSessionID` for hierarchy.
- `internal/pubsub/broker.go` — `Broker[T]` with `Publish(EventType, T)` and `Subscribe(ctx) <-chan Event[T]`. `NewBroker[T]()` for typed event streams.
- `internal/permission/permission.go` — `AutoApproveSession(sessionID)` auto-approves all permission requests for a session. Used by `RunNonInteractive()`.
- `internal/db/sql/milestones.sql`, `slices.sql`, `tasks.sql` — SQLC queries: CRUD + status/phase updates. No session-cost aggregation queries exist yet.
- `internal/db/migrations/20260327000000_add_auto_tables.sql` — M001's migration for milestones/slices/tasks tables.

### Build Order

1. **Consolidate M001 code** — Copy `state.go`, `dispatch.go`, and test files from M001 worktree into M002 branch. Verify tests pass. This unblocks everything.
2. **Engine core** (`internal/auto/engine.go`) — The `Engine` struct with `Run()` (continuous loop), `Step()` (single iteration for `crush next`), `Pause()`, `Stop()`. Depends on `DeriveState()` and needs access to `SessionAgent` construction. Pub/sub event publishing for state transitions.
3. **Prompt templates** (`internal/auto/templates/*.md.tpl`) — One template per unit type. `//go:embed` pattern matching `internal/agent/templates/`. Templates need `prompt.NewPrompt()` and `Build()`.
4. **Init flow** — `crush auto init` opens an interactive planning session. Uses the existing `InitializePrompt()` pattern but with a planning-specific prompt that produces milestone/slice/task structure.
5. **CLI commands** (`internal/cmd/auto.go`) — `crush auto [init|start|stop|pause|status]` and `crush next`. Cobra subcommand group. Lock file management in `.crush/auto.lock`.
6. **Crash recovery** — On `crush auto start`, check DB state for active units. If found, synthesize a recovery briefing and resume from there.

### Verification Approach

- **Unit tests**: Engine lifecycle (start/step/pause/stop), prompt template rendering, lock file management, crash recovery.
- **Integration tests**: Seed a milestone with slices/tasks → run `Engine.Step()` → verify DB state advances correctly. Use in-memory SQLite with `file:<test>?mode=memory&cache=shared` pattern from K004.
- **CLI tests**: Cobra command parsing, flag validation, mutual exclusivity checks.
- **`go vet ./...`** and **`task lint:fix`** after each slice.

## Constraints

- **CGO_ENABLED=0** — no CGo, pure Go SQLite driver (`modernc.org/sqlite`).
- **`GOEXPERIMENT=greenteagc`** — builds with experimental GC.
- **Coordinator is tightly coupled to coder prompt** — `NewCoordinator()` hardcodes `coderPrompt()` and builds a single `SessionAgent`. Auto-mode cannot easily reuse `Coordinator` for different system prompts per unit type. Must construct `SessionAgent` instances directly or add a new method.
- **`buildAgentModels()` and `buildTools()` are private** — These are methods on `*coordinator`, not exported. Auto-mode engine needs model/tool construction. Options: (a) extract into exported functions, (b) accept `App` dependencies and call through existing interfaces, (c) add a new method to the `Coordinator` interface.
- **M001 code not merged** — `state.go`, `dispatch.go`, and tests exist only in the M001 worktree. Must be consolidated before M002 work begins.

## Common Pitfalls

- **Stale context between units** — Each unit MUST get a fresh `SessionAgent` or at minimum a new session ID. Reusing the same session across units would pollute context and degrade LLM output quality.
- **Lock file race conditions** — PID-based stale detection (checking if PID is alive) can have TOCTOU races. Use `flock(2)` or similar atomic locking if possible, but since CGO is disabled, fall back to PID + timestamp heuristics with a generous timeout.
- **Test DB isolation** — Use `file:<test-name>?mode=memory&cache=shared` per K004. Never share DB connections across parallel tests.
- **Coordinator model refresh** — `coordinator.UpdateModels()` rebuilds tools and models before each run. Auto-mode engine needs similar refresh logic or should construct fresh agents per unit.
- **M001 worktree drift** — If M001 branch diverges from M002's base, consolidation may require conflict resolution. Check for new commits on M001 before starting.

## Open Risks

- **System prompt injection path** — The recommended approach (direct `SessionAgent` construction) requires extracting `buildAgentModels()` and `buildTools()` from the coordinator, or duplicating some initialization logic. This is the highest-risk code change since it touches the agent infrastructure boundary.
- **Planning prompt quality** — `crush auto init` depends on the LLM producing well-structured milestone/slice/task output from a planning prompt. If the output format is unreliable, parsing will fail. May need structured output or retry logic.
- **Session cost aggregation** — No existing SQLC query sums child session costs by parent ID. Need a new query or in-memory aggregation for `crush auto status`.
- **M001 completion timing** — If M001 is not fully merged/validated, M002 cannot start. The context states M001 is "complete and validated" but the code is in a separate worktree branch, not merged to main.

## Sources

- Codebase exploration of `internal/agent/`, `internal/auto/`, `internal/app/`, `internal/cmd/`, `internal/session/`, `internal/pubsub/`, `internal/permission/`, `internal/config/`, `internal/db/`
- M001 worktree at `/Users/icehunter/.gsd/projects/c746f2765afc/worktrees/M001` for `state.go`, `dispatch.go`, and integration tests
