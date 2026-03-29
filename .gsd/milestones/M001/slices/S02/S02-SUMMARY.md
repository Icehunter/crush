---
id: S02
parent: M001
milestone: M001
provides:
  - internal/auto/ package with Milestone, Slice, Task domain structs
  - Status and Phase typed enums with IsValid() validation
  - FromDB() and ToDBCreate() conversion functions for all three entity types
requires:
  - slice: S01
    provides: SQLC-generated db.Milestone, db.Slice, db.Task structs and CreateXParams types
affects:
  - S03
key_files:
  - internal/auto/status.go
  - internal/auto/milestone.go
  - internal/auto/slice.go
  - internal/auto/task.go
  - internal/auto/status_test.go
  - internal/auto/model_test.go
key_decisions:
  - NullString helpers placed as unexported functions in slice.go (D003)
  - Status/Phase enum values match DB defaults exactly — case-sensitive string comparison
patterns_established:
  - Domain structs wrap SQLC types with typed enums and plain strings; FromDB()/ToDBCreate() pattern for bidirectional conversion
  - sql.NullString ↔ plain string: empty string maps to Valid=false, non-empty maps to Valid=true
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M001/slices/S02/tasks/T01-SUMMARY.md
  - .gsd/milestones/M001/slices/S02/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T18:44:07.902Z
blocker_discovered: false
---

# S02: Domain Model + Status Enums

**Typed Go domain model in internal/auto/ with Status/Phase enums, Milestone/Slice/Task structs, and bidirectional DB conversion functions — 16 tests proving correctness.**

## What Happened

Created the `internal/auto/` package providing a typed domain layer over the SQLC-generated DB structs from S01. T01 built four source files: `status.go` defines `Status` and `Phase` as typed `string` enums with `IsValid()` methods, using constants whose string values exactly match the DB schema defaults (`pending`, `active`, `completed`, `blocked`, `pre_planning`, `planning`, `researching`, `executing`, `summarizing`, `validating`, `completed`). `milestone.go`, `slice.go`, and `task.go` each define a domain struct with typed enum fields and plain `string` instead of `sql.NullString`, plus `FromDB()` and `ToDBCreate()` bidirectional conversion functions. Unexported `nullStringToString`/`stringToNullString` helpers live in `slice.go` and are shared by both Slice and Task converters.

T02 added comprehensive test coverage: `status_test.go` with 6 tests validating every enum constant passes `IsValid()`, DB value matching, and invalid string rejection; `model_test.go` with 10 tests covering round-trip conversion for each entity type, NullString edge cases (empty ↔ NULL, non-empty round-trips), and typed Status/Phase field preservation through conversion. All tests use `t.Parallel()` and `testify/require` per project conventions.

## Verification

All verification passed:
- `go build ./internal/auto/...` — exit 0, clean compilation
- `go vet ./internal/auto/...` — exit 0, no diagnostics
- `go test ./internal/auto/ -v -count=1` — 16/16 tests PASS
- `go test ./internal/db/ -run TestAuto -v -count=1` — 4/4 tests PASS, no regressions

## Requirements Advanced

- R001 — Domain model layer provides typed Go structs wrapping the DB entities, advancing toward first-class entity support with status and phase tracking

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

T02 needed to copy internal/auto/ source files from the main repo into the worktree since they were untracked. No plan deviations.

## Known Limitations

None. The domain model is complete for the planned entity types.

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/status.go` — Status and Phase typed string enums with IsValid() methods
- `internal/auto/milestone.go` — Milestone domain struct with MilestoneFromDB() and ToDBCreate()
- `internal/auto/slice.go` — Slice domain struct with SliceFromDB(), ToDBCreate(), and NullString helpers
- `internal/auto/task.go` — Task domain struct with TaskFromDB() and ToDBCreate()
- `internal/auto/status_test.go` — 6 tests for enum validity, DB value matching, invalid rejection
- `internal/auto/model_test.go` — 10 tests for round-trip conversion, NullString edge cases, type preservation
