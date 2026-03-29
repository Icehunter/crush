# S03: Manual Stepper — crush next — UAT

**Milestone:** M002
**Written:** 2026-03-27T22:25:25.436Z

# UAT: S03 — Manual Stepper — crush next

**Milestone:** M002

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This is a CLI command with placeholder adapters — no runtime state to test beyond build/vet/test gates and command structure

## Preconditions

- Crush built successfully (`go build .`)
- All cmd package tests pass (`go test ./internal/cmd/... -count=1`)
- No vet errors (`go vet ./internal/cmd/...`)

## Smoke Test

Run `go build . && ./crush next --help` — should print usage showing `next [milestone-id]` and the short description.

## Test Cases

### 1. Command Registration

1. Iterate `rootCmd.Commands()` looking for a command with `Name() == "next"`
2. **Expected:** Found — `crush next` is registered as a top-level command

### 2. Argument Validation — No Args

1. Call `nextCmd.Args(nextCmd, []string{})`
2. **Expected:** Returns error (command requires exactly 1 argument)

### 3. Argument Validation — One Arg

1. Call `nextCmd.Args(nextCmd, []string{"M001"})`
2. **Expected:** Returns nil (valid)

### 4. Help Content

1. Check `nextCmd.Use` contains "milestone-id"
2. Check `nextCmd.Short` contains "next"
3. **Expected:** Both assertions pass — help text documents the command purpose

### 5. Event Output Format

1. Inspect `nextCmd.RunE` source: broker subscription prints `▶ unit` on start, `✓ unit` on complete, `✗ unit: err` on failure
2. **Expected:** Stderr output uses unicode indicators for visual clarity

### 6. All-Done Sentinel

1. Inspect `nextCmd.RunE` source: when no unit description captured, prints "All work complete."
2. **Expected:** Clean exit message when no work remains

## Edge Cases

### Missing Milestone Argument

1. Run `crush next` with no arguments
2. **Expected:** Cobra rejects with "accepts 1 arg(s), received 0"

### Too Many Arguments

1. Run `crush next M001 M002`
2. **Expected:** Cobra rejects with "accepts 1 arg(s), received 2"

## Failure Signals

- `go build .` fails — compilation error in next.go or root.go
- `go test ./internal/cmd/... -run TestNext` fails — command registration or args broken
- `go vet ./internal/cmd/...` reports issues — code quality regression

## Not Proven By This UAT

- Actual end-to-end execution of a unit (placeholder adapters return errors)
- Integration with real DB state and session creation
- Behavior under concurrent `crush auto` and `crush next` invocations

## Notes for Tester

The placeholder adapters in next.go are intentionally non-functional stubs — they mirror the ones in auto.go. Real adapter wiring is deferred to a later milestone. Testing the command structure and argument handling is sufficient for this slice.
