# S02: Domain Model + Status Enums — Research

**Date:** 2026-03-27
**Depth:** Light — straightforward application of established `session.TodoStatus` pattern to new auto-mode entities.

## Summary

S02 creates a Go domain model in `internal/auto/` that wraps the SQLC-generated `db.Milestone`, `db.Slice`, and `db.Task` structs with typed status/phase enums and DB↔domain conversion functions. The codebase has an exact precedent: `internal/session/session.go` defines `TodoStatus` as a typed `string` constant set, defines a `Session` domain struct separate from `db.Session`, and provides `fromDBItem()` for conversion. S02 replicates this pattern for three entity types.

The work is low-risk. The DB schema (S01) already defines the valid status and phase values as TEXT column defaults. The domain model simply gives those strings type safety in Go.

## Recommendation

Create a single `internal/auto/` package with three files — `milestone.go`, `slice.go`, `task.go` — plus a shared `status.go` for enums. Follow the `session.go` pattern exactly: typed `string` constants for enums, domain structs with Go-native types (no `sql.Null*`), and `FromDB*()` conversion functions. Add a `status_test.go` and `model_test.go` to prove enum validity and round-trip conversion.

## Implementation Landscape

### Key Files

- `internal/session/session.go` (lines 18-26) — **Pattern to follow.** `TodoStatus` typed string enum with constants. `fromDBItem()` converts `db.Session` → `Session`, unwrapping `sql.NullString` to plain `string`.
- `internal/db/models.go` — **Input.** SQLC-generated `Milestone`, `Slice`, `Task` structs with `string` status/phase and `sql.NullString` for optional fields.
- `internal/db/migrations/20260327000000_add_auto_tables.sql` — **Source of truth** for valid status/phase values: status defaults to `'pending'`, phase defaults to `'pre_planning'`.
- `internal/auto/` — **New package.** Does not exist yet. All domain model code goes here.

### Status Enum Values

From the milestone context and schema defaults:

**Status** (shared across all three entities):
- `pending` — not yet started
- `active` — currently in progress
- `completed` — finished
- `blocked` — waiting on dependency (slices/tasks only, but define for all)

**Phase** (shared across all three entities):
- `pre_planning` — before any planning work (schema default)
- `planning` — decomposition/plan creation
- `researching` — research phase (slices)
- `executing` — task execution
- `summarizing` — writing summaries/completion
- `validating` — validation gates
- `completed` — done

These are the phases listed in the milestone context document.

### Domain Structs

Each domain struct mirrors the DB struct but with:
1. Typed `Status` and `Phase` fields instead of raw `string`
2. Plain `string` instead of `sql.NullString` for optional fields (`DependsOn`, `Description`)
3. `time.Time` or keep `int64` for timestamps (keep `int64` to match existing pattern — `session.Session` uses `int64`)

### Build Order

1. **`internal/auto/status.go`** — Define `Status` and `Phase` typed enums with all constants. This is the foundation everything else uses. Includes `IsValid()` methods.
2. **`internal/auto/milestone.go`** — `Milestone` domain struct + `MilestoneFromDB(db.Milestone) Milestone` + `(m Milestone) ToDB() db.CreateMilestoneParams` conversion.
3. **`internal/auto/slice.go`** — `Slice` domain struct + `SliceFromDB()` + `ToDB()`.
4. **`internal/auto/task.go`** — `Task` domain struct + `TaskFromDB()` + `ToDB()`.
5. **`internal/auto/status_test.go`** — Enum validity tests.
6. **`internal/auto/model_test.go`** — Round-trip conversion tests (create domain → ToDB → simulate DB round-trip → FromDB → compare).

### Verification Approach

```bash
# Package compiles
go build ./internal/auto/...

# Tests pass
go test ./internal/auto/ -v -count=1

# No vet issues
go vet ./internal/auto/...

# Existing tests still pass (no regressions)
go test ./internal/db/ -run TestAuto -v -count=1
```

## Constraints

- `CGO_ENABLED=0` — no C dependencies (not relevant here, but inherited constraint).
- Domain model must not import anything beyond `internal/db` and stdlib. Keep it dependency-free for S03 to consume cleanly.
- JSON tags should use `snake_case` per project style (`AGENTS.md`).
- Status/phase constants must exactly match the TEXT values stored in SQLite — case-sensitive string comparison.
- The `FromDB` functions are exported (capital F) because S03's state derivation engine will call them from a different file or test. Follow the pattern of `session.go` where `fromDBItem` is unexported because it's only used within the same package — but here, S03 needs cross-package access, so export them.

## Common Pitfalls

- **Enum drift** — If phase values in Go don't match what the DB stores, state derivation (S03) will silently fail to match. Mitigate: test that every constant round-trips through the DB schema defaults.
- **`sql.NullString` zero-value** — `db.Slice.DependsOn` is `sql.NullString`. When `Valid` is false, `.String` is `""`. The `FromDB` function must handle this explicitly (empty string = no dependency), not propagate the `sql.Null` wrapper.