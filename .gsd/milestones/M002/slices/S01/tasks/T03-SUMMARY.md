---
id: T03
parent: S01
milestone: M002
provides: []
requires: []
affects: []
key_files: ["internal/auto/prompts.go", "internal/auto/prompts_test.go", "internal/auto/templates/research.md.tpl", "internal/auto/templates/plan_slice.md.tpl", "internal/auto/templates/execute_task.md.tpl", "internal/auto/templates/summarize.md.tpl", "internal/auto/templates/validate.md.tpl", "internal/auto/engine.go"]
key_decisions: ["Used Go embed.FS to bundle templates at compile time rather than loading from disk at runtime", "Replaced the placeholder buildPrompt switch in engine.go with the real BuildPrompt using templates"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "go test ./internal/auto/... -count=1 -v — all tests pass. go vet ./internal/auto/... — no errors."
completed_at: 2026-03-27T19:52:00.072Z
blocker_discovered: false
---

# T03: Created five Go template files for auto-mode unit types and a BuildPrompt function that renders them with PromptContext, replacing the placeholder buildPrompt in engine.go

> Created five Go template files for auto-mode unit types and a BuildPrompt function that renders them with PromptContext, replacing the placeholder buildPrompt in engine.go

## What Happened
---
id: T03
parent: S01
milestone: M002
key_files:
  - internal/auto/prompts.go
  - internal/auto/prompts_test.go
  - internal/auto/templates/research.md.tpl
  - internal/auto/templates/plan_slice.md.tpl
  - internal/auto/templates/execute_task.md.tpl
  - internal/auto/templates/summarize.md.tpl
  - internal/auto/templates/validate.md.tpl
  - internal/auto/engine.go
key_decisions:
  - Used Go embed.FS to bundle templates at compile time rather than loading from disk at runtime
  - Replaced the placeholder buildPrompt switch in engine.go with the real BuildPrompt using templates
duration: ""
verification_result: passed
completed_at: 2026-03-27T19:52:00.073Z
blocker_discovered: false
---

# T03: Created five Go template files for auto-mode unit types and a BuildPrompt function that renders them with PromptContext, replacing the placeholder buildPrompt in engine.go

**Created five Go template files for auto-mode unit types and a BuildPrompt function that renders them with PromptContext, replacing the placeholder buildPrompt in engine.go**

## What Happened

Created internal/auto/templates/ with five .md.tpl files — one per UnitType (research, plan_slice, execute_task, summarize, validate). Each template receives a PromptContext struct carrying milestone/slice/task metadata, prior summaries, and working directory. Optional fields use Go template conditionals. Created BuildPrompt function with embed.FS and templateNames map. Updated engine.go to use BuildPrompt instead of the placeholder. All 36+ tests pass, no vet errors.

## Verification

go test ./internal/auto/... -count=1 -v — all tests pass. go vet ./internal/auto/... — no errors.

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go test ./internal/auto/... -count=1 -v` | 0 | ✅ pass | 434ms |
| 2 | `go vet ./internal/auto/...` | 0 | ✅ pass | 500ms |


## Deviations

None.

## Known Issues

None.

## Files Created/Modified

- `internal/auto/prompts.go`
- `internal/auto/prompts_test.go`
- `internal/auto/templates/research.md.tpl`
- `internal/auto/templates/plan_slice.md.tpl`
- `internal/auto/templates/execute_task.md.tpl`
- `internal/auto/templates/summarize.md.tpl`
- `internal/auto/templates/validate.md.tpl`
- `internal/auto/engine.go`


## Deviations
None.

## Known Issues
None.
