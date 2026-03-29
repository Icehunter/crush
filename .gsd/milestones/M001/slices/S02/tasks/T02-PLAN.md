---
estimated_steps: 12
estimated_files: 2
skills_used: []
---

# T02: Add tests for enum validity and round-trip conversion

Write tests in `internal/auto/` proving correctness:

1. **`status_test.go`** — Test that every Status and Phase constant passes IsValid(). Test that invalid strings like `"bogus"` fail IsValid(). Test that all constant values exactly match DB schema defaults (`pending`, `pre_planning`, etc.).

2. **`model_test.go`** — Round-trip conversion tests for each entity type:
   - Create a domain Milestone → call ToDBCreate() → construct a db.Milestone from the params (simulate what DB returns) → call MilestoneFromDB() → compare fields to original.
   - Same pattern for Slice and Task.
   - Test sql.NullString edge cases: empty DependsOn converts to NullString{Valid:false}, non-empty DependsOn round-trips correctly. Same for Task.Description.
   - Test that Status/Phase fields preserve their typed values through conversion.

**Key constraints:**
- Use `require` from testify (project standard).
- Use `t.Parallel()` on all test functions.
- Run `gofumpt -w .` after writing.
- Verify existing DB tests still pass: `go test ./internal/db/ -run TestAuto -v -count=1`.

## Inputs

- ``internal/auto/status.go` — Status and Phase enums to test`
- ``internal/auto/milestone.go` — MilestoneFromDB and ToDBCreate to test`
- ``internal/auto/slice.go` — SliceFromDB and ToDBCreate to test`
- ``internal/auto/task.go` — TaskFromDB and ToDBCreate to test`
- ``internal/db/models.go` — DB struct types needed to construct test fixtures`

## Expected Output

- ``internal/auto/status_test.go` — Enum validity tests for Status and Phase`
- ``internal/auto/model_test.go` — Round-trip conversion tests for Milestone, Slice, Task with NullString edge cases`

## Verification

go test ./internal/auto/ -v -count=1 && go test ./internal/db/ -run TestAuto -v -count=1
