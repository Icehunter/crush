---
estimated_steps: 27
estimated_files: 4
skills_used: []
---

# T01: Consolidate M002 auto package and add AutoConfig struct to crush.json

## Description

This task brings the M002 engine code into the M003 worktree and adds the `AutoConfig` struct to the config system. Without the auto package files, nothing in this slice compiles. The config addition delivers R014.

## Steps

1. Copy all `internal/auto/` files from the M002 worktree (`/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/`) into the M003 worktree at `internal/auto/`. This includes: `engine.go`, `engine_test.go`, `engine_integration_test.go`, `events.go`, `init.go`, `init_test.go`, `init_tools.go`, `init_tools_test.go`, `lock.go`, `lock_test.go`, `milestone.go`, `prompts.go`, `prompts_test.go`, `slice.go`, `state.go`, `state_test.go`, `status.go`, `task.go`, `unit.go`, and the `templates/` directory with all `.md.tpl` files.
2. Run `go build ./internal/auto/` to verify the consolidated files compile in the M003 worktree. Fix any import issues.
3. Add an `AutoConfig` struct to `internal/config/config.go`:
   ```go
   type AutoConfig struct {
       VerificationCommands []string `json:"verification_commands,omitempty" jsonschema:"description=Shell commands to run after each task execution for verification"`
       BudgetCeiling        float64  `json:"budget_ceiling,omitempty" jsonschema:"description=Maximum dollar cost before auto-mode pauses"`
       StuckThreshold       int      `json:"stuck_threshold,omitempty" jsonschema:"description=Number of consecutive failures before stuck detection triggers,default=5"`
       WorktreeMode         string   `json:"worktree_mode,omitempty" jsonschema:"description=Git worktree isolation mode for auto-mode execution"`
   }
   ```
4. Add an `Auto *AutoConfig` field to the `Config` struct: `Auto *AutoConfig \`json:"auto,omitempty\"\``
5. Add a test in `internal/config/load_test.go` (or a new `internal/config/auto_test.go`) that round-trips a crush.json with an `auto` section through `loadFromBytes` and asserts the `VerificationCommands` field is populated.
6. Run `go vet ./internal/auto/ ./internal/config/` and `go test ./internal/config/ -run TestAutoConfig -count=1`.
7. Format with `gofumpt -w internal/config/config.go internal/config/auto_test.go`.

## Must-Haves

- [ ] All M002 auto package files exist in M003 worktree at `internal/auto/`
- [ ] `go build ./internal/auto/` succeeds
- [ ] `AutoConfig` struct with all four fields exists on `Config`
- [ ] Config parsing test passes for `auto.verification_commands`

## Verification

- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go build ./internal/auto/`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go test ./internal/config/ -run TestAutoConfig -count=1 -v`
- `cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go vet ./internal/auto/ ./internal/config/`

## Inputs

- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/engine.go` — M002 engine source to consolidate`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/events.go` — M002 event types`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/state.go` — M002 state derivation`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/unit.go` — M002 unit types`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/lock.go` — M002 lock file`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/prompts.go` — M002 prompt builder`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/init.go` — M002 init logic`
- ``/Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M002/internal/auto/init_tools.go` — M002 init tools`
- ``internal/config/config.go` — existing Config struct to extend`

## Expected Output

- ``internal/auto/engine.go` — consolidated engine from M002`
- ``internal/auto/events.go` — consolidated events from M002`
- ``internal/auto/state.go` — consolidated state derivation from M002`
- ``internal/auto/unit.go` — consolidated unit types from M002`
- ``internal/auto/lock.go` — consolidated lock file from M002`
- ``internal/auto/prompts.go` — consolidated prompts from M002`
- ``internal/config/config.go` — extended with AutoConfig struct and Auto field`
- ``internal/config/auto_test.go` — config parsing test for auto section`

## Verification

cd /Volumes/Engineering/Icehunter/crush/.gsd/worktrees/M003 && go build ./internal/auto/ && go test ./internal/config/ -run TestAutoConfig -count=1 -v && go vet ./internal/auto/ ./internal/config/
