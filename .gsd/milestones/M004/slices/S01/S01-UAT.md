# S01: Event Wiring + Sidebar Panel — UAT

**Milestone:** M004
**Written:** 2026-03-28T06:09:20.700Z

## UAT: S01 — Event Wiring + Sidebar Panel

### Preconditions
- Crush built from this branch (`go build ./...` succeeds)
- Auto-mode engine not yet wired (events must be simulated or tested via unit tests)

### Test Case 1: Event Types Are Complete and Distinct
**Steps:**
1. Run `go test ./internal/auto/... -v -run TestAutoEventType_Constants`
**Expected:** Test passes. All 8 constants (auto_started, auto_paused, auto_resumed, auto_completed, auto_error, unit_started, unit_completed, state_changed) are distinct non-empty strings.

### Test Case 2: AutoSnapshot Construction
**Steps:**
1. Run `go test ./internal/auto/... -v -run TestAutoSnapshot_Construction`
**Expected:** Test passes. Snapshot with 3 slices is constructed, all fields accessible.

### Test Case 3: Broker Publish/Subscribe Round-Trip
**Steps:**
1. Run `go test ./internal/app/... -v -run TestAutoEventBroker_PublishSubscribe`
**Expected:** Test passes. Subscribe to broker, publish an AutoEvent, receive it on the channel with correct payload.

### Test Case 4: Broker Clean Shutdown
**Steps:**
1. Run `go test ./internal/app/... -v -run TestAutoEventBroker_Shutdown`
**Expected:** Test passes. Broker shuts down without deadlock or panic.

### Test Case 5: Broker Accessor Returns Non-nil
**Steps:**
1. Run `go test ./internal/app/... -v -run TestAutoEventBroker_Accessor`
**Expected:** Test passes. `AutoBroker()` returns the same broker instance.

### Test Case 6: Sidebar Returns Empty When Auto-Mode Inactive
**Steps:**
1. Run `go test ./internal/ui/model/... -v -run TestAutoModeInfo_Nil`
**Expected:** Test passes. `autoModeInfo()` returns empty string when `autoSnapshot` is nil.

### Test Case 7: Sidebar Renders Running State
**Steps:**
1. Run `go test ./internal/ui/model/... -v -run TestAutoModeInfo_Running`
**Expected:** Test passes. Output contains play icon, "Running", milestone title, slice tree with progress fractions, active unit, cost, and elapsed time.

### Test Case 8: Sidebar Renders Paused State
**Steps:**
1. Run `go test ./internal/ui/model/... -v -run TestAutoModeInfo_Paused`
**Expected:** Test passes. Output contains pause icon and "Paused" indicator.

### Test Case 9: Sidebar Truncates Long Titles
**Steps:**
1. Run `go test ./internal/ui/model/... -v -run TestAutoModeInfo_Truncation`
**Expected:** Test passes. Output lines fit within 30-char sidebar width even with 60+ char milestone/slice titles.

### Test Case 10: Sidebar Handles Empty Slice List
**Steps:**
1. Run `go test ./internal/ui/model/... -v -run TestAutoModeInfo_EmptySlices`
**Expected:** Test passes. Snapshot with zero slices renders without panic.

### Test Case 11: Full Project Builds Clean
**Steps:**
1. Run `go build ./...`
**Expected:** Exits 0 with no output. All packages including UI compile with new auto-mode types and sidebar integration.

### Test Case 12: No Vet Warnings
**Steps:**
1. Run `go vet ./...`
**Expected:** Exits 0 with no output. The csync/maps.go fix resolved the pre-existing vet error.

### Edge Cases Covered by Unit Tests
- Nil autoSnapshot → empty string (no sidebar section)
- Zero slices in snapshot → renders header/cost/time without slice tree
- Very long titles → truncated to fit 30-char width
- All 4 status states (running/paused/completed/error) → correct icon and color
