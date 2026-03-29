# S01: DB Schema + SQLC Queries — UAT

**Milestone:** M001
**Written:** 2026-03-27T18:36:27.635Z

# S01: DB Schema + SQLC Queries — UAT

**Milestone:** M001
**Written:** 2026-03-27

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This slice produces schema, generated code, and tests — no runtime behavior to verify beyond compilation and test execution

## Preconditions

- Go toolchain installed (go 1.24+)
- sqlc installed (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- Working directory is the crush project root

## Smoke Test

Run `go test ./internal/db/ -run TestAuto -v -count=1` — all 4 tests should pass.

## Test Cases

### 1. SQLC Generation Produces Clean Output

1. Run `sqlc generate` from project root
2. **Expected:** Exit code 0, no warnings or errors

### 2. Generated Code Compiles

1. Run `go build ./internal/db/...`
2. **Expected:** Exit code 0

### 3. Go Vet Passes Project-Wide

1. Run `go vet ./...`
2. **Expected:** Exit code 0, no diagnostics

### 4. Milestone CRUD Lifecycle

1. Run `go test ./internal/db/ -run TestAutoMilestones -v`
2. **Expected:** Test creates milestone, reads it back, updates status to 'active', updates phase to 'planning', deletes it, confirms list is empty. PASS.

### 5. Slice CRUD with Ordering

1. Run `go test ./internal/db/ -run TestAutoSlices -v`
2. **Expected:** Test creates milestone, creates two slices with different sort_order, lists by milestone confirming order, updates status/phase, deletes one. PASS.

### 6. Task CRUD with Multi-Level Queries

1. Run `go test ./internal/db/ -run TestAutoTasks -v`
2. **Expected:** Test creates milestone → slice → two tasks, lists by slice (ordered), lists by milestone (both appear), updates status/phase, deletes one. PASS.

### 7. Cascade Delete

1. Run `go test ./internal/db/ -run TestAutoCascadeDelete -v`
2. **Expected:** Test creates milestone → slice → task, deletes milestone, confirms GetSlice and GetTask both return sql.ErrNoRows. PASS.

### 8. Model Structs Exist

1. Run `grep 'type Milestone struct' internal/db/models.go`
2. Run `grep 'type Slice struct' internal/db/models.go`
3. Run `grep 'type Task struct' internal/db/models.go`
4. **Expected:** All three grep commands exit 0

## Edge Cases

### Empty Database Queries

1. Run TestAutoMilestones — it verifies ListMilestones returns empty after delete
2. **Expected:** No error on listing empty tables

### Foreign Key Enforcement

1. TestAutoCascadeDelete verifies ON DELETE CASCADE works correctly
2. **Expected:** Child records are automatically deleted when parent is removed

## Failure Signals

- `sqlc generate` produces errors about unknown columns or table references
- `go build ./internal/db/...` fails with type errors in generated code
- `go vet ./...` reports lock-by-value or other structural issues
- Any TestAuto* test fails

## Not Proven By This UAT

- Runtime migration behavior on existing production databases with data
- Performance under concurrent access
- Integration with the domain model layer (S02 scope)
- State derivation logic (S03 scope)

## Notes for Tester

- Tests use in-memory SQLite — no database files are created on disk
- The csync/maps.go fix is a pre-existing issue unrelated to this slice's schema work, but was required to pass go vet
