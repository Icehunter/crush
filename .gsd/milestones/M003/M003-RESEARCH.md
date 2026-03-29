# M003: Safety Rails — Research

**Date:** 2026-03-27
**Status:** Complete

## Summary

M003 adds four safety rails to Crush's auto-mode: verification gates (run test/lint commands after task execution), dollar-cost budget ceiling, stuck detection with retry-then-pause escalation, and context pressure monitoring. The codebase is well-structured for this work — the `Engine.step()` method in `internal/auto/engine.go` is the natural insertion point for all four rails, and existing patterns (shell execution, session cost tracking, token usage via `fantasy.Usage`, context window metadata via `catwalk.Model.ContextWindow`) provide the building blocks.

The primary recommendation is to build in order of risk: **config first** (enables everything else), **verification gates** (highest user value, moderate complexity around output parsing), **budget ceiling** (straightforward SQL aggregation + engine check), **stuck detection** (in-memory sliding window, depends on verification to generate failure signals), and **context pressure** (most complex, requires interaction with `fantasy`'s `StopCondition` and `PrepareStep` hooks).

All four rails integrate at the engine level (`Engine.step()`) and don't require changes to the LLM abstraction layer or TUI. The `Auto` config struct is a new top-level field on `Config`, following the same pattern as `Options`, `Tools`, and `Permissions`.

## Recommendation

**Approach: Four slices in dependency order, all scoped to `internal/auto/` and `internal/config/`.**

1. **S01: Auto Config + Verification Gates** — Add `Auto` struct to config, parse from crush.json, implement verification runner that executes configured commands via `shell.Shell` after each task dispatch. On failure, re-dispatch with diagnostic prompt containing failure output. This is the highest-value rail and unblocks stuck detection.

2. **S02: Budget Ceiling** — Add SQLC query to sum child session costs by parent session ID. Engine checks cumulative cost before each dispatch. Publish `EventBudgetExceeded` and pause. Straightforward.

3. **S03: Stuck Detection + Context Pressure** — Implement sliding window over recent dispatch results. If >50% failures for same unit, retry with diagnostic. If diagnostic retry fails, pause. Context pressure monitors `fantasy.Usage.InputTokens` against `catwalk.Model.ContextWindow`, sends wrap-up signal at configurable threshold (default 80%).

This ordering ensures each slice is independently testable and the dependency chain is clean: S01 provides the config that S02 and S03 consume, and S01's verification failure signal feeds S03's stuck detection.

## Implementation Landscape

### Key Files

- `internal/config/config.go` — Add `Auto *AutoConfig` field to `Config` struct. Follow the pattern of `Options`, `Permissions`, `Tools`.
- `internal/config/load.go` — No changes needed; JSON unmarshaling handles new struct fields automatically via `go-jsons` merge.
- `internal/auto/engine.go` (on `milestone/M002` branch) — `Engine.step()` is the insertion point. After `e.dispatch.RunWithForcedTier()` succeeds, run verification. Before dispatch, check budget. Track dispatch results for stuck detection.
- `internal/auto/events.go` — Add new event types: `EventVerificationFailed`, `EventVerificationRetry`, `EventBudgetExceeded`, `EventStuckDetected`, `EventContextPressure`.
- `internal/shell/shell.go` — `Shell.Exec()` returns `(stdout, stderr, error)`. Exit code via `shell.ExitCode(err)`. This is the API for running verification commands.
- `internal/db/sql/sessions.sql` — Add `SumChildSessionCosts` query: `SELECT COALESCE(SUM(cost), 0) FROM sessions WHERE parent_session_id = ?`.
- `internal/session/session.go` — `Session.Cost` field tracks per-session dollar cost. `Session.PromptTokens` + `Session.CompletionTokens` for token counts.
- `charm.land/fantasy@v0.17.1` — `AgentResult.TotalUsage` (type `Usage`) provides `InputTokens`, `OutputTokens`, `TotalTokens`. Used for context pressure monitoring.
- `charm.land/catwalk@v0.31.1` — `catwalk.Model.ContextWindow` (int64) provides the model's max context window. Already used in `internal/agent/agent.go` for auto-summarization thresholds.
- `internal/agent/agent.go` — Lines 52-55 define context window threshold constants (`largeContextWindowThreshold = 200_000`, `largeContextWindowBuffer = 20_000`, `smallContextWindowRatio = 0.2`). Lines 443-453 show the existing `StopWhen` pattern for context pressure. Auto-mode's context pressure monitoring should reuse these thresholds or make them configurable.

### Build Order

1. **Config struct first** — `AutoConfig` struct with `VerificationCommands`, `BudgetCeiling`, `StuckThreshold`, `ContextPressureThreshold`, `WorktreeMode`. This unblocks all other slices.
2. **Verification gates** — New `Verifier` struct in `internal/auto/verify.go`. Runs commands via `shell.Shell`, parses exit codes. `Engine.step()` calls verifier after successful dispatch. On failure, re-dispatches with diagnostic prompt.
3. **Budget ceiling** — New SQLC query + `BudgetChecker` in `internal/auto/budget.go`. Engine calls `BudgetChecker.Check()` before dispatch.
4. **Stuck detection** — New `StuckDetector` struct in `internal/auto/stuck.go`. Sliding window of dispatch results. Engine feeds results into detector, detector returns stuck/not-stuck.
5. **Context pressure** — New `ContextMonitor` in `internal/auto/context.go`. Consumes `fantasy.Usage` after each dispatch, compares against model's `ContextWindow`. Signals wrap-up via a follow-up message or forces session handoff.

### Verification Approach

- **Unit tests:** Each rail gets its own `_test.go`. Mock `shell.Shell` for verification commands. Mock `SessionCreator` / DB for budget queries. In-memory sliding window for stuck detection.
- **Integration tests:** `Engine.Run()` with a mock dispatcher that simulates failures → verify verification retries, stuck detection pauses, budget enforcement pauses.
- **Config tests:** JSON round-trip for `Auto` section. Defaults applied correctly. Missing section means disabled.
- **Commands:** `go test ./internal/auto/... ./internal/config/...`

## Constraints

- **CGO_ENABLED=0** — No C dependencies. SQLite via `modernc.org/sqlite` (already in use). All new code must be pure Go.
- **Shell execution via `mvdan.cc/sh`** — Verification commands run through the POSIX shell emulator, not `os/exec`. This means they inherit the same security model and command blocking as the bash tool.
- **`fantasy` is an external dependency** — Cannot modify its `AgentResult` or `Usage` types. Context pressure monitoring must work with the data `fantasy` provides via return values, not by modifying `fantasy` internals.
- **Config is loaded once** — `ConfigStore` loads at startup and is treated as read-only after that. The `Auto` config section follows this pattern — no hot-reload.
- **M002 branch not merged to main** — All M003 work builds on `milestone/M002`. The engine, events, prompts, lock file, and state machine are on that branch. The current `main` only has M001's domain model files.

## Common Pitfalls

- **Verification command timeout** — Shell commands can hang. Must enforce a timeout (context deadline) on verification command execution. The existing `Shell.Exec()` accepts a `context.Context` — use `context.WithTimeout()`.
- **Budget cost aggregation race** — Multiple child sessions may update cost concurrently. The SQLC `SUM(cost)` query reads committed state, so this is safe as long as we check *before* dispatching (not after). A dispatch that brings cost over the limit is acceptable; the next dispatch will be blocked.
- **Context pressure vs. auto-summarization** — `internal/agent/agent.go` already has a `StopWhen` condition that triggers auto-summarization at ~80% context usage. Auto-mode's context pressure monitoring is a *separate* concern — it decides whether to continue with the same session or start a fresh one for the same task. These must not conflict.
- **Diagnostic prompt injection** — When re-dispatching with verification failure output, the failure output becomes part of the prompt. Must sanitize or truncate to prevent prompt injection or context overflow from verbose test output.

## Open Risks

- **Verification command exit code semantics** — Most tools (go test, eslint, golangci-lint) use exit code 0 for success and non-zero for failure. But some tools have nuanced exit codes (e.g., eslint exit 2 = fatal error vs. 1 = lint errors). For M003, treating any non-zero exit code as failure is sufficient. Nuanced exit code handling can be added later.
- **Context pressure forced handoff** — When auto-mode forces a session handoff (context too full), the new session needs enough context to continue the task. This requires synthesizing a "continue-here" prompt with task state. The quality of this handoff is hard to test automatically — it depends on LLM behavior.
- **Cost tracking accuracy** — `Session.Cost` is updated incrementally via `UpdateSessionTitleAndUsage`. If a dispatch crashes mid-execution, the cost may not reflect all API calls made. This is an existing limitation in Crush's cost tracking, not something M003 introduces.

## Sources

- `charm.land/fantasy@v0.17.1` — `AgentResult.TotalUsage` (type `Usage`): `InputTokens`, `OutputTokens`, `TotalTokens`, `ReasoningTokens`, `CacheCreationTokens`, `CacheReadTokens` (inspected in `model.go`)
- `charm.land/catwalk@v0.31.1` — `catwalk.Model.ContextWindow` (int64) at line 83 of `provider.go`
- `internal/agent/agent.go` — Existing context pressure thresholds at lines 52-55, `StopWhen` condition at lines 443-453
- `internal/shell/shell.go` — `Shell.Exec(ctx, command) (stdout, stderr, error)`, `ExitCode(err) int`
- `internal/auto/engine.go` (M002 branch) — `Engine.step()` derive→dispatch→advance loop, `SessionCreator`, `Dispatcher`, `StatusAdvancer` interfaces
