# S01: Core Auto Loop Engine + CLI — UAT

**Milestone:** M002
**Written:** 2026-03-27T21:52:07.772Z

# UAT: S01 — Core Auto Loop Engine + CLI

## Preconditions
- Crush built successfully (`go build .`)
- All 42 auto package tests pass (`go test ./internal/auto/... -count=1`)
- No vet errors (`go vet ./internal/auto/... ./internal/cmd/...`)

## Test Cases

### TC1: State Derivation — Single Task Dispatch
1. Create a StateQuerier mock with one active milestone, one slice in executing phase, one pending task
2. Call DeriveState()
3. **Expected:** Returns Unit with type=execute_task, correct milestone/slice/task IDs

### TC2: State Derivation — Dependency Blocking
1. Create mock with slice S02 depending on S01, S01 status=active (not completed)
2. Call DeriveState() looking at S02
3. **Expected:** S02 is skipped, returns whatever is dispatchable from S01 (or done sentinel if S01 has no work)

### TC3: State Derivation — Phase Progression
1. Create mock with slice in pre_planning phase
2. Call DeriveState() — **Expected:** research unit
3. Advance to planning phase, call again — **Expected:** plan_slice unit
4. Advance to executing phase with tasks — **Expected:** execute_task unit
5. Complete all tasks — **Expected:** summarize_slice unit
6. Advance to summarizing — **Expected:** validate_milestone unit (at milestone level)

### TC4: State Derivation — All Done
1. Create mock with all milestones completed
2. Call DeriveState()
3. **Expected:** Returns sentinel Unit with empty UnitType (done signal)

### TC5: Engine Run — Full Lifecycle
1. Seed mock querier with 6-unit sequence: research → plan → execute × 2 → summarize → validate
2. Call Engine.Run()
3. **Expected:** All 6 units dispatched in order, events published (6 started + 6 completed), engine reaches idle state

### TC6: Engine Step — Single Unit
1. Seed mock querier with multiple units
2. Call Engine.Step()
3. **Expected:** Exactly 1 unit dispatched, then returns

### TC7: Engine Pause — Finish Then Stop
1. Start Engine.Run() in goroutine
2. After first unit completes, call Engine.Pause()
3. **Expected:** Current unit finishes, engine enters EnginePaused state, no more units dispatched

### TC8: Lock File — Prevents Concurrent Instances
1. Acquire lock file
2. Attempt second acquire from different Engine instance
3. **Expected:** Second acquire fails with error containing "already running"

### TC9: Lock File — Stale PID Reclamation
1. Create lock file with PID of a dead process
2. Attempt acquire
3. **Expected:** Stale lock detected, reclaimed, new lock acquired

### TC10: CLI Registration
1. Run `crush auto --help`
2. **Expected:** Shows start, pause, stop, status subcommands

### TC11: Prompt Templates — All Unit Types Render
1. Call BuildPrompt() for each of the 5 unit types with valid PromptContext
2. **Expected:** Each returns non-empty string without error, contains expected sections

## Edge Cases

### EC1: Empty Database
- DeriveState with no milestones → returns done sentinel

### EC2: Lock File Race — Unparseable File
- Lock file exists but is empty/corrupted → treated as "held", not stale

### EC3: Child Session Creation
- Run 2 units → verify 1 parent session + 2 child sessions created
