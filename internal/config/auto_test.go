package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAutoConfig(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"auto": {
			"verification_commands": ["go build ./...", "go test ./..."],
			"budget_ceiling": 5.0,
			"stuck_threshold": 3,
			"worktree_mode": "per-slice"
		}
	}`)

	cfg, err := loadFromBytes([][]byte{raw})
	require.NoError(t, err)
	require.NotNil(t, cfg.Auto)
	require.Equal(t, []string{"go build ./...", "go test ./..."}, cfg.Auto.VerificationCommands)
	require.Equal(t, 5.0, cfg.Auto.BudgetCeiling)
	require.Equal(t, 3, cfg.Auto.StuckThreshold)
	require.Equal(t, "per-slice", cfg.Auto.WorktreeMode)
}

func TestAutoConfig_Empty(t *testing.T) {
	t.Parallel()

	raw := []byte(`{}`)

	cfg, err := loadFromBytes([][]byte{raw})
	require.NoError(t, err)
	require.Nil(t, cfg.Auto)
}

func TestAutoConfig_PartialFields(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"auto": {
			"verification_commands": ["make lint"]
		}
	}`)

	cfg, err := loadFromBytes([][]byte{raw})
	require.NoError(t, err)
	require.NotNil(t, cfg.Auto)
	require.Equal(t, []string{"make lint"}, cfg.Auto.VerificationCommands)
	require.Equal(t, 0.0, cfg.Auto.BudgetCeiling)
	require.Equal(t, 0, cfg.Auto.StuckThreshold)
	require.Empty(t, cfg.Auto.WorktreeMode)
}
