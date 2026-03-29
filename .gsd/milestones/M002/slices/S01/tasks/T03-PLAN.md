---
estimated_steps: 28
estimated_files: 7
skills_used: []
---

# T03: Auto-mode system prompt templates and rendering

Create Go template files for each auto-mode unit type and a prompt builder that renders them with runtime context. The engine calls this to get the system prompt before dispatching to the Coordinator.

## Steps

1. Create `internal/auto/templates/` directory with Go template files for each unit type:
   - `research.md.tpl` — prompt for researching a slice (read codebase, understand scope, produce research notes)
   - `plan_slice.md.tpl` — prompt for planning a slice (decompose into tasks, define verification)
   - `execute_task.md.tpl` — prompt for executing a task (implement code changes, run verification)
   - `summarize.md.tpl` — prompt for summarizing a slice (what was done, deviations, follow-ups)
   - `validate.md.tpl` — prompt for validating a milestone (check success criteria, run integration tests)
   Each template receives: milestone title, slice title/goal, task title/description (when applicable), prior context summaries, available tools.

2. Create `internal/auto/prompts.go` with:
   - `//go:embed templates/*.md.tpl` for all templates
   - `type PromptContext struct` carrying milestone/slice/task info, prior summaries, working directory
   - `func BuildPrompt(unitType UnitType, ctx PromptContext) (string, error)` that selects the right template and renders it
   - Template rendering uses `text/template` with the standard Go template library

3. Create `internal/auto/prompts_test.go` testing:
   - Each template renders without error given valid PromptContext
   - Correct template selected for each UnitType
   - Output contains expected sections (e.g. execute_task prompt includes task description)
   - Missing optional fields don't cause errors

4. Run `go test ./internal/auto/... -count=1` and `go vet ./internal/auto/...`.

## Must-Haves

- [ ] Templates exist for all 5 unit types: research, plan_slice, execute_task, summarize, validate
- [ ] BuildPrompt selects correct template by UnitType and renders with PromptContext
- [ ] Templates include placeholders for milestone/slice/task context and prior summaries
- [ ] Tests verify rendering for each unit type without errors

## Verification

- `go test ./internal/auto/... -count=1 -v` — all prompt tests pass
- `go vet ./internal/auto/...` — no vet errors

## Inputs

- `internal/auto/unit.go`
- `internal/auto/events.go`
- `internal/agent/prompts.go`
- `internal/agent/templates/coder.md.tpl`

## Expected Output

- `internal/auto/prompts.go`
- `internal/auto/prompts_test.go`
- `internal/auto/templates/research.md.tpl`
- `internal/auto/templates/plan_slice.md.tpl`
- `internal/auto/templates/execute_task.md.tpl`
- `internal/auto/templates/summarize.md.tpl`
- `internal/auto/templates/validate.md.tpl`

## Verification

go test ./internal/auto/... -count=1 && go vet ./internal/auto/...
