---
id: M002
title: "Auto Loop + Session Management"
status: complete
completed_at: 2026-03-28T04:36:51.882Z
key_decisions:
  - D002: Pause finishes current unit then stops — no wasted work, clean unit boundaries
  - D006: Three-slice decomposition — S01 (engine+CLI), S02 (auto init), S03 (crush next) — engine without CLI isn't runnable
  - D007: Filesystem-based pause/stop signaling — .pause file for pause, PID from lock file + SIGTERM for stop
  - Consuming-package interfaces with lightweight Row types instead of importing db.Queries — enables full mock testing
  - Planning tools return JSON success/error responses for LLM consumption, not Go errors
  - Go embed.FS for compile-time template bundling
  - crush next is a top-level command, not under crush auto
key_files:
  - internal/auto/engine.go
  - internal/auto/state.go
  - internal/auto/lock.go
  - internal/auto/prompts.go
  - internal/auto/unit.go
  - internal/auto/events.go
  - internal/auto/status.go
  - internal/auto/init.go
  - internal/auto/init_tools.go
  - internal/auto/milestone.go
  - internal/auto/slice.go
  - internal/auto/task.go
  - internal/auto/templates/research.md.tpl
  - internal/auto/templates/plan_slice.md.tpl
  - internal/auto/templates/execute_task.md.tpl
  - internal/auto/templates/summarize.md.tpl
  - internal/auto/templates/validate.md.tpl
  - internal/auto/templates/init.md.tpl
  - internal/cmd/auto.go
  - internal/cmd/next.go
  - internal/cmd/root.go
lessons_learned:
  - Consuming-package interfaces with lightweight Row types are significantly more testable than importing the DB layer directly — mock coupling is minimal
  - Test querier+advancer coupling is essential: mock advancer must call querier.Advance() so DeriveState sees fresh state each iteration, otherwise tests loop infinitely
  - JSON error responses from LLM tools enable self-correction — returning Go errors breaks the LLM conversation
  - Filesystem-based signaling (pause files, lock files) is simple and observable — ls shows state without needing IPC
  - Placeholder adapter stubs are acceptable when the interface boundary is well-defined — real wiring can happen later without changing the engine or CLI code
  - O_EXCL lock files should treat unparseable content as held (not stale) for safety
---

# M002: Auto Loop + Session Management

**Built the complete auto-mode execution engine with state derivation, engine loop (run/step/pause/stop), lock file, prompt templates, interactive planning (crush auto init), and manual stepper (crush next) — 83 tests pass across the auto package.**

## What Happened

M002 delivers the autonomous execution engine for Crush across three slices.

**S01: Core Auto Loop Engine + CLI** (high risk, 4 tasks) built the foundation: UnitType enum with 5 dispatch phases, DeriveState() walking milestone→slice→task hierarchy with dependency gating, Engine struct with Run/Step/Pause/Stop lifecycle, LockFile with O_EXCL atomic creation and stale PID detection, 5 embedded Go template prompts (research, plan_slice, execute_task, summarize, validate), and CLI commands (crush auto start/pause/stop/status). 42 tests cover state derivation, engine mechanics, lock files, prompts, and integration scenarios including full 6-unit lifecycle and child session creation.

**S02: Interactive Planning — crush auto init** (medium risk, 3 tasks) added the planning entry point: three LLM-callable planning tools (create_milestone, create_slice, create_task) returning JSON for LLM self-correction, init.md.tpl template, RunInit() function constructing a restricted SessionAgent, and the crush auto init CLI command. 27 new tests bring the total to 69.

**S03: Manual Stepper — crush next** (low risk, 1 task) added the single-step counterpart: crush next [milestone-id] calls Engine.Step(), subscribes to broker events for stderr output, and exits. 3 tests for command registration, args validation, and help content.

All CLI adapters use placeholder stubs — real DB wiring is deferred until the auto-mode schema lands in the production DB path. The engine, interfaces, templates, and CLI structure are complete and ready for wiring.

Key architectural patterns established: consuming-package interfaces with lightweight Row types for full testability, filesystem-based pause/stop signaling, JSON error responses for LLM tool consumption, Go embed.FS for compile-time template bundling.

## Success Criteria Results

No formal success criteria were recorded in the milestone definition (empty arrays in DB). The implicit criteria from the roadmap — deliver all 3 slices — are met:

- ✅ S01 Core Auto Loop Engine + CLI — delivered with 42 tests
- ✅ S02 Interactive Planning — crush auto init — delivered with 27 additional tests
- ✅ S03 Manual Stepper — crush next — delivered with 3 tests
- ✅ `go build .` compiles clean
- ✅ `go test ./internal/auto/... -count=1` — 83 tests pass
- ✅ `go vet ./internal/auto/... ./internal/cmd/...` — clean
- ✅ `go test ./internal/cmd/... -count=1` — passes with no regressions

## Definition of Done Results

- ✅ All 3 slices marked complete with summaries and UAT documents
- ✅ S01-SUMMARY.md, S02-SUMMARY.md, S03-SUMMARY.md all exist
- ✅ S01-UAT.md, S02-UAT.md, S03-UAT.md all exist
- ✅ Cross-slice integration: S02 and S03 both depend on S01 and consume its Engine, DeriveState, and pubsub types correctly
- ✅ 213 files changed in git diff (non-.gsd code changes verified)

## Requirement Outcomes

### R004 (active → active, advanced)
Engine.Run() implements the derive→dispatch→advance loop with fresh sessions per unit. TestIntegration_FullLoopLifecycle proves the full 6-unit sequence. CLI adapters are placeholder stubs pending DB wiring — requirement advanced but not yet validated.

### R005 (active → active, advanced)
SessionCreator interface creates child sessions per unit under a milestone parent session. TestIntegration_ChildSessionsCreated verifies 1 parent + N children pattern. Advanced but not validated — real session creation needs production DB wiring.

### R006 (active → active, advanced)
6 Go template prompts delivered: research.md.tpl, plan_slice.md.tpl, execute_task.md.tpl, summarize.md.tpl, validate.md.tpl, init.md.tpl. BuildPrompt() and BuildInitPrompt() render with PromptContext.

### R007 (active → active, advanced)
crush auto start/pause/stop/status CLI subcommands registered and tested. Placeholder adapters — not yet end-to-end functional.

### R008 (active → active, advanced)
crush next [milestone-id] registered as top-level command with Engine.Step() and event subscription. Placeholder adapters.

### R009 (active → active, advanced)
Lock file prevents concurrent instances (O_EXCL + stale PID detection). Crash recovery reads DB state. Tests cover concurrent acquire, stale reclamation.

### R017 (active → active, advanced)
AutoEvent types defined with 6 EventType constants (UnitStarted, UnitCompleted, UnitFailed, LoopPaused, LoopStopped, StateTransition). Engine publishes via pubsub.Broker. Integration tests verify event ordering.

### R019 (active → active, advanced)
crush auto init decomposes user vision into milestones/slices/tasks via LLM tool calls. Planning tools + init template delivered.

## Deviations

CLI adapters (cmdStateQuerier, cmdSessionCreator, cmdDispatcher, cmdStatusAdvancer) are placeholder stubs returning errors — the auto-mode DB schema exists but production DB wiring is deferred. This means crush auto start and crush next will not execute real units yet. The engine, interfaces, and CLI structure are complete and ready for wiring. No formal success criteria or definition of done were recorded in the milestone DB entry, so verification was based on the roadmap's slice completion and test passing.

## Follow-ups

Wire CLI adapters to real DB queries for end-to-end execution. Add telemetry/PostHog events for auto-mode lifecycle. Implement verification commands (R010), budget ceiling (R011), stuck detection (R012), and context window management (R013) in M003. Add TUI sidebar (R015) and TUI controls (R016) in M004.
