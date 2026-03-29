---
estimated_steps: 16
estimated_files: 3
skills_used: []
---

# T01: Implement crush next command with tests

Create the `crush next [milestone-id]` top-level cobra command and its test file. The command follows the exact same pattern as `autoStartCmd` in `internal/cmd/auto.go` but calls `Engine.Step()` instead of `Engine.Run()`. It subscribes to broker events to capture what unit ran, prints it to stderr, then exits.

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

## Inputs

- ``internal/cmd/auto.go` — adapter types (cmdStateQuerier, cmdSessionCreator, cmdDispatcher, cmdStatusAdvancer) and autoStartCmd pattern to follow`
- ``internal/cmd/root.go` — rootCmd.AddCommand() block where nextCmd must be registered`
- ``internal/auto/engine.go` — Engine.Step() method signature`

## Expected Output

- ``internal/cmd/next.go` — new cobra command for crush next`
- ``internal/cmd/next_test.go` — tests for command registration, arg validation, help output`
- ``internal/cmd/root.go` — nextCmd added to AddCommand block`

## Verification

go build . && go vet ./internal/cmd/... && go test ./internal/cmd/... -count=1
