---
estimated_steps: 15
estimated_files: 4
skills_used: []
---

# T01: Create internal/auto package with status enums and domain models

Create the `internal/auto/` package with four files:

1. **`status.go`** — Define `Status` and `Phase` as typed `string` types following the `session.TodoStatus` pattern. Constants:
   - Status: `StatusPending`, `StatusActive`, `StatusCompleted`, `StatusBlocked`
   - Phase: `PhasePre Planning`, `PhasePlanning`, `PhaseResearching`, `PhaseExecuting`, `PhaseSummarizing`, `PhaseValidating`, `PhaseCompleted`
   - Add `IsValid()` methods on both types that check against the known constants.

2. **`milestone.go`** — `Milestone` domain struct with typed `Status` and `Phase` fields, plain `string` for other fields, `int64` for timestamps. Export `MilestoneFromDB(db.Milestone) Milestone` and `(m Milestone) ToDBCreate() db.CreateMilestoneParams` conversion functions.

3. **`slice.go`** — `Slice` domain struct. `DependsOn` is plain `string` (empty string when DB has NULL). Export `SliceFromDB(db.Slice) Slice` and `(s Slice) ToDBCreate() db.CreateSliceParams`.

4. **`task.go`** — `Task` domain struct. `Description` is plain `string`. Export `TaskFromDB(db.Task) Task` and `(t Task) ToDBCreate() db.CreateTaskParams`.

**Key constraints:**
- Status/phase string values must exactly match what the DB stores (case-sensitive): `pending`, `active`, `completed`, `blocked`, `pre_planning`, `planning`, `researching`, `executing`, `summarizing`, `validating`, `completed`.
- `sql.NullString` → plain `string`: when `Valid` is false, use empty string.
- Plain `string` → `sql.NullString`: when string is empty, set `Valid = false`.
- JSON tags use `snake_case` per project style.
- Import only `internal/db` and stdlib. No other dependencies.
- Comments start with capital letters, end with periods. Format with `gofumpt -w .`.

## Inputs

- ``internal/db/models.go` — SQLC-generated Milestone, Slice, Task structs with string status/phase and sql.NullString optional fields`
- ``internal/db/milestones.sql.go` — CreateMilestoneParams struct definition`
- ``internal/db/slices.sql.go` — CreateSliceParams struct definition`
- ``internal/db/tasks.sql.go` — CreateTaskParams struct definition`
- ``internal/session/session.go` — TodoStatus typed string enum pattern to follow (lines 18-26)`

## Expected Output

- ``internal/auto/status.go` — Status and Phase typed string enums with IsValid() methods`
- ``internal/auto/milestone.go` — Milestone domain struct with MilestoneFromDB and ToDBCreate`
- ``internal/auto/slice.go` — Slice domain struct with SliceFromDB and ToDBCreate`
- ``internal/auto/task.go` — Task domain struct with TaskFromDB and ToDBCreate`

## Verification

go build ./internal/auto/... && go vet ./internal/auto/...
