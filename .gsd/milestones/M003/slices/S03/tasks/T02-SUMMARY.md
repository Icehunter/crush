---
id: T02
parent: S03
milestone: M003
provides: []
requires: []
affects: []
key_files: ["internal/auto/context.go", "internal/auto/context_test.go", "internal/auto/events.go", "internal/auto/engine.go", "internal/auto/engine_context_integration_test.go"]
key_decisions: ["Context pressure check runs after dispatch succeeds and after stuck recording, before status advance", "NewContextMonitor returns nil for zero contextWindow or nil querier, making the gate a safe no-op"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "go vet, go build, targeted tests (TestContextMonitor|TestIntegration_Context — 9/9 pass), and full suite (go test ./internal/auto/ — all pass) all green."
completed_at: 2026-03-28T05:32:16.445Z
blocker_discovered: false
---

# T02: Added ContextMonitor with TokenQuerier interface, wired into engine step() to pause and publish EventContextPressure when session token usage exceeds configurable threshold

> Added ContextMonitor with TokenQuerier interface, wired into engine step() to pause and publish EventContextPressure when session token usage exceeds configurable threshold

## What Happened
---
id: T02
parent: S03
milestone: M003
key_files:
  - internal/auto/context.go
  - internal/auto/context_test.go
  - internal/auto/events.go
  - internal/auto/engine.go
  - internal/auto/engine_context_integration_test.go
key_decisions:
  - Context pressure check runs after dispatch succeeds and after stuck recording, before status advance
  - NewContextMonitor returns nil for zero contextWindow or nil querier, making the gate a safe no-op
duration: ""
verification_result: passed
completed_at: 2026-03-28T05:32:16.445Z
blocker_discovered: false
---

# T02: Added ContextMonitor with TokenQuerier interface, wired into engine step() to pause and publish EventContextPressure when session token usage exceeds configurable threshold

**Added ContextMonitor with TokenQuerier interface, wired into engine step() to pause and publish EventContextPressure when session token usage exceeds configurable threshold**

## What Happened

Created internal/auto/context.go with TokenQuerier interface and ContextMonitor struct that compares cumulative session token usage against a configurable fraction of the model context window. NewContextMonitor returns nil for invalid inputs, making the gate a safe no-op when disabled. Added EventContextPressure constant. Extended NewEngine with contextMonitor parameter and updated all 17 existing call sites. Wired context pressure check into engine.step() after dispatch success and stuck recording. Wrote 6 unit tests and 3 integration tests.

## Verification

go vet, go build, targeted tests (TestContextMonitor|TestIntegration_Context — 9/9 pass), and full suite (go test ./internal/auto/ — all pass) all green.

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go vet ./internal/auto/...` | 0 | ✅ pass | 3900ms |
| 2 | `go build ./internal/auto/` | 0 | ✅ pass | 3900ms |
| 3 | `go test ./internal/auto/ -run 'TestContextMonitor|TestIntegration_Context' -count=1 -v` | 0 | ✅ pass | 3500ms |
| 4 | `go test ./internal/auto/ -count=1` | 0 | ✅ pass | 5100ms |


## Deviations

None.

## Known Issues

None.

## Files Created/Modified

- `internal/auto/context.go`
- `internal/auto/context_test.go`
- `internal/auto/events.go`
- `internal/auto/engine.go`
- `internal/auto/engine_context_integration_test.go`


## Deviations
None.

## Known Issues
None.
