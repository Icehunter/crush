---
estimated_steps: 36
estimated_files: 1
skills_used: []
---

# T02: Add comprehensive WorktreeManager tests with real git repos

Write `internal/auto/worktree_test.go` exercising the full worktree lifecycle and error paths. Tests use `t.TempDir()` to create real git repositories (via `git init`, `git add`, `git commit`), then exercise Create → commit in worktree → Merge → Remove. Also test error cases: missing git, branch already exists, no changes to merge, remove nonexistent worktree.

## Steps

1. Create `internal/auto/worktree_test.go`.
2. Write a `setupGitRepo(t *testing.T) string` test helper that: creates a temp dir, runs `git init`, `git config user.email/name`, creates an initial file, `git add .`, `git commit -m "initial"`. Returns the repo path. Use `exec.CommandContext` directly for setup.
3. Write `TestWorktreeManager_Create` — creates a manager, calls Create with a milestone ID, asserts: worktree directory exists, branch `auto/<MID>` exists (via `git branch --list`), Exists() returns true.
4. Write `TestWorktreeManager_FullLifecycle` — create worktree → create a file in the worktree and commit it → Merge back → verify the file appears in the main branch → Remove → verify worktree dir gone and branch deleted.
5. Write `TestWorktreeManager_Create_BranchExists` — create a branch `auto/<MID>` manually, then call Create. Should succeed by reusing the existing branch (without `-b` flag).
6. Write `TestWorktreeManager_Remove_Nonexistent` — call Remove on a milestone with no worktree. Should not error (idempotent).
7. Write `TestWorktreeManager_Merge_NoChanges` — create worktree, don't make any changes, call Merge. Should handle gracefully (no error or clear informational error).
8. Write `TestWorktreeManager_EnsureGit` — verify EnsureGit returns nil when git is available. (Testing missing git requires PATH manipulation — include if straightforward with `t.Setenv`).
9. Write `TestWorktreeManager_WorktreePath` and `TestWorktreeManager_BranchName` — unit tests for deterministic path/branch generation.
10. Write `TestWorktreeManager_Exists_False` — Exists returns false for a nonexistent milestone.
11. All tests use `t.Parallel()` where safe. Use `require` from testify.
12. Run `gofumpt -w internal/auto/worktree_test.go`.
13. Run `go test ./internal/auto/... -run TestWorktree -v` and ensure all pass.

## Failure Modes

| Dependency | On error | On timeout | On malformed response |
|------------|----------|-----------|----------------------|
| git binary | EnsureGit returns clear error | exec.CommandContext respects ctx deadline | stderr captured in error message |
| filesystem (t.TempDir) | test fails with clear path | N/A | N/A |

## Negative Tests

- Remove on nonexistent worktree — should be idempotent, no error
- Merge with no changes — should handle gracefully
- Create when branch already exists — should fall back to reuse
- EnsureGit with manipulated PATH — should return error
- Exists on nonexistent milestone — returns false

## Must-Haves

- [ ] Full lifecycle test (create → commit → merge → remove) passes
- [ ] Error case tests for nonexistent removal, no-change merge, existing branch
- [ ] All tests use t.Parallel() and t.TempDir()
- [ ] Tests use exec.CommandContext for git setup, not shell package
- [ ] All assertions use testify require

## Verification

- `go test ./internal/auto/... -run TestWorktree -v` — all tests pass
- `go test ./internal/auto/... -count=1` — no cached results, clean run
- `go vet ./internal/auto/...` — no warnings

## Inputs

- `internal/auto/worktree.go`
- `internal/config/config.go`

## Expected Output

- `internal/auto/worktree_test.go`

## Verification

go test ./internal/auto/... -run TestWorktree -v -count=1 && go vet ./internal/auto/...
