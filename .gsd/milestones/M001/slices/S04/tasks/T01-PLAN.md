---
estimated_steps: 26
estimated_files: 1
skills_used: []
---

# T01: Create dispatch.go with Rule struct, rules table, and Dispatch function

Create `internal/auto/dispatch.go` implementing the declarative dispatch rules table.

## Steps

1. Create `internal/auto/dispatch.go` in the `auto` package.
2. Define the `Rule` struct with three fields:
   - `Name string` — human-readable rule name for debugging/testing
   - `Match func(*State) bool` — condition function that inspects State fields
   - `Action Action` — the action to return when Match is true
3. Define an unexported `rules` variable (`var rules = []Rule{...}`) containing rules ordered most-specific first:
   - `execute-task`: Match when `State.Action == ActionExecuteTask` (task ready for execution)
   - `plan-slice`: Match when `State.Action == ActionPlanSlice`
   - `plan-milestone`: Match when `State.Action == ActionPlanMilestone`
   - `complete-slice`: Match when `State.Action == ActionCompleteSlice`
   - `complete-milestone`: Match when `State.Action == ActionCompleteMilestone`
   - `none`: Match always returns true (catch-all fallback), Action is ActionNone
4. Implement exported `Dispatch(state *State) Action` function:
   - If `state == nil`, return `ActionNone` immediately (nil-safety)
   - Walk `rules` top-down, return `rule.Action` for the first rule where `rule.Match(state)` returns true
   - If no rule matches (should not happen with catch-all), return `ActionNone`
5. Implement exported `Rules() []Rule` function that returns a copy of the rules slice for introspection/testing.
6. Run `gofumpt -w internal/auto/dispatch.go` to format.
7. Verify: `go build ./internal/auto/...` and `go vet ./internal/auto/...` both exit 0.

## Constraints
- Rules must be a Go slice of Rule structs (R003), NOT a switch statement
- Match functions must inspect State fields (not just forward State.Action blindly) — use `state.Action == X` in Match closures, which reads the State struct field. This makes rules extensible for future conditions (e.g., checking Phase).
- Follow existing package style: typed string constants, comments ending in periods, gofumpt formatting
- Keep rules as `var` not `const` — slice of structs cannot be const in Go

## Inputs

- ``internal/auto/state.go` — State struct and Action constants (ActionNone, ActionPlanMilestone, ActionPlanSlice, ActionExecuteTask, ActionCompleteSlice, ActionCompleteMilestone)`
- ``internal/auto/status.go` — Status and Phase enums used in condition matching`

## Expected Output

- ``internal/auto/dispatch.go` — Rule struct, rules table (6 rules), Dispatch(*State) Action function, Rules() []Rule function`

## Verification

go build ./internal/auto/... && go vet ./internal/auto/... && grep -q 'func Dispatch' internal/auto/dispatch.go && grep -q 'type Rule struct' internal/auto/dispatch.go && grep -q 'func Rules' internal/auto/dispatch.go
