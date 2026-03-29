---
estimated_steps: 57
estimated_files: 1
skills_used: []
---

# T02: Write smoke test proving migration and CRUD operations work end-to-end

## Description

Write a Go test file in `internal/db/` that opens an in-memory SQLite database, runs all migrations via goose, and exercises the generated SQLC queries for milestones, slices, and tasks. This proves the schema is correct, foreign keys work, cascade deletes work, and the generated Go code is functional.

## Steps

1. Create `internal/db/auto_test.go` with package `db` (not `db_test` — needs access to internals and matches the generated code package).

2. Write a test helper `setupTestDB(t *testing.T) (*sql.DB, *Queries)` that:
   - Opens an in-memory SQLite DB using the same driver as production (`modernc.org/sqlite` or `github.com/nicruces/go-sqlite3` depending on build tag — check `connect_modernc.go` and `connect_ncruces.go` to see which `openDB` function to call)
   - Sets `PRAGMA foreign_keys = ON` (matching production)
   - Runs `goose.Up(db, "migrations")` with `goose.SetBaseFS(FS)` (matching Connect())
   - Returns the DB and a `New(db)` Queries instance
   - Uses `t.Cleanup` to close the DB

3. Write `TestAutoMilestones(t *testing.T)` that:
   - Creates a milestone via `CreateMilestone`
   - Asserts fields match (id, title, status='pending', phase='pre_planning')
   - Gets it back via `GetMilestone` and asserts equality
   - Lists milestones and asserts length == 1
   - Updates status via `UpdateMilestoneStatus` to 'active' and asserts
   - Updates phase via `UpdateMilestonePhase` to 'planning' and asserts
   - Deletes via `DeleteMilestone` and asserts ListMilestones returns empty

4. Write `TestAutoSlices(t *testing.T)` that:
   - Creates a milestone (prerequisite)
   - Creates two slices with different sort_order values
   - Gets a slice by ID and asserts fields
   - Lists by milestone and asserts order matches sort_order
   - Updates status and phase
   - Deletes a slice and confirms list length decreases

5. Write `TestAutoTasks(t *testing.T)` that:
   - Creates milestone → slice → two tasks with different sort_order
   - Gets a task by ID and asserts fields
   - Lists by slice and asserts order
   - Lists by milestone and asserts both tasks appear
   - Updates status and phase
   - Deletes a task

6. Write `TestAutoCascadeDelete(t *testing.T)` that:
   - Creates milestone → slice → task
   - Deletes the milestone
   - Asserts GetSlice and GetTask return sql.ErrNoRows

7. Run `go test ./internal/db/ -run TestAuto -v` and confirm all pass.

## Must-Haves

- [ ] Test file compiles and runs in the `db` package
- [ ] All four test functions pass
- [ ] Tests use in-memory SQLite with real migrations (not manual DDL)
- [ ] Foreign key cascade delete is tested
- [ ] Tests use `t.Parallel()` where safe and `require` for assertions

## Verification

- `cd /Volumes/Engineering/Icehunter/crush && go test ./internal/db/ -run TestAuto -v -count=1 2>&1 | tail -20`
- All TestAuto* tests show PASS

## Inputs

- `internal/db/milestones.sql.go` — generated milestone query functions from T01
- `internal/db/slices.sql.go` — generated slice query functions from T01
- `internal/db/tasks.sql.go` — generated task query functions from T01
- `internal/db/models.go` — generated Milestone, Slice, Task structs from T01
- `internal/db/db.go` — New() constructor and Queries type from T01
- `internal/db/connect.go` — Connect() pattern reference for test setup
- `internal/db/embed.go` — FS embed for migration files
- `internal/db/connect_modernc.go` — driver-specific openDB for test helper

## Expected Output

- `internal/db/auto_test.go` — test file with TestAutoMilestones, TestAutoSlices, TestAutoTasks, TestAutoCascadeDelete

## Inputs

- `internal/db/milestones.sql.go`
- `internal/db/slices.sql.go`
- `internal/db/tasks.sql.go`
- `internal/db/models.go`
- `internal/db/db.go`
- `internal/db/connect.go`
- `internal/db/embed.go`
- `internal/db/connect_modernc.go`

## Expected Output

- `internal/db/auto_test.go`

## Verification

cd /Volumes/Engineering/Icehunter/crush && go test ./internal/db/ -run TestAuto -v -count=1 2>&1 | grep -E '(PASS|FAIL|ok)' | tail -10
