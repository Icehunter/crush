---
id: S01
parent: M002
milestone: M002
provides:
  - Engine struct with Run/Step/Pause/Stop lifecycle
  - DeriveState() pure-logic state derivation from DB
  - StateQuerier/SessionCreator/Dispatcher/StatusAdvancer interfaces for adapting to real services
  - LockFile with atomic acquire and stale PID detection
  - BuildPrompt() with 5 embedded Go templates for all unit types
  - AutoEvent types for pubsub integration
  - CLI commands: crush auto start/pause/stop/status (registered in root.go)
requires:
  []
affects:
  - S02
  - S03
key_files:
  - internal/auto/state.go
  - internal/auto/engine.go
  - internal/auto/lock.go
  - internal/auto/prompts.go
  - internal/auto/unit.go
  - internal/auto/events.go
  - internal/auto/status.go
  - internal/auto/templates/research.md.tpl
  - internal/auto/templates/plan_slice.md.tpl
  - internal/auto/templates/execute_task.md.tpl
  - internal/auto/templates/summarize.md.tpl
  - internal/auto/templates/validate.md.tpl
  - internal/cmd/auto.go
  - internal/cmd/root.go
key_decisions:
  - D007: Consuming-package interfaces (StateQuerier, SessionCreator, Dispatcher, StatusAdvancer) with lightweight Row types for testability
  - D008: Filesystem-based pause/stop signaling — .pause file for pause, PID from lock file + SIGTERM for stop
  - D002: Pause finishes current unit then stops (no wasted work)
  - Go embed.FS for compile-time template bundling
  - Lock file uses O_CREATE|O_EXCL with stale PID detection via signal 0
patterns_established:
  - Consuming-package interface pattern: define interfaces in internal/auto/ with lightweight Row types rather than importing db.Queries — enables full mock testing
  - Test querier+advancer coupling: mock advancer calls querier.Advance() so DeriveState sees fresh state each iteration
  - O_EXCL lock files: treat unparseable lock as held, not stale
  - Go template embed pattern: //go:embed templates/*.md.tpl with BuildPrompt(unitType, context) dispatcher
observability_surfaces:
  - AutoEvent published via pubsub.Broker on every state transition (UnitStarted, UnitCompleted, UnitFailed, LoopPaused, LoopStopped, StateTransition)
  - Engine.Status() returns EngineStatus snapshot (running/paused/idle, active unit, milestone progress)
  - stderr progress indicators in crush auto start: ▶ ✓ ✗ ⏸ ⏹
  - Lock file at {dataDir}/auto.lock with JSON PID + timestamp
  - Pause signal file at {dataDir}/auto.lock.pause
drill_down_paths:
  - .gsd/milestones/M002/slices/S01/tasks/T01-SUMMARY.md
  - .gsd/milestones/M002/slices/S01/tasks/T02-SUMMARY.md
  - .gsd/milestones/M002/slices/S01/tasks/T03-SUMMARY.md
  - .gsd/milestones/M002/slices/S01/tasks/T04-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T21:52:07.771Z
blocker_discovered: false
---

# S01: Core Auto Loop Engine + CLI

**Built the complete auto-mode engine: state derivation, engine loop with pause/stop, lock file, prompt templates, and CLI commands — 42 tests pass, project compiles.**

## What Happened

This slice built the autonomous execution engine for Crush from scratch across four tasks.

**T01 — State Derivation Layer** created the pure-logic foundation: UnitType enum with 5 dispatch phases (research, plan_slice, execute_task, summarize_slice, validate_milestone), Unit struct carrying dispatch context, AutoEvent types for pubsub, and DeriveState() which walks milestone→slice→task hierarchy respecting status, phase, sort_order, and depends_on. A StateQuerier interface with lightweight Row types decouples from sqlc, enabling full testability. 18 tests cover all scenarios.

**T02 — Engine + Lock File** built the Engine struct with Run/Step/Pause/Stop lifecycle. Run() acquires lock, loops derive→dispatch→advance→publish with pause/stop checks. Lock file uses O_CREATE|O_EXCL atomic creation with stale PID detection via signal 0. Defined SessionCreator/Dispatcher/StatusAdvancer interfaces for testability. Fixed pre-existing csync/maps.go vet failure. 14 new tests.

**T03 — Prompt Templates** created 5 Go template files (research, plan_slice, execute_task, summarize, validate) embedded at compile time via embed.FS. BuildPrompt() selects the right template by UnitType and renders with PromptContext. Replaced placeholder buildPrompt in engine.go with the real implementation.

**T04 — CLI + Integration Tests** wired everything into cobra commands: `crush auto start` launches the engine, `crush auto pause` writes a signal file, `crush auto stop` sends SIGTERM via PID from lock file, `crush auto status` reports engine state and next unit. Commands registered in root.go. 4 integration tests prove full lifecycle (6-unit sequence), step execution, pause mid-loop, and child session creation. Fixed flaky concurrent lock test and infinite-loop bug in test querier. CLI adapters are placeholder stubs until DB schema lands.

## Verification

All slice-level verification checks pass:
- `go test ./internal/auto/... -count=1` — 42 tests pass (18 state + 14 engine + 6 prompt + 4 integration)
- `go vet ./internal/auto/... ./internal/cmd/...` — clean
- `go build .` — project compiles successfully
- `go test ./internal/cmd/... -count=1` — no regressions in existing cmd tests

## Requirements Advanced

- R004 — Engine.Run() implements the derive→dispatch→advance loop with fresh sessions per unit. 42 tests prove the full lifecycle works.
- R005 — SessionCreator interface creates child sessions per unit under a milestone parent session. Integration test TestIntegration_ChildSessionsCreated verifies 1 parent + N children.
- R017 — AutoEvent types defined with 6 EventType constants. Engine publishes events via pubsub.Broker on every state transition. Integration tests verify event ordering.

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

CLI adapters (cmdStateQuerier, cmdSessionCreator, cmdDispatcher, cmdStatusAdvancer) are placeholder stubs returning "not wired yet" errors — the auto-mode DB schema for milestones/slices/tasks doesn't exist in the production DB yet, so full wiring is deferred. The engine, interfaces, and CLI structure are all correct and ready for wiring. StateQuerier uses lightweight Row types instead of importing db.Queries directly — cleaner consuming-package pattern.

## Known Limitations

`crush auto start` will fail at runtime until DB adapters are wired to real milestone/slice/task tables. The CLI commands compile and register correctly but the adapters are stubs.

## Follow-ups

Wire CLI adapters to real DB queries once auto-mode schema (milestones, slices, tasks tables) lands. Add telemetry/PostHog events for auto-mode lifecycle. Add `crush auto init` (S02) and `crush next` (S03).

## Files Created/Modified

- `internal/auto/unit.go` — UnitType enum (5 phases), Unit struct with dispatch context, String() formatter
- `internal/auto/events.go` — AutoEvent struct with 6 typed EventType constants for pubsub
- `internal/auto/status.go` — Status and Phase enums with string constants
- `internal/auto/state.go` — StateQuerier interface with Row types, DeriveState() walking milestone→slice→task
- `internal/auto/state_test.go` — 18 tests for state derivation
- `internal/auto/engine.go` — Engine struct with Run/Step/Pause/Stop, SessionCreator/Dispatcher/StatusAdvancer interfaces
- `internal/auto/engine_test.go` — 14 engine tests + fixed querier coupling
- `internal/auto/lock.go` — LockFile with O_EXCL acquire, stale PID detection, release
- `internal/auto/lock_test.go` — 6 lock file tests including concurrent acquire
- `internal/auto/prompts.go` — BuildPrompt with embed.FS template selection and PromptContext rendering
- `internal/auto/prompts_test.go` — Prompt template rendering tests for all 5 unit types
- `internal/auto/templates/research.md.tpl` — Research unit prompt template
- `internal/auto/templates/plan_slice.md.tpl` — Plan slice prompt template
- `internal/auto/templates/execute_task.md.tpl` — Execute task prompt template
- `internal/auto/templates/summarize.md.tpl` — Summarize slice prompt template
- `internal/auto/templates/validate.md.tpl` — Validate milestone prompt template
- `internal/auto/engine_integration_test.go` — 4 integration tests: full lifecycle, step, pause, child sessions
- `internal/cmd/auto.go` — Cobra commands for crush auto start/pause/stop/status
- `internal/cmd/root.go` — Registered autoCmd in root command
- `internal/csync/maps.go` — Fixed value receiver → pointer receiver for JSONSchemaAlias (pre-existing vet error)
