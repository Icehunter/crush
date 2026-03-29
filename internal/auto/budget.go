package auto

import (
	"context"
	"database/sql"
)

// BudgetChecker returns the cumulative cost of child sessions for a given
// parent session. The engine calls this before each dispatch to enforce the
// configured budget ceiling.
type BudgetChecker interface {
	CheckBudget(ctx context.Context, parentSessionID string) (float64, error)
}

// BudgetQuerier is the minimal database interface needed by DBBudgetChecker.
// It avoids importing the full db package and makes unit testing trivial.
type BudgetQuerier interface {
	SumChildSessionCosts(ctx context.Context, parentSessionID sql.NullString) (float64, error)
}

// DBBudgetChecker implements BudgetChecker by querying the sessions table.
type DBBudgetChecker struct {
	q BudgetQuerier
}

// NewDBBudgetChecker creates a DBBudgetChecker backed by the given querier.
func NewDBBudgetChecker(q BudgetQuerier) *DBBudgetChecker {
	return &DBBudgetChecker{q: q}
}

// CheckBudget returns the total cost accumulated across all child sessions
// of the given parent session.
func (b *DBBudgetChecker) CheckBudget(ctx context.Context, parentSessionID string) (float64, error) {
	return b.q.SumChildSessionCosts(ctx, sql.NullString{String: parentSessionID, Valid: true})
}
