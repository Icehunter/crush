# S03: Manual Stepper — crush next

**Goal:** Deliver `crush next [milestone-id]` as a top-level CLI command that executes exactly one auto-mode unit via Engine.Step() and prints what happened.
**Demo:** After this: # S03: Manual Stepper — crush next — UAT

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


## Tasks
- [x] **T01: Added crush next [milestone-id] top-level command that executes one auto-mode unit via Engine.Step() and prints progress to stderr** — Create the `crush next [milestone-id]` top-level cobra command and its test file. The command follows the exact same pattern as `autoStartCmd` in `internal/cmd/auto.go` but calls `Engine.Step()` instead of `Engine.Run()`. It subscribes to broker events to capture what unit ran, prints it to stderr, then exits.

This is a top-level command (`crush next`), NOT a subcommand of `crush auto`.

## Steps

1. Create `internal/cmd/next.go`:
   - Package `cmd`, import `auto`, `config`, `pubsub`, `cobra`, `slog`, `fmt`, `os`, `context`
   - Define `var nextCmd = &cobra.Command{...}` with `Use: "next [milestone-id]"`, `Short: "Execute the next auto-mode unit for a milestone"`, `Args: cobra.ExactArgs(1)`
   - In `RunE`: call `setupApp(cmd)`, get `cfg.Options.DataDirectory`, create `pubsub.NewBroker[auto.AutoEvent]()`, construct engine with same placeholder adapters (`&cmdStateQuerier{}`, `&cmdSessionCreator{}`, `&cmdDispatcher{}`, `&cmdStatusAdvancer{}`), subscribe to broker events in a goroutine that captures the unit description, call `eng.Step(ctx, milestoneID)`, print result to stderr (either the unit that ran or "All work complete."), return nil
   - The adapter types (`cmdStateQuerier`, `cmdSessionCreator`, `cmdDispatcher`, `cmdStatusAdvancer`) already exist in `auto.go` — reuse them directly, do NOT redefine

2. Register in `internal/cmd/root.go`:
   - Add `nextCmd` to the `rootCmd.AddCommand(...)` block (alongside `autoCmd`)

3. Create `internal/cmd/next_test.go`:
   - Test that `rootCmd` contains a "next" subcommand (iterate `rootCmd.Commands()` checking `cmd.Name() == "next"`)
   - Test that the command requires exactly 1 arg: call `nextCmd.Args(nextCmd, []string{})` expecting error, call with `[]string{"M001"}` expecting nil
   - Test `--help` output contains "milestone-id" and "next"

4. Run `gofumpt -w internal/cmd/next.go internal/cmd/next_test.go` to format

5. Verify: `go build .`, `go vet ./internal/cmd/...`, `go test ./internal/cmd/... -count=1`
  - Estimate: 30m
  - Files: internal/cmd/next.go, internal/cmd/next_test.go, internal/cmd/root.go
  - Verify: go build . && go vet ./internal/cmd/... && go test ./internal/cmd/... -count=1
