package auto

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupGitRepo creates a temporary git repository with an initial commit.
// It returns the absolute path to the repo root.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	ctx := context.Background()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "config", "commit.gpgSign", "false"},
	}
	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "setup %v: %s", args, out)
	}

	// Create an initial file and commit it.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644))

	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "setup %v: %s", args, out)
	}

	return dir
}

// gitRun is a small helper that runs a git command in a given dir and
// returns its trimmed stdout.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func TestWorktreeManager_WorktreePath(t *testing.T) {
	t.Parallel()

	m := NewWorktreeManager("/some/project")
	got := m.WorktreePath("M001")
	require.Equal(t, filepath.Join("/some/project", ".crush", "worktrees", "M001"), got)
}

func TestWorktreeManager_BranchName(t *testing.T) {
	t.Parallel()

	m := NewWorktreeManager("/some/project")
	require.Equal(t, "auto/M001", m.BranchName("M001"))
	require.Equal(t, "auto/M999", m.BranchName("M999"))
}

func TestWorktreeManager_EnsureGit(t *testing.T) {
	t.Parallel()

	m := NewWorktreeManager(t.TempDir())
	err := m.EnsureGit(context.Background())
	require.NoError(t, err, "git should be available in test environment")
}

func TestWorktreeManager_EnsureGit_MissingPath(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv().
	t.Setenv("PATH", "")

	m := NewWorktreeManager(t.TempDir())
	err := m.EnsureGit(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "git binary not found")
}

func TestWorktreeManager_Exists_False(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	m := NewWorktreeManager(repo)
	require.False(t, m.Exists("nonexistent"))
}

func TestWorktreeManager_Create(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	m := NewWorktreeManager(repo)
	ctx := context.Background()

	err := m.Create(ctx, "M001")
	require.NoError(t, err)

	// Worktree directory should exist.
	require.True(t, m.Exists("M001"))

	// The branch auto/M001 should be listed.
	branches := gitRun(t, repo, "branch", "--list", "auto/M001")
	require.Contains(t, branches, "auto/M001")
}

func TestWorktreeManager_Create_BranchExists(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	ctx := context.Background()

	// Pre-create the branch manually.
	gitRun(t, repo, "branch", "auto/M002")

	m := NewWorktreeManager(repo)
	err := m.Create(ctx, "M002")
	require.NoError(t, err)
	require.True(t, m.Exists("M002"))
}

func TestWorktreeManager_Remove_Nonexistent(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	m := NewWorktreeManager(repo)
	ctx := context.Background()

	// Removing a worktree that was never created should not error
	// (idempotent).
	err := m.Remove(ctx, "M999")
	require.NoError(t, err)
}

func TestWorktreeManager_Merge_NoChanges(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	m := NewWorktreeManager(repo)
	ctx := context.Background()

	// Create a worktree but don't make any changes in it.
	require.NoError(t, m.Create(ctx, "M003"))

	// Merging with no divergent changes should succeed without error.
	err := m.Merge(ctx, "M003")
	require.NoError(t, err)
}

func TestWorktreeManager_FullLifecycle(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)
	m := NewWorktreeManager(repo)
	ctx := context.Background()

	const mid = "M010"

	// 1. Create worktree.
	require.NoError(t, m.Create(ctx, mid))
	require.True(t, m.Exists(mid))

	wtPath := m.WorktreePath(mid)

	// 2. Make a change inside the worktree and commit it.
	newFile := filepath.Join(wtPath, "feature.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("hello from worktree\n"), 0o644))

	gitRun(t, wtPath, "add", ".")
	gitRun(t, wtPath, "commit", "-m", "add feature")

	// Verify the file is NOT on the main branch yet.
	_, err := os.Stat(filepath.Join(repo, "feature.txt"))
	require.True(t, os.IsNotExist(err), "feature.txt should not exist on main before merge")

	// 3. Merge back.
	require.NoError(t, m.Merge(ctx, mid))

	// Verify the file now exists on the main branch.
	data, err := os.ReadFile(filepath.Join(repo, "feature.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello from worktree\n", string(data))

	// Verify the squash-merge commit message.
	log := gitRun(t, repo, "log", "-1", "--format=%s")
	require.Equal(t, "auto: milestone M010 completed", log)

	// 4. Remove.
	require.NoError(t, m.Remove(ctx, mid))
	require.False(t, m.Exists(mid))

	// Branch should be deleted.
	branches := gitRun(t, repo, "branch", "--list", "auto/"+mid)
	require.Empty(t, branches)
}
