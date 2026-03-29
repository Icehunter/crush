---
id: S01
parent: M004
milestone: M004
provides:
  - auto.AutoEvent and auto.AutoSnapshot types for producing/consuming auto-mode progress
  - pubsub.Broker[auto.AutoEvent] accessible via app.AutoBroker() for engine integration
  - app.PublishAutoEvent() for engine to publish state transitions
  - UI sidebar renders autoSnapshot when non-nil — no further TUI wiring needed for live updates
requires:
  []
affects:
  - S02
key_files:
  - internal/auto/event.go
  - internal/auto/event_test.go
  - internal/app/auto_events.go
  - internal/app/auto_events_test.go
  - internal/app/app.go
  - internal/ui/model/ui.go
  - internal/ui/model/sidebar.go
  - internal/ui/model/sidebar_auto_test.go
key_decisions:
  - Followed existing LSP event pattern exactly for auto-mode events — typed string constants, flat structs, package-level broker, setupSubscriber registration
  - Placed auto section between sidebarHeader and files in drawSidebar(), reducing file/LSP/MCP space when active
  - Used common.Section() with info parameter for status display, matching LSP/MCP section pattern
patterns_established:
  - Auto-mode event contract: AutoEvent envelope with optional AutoSnapshot pointer — producers attach snapshots on state changes, consumers nil-check before rendering
  - Sidebar section insertion pattern: conditional section in drawSidebar() with height reduction for downstream sections
observability_surfaces:
  - TUI sidebar auto-mode panel: shows milestone tree, slice progress, active unit, cost, elapsed time in real-time
drill_down_paths:
  - .gsd/milestones/M004/slices/S01/tasks/T01-SUMMARY.md
  - .gsd/milestones/M004/slices/S01/tasks/T02-SUMMARY.md
  - .gsd/milestones/M004/slices/S01/tasks/T03-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-28T06:09:20.700Z
blocker_discovered: false
---

# S01: Event Wiring + Sidebar Panel

**Defined auto-mode event types, wired pubsub broker through App into TUI, and built a live sidebar panel showing milestone tree, active unit, cost, and elapsed time.**

## What Happened

This slice established the event contract and rendering pipeline for auto-mode visibility in the TUI sidebar.

**T01 — Event Types:** Created `internal/auto/event.go` with `AutoEventType` (8 typed string constants: auto_started, auto_paused, auto_resumed, auto_completed, auto_error, unit_started, unit_completed, state_changed), `AutoEvent` (envelope struct with Type, MilestoneID, SliceID, TaskID, Phase, Error, Snapshot pointer), `AutoSnapshot` (point-in-time progress: milestone info, slice list, active unit, cost, elapsed, status), and `SliceProgress` (per-slice ID, Title, Status, TasksDone, TasksTotal). Pattern follows `internal/app/lsp_events.go` exactly. 3 tests.

**T02 — Broker Wiring:** Created `internal/app/auto_events.go` with package-level `autoBroker = pubsub.NewBroker[auto.AutoEvent]()`, `SubscribeAutoEvents()`, `PublishAutoEvent()`, and `AutoBroker()` accessor. Registered in `setupEvents()` via `setupSubscriber()` — identical to LSP event wiring. Added `autoSnapshot *auto.AutoSnapshot` field to the `UI` struct and a `case pubsub.Event[auto.AutoEvent]:` handler in `Update()` that stores the snapshot. Also fixed a pre-existing `go vet` error in `internal/csync/maps.go` (value receiver on struct with sync.RWMutex). 3 broker tests.

**T03 — Sidebar Rendering:** Added `autoModeInfo(width int) string` method on `*UI` in `sidebar.go`. Renders status header with semantic icons/colors (▶ Running in accent, ⏸ Paused in muted, ✓ Done in success, ✗ Error in error), milestone title, slice tree with status icons and progress fractions, active unit line, cost, and elapsed time. Modified `drawSidebar()` to insert auto section between header and files, reducing space for files/LSPs/MCPs when active. Returns empty string when `autoSnapshot` is nil so sidebar renders normally when auto-mode is inactive. Used `common.Section()` with info parameter matching LSP/MCP section pattern. 5 tests covering nil, running, paused, truncation, and empty slices.

## Verification

All slice verification checks pass:
- `go test ./internal/auto/... -v -run TestAutoEvent` — 2/2 pass (event type tests)
- `go test ./internal/auto/... -v` — 3/3 pass (all auto package tests)
- `go test ./internal/app/... -v -run TestAutoEventBroker` — 3/3 pass (broker publish/subscribe/accessor)
- `go test ./internal/ui/model/... -v -run TestAutoModeInfo` — 5/5 pass (nil, running, paused, truncation, empty)
- `go build ./...` — full project compiles clean
- `go vet ./...` — clean (after csync/maps.go fix)

## Requirements Advanced

- R015 — Sidebar panel renders milestone tree with status icons, slice progress fractions, active unit, cost, and elapsed time. 5 unit tests verify rendering for all states.
- R017 — AutoEvent types with 8 EventType constants defined. Broker wired through App via setupEvents(). UI subscribes and stores snapshots. 3 broker tests verify publish/subscribe lifecycle.

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

gofumpt not on PATH in worktree; goimports used as fallback formatter. No functional impact.

## Known Limitations

The auto-mode sidebar rendering is wired end-to-end but the auto-mode engine (from M002/M003) is not yet merged into this branch. Full live integration requires branch reconciliation when M002/M003 merge to main. The event types and broker are ready to receive events from the engine — no additional wiring needed on the TUI side.

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/event.go` — New: AutoEventType constants (8), AutoEvent, AutoSnapshot, SliceProgress structs
- `internal/auto/event_test.go` — New: 3 tests for event type constants, snapshot construction, event-with-snapshot
- `internal/app/auto_events.go` — New: autoBroker, SubscribeAutoEvents, PublishAutoEvent, AutoBroker accessor
- `internal/app/auto_events_test.go` — New: 3 tests for broker publish/subscribe, shutdown, accessor
- `internal/app/app.go` — Modified: added auto event subscriber in setupEvents()
- `internal/ui/model/ui.go` — Modified: added autoSnapshot field, auto event handler in Update()
- `internal/ui/model/sidebar.go` — Modified: added autoModeInfo() method, inserted auto section in drawSidebar()
- `internal/ui/model/sidebar_auto_test.go` — New: 5 tests for sidebar auto rendering (nil, running, paused, truncation, empty)
- `internal/csync/maps.go` — Fixed: pre-existing go vet error — value receiver on struct with sync.RWMutex
