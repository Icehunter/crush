---
id: M003
title: "Safety Rails — Context"
status: complete
completed_at: 2026-03-28T05:43:41.725Z
key_decisions:
  - D009: Context pressure monitoring operates at engine level (session lifecycle), independent of agent-level StopWhen auto-summarization — still valid
  - D010: R011 validated with 7 tests proving budget enforcement end-to-end — still valid
  - D011: R012 validated with 12 tests proving stuck detection and escalation — still valid
  - Consistent safety gate pattern: nil/zero disables, gate checks at defined point in step(), exceeded pauses and publishes typed event
  - Short-circuit verification on first command failure rather than running all commands
  - Narrow single-method interfaces (BudgetQuerier, TokenQuerier) for clean testability
key_files:
  - internal/auto/verify.go
  - internal/auto/verify_test.go
  - internal/auto/budget.go
  - internal/auto/budget_test.go
  - internal/auto/stuck.go
  - internal/auto/stuck_test.go
  - internal/auto/context.go
  - internal/auto/context_test.go
  - internal/auto/events.go
  - internal/auto/engine.go
  - internal/auto/engine_integration_test.go
  - internal/auto/engine_budget_integration_test.go
  - internal/auto/engine_stuck_integration_test.go
  - internal/auto/engine_context_integration_test.go
  - internal/auto/engine_verify_integration_test.go
  - internal/config/config.go
  - internal/config/auto_test.go
  - internal/db/sql/sessions.sql
  - internal/db/sessions.sql.go
lessons_learned:
  - Consistent gate pattern across all four safety rails (nil/zero disable, check-at-point, pause-and-publish) reduced cognitive load and made each subsequent gate faster to implement
  - Narrow single-method interfaces (BudgetQuerier, TokenQuerier) decouple gates from the full DB layer and make mocking trivial
  - Pre-existing go vet failures surface in worktrees — run go vet early (K005 reinforced)
  - NewEngine parameter list grew with each gate (verifier, budgetChecker, budgetCeiling, stuckDetector, contextMonitor) — updating all call sites across 3-4 test files per gate was mechanical but time-consuming. Consider an options pattern or config struct for future gates.
---

# M003: Safety Rails — Context

**Added four safety rails to the auto-mode engine: verification gates with retry, dollar-cost budget ceiling, stuck detection with diagnostic escalation, and context pressure monitoring — all proven by ~100 tests.**

## What Happened

M003 delivered four safety rails that make auto-mode trustworthy for unattended use, completing the safety layer of the autonomous orchestration system.

**S01 (Auto Config + Verification Gates)** laid the foundation: added the `auto` configuration section to crush.json with four fields (verification_commands, budget_ceiling, stuck_threshold, worktree_mode), implemented the Verifier interface and ShellVerifier that runs configurable shell commands after task execution, and wired a verification gate into engine.step() with single-retry-on-failure using truncated diagnostic prompts. 10 tests prove pass/fail/empty/truncate paths and full engine verify→retry→succeed and verify→retry→fail integration paths.

**S02 (Budget Ceiling)** added dollar-cost tracking via SumChildSessionCosts SQL query and a BudgetChecker interface with DBBudgetChecker implementation. The engine budget gate checks cumulative child session costs before each dispatch — when the ceiling is reached, the engine pauses and publishes EventBudgetExceeded. 7 tests prove cost aggregation and engine gate behavior including zero-ceiling and nil-checker skip paths.

**S03 (Stuck Detection + Context Pressure)** delivered two final rails. StuckDetector uses a per-unit circular ring buffer sliding window — when >50% of recent dispatches fail for the same unit, the engine retries with a diagnostic prompt, then pauses with EventStuckDetected if still stuck. ContextMonitor compares cumulative session token usage against a configurable fraction of the model context window, pausing with EventContextPressure when exceeded. 12 stuck tests + 9 context tests prove all paths including threshold boundaries, sliding eviction, nil/zero safety, and full engine integration.

All four safety gates follow a consistent pattern established during the milestone: nil/zero config disables the gate, the gate checks at a defined point in step(), and exceeded conditions pause the engine and publish a typed event via pubsub.

## Success Criteria Results

- ✅ **Verification gates run after task execution with retry**: ShellVerifier runs configured commands, short-circuits on first failure, truncates output to 4096 bytes for diagnostic prompt. Engine retries once on failure. Proven by TestIntegration_VerifyRetrySucceed and TestIntegration_VerifyRetryFail.
- ✅ **Dollar-cost budget ceiling pauses auto-mode**: SumChildSessionCosts query + engine budget gate + EventBudgetExceeded. Proven by TestIntegration_BudgetExceeded_PausesEngine and 3 additional integration tests.
- ✅ **Stuck detection with retry-then-pause escalation**: StuckDetector with per-unit sliding window, diagnostic retry, then pause+EventStuckDetected. Proven by TestIntegration_StuckRetrySucceed, TestIntegration_StuckRetryFail_PausesEngine.
- ✅ **Context pressure monitoring with wrap-up signaling**: ContextMonitor with TokenQuerier, pauses+EventContextPressure when threshold exceeded. Proven by TestIntegration_ContextPressure_PausesEngine.
- ✅ **Auto config section in crush.json**: Four fields round-trip through parsing. Proven by TestAutoConfig, TestAutoConfig_Empty, TestAutoConfig_PartialFields.
- ✅ **All tests pass**: `go test ./internal/auto/ -count=1` passes (~100 tests), `go test ./internal/config/ -run TestAutoConfig -count=1` passes (3 tests).

## Definition of Done Results

- ✅ All 3 slices complete: S01 ✅, S02 ✅, S03 ✅
- ✅ All slice summaries exist: S01-SUMMARY.md, S02-SUMMARY.md, S03-SUMMARY.md
- ✅ Cross-slice integration verified: S02 and S03 both consume S01's engine infrastructure (NewEngine, step(), pubsub events). All NewEngine call sites updated across all test files. Full test suite passes with all gates active.
- ✅ `go vet ./internal/auto/ ./internal/db/` clean
- ✅ `go build ./internal/auto/` clean

## Requirement Outcomes

- **R010** (quality-attribute): active → **validated**. Integration tests prove verify→retry→succeed and verify→retry→fail paths. ShellVerifier unit tests prove command execution, short-circuit, truncation. 10 tests total.
- **R011** (constraint): active → **validated**. 7 tests prove budget enforcement: 3 unit for cost aggregation + 4 integration for engine gate. SumChildSessionCosts + engine step() gate + EventBudgetExceeded.
- **R012** (failure-visibility): active → **validated**. StuckDetector with per-unit sliding window. 9 unit + 3 integration tests prove stuck detection, diagnostic retry, pause escalation, below-threshold normal operation.
- **R013** (continuity): active → **validated**. ContextMonitor with TokenQuerier. 6 unit + 3 integration tests prove threshold comparison, pause+EventContextPressure, nil/zero safety, error propagation.
- **R014** (operability): active → **validated**. 3 config parsing tests prove all four auto config fields round-trip through crush.json.

## Deviations

S01/T01 required copying M002 DB artifacts beyond task plan scope. Pre-existing go vet failure in csync/maps.go fixed during S01 closure. sqlc generated SumChildSessionCosts with sql.NullString param requiring transparent wrapping in S02. Stuck gate checks before dispatch (at step entry) rather than after in S03 — a design choice within plan constraints.

## Follow-ups

NewEngine parameter list is growing — consider an EngineConfig/options struct for future gates. Verification retry count is hardcoded to 1 — could be made configurable. Budget checking queries DB on every step() — add time-based cache if dispatch frequency increases. CLI adapters remain placeholder stubs pending real DB wiring (M004+ scope).
