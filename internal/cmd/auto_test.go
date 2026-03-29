package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAutoCmd_SubcommandTree(t *testing.T) {
	t.Parallel()

	subs := autoCmd.Commands()
	names := make([]string, 0, len(subs))
	for _, c := range subs {
		names = append(names, c.Name())
	}

	require.Len(t, subs, 4)
	require.ElementsMatch(t, []string{"start", "pause", "stop", "status"}, names)
}

func TestAutoCmd_IsRegistered(t *testing.T) {
	t.Parallel()

	found, _, err := rootCmd.Find([]string{"auto"})
	require.NoError(t, err)
	require.Equal(t, "auto", found.Name())
}

func TestNextCmd_IsRegistered(t *testing.T) {
	t.Parallel()

	found, _, err := rootCmd.Find([]string{"next"})
	require.NoError(t, err)
	require.Equal(t, "next", found.Name())
}

func TestAutoStartCmd_RequiresArg(t *testing.T) {
	t.Parallel()

	// ExactArgs(1) should reject zero args.
	err := autoStartCmd.Args(autoStartCmd, []string{})
	require.Error(t, err)

	// And accept exactly one.
	err = autoStartCmd.Args(autoStartCmd, []string{"M001"})
	require.NoError(t, err)
}

func TestNextCmd_RequiresArg(t *testing.T) {
	t.Parallel()

	err := nextCmd.Args(nextCmd, []string{})
	require.Error(t, err)

	err = nextCmd.Args(nextCmd, []string{"M001"})
	require.NoError(t, err)
}

func TestAutoStatusCmd_HasJSONFlag(t *testing.T) {
	t.Parallel()

	f := autoStatusCmd.Flags().Lookup("json")
	require.NotNil(t, f, "--json flag should be registered on auto status")
	require.Equal(t, "false", f.DefValue)
}
