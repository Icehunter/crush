# S03: Git Worktree Isolation — UAT

**Milestone:** M004
**Written:** 2026-03-28T06:39:06.225Z

# S03 UAT: Git Worktree Isolation

## Preconditions
- Go 1.24+ installed
- Git 2.x+ installed and on PATH
- Working copy of the crush repo at the worktree path

## Test 1: WorktreeManager builds and vets cleanly
1. Run `go build ./internal/auto/...` → exits 0, no errors
2. Run `go build ./internal/config/...` → exits 0, no errors
3. Run `go vet ./internal/auto/... ./internal/config/...` → exits 0, no warnings

## Test 2: Config field exists and serializes
1. Run `grep 'WorktreeMode' internal/config/config.go` → shows `WorktreeMode string` field with `json:"worktree_mode"` tag
2. Create a crush.json with `{"auto": {"worktree_mode": "per-milestone"}}` → config loads without error

## Test 3: Full test suite passes
1. Run `go test ./internal/auto/... -run TestWorktree -v -count=1`
2. Expected: 10 tests pass (PASS), exit 0
3. Tests include: WorktreePath, BranchName, EnsureGit, EnsureGit_MissingPath, Exists_False, Create, Create_BranchExists, Remove_Nonexistent, Merge_NoChanges, FullLifecycle

## Test 4: Full lifecycle test validates end-to-end flow
1. TestWorktreeManager_FullLifecycle creates a real git repo
2. Creates worktree at `.crush/worktrees/<MID>/` on branch `auto/<MID>`
3. Commits a file in the worktree
4. Squash-merges back — file appears on main branch
5. Commit message matches `auto: milestone <MID> completed`
6. Removes worktree — directory gone, branch deleted

## Test 5: Error path — idempotent Remove
1. TestWorktreeManager_Remove_Nonexistent calls Remove for a milestone with no worktree
2. Expected: no error returned (idempotent)

## Test 6: Error path — Merge with no changes
1. TestWorktreeManager_Merge_NoChanges creates worktree, makes no changes, calls Merge
2. Expected: no error (graceful handling of nothing-to-commit)

## Test 7: Error path — Create with existing branch
1. TestWorktreeManager_Create_BranchExists creates branch manually, then calls Create
2. Expected: succeeds by reusing existing branch (fallback without -b flag)

## Test 8: Error path — missing git binary
1. TestWorktreeManager_EnsureGit_MissingPath sets PATH to empty via t.Setenv
2. Expected: EnsureGit returns error containing "git"
