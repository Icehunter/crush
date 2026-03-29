---
id: S03
parent: M004
milestone: M004
provides:
  - WorktreeManager with Create/Merge/Remove/Exists API for git worktree lifecycle
  - WorktreeMode config field in Options for enabling per-milestone isolation
requires:
  []
affects:
  []
key_files:
  - internal/auto/worktree.go
  - internal/auto/worktree_test.go
  - internal/config/config.go
key_decisions:
  - Squash-merge commit message format: 'auto: milestone <MID> completed'
  - isNotFoundError helper enables idempotent Remove by tolerating missing worktree/branch
  - runGit includes both stdout and stderr in error messages for nothing-to-commit detection
  - Only per-milestone mode supported in MVP (D013)
patterns_established:
  - WorktreeManager uses exec.CommandContext for all git operations (not shell package), consistent with project convention for external process execution
  - Idempotent Remove pattern: check error output for 'not a working tree' / 'not found' before failing
observability_surfaces:
  - slog.Info/Error logging for each worktree operation with milestone ID context
drill_down_paths:
  - .gsd/milestones/M004/slices/S03/tasks/T01-SUMMARY.md
  - .gsd/milestones/M004/slices/S03/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-28T06:39:06.225Z
blocker_discovered: false
---

# S03: Git Worktree Isolation

**WorktreeManager with full git worktree lifecycle (Create/Merge/Remove/Exists) and worktree_mode config, validated by 10 tests against real git repos**

## What Happened

T01 created `internal/auto/worktree.go` containing the `WorktreeManager` struct with all required operations: `NewWorktreeManager` constructor, `EnsureGit` (exec.LookPath verification), deterministic `WorktreePath`/`BranchName` from milestone ID, `Exists` (os.Stat check), `Create` (git worktree add with branch-exists fallback), `Merge` (git merge --squash + commit with nothing-to-commit handling), and `Remove` (git worktree remove + branch -D, idempotent via isNotFoundError helper). The `WorktreeMode` string field was added to the `Options` struct in `internal/config/config.go` with JSON tag `worktree_mode`.

T02 created `internal/auto/worktree_test.go` with 10 test functions exercising the manager against real git repositories via `t.TempDir()`. Tests cover: deterministic path/branch generation, EnsureGit with and without git on PATH, Exists returning false for nonexistent milestones, Create with new and existing branches, idempotent Remove of nonexistent worktrees, Merge with no changes handling, and a full lifecycle test (create → commit in worktree → merge back → verify file on main → verify commit message format → remove → verify cleanup). Testing uncovered and fixed two bugs: runGit now includes both stdout and stderr in error output (needed for nothing-to-commit detection), and Merge checks for both "nothing to commit" and "nothing added to commit" git output variants.

## Verification

All slice-level verification checks pass from the worktree: `go build ./internal/auto/...` (exit 0), `go build ./internal/config/...` (exit 0), `go vet ./internal/auto/... ./internal/config/...` (exit 0, no warnings), `grep -q WorktreeMode internal/config/config.go` (found), `grep -q WorktreeManager internal/auto/worktree.go` (found). All 10 tests pass with `go test ./internal/auto/... -run TestWorktree -v -count=1` (0.296s, exit 0).

## Requirements Advanced

- R018 — Full implementation of WorktreeManager with Create/Merge/Remove/Exists at .crush/worktrees/<MID>/ with squash-merge back, validated by 10 tests

## Requirements Validated

- R018 — 10-test suite in worktree_test.go proves full lifecycle: create worktree, commit in worktree, squash-merge back to integration branch, remove worktree+branch. Error paths covered: idempotent remove, no-change merge, existing branch reuse, missing git.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

T02 fixed two bugs in worktree.go discovered during test development: (1) runGit error output now includes both stdout and stderr for nothing-to-commit detection, (2) Merge checks both 'nothing to commit' and 'nothing added to commit' variants. One test (EnsureGit_MissingPath) cannot use t.Parallel() due to t.Setenv restriction.

## Known Limitations

Only "per-milestone" worktree mode is supported (D013). The WorktreeManager is not yet integrated into the auto-mode engine loop — a future slice must call Create at milestone start and Merge+Remove at milestone completion. Merge conflicts are surfaced as errors but not auto-resolved.

## Follow-ups

Integrate WorktreeManager into the engine loop: call Create when starting a milestone with worktree_mode="per-milestone", set the engine's working directory to the worktree path, call Merge+Remove on milestone completion. Add conflict resolution strategy (pause and surface to user).

## Files Created/Modified

- `internal/auto/worktree.go` — New file: WorktreeManager with Create/Merge/Remove/Exists/EnsureGit operations and runGit helper
- `internal/auto/worktree_test.go` — New file: 10 test functions covering full lifecycle and error paths against real git repos
- `internal/config/config.go` — Added WorktreeMode string field to Options struct with JSON tag worktree_mode
