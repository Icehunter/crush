---
id: S03
parent: M002
milestone: M002
provides:
  - crush next [milestone-id] CLI command for single-step auto-mode execution
requires:
  - slice: S01
    provides: auto.Engine, auto.AutoEvent, pubsub.Broker, placeholder adapter types in auto.go
affects:
  []
key_files:
  - internal/cmd/next.go
  - internal/cmd/next_test.go
  - internal/cmd/root.go
key_decisions:
  - crush next is a top-level command, not a subcommand of crush auto
  - Help test uses direct field assertions instead of Execute() to avoid rootCmd TUI side-effects
patterns_established:
  - (none)
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M002/slices/S03/tasks/T01-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T22:25:25.436Z
blocker_discovered: false
---

# S03: Manual Stepper — crush next

**Top-level `crush next [milestone-id]` command that executes one auto-mode unit via Engine.Step() and prints progress to stderr**

## What Happened

This single-task slice added the `crush next` command — a manual stepper that runs exactly one auto-mode unit then exits. The command was implemented in `internal/cmd/next.go` following the same pattern as `autoStartCmd`: it calls `setupApp`, constructs an Engine with the existing placeholder adapters from `auto.go`, subscribes to broker events to capture unit progress, calls `eng.Step()`, and prints what happened to stderr. Three tests cover command registration, exact-args validation, and help content. The command is registered as a top-level command on `rootCmd` (not under `crush auto`).

## Verification

All three verification gates passed: `go build .` (exit 0), `go vet ./internal/cmd/...` (exit 0), `go test ./internal/cmd/... -count=1 -run TestNext -v` (3/3 pass: TestNextCommandRegistered, TestNextCommandArgValidation, TestNextCommandHelp).

## Requirements Advanced

- R007 — Added crush next as a complementary manual-step command alongside the crush auto subcommands

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

Help test uses direct field assertions instead of Execute() to avoid rootCmd TUI side-effects in the test environment.

## Known Limitations

The command uses placeholder adapters (cmdStateQuerier, cmdSessionCreator, cmdDispatcher, cmdStatusAdvancer) that are not yet wired to real implementations — same as `crush auto start`. Real wiring will happen in a later milestone.

## Follow-ups

None.

## Files Created/Modified

- `internal/cmd/next.go` — New top-level crush next command — cobra command calling Engine.Step() with event subscription and stderr output
- `internal/cmd/next_test.go` — Three tests: command registration, exact-args validation, help content
- `internal/cmd/root.go` — Added nextCmd to rootCmd.AddCommand block
