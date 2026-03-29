# S02: Interactive Planning — crush auto init — Research

**Date:** 2026-03-27
**Status:** Complete

## Summary

S02 implements `crush auto init`, the interactive planning flow where a user describes a vision and Crush decomposes it into milestones/slices/tasks in SQLite. This is a **targeted research** task — the patterns are well-established in the codebase (`crush run` for non-interactive dispatch, `internal/auto/prompts.go` for template rendering, `internal/auto/engine.go` for session creation) and the work is primarily wiring them together with a new prompt template and a new CLI command.

The core challenge is that `auto init` needs the LLM to **write structured data to the DB** — milestones, slices, and tasks. Two viable approaches exist: (1) give the LLM dedicated tools (`create_milestone`, `create_slice`, `create_task`) that call `db.Queries.CreateMilestone/Slice/Task` directly, or (2) use the LLM's text output with structured parsing (JSON/YAML block). Approach (1) is strongly preferred because it uses the existing `fantasy.AgentTool` pattern, is self-correcting (the LLM sees tool errors and retries), and produces the exact DB records needed without brittle parsing.

The flow is: user runs `crush auto init` → creates a session → dispatches a planning prompt via `Coordinator.Run()` → the LLM uses `create_milestone`, `create_slice`, `create_task` tools to populate the DB → the first milestone is set to `active` status → `crush auto start` can pick up from there. This reuses the existing `RunNonInteractive` pattern almost verbatim.

## Recommendation

**Build `crush auto init` as a non-interactive CLI command** that dispatches a planning prompt with 3 custom tools (`create_milestone`, `create_slice`, `create_task`). The command takes a vision string as argument (or reads from stdin), creates a session, builds the planning prompt from a new `init.md.tpl` template, and dispatches via `app.AgentCoordinator.Run()`.

The 3 planning tools should live in `internal/auto/inittools.go` (or `internal/auto/init_tools.go`) and implement `fantasy.AgentTool`. They wrap `db.Queries` calls directly. The init command needs DB access — it gets this by calling `db.New(conn)` from the `setupApp()` flow (same pattern as `internal/cmd/session.go` line 121 and `internal/cmd/stats.go` line 179).

**However**, there's a key design question: `Coordinator.Run()` uses a fixed set of tools built in `buildAgent()` → `buildTools()`. The init flow needs **different tools** (the planning tools, not the standard coder tools). Two options:

- **(A) Bypass Coordinator entirely** — Construct a `SessionAgent` directly via `agent.NewSessionAgent()`, set a custom system prompt, set the 3 planning tools, and call `Run()`. This is the approach the M002 research recommended for auto-mode in general. It avoids modifying the Coordinator contract.
- **(B) Add planning tools to the standard tool set** — Register the 3 planning tools alongside bash/edit/view/etc. The LLM only uses them when the init prompt instructs it to. This is simpler but pollutes the standard tool set.

**Recommend (A)** — bypass the Coordinator for init. The init flow is a special-purpose dispatch with a unique prompt and unique tools. It should construct a `SessionAgent` directly. This is well-supported: `NewSessionAgent()`, `SetSystemPrompt()`, `SetTools()`, and `SetModels()` are all public API on the `SessionAgent` interface.

## Implementation Landscape

### Key Files

- `internal/cmd/auto.go` — Add `autoInitCmd` as a new subcommand under `autoCmd`. Pattern: `autoStartCmd` in the same file. Needs to accept a vision string as args or stdin.
- `internal/auto/init.go` (new) — Core init logic: `RunInit(ctx, app, vision) error`. Creates session, builds prompt, constructs `SessionAgent` with planning tools, dispatches. Separated from cmd for testability.
- `internal/auto/init_tools.go` (new) — 3 `fantasy.AgentTool` implementations: `NewCreateMilestoneTool(q)`, `NewCreateSliceTool(q)`, `NewCreateTaskTool(q)`. Each takes `*db.Queries` and wraps the corresponding `CreateMilestone/Slice/Task` SQLC call. Uses `fantasy.NewAgentTool()` pattern from `internal/agent/tools/bash.go`.
- `internal/auto/templates/init.md.tpl` (new) — Planning prompt template. Instructs the LLM to decompose the vision into milestones → slices → tasks using the provided tools. Needs to specify the expected structure (IDs, titles, status/phase conventions, sort_order, depends_on format).
- `internal/auto/prompts.go` — Add `UnitInit` type constant and register the init template in `templateNames` map. Or, since init is not a standard unit type in the engine loop, add a separate `BuildInitPrompt(vision string) (string, error)` function.
- `internal/auto/init_test.go` (new) — Tests: tool parameter validation, DB writes, end-to-end init with mock agent.
- `internal/agent/agent.go` — `NewSessionAgent()` and `SessionAgent` interface — already public, no changes needed.
- `internal/agent/coordinator.go` — `buildAgentModels()` is private (line 540). The init flow needs model construction. Options: (a) export `BuildAgentModels()`, (b) accept models as params to `RunInit()`, (c) have the cmd layer call `coordinator.Model()` to get the current model and pass it through. Recommend (c) — `Coordinator.Model()` (line 985) already returns the main model.
- `internal/session/session.go` — `service.Create(ctx, title)` creates a top-level session. Used by init to create the planning session.
- `internal/db/sql/milestones.sql`, `slices.sql`, `tasks.sql` — SQLC queries already exist for Create operations. No new SQL needed.

### Build Order

1. **Init tools** (`init_tools.go`) — Create the 3 planning tools wrapping DB queries. Test them with in-memory SQLite. This is the novel component and should be proven first.
2. **Init prompt template** (`templates/init.md.tpl`) — The planning prompt that instructs the LLM how to decompose a vision. Add `BuildInitPrompt()` to `prompts.go`.
3. **Init logic** (`init.go`) — The `RunInit()` function that wires session creation, prompt building, agent construction, and dispatch. Requires init tools and prompt.
4. **CLI command** (`auto.go`) — Wire `crush auto init "vision"` cobra command calling `RunInit()`. Register in `init()`.
5. **Integration test** — Seed a vision, run init with a mock dispatcher, verify milestones/slices/tasks in DB.

### Verification Approach

- `go test ./internal/auto/... -count=1` — All existing 42 tests pass + new init tool tests + init integration test
- `go vet ./internal/auto/... ./internal/cmd/...` — Clean
- `go build .` — Project compiles
- Manual: `crush auto init "Build a REST API with auth"` produces records in SQLite (requires configured LLM provider, so mainly tested via mocks)

### Constraints

- **`buildAgentModels()` is private** — Cannot call from `internal/auto`. Must get model info through public `Coordinator` interface or accept it as a parameter. `Coordinator.Model()` returns the main model only — for init, planning tier is most appropriate. The `Coordinator.UpdateModels()` → `Coordinator` doesn't expose background/planning models directly. May need to add a method or pass models from the cmd layer.
- **Tool description files** — Standard tools have `.md` description files (e.g., `bash.md`). The planning tools should follow this pattern with embedded descriptions.
- **Session auto-approve** — Init runs non-interactively, so the session needs `AutoApproveSession()`. Follow `RunNonInteractive` pattern.
- **DB migration** — The milestones/slices/tasks tables already exist from M001's migration. No new schema needed.
- **First milestone should be `active`** — After init creates the structure, the first milestone should have `status=active` so `DeriveState()` picks it up. The init prompt should instruct the LLM to set this, or the `create_milestone` tool should default to `active` for the first one.

### Common Pitfalls

- **Tool parameter validation** — The planning tools must validate required fields (ID, title, milestone_id for slices, slice_id for tasks) and return clear errors so the LLM can self-correct.
- **Sort order** — Slices and tasks need meaningful `sort_order` values. The tools should accept sort_order as a parameter, and the prompt should instruct the LLM to set them sequentially.
- **Depends_on format** — Comma-separated slice IDs. The prompt must specify this format. The tool should validate that referenced IDs exist.
- **ID format** — The prompt should instruct the LLM to use short, readable IDs (e.g., "M001", "S01", "T01") rather than UUIDs. The SQLC schema uses `TEXT` for IDs, so any string works.
