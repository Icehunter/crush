---
estimated_steps: 31
estimated_files: 2
skills_used: []
---

# T01: Implement WorktreeManager and worktree_mode config field

Build the WorktreeManager in `internal/auto/worktree.go` with Create, Merge, Remove, and Exists operations that shell out to git via `exec.CommandContext`. Add `worktree_mode` string field to the Options struct in `internal/config/config.go`. The manager handles the full lifecycle: creating a worktree at `.crush/worktrees/<MID>/` on branch `auto/<MID>`, squash-merging changes back to the integration branch, and removing the worktree + branch.

## Steps

1. Add `WorktreeMode string` field to the `Options` struct in `internal/config/config.go` with JSON tag `worktree_mode` and appropriate jsonschema description. Valid values: `""` (disabled, default) and `"per-milestone"`.
2. Create `internal/auto/worktree.go` with a `WorktreeManager` struct that holds `projectRoot string` (absolute path to project root).
3. Implement `NewWorktreeManager(projectRoot string) *WorktreeManager` constructor.
4. Implement `EnsureGit(ctx context.Context) error` — uses `exec.LookPath("git")` to verify git is available. Return a clear error if not found.
5. Implement `WorktreePath(milestoneID string) string` — returns `filepath.Join(projectRoot, ".crush", "worktrees", milestoneID)`.
6. Implement `BranchName(milestoneID string) string` — returns `"auto/" + milestoneID`.
7. Implement `Exists(milestoneID string) bool` — checks if the worktree directory exists on disk.
8. Implement `Create(ctx context.Context, milestoneID string) error` — runs `git worktree add <path> -b auto/<MID>` from projectRoot. If branch already exists, falls back to `git worktree add <path> auto/<MID>` (without `-b`). Creates parent `.crush/worktrees/` directory if needed.
9. Implement `Merge(ctx context.Context, milestoneID string) error` — from projectRoot, runs `git merge --squash auto/<MID>` then `git commit -m "auto: milestone <MID> completed"`. If merge fails (conflicts), return error with stderr. If no changes to merge, handle gracefully (git merge --squash returns 0 but commit may fail with 'nothing to commit').
10. Implement `Remove(ctx context.Context, milestoneID string) error` — runs `git worktree remove <path> --force` then `git branch -D auto/<MID>`. Tolerates missing worktree/branch (idempotent).
11. Add a private `runGit(ctx context.Context, args ...string) (string, error)` helper that executes git commands from projectRoot, captures stdout+stderr, and returns combined output on error.
12. Add `slog.Info`/`slog.Error` logging for each operation with milestone ID context.
13. Run `gofumpt -w internal/auto/worktree.go internal/config/config.go`.

## Must-Haves

- [ ] WorktreeManager struct with projectRoot field
- [ ] EnsureGit checks git binary availability
- [ ] Create shells out to `git worktree add` with branch creation
- [ ] Merge shells out to `git merge --squash` + `git commit`
- [ ] Remove shells out to `git worktree remove` + `git branch -D`
- [ ] Exists checks worktree directory on disk
- [ ] WorktreePath and BranchName are deterministic from milestoneID
- [ ] Options.WorktreeMode field with JSON tag `worktree_mode`
- [ ] All git commands use exec.CommandContext (not shell package)

## Verification

- `go build ./internal/auto/...` compiles without errors
- `go build ./internal/config/...` compiles without errors
- `go vet ./internal/auto/... ./internal/config/...` has no new warnings
- `grep -q 'WorktreeMode' internal/config/config.go` confirms config field exists
- `grep -q 'WorktreeManager' internal/auto/worktree.go` confirms manager exists

## Inputs

- `internal/config/config.go`
- `internal/auto/event.go`

## Expected Output

- `internal/auto/worktree.go`
- `internal/config/config.go`

## Verification

go build ./internal/auto/... && go build ./internal/config/... && go vet ./internal/auto/... ./internal/config/...
