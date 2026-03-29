# S02: Budget Ceiling — UAT

**Milestone:** M003
**Written:** 2026-03-28T05:18:18.811Z

# S02: Budget Ceiling — UAT

**Milestone:** M003
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: Budget ceiling is an engine-internal gate with no UI or external runtime dependencies. All behavior is provable through unit and integration tests against the engine's step() method.

## Preconditions

- Go toolchain available
- Working directory is the crush project root (or worktree)
- `go build ./internal/db/` succeeds (DB layer compiles)

## Smoke Test

Run `go test ./internal/auto/ -run TestIntegration_BudgetExceededPausesEngine -count=1 -v` — should show engine logging "Budget ceiling reached" and test passing.

## Test Cases

### 1. Budget checker returns correct cost aggregation

1. Run `go test ./internal/auto/ -run TestDBBudgetChecker_ReturnsCost -count=1 -v`
2. **Expected:** Mock querier returns 1.50, checker returns 1.50, test PASS

### 2. Zero cost handled correctly

1. Run `go test ./internal/auto/ -run TestDBBudgetChecker_ZeroCost -count=1 -v`
2. **Expected:** Mock querier returns 0.0, checker returns 0.0, test PASS

### 3. Query error propagated

1. Run `go test ./internal/auto/ -run TestDBBudgetChecker_QueryError -count=1 -v`
2. **Expected:** Mock querier returns error, checker returns same error, test PASS

### 4. Budget exceeded pauses engine

1. Run `go test ./internal/auto/ -run TestIntegration_BudgetExceededPausesEngine -count=1 -v`
2. **Expected:** Engine publishes EventBudgetExceeded with cost/ceiling message, engine state is EnginePaused, dispatcher NOT called, test PASS

### 5. Under-ceiling budget dispatches normally

1. Run `go test ./internal/auto/ -run TestIntegration_BudgetUnderCeilingDispatches -count=1 -v`
2. **Expected:** Budget check passes (cost < ceiling), dispatch proceeds, unit completes, test PASS

### 6. Zero ceiling disables budget enforcement

1. Run `go test ./internal/auto/ -run TestIntegration_BudgetZeroCeilingSkipsCheck -count=1 -v`
2. **Expected:** Budget checker never called, dispatch proceeds normally, test PASS

### 7. Nil checker disables budget enforcement

1. Run `go test ./internal/auto/ -run TestIntegration_BudgetNilCheckerSkipsCheck -count=1 -v`
2. **Expected:** No panic, dispatch proceeds normally, test PASS

## Edge Cases

### Budget exactly at ceiling

1. TestIntegration_BudgetExceededPausesEngine uses cost (0.75) >= ceiling (0.50)
2. **Expected:** Engine pauses — the gate uses `>=` comparison, so exact match also triggers pause

### Both nil checker and zero ceiling

1. TestIntegration_BudgetNilCheckerSkipsCheck passes nil checker with 0 ceiling
2. **Expected:** Guard clause `e.budgetCeiling > 0 && e.budgetChecker != nil` short-circuits, no panic

## Failure Signals

- `go test ./internal/auto/ -count=1` fails — regression in budget gate or broken NewEngine call sites
- `go build ./internal/db/` fails — SumChildSessionCosts query or generated code broken
- `go vet ./internal/auto/ ./internal/db/` reports issues — type safety or interface compliance broken

## Not Proven By This UAT

- Real dollar-cost accumulation from actual LLM API calls (requires live provider)
- crush.json config loading and wiring of budget_ceiling value into NewEngine (config layer not yet built)
- UI/TUI notification when budget is exceeded (deferred to M004)
- Cost tracking granularity below session level

## Notes for Tester

- All tests use mock implementations — no real DB or API calls needed
- The BudgetQuerier interface is intentionally narrow (single method) for easy mocking
- EventBudgetExceeded message format is "total cost $X.XXXX >= ceiling $Y.XXXX"
