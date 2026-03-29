# M001: Hierarchical Task Model + State Machine — Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

## Project Description

Adding autonomous task orchestration to Crush — a Go terminal AI coding assistant by Charm. This milestone builds the data foundation: SQLite tables for milestones/slices/tasks, Go domain model, state derivation engine, and dispatch rules table.

## Why This Milestone

Everything in auto-mode depends on having structured task data and a reliable state machine. The auto loop (M002) needs to ask "what's next?" and get a deterministic answer. Without this foundation, the loop has nothing to dispatch against.

## User-Visible Outcome

### When this milestone is complete, the user can:

- Run integration tests that seed milestones/slices/tasks and prove state derivation → dispatch produces correct action sequences
- See new DB tables via `sqlite3 .crush/crush.db .schema` showing `milestones`, `slices`, `tasks` tables

### Entry point / environment

- Entry point: `go test ./internal/auto/...`
- Environment: local dev
- Live dependencies involved: none (SQLite only)

## Completion Class

- Contract complete means: unit tests pass for domain model, state derivation, and dispatch rules. Integration test proves derive → dispatch cycle.
- Integration complete means: SQLC-generated code compiles, migrations run, domain model converts DB rows correctly.
- Operational complete means: none — no runtime services in this milestone.

## Final Integrated Acceptance

To call this milestone complete, we must prove:

- Seeding milestones/slices/tasks via DB queries and calling DeriveState() returns the correct active unit and phase
- The dispatch rules table, given any valid state, returns the correct next action
- The full derive → dispatch cycle, run in sequence over a multi-slice milestone, produces the expected action order

## Risks and Unknowns

- **SQLC schema design** — getting the table relationships and status enums right. Wrong schema is expensive to fix after M002 builds on it.
- **Phase model completeness** — the set of phases (pre_planning, planning, executing, summarizing, validating, completed) must cover all transitions the auto loop will need. Missing a phase means a rewrite.
- **Dispatch rule ordering** — rules are evaluated top-down. Wrong ordering produces wrong actions silently.

## Existing Codebase / Prior Art

- `internal/db/migrations/` — goose migration pattern. Latest: `20260127000000_add_read_files_table.sql`
- `internal/db/sql/` — SQLC query files: `sessions.sql`, `messages.sql`, `files.sql`, `read_files.sql`, `stats.sql`
- `internal/db/` — generated SQLC code, `db.go` registers migrations
- `internal/session/session.go` — `Session` struct with `ParentSessionID`, `Todo` type with status enum. Established pattern for DB-backed entities.
- `internal/config/config.go` — `Config` struct with `Options`. Will need `Auto` section added in M003.
- `internal/pubsub/` — `Broker[T]` and `Event[T]`. Pattern for typed events.

> See `.gsd/DECISIONS.md` for all architectural and pattern decisions.

## Relevant Requirements

- R001 — Hierarchical task model in SQLite (primary)
- R002 — State derivation from DB (primary)
- R003 — Declarative dispatch rules (primary)

## Scope

### In Scope

- SQLite tables: `milestones`, `slices`, `tasks` with status, phase, ordering columns
- Goose migration file
- SQLC query files for CRUD operations
- Go domain model: `Milestone`, `Slice`, `Task` structs with phase/status enums
- DB-to-domain conversion functions
- `DeriveState()` function that queries DB and returns current state
- Dispatch rules table: Go slice of rule structs (condition → action)
- Unit tests for all components
- Integration test proving derive → dispatch produces correct sequences

### Out of Scope / Non-Goals

- Auto loop execution (M002)
- CLI commands (M002)
- Prompt templates (M002)
- Verification gates (M003)
- TUI integration (M004)
- Any runtime services — this milestone is purely data model + logic

## Technical Constraints

- `CGO_ENABLED=0` — SQLite via modernc.org/sqlite (already in deps)
- SQLC queries must follow existing patterns in `internal/db/sql/`
- Migration file naming: `YYYYMMDD000000_description.sql`
- Domain model in new `internal/auto/` package
- Status/phase enums stored as TEXT in SQLite (matching existing `TodoStatus` pattern)

## Integration Points

- `internal/db/` — generated SQLC code, migration registration
- `internal/session/` — sessions will link to milestones via parent session ID (consumed by M002)

## Open Questions

- None — schema design decisions will be made during slice planning based on what M002 needs
