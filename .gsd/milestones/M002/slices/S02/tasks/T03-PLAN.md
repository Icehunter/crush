---
estimated_steps: 25
estimated_files: 1
skills_used: []
---

# T03: Add integration test proving init flow populates DB from vision

## Description

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

## Inputs

- ``internal/auto/init_tools.go` — planning tool constructors from T01`
- ``internal/auto/init.go` — RunInit function from T02`
- ``internal/auto/prompts.go` — BuildInitPrompt from T02`
- ``internal/auto/templates/init.md.tpl` — template from T02`
- ``internal/auto/engine_integration_test.go` — existing integration test patterns (in-memory SQLite, goose migrations)`

## Expected Output

- ``internal/auto/init_test.go` — integration tests proving init flow creates correct DB records`

## Verification

go test ./internal/auto/... -count=1 -v && go vet ./internal/auto/... ./internal/cmd/... && go build .
