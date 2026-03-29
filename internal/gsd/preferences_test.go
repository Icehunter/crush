package gsd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoadPreferences_GlobalOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	globalPath := filepath.Join(dir, "global-PREFERENCES.md")
	err := os.WriteFile(globalPath, []byte(`---
version: 1
mode: auto
budget_ceiling: "$5.00"
git:
  auto_push: true
  remote: origin
  main_branch: main
unique_milestone_ids: true
---

# Global Preferences
`), 0o644)
	require.NoError(t, err)

	prefs, err := LoadPreferences(globalPath, "")
	require.NoError(t, err)
	require.Equal(t, 1, prefs.Version)
	require.Equal(t, "auto", prefs.Mode)
	require.Equal(t, "$5.00", prefs.BudgetCeiling)
	require.NotNil(t, prefs.Git.AutoPush)
	require.True(t, *prefs.Git.AutoPush)
	require.Equal(t, "origin", prefs.Git.Remote)
	require.Equal(t, "main", prefs.Git.MainBranch)
	require.NotNil(t, prefs.UniqueMilestoneIDs)
	require.True(t, *prefs.UniqueMilestoneIDs)
}

func TestLoadPreferences_ProjectOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	projectPath := filepath.Join(dir, "project-PREFERENCES.md")
	err := os.WriteFile(projectPath, []byte(`---
version: 2
mode: plan
token_profile: large
verification_commands:
  - "go test ./..."
  - "go vet ./..."
phases:
  skip_research: true
notifications:
  enabled: true
  on_error: true
---

# Project Preferences
`), 0o644)
	require.NoError(t, err)

	prefs, err := LoadPreferences("", projectPath)
	require.NoError(t, err)
	require.Equal(t, 2, prefs.Version)
	require.Equal(t, "plan", prefs.Mode)
	require.Equal(t, "large", prefs.TokenProfile)
	require.Equal(t, []string{"go test ./...", "go vet ./..."}, prefs.VerificationCommands)
	require.NotNil(t, prefs.Phases.SkipResearch)
	require.True(t, *prefs.Phases.SkipResearch)
	require.NotNil(t, prefs.Notifications.Enabled)
	require.True(t, *prefs.Notifications.Enabled)
	require.NotNil(t, prefs.Notifications.OnError)
	require.True(t, *prefs.Notifications.OnError)
}

func TestLoadPreferences_MergeBehavior(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	globalPath := filepath.Join(dir, "global.md")
	err := os.WriteFile(globalPath, []byte(`---
version: 1
mode: auto
budget_ceiling: "$5.00"
git:
  auto_push: true
  remote: origin
  main_branch: main
parallel:
  enabled: true
  max_workers: 4
verification_commands:
  - "make test"
---
`), 0o644)
	require.NoError(t, err)

	projectPath := filepath.Join(dir, "project.md")
	err = os.WriteFile(projectPath, []byte(`---
version: 2
mode: plan
git:
  main_branch: develop
  snapshots: false
verification_commands:
  - "go test ./..."
---
`), 0o644)
	require.NoError(t, err)

	prefs, err := LoadPreferences(globalPath, projectPath)
	require.NoError(t, err)

	// Project overrides.
	require.Equal(t, 2, prefs.Version)
	require.Equal(t, "plan", prefs.Mode)
	require.Equal(t, []string{"go test ./..."}, prefs.VerificationCommands)

	// Global values preserved when project doesn't set them.
	require.Equal(t, "$5.00", prefs.BudgetCeiling)
	require.NotNil(t, prefs.Git.AutoPush)
	require.True(t, *prefs.Git.AutoPush)
	require.Equal(t, "origin", prefs.Git.Remote)

	// Nested struct field override.
	require.Equal(t, "develop", prefs.Git.MainBranch)
	require.NotNil(t, prefs.Git.Snapshots)
	require.False(t, *prefs.Git.Snapshots)

	// Global parallel preserved.
	require.NotNil(t, prefs.Parallel.Enabled)
	require.True(t, *prefs.Parallel.Enabled)
	require.Equal(t, 4, prefs.Parallel.MaxWorkers)
}

func TestLoadPreferences_MissingFiles(t *testing.T) {
	t.Parallel()

	prefs, err := LoadPreferences("/nonexistent/global.md", "/nonexistent/project.md")
	require.NoError(t, err)
	require.NotNil(t, prefs)
	require.Equal(t, 0, prefs.Version)
	require.Equal(t, "", prefs.Mode)
}

func TestLoadPreferences_BothEmpty(t *testing.T) {
	t.Parallel()

	prefs, err := LoadPreferences("", "")
	require.NoError(t, err)
	require.NotNil(t, prefs)
}

func TestLoadPreferences_MalformedYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	badFile := filepath.Join(dir, "bad.md")
	err := os.WriteFile(badFile, []byte(`---
version: [[[not valid yaml
  broken: {{{
---
`), 0o644)
	require.NoError(t, err)

	_, err = LoadPreferences(badFile, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "global preferences")

	_, err = LoadPreferences("", badFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "project preferences")
}

func TestLoadPreferences_NoFrontmatter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	noFM := filepath.Join(dir, "nofm.md")
	err := os.WriteFile(noFM, []byte("# Just a heading\nSome content.\n"), 0o644)
	require.NoError(t, err)

	// File without frontmatter is treated as empty preferences, not an error.
	prefs, err := LoadPreferences(noFM, "")
	require.NoError(t, err)
	require.NotNil(t, prefs)
	require.Equal(t, 0, prefs.Version)
}

func TestSplitFrontmatter(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		fm, body, err := splitFrontmatter("---\nkey: value\n---\nbody text")
		require.NoError(t, err)
		require.Equal(t, "key: value", fm)
		require.Equal(t, "\nbody text", body)
	})

	t.Run("no frontmatter", func(t *testing.T) {
		t.Parallel()
		_, _, err := splitFrontmatter("no frontmatter here")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no YAML frontmatter found")
	})

	t.Run("unclosed", func(t *testing.T) {
		t.Parallel()
		_, _, err := splitFrontmatter("---\nkey: value\n")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unclosed frontmatter")
	})

	t.Run("windows line endings", func(t *testing.T) {
		t.Parallel()
		fm, _, err := splitFrontmatter("---\r\nkey: value\r\n---\r\nbody")
		require.NoError(t, err)
		require.Equal(t, "key: value", fm)
	})
}

func TestLoadPreferences_DynamicRouting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	path := filepath.Join(dir, "prefs.md")
	err := os.WriteFile(path, []byte(`---
dynamic_routing:
  enabled: true
  tier_models:
    scout: "claude-3-haiku"
    worker: "claude-sonnet-4-20250514"
  escalate_on_failure: true
  budget_pressure: conservative
---
`), 0o644)
	require.NoError(t, err)

	prefs, err := LoadPreferences(path, "")
	require.NoError(t, err)
	require.NotNil(t, prefs.DynamicRouting.Enabled)
	require.True(t, *prefs.DynamicRouting.Enabled)
	require.Equal(t, "claude-3-haiku", prefs.DynamicRouting.TierModels["scout"])
	require.Equal(t, "claude-sonnet-4-20250514", prefs.DynamicRouting.TierModels["worker"])
	require.NotNil(t, prefs.DynamicRouting.EscalateOnFailure)
	require.True(t, *prefs.DynamicRouting.EscalateOnFailure)
	require.Equal(t, "conservative", prefs.DynamicRouting.BudgetPressure)
}

func TestApplyToAutoConfig_AllFields(t *testing.T) {
	t.Parallel()

	prefs := &Preferences{
		BudgetCeiling:        "$5.00",
		VerificationCommands: []string{"go test ./...", "go vet ./..."},
		Git: GitPreferences{
			Isolation: "worktree",
		},
	}

	ac := &config.AutoConfig{}
	prefs.ApplyToAutoConfig(ac)

	require.Equal(t, 5.0, ac.BudgetCeiling)
	require.Equal(t, []string{"go test ./...", "go vet ./..."}, ac.VerificationCommands)
	require.Equal(t, "worktree", ac.WorktreeMode)
}

func TestApplyToAutoConfig_PartialOverride(t *testing.T) {
	t.Parallel()

	prefs := &Preferences{
		BudgetCeiling: "$10.50",
		// VerificationCommands and Git.Isolation left empty.
	}

	ac := &config.AutoConfig{
		VerificationCommands: []string{"existing"},
		WorktreeMode:         "none",
		BudgetCeiling:        1.0,
	}
	prefs.ApplyToAutoConfig(ac)

	require.Equal(t, 10.5, ac.BudgetCeiling)
	require.Equal(t, []string{"existing"}, ac.VerificationCommands, "should not override empty slice")
	require.Equal(t, "none", ac.WorktreeMode, "should not override empty string")
}

func TestApplyToAutoConfig_NilAutoConfig(t *testing.T) {
	t.Parallel()

	prefs := &Preferences{BudgetCeiling: "$5.00"}
	// Should not panic.
	prefs.ApplyToAutoConfig(nil)
}

func TestApplyToAutoConfig_InvalidBudget(t *testing.T) {
	t.Parallel()

	prefs := &Preferences{BudgetCeiling: "not-a-number"}
	ac := &config.AutoConfig{BudgetCeiling: 2.0}
	prefs.ApplyToAutoConfig(ac)

	// Invalid budget string should leave existing value untouched.
	require.Equal(t, 2.0, ac.BudgetCeiling)
}

func TestApplyToAutoConfig_BudgetWithoutDollarSign(t *testing.T) {
	t.Parallel()

	prefs := &Preferences{BudgetCeiling: "7.50"}
	ac := &config.AutoConfig{}
	prefs.ApplyToAutoConfig(ac)

	require.Equal(t, 7.5, ac.BudgetCeiling)
}

func TestParseBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  float64
		ok    bool
	}{
		{"$5.00", 5.0, true},
		{"5.00", 5.0, true},
		{"$0.50", 0.5, true},
		{"  $10  ", 10.0, true},
		{"invalid", 0, false},
		{"$", 0, false},
	}

	for _, tt := range tests {
		v, err := parseBudget(tt.input)
		if tt.ok {
			require.NoError(t, err, "input: %q", tt.input)
			require.Equal(t, tt.want, v, "input: %q", tt.input)
		} else {
			require.Error(t, err, "input: %q", tt.input)
		}
	}
}
