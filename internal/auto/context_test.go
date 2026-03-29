package auto

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockTokenQuerier implements TokenQuerier for testing.
type mockTokenQuerier struct {
	prompt     int64
	completion int64
	err        error
}

func (m *mockTokenQuerier) GetTokenUsage(_ context.Context, _ string) (int64, int64, error) {
	return m.prompt, m.completion, m.err
}

func TestContextMonitor_BelowThreshold(t *testing.T) {
	t.Parallel()
	q := &mockTokenQuerier{prompt: 500, completion: 200}
	cm := NewContextMonitor(0.8, 10000, q)
	require.NotNil(t, cm)

	exceeded, err := cm.Check(context.Background(), "sess-1")
	require.NoError(t, err)
	require.False(t, exceeded, "700/10000 = 0.07, below 0.8 threshold")
}

func TestContextMonitor_AtThreshold(t *testing.T) {
	t.Parallel()
	q := &mockTokenQuerier{prompt: 6000, completion: 2000}
	cm := NewContextMonitor(0.8, 10000, q)
	require.NotNil(t, cm)

	exceeded, err := cm.Check(context.Background(), "sess-1")
	require.NoError(t, err)
	require.True(t, exceeded, "8000/10000 = 0.8, at threshold")
}

func TestContextMonitor_AboveThreshold(t *testing.T) {
	t.Parallel()
	q := &mockTokenQuerier{prompt: 7000, completion: 3000}
	cm := NewContextMonitor(0.8, 10000, q)
	require.NotNil(t, cm)

	exceeded, err := cm.Check(context.Background(), "sess-1")
	require.NoError(t, err)
	require.True(t, exceeded, "10000/10000 = 1.0, above 0.8 threshold")
}

func TestContextMonitor_ZeroContextWindow(t *testing.T) {
	t.Parallel()
	q := &mockTokenQuerier{prompt: 100, completion: 100}
	cm := NewContextMonitor(0.8, 0, q)
	require.Nil(t, cm, "zero context window should disable monitor")
}

func TestContextMonitor_NilQuerier(t *testing.T) {
	t.Parallel()
	cm := NewContextMonitor(0.8, 10000, nil)
	require.Nil(t, cm, "nil querier should disable monitor")
}

func TestContextMonitor_QuerierError(t *testing.T) {
	t.Parallel()
	q := &mockTokenQuerier{err: errors.New("db failure")}
	cm := NewContextMonitor(0.8, 10000, q)
	require.NotNil(t, cm)

	exceeded, err := cm.Check(context.Background(), "sess-1")
	require.Error(t, err)
	require.False(t, exceeded)
	require.EqualError(t, err, "db failure")
}
