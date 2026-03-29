---
id: T01
parent: S03
milestone: M002
provides: []
requires: []
affects: []
key_files: ["internal/cmd/next.go", "internal/cmd/next_test.go", "internal/cmd/root.go"]
key_decisions: ["Help test uses direct field assertions instead of Execute() to avoid rootCmd TUI side-effects"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "go build . (exit 0), go vet ./internal/cmd/... (exit 0), go test ./internal/cmd/... -count=1 -run TestNext -v (3/3 pass: TestNextCommandRegistered, TestNextCommandArgValidation, TestNextCommandHelp)"
completed_at: 2026-03-27T22:24:31.050Z
blocker_discovered: false
---

# T01: Added crush next [milestone-id] top-level command that executes one auto-mode unit via Engine.Step() and prints progress to stderr

> Added crush next [milestone-id] top-level command that executes one auto-mode unit via Engine.Step() and prints progress to stderr

## What Happened
---
id: T01
parent: S03
milestone: M002
key_files:
  - internal/cmd/next.go
  - internal/cmd/next_test.go
  - internal/cmd/root.go
key_decisions:
  - Help test uses direct field assertions instead of Execute() to avoid rootCmd TUI side-effects
duration: ""
verification_result: passed
completed_at: 2026-03-27T22:24:31.050Z
blocker_discovered: false
---

# T01: Added crush next [milestone-id] top-level command that executes one auto-mode unit via Engine.Step() and prints progress to stderr

**Added crush next [milestone-id] top-level command that executes one auto-mode unit via Engine.Step() and prints progress to stderr**

## What Happened

Created internal/cmd/next.go with a cobra command following the autoStartCmd pattern — sets up app, constructs Engine with placeholder adapters, subscribes to broker events, calls eng.Step() for one unit, drains events, prints result to stderr. Registered in root.go AddCommand block. Created three tests: command registration, exact-args validation, and help content assertions.

## Verification

go build . (exit 0), go vet ./internal/cmd/... (exit 0), go test ./internal/cmd/... -count=1 -run TestNext -v (3/3 pass: TestNextCommandRegistered, TestNextCommandArgValidation, TestNextCommandHelp)

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go build .` | 0 | ✅ pass | 2000ms |
| 2 | `go vet ./internal/cmd/...` | 0 | ✅ pass | 1000ms |
| 3 | `go test ./internal/cmd/... -count=1 -run TestNext -v` | 0 | ✅ pass | 1000ms |


## Deviations

Help test changed from Execute()-based to direct field assertions to avoid rootCmd TUI side-effects in test environment.

## Known Issues

None.

## Files Created/Modified

- `internal/cmd/next.go`
- `internal/cmd/next_test.go`
- `internal/cmd/root.go`


## Deviations
Help test changed from Execute()-based to direct field assertions to avoid rootCmd TUI side-effects in test environment.

## Known Issues
None.
