---
depends_on: [M001]
---

# M002: Auto Loop + Session Management — Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

## Project Description

Core autonomous execution engine for Crush. Builds the main auto loop, unit-type-specific prompt templates, CLI commands (`crush auto`, `crush next`, `crush auto init`), and crash recovery. After this milestone, auto-mode works end-to-end.

## Why This Milestone

M001 provides the data model and state machine. M002 wires it into a running loop that dispatches LLM calls with fresh sessions, advances state, and handles the full GSD lifecycle (research → plan → execute → verify → summarize → validate).

## User-Visible Outcome

### When this milestone is complete, the user can:

- Run `crush auto init` to interactively describe a vision and have Crush generate the milestone/slice/task structure
- Run `crush auto start` and watch Crush autonomously plan slices, execute tasks, and advance
- Run `crush next` to step through one unit at a time without auto-mode running
- Run `crush auto status` to see structured status: active milestone/slice/task, phase, cost so far, recent actions
- Run `crush auto stop` / `crush auto pause` to control the loop (pause finishes current unit before pausing)

### Entry point / environment

- Entry point: `crush auto init`, `crush auto start`, `crush next` CLI commands
- Environment: local dev terminal
- Live dependencies involved: configured LLM provider (Anthropic, OpenAI, etc.)

## Completion Class

- Contract complete means: unit tests for auto loop lifecycle, prompt template rendering, CLI command parsing
- Integration complete means: `crush auto start` with a seeded milestone dispatches units and advances through slices
- Operational complete means: crash recovery resumes from DB state. Pause/resume works. Lock file prevents concurrent instances.

## Final Integrated Acceptance

To call this milestone complete, we must prove:

- `crush auto init` produces a valid milestone/slice/task structure in the DB from a user-described vision
- `crush auto start` runs the full derive → dispatch → execute → finalize loop, advancing through at least one complete slice
- `crush next` executes exactly one unit and returns control
- `crush auto pause` finishes the current unit then stops dispatching
- Killing the process and re-running `crush auto start` resumes from the correct state

## Implementation Decisions

- **Dispatch via Coordinator:** Auto-mode dispatches units through existing `Coordinator.RunWithForcedTier()`. This reuses all existing agent infrastructure: model selection, session management, permission handling, OAuth refresh.
- **Model tier by unit type:** Research and planning units use the planning model. Task execution uses the main model. Summaries and validation use the background model.
- **Fresh child session per unit:** Each dispatched unit creates a new child session under a milestone-level parent session. Uses `session.Service.CreateTaskSession()`. Cost rolls up via session hierarchy.
- **Custom system prompts:** Auto-mode needs different system prompts per unit type. Will need to either expose prompt injection on the Coordinator or add a `RunWithPrompt()` method. The existing `buildAgent()` hardcodes the coder system prompt — auto-mode templates need a different path.
- **Interactive init:** `crush auto init` opens an interactive session where the user describes the vision. Crush generates the milestone/slice/task structure via a planning prompt. Results stored in SQLite.
- **Structured status:** `crush auto status` shows active milestone/slice/task, phase, cost so far, recent actions as formatted terminal output.
- **Finish-then-pause:** `crush auto pause` signals the loop to stop after the current unit completes. No in-progress work is discarded.
- **Lock file:** `.crush/auto.lock` in project root. Contains PID and timestamp. Stale lock detection by checking if PID is still alive.
- **Pub/sub events:** `pubsub.Broker[AutoEvent]` publishes state transitions, unit starts/completions, errors. TUI consumes these in M004.
- **Built-in templates:** Go templates in `internal/auto/templates/*.md.tpl`, loaded via `//go:embed`. Not user-customizable.
- **Non-interactive permissions:** Auto-mode sessions auto-approve all permission requests, matching the existing `RunNonInteractive()` pattern.

## Agent's Discretion

- Exact prompt template content for each unit type
- How much prior context (summaries) to inject per unit type
- Session naming conventions for auto-mode sessions
- Stale lock detection heuristics (timeout, PID check)
- Error message formatting for `crush auto status`

## Risks and Unknowns

- **System prompt injection path** — the existing Coordinator/SessionAgent builds agents with the coder system prompt. Auto-mode needs per-unit-type prompts. May require adding a `RunWithPrompt()` method or refactoring `buildAgent()` to accept a custom prompt.
- **Session cost aggregation** — need to verify that child session costs are queryable by parent session ID for milestone-level budget tracking.
- **Planning prompt quality** — the init flow depends on the planning prompt producing well-structured milestone/slice/task output that can be parsed and stored in SQLite. Prompt engineering risk.

## Existing Codebase / Prior Art

- `internal/agent/coordinator.go` — `Coordinator.Run()` / `RunWithForcedTier()` dispatches prompts. `buildAgent()` constructs `SessionAgent` with system prompt + tools.
- `internal/agent/agent.go` — `sessionAgent.Run()` handles queuing, model selection, returns `*fantasy.AgentResult`.
- `internal/agent/prompts.go` — `prompt.NewPrompt("name", template, opts...)` → `.Build(ctx, provider, model, cfg)`.
- `internal/app/app.go` — `RunNonInteractive()` is the headless pattern. Auto-approves permissions via `app.Permissions.AutoApproveSession()`.
- `internal/cmd/run.go` — cobra command pattern for `crush run`. Model for `crush auto` and `crush next`.
- `internal/session/session.go` — `CreateTaskSession()` creates child sessions. `Session.Cost` for budget tracking.
- `internal/pubsub/` — `Broker[T]`, `Event[T]` with typed `EventType`.

## Relevant Requirements

- R004 — Auto loop (primary)
- R005 — Fresh child sessions (primary)
- R006 — Unit-type-specific prompt templates (primary)
- R007 — `crush auto` CLI commands (primary)
- R008 — `crush next` standalone runner (primary)
- R009 — Crash recovery (primary)
- R017 — Pub/sub events (primary)
- R019 — Crush-native planning (primary)

## Scope

### In Scope

- Main auto loop: derive → dispatch → execute → finalize cycle
- AutoSession mutable state container
- Prompt templates for each unit type (research, plan, execute, summarize, validate)
- CLI: `crush auto [init|start|stop|pause|status]`, `crush next`
- Lock file management and crash recovery
- Pub/sub event publishing for auto-mode state changes
- Auto-approve permissions for auto-mode sessions
- Stubbed hooks for verification gates, budget, and stuck detection (real implementation in M003)

### Out of Scope

- Verification command execution (M003)
- Budget enforcement beyond tracking (M003)
- Stuck detection logic (M003)
- Context pressure monitoring (M003)
- TUI dashboard (M004)
- Git worktrees (M004)

## Open Questions

- Exact mechanism for injecting custom system prompts into the Coordinator dispatch path — may require a new method or parameter on `Run()`
