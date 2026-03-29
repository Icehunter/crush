# S05: End-to-End Proof + Worktree Lifecycle

**Goal:** Wire remaining nil dependencies in buildAutoEngine (ShellVerifier, StuckDetector), integrate WorktreeManager into Engine.Run lifecycle, and prove the full derive→dispatch→advance chain with E2E integration tests using all real adapters.
**Demo:** After this: # S05: End-to-End Proof + Worktree Lifecycle — UAT

**Milestone:** M005
**Written:** 2026-03-28T19:38:26.429Z

# S05: End-to-End Proof + Worktree Lifecycle — UAT

**Milestone:** M005
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: All changes are wiring and integration tests — verification is compilation, test execution, and structural grep checks. No runtime UI or LLM behavior to observe.

## Preconditions

- M005 worktree checked out with S01-S05 changes
- Go toolchain available (go 1.24+)
- Git available (for worktree lifecycle tests)

## Smoke Test

Run `go build ./...` — must exit 0. Then `go test ./internal/auto/... -count=1 -run TestE2E -v` — all 6 E2E tests must pass.

## Test Cases

### 1. Full compilation gate

1. Run `go build ./...`
2. **Expected:** Exit code 0, no errors.

### 2. Static analysis gate

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no new warnings.

### 3. E2E — ShellVerifier pass path

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_FullAssemblyWithVerifier -v`
2. **Expected:** Test passes — engine dispatches unit, runs verification command `true`, verification passes, unit completes.

### 4. E2E — ShellVerifier fail path

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_FullAssemblyWithFailingVerifier -v`
2. **Expected:** Test passes — engine dispatches unit, runs verification command `false`, verification fails, diagnostic retry fires.

### 5. E2E — StuckDetector triggers

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_FullAssemblyWithStuckDetector -v`
2. **Expected:** Test passes — stuck detector recognizes repeated failures, triggers diagnostic retry with "stuck" warning.

### 6. E2E — Worktree lifecycle (create/merge/remove)

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_WorktreeLifecycle -v`
2. **Expected:** Test passes — worktree created before dispatch, merged after completion, directory removed. Logs show "Creating worktree", "Merging worktree", "Removing worktree".

### 7. E2E — Worktree resume existing

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_WorktreeResumeExisting -v`
2. **Expected:** Test passes — pre-existing worktree detected, logs "Resuming existing worktree", run completes without duplicate creation error.

### 8. E2E — Full adapter composition with real DB

1. Run `go test ./internal/auto/... -count=1 -run TestE2E_BuildAutoEngineComposition -v`
2. **Expected:** Test passes — real DB adapters compose with engine, steps through execute_task → summarize_slice → validate_milestone → done.

### 9. SetWorktreeManager present in engine

1. Run `grep -c 'SetWorktreeManager' internal/auto/engine.go`
2. **Expected:** Count ≥ 2 (field + method).

### 10. buildAutoEngine wires all three components

1. Run `grep -q 'ShellVerifier' internal/cmd/auto.go && grep -q 'NewStuckDetector' internal/cmd/auto.go && grep -q 'NewWorktreeManager' internal/cmd/auto.go && echo "OK"`
2. **Expected:** Prints "OK".

### 11. Full auto test suite regression check

1. Run `go test ./internal/auto/... -count=1`
2. **Expected:** All 100+ tests pass.

### 12. CLI test regression check

1. Run `go test ./internal/cmd/ -count=1`
2. **Expected:** All CLI tests pass.

## Edge Cases

### No verification commands configured

1. `buildAutoEngine` passes nil ShellVerifier when `cfg.Auto.VerificationCommands` is empty.
2. Engine runs without verification gate — covered by existing non-E2E tests.

### StuckThreshold is 0

1. `buildAutoEngine` passes nil StuckDetector when `cfg.Auto.StuckThreshold` is 0.
2. Engine runs without stuck detection — covered by existing tests.

### WorktreeMode not "per-milestone"

1. `buildAutoEngine` skips WorktreeManager construction for any mode other than "per-milestone".
2. Engine.Run skips worktree lifecycle when worktreeManager is nil.

### Worktree merge with no changes

1. Covered by TestE2E_WorktreeLifecycle — mock dispatcher doesn't create files, so merge encounters "nothing to commit".
2. **Expected:** Merge logs "No changes to merge", continues to remove.

## Failure Signals

- `go build ./...` fails → type mismatch in wiring
- TestE2E_WorktreeLifecycle fails → worktree create/merge/remove lifecycle broken
- TestE2E_BuildAutoEngineComposition fails → adapter composition doesn't work with real DB
- TestE2E_FullAssemblyWithVerifier fails → ShellVerifier integration broken
- Any existing test fails → S05 changes introduced regression

## Not Proven By This UAT

- Real LLM dispatch (Dispatcher is mocked — requires API keys and live models)
- File content changes within worktree during dispatch (mock dispatcher is a no-op)
- Production crush.json parsing of WorktreeMode/VerificationCommands/StuckThreshold (config parsing tested in S01)


## Tasks
- [x] **T01: Wired ShellVerifier, StuckDetector, and WorktreeManager from config into buildAutoEngine, added SetWorktreeManager setter and worktree create/merge/cleanup lifecycle to Engine.Run** — Wire the three remaining nil dependencies in buildAutoEngine() and integrate WorktreeManager into the Engine.Run lifecycle.

## Steps

1. **Add SetWorktreeManager setter to Engine** in `internal/auto/engine.go`:
   - Add two fields to Engine struct: `worktreeManager *WorktreeManager` and `worktreeMode string`
   - Add method `func (e *Engine) SetWorktreeManager(wm *WorktreeManager, mode string)` that sets both fields
   - This avoids modifying the NewEngine constructor and its 26+ call sites (K012)

2. **Integrate worktree lifecycle into Engine.Run** in `internal/auto/engine.go`:
   - After lock acquisition and before the main loop, if `e.worktreeManager != nil && e.worktreeMode == "per-milestone"`:
     - If `!e.worktreeManager.Exists(milestoneID)`, call `e.worktreeManager.Create(ctx, milestoneID)` and return error on failure
     - If Exists returns true, log "Resuming existing worktree" and continue
   - After the main loop completes successfully (no error, not paused), if worktreeManager is set:
     - Call `e.worktreeManager.Merge(ctx, milestoneID)` then `e.worktreeManager.Remove(ctx, milestoneID)`
     - Log errors but don't fail the run — the work is done, cleanup is best-effort
   - Do NOT modify Engine.Step — Step is for single-unit execution without lifecycle management

3. **Wire ShellVerifier in buildAutoEngine** in `internal/cmd/auto.go`:
   - After existing adapter construction, check `cfg.Auto.VerificationCommands`
   - If non-empty: `verifier := auto.NewShellVerifier(cfg.Auto.VerificationCommands, slog.Default())`
   - Replace the `nil, // verifier` with the constructed verifier (or nil if no commands configured)

4. **Wire StuckDetector in buildAutoEngine** in `internal/cmd/auto.go`:
   - Check `cfg.Auto.StuckThreshold`
   - If > 0: `stuckDetector := auto.NewStuckDetector(cfg.Auto.StuckThreshold)`
   - Replace the `nil, // stuckDetector` with the constructed detector (or nil if threshold is 0)

5. **Wire WorktreeManager in buildAutoEngine** in `internal/cmd/auto.go`:
   - After engine construction, check `cfg.Auto.WorktreeMode == "per-milestone"`
   - If so: create `auto.NewWorktreeManager(projectRoot)` where projectRoot is the CWD or cfg.Options.ProjectRoot
   - Call `engine.SetWorktreeManager(wm, cfg.Auto.WorktreeMode)`
   - Note: the project root for WorktreeManager should be the current working directory (use `os.Getwd()`) since that's where git operates

6. Run `go build ./...` and `go vet ./...` to confirm compilation.
  - Estimate: 45m
  - Files: internal/auto/engine.go, internal/cmd/auto.go
  - Verify: `go build ./...` exits 0 && `go vet ./...` exits 0 && `grep -q 'SetWorktreeManager' internal/auto/engine.go` && `grep -q 'ShellVerifier' internal/cmd/auto.go` && `grep -q 'NewStuckDetector' internal/cmd/auto.go` && `grep -q 'NewWorktreeManager' internal/cmd/auto.go` && `go test ./internal/auto/... -count=1` (existing tests still pass) && `go test ./internal/cmd/ -count=1` (CLI tests still pass)
- [x] **T02: Added 6 E2E integration tests proving derive→dispatch→advance chain with real ShellVerifier, StuckDetector, WorktreeManager, and DB adapters** — Write integration tests in a new file `internal/auto/engine_e2e_test.go` that prove the complete derive→dispatch→advance chain works with all real adapters composed together, including safety rails and worktree lifecycle.

## Steps

1. **Create `internal/auto/engine_e2e_test.go`** with the following test functions:

2. **TestE2E_FullAssemblyWithVerifier** — Proves engine with ShellVerifier runs verification commands after task dispatch:
   - Use `fixedSequenceQuerier` with one UnitExecuteTask
   - Wire a ShellVerifier with `[]string{"true"}` (always-pass command)
   - Use `mockSessionCreator`, `recordingDispatcher`, `mockAdvancer`
   - Call `engine.Step()` and verify: dispatch called once, advancer called once, no error
   - Then test with a failing verifier: ShellVerifier with `[]string{"false"}` — dispatch should retry then complete or report verification failure

3. **TestE2E_FullAssemblyWithStuckDetector** — Proves engine with StuckDetector fires after repeated failures:
   - Use `fixedSequenceQuerier` with one UnitExecuteTask
   - Wire a StuckDetector with threshold 3
   - Use a dispatcher that always returns an error
   - Call `engine.Step()` multiple times to fill the stuck window
   - Verify the stuck detector records failures correctly

4. **TestE2E_WorktreeLifecycle** — Proves Engine.Run creates, uses, and cleans up a worktree:
   - Use `setupGitRepo(t)` from worktree_test.go to create a real git repo
   - Create an engine with `SetWorktreeManager(wm, "per-milestone")`
   - Use `fixedSequenceQuerier` with a minimal unit sequence (1 task) that terminates quickly
   - Call `engine.Run()` with a milestone ID
   - Verify: worktree was created (directory exists during run), then merged and removed after Run returns
   - To verify the create-then-cleanup, check that the worktree directory does NOT exist after Run completes (merge+remove happened)
   - Also verify the branch exists in git (merge happened before remove)

5. **TestE2E_WorktreeResumeExisting** — Proves Run skips Create when worktree already exists:
   - Use `setupGitRepo(t)` and manually call `wm.Create()` before engine.Run()
   - Verify Run succeeds without error (doesn't try to create duplicate worktree)

6. **TestE2E_BuildAutoEngineComposition** — Proves the adapter composition pattern from buildAutoEngine produces a valid engine:
   - This is a unit-level test that constructs all adapters the same way buildAutoEngine does (using setupAdapterTestDB from existing adapter tests)
   - Seed the DB with a milestone/slice/task
   - Call engine.Step() and verify DeriveState finds work, dispatch is called, advancer advances
   - This test uses real DBStateQuerier, real DBStatusAdvancer, real DBTokenQuerier, real DBBudgetChecker, but mock SessionCreator and mock Dispatcher

7. Run `go test ./internal/auto/... -count=1 -v` to verify all tests pass.

8. Run `go build ./...` and `go vet ./...` as final gates.
  - Estimate: 1h
  - Files: internal/auto/engine_e2e_test.go
  - Verify: `go build ./...` exits 0 && `go vet ./...` exits 0 && `go test ./internal/auto/... -count=1 -run 'TestE2E' -v` (all E2E tests pass) && `go test ./internal/auto/... -count=1` (full suite passes) && `go test ./internal/cmd/ -count=1` (no CLI regressions)
