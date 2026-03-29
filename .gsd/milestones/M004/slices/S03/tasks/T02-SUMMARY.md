---
id: T02
parent: S03
milestone: M004
provides: []
requires: []
affects: []
key_files: ["internal/auto/worktree_test.go", "internal/auto/worktree.go"]
key_decisions: ["Fixed runGit error output to include both stdout and stderr so nothing-to-commit detection works", "Added nothing-added-to-commit variant check in Merge for cases with untracked worktree dirs"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "All 10 tests pass with go test ./internal/auto/... -run TestWorktree -v -count=1 (exit 0). go vet ./internal/auto/... clean. go build ./internal/auto/... and go build ./internal/config/... both succeed."
completed_at: 2026-03-28T06:37:03.553Z
blocker_discovered: false
---

# T02: Added 10 comprehensive WorktreeManager tests with real git repos covering full lifecycle and error paths

> Added 10 comprehensive WorktreeManager tests with real git repos covering full lifecycle and error paths

## What Happened
---
id: T02
parent: S03
milestone: M004
key_files:
  - internal/auto/worktree_test.go
  - internal/auto/worktree.go
key_decisions:
  - Fixed runGit error output to include both stdout and stderr so nothing-to-commit detection works
  - Added nothing-added-to-commit variant check in Merge for cases with untracked worktree dirs
duration: ""
verification_result: passed
completed_at: 2026-03-28T06:37:03.554Z
blocker_discovered: false
---

# T02: Added 10 comprehensive WorktreeManager tests with real git repos covering full lifecycle and error paths

**Added 10 comprehensive WorktreeManager tests with real git repos covering full lifecycle and error paths**

## What Happened

Created internal/auto/worktree_test.go with 10 test functions against real git repositories via t.TempDir(). Tests cover: deterministic path/branch generation, EnsureGit with and without git on PATH, Exists returning false for nonexistent milestones, Create with new and existing branches, idempotent Remove of nonexistent worktrees, Merge with no changes handling, and a full lifecycle test (create → commit in worktree → merge back → verify file on main → verify commit message → remove → verify cleanup). During testing, discovered and fixed two bugs in worktree.go: runGit now includes both stdout and stderr in error messages (needed for 'nothing to commit' detection), and Merge checks for both 'nothing to commit' and 'nothing added to commit' variants.

## Verification

All 10 tests pass with go test ./internal/auto/... -run TestWorktree -v -count=1 (exit 0). go vet ./internal/auto/... clean. go build ./internal/auto/... and go build ./internal/config/... both succeed.

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go test ./internal/auto/... -run TestWorktree -v -count=1` | 0 | ✅ pass | 3600ms |
| 2 | `go vet ./internal/auto/...` | 0 | ✅ pass | 500ms |
| 3 | `go build ./internal/auto/...` | 0 | ✅ pass | 500ms |
| 4 | `go build ./internal/config/...` | 0 | ✅ pass | 500ms |


## Deviations

Fixed two bugs in worktree.go discovered by tests: (1) runGit now includes stdout+stderr in error messages, (2) Merge checks for both 'nothing to commit' and 'nothing added to commit' variants. EnsureGit_MissingPath test cannot use t.Parallel() due to t.Setenv restriction.

## Known Issues

None.

## Files Created/Modified

- `internal/auto/worktree_test.go`
- `internal/auto/worktree.go`


## Deviations
Fixed two bugs in worktree.go discovered by tests: (1) runGit now includes stdout+stderr in error messages, (2) Merge checks for both 'nothing to commit' and 'nothing added to commit' variants. EnsureGit_MissingPath test cannot use t.Parallel() due to t.Setenv restriction.

## Known Issues
None.
