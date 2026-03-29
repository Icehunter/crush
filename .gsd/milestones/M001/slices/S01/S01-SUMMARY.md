---
id: S01
parent: M001
milestone: M001
provides:
  - Milestone, Slice, Task SQLite tables with status/phase/ordering columns
  - Typed SQLC CRUD queries for milestones, slices, and tasks
  - Generated Go structs: Milestone, Slice, Task in internal/db/models.go
  - setupTestDB test helper pattern for in-memory SQLite testing
requires:
  []
affects:
  - S02
  - S03
  - S04
key_files:
  - internal/db/migrations/20260327000000_add_auto_tables.sql
  - internal/db/sql/milestones.sql
  - internal/db/sql/slices.sql
  - internal/db/sql/tasks.sql
  - internal/db/milestones.sql.go
  - internal/db/slices.sql.go
  - internal/db/tasks.sql.go
  - internal/db/models.go
  - internal/db/db.go
  - internal/db/querier.go
  - internal/db/auto_test.go
  - internal/csync/maps.go
key_decisions:
  - Used DEFAULT (strftime('%s','now')) on timestamp columns to match project's defensive schema pattern
  - Used in-memory SQLite with unique DSN per test name for safe t.Parallel()
  - Fixed csync/maps.go pointer receiver to unblock go vet gate
patterns_established:
  - Auto-mode DB entities use TEXT PK, TEXT status/phase with defaults, INTEGER timestamps with strftime triggers
  - SQLC query names are globally unique with table-name prefix (CreateMilestone, CreateSlice, CreateTask)
  - Test helper setupTestDB opens in-memory SQLite with goose migrations for isolated parallel tests
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M001/slices/S01/tasks/T01-SUMMARY.md
  - .gsd/milestones/M001/slices/S01/tasks/T02-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T18:36:27.635Z
blocker_discovered: false
---

# S01: DB Schema + SQLC Queries

**SQLite tables for milestones, slices, and tasks with typed SQLC CRUD queries, goose migration, and 4 passing end-to-end tests**

## What Happened

Created goose migration 20260327000000_add_auto_tables.sql with three tables (milestones, slices, tasks) following existing project conventions: TEXT primary keys, TEXT status/phase columns with defaults, INTEGER timestamps with strftime triggers, foreign keys with ON DELETE CASCADE, and indexes on foreign key columns. Wrote 19 SQLC queries across three query files (milestones.sql, slices.sql, tasks.sql) covering full CRUD operations. Generated typed Go code via sqlc generate. Built internal/db/auto_test.go with 4 parallel test functions proving end-to-end correctness: CRUD lifecycles for each entity and cascade delete behavior. Fixed a pre-existing go vet failure in internal/csync/maps.go where JSONSchemaAlias used a value receiver on a struct containing sync.RWMutex — changed to pointer receiver.

## Verification

All verification gates pass: sqlc generate (exit 0), go build ./internal/db/... (exit 0), go vet ./... (exit 0), grep confirms Milestone/Slice/Task structs in models.go, all 4 TestAuto* tests pass (TestAutoMilestones, TestAutoSlices, TestAutoTasks, TestAutoCascadeDelete).

## Requirements Advanced

- R001 — Milestones, slices, and tasks now stored as first-class entities in SQLite with status, phase, ordering columns, and full CRUD operations via typed SQLC queries

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

Fixed pre-existing go vet failure in internal/csync/maps.go (value receiver → pointer receiver on JSONSchemaAlias). Files needed to be consolidated from main repo into worktree.

## Known Limitations

None. Schema and queries are complete for S01 scope.

## Follow-ups

None.

## Files Created/Modified

- `internal/db/migrations/20260327000000_add_auto_tables.sql` — Goose migration creating milestones, slices, tasks tables with triggers and indexes
- `internal/db/sql/milestones.sql` — SQLC queries for milestone CRUD (6 queries)
- `internal/db/sql/slices.sql` — SQLC queries for slice CRUD (6 queries)
- `internal/db/sql/tasks.sql` — SQLC queries for task CRUD (7 queries)
- `internal/db/milestones.sql.go` — Generated Go code for milestone queries
- `internal/db/slices.sql.go` — Generated Go code for slice queries
- `internal/db/tasks.sql.go` — Generated Go code for task queries
- `internal/db/models.go` — Updated with Milestone, Slice, Task structs
- `internal/db/db.go` — Updated with new prepared statements
- `internal/db/querier.go` — Updated with new interface methods
- `internal/db/auto_test.go` — 4 parallel tests for CRUD and cascade delete
- `internal/csync/maps.go` — Fixed value receiver to pointer receiver on JSONSchemaAlias
