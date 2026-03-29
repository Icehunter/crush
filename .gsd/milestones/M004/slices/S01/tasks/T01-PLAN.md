---
estimated_steps: 33
estimated_files: 2
skills_used: []
---

# T01: Define AutoEvent types, AutoSnapshot, and SliceProgress structs

## Description

Create the auto-mode event type system in `internal/auto/`. This is the contract between the auto-mode engine (producer, from M002/M003) and the TUI sidebar (consumer, this slice). The worktree does not have `internal/auto/` — it must be created fresh.

**Important context for executor:** The `internal/auto/` directory does NOT exist in this worktree. Prior milestones (M001-M003) created domain models and engine code on separate git branches. This task creates only the event types needed for TUI integration. When those branches merge, there may be reconciliation needed, but the event types are net-new.

The existing pubsub system uses `pubsub.Event[T]` with `EventType` constants. Follow the same pattern — see `internal/pubsub/events.go` for the base types and `internal/app/lsp_events.go` for a concrete example.

## Steps

1. Create `internal/auto/event.go` with:
   - `AutoEventType` string type with constants: `EventAutoStarted`, `EventAutoPaused`, `EventAutoResumed`, `EventAutoCompleted`, `EventAutoError`, `EventUnitStarted`, `EventUnitCompleted`, `EventStateChanged`
   - `AutoEvent` struct: `Type AutoEventType`, `MilestoneID string`, `SliceID string`, `TaskID string`, `Phase string`, `Error string`, `Snapshot *AutoSnapshot`
   - `AutoSnapshot` struct: `MilestoneID string`, `MilestoneTitle string`, `Slices []SliceProgress`, `ActiveUnit string`, `TotalCost float64`, `ElapsedSeconds float64`, `Status string` (running/paused/completed/error)
   - `SliceProgress` struct: `ID string`, `Title string`, `Status string` (pending/active/completed/blocked), `TasksDone int`, `TasksTotal int`
2. Create `internal/auto/event_test.go` with tests:
   - `TestAutoEventType_Constants` — verify all 8 event type constants are distinct non-empty strings
   - `TestAutoSnapshot_Construction` — build a snapshot with 3 slices, verify field access
   - `TestAutoEvent_WithSnapshot` — create an event with a snapshot, verify Snapshot is accessible
3. Run `gofumpt -w internal/auto/` to format
4. Run `go test ./internal/auto/... -v` to verify tests pass
5. Run `go vet ./internal/auto/...` to verify no warnings

## Must-Haves

- [ ] `internal/auto/event.go` exists with all 8 AutoEventType constants
- [ ] AutoEvent struct has Type, MilestoneID, SliceID, TaskID, Phase, Error, Snapshot fields
- [ ] AutoSnapshot struct has MilestoneID, MilestoneTitle, Slices, ActiveUnit, TotalCost, ElapsedSeconds, Status fields
- [ ] SliceProgress struct has ID, Title, Status, TasksDone, TasksTotal fields
- [ ] All tests pass

## Verification

- `go test ./internal/auto/... -v -run TestAutoEvent` passes
- `go vet ./internal/auto/...` clean
- `go build ./internal/auto/...` compiles

## Inputs

- `internal/pubsub/events.go` — base Event[T] and EventType definitions to follow
- `internal/app/lsp_events.go` — concrete example of typed event structs

## Expected Output

- `internal/auto/event.go` — AutoEvent types, AutoSnapshot, SliceProgress
- `internal/auto/event_test.go` — unit tests for event types

## Inputs

- `internal/pubsub/events.go`
- `internal/app/lsp_events.go`

## Expected Output

- `internal/auto/event.go`
- `internal/auto/event_test.go`

## Verification

go test ./internal/auto/... -v -run TestAutoEvent && go vet ./internal/auto/... && go build ./internal/auto/...
