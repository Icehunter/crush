---
estimated_steps: 41
estimated_files: 10
skills_used: []
---

# T01: Build planning tools (create_milestone, create_slice, create_task) with unit tests

## Description

Build the three `fantasy.AgentTool` implementations that the LLM uses during `crush auto init` to populate the DB with milestones, slices, and tasks. Each tool wraps a `db.Queries` Create call with field validation. This is the novel, highest-risk component — the tools must validate required fields, return clear errors for LLM self-correction, and produce correct DB records.

**Important:** The worktree is missing files from main that landed after the branch point (K003). Before building tools, consolidate: `internal/auto/milestone.go`, `internal/auto/slice.go`, `internal/auto/task.go` (domain models with ToDBCreate()), `internal/db/milestones.sql.go`, `internal/db/slices.sql.go`, `internal/db/tasks.sql.go` (SQLC generated), `internal/db/models.go` (updated with Milestone/Slice/Task types), the migration `internal/db/migrations/20260327000000_add_auto_tables.sql`, and the SQL source files `internal/db/sql/milestones.sql`, `internal/db/sql/slices.sql`, `internal/db/sql/tasks.sql`. Copy them from `/Volumes/Engineering/Icehunter/crush/` (the main working copy).

## Steps

1. Consolidate missing files from main into the worktree:
   - Copy `internal/auto/milestone.go`, `internal/auto/slice.go`, `internal/auto/task.go` from main
   - Copy `internal/db/milestones.sql.go`, `internal/db/slices.sql.go`, `internal/db/tasks.sql.go` from main
   - Copy updated `internal/db/models.go` from main (has Milestone/Slice/Task structs)
   - Copy `internal/db/migrations/20260327000000_add_auto_tables.sql` from main
   - Copy `internal/db/sql/milestones.sql`, `internal/db/sql/slices.sql`, `internal/db/sql/tasks.sql` from main
   - Run `go build .` to verify the worktree compiles with all consolidated files

2. Create `internal/auto/init_tools.go` with three tool constructors:
   - `NewCreateMilestoneTool(q *db.Queries) fantasy.AgentTool` — accepts id, title; validates non-empty; sets status=active (first milestone) or pending; calls q.CreateMilestone
   - `NewCreateSliceTool(q *db.Queries) fantasy.AgentTool` — accepts id, milestone_id, title, sort_order, depends_on (optional); validates id/milestone_id/title non-empty; calls q.CreateSlice with status=pending, phase=pre_planning
   - `NewCreateTaskTool(q *db.Queries) fantasy.AgentTool` — accepts id, slice_id, milestone_id, title, description, sort_order; validates required fields; calls q.CreateTask with status=pending, phase=pre_planning
   - Follow the `fantasy.NewAgentTool()` pattern from `internal/agent/tools/bash.go`
   - Tool parameter structs use json tags with description tags for LLM schema generation
   - Return JSON success responses with the created record's ID and title

3. Create `internal/auto/init_tools_test.go` with tests:
   - Test each tool with valid params → verify DB record created correctly
   - Test each tool with missing required fields → verify error message returned
   - Test create_slice with invalid milestone_id → verify foreign key or validation error
   - Test create_task with invalid slice_id → verify validation error
   - Test sort_order is respected
   - Use in-memory SQLite with goose migrations per K004 pattern

4. Run `go test ./internal/auto/... -count=1` — all existing 42 tests + new tool tests pass
5. Run `go vet ./internal/auto/...` — clean

## Must-Haves

- [ ] All consolidated files from main compile in the worktree
- [ ] Three tools implement fantasy.AgentTool with proper parameter validation
- [ ] Tools write correct records to SQLite via db.Queries
- [ ] Missing/empty required fields return clear error messages
- [ ] Unit tests cover happy path and validation error paths
- [ ] All 42 existing auto tests still pass

## Negative Tests

- **Malformed inputs**: empty id, empty title, missing milestone_id for slices, missing slice_id for tasks
- **Boundary conditions**: sort_order=0, very long titles, depends_on with non-existent IDs

## Verification

- `go build .` compiles
- `go test ./internal/auto/... -count=1` — all tests pass (42 existing + new tool tests)
- `go vet ./internal/auto/...` — clean

## Inputs

- ``internal/agent/tools/bash.go` — fantasy.NewAgentTool() pattern to follow`
- ``internal/auto/status.go` — Status/Phase enum constants`
- ``internal/auto/state.go` — MilestoneRow/SliceRow/TaskRow lightweight types`
- ``/Volumes/Engineering/Icehunter/crush/internal/auto/milestone.go` — domain model to consolidate from main`
- ``/Volumes/Engineering/Icehunter/crush/internal/auto/slice.go` — domain model to consolidate from main`
- ``/Volumes/Engineering/Icehunter/crush/internal/auto/task.go` — domain model to consolidate from main`
- ``/Volumes/Engineering/Icehunter/crush/internal/db/milestones.sql.go` — SQLC generated code to consolidate`
- ``/Volumes/Engineering/Icehunter/crush/internal/db/slices.sql.go` — SQLC generated code to consolidate`
- ``/Volumes/Engineering/Icehunter/crush/internal/db/tasks.sql.go` — SQLC generated code to consolidate`

## Expected Output

- ``internal/auto/init_tools.go` — three fantasy.AgentTool constructors wrapping db.Queries`
- ``internal/auto/init_tools_test.go` — unit tests for tool validation and DB writes`
- ``internal/auto/milestone.go` — consolidated domain model from main`
- ``internal/auto/slice.go` — consolidated domain model from main`
- ``internal/auto/task.go` — consolidated domain model from main`
- ``internal/db/milestones.sql.go` — consolidated SQLC generated code`
- ``internal/db/slices.sql.go` — consolidated SQLC generated code`
- ``internal/db/tasks.sql.go` — consolidated SQLC generated code`

## Verification

go build . && go test ./internal/auto/... -count=1 && go vet ./internal/auto/...
