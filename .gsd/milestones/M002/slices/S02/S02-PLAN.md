# S02: Interactive Planning — crush auto init

**Goal:** `crush auto init "vision"` accepts a user vision string, dispatches a planning prompt to the LLM with 3 custom tools (create_milestone, create_slice, create_task), and the LLM populates SQLite with a structured milestone/slice/task decomposition. The first milestone is set to active so DeriveState() picks it up.
**Demo:** After this: # S02: Interactive Planning — crush auto init — UAT

**Milestone:** M002
**Written:** 2026-03-27T22:19:50.519Z

# UAT: S02 — Interactive Planning — crush auto init

## Preconditions
- Crush built successfully (`go build .`)
- All auto package tests pass (`go test ./internal/auto/... -count=1`)
- No vet errors (`go vet ./internal/auto/... ./internal/cmd/...`)
- S01 auto tables migration applied (milestones, slices, tasks tables exist)

## Test Cases

### TC1: Planning Tools — Create Milestone with Active Status
1. Set up in-memory SQLite with goose migrations
2. Call create_milestone tool with `{"id": "M001", "title": "Build REST API"}`
3. **Expected:** Milestone M001 created with status=active (first milestone auto-activates)
4. Call create_milestone tool with `{"id": "M002", "title": "Add auth"}`
5. **Expected:** Milestone M002 created with status=pending (second milestone stays pending)

### TC2: Planning Tools — Create Slice with Parent Validation
1. Create milestone M001 first
2. Call create_slice tool with `{"id": "S01", "milestone_id": "M001", "title": "Setup project", "sort_order": 1}`
3. **Expected:** Slice S01 created with milestone_id=M001, status=pending, phase=pre_planning, sort_order=1
4. Call create_slice with `{"id": "S02", "milestone_id": "M001", "title": "Add endpoints", "sort_order": 2, "depends_on": "S01"}`
5. **Expected:** Slice S02 created with depends_on=S01, sort_order=2

### TC3: Planning Tools — Create Task with Full Relationships
1. Create milestone M001 and slice S01 first
2. Call create_task tool with `{"id": "T01", "slice_id": "S01", "milestone_id": "M001", "title": "Init go module", "description": "Run go mod init", "sort_order": 1}`
3. **Expected:** Task T01 created with correct parent relationships, status=pending, phase=pre_planning

### TC4: Planning Tools — Field Validation Errors
1. Call create_milestone with empty id: `{"id": "", "title": "Test"}`
2. **Expected:** JSON error response containing "id" and "required" (not a Go error)
3. Call create_slice with missing milestone_id: `{"id": "S01", "title": "Test"}`
4. **Expected:** JSON error response mentioning missing milestone_id
5. Call create_task with missing slice_id
6. **Expected:** JSON error response mentioning missing slice_id
7. Verify all errors are JSON text responses the LLM can parse and self-correct from

### TC5: BuildInitPrompt — Template Rendering
1. Call BuildInitPrompt with InitPromptContext{Vision: "Build a REST API with auth", WorkingDir: "/tmp/test"}
2. **Expected:** Returned string contains "Build a REST API with auth"
3. **Expected:** Returned string references create_milestone, create_slice, create_task tools
4. **Expected:** Returned string specifies ID conventions (M001, S01, T01)

### TC6: Full Integration — Tool Sequence Creates Structured Plan
1. Set up in-memory SQLite with migrations
2. Execute tool sequence simulating LLM: create_milestone(M001) → create_slice(S01, S02) → create_task(T01, T02, T03)
3. Verify 1 milestone in DB with status=active
4. Verify 2 slices with correct sort_order (1, 2) and status=pending
5. Verify 3 tasks with correct slice_id, milestone_id, and sort_order
6. Verify all phases are pre_planning

### TC7: CLI Registration — crush auto init
1. Run `crush auto init --help`
2. **Expected:** Shows usage with vision positional argument
3. **Expected:** Command is registered under the `crush auto` parent command

## Edge Cases

### EC1: Duplicate Milestone ID
- Call create_milestone twice with same ID
- **Expected:** Second call returns error (DB unique constraint)

### EC2: Very Long Vision String
- Pass 5000+ character vision to BuildInitPrompt
- **Expected:** Template renders without truncation

### EC3: Sort Order Zero
- Create slice with sort_order=0
- **Expected:** Tool accepts it (no minimum constraint)

### EC4: DeriveState Integration
- After init creates milestone with status=active, call DeriveState()
- **Expected:** Returns the first dispatchable unit from the newly-created plan (research unit for first slice)


## Tasks
- [x] **T01: Build three fantasy.AgentTool planning tools (create_milestone, create_slice, create_task) with field validation, SQLite persistence, and 22 unit tests** — ## Description

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
  - Estimate: 1h30m
  - Files: internal/auto/init_tools.go, internal/auto/init_tools_test.go, internal/auto/milestone.go, internal/auto/slice.go, internal/auto/task.go, internal/db/milestones.sql.go, internal/db/slices.sql.go, internal/db/tasks.sql.go, internal/db/models.go, internal/db/migrations/20260327000000_add_auto_tables.sql
  - Verify: go build . && go test ./internal/auto/... -count=1 && go vet ./internal/auto/...
- [x] **T02: Wired end-to-end init flow: init.md.tpl prompt template, BuildInitPrompt() renderer, RunInit() with planning-only SessionAgent, and crush auto init cobra command** — ## Description

Wire the init flow end-to-end: a Go template prompt that instructs the LLM to decompose a vision using the planning tools, a `RunInit()` function that constructs a SessionAgent with those tools and dispatches, and a `crush auto init "vision"` cobra command.

The research recommends approach (A): bypass the Coordinator entirely, construct a SessionAgent directly with `agent.NewSessionAgent()`, set the init prompt as system prompt, set the 3 planning tools, and call Run(). This avoids polluting the standard tool set.

However, constructing a SessionAgent requires model objects (`agent.Model`). The Coordinator's `buildAgentModels()` is private. The simplest path: `RunInit()` accepts models as parameters, and the CLI command layer gets them from `app.AgentCoordinator.Model()` (which returns the main model) or from `app.AgentCoordinator.UpdateModels()` then accessing the coordinator's public Model() method. Since init is a planning task, using the main model is acceptable.

## Steps

1. Create `internal/auto/templates/init.md.tpl` — the planning prompt template:
   - Instructs the LLM: "You are a project planner. Decompose the following vision into milestones, slices, and tasks."
   - Specifies ID format conventions: M001, S01, T01 etc.
   - Specifies status/phase conventions: first milestone active, others pending; all slices pending/pre_planning; all tasks pending/pre_planning
   - Specifies sort_order: sequential integers starting from 1
   - Specifies depends_on format: comma-separated slice IDs or empty
   - Template receives `InitPromptContext` with Vision and WorkingDir fields
   - Rendered via `BuildInitPrompt()` function

2. Add `BuildInitPrompt()` to `internal/auto/prompts.go`:
   - Define `InitPromptContext` struct with Vision and WorkingDir
   - Parse and execute `templates/init.md.tpl` with the context
   - Similar to existing `BuildPrompt()` but with different context type

3. Create `internal/auto/init.go` with `RunInit()` function:
   - Signature: `func RunInit(ctx context.Context, cfg InitConfig) error`
   - `InitConfig` holds: vision string, db.Queries, session.Service, agent model, slog.Logger
   - Creates a top-level session via session.Service
   - Auto-approves the session for non-interactive use (permission.Service if available, or pass through)
   - Builds the init prompt via BuildInitPrompt()
   - Constructs planning tools via NewCreateMilestoneTool/NewCreateSliceTool/NewCreateTaskTool
   - Creates SessionAgent via agent.NewSessionAgent() with planning tools and init prompt
   - Calls agent.Run() with the vision as the user prompt
   - Returns error if dispatch fails

4. Add `autoInitCmd` to `internal/cmd/auto.go`:
   - `crush auto init "Build a REST API with auth"` — takes vision as positional arg
   - Calls setupApp(), gets model from app.AgentCoordinator.Model()
   - Calls auto.RunInit() with the vision and wired dependencies
   - Register in init() alongside existing subcommands

5. Verify: `go build .` compiles, `go vet ./internal/auto/... ./internal/cmd/...` clean

## Must-Haves

- [ ] init.md.tpl template renders correctly with vision text
- [ ] BuildInitPrompt() returns non-empty string without error
- [ ] RunInit() constructs SessionAgent with planning tools only (not standard tools)
- [ ] crush auto init registered as cobra subcommand
- [ ] Project compiles and vets clean

## Verification

- `go build .` compiles
- `go test ./internal/auto/... -count=1` — all tests pass (including prompt template test for init)
- `go vet ./internal/auto/... ./internal/cmd/...` — clean
  - Estimate: 1h30m
  - Files: internal/auto/templates/init.md.tpl, internal/auto/prompts.go, internal/auto/init.go, internal/cmd/auto.go
  - Verify: go build . && go test ./internal/auto/... -count=1 && go vet ./internal/auto/... ./internal/cmd/...
- [x] **T03: Added 4 integration tests proving init planning tools create correct DB records with proper status, relationships, and sort order** — ## Description

Write an integration test that proves the full `crush auto init` flow: seed a vision, run init with a mock dispatcher that simulates LLM tool calls, and verify milestones/slices/tasks appear in the DB with correct status, phase, sort_order, and relationships.

The test should exercise the real code path through RunInit() but replace the SessionAgent dispatch with a mock that calls the planning tools directly (simulating what the LLM would do). This proves the tools, DB writes, and wiring all work together.

## Steps

1. Create `internal/auto/init_test.go` with integration tests:
   - `TestRunInit_CreatesStructuredPlan`: Set up in-memory SQLite with goose migrations, create a mock dispatcher that calls create_milestone (M001), create_slice (S01, S02), create_task (T01, T02, T03). Call the planning tools directly (simulating LLM behavior). Verify:
     - 1 milestone exists with status=active
     - 2 slices exist under M001 with correct sort_order and status=pending
     - 3 tasks exist with correct slice_id, milestone_id, sort_order
   - `TestRunInit_FirstMilestoneIsActive`: Create 2 milestones via tools, verify first is active, second is pending
   - `TestBuildInitPrompt_RendersVision`: Call BuildInitPrompt with a test vision, verify output contains the vision text and tool usage instructions

2. Verify the prompt template test for init.md.tpl renders correctly (may already be covered by prompts_test.go pattern — add if missing)

3. Run full test suite: `go test ./internal/auto/... -count=1` — all tests pass
4. Run `go vet ./internal/auto/... ./internal/cmd/...` — clean
5. Run `go build .` — compiles

## Must-Haves

- [ ] Integration test proves tools create correct DB records
- [ ] First milestone is active, subsequent milestones are pending
- [ ] Slices and tasks have correct parent relationships and sort_order
- [ ] BuildInitPrompt test verifies template renders with vision
- [ ] All existing 42 tests + all new tests pass

## Verification

- `go test ./internal/auto/... -count=1 -v` — all tests pass including new integration tests
- `go vet ./internal/auto/... ./internal/cmd/...` — clean
- `go build .` — compiles
  - Estimate: 45m
  - Files: internal/auto/init_test.go
  - Verify: go test ./internal/auto/... -count=1 -v && go vet ./internal/auto/... ./internal/cmd/... && go build .
