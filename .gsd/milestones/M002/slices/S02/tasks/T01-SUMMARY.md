---
id: T01
parent: S02
milestone: M002
provides: []
requires: []
affects: []
key_files: ["internal/auto/init_tools.go", "internal/auto/init_tools_test.go", "internal/db/db.go", "internal/auto/milestone.go", "internal/auto/slice.go", "internal/auto/task.go", "internal/db/milestones.sql.go", "internal/db/slices.sql.go", "internal/db/tasks.sql.go", "internal/db/models.go"]
key_decisions: ["Used db.New() instead of db.Prepare() in tests to avoid is_new column schema mismatch", "Validation errors returned as JSON text responses (not Go errors) so LLM can self-correct", "First milestone auto-set to active by querying existing milestone count"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "go build . — compiles clean. go test ./internal/auto/... -count=1 — 64 tests pass (42 existing + 22 new). go vet ./internal/auto/... — clean."
completed_at: 2026-03-27T22:09:21.934Z
blocker_discovered: false
---

# T01: Build three fantasy.AgentTool planning tools (create_milestone, create_slice, create_task) with field validation, SQLite persistence, and 22 unit tests

> Build three fantasy.AgentTool planning tools (create_milestone, create_slice, create_task) with field validation, SQLite persistence, and 22 unit tests

## What Happened
---
id: T01
parent: S02
milestone: M002
key_files:
  - internal/auto/init_tools.go
  - internal/auto/init_tools_test.go
  - internal/db/db.go
  - internal/auto/milestone.go
  - internal/auto/slice.go
  - internal/auto/task.go
  - internal/db/milestones.sql.go
  - internal/db/slices.sql.go
  - internal/db/tasks.sql.go
  - internal/db/models.go
key_decisions:
  - Used db.New() instead of db.Prepare() in tests to avoid is_new column schema mismatch
  - Validation errors returned as JSON text responses (not Go errors) so LLM can self-correct
  - First milestone auto-set to active by querying existing milestone count
duration: ""
verification_result: passed
completed_at: 2026-03-27T22:09:21.935Z
blocker_discovered: false
---

# T01: Build three fantasy.AgentTool planning tools (create_milestone, create_slice, create_task) with field validation, SQLite persistence, and 22 unit tests

**Build three fantasy.AgentTool planning tools (create_milestone, create_slice, create_task) with field validation, SQLite persistence, and 22 unit tests**

## What Happened

Consolidated missing files from main (domain models, SQLC-generated code, migrations, SQL sources), added prepared statement fields to db.go, built three tool constructors following the fantasy.NewAgentTool pattern, and created comprehensive tests using real in-memory SQLite with goose migrations. Each tool validates required fields (returning JSON errors for LLM self-correction), builds a domain model, writes to SQLite via db.Queries, and returns structured JSON responses. The create_milestone tool auto-sets the first milestone to active status.

## Verification

go build . — compiles clean. go test ./internal/auto/... -count=1 — 64 tests pass (42 existing + 22 new). go vet ./internal/auto/... — clean.

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go build .` | 0 | ✅ pass | 4800ms |
| 2 | `go test ./internal/auto/... -count=1` | 0 | ✅ pass | 3000ms |
| 3 | `go vet ./internal/auto/...` | 0 | ✅ pass | 500ms |


## Deviations

Had to manually add milestone/slice/task prepared statement fields to db.go instead of copying from main (main's db.go has unrelated is_new column issue). Used db.New() in tests instead of db.Prepare() to work around schema mismatch.

## Known Issues

Pre-existing: files.sql.go references is_new column with no corresponding migration. Affects db.Prepare() with in-memory DBs but not production.

## Files Created/Modified

- `internal/auto/init_tools.go`
- `internal/auto/init_tools_test.go`
- `internal/db/db.go`
- `internal/auto/milestone.go`
- `internal/auto/slice.go`
- `internal/auto/task.go`
- `internal/db/milestones.sql.go`
- `internal/db/slices.sql.go`
- `internal/db/tasks.sql.go`
- `internal/db/models.go`


## Deviations
Had to manually add milestone/slice/task prepared statement fields to db.go instead of copying from main (main's db.go has unrelated is_new column issue). Used db.New() in tests instead of db.Prepare() to work around schema mismatch.

## Known Issues
Pre-existing: files.sql.go references is_new column with no corresponding migration. Affects db.Prepare() with in-memory DBs but not production.
