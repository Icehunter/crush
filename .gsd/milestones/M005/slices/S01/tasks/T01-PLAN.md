---
estimated_steps: 8
estimated_files: 10
skills_used: []
---

# T01: Merge M003 into M005 and resolve conflicts

Perform `git merge milestone/M003` on the M005 branch. All .go source files merge cleanly (confirmed by merge-tree). Only `.gsd/` metadata files conflict — resolve those by keeping M005's versions (they include M004 completion data). **Critical:** The coordinator.go change from M003 removes the `ReasoningEffort` check in `isAnthropicThinking()` — this is a regression because M005 uses ReasoningEffort extensively. Reject that hunk by reverting coordinator.go to M005's version after merge.

After this task, the tree will NOT compile because both `event.go` (M004) and `events.go` (M003) define `EventUnitStarted` and `EventUnitCompleted` with different types. That's expected — T02 resolves it.

Steps:
1. Run `git merge milestone/M003` from M005 branch
2. For each `.gsd/` conflict: `git checkout --ours .gsd/` to keep M005 versions
3. Verify coordinator.go: `git diff HEAD -- internal/agent/coordinator.go` and revert the `isAnthropicThinking` change with `git checkout HEAD~1 -- internal/agent/coordinator.go` (or manual restore)
4. `git add` all resolved files and commit the merge
5. Verify new M003 files exist: `ls internal/auto/engine.go internal/auto/state.go internal/auto/stuck.go`

## Inputs

- ``internal/auto/event.go` — M004's TUI event types (will conflict with M003's events.go)`
- ``internal/agent/coordinator.go` — M005's version with ReasoningEffort support`
- ``internal/config/config.go` — base for M003's AutoConfig additions`

## Expected Output

- ``internal/auto/engine.go` — M003 engine now on M005 branch`
- ``internal/auto/state.go` — M003 state derivation now on M005 branch`
- ``internal/auto/events.go` — M003 engine events (coexists with event.go, won't compile yet)`
- ``internal/auto/stuck.go` — M003 stuck detector now on M005 branch`
- ``internal/auto/verify.go` — M003 shell verifier now on M005 branch`
- ``internal/auto/budget.go` — M003 budget checker now on M005 branch`
- ``internal/auto/context.go` — M003 context monitor now on M005 branch`
- ``internal/auto/lock.go` — M003 lock file now on M005 branch`
- ``internal/auto/prompts.go` — M003 prompt builder now on M005 branch`
- ``internal/auto/init.go` — M003 init tool definitions now on M005 branch`
- ``internal/auto/init_tools.go` — M003 planning tool implementations now on M005 branch`
- ``internal/auto/unit.go` — M003 unit type now on M005 branch`
- ``internal/auto/templates/execute_task.md.tpl` — M003 prompt templates now on M005 branch`

## Verification

git log --oneline -1 shows merge commit. `ls internal/auto/engine.go internal/auto/state.go internal/auto/stuck.go internal/auto/events.go` all exist. `grep -c ReasoningEffort internal/agent/coordinator.go` returns > 0 (regression not applied).
