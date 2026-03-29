---
estimated_steps: 59
estimated_files: 5
skills_used: []
---

# T02: Wire budget checker into engine step() and add integration tests

Add budgetChecker and budgetCeiling fields to Engine, update NewEngine to accept them, wire the budget gate into step() before dispatch, update all existing NewEngine call sites, and add integration tests proving budget enforcement.

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

## Inputs

- `internal/auto/engine.go`
- `internal/auto/budget.go`
- `internal/auto/events.go`
- `internal/auto/engine_test.go`
- `internal/auto/engine_integration_test.go`
- `internal/auto/engine_verify_integration_test.go`

## Expected Output

- `internal/auto/engine.go`
- `internal/auto/engine_test.go`
- `internal/auto/engine_integration_test.go`
- `internal/auto/engine_verify_integration_test.go`
- `internal/auto/engine_budget_integration_test.go`

## Verification

go test ./internal/auto/ -count=1 && go vet ./internal/auto/
