---
id: S02
parent: M002
milestone: M002
provides:
  - crush auto init command — LLM-driven vision decomposition into SQLite
  - Three planning tools (create_milestone, create_slice, create_task) reusable for any planning flow
  - init.md.tpl prompt template for structured planning output
  - BuildInitPrompt() and InitPromptContext for rendering planning prompts
requires:
  - slice: S01
    provides: DeriveState() engine, auto DB tables, domain models, phase/status enums
affects:
  - S03
key_files:
  - internal/auto/init_tools.go
  - internal/auto/init_tools_test.go
  - internal/auto/init.go
  - internal/auto/init_test.go
  - internal/auto/templates/init.md.tpl
  - internal/auto/prompts.go
  - internal/cmd/auto.go
key_decisions:
  - Validation errors returned as JSON text (not Go errors) so LLM can self-correct
  - First milestone auto-set to active by querying existing milestone count
  - SessionAgent constructed with IsSubAgent: true to avoid UI side effects during non-interactive planning
  - Dedicated pubsub.Broker scoped to init session lifetime
  - CLI command opens separate db.Connect matching existing session.go/stats.go pattern
patterns_established:
  - Planning tools return JSON success/error responses for LLM consumption — not Go errors
  - RunInit() bypasses Coordinator and constructs SessionAgent directly with a restricted tool set
  - init.md.tpl template pattern for planning-only prompts with structured output conventions
observability_surfaces:
  - none
drill_down_paths:
  - .gsd/milestones/M002/slices/S02/tasks/T01-SUMMARY.md
  - .gsd/milestones/M002/slices/S02/tasks/T02-SUMMARY.md
  - .gsd/milestones/M002/slices/S02/tasks/T03-SUMMARY.md
duration: ""
verification_result: passed
completed_at: 2026-03-27T22:19:50.518Z
blocker_discovered: false
---

# S02: Interactive Planning — crush auto init

**Delivered crush auto init — LLM-driven project decomposition from a user vision string into structured milestone/slice/task records in SQLite.**

## What Happened

S02 implements the interactive planning flow for crush auto mode. The slice delivers three components:

**T01 — Planning Tools (22 new tests).** Three `fantasy.AgentTool` implementations (`create_milestone`, `create_slice`, `create_task`) that the LLM calls during init to populate SQLite. Each tool validates required fields and returns JSON error messages (not Go errors) so the LLM can self-correct. The `create_milestone` tool auto-sets the first milestone to active status by querying existing milestone count, ensuring DeriveState() from S01 picks it up immediately.

**T02 — End-to-End Wiring (2 new tests).** Four deliverables: (1) `init.md.tpl` prompt template instructing the LLM on ID/status/sort conventions, (2) `BuildInitPrompt()` renderer with `InitPromptContext`, (3) `RunInit()` function that constructs a SessionAgent with only the three planning tools (no standard tools) and dispatches non-interactively using `IsSubAgent: true`, (4) `crush auto init "vision"` cobra command wired through App dependencies. The CLI command opens its own DB connection matching existing patterns in session.go/stats.go.

**T03 — Integration Tests (4 new tests).** Proves the full tool-call sequence end-to-end: creates 1 milestone + 2 slices + 3 tasks, verifies status (first milestone active, rest pending), phase (all pre_planning), sort_order, parent relationships, and optional fields like depends_on.

Test count progression: 42 (S01 baseline) → 64 (T01) → 66 (T02) → 69 top-level tests passing. All compile, vet, and test gates green.

## Verification

All three slice-level verification checks pass:
- `go build .` — compiles clean (exit 0)
- `go test ./internal/auto/... -count=1` — all tests pass (exit 0)
- `go vet ./internal/auto/... ./internal/cmd/...` — clean (exit 0)

69 top-level test functions pass across the auto package, covering the S01 engine + S02 init flow.

## Requirements Advanced

- R019 — Implements the planning half — crush auto init decomposes a user vision into milestones/slices/tasks via LLM tool calls
- R006 — Adds init.md.tpl planning prompt template as an additional unit type beyond the five delivered in S01

## Requirements Validated

None.

## New Requirements Surfaced

None.

## Requirements Invalidated or Re-scoped

None.

## Deviations

T01 needed manual db.go edits (adding prepared statement fields) instead of copying from main due to is_new column schema mismatch. Used db.New() in tests instead of db.Prepare() as a workaround. No other deviations.

## Known Limitations

Pre-existing: files.sql.go references is_new column with no corresponding migration, affecting db.Prepare() with in-memory DBs but not production. RunInit() requires a live LLM connection — no offline/mock mode for the full flow (integration tests exercise tools directly, not through SessionAgent dispatch).

## Follow-ups

None.

## Files Created/Modified

- `internal/auto/init_tools.go` — Three fantasy.AgentTool constructors for create_milestone, create_slice, create_task with field validation and SQLite persistence
- `internal/auto/init_tools_test.go` — 22 unit tests covering happy path and validation errors for all three tools
- `internal/auto/init.go` — RunInit() function and InitConfig struct — constructs planning-only SessionAgent and dispatches
- `internal/auto/init_test.go` — 4 integration tests proving end-to-end tool-call sequence creates correct DB records
- `internal/auto/templates/init.md.tpl` — Go template prompt instructing LLM on ID/status/sort conventions for planning
- `internal/auto/prompts.go` — Added BuildInitPrompt() and InitPromptContext for init template rendering
- `internal/auto/prompts_test.go` — Test for BuildInitPrompt rendering
- `internal/cmd/auto.go` — Added autoInitCmd cobra command wired to RunInit
- `internal/db/db.go` — Added milestone/slice/task prepared statement fields
- `internal/auto/milestone.go` — Consolidated domain model from main
- `internal/auto/slice.go` — Consolidated domain model from main
- `internal/auto/task.go` — Consolidated domain model from main
- `internal/db/milestones.sql.go` — Consolidated SQLC-generated code from main
- `internal/db/slices.sql.go` — Consolidated SQLC-generated code from main
- `internal/db/tasks.sql.go` — Consolidated SQLC-generated code from main
- `internal/db/models.go` — Consolidated updated models with Milestone/Slice/Task structs
