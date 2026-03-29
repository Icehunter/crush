# S03: State Derivation Engine — Research

**Date:** 2026-03-27

## Summary

S03 builds the `DeriveState()` function — the core query that answers "what should auto-mode work on next?" by walking the milestone → slice → task hierarchy in the DB and returning a typed `State` struct with the active work unit and phase. This is the primary deliverable for R002.

The foundation is solid. S01 provides ordered SQLC list queries (`ListMilestones` DESC by created_at, `ListSlicesByMilestone` by sort_order, `ListTasksBySlice` by sort_order). S02 provides typed domain structs (`Milestone`, `Slice`, `Task`) with `Status`/`Phase` enums and `FromDB()` converters. The derivation logic is a straightforward top-down walk: find the active milestone, then its first actionable slice (respecting dependency ordering), then its first actionable task.

The main design decisions are: (1) the shape of the `State` return struct, (2) the rules for "actionable" (active > pending, skip completed/blocked), (3) how to handle edge cases (empty DB, all completed, blocked slices with unmet dependencies). No new DB queries are needed — the existing list queries provide everything.

## Recommendation

Build `DeriveState()` as a pure function taking `*db.Queries` and `context.Context`, returning a `State` struct and error. The `State` struct should contain optional pointers to `*Milestone`, `*Slice`, `*Task` (nil when nothing is active at that level), plus a derived `Action` enum indicating what the caller should do (e.g., `ActionPlanMilestone`, `ActionExecuteTask`, `ActionNone`). Keep it in `internal/auto/` alongside the existing domain model.

The derivation algorithm:
1. `ListMilestones()` → find first with `status=active`. If none, find first `pending`. If none, return `ActionNone`.
2. `ListSlicesByMilestone(milestone.ID)` → walk in sort_order. Skip `completed`. For each non-completed slice, check if `depends_on` is satisfied (dependency slice is completed). First actionable slice wins.
3. `ListTasksBySlice(slice.ID)` → find first `active` task. If none, find first `pending`. If none, slice-level action.
4. Return `State` with the active entities and an action derived from the phase of the deepest active entity.

This approach avoids adding new SQL queries — all data is fetched with existing list operations and filtered in Go. The function is deterministic and testable with seeded in-memory DBs.

## Implementation Landscape

### Key Files

- `internal/auto/state.go` — New file. `State` struct, `Action` type enum, `DeriveState()` function. This is the core deliverable.
- `internal/auto/state_test.go` — New file. Tests seeding milestones/slices/tasks via `db.Queries`, calling `DeriveState()`, asserting correct `State` output for various scenarios.
- `internal/auto/status.go` — Existing. `Status` and `Phase` enums. No changes needed, but heavily consumed.
- `internal/auto/milestone.go`, `slice.go`, `task.go` — Existing domain structs. No changes needed; `FromDB()` converters used by `DeriveState()`.
- `internal/db/auto_test.go` — Existing. Contains `setupTestDB()` helper that S03 tests should reuse or replicate (it's in package `db`, so S03 tests in package `auto` will need their own setup using the same pattern).

### Build Order

1. **First: `State` struct + `Action` enum** — Define the output shape. This is the API contract that S04 (dispatch rules) consumes. Getting this right first means S04 can plan against a stable interface.
2. **Second: `DeriveState()` function** — The algorithm. Takes `context.Context` and `*db.Queries`, returns `(*State, error)`. Walk hierarchy top-down.
3. **Third: Tests** — Seed DB scenarios and assert derivation correctness. Scenarios: empty DB, single pending milestone, active milestone with mixed slice statuses, dependency chain, all completed, blocked task.

### Verification Approach

- `go build ./internal/auto/...` — compiles clean
- `go vet ./internal/auto/...` — no diagnostics
- `go test ./internal/auto/ -v -count=1` — all tests pass (both existing S02 tests and new S03 tests)
- `go test ./internal/db/ -run TestAuto -v -count=1` — no regressions in DB tests

## Constraints

- `CGO_ENABLED=0` — SQLite via modernc.org/sqlite (already satisfied by existing deps).
- `DeriveState()` must accept `*db.Queries` (the SQLC-generated query struct), not raw `*sql.DB`. This follows the established pattern in `internal/db/auto_test.go`.
- Tests need their own `setupTestDB()` since the existing one is in package `db` (unexported). Use the same pattern: in-memory SQLite, goose migrations, foreign keys ON.
- The `State` struct is the contract S04 consumes. Its shape must be stable before S04 starts.

## Common Pitfalls

- **Dependency checking via `depends_on` field** — The `DependsOn` field on `Slice` is a plain string (slice ID). Derivation must check that the referenced slice has `status=completed` before considering a dependent slice actionable. If `DependsOn` is empty, the slice has no dependency. Be careful: `DependsOn` could reference a slice ID that doesn't exist (data integrity issue) — treat as "dependency not met" rather than panicking.
- **Sort order vs status priority** — Slices/tasks are listed by `sort_order`. The derivation should NOT skip ahead to find an `active` item out of order. Walk in order: first non-completed item that's actionable wins. An `active` status at position 3 means positions 1-2 should already be completed.
- **Phase vs Status confusion** — `Status` tracks lifecycle (pending/active/completed/blocked). `Phase` tracks workflow step (planning/executing/etc). `DeriveState()` returns the active entity and its phase — the phase tells the caller WHAT to do, the status tells WHETHER to do it.
- **Test DB setup in package `auto`** — The `setupTestDB` in `internal/db/auto_test.go` is in package `db` and unexported. S03 tests need to import `internal/db` and create their own setup. Use the same goose + in-memory SQLite pattern. Import `internal/db` for `db.FS` (the embedded migrations filesystem).

## Open Risks

- **`depends_on` is a single string, not a list** — The current schema stores `depends_on` as a single TEXT field (one slice ID). If future milestones need multi-dependency slices, this will need a schema change. For S03, treat it as a single optional dependency — this matches the M001 roadmap where slices have linear dependencies (S02→S03→S04).
- **No "active milestone selection" heuristic beyond ordering** — `ListMilestones` returns all milestones ordered by `created_at DESC`. If multiple milestones are `active`, the first one wins (most recently created). This may need revisiting in M002 but is sufficient for M001's integration proof.