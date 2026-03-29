package auto

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockBudgetQuerier implements BudgetQuerier for unit testing.
type mockBudgetQuerier struct {
	cost float64
	err  error
}

func (m *mockBudgetQuerier) SumChildSessionCosts(_ context.Context, _ sql.NullString) (float64, error) {
	return m.cost, m.err
}

func TestDBBudgetChecker_ReturnsCost(t *testing.T) {
	t.Parallel()

	q := &mockBudgetQuerier{cost: 1.50}
	checker := NewDBBudgetChecker(q)

	cost, err := checker.CheckBudget(context.Background(), "parent-1")
	require.NoError(t, err)
	require.InDelta(t, 1.50, cost, 0.001)
}

func TestDBBudgetChecker_ZeroCost(t *testing.T) {
	t.Parallel()

	q := &mockBudgetQuerier{cost: 0.0}
	checker := NewDBBudgetChecker(q)

	cost, err := checker.CheckBudget(context.Background(), "parent-2")
	require.NoError(t, err)
	require.InDelta(t, 0.0, cost, 0.001)
}

func TestDBBudgetChecker_QueryError(t *testing.T) {
	t.Parallel()

	q := &mockBudgetQuerier{err: errors.New("db connection lost")}
	checker := NewDBBudgetChecker(q)

	_, err := checker.CheckBudget(context.Background(), "parent-3")
	require.Error(t, err)
	require.EqualError(t, err, "db connection lost")
}
