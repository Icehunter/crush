---
estimated_steps: 55
estimated_files: 2
skills_used: []
---

# T03: Render auto-mode progress section in TUI sidebar with milestone tree, cost, and time

## Description

Add an `autoModeInfo()` render method to the sidebar that displays auto-mode progress when `autoSnapshot` is non-nil. Wire it into `drawSidebar()` so the auto-mode section appears above the files section when auto-mode is active. All content must fit within the 30-char sidebar width.

**Read `internal/ui/AGENTS.md` before starting.** Key rules:
- Sidebar is a method on UI, not a sub-model
- Use `*common.Common` for styles access
- Use lipgloss for styling, `ansi` package for string width manipulation
- Never block in Update or do IO in render methods

The sidebar currently renders: logo, title, cwd, model info, files, LSPs, MCPs. When auto-mode is active, insert the auto-mode section between model info and files, and reduce the space given to files/LSPs/MCPs.

## Steps

1. In `internal/ui/model/sidebar.go`, add `autoModeInfo(width int) string` method on `*UI`:
   - Return empty string if `m.autoSnapshot == nil`
   - Render header: status icon + "Auto Mode" + status text (e.g. "▶ Running", "⏸ Paused", "✓ Done", "✗ Error"). Use `t.Accent` for running, `t.Muted` for paused, `t.Success`/`t.Error` for done/error.
   - Render milestone title truncated to width
   - Render slice tree: for each slice, show status icon (✓/▶/○/✗) + truncated title + progress fraction (e.g. "2/5")
   - Render active unit line: "→ " + active unit ID truncated to width
   - Render cost line: "Cost: $X.XX" 
   - Render elapsed time: "Time: Xm Xs" or "Xh Xm" for longer durations
   - Use `ansi.Truncate` for safe truncation of styled strings
2. In `drawSidebar()`, after the model info block and before files:
   - Call `autoSection := m.autoModeInfo(width)`
   - If non-empty, insert it into the blocks slice
   - Reduce `remainingHeight` by the auto section height to give less space to files/LSPs/MCPs
3. Create `internal/ui/model/sidebar_auto_test.go` with tests:
   - `TestAutoModeInfo_Nil` — returns empty string when autoSnapshot is nil
   - `TestAutoModeInfo_Running` — renders all sections with running status
   - `TestAutoModeInfo_Paused` — renders with paused status icon
   - `TestAutoModeInfo_Truncation` — verify output fits within 30-char width with long titles
   - `TestAutoModeInfo_EmptySlices` — handles snapshot with zero slices gracefully
4. Format with `gofumpt -w internal/ui/model/` and run `go vet ./internal/ui/model/...`

## Must-Haves

- [ ] `autoModeInfo()` returns empty string when autoSnapshot is nil
- [ ] Running state shows play icon and "Running" in accent color
- [ ] Paused state shows pause icon and "Paused" in muted color  
- [ ] Slice tree shows status icons and progress fractions
- [ ] Active unit line shows current unit ID
- [ ] Cost and elapsed time are displayed
- [ ] All output fits within 30-char width (verified by test)
- [ ] Sidebar renders normally when auto-mode is inactive
- [ ] Tests pass

## Verification

- `go test ./internal/ui/model/... -v -run TestAutoModeInfo` passes
- `go build ./...` compiles
- `go vet ./...` clean

## Negative Tests

- **Malformed inputs**: nil autoSnapshot, snapshot with zero slices, snapshot with empty strings
- **Boundary conditions**: very long milestone title (60+ chars), 20+ slices, zero cost, zero elapsed time

## Inputs

- `internal/auto/event.go` — AutoSnapshot and SliceProgress types from T01
- `internal/ui/model/ui.go` — UI struct with autoSnapshot field from T02
- `internal/ui/model/sidebar.go` — existing drawSidebar() and render patterns
- `internal/ui/styles/styles.go` — style definitions (Accent, Muted, Success, Error)
- `internal/ui/common/common.go` — Common struct and helpers

## Expected Output

- `internal/ui/model/sidebar.go` — modified with autoModeInfo() method and drawSidebar() integration
- `internal/ui/model/sidebar_auto_test.go` — unit tests for auto-mode sidebar rendering

## Inputs

- `internal/auto/event.go`
- `internal/ui/model/ui.go`
- `internal/ui/model/sidebar.go`
- `internal/ui/styles/styles.go`
- `internal/ui/common/common.go`

## Expected Output

- `internal/ui/model/sidebar.go`
- `internal/ui/model/sidebar_auto_test.go`

## Verification

go test ./internal/ui/model/... -v -run TestAutoModeInfo && go build ./... && go vet ./...
