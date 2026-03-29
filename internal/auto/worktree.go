package auto

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager manages git worktrees for milestone isolation in
// auto-mode. Each milestone gets its own worktree at
// .crush/worktrees/<MID>/ on a branch named auto/<MID>.
type WorktreeManager struct {
	projectRoot string
}

// NewWorktreeManager creates a WorktreeManager rooted at the given
// absolute project path.
func NewWorktreeManager(projectRoot string) *WorktreeManager {
	return &WorktreeManager{projectRoot: projectRoot}
}

// EnsureGit verifies that the git binary is available on PATH.
func (w *WorktreeManager) EnsureGit(_ context.Context) error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git binary not found on PATH: %w", err)
	}
	return nil
}

// WorktreePath returns the absolute path where the worktree for
// milestoneID will be placed.
func (w *WorktreeManager) WorktreePath(milestoneID string) string {
	return filepath.Join(w.projectRoot, ".crush", "worktrees", milestoneID)
}

// BranchName returns the git branch name used for the given milestone.
func (w *WorktreeManager) BranchName(milestoneID string) string {
	return "auto/" + milestoneID
}

// Exists reports whether the worktree directory for milestoneID exists
// on disk.
func (w *WorktreeManager) Exists(milestoneID string) bool {
	info, err := os.Stat(w.WorktreePath(milestoneID))
	return err == nil && info.IsDir()
}

// Create adds a git worktree for the given milestone. It creates the
// parent directory if needed, then runs `git worktree add` with a new
// branch. If the branch already exists, it falls back to attaching the
// existing branch.
func (w *WorktreeManager) Create(ctx context.Context, milestoneID string) error {
	slog.Info("Creating worktree", "milestone", milestoneID)

	wtPath := w.WorktreePath(milestoneID)
	branch := w.BranchName(milestoneID)

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(wtPath), 0o755); err != nil {
		return fmt.Errorf("creating worktree parent directory: %w", err)
	}

	// Try creating with a new branch first.
	_, err := w.runGit(ctx, "worktree", "add", wtPath, "-b", branch)
	if err != nil {
		// Branch may already exist — fall back to attaching it.
		slog.Info("Branch may already exist, retrying without -b", "branch", branch)
		if _, err2 := w.runGit(ctx, "worktree", "add", wtPath, branch); err2 != nil {
			return fmt.Errorf("creating worktree for milestone %s: %w", milestoneID, err2)
		}
	}

	slog.Info("Worktree created", "milestone", milestoneID, "path", wtPath)
	return nil
}

// Merge squash-merges the milestone branch back into the current branch
// and commits the result. If there are no changes to merge, it returns
// nil without error.
func (w *WorktreeManager) Merge(ctx context.Context, milestoneID string) error {
	slog.Info("Merging worktree", "milestone", milestoneID)

	branch := w.BranchName(milestoneID)

	if _, err := w.runGit(ctx, "merge", "--squash", branch); err != nil {
		return fmt.Errorf("squash-merging milestone %s: %w", milestoneID, err)
	}

	// Commit the squash-merge. If there is nothing to commit (no
	// changes on the milestone branch), git commit exits non-zero.
	// Treat that as a no-op, not an error.
	_, err := w.runGit(ctx, "commit", "-m", fmt.Sprintf("auto: milestone %s completed", milestoneID))
	if err != nil {
		// Check whether the failure is simply "nothing to commit".
		if strings.Contains(err.Error(), "nothing to commit") ||
			strings.Contains(err.Error(), "nothing added to commit") {
			slog.Info("No changes to merge", "milestone", milestoneID)
			return nil
		}
		return fmt.Errorf("committing squash-merge for milestone %s: %w", milestoneID, err)
	}

	slog.Info("Worktree merged", "milestone", milestoneID)
	return nil
}

// Remove forcefully removes the worktree and deletes its branch.
// The operation is idempotent — missing worktrees or branches are
// tolerated.
func (w *WorktreeManager) Remove(ctx context.Context, milestoneID string) error {
	slog.Info("Removing worktree", "milestone", milestoneID)

	wtPath := w.WorktreePath(milestoneID)
	branch := w.BranchName(milestoneID)

	// Remove the worktree; tolerate "not a worktree" errors.
	if _, err := w.runGit(ctx, "worktree", "remove", wtPath, "--force"); err != nil {
		if !isNotFoundError(err) {
			slog.Error("Failed to remove worktree", "milestone", milestoneID, "error", err)
		}
	}

	// Delete the branch; tolerate "not found" errors.
	if _, err := w.runGit(ctx, "branch", "-D", branch); err != nil {
		if !isNotFoundError(err) {
			slog.Error("Failed to delete branch", "milestone", milestoneID, "error", err)
		}
	}

	slog.Info("Worktree removed", "milestone", milestoneID)
	return nil
}

// runGit executes a git command from projectRoot and returns its
// combined stdout. On failure the returned error includes stderr.
func (w *WorktreeManager) runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		combined := strings.TrimSpace(stderr.String() + "\n" + stdout.String())
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, combined)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// isNotFoundError returns true when the error message suggests the
// target (worktree or branch) does not exist.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not a working tree") ||
		strings.Contains(msg, "is not a valid") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "No such file or directory") ||
		errors.Is(err, os.ErrNotExist)
}
