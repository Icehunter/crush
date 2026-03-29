---
estimated_steps: 11
estimated_files: 5
skills_used: []
---

# T02: Unify AutoEvent types and fix all compilation errors

Resolve the duplicate-symbol compile error by unifying M003's engine events (`events.go`) and M004's TUI events (`event.go`) into a single event system. The engine is the source of truth — M003's `events.go` has 12 event types vs M004's 8, and uses the correct `pubsub.EventType` type.

Steps:
1. Move `AutoSnapshot` and `SliceProgress` structs from `event.go` into `events.go` (after the `AutoEvent` struct)
2. Add `Snapshot *AutoSnapshot` field to M003's `AutoEvent` struct
3. Add M004's lifecycle event constants that M003 is missing (`EventAutoStarted`, `EventAutoPaused`, `EventAutoResumed`, `EventAutoCompleted`, `EventAutoError`, `EventStateChanged`) as `pubsub.EventType` constants — use the same string values M004 used
4. Delete `event.go` entirely
5. Rewrite `event_test.go` to test the unified struct: test that all event constants are unique, test `AutoSnapshot` construction, test `AutoEvent` with both engine fields (Unit, Error, Timestamp, Message) and TUI fields (Snapshot)
6. Update `internal/app/auto_events_test.go`: change `auto.AutoEvent{Type: auto.EventUnitStarted, ...}` to use the new struct shape (remove `Type` field since events are identified by `pubsub.EventType` not a struct field; use `NewAutoEvent()` where appropriate; keep `Snapshot` field)
7. Update `internal/ui/model/ui.go` line ~631: the handler `case pubsub.Event[auto.AutoEvent]: m.autoSnapshot = msg.Payload.Snapshot` should still work since the unified struct has `Snapshot *AutoSnapshot`
8. Run `go build ./...` to confirm compilation
9. Run `gofumpt -w .` to format

## Inputs

- ``internal/auto/events.go` — M003 engine event types (from T01 merge)`
- ``internal/auto/event.go` — M004 TUI event types (to be deleted after extracting types)`
- ``internal/auto/event_test.go` — M004 event tests (to be rewritten)`
- ``internal/app/auto_events_test.go` — app broker tests referencing AutoEvent fields`
- ``internal/ui/model/ui.go` — TUI handler reading `msg.Payload.Snapshot``

## Expected Output

- ``internal/auto/events.go` — unified event file with all constants, AutoEvent with Snapshot field, AutoSnapshot + SliceProgress types`
- ``internal/auto/event_test.go` — rewritten tests for unified event types`
- ``internal/app/auto_events_test.go` — updated to match unified AutoEvent struct`

## Verification

`go build ./...` exits 0. `ls internal/auto/event.go` returns 'No such file' (deleted). `grep -c 'AutoSnapshot' internal/auto/events.go` returns > 0. `grep -c 'Snapshot' internal/auto/events.go` returns > 0.
