# S02: Budget Ceiling

**Goal:** Engine checks cumulative child session costs before each dispatch and pauses when the configured budget ceiling is reached.
**Demo:** After this: Set auto.budget_ceiling to 0.50 in crush.json. Engine checks cumulative child session costs before each dispatch. When ceiling is reached, engine pauses and publishes EventBudgetExceeded. Tests prove budget enforcement with mock session costs.

## Tasks
- [x] **T01: Added SumChildSessionCosts sqlc query, BudgetChecker interface with DBBudgetChecker implementation, EventBudgetExceeded event, and 3 unit tests** — Add the SQLC query for summing child session costs, create the BudgetChecker interface and DBBudgetChecker implementation in internal/auto/budget.go, add EventBudgetExceeded to events.go, and write unit tests for the budget checker.

## Steps

1. Add `-- name: SumChildSessionCosts :one` query to `internal/db/sql/sessions.sql`: `SELECT CAST(COALESCE(SUM(cost), 0) AS REAL) AS total_cost FROM sessions WHERE parent_session_id = ?;`
2. Run `sqlc generate` from the project root. If `sqlc` is not on PATH, manually add the generated `SumChildSessionCosts` method to `internal/db/sessions.sql.go` and update `internal/db/querier.go` following the exact pattern of existing queries (matching the sqlc output format with prepared statements).
3. Verify: `go build ./internal/db/` compiles cleanly.
4. Create `internal/auto/budget.go` with:
   - `BudgetChecker` interface: `CheckBudget(ctx context.Context, parentSessionID string) (float64, error)`
   - `DBBudgetChecker` struct wrapping a querier interface (define a minimal `BudgetQuerier` interface with just the `SumChildSessionCosts` method to avoid importing the full db package)
   - Implementation: calls `SumChildSessionCosts`, returns the total cost
5. Add `EventBudgetExceeded pubsub.EventType = "budget_exceeded"` to `internal/auto/events.go`.
6. Create `internal/auto/budget_test.go` with unit tests:
   - `TestDBBudgetChecker_ReturnsCost`: mock querier returns 1.50, checker returns 1.50
   - `TestDBBudgetChecker_ZeroCost`: mock querier returns 0.0, checker returns 0.0
   - `TestDBBudgetChecker_QueryError`: mock querier returns error, checker propagates error
7. Verify: `go test ./internal/auto/ -run TestDBBudgetChecker -count=1 -v` passes, `go vet ./internal/auto/` clean.

## Must-Haves

- [ ] SumChildSessionCosts query added to sessions.sql and generated code compiles
- [ ] BudgetChecker interface and DBBudgetChecker implementation in budget.go
- [ ] EventBudgetExceeded constant in events.go
- [ ] 3+ unit tests for budget checker pass
- [ ] `go vet ./internal/auto/ ./internal/db/` clean

## Verification

- `go build ./internal/db/` — exit 0
- `go test ./internal/auto/ -run TestDBBudgetChecker -count=1 -v` — 3/3 pass
- `go vet ./internal/auto/ ./internal/db/` — clean

## Negative Tests

- **Query error**: TestDBBudgetChecker_QueryError verifies error propagation from DB layer
- **Zero cost**: TestDBBudgetChecker_ZeroCost verifies correct handling of no child sessions
  - Estimate: 45m
  - Files: internal/db/sql/sessions.sql, internal/db/sessions.sql.go, internal/db/querier.go, internal/auto/budget.go, internal/auto/budget_test.go, internal/auto/events.go
  - Verify: go build ./internal/db/ && go test ./internal/auto/ -run TestDBBudgetChecker -count=1 -v && go vet ./internal/auto/ ./internal/db/
- [x] **T02: Wired budget checker into engine step() with budget gate before dispatch, updated all 10 NewEngine call sites, and added 4 integration tests for budget enforcement** — Add budgetChecker and budgetCeiling fields to Engine, update NewEngine to accept them, wire the budget gate into step() before dispatch, update all existing NewEngine call sites, and add integration tests proving budget enforcement.

## Steps

1. In `internal/auto/engine.go`:
   - Add `budgetChecker BudgetChecker` and `budgetCeiling float64` fields to the `Engine` struct.
   - Update `NewEngine` signature: add `budgetChecker BudgetChecker` and `budgetCeiling float64` parameters after `verifier`. Update the constructor body to assign them.
   - In `step()`, add budget gate **after** `DeriveState` and **before** `CreateChildSession`:
     ```go
     if e.budgetCeiling > 0 && e.budgetChecker != nil {
         totalCost, err := e.budgetChecker.CheckBudget(ctx, parentSessionID)
         if err != nil {
             return fmt.Errorf("check budget: %w", err)
         }
         if totalCost >= e.budgetCeiling {
             e.logger.Info("Budget ceiling reached", "total_cost", totalCost, "ceiling", e.budgetCeiling)
             e.publish(EventBudgetExceeded, unit, nil, fmt.Sprintf("total cost $%.4f >= ceiling $%.4f", totalCost, e.budgetCeiling))
             e.mu.Lock()
             e.state = EnginePaused
             e.mu.Unlock()
             e.paused.Store(true)
             return nil
         }
     }
     ```
2. Update ALL existing `NewEngine` call sites in test files to add `nil, 0` for the two new parameters:
   - `internal/auto/engine_test.go` — `newTestEngine` helper and any direct `NewEngine` calls
   - `internal/auto/engine_integration_test.go` — all direct `NewEngine` calls
   - `internal/auto/engine_verify_integration_test.go` — all 3 `NewEngine` calls
3. Verify existing tests still pass: `go test ./internal/auto/ -count=1 -v`
4. Create integration tests in `internal/auto/engine_budget_integration_test.go`:
   - `TestIntegration_BudgetExceededPausesEngine`: mock budget checker returns cost >= ceiling → engine publishes EventBudgetExceeded, state is EnginePaused, dispatcher NOT called
   - `TestIntegration_BudgetUnderCeilingDispatches`: mock budget checker returns cost < ceiling → dispatch proceeds normally
   - `TestIntegration_BudgetZeroCeilingSkipsCheck`: budget ceiling is 0 → check is skipped entirely, dispatch proceeds
   - `TestIntegration_BudgetNilCheckerSkipsCheck`: budget checker is nil → check is skipped, dispatch proceeds
5. Verify: `go test ./internal/auto/ -run TestIntegration_Budget -count=1 -v` passes.
6. Final verification: `go test ./internal/auto/ -count=1` (all tests pass), `go vet ./internal/auto/` clean.

## Must-Haves

- [ ] Engine has budgetChecker and budgetCeiling fields
- [ ] NewEngine accepts budget checker and ceiling parameters
- [ ] step() checks budget before dispatch and pauses when exceeded
- [ ] All existing NewEngine call sites updated — zero compilation errors
- [ ] All existing tests pass unchanged
- [ ] 4 integration tests for budget enforcement pass
- [ ] `go vet ./internal/auto/` clean

## Verification

- `go test ./internal/auto/ -count=1` — ALL tests pass (existing + new)
- `go test ./internal/auto/ -run TestIntegration_Budget -count=1 -v` — 4/4 pass
- `go vet ./internal/auto/` — clean

## Observability Impact

- Signals added: EventBudgetExceeded event published via broker with total cost and ceiling in message
- How a future agent inspects this: subscribe to EventBudgetExceeded, check Engine.Status().State == EnginePaused
- Failure state exposed: budget exceeded is a clean pause, not an error — state is EnginePaused

## Failure Modes

| Dependency | On error | On timeout | On malformed response |
|------------|----------|-----------|----------------------|
| BudgetChecker.CheckBudget | Return error, fail step | N/A (in-process DB call) | N/A (typed return) |

## Negative Tests

- **Nil checker**: TestIntegration_BudgetNilCheckerSkipsCheck — engine works normally with nil budget checker
- **Zero ceiling**: TestIntegration_BudgetZeroCeilingSkipsCheck — 0 ceiling disables budget enforcement
- **Budget exceeded**: TestIntegration_BudgetExceededPausesEngine — engine pauses, no dispatch occurs
  - Estimate: 45m
  - Files: internal/auto/engine.go, internal/auto/engine_test.go, internal/auto/engine_integration_test.go, internal/auto/engine_verify_integration_test.go, internal/auto/engine_budget_integration_test.go
  - Verify: go test ./internal/auto/ -count=1 && go vet ./internal/auto/
