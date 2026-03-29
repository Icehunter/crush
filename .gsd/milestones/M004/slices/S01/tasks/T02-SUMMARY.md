---
id: T02
parent: S01
milestone: M004
provides: []
requires: []
affects: []
key_files: ["internal/app/auto_events.go", "internal/app/auto_events_test.go", "internal/app/app.go", "internal/ui/model/ui.go", "internal/csync/maps.go"]
key_decisions: ["Followed exact LSP event broker pattern for auto-mode events", "Fixed pre-existing go vet error in csync/maps.go (value receiver on struct with sync.RWMutex)"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "All verification checks pass: go test ./internal/app/... -v -run TestAutoEventBroker (3 tests pass), go vet ./... (clean after csync fix), go build ./... (full project compiles), go test ./internal/auto/... -v -run TestAutoEvent (2 tests pass)."
completed_at: 2026-03-28T06:03:39.281Z
blocker_discovered: false
---

# T02: Wired pubsub.Broker[auto.AutoEvent] through App with subscribe/publish/accessor functions, registered in setupEvents(), and added autoSnapshot field + event handler to UI struct

> Wired pubsub.Broker[auto.AutoEvent] through App with subscribe/publish/accessor functions, registered in setupEvents(), and added autoSnapshot field + event handler to UI struct

## What Happened
---
id: T02
parent: S01
milestone: M004
key_files:
  - internal/app/auto_events.go
  - internal/app/auto_events_test.go
  - internal/app/app.go
  - internal/ui/model/ui.go
  - internal/csync/maps.go
key_decisions:
  - Followed exact LSP event broker pattern for auto-mode events
  - Fixed pre-existing go vet error in csync/maps.go (value receiver on struct with sync.RWMutex)
duration: ""
verification_result: passed
completed_at: 2026-03-28T06:03:39.281Z
blocker_discovered: false
---

# T02: Wired pubsub.Broker[auto.AutoEvent] through App with subscribe/publish/accessor functions, registered in setupEvents(), and added autoSnapshot field + event handler to UI struct

**Wired pubsub.Broker[auto.AutoEvent] through App with subscribe/publish/accessor functions, registered in setupEvents(), and added autoSnapshot field + event handler to UI struct**

## What Happened

Created internal/app/auto_events.go with a package-level autoBroker, SubscribeAutoEvents(), PublishAutoEvent(), and AutoBroker() accessor following the exact LSP event pattern. Added auto subscriber to setupEvents() in app.go. In the UI, added autoSnapshot *auto.AutoSnapshot field and a pubsub.Event[auto.AutoEvent] case handler in Update() that stores the snapshot. Also fixed a pre-existing go vet failure in csync/maps.go where JSONSchemaAlias() used a value receiver on a struct containing sync.RWMutex.

## Verification

All verification checks pass: go test ./internal/app/... -v -run TestAutoEventBroker (3 tests pass), go vet ./... (clean after csync fix), go build ./... (full project compiles), go test ./internal/auto/... -v -run TestAutoEvent (2 tests pass).

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go test ./internal/app/... -v -run TestAutoEventBroker` | 0 | ✅ pass | 4000ms |
| 2 | `go vet ./...` | 0 | ✅ pass | 4000ms |
| 3 | `go build ./...` | 0 | ✅ pass | 3000ms |
| 4 | `go test ./internal/auto/... -v -run TestAutoEvent` | 0 | ✅ pass | 1000ms |


## Deviations

Fixed pre-existing go vet error in internal/csync/maps.go:134 — changed JSONSchemaAlias() from value receiver to pointer receiver.

## Known Issues

gofumpt not on PATH in worktree; used goimports as fallback.

## Files Created/Modified

- `internal/app/auto_events.go`
- `internal/app/auto_events_test.go`
- `internal/app/app.go`
- `internal/ui/model/ui.go`
- `internal/csync/maps.go`


## Deviations
Fixed pre-existing go vet error in internal/csync/maps.go:134 — changed JSONSchemaAlias() from value receiver to pointer receiver.

## Known Issues
gofumpt not on PATH in worktree; used goimports as fallback.
