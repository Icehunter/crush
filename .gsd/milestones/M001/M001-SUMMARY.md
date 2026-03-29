---
id: M001
title: "Hierarchical Task Model + State Machine"
status: complete
completed_at: 2026-03-27T19:23:47.578Z
key_decisions:
  - D003: NullString helpers placed as unexported functions in slice.go — avoids separate utils file for two small functions
  - DB schema uses TEXT PK, TEXT status/phase with defaults, INTEGER timestamps with strftime triggers — matches Crush's existing defensive schema pattern
  - SQLC query names globally unique with table-name prefix (CreateMilestone, CreateSlice, CreateTask)
  - DeriveState dependency check errors treated as 'not met' — skips slice safely without panicking
  - Dispatch rules use Match closures reading State.Action directly — extensible for future multi-field conditions
  - In-package tests (package auto) for state_test.go to access internal helpers directly
key_files:
  - internal/auto/status.go
  - internal/auto/milestone.go
  - internal/auto/slice.go
  - internal/auto/task.go
  - internal/auto/state.go
  - internal/auto/dispatch.go
  - internal/auto/status_test.go
  - internal/auto/model_test.go
  - internal/auto/state_test.go
  - internal/auto/dispatch_test.go
  - internal/auto/integration_test.go
  - internal/db/migrations/20260327000000_add_auto_tables.sql
  - internal/db/sql/milestones.sql
  - internal/db/sql/slices.sql
  - internal/db/sql/tasks.sql
lessons_learned:
  - Worktree isolation means untracked files from the main repo don't appear — files created in the main working copy needed to be consolidated into the worktree
  - In-memory SQLite with unique DSN per test name enables safe t.Parallel() without shared state
  - Lean roadmap format (no explicit success criteria section) still works when slice-level verification is thorough
  - Pre-existing vet failures surface when building in a worktree — fixing them early prevents gate failures downstream
---

# M001: Hierarchical Task Model + State Machine

**Built the internal/auto/ package with SQLite-backed milestones/slices/tasks, typed domain model, state derivation engine, and declarative dispatch rules — proven by 45 tests including full lifecycle integration.**

## What Happened

M001 delivered the data foundation and state machine for Crush's autonomous task orchestration. Work proceeded across 5 slices in dependency order:

S01 created the SQLite schema via goose migration (milestones, slices, tasks tables with status/phase/ordering columns, foreign keys with cascade delete, timestamp triggers) and 19 SQLC queries across three query files. 4 parallel tests proved CRUD and cascade delete behavior.

S02 built the domain model layer in `internal/auto/` — typed Status and Phase string enums with IsValid() validation, Milestone/Slice/Task domain structs wrapping the SQLC types, and bidirectional FromDB()/ToDBCreate() conversion functions. 16 tests covered round-trip conversion, NullString edge cases, and enum correctness.

S03 implemented DeriveState() — the core algorithm that walks milestone→slice→task hierarchy with dependency gating to find the next actionable unit. Returns a State struct with an Action enum (6 constants: none, plan_milestone, plan_slice, execute_task, complete_slice, complete_milestone) plus optional entity pointers. 14 test scenarios covered every branch including empty DB, planning phases, dependency satisfaction/blocking/missing, completion rollups, and sort order.

S04 created the dispatch rules table — a Go slice of Rule structs (Name, Match func, Action) evaluated top-down with a catch-all fallback. Dispatch() maps any State to the correct next action. 11 tests proved every rule fires correctly with edge cases for nil/empty/unknown states.

S05 composed DeriveState→Dispatch into integration tests against real SQLite. TestIntegration_FullLifecycle walks a complete milestone through 11 steps (plan→execute→complete cycle). Three additional tests proved empty DB, dependency gating, and terminal state behavior.

A pre-existing go vet failure in internal/csync/maps.go (value receiver on struct containing sync.RWMutex) was fixed as a side effect in S01.

## Success Criteria Results

The roadmap used a lean format without explicit success criteria bullets. Verification based on deliverables:

- **SQLite schema for milestones/slices/tasks:** ✅ Migration 20260327000000_add_auto_tables.sql creates all three tables with proper constraints, indexes, and triggers.
- **Typed SQLC queries:** ✅ 19 queries across milestones.sql, slices.sql, tasks.sql with generated Go code.
- **Domain model with status/phase enums:** ✅ internal/auto/ package with Status (4 values), Phase (7 values), and three entity structs.
- **DeriveState() function:** ✅ Walks hierarchy with dependency gating, returns next actionable unit.
- **Dispatch rules table:** ✅ 6 ordered rules mapping State→Action, evaluated top-down.
- **Integration proof:** ✅ 4 integration tests proving DeriveState→Dispatch cycle across full lifecycle.
- **All 45 tests pass:** ✅ Confirmed via `go test ./internal/auto/ -v -count=1`.

## Definition of Done Results

- All 5 slices complete: ✅ S01–S05 all marked ✅ in roadmap
- All slice summaries exist: ✅ S01-SUMMARY.md through S05-SUMMARY.md present
- go build ./internal/auto/...: ✅ exits 0
- go vet ./internal/auto/...: ✅ exits 0
- go test ./internal/auto/: ✅ 45/45 PASS
- No cross-slice integration issues: ✅ Each slice built on the prior's exports cleanly. Integration tests in S05 prove the full stack works together.

## Requirement Outcomes

### R001 — active → validated
SQLite tables for milestones/slices/tasks created (S01), domain model with typed enums and DB conversion (S02), and integration tests proving full lifecycle (S05). 45 tests pass.

### R002 — active → validated (during S03)
DeriveState() implemented and proven by 14-scenario test suite covering all branches.

### R003 — active → validated (during S04/S05)
Dispatch rules table with 6 rules. 11 unit tests + 4 integration tests prove correct action for every state transition.

No requirements invalidated or re-scoped.

## Deviations

S01 fixed a pre-existing go vet failure in internal/csync/maps.go (value receiver → pointer receiver on JSONSchemaAlias containing sync.RWMutex). S04 used gofmt instead of gofumpt (not on PATH in worktree). S05 found 41 existing tests instead of the estimated 42. No plan-level deviations.

## Follow-ups

M002 builds the auto-mode execution loop on top of M001's state machine: core loop, prompt templates, CLI commands, crash recovery. The DeriveState→Dispatch cycle is ready to be called from a real execution loop.
