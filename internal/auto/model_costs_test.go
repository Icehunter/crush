package auto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupModelCost_Known(t *testing.T) {
	t.Parallel()
	c := LookupModelCost("claude-opus-4-6")
	require.Equal(t, 15.0, c.PromptPerMToken)
	require.Equal(t, 75.0, c.CompletionPerMToken)
}

func TestLookupModelCost_Unknown(t *testing.T) {
	t.Parallel()
	c := LookupModelCost("totally-unknown-model")
	require.Equal(t, float64(0), c.PromptPerMToken)
	require.Equal(t, float64(0), c.CompletionPerMToken)
}

func TestModelCost_EstimateCost(t *testing.T) {
	t.Parallel()
	c := ModelCost{PromptPerMToken: 3.0, CompletionPerMToken: 15.0}

	// 1000 prompt tokens + 500 completion tokens
	cost := c.EstimateCost(1000, 500)
	expected := (1000.0/1_000_000)*3.0 + (500.0/1_000_000)*15.0
	require.InDelta(t, expected, cost, 0.0000001)
}

func TestModelCost_EstimateCost_Zero(t *testing.T) {
	t.Parallel()
	c := ModelCost{}
	require.Equal(t, float64(0), c.EstimateCost(1000, 500))
}
