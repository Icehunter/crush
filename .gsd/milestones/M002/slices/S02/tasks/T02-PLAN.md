---
estimated_steps: 43
estimated_files: 4
skills_used: []
---

# T02: Create init prompt template, RunInit() function, and `crush auto init` CLI command

## Description

Wire the init flow end-to-end: a Go template prompt that instructs the LLM to decompose a vision using the planning tools, a `RunInit()` function that constructs a SessionAgent with those tools and dispatches, and a `crush auto init "vision"` cobra command.

The research recommends approach (A): bypass the Coordinator entirely, construct a SessionAgent directly with `agent.NewSessionAgent()`, set the init prompt as system prompt, set the 3 planning tools, and call Run(). This avoids polluting the standard tool set.

However, constructing a SessionAgent requires model objects (`agent.Model`). The Coordinator's `buildAgentModels()` is private. The simplest path: `RunInit()` accepts models as parameters, and the CLI command layer gets them from `app.AgentCoordinator.Model()` (which returns the main model) or from `app.AgentCoordinator.UpdateModels()` then accessing the coordinator's public Model() method. Since init is a planning task, using the main model is acceptable.

## Steps

1. Create `internal/auto/templates/init.md.tpl` — the planning prompt template:
   - Instructs the LLM: "You are a project planner. Decompose the following vision into milestones, slices, and tasks."
   - Specifies ID format conventions: M001, S01, T01 etc.
   - Specifies status/phase conventions: first milestone active, others pending; all slices pending/pre_planning; all tasks pending/pre_planning
   - Specifies sort_order: sequential integers starting from 1
   - Specifies depends_on format: comma-separated slice IDs or empty
   - Template receives `InitPromptContext` with Vision and WorkingDir fields
   - Rendered via `BuildInitPrompt()` function

2. Add `BuildInitPrompt()` to `internal/auto/prompts.go`:
   - Define `InitPromptContext` struct with Vision and WorkingDir
   - Parse and execute `templates/init.md.tpl` with the context
   - Similar to existing `BuildPrompt()` but with different context type

3. Create `internal/auto/init.go` with `RunInit()` function:
   - Signature: `func RunInit(ctx context.Context, cfg InitConfig) error`
   - `InitConfig` holds: vision string, db.Queries, session.Service, agent model, slog.Logger
   - Creates a top-level session via session.Service
   - Auto-approves the session for non-interactive use (permission.Service if available, or pass through)
   - Builds the init prompt via BuildInitPrompt()
   - Constructs planning tools via NewCreateMilestoneTool/NewCreateSliceTool/NewCreateTaskTool
   - Creates SessionAgent via agent.NewSessionAgent() with planning tools and init prompt
   - Calls agent.Run() with the vision as the user prompt
   - Returns error if dispatch fails

4. Add `autoInitCmd` to `internal/cmd/auto.go`:
   - `crush auto init "Build a REST API with auth"` — takes vision as positional arg
   - Calls setupApp(), gets model from app.AgentCoordinator.Model()
   - Calls auto.RunInit() with the vision and wired dependencies
   - Register in init() alongside existing subcommands

5. Verify: `go build .` compiles, `go vet ./internal/auto/... ./internal/cmd/...` clean

## Must-Haves

- [ ] init.md.tpl template renders correctly with vision text
- [ ] BuildInitPrompt() returns non-empty string without error
- [ ] RunInit() constructs SessionAgent with planning tools only (not standard tools)
- [ ] crush auto init registered as cobra subcommand
- [ ] Project compiles and vets clean

## Verification

- `go build .` compiles
- `go test ./internal/auto/... -count=1` — all tests pass (including prompt template test for init)
- `go vet ./internal/auto/... ./internal/cmd/...` — clean

## Inputs

- ``internal/auto/init_tools.go` — planning tool constructors from T01`
- ``internal/auto/prompts.go` — existing BuildPrompt pattern to follow`
- ``internal/auto/templates/research.md.tpl` — example template for style reference`
- ``internal/agent/agent.go` — NewSessionAgent, SessionAgentOptions, SessionAgentCall`
- ``internal/agent/coordinator.go` — Coordinator.Model() for getting model objects`
- ``internal/cmd/auto.go` — existing autoCmd to add init subcommand to`
- ``internal/app/app.go` — RunNonInteractive pattern, setupApp, AgentCoordinator access`

## Expected Output

- ``internal/auto/templates/init.md.tpl` — planning prompt template`
- ``internal/auto/prompts.go` — extended with BuildInitPrompt() and InitPromptContext`
- ``internal/auto/init.go` — RunInit() function with InitConfig`
- ``internal/cmd/auto.go` — extended with autoInitCmd cobra command`

## Verification

go build . && go test ./internal/auto/... -count=1 && go vet ./internal/auto/... ./internal/cmd/...
