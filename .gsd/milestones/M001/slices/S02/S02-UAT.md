# S02: Domain Model + Status Enums — UAT

**Milestone:** M001
**Written:** 2026-03-27T18:44:07.902Z

# S02: Domain Model + Status Enums — UAT

**Milestone:** M001
**Written:** 2026-03-27

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This slice produces Go source code and tests with no runtime behavior — compilation and test execution prove correctness

## Preconditions

- Go toolchain installed (go 1.24+)
- Working directory is the crush project root
- S01 DB schema and SQLC-generated code already in place

## Smoke Test

Run `go test ./internal/auto/ -v -count=1` — all 16 tests should pass.

## Test Cases

### 1. Package Compiles Clean

1. Run `go build ./internal/auto/...`
2. Run `go vet ./internal/auto/...`
3. **Expected:** Both exit code 0, no output

### 2. Status Enum Constants Are Valid

1. Run `go test ./internal/auto/ -run TestStatusConstants_AreValid -v`
2. **Expected:** PASS — all four Status constants (pending, active, completed, blocked) pass IsValid()

### 3. Phase Enum Constants Are Valid

1. Run `go test ./internal/auto/ -run TestPhaseConstants_AreValid -v`
2. **Expected:** PASS — all seven Phase constants pass IsValid()

### 4. Status Values Match DB Schema

1. Run `go test ./internal/auto/ -run TestStatusConstants_MatchDBValues -v`
2. **Expected:** PASS — string values match case-sensitive DB defaults

### 5. Phase Values Match DB Schema

1. Run `go test ./internal/auto/ -run TestPhaseConstants_MatchDBValues -v`
2. **Expected:** PASS — string values match case-sensitive DB defaults

### 6. Invalid Strings Rejected

1. Run `go test ./internal/auto/ -run TestStatus_InvalidString -v`
2. Run `go test ./internal/auto/ -run TestPhase_InvalidString -v`
3. **Expected:** Both PASS — "bogus" fails IsValid() for both types

### 7. Milestone Round-Trip Conversion

1. Run `go test ./internal/auto/ -run TestMilestone_RoundTrip -v`
2. **Expected:** PASS — domain Milestone → ToDBCreate() → simulated DB row → MilestoneFromDB() preserves all fields

### 8. Slice Round-Trip with NullString Edge Cases

1. Run `go test ./internal/auto/ -run TestSlice -v`
2. **Expected:** All 3 slice tests PASS — round-trip works, empty DependsOn → NullString{Valid:false}, non-empty DependsOn round-trips

### 9. Task Round-Trip with NullString Edge Cases

1. Run `go test ./internal/auto/ -run TestTask -v`
2. **Expected:** All 3 task tests PASS — round-trip works, empty Description → NullString{Valid:false}, non-empty Description round-trips

### 10. No Regressions in DB Tests

1. Run `go test ./internal/db/ -run TestAuto -v -count=1`
2. **Expected:** All 4 DB tests PASS

## Edge Cases

### Invalid Enum Strings

1. Call `Status("bogus").IsValid()` and `Phase("bogus").IsValid()`
2. **Expected:** Both return false

### Empty NullString Fields

1. Create Slice with empty DependsOn, convert via ToDBCreate()
2. **Expected:** db.CreateSliceParams.DependsOn has Valid=false

### NullString With Valid=true But Empty String

1. Run `go test ./internal/auto/ -run TestNullString_ValidTrueEmptyString -v`
2. **Expected:** PASS — converts to empty string (not panic or error)

## Failure Signals

- `go build ./internal/auto/...` fails with type errors
- Any of the 16 tests fail
- DB tests regress after adding internal/auto/

## Not Proven By This UAT

- Runtime integration of domain structs with actual DB operations (S03+ scope)
- State derivation logic using these enums (S03 scope)
- Dispatch rules consuming Status/Phase values (S04 scope)

## Notes for Tester

- All tests use t.Parallel() — safe to run with -race flag
- Tests only exercise in-memory conversion, no database needed
- NullString helper functions are unexported but tested indirectly through model round-trip tests
