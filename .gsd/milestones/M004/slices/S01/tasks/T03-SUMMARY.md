---
id: T03
parent: S01
milestone: M004
provides: []
requires: []
affects: []
key_files: ["internal/ui/model/sidebar.go", "internal/ui/model/sidebar_auto_test.go"]
key_decisions: ["Used common.Section() with info parameter for status display, matching LSP/MCP section pattern", "Placed auto section between sidebarHeader and files to reduce file/LSP/MCP space when active"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "go test ./internal/ui/model/... -v -run TestAutoModeInfo — 5/5 pass. go build ./... — clean. go vet ./... — clean. go test ./internal/auto/... -v -run TestAutoEvent — 2/2 pass (slice-level check)."
completed_at: 2026-03-28T06:07:29.669Z
blocker_discovered: false
---

# T03: Added autoModeInfo() sidebar render with milestone tree, cost, and elapsed time

> Added autoModeInfo() sidebar render with milestone tree, cost, and elapsed time

## What Happened
---
id: T03
parent: S01
milestone: M004
key_files:
  - internal/ui/model/sidebar.go
  - internal/ui/model/sidebar_auto_test.go
key_decisions:
  - Used common.Section() with info parameter for status display, matching LSP/MCP section pattern
  - Placed auto section between sidebarHeader and files to reduce file/LSP/MCP space when active
duration: ""
verification_result: passed
completed_at: 2026-03-28T06:07:29.669Z
blocker_discovered: false
---

# T03: Added autoModeInfo() sidebar render with milestone tree, cost, and elapsed time

**Added autoModeInfo() sidebar render with milestone tree, cost, and elapsed time**

## What Happened

Added autoModeInfo(width int) string method on *UI in sidebar.go that renders auto-mode progress when autoSnapshot is non-nil. Section displays status header with semantic icons/colors, milestone title, slice tree with progress fractions, active unit, cost, and elapsed time. Modified drawSidebar() to insert auto section between header and files. Created 5 unit tests covering nil, running, paused, truncation, and empty slices.

## Verification

go test ./internal/ui/model/... -v -run TestAutoModeInfo — 5/5 pass. go build ./... — clean. go vet ./... — clean. go test ./internal/auto/... -v -run TestAutoEvent — 2/2 pass (slice-level check).

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go test ./internal/ui/model/... -v -run TestAutoModeInfo` | 0 | ✅ pass | 624ms |
| 2 | `go build ./...` | 0 | ✅ pass | 3400ms |
| 3 | `go vet ./...` | 0 | ✅ pass | 3400ms |
| 4 | `go test ./internal/auto/... -v -run TestAutoEvent` | 0 | ✅ pass | 100ms |


## Deviations

None.

## Known Issues

None.

## Files Created/Modified

- `internal/ui/model/sidebar.go`
- `internal/ui/model/sidebar_auto_test.go`


## Deviations
None.

## Known Issues
None.
