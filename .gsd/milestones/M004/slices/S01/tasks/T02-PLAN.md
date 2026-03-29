---
estimated_steps: 47
estimated_files: 4
skills_used: []
---

# T02: Wire AutoEvent broker through App and subscribe in TUI Update loop

## Description

Create a `pubsub.Broker[auto.AutoEvent]` in the App, expose it for publishing, and wire it into the TUI event pipeline so auto-mode events arrive as `tea.Msg` in `UI.Update()`. Store the latest `AutoSnapshot` on the UI struct for sidebar rendering.

Follow the exact pattern used for LSP events:
- `internal/app/lsp_events.go` creates a package-level broker, exposes `SubscribeLSPEvents()`, and publishes via helper functions
- `internal/app/app.go` calls `setupSubscriber(ctx, wg, "lsp", SubscribeLSPEvents, app.events)` in `setupEvents()`
- `internal/ui/model/ui.go` handles `case pubsub.Event[app.LSPEvent]:` in `Update()`

**Important:** Read `internal/ui/AGENTS.md` before modifying UI code. Key rules: never block in Update, use tea.Cmd for IO, sidebar is a method on UI not a sub-model.

## Steps

1. Create `internal/app/auto_events.go` with:
   - Package-level `autoBroker = pubsub.NewBroker[auto.AutoEvent]()`
   - `SubscribeAutoEvents(ctx context.Context) <-chan pubsub.Event[auto.AutoEvent]` function
   - `PublishAutoEvent(eventType pubsub.EventType, event auto.AutoEvent)` function
   - `AutoBroker() *pubsub.Broker[auto.AutoEvent]` accessor for engine integration
2. In `internal/app/app.go`, add to `setupEvents()`: `setupSubscriber(ctx, app.serviceEventsWG, "auto", SubscribeAutoEvents, app.events)`
3. In `internal/ui/model/ui.go`:
   - Add `autoSnapshot *auto.AutoSnapshot` field to the `UI` struct (near `lspStates` field)
   - Add import for `github.com/charmbracelet/crush/internal/auto`
   - Add `case pubsub.Event[auto.AutoEvent]:` handler in `Update()` that extracts `msg.Payload.Snapshot` and stores it on `m.autoSnapshot`
4. Create `internal/app/auto_events_test.go` with:
   - `TestAutoEventBroker_PublishSubscribe` — subscribe, publish an event, receive it on the channel
   - `TestAutoEventBroker_Shutdown` — verify clean shutdown
5. Format with `gofumpt -w` and verify with `go build ./...` and `go vet ./...`

## Must-Haves

- [ ] `internal/app/auto_events.go` exists with broker, subscribe, and publish functions
- [ ] `setupEvents()` in `app.go` includes auto event subscriber
- [ ] UI struct has `autoSnapshot` field
- [ ] `Update()` handles `pubsub.Event[auto.AutoEvent]` and stores snapshot
- [ ] Broker test passes

## Verification

- `go test ./internal/app/... -v -run TestAutoEventBroker` passes
- `go build ./...` compiles (full project including UI)
- `go vet ./...` clean

## Failure Modes

| Dependency | On error | On timeout | On malformed response |
|------------|----------|-----------|----------------------|
| pubsub.Broker channel | Event silently dropped (existing behavior) | N/A — non-blocking publish | N/A — typed generics prevent malformed |
| UI Update loop | Event arrives as tea.Msg, nil snapshot stored as nil (no crash) | N/A — channel-buffered delivery | N/A — type-safe |

## Inputs

- `internal/auto/event.go` — AutoEvent and AutoSnapshot types from T01
- `internal/app/lsp_events.go` — pattern to follow for broker wiring
- `internal/app/app.go` — setupEvents() and setupSubscriber pattern
- `internal/ui/model/ui.go` — UI struct and Update() method

## Expected Output

- `internal/app/auto_events.go` — broker, subscribe, publish functions
- `internal/app/auto_events_test.go` — broker integration tests
- `internal/app/app.go` — modified with auto subscriber in setupEvents()
- `internal/ui/model/ui.go` — modified with autoSnapshot field and event handler

## Inputs

- `internal/auto/event.go`
- `internal/app/lsp_events.go`
- `internal/app/app.go`
- `internal/ui/model/ui.go`

## Expected Output

- `internal/app/auto_events.go`
- `internal/app/auto_events_test.go`
- `internal/app/app.go`
- `internal/ui/model/ui.go`

## Verification

go test ./internal/app/... -v -run TestAutoEventBroker && go build ./... && go vet ./...
