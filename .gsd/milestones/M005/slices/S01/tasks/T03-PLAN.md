---
estimated_steps: 14
estimated_files: 7
skills_used: []
---

# T03: Verify full build and test suite passes

Run the complete verification suite to confirm the merge and unification are correct. Fix any remaining compilation errors or test failures.

Steps:
1. Run `go build ./...` — must exit 0
2. Run `go vet ./...` — must exit 0 (check for pre-existing issues per K005)
3. Run `go test ./internal/auto/... -v -count=1` — all engine, state, safety, event, worktree tests pass
4. Run `go test ./internal/config/... -run TestAutoConfig -v` — config parsing tests pass
5. Run `go test ./internal/app/... -run TestAutoEvent -v` — broker tests pass
6. Run `go test ./internal/db/... -v -count=1` — DB tests including SumChildSessionCosts pass
7. If any test fails, diagnose and fix. Common issues:
   - M003 engine tests may reference `NewAutoEvent()` — verify it still exists in unified `events.go`
   - Mock types in engine tests may need updated event field references
   - The `internal/db/auto_test.go` file should be deleted by M003's merge — verify it's gone
8. Run `gofumpt -w .` for final formatting
9. Commit all fixes

## Inputs

- ``internal/auto/events.go` — unified event file from T02`
- ``internal/auto/event_test.go` — rewritten tests from T02`
- ``internal/auto/engine.go` — M003 engine (from T01 merge)`
- ``internal/auto/engine_test.go` — M003 engine tests (from T01 merge)`
- ``internal/app/auto_events_test.go` — updated broker tests from T02`

## Expected Output

- ``internal/auto/events.go` — finalized unified events (may have minor fixes)`
- ``internal/auto/event_test.go` — passing event tests`
- ``internal/auto/engine_test.go` — passing engine tests (may have minor fixes)`

## Verification

`go build ./...` exits 0 && `go vet ./...` exits 0 && `go test ./internal/auto/... -count=1` exits 0 && `go test ./internal/config/... -run TestAutoConfig` exits 0 && `go test ./internal/app/... -run TestAutoEvent` exits 0 && `go test ./internal/db/... -count=1` exits 0
