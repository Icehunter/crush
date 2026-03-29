---
estimated_steps: 28
estimated_files: 6
skills_used: []
---

# T01: Add SumChildSessionCosts query, BudgetChecker interface, and unit tests

Add the SQLC query for summing child session costs, create the BudgetChecker interface and DBBudgetChecker implementation in internal/auto/budget.go, add EventBudgetExceeded to events.go, and write unit tests for the budget checker.

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

## Inputs

- `internal/db/sql/sessions.sql`
- `internal/db/querier.go`
- `internal/db/sessions.sql.go`
- `internal/auto/events.go`
- `internal/auto/verify.go`

## Expected Output

- `internal/db/sql/sessions.sql`
- `internal/db/sessions.sql.go`
- `internal/db/querier.go`
- `internal/auto/budget.go`
- `internal/auto/budget_test.go`
- `internal/auto/events.go`

## Verification

go build ./internal/db/ && go test ./internal/auto/ -run TestDBBudgetChecker -count=1 -v && go vet ./internal/auto/ ./internal/db/
