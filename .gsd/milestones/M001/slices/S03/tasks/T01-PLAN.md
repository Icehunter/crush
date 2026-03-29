---
estimated_steps: 47
estimated_files: 1
skills_used: []
---

# T01: Implement State struct, Action enum, and DeriveState() function

## Description

Create `internal/auto/state.go` containing the `State` struct, `Action` enum, and `DeriveState()` function. This is the core deliverable for R002 — the function that answers "what should auto-mode work on next?" by walking the milestone → slice → task hierarchy in the DB.

## Steps

1. Create `internal/auto/state.go` with package declaration and imports (`context`, `fmt`, `github.com/charmbracelet/crush/internal/db`).
2. Define the `Action` type as `type Action string` with constants:
   - `ActionNone` — nothing to do (empty DB or all completed)
   - `ActionPlanMilestone` — active/pending milestone needs planning
   - `ActionPlanSlice` — active/pending slice needs planning
   - `ActionExecuteTask` — active/pending task ready for execution
   - `ActionCompleteSlice` — all tasks in slice completed, slice needs completion
   - `ActionCompleteMilestone` — all slices in milestone completed, milestone needs completion
3. Define the `State` struct:
   ```go
   type State struct {
       Action    Action     `json:"action"`
       Milestone *Milestone `json:"milestone,omitempty"`
       Slice     *Slice     `json:"slice,omitempty"`
       Task      *Task      `json:"task,omitempty"`
   }
   ```
4. Implement `DeriveState(ctx context.Context, q *db.Queries) (*State, error)` with this algorithm:
   - Call `q.ListMilestones(ctx)`. Find first with `status=active`. If none, find first `pending`. If none, return `&State{Action: ActionNone}`.
   - Convert to domain `Milestone` via `MilestoneFromDB()`.
   - If milestone phase is `pre_planning` or `planning`, return `ActionPlanMilestone`.
   - Call `q.ListSlicesByMilestone(ctx, milestone.ID)`. Walk in sort_order:
     - Skip `completed` slices.
     - For each non-completed slice: if `DependsOn` is non-empty, call `q.GetSlice(ctx, dependsOn)` — if error or status != `completed`, skip (dependency not met). First actionable slice wins.
   - If no actionable slice found: check if ALL slices are completed → `ActionCompleteMilestone`. Otherwise `ActionNone` (blocked).
   - If actionable slice found and its phase is `pre_planning` or `planning`, return `ActionPlanSlice`.
   - Call `q.ListTasksBySlice(ctx, slice.ID)`. Find first `active` task. If none, find first `pending`.
   - If no actionable task: check if ALL tasks completed → `ActionCompleteSlice`. Otherwise return with slice-level info.
   - Return `ActionExecuteTask` with the active task.
5. Run `gofumpt -w internal/auto/state.go` to format.
6. Run `go build ./internal/auto/...` and `go vet ./internal/auto/...` to verify clean compilation.

## Must-Haves

- [ ] `Action` type with 6 named constants
- [ ] `State` struct with Action, optional Milestone/Slice/Task pointers
- [ ] `DeriveState()` accepts `context.Context` and `*db.Queries`, returns `(*State, error)`
- [ ] Dependency checking: non-empty `DependsOn` checked against actual slice status
- [ ] Missing/invalid dependency ID treated as "not met" (skip, don't panic)
- [ ] Walk respects sort_order — no skipping ahead to find active items out of order

## Verification

- `go build ./internal/auto/...` exits 0
- `go vet ./internal/auto/...` exits 0
- `grep -q 'func DeriveState' internal/auto/state.go` exits 0
- `grep -q 'type Action string' internal/auto/state.go` exits 0
- `grep -q 'type State struct' internal/auto/state.go` exits 0

## Inputs

- ``internal/auto/status.go` — Status and Phase enums (StatusPending, StatusActive, StatusCompleted, StatusBlocked; PhasePrePlanning, PhasePlanning, etc.)`
- ``internal/auto/milestone.go` — Milestone domain struct and MilestoneFromDB() converter`
- ``internal/auto/slice.go` — Slice domain struct with DependsOn field and SliceFromDB() converter`
- ``internal/auto/task.go` — Task domain struct and TaskFromDB() converter`
- ``internal/db/milestones.sql.go` — ListMilestones() returning []Milestone ordered by created_at DESC`
- ``internal/db/slices.sql.go` — ListSlicesByMilestone() returning []Slice ordered by sort_order, GetSlice() for dependency lookup`
- ``internal/db/tasks.sql.go` — ListTasksBySlice() returning []Task ordered by sort_order`

## Expected Output

- ``internal/auto/state.go` — State struct, Action enum, and DeriveState() function`

## Verification

go build ./internal/auto/... && go vet ./internal/auto/... && grep -q 'func DeriveState' internal/auto/state.go
