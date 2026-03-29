# S03: Stuck Detection + Context Pressure — Research

**Date:** 2026-03-27
**Status:** Complete

## Summary

S03 adds two safety rails to the auto-mode engine: (1) **stuck detection** — a sliding window over recent dispatch results that detects repeated failures for the same unit, retries with a diagnostic prompt, and pauses if still stuck; and (2) **context pressure monitoring** — tracking token usage against the model's context window and signaling wrap-up at a configurable threshold.

Both features integrate into `Engine.step()` following the same gate pattern established by S01 (verification) and S02 (budget). The codebase is well-prepared: the engine already has the `Verifier` and `BudgetChecker` interface patterns, events infrastructure, and mock testing patterns. The main design challenge is the **Dispatcher interface gap** — the current `Dispatcher` interface returns only `error`, not `*fantasy.AgentResult`, so the engine has no access to token usage from dispatches. Context pressure monitoring must either (a) expand the `Dispatcher` interface to return usage data, or (b) query session token counts from the DB after each dispatch. Option (b) is cleaner — it avoids changing an interface that S01 and S02 tests already mock, and the session's `PromptTokens`/`CompletionTokens` fields are already updated by the agent after each run.

**Recommendation:** Build stuck detection first (in-memory, no interface changes), then context pressure (needs a way to read session token usage + model context window). Two new files: `stuck.go` and `context.go`, plus engine wiring and integration tests.

## Recommendation

**Two features, built in order:**

1. **Stuck detection** (`stuck.go`) — Pure in-memory sliding window. New `StuckDetector` struct tracks the last N dispatch outcomes keyed by unit ID. Engine feeds results after each step. When >50% of the window entries for a unit are failures, the detector flags it. Engine retries with a diagnostic prompt (reusing the existing retry pattern from verification). If retry fails, engine pauses and publishes `EventStuckDetected`. The `stuck_threshold` config field (already in `AutoConfig`, default 5) controls window size.

2. **Context pressure** (`context.go`) — New `ContextMonitor` struct. After each dispatch, engine queries cumulative session token usage and compares against a context window value. When usage exceeds a configurable threshold (default 80%), publish `EventContextPressure` and pause. The context window and token data must be supplied to the monitor — either via a new `TokenQuerier` interface (reads session tokens from DB) or passed directly by the engine. The monitor does NOT manage session handoff (that's a future concern) — it simply signals that context is getting full and pauses.

This ordering works because stuck detection is self-contained (in-memory state only) while context pressure requires a mechanism to read token usage.

## Implementation Landscape

### Key Files

- `internal/auto/engine.go` — Add `stuckDetector` and `contextMonitor` fields to `Engine`. Wire stuck detection after verification gate (feed pass/fail results into detector). Wire context pressure check after dispatch. Add two new constructor parameters.
- `internal/auto/stuck.go` — New file. `StuckDetector` struct with `Record(unitKey string, passed bool)`, `IsStuck(unitKey string) bool` methods. Sliding window backed by a circular buffer per unit key. Thread-safe (sync.Mutex).
- `internal/auto/stuck_test.go` — New file. Unit tests: window fills up, >50% failures triggers stuck, mixed results don't trigger, window slides (old entries drop off), empty window returns not-stuck.
- `internal/auto/context.go` — New file. `ContextMonitor` struct with `Check(promptTokens, completionTokens, contextWindow int64) bool` method. Returns true when usage exceeds threshold. Threshold is a float64 (0.0-1.0, default 0.8).
- `internal/auto/context_test.go` — New file. Unit tests: below threshold returns false, at threshold returns true, above threshold returns true, zero context window returns false (safety).
- `internal/auto/engine_stuck_integration_test.go` — New file. Integration tests: dispatch→fail→stuck-detected→retry→succeed path, dispatch→fail→stuck-detected→retry→fail→pause path.
- `internal/auto/engine_context_integration_test.go` — New file. Integration test: dispatch causes token usage to exceed threshold → EventContextPressure published → engine pauses.
- `internal/auto/events.go` — Add `EventStuckDetected` and `EventContextPressure` constants.

### Patterns to Follow

- **Interface pattern from S01/S02:** `Verifier` and `BudgetChecker` are interfaces with mock implementations in tests. `StuckDetector` can be a concrete struct (not interface) since it's pure in-memory — no need to mock it. `ContextMonitor` is also a simple struct.
- **Engine gate pattern:** Verification runs after dispatch, budget runs before dispatch. Stuck detection feeds results after verification gate. Context pressure checks after dispatch.
- **Event pattern:** `EventBudgetExceeded` in S02 shows the pattern — publish event, set `paused` flag, return nil.
- **Mock pattern from integration tests:** `fixedSequenceQuerier`, `recordingDispatcher`, `mockAdvancer` — use these for stuck/context integration tests.
- **`NewEngine` constructor:** Already takes verifier + budgetChecker. Add stuckDetector + contextMonitor. Since these are concrete structs not interfaces, pass them directly (nil means disabled, like verifier/budgetChecker).

### Dispatcher Interface — Do NOT Change

The `Dispatcher` interface returns only `error`. The real `Coordinator.RunWithForcedTier` returns `(*fantasy.AgentResult, error)` with token usage in `AgentResult.TotalUsage`. However, changing `Dispatcher` to return `AgentResult` would break all existing mocks and tests (S01, S02 integration tests).

**Instead:** For context pressure, the engine needs a way to query cumulative session token usage after dispatch. Two approaches:
1. **Query via SessionCreator** — extend `SessionCreator` interface to add `GetSessionTokens(ctx, sessionID) (promptTokens, completionTokens int64, err error)`. This is clean but requires a new DB query.
2. **Pass context window + threshold to engine, use a TokenQuerier interface** — minimal new interface: `TokenQuerier` with `GetTokenUsage(ctx, sessionID) (int64, int64, error)`.
3. **Simplest: just track dispatch count as a proxy** — but this is inaccurate.

**Recommended: Option 2** — a small `TokenQuerier` interface, same pattern as `BudgetQuerier`. The engine calls it after dispatch to get cumulative tokens, compares against context window × threshold.

### Build Order

1. **Stuck detection** — `stuck.go` + `stuck_test.go` + engine wiring + integration test. No interface changes. Pure in-memory. Lowest risk.
2. **Context pressure** — `context.go` + `context_test.go` + `TokenQuerier` interface + engine wiring + integration test. Slightly more complex due to token querying.
3. **Event types** — Add `EventStuckDetected` and `EventContextPressure` to `events.go` early (needed by both).

### Verification Approach

- `go vet ./internal/auto/...` — clean
- `go build ./internal/auto/` — clean
- `go test ./internal/auto/ -run 'TestStuck' -count=1 -v` — stuck detection unit tests
- `go test ./internal/auto/ -run 'TestContextMonitor' -count=1 -v` — context pressure unit tests
- `go test ./internal/auto/ -run 'TestIntegration_Stuck' -count=1 -v` — stuck integration tests
- `go test ./internal/auto/ -run 'TestIntegration_Context' -count=1 -v` — context integration tests
- `go test ./internal/auto/ -count=1` — all auto tests pass (no regressions)
- `go test ./internal/config/ -count=1` — config tests still pass

## Constraints

- **CGO_ENABLED=0** — Pure Go only. No C dependencies.
- **Dispatcher interface is frozen** — Returns `error` only. Token usage must come from another source (session DB or new interface).
- **Config is loaded once** — `AutoConfig.StuckThreshold` is read at engine creation time, not hot-reloaded.
- **Context pressure is engine-level, not agent-level** — D009 established that agent-level auto-summarization (`StopWhen` in agent.go) and engine-level context pressure are independent concerns. The engine decides session lifecycle; the agent handles within-session summarization.
- **Session token counts are updated by the agent** — `session.PromptTokens` and `session.CompletionTokens` are updated via `UpdateTitleAndUsage` after each agent step. The engine can read these values after dispatch returns.

## Common Pitfalls

- **Stuck detection unit key** — Must uniquely identify a unit. Use `fmt.Sprintf("%s/%s/%s", unit.MilestoneID, unit.SliceID, unit.TaskID)` to distinguish different tasks. If only MilestoneID is used, all tasks in a milestone share one window.
- **Stuck detection vs. verification retry** — The verification gate already retries once on failure. Stuck detection is a *separate* concern — it tracks whether the engine keeps failing on the same unit across multiple loop iterations (not within a single step). The engine's `Run()` loop already retries failed steps with a 2-second backoff. Stuck detection should track outcomes at the `step()` level, not inside `runVerificationGate()`.
- **Context pressure false positives** — If the model has a very large context window (e.g., 1M tokens), 80% threshold is 800K tokens. A single dispatch is unlikely to reach this. Context pressure is mainly relevant for smaller context windows or long-running sessions with many dispatches.
- **Thread safety on StuckDetector** — Engine calls `Record()` and `IsStuck()` from the same goroutine (the loop), but a `sync.Mutex` is still prudent for the `Status()` snapshot path.

## Open Risks

- **Token usage accuracy after dispatch** — `session.PromptTokens`/`CompletionTokens` are updated by the agent asynchronously via `UpdateTitleAndUsage`. There could be a race where the engine reads session tokens before the agent has written them. Mitigation: the `Dispatcher.RunWithForcedTier()` call blocks until the agent finishes, so by the time it returns, the session should be updated. But verify this assumption.
- **Context window value availability** — The engine needs to know the model's context window (`catwalk.Model.ContextWindow`). This value is on `agent.Model.CatwalkCfg.ContextWindow`. The engine doesn't currently have access to the agent's model config. Options: (a) pass context window as a constructor parameter, (b) add it to a config field, (c) have `TokenQuerier` also return the context window. Option (a) is simplest.
- **Scope of "pause"** — Both stuck detection and context pressure cause the engine to pause. The pause mechanism is already implemented (set `paused` flag, publish event, return nil from step). This is the same pattern as budget exceeded.
