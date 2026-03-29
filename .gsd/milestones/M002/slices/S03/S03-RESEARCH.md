# S03: Manual Stepper — crush next — Research

**Date:** 2026-03-27
**Status:** Complete
**Depth:** Light research — straightforward CLI addition following established patterns

## Summary

`crush next` is a top-level CLI command that executes exactly one auto-mode unit and returns control. It's a thin wrapper around the existing `Engine.Step()` method (already built and tested in S01). The pattern is identical to `crush auto start` but calls `Step()` instead of `Run()`. This is low-risk, well-understood work with clear patterns to follow.

## Recommendation

Create a single file `internal/cmd/next.go` with a cobra command registered as `crush next [milestone-id]`. It follows the exact same adapter pattern as `autoStartCmd` in `internal/cmd/auto.go`:
1. `setupApp(cmd)` → get config/dataDir
2. Create placeholder adapters (same `cmdStateQuerier`, `cmdSessionCreator`, `cmdDispatcher`, `cmdStatusAdvancer`)
3. Construct `auto.NewEngine(...)` 
4. Call `eng.Step(ctx, milestoneID)` (not `Run()`)
5. Print the result (what unit was executed, or "all work complete")

The adapters already exist in `auto.go` — they can be reused directly since they're package-level types.

Add a test verifying the command registers and parses args correctly.

## Implementation Landscape

### Key Files

- **`internal/cmd/auto.go`** — Contains the adapter types (`cmdStateQuerier`, `cmdSessionCreator`, `cmdDispatcher`, `cmdStatusAdvancer`) and the `autoStartCmd` pattern to follow. The adapters are unexported package-level types, directly reusable by `nextCmd`.
- **`internal/auto/engine.go:183`** — `Engine.Step(ctx, milestoneID)` already exists, acquires lock, runs one derive→dispatch→advance cycle, releases lock. Returns `nil` on success or when done.
- **`internal/cmd/root.go:45`** — `rootCmd.AddCommand(...)` block where `nextCmd` must be registered as a top-level command (not under `autoCmd`).
- **`internal/cmd/run.go`** — Pattern reference for top-level non-interactive commands.

### What Exists

- `Engine.Step()` — fully implemented and tested (S01 T02, T04). Acquires lock, creates parent session, derives state, dispatches one unit, advances status, releases lock. 
- All 4 interface adapters in `auto.go` — placeholder stubs ready to use.
- Event subscriber pattern in `autoStartCmd` — can be simplified for single-step (just print result).
- Integration test `TestIntegration_EngineStep` in `engine_integration_test.go` proves Step works.

### What Needs Building

1. **`internal/cmd/next.go`** — New file with `nextCmd` cobra command:
   - `Use: "next [milestone-id]"`, `Args: cobra.ExactArgs(1)`
   - Calls `setupApp(cmd)`, constructs engine with same adapters as `autoStartCmd`
   - Subscribes to broker events briefly to capture what unit ran
   - Calls `eng.Step(ctx, milestoneID)`
   - Prints to stderr: the unit that was executed or "all work complete"
   - Registered in `root.go` `AddCommand` block

2. **`internal/cmd/root.go`** — Add `nextCmd` to the `rootCmd.AddCommand(...)` list.

3. **`internal/cmd/next_test.go`** — Test that the command exists, accepts milestone-id arg, and rejects missing args.

### Verification

- `go build .` — project compiles
- `go vet ./internal/cmd/...` — clean
- `go test ./internal/cmd/... -count=1` — no regressions, new test passes
- Manual: `crush next --help` shows correct usage

## Constraints

- `nextCmd` is a **top-level** command (`crush next`), not a subcommand of `crush auto`. This matches the roadmap description and the UX intent (quick, standalone stepping).
- Adapters are placeholder stubs (same as S01). Full DB wiring is out of scope.
- Must not break any existing cmd tests.

## Relevant Requirements

- **R008** (primary) — `crush next` standalone runner. This slice delivers it.
- **R007** (supporting) — CLI subcommands. `crush next` complements the `crush auto` family.
- **R009** (supporting) — Lock file. `Engine.Step()` already acquires/releases the lock file, preventing concurrent instances.
