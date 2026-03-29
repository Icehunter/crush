# S02: Budget Ceiling — Research

**Date:** 2026-03-28
**Status:** Complete

## Summary

S02 adds dollar-cost budget enforcement to the auto-mode engine. The engine already creates a parent session per `Run()` call and child sessions per dispatched unit. Session costs are tracked incrementally via `UpdateSessionTitleAndUsage` (cost field on the sessions table). The missing piece is (1) a SQLC query that sums child session costs by parent session ID, (2) a `BudgetChecker` abstraction the engine calls before each dispatch, and (3) wiring the check into `engine.step()` with an `EventBudgetExceeded` event that pauses the loop.

This is straightforward work — the config field `BudgetCeiling float64` already exists on `AutoConfig` (delivered in S01), the session table already has `parent_session_id` and `cost` columns, and the engine's `step()` method has a clear insertion point before the dispatch call. The main design decision is where the budget check sits relative to existing gates: it should run **before** dispatch (no point dispatching if we're over budget), unlike verification which runs **after**.

## Recommendation

**Approach: New `BudgetChecker` interface in `internal/auto/`, backed by a SQLC query, wired into `engine.step()` before dispatch.**

1. Add a `SumChildSessionCosts` query to `internal/db/sql/sessions.sql` and regenerate sqlc.
2. Define a `BudgetChecker` interface in `internal/auto/budget.go` with a single method `CheckBudget(ctx, parentSessionID) (totalCost float64, err error)`. Implement it with the SQLC query. The engine compares `totalCost >= ceiling` and short-circuits.
3. Wire the checker into `engine.step()`: after deriving state and before creating the child session, call `CheckBudget`. If over ceiling, publish `EventBudgetExceeded`, set engine state to paused, return nil (clean exit, not an error).
4. Add `EventBudgetExceeded` to `events.go`.
5. Unit test the budget checker with a mock querier. Integration test the engine with a mock budget checker that returns configurable costs.

## Implementation Landscape

### Key Files

- `internal/db/sql/sessions.sql` — Add new query: `-- name: SumChildSessionCosts :one` / `SELECT COALESCE(SUM(cost), 0) AS total_cost FROM sessions WHERE parent_session_id = ?`. Then run `sqlc generate` to produce the Go code.
- `internal/db/sessions.sql.go` (generated) — Will contain the new `SumChildSessionCosts` method after sqlc generation.
- `internal/db/querier.go` (generated) — Will add `SumChildSessionCosts` to the `Querier` interface.
- `internal/auto/budget.go` (new) — `BudgetChecker` interface with `CheckBudget(ctx context.Context, parentSessionID string) (float64, error)`. Concrete `DBBudgetChecker` wraps the SQLC query. A `NilBudgetChecker` (or nil check) for when budget ceiling is 0 / not configured.
- `internal/auto/engine.go` — Add `budgetChecker BudgetChecker` field and `budgetCeiling float64` to `Engine`. Update `NewEngine` signature to accept them. In `step()`, add budget gate before child session creation: if `budgetCeiling > 0 && budgetChecker != nil`, call `CheckBudget`, compare to ceiling.
- `internal/auto/events.go` — Add `EventBudgetExceeded pubsub.EventType = "budget_exceeded"`.
- `internal/auto/budget_test.go` (new) — Unit tests for `DBBudgetChecker` with mock DB, and standalone budget logic tests.
- `internal/auto/engine_integration_test.go` — Add integration test: engine with mock budget checker returns costs that exceed ceiling → engine pauses after publishing `EventBudgetExceeded`, no dispatch occurs for the over-budget unit.

### Build Order

1. **SQLC query first** — Add the SQL query, run `sqlc generate`. This produces the Go method that the budget checker wraps. Verify with `go build ./internal/db/`.
2. **Budget checker + event** — Create `budget.go` with interface + implementation, add event constant. Unit test the checker logic.
3. **Engine wiring + integration tests** — Add budget checker to engine, wire into `step()`, update `NewEngine`. Integration test proving the budget gate pauses the loop.

### Verification Approach

- `sqlc generate` succeeds and `go build ./internal/db/` compiles.
- `go test ./internal/auto/ -run TestBudget -count=1 -v` — unit tests for budget checker.
- `go test ./internal/auto/ -run TestIntegration_Budget -count=1 -v` — integration test proving engine pauses when budget exceeded.
- `go vet ./internal/auto/ ./internal/db/` — clean.

## Constraints

- **CGO_ENABLED=0** — SQLite via `modernc.org/sqlite`. SQLC generates pure Go code. No issues.
- **SQLC generation** — Must run `sqlc generate` from project root after adding the query. The generated code goes to `internal/db/`. The worktree must have `sqlc` available or we manually write the generated code to match the pattern.
- **Engine constructor change** — Adding `budgetChecker` and `budgetCeiling` to `NewEngine` will break all existing callers (tests, init.go). Every `NewEngine` call site in tests must be updated. Following the S01 pattern of adding the new param to the constructor.
- **Budget ceiling of 0 means disabled** — A zero value for `BudgetCeiling` in config means no budget enforcement. The engine should skip the check entirely when ceiling is 0 or checker is nil.
- **Race condition is acceptable** — A dispatch that starts before budget is exceeded but finishes after can push total cost over the ceiling. The next dispatch will be blocked. This is documented as acceptable in M003 context (check before dispatch, not after).

## Common Pitfalls

- **SQLC `COALESCE` return type** — `COALESCE(SUM(cost), 0)` returns `float64` in sqlc for SQLite. Verify the generated type matches expectations. If sqlc infers `interface{}`, use a cast alias: `CAST(COALESCE(SUM(cost), 0) AS REAL)`.
- **Nil budget checker** — Engine must handle `budgetChecker == nil` gracefully (skip check). Don't require all callers to provide a checker — tests that don't care about budget should pass nil.
- **Test constructor drift** — S01's `newTestEngine` helper and all `NewEngine` calls in tests need updating. Do this carefully to avoid breaking existing tests.

## Sources

- `internal/db/sql/sessions.sql` — Existing session queries, pattern for new queries.
- `internal/auto/engine.go` — `step()` method, `NewEngine` constructor, verification gate pattern to follow.
- `internal/auto/verify.go` — `Verifier` interface pattern: similar interface + nil-check pattern for budget checker.
- `internal/config/config.go` — `AutoConfig.BudgetCeiling` field already defined.
