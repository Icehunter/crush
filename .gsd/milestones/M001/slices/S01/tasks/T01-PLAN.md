---
estimated_steps: 65
estimated_files: 10
skills_used: []
---

# T01: Create migration, SQLC queries, and generate typed Go code

## Description

Write the goose migration DDL for milestones/slices/tasks tables, write SQLC query files for all three tables, run `sqlc generate` to produce typed Go code, and verify compilation. This is the core deliverable of S01 — after this task, the schema exists and Go code can interact with it.

## Steps

1. Create `internal/db/migrations/20260327000000_add_auto_tables.sql` with goose Up/Down sections. The Up section creates three tables:
   - `milestones`: id TEXT PK, title TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', phase TEXT NOT NULL DEFAULT 'pre_planning', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL. Add updated_at trigger.
   - `slices`: id TEXT PK, milestone_id TEXT NOT NULL FK→milestones(id) ON DELETE CASCADE, title TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', phase TEXT NOT NULL DEFAULT 'pre_planning', sort_order INTEGER NOT NULL DEFAULT 0, depends_on TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL. Add index on milestone_id. Add updated_at trigger.
   - `tasks`: id TEXT PK, slice_id TEXT NOT NULL FK→slices(id) ON DELETE CASCADE, milestone_id TEXT NOT NULL FK→milestones(id) ON DELETE CASCADE, title TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', phase TEXT NOT NULL DEFAULT 'pre_planning', sort_order INTEGER NOT NULL DEFAULT 0, description TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL. Add indexes on slice_id and milestone_id. Add updated_at trigger.
   - The Down section drops tables in reverse order (tasks, slices, milestones) and their indexes.
   - Follow existing trigger pattern: `strftime('%s', 'now')` for timestamps (seconds, not milliseconds).

2. Create `internal/db/sql/milestones.sql` with SQLC queries:
   - `CreateMilestone :one` — INSERT with RETURNING *
   - `GetMilestone :one` — SELECT by id
   - `ListMilestones :many` — SELECT all ORDER BY created_at DESC
   - `UpdateMilestoneStatus :one` — UPDATE status WHERE id RETURNING *
   - `UpdateMilestonePhase :one` — UPDATE phase WHERE id RETURNING *
   - `DeleteMilestone :exec` — DELETE by id

3. Create `internal/db/sql/slices.sql` with SQLC queries:
   - `CreateSlice :one` — INSERT with RETURNING *
   - `GetSlice :one` — SELECT by id
   - `ListSlicesByMilestone :many` — SELECT WHERE milestone_id ORDER BY sort_order
   - `UpdateSliceStatus :one` — UPDATE status WHERE id RETURNING *
   - `UpdateSlicePhase :one` — UPDATE phase WHERE id RETURNING *
   - `DeleteSlice :exec` — DELETE by id

4. Create `internal/db/sql/tasks.sql` with SQLC queries:
   - `CreateTask :one` — INSERT with RETURNING *
   - `GetTask :one` — SELECT by id
   - `ListTasksBySlice :many` — SELECT WHERE slice_id ORDER BY sort_order
   - `ListTasksByMilestone :many` — SELECT WHERE milestone_id ORDER BY sort_order
   - `UpdateTaskStatus :one` — UPDATE status WHERE id RETURNING *
   - `UpdateTaskPhase :one` — UPDATE phase WHERE id RETURNING *
   - `DeleteTask :exec` — DELETE by id

5. Run `sqlc generate` from project root. Must exit 0.
6. Run `go build ./internal/db/...`. Must exit 0.
7. Verify generated files exist: `internal/db/milestones.sql.go`, `internal/db/slices.sql.go`, `internal/db/tasks.sql.go`, and that `internal/db/models.go` contains Milestone, Slice, Task structs.

## Must-Haves

- [ ] Migration file follows goose format with Up and Down sections
- [ ] All three tables have TEXT PK, TEXT status/phase with defaults, INTEGER timestamps with triggers
- [ ] Foreign keys with ON DELETE CASCADE from slices→milestones and tasks→slices,milestones
- [ ] SQLC query names are globally unique (prefixed with table name)
- [ ] `sqlc generate` exits 0
- [ ] `go build ./internal/db/...` exits 0
- [ ] Generated models.go contains Milestone, Slice, Task structs

## Verification

- `cd /Volumes/Engineering/Icehunter/crush && sqlc generate && echo OK`
- `cd /Volumes/Engineering/Icehunter/crush && go build ./internal/db/... && echo OK`
- `grep -q 'type Milestone struct' internal/db/models.go`
- `grep -q 'type Slice struct' internal/db/models.go`
- `grep -q 'type Task struct' internal/db/models.go`

## Inputs

- `internal/db/migrations/20260127000000_add_read_files_table.sql` — latest migration, pattern reference for goose format
- `internal/db/sql/sessions.sql` — pattern reference for SQLC query syntax (Create with RETURNING *, Get, List, Update, Delete)
- `internal/db/sql/read_files.sql` — pattern reference for filtered listing queries
- `sqlc.yaml` — SQLC configuration (no changes needed)
- `internal/db/embed.go` — confirms migrations are auto-discovered (no changes needed)

## Expected Output

- `internal/db/migrations/20260327000000_add_auto_tables.sql` — goose migration creating milestones, slices, tasks tables
- `internal/db/sql/milestones.sql` — SQLC queries for milestones CRUD
- `internal/db/sql/slices.sql` — SQLC queries for slices CRUD
- `internal/db/sql/tasks.sql` — SQLC queries for tasks CRUD
- `internal/db/milestones.sql.go` — generated Go code for milestone queries
- `internal/db/slices.sql.go` — generated Go code for slice queries
- `internal/db/tasks.sql.go` — generated Go code for task queries
- `internal/db/models.go` — updated with Milestone, Slice, Task structs
- `internal/db/db.go` — updated with new prepared statements
- `internal/db/querier.go` — updated with new interface methods

## Inputs

- `internal/db/migrations/20260127000000_add_read_files_table.sql`
- `internal/db/sql/sessions.sql`
- `internal/db/sql/read_files.sql`
- `sqlc.yaml`
- `internal/db/embed.go`

## Expected Output

- `internal/db/migrations/20260327000000_add_auto_tables.sql`
- `internal/db/sql/milestones.sql`
- `internal/db/sql/slices.sql`
- `internal/db/sql/tasks.sql`
- `internal/db/milestones.sql.go`
- `internal/db/slices.sql.go`
- `internal/db/tasks.sql.go`
- `internal/db/models.go`
- `internal/db/db.go`
- `internal/db/querier.go`

## Verification

cd /Volumes/Engineering/Icehunter/crush && sqlc generate && go build ./internal/db/... && grep -q 'type Milestone struct' internal/db/models.go && grep -q 'type Slice struct' internal/db/models.go && grep -q 'type Task struct' internal/db/models.go && echo PASS
