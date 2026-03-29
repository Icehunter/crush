# S03: Git Worktree Isolation — Research

**Researched:** 2026-03-27
**Depth:** Targeted (new code, but straightforward git CLI operations with known patterns)

## Summary

S03 implements opt-in git worktree isolation for auto-mode. When `auto.worktree_mode` is set to `"per-milestone"` in crush.json, auto-mode creates a git worktree at `.crush/worktrees/<MID>/` on a new branch `auto/<MID>`, runs all work there, and squash-merges back on milestone completion. This is **pure new code** — no git shell-outs exist in the codebase today. The config field already exists and is tested (M003). The main risks are DB path resolution (worktree has different CWD) and merge conflict handling.

## Requirement Coverage

- **R018** (primary, active): "Optional git worktree per milestone. Work happens in `.crush/worktrees/<MID>/`. Squash-merged back on completion." — This slice directly implements R018.
- **R014** (supporting, validated): `AutoConfig.WorktreeMode` field already exists in config. This slice consumes it.
- **R009** (tangent): Crash recovery should extend to worktree cleanup — stale worktrees after crashes.

## Recommendation

Build a standalone `internal/auto/worktree.go` module with three operations (Create, Remove, Merge) plus a Manager that orchestrates the lifecycle. Use `exec.CommandContext` for git commands (not the shell package's POSIX emulator — git needs the real binary). Test with real git repos in `t.TempDir()`. The config interpretation and engine integration are small additions on top.

## Implementation Landscape

### What Exists

| File | Role | Relevant Detail |
|------|------|-----------------|
| `internal/config/config.go:407-413` | `AutoConfig` struct | `WorktreeMode string` with JSON tag `worktree_mode`. No validation of allowed values. |
| `internal/config/auto_test.go` | Config round-trip tests | 3 tests prove parsing. Test uses `"per-slice"` as value — but context doc says per-milestone only. |
| `internal/auto/engine.go:66-86` | Engine struct | Has `dataDir` field (lock file location). Constructor takes `dataDir string`. No `workingDir` field. |
| `internal/auto/verify.go:60-67` | ShellVerifier | Accepts `workingDir` param per call — pattern for worktree-aware execution. |
| `internal/auto/lock.go` | LockFile | Uses `dataDir` for lock path. Lock file must remain in main tree, not worktree. |
| `internal/config/store.go:40-42` | `ConfigStore.WorkingDir()` | Returns project root CWD. This is the anchor for resolving `.crush/` paths. |
| `internal/shell/shell.go` | Shell execution | POSIX emulator via `mvdan.cc/sh`. Can set `WorkingDir`. |

### What Must Be Built

| Component | Location | Description |
|-----------|----------|-------------|
| `WorktreeManager` | `internal/auto/worktree.go` | Create/Remove/Merge operations using `exec.CommandContext("git", ...)` |
| `WorktreeManager` tests | `internal/auto/worktree_test.go` | Integration tests with real git repos in `t.TempDir()` |
| Config validation | `internal/auto/worktree.go` or config | Validate `worktree_mode` values: `""` (disabled), `"per-milestone"` |
| Engine integration point | `internal/auto/engine.go` | Before `Run()` loop: if worktree mode enabled, create worktree, adjust working dir. After completion: merge + remove. |

### Git Command Sequences

**Create worktree:**
```
git worktree add .crush/worktrees/<MID> -b auto/<MID>
```
- Creates worktree directory and new branch in one command
- Fails if branch already exists (handle: `git worktree add .crush/worktrees/<MID> auto/<MID>` without `-b` to reuse existing branch)

**Merge back (from main tree):**
```
git merge --squash auto/<MID>
git commit -m "auto: milestone <MID> completed"
```
- Must run from the integration branch (main/current), not from the worktree
- `--squash` stages all changes without committing — then commit

**Remove worktree:**
```
git worktree remove .crush/worktrees/<MID>
git branch -D auto/<MID>
```
- Remove worktree first, then delete the branch
- `--force` flag available if worktree has uncommitted changes

### Key Constraints

1. **DB path must be absolute**: The engine's `dataDir` and any DB connection must resolve to the main tree's `.crush/` directory, not the worktree's. `ConfigStore.WorkingDir()` provides the anchor. Resolve to absolute path before any `os.Chdir` or worktree creation.

2. **Lock file stays in main tree**: `LockFile` uses `dataDir` which is `.crush/` in the main tree. This is correct — the lock prevents concurrent auto-mode regardless of worktree.

3. **Git must be available**: First shell-out to git in the codebase. Need a `git` binary check (e.g., `exec.LookPath("git")`).

4. **Worktree path**: `.crush/worktrees/<MID>/` relative to project root. The `.crush/` directory is already in `.gitignore` (it contains DB, sessions, etc.), so worktrees inside it won't pollute the repo.

5. **Shell execution**: Use `exec.CommandContext` directly for git commands, not `shell.Shell.Exec()`. The shell package is a POSIX emulator — git operations need the real git binary. This follows the pattern in `internal/config/load.go`, `internal/cmd/session.go`, `internal/agent/tools/rg.go` which all use `exec.CommandContext`.

6. **Worktree mode values**: Context doc says "per-milestone". Test fixture uses "per-slice". **Decision needed**: support only `"per-milestone"` for MVP (simpler — one worktree per milestone run). `"per-slice"` would require creating/merging worktrees for each slice, which is complex and not described in R018.

7. **Crash cleanup**: If auto-mode crashes with an active worktree, the worktree persists on disk. The `Run()` method should use `defer` for cleanup. Additionally, a startup check could detect and clean stale worktrees.

### Natural Task Boundaries

1. **T01: WorktreeManager core** — `worktree.go` with `Create(ctx, projectRoot, milestoneID)`, `Merge(ctx, projectRoot, milestoneID)`, `Remove(ctx, projectRoot, milestoneID)`, `Exists(projectRoot, milestoneID)`. Pure functions that shell out to git. No engine dependency.

2. **T02: WorktreeManager tests** — Create real git repos in `t.TempDir()`, test full lifecycle: create → commit in worktree → merge back → remove. Test error cases: missing git, branch exists, dirty worktree.

3. **T03: Engine integration** — Add worktree lifecycle to `Engine.Run()`: create before loop, merge+remove after loop (in defer). Add `projectRoot` field to Engine or derive from `dataDir`. Config interpretation: read `WorktreeMode` from `AutoConfig`, only act if `"per-milestone"`.

4. **T04: Stale worktree cleanup** — Optional: on engine startup, check for orphaned worktrees and clean them up. Relates to R009 crash recovery.

### Verification Strategy

- **Unit tests**: `worktree_test.go` — create temp git repo, exercise Create/Merge/Remove, assert branch state and file contents
- **Integration**: Engine test with worktree mode enabled — verify working directory switches, verify files appear in main tree after merge
- **Manual**: Set `auto.worktree_mode: "per-milestone"` in crush.json, start auto-mode, verify `.crush/worktrees/<MID>/` appears, verify squash merge on completion
- **Build**: `go build ./internal/auto/...` passes
- **Test**: `go test ./internal/auto/... -run TestWorktree` passes

## Skill Discovery

No relevant skills found. This is pure Go + git CLI work. The `<available_skills>` list has no Go or git-specific skills. No `npx skills find` needed — git worktree management is standard CLI usage, not a library integration.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Git not available on system | Low | `exec.LookPath("git")` check before worktree operations. Return clear error. |
| Merge conflicts on squash-merge | Medium | `git merge --squash` can fail if integration branch diverged. Surface error to user, don't auto-resolve. Leave worktree intact for manual resolution. |
| DB path resolution in worktree | Medium | Resolve `dataDir` to absolute path at Engine construction time (before any worktree ops). Pass absolute path throughout. |
| Stale worktrees after crash | Low | Defer cleanup in `Run()`. Startup check for orphans (T04, optional). |
| "per-slice" vs "per-milestone" ambiguity | Low | Implement "per-milestone" only. Document that "per-slice" is not supported in MVP. Validate config value. |
