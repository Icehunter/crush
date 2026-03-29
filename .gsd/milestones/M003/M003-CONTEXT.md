---
depends_on: [M002]
---

# M003: Safety Rails — Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

## Project Description

Safety rails for autonomous operation: verification gates that run configurable commands after task execution, dollar-cost budget ceiling, stuck detection with retry-then-pause escalation, and context pressure monitoring. After this milestone, auto-mode is safe to run unattended.

## Why This Milestone

M002 delivers a working auto loop, but without guardrails it can produce broken code (no verification), burn unlimited budget, or loop infinitely on unsolvable problems. M003 makes auto-mode trustworthy for unattended use.

## User-Visible Outcome

### When this milestone is complete, the user can:

- Configure `auto.verification_commands` in crush.json (e.g., `["go test ./...", "golangci-lint run"]`) and see auto-mode run them after each task, auto-retrying with a diagnostic prompt on failure
- Set `auto.budget_ceiling` (e.g., `5.00` for $5) and have auto-mode pause when cumulative cost reaches the limit
- See auto-mode pause and report diagnostics when stuck on a task, rather than retrying infinitely
- Trust that context pressure is managed — auto-mode signals wrap-up before context degrades

### Entry point / environment

- Entry point: `crush auto start` with crush.json `auto` section configured
- Environment: local dev terminal
- Live dependencies involved: configured LLM provider, project test/lint tooling

## Completion Class

- Contract complete means: unit tests for verification gate, budget enforcement, stuck detection, and context pressure. Config parsing tests.
- Integration complete means: auto loop runs with verification commands enabled and correctly retries on failure. Budget enforcement pauses the loop.
- Operational complete means: stuck detection pauses after threshold. Context pressure triggers clean wrap-up.

## Final Integrated Acceptance

To call this milestone complete, we must prove:

- A deliberately broken task triggers verification failure, auto-retry with diagnostic, and produces a fixed result
- A deliberately failing task that can't be fixed triggers stuck detection and pauses with a diagnostic report
- Setting a low budget ceiling causes auto-mode to pause before exceeding it
- Auto-mode running a large task manages context pressure and writes clean handoff points

## Implementation Decisions

- **Verification via shell:** Verification commands run through Crush's existing `internal/shell/` package (POSIX shell emulation). Same execution model as the bash tool.
- **Config in `auto` section:** New `Auto` struct on `Config` with fields: `VerificationCommands []string`, `BudgetCeiling float64`, `StuckThreshold int`, `WorktreeMode string` (consumed by M004).
- **Budget aggregation:** New SQLC query to sum child session costs by parent session ID. Budget check runs before each dispatch.
- **Stuck detection sliding window:** Keep last N (configurable, default 5) dispatch results in memory. If >50% are failures for the same unit, retry with diagnostic prompt. If diagnostic retry also fails, pause and surface to user.
- **Retry-then-pause:** On verification failure, re-dispatch the same task with a diagnostic prompt containing the failure output. If the retry also fails, invoke stuck detection. No skip-and-continue.
- **Context pressure:** Use `fantasy.AgentResult.TotalUsage` token counts. When approaching model's context limit (e.g., 80% of max), signal wrap-up to the agent via a follow-up message. If the agent doesn't wrap up, force-save state and start a fresh session for the same task.

## Agent's Discretion

- Diagnostic prompt content — how to frame verification failures for the retry prompt
- Context pressure threshold — what percentage of context to trigger wrap-up at
- Stuck detection window size default
- Error message formatting for stuck and budget pause reports

## Risks and Unknowns

- **Verification command output parsing** — need to determine pass/fail from command exit codes and potentially parse output for specific failure details to include in diagnostic prompts.
- **Context limit detection** — different models have different context windows. Need to get the model's max context from fantasy/catwalk metadata.

## Existing Codebase / Prior Art

- `internal/shell/` — POSIX shell emulation. `Shell.Run()` executes commands and returns output + exit code.
- `internal/session/session.go` — `Session.Cost`, `Session.PromptTokens`, `Session.CompletionTokens`. `parent_session_id` for hierarchy.
- `internal/config/config.go` — `Config.Options` struct. Pattern for adding new config sections.
- `internal/db/sql/sessions.sql` — existing session queries. Need new query for cost aggregation by parent.
- `charm.land/fantasy` — `AgentResult.TotalUsage` for token tracking. `catwalk.Model` for model metadata.

## Relevant Requirements

- R010 — Verification gates (primary)
- R011 — Dollar-cost budget ceiling (primary)
- R012 — Stuck detection (primary)
- R013 — Context pressure monitoring (primary)
- R014 — Auto config section in crush.json (primary)

## Scope

### In Scope

- `Auto` config struct and crush.json parsing
- Verification gate: run commands, parse exit code, retry with diagnostic on failure
- Budget tracking: aggregate child session costs, enforce ceiling, pause when reached
- Stuck detection: sliding window, diagnostic retry, pause-and-report escalation
- Context pressure: token usage monitoring, wrap-up signal, forced handoff
- SQLC query for cost aggregation by parent session ID

### Out of Scope

- TUI rendering of safety rail state (M004)
- Git worktrees (M004)
- Token-based budget ceiling (deferred, R022)
- User-customizable prompts (deferred, R021)
