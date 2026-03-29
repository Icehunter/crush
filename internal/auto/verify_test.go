package auto

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShellVerifier_AllPass(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	v := NewShellVerifier([]string{"echo hello", "echo world"}, nil)
	results, err := v.RunVerification(context.Background(), dir)
	require.NoError(t, err)
	require.Len(t, results, 2)

	for i, r := range results {
		require.True(t, r.Passed, "command %d should pass", i)
		require.Equal(t, 0, r.ExitCode, "command %d exit code", i)
		require.NotZero(t, r.Duration, "command %d duration", i)
	}
	require.Contains(t, results[0].Stdout, "hello")
	require.Contains(t, results[1].Stdout, "world")
}

func TestShellVerifier_FirstFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	v := NewShellVerifier([]string{"false", "echo should-not-run"}, nil)
	results, err := v.RunVerification(context.Background(), dir)
	require.NoError(t, err, "system error should be nil; failure is in results")
	require.Len(t, results, 1, "should short-circuit after first failure")
	require.False(t, results[0].Passed)
	require.Equal(t, 1, results[0].ExitCode)
}

func TestShellVerifier_EmptyCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	v := NewShellVerifier(nil, nil)
	results, err := v.RunVerification(context.Background(), dir)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestShellVerifier_NonexistentCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	v := NewShellVerifier([]string{"this_command_does_not_exist_xyzzy"}, nil)
	results, err := v.RunVerification(context.Background(), dir)
	require.NoError(t, err, "system error should be nil")
	require.Len(t, results, 1)
	require.False(t, results[0].Passed)
}

func TestShellVerifier_ContextCancelled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	v := NewShellVerifier([]string{"echo hello"}, nil)
	results, err := v.RunVerification(ctx, dir)
	require.Error(t, err, "should propagate context cancellation")
	require.Empty(t, results)
}

func TestVerifier_TruncateOutput(t *testing.T) {
	t.Parallel()

	t.Run("short string unchanged", func(t *testing.T) {
		t.Parallel()
		out := truncateOutput("hello", 100)
		require.Equal(t, "hello", out)
	})

	t.Run("exact length unchanged", func(t *testing.T) {
		t.Parallel()
		s := strings.Repeat("a", 100)
		out := truncateOutput(s, 100)
		require.Equal(t, s, out)
	})

	t.Run("long string truncated to tail", func(t *testing.T) {
		t.Parallel()
		s := strings.Repeat("a", 200)
		out := truncateOutput(s, 50)
		require.Len(t, out, len(out)) // Sanity.
		require.Contains(t, out, "truncated 150 bytes")
		// The tail should be the last 50 'a' characters.
		require.True(t, strings.HasSuffix(out, strings.Repeat("a", 50)))
	})

	t.Run("zero maxLen returns original", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "abc", truncateOutput("abc", 0))
	})
}

func TestFormatFailureDiagnostic(t *testing.T) {
	t.Parallel()

	results := []VerificationResult{
		{Command: "go vet ./...", ExitCode: 0, Passed: true},
		{Command: "go test ./...", ExitCode: 1, Passed: false, Stderr: "FAIL pkg"},
	}
	diag := FormatFailureDiagnostic(results)
	require.Contains(t, diag, "go test ./...")
	require.Contains(t, diag, "FAIL pkg")
	require.NotContains(t, diag, "go vet", "passed commands should be excluded")
}
