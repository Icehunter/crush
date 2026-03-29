---
id: T02
parent: S02
milestone: M002
provides: []
requires: []
affects: []
key_files: ["internal/auto/templates/init.md.tpl", "internal/auto/prompts.go", "internal/auto/init.go", "internal/cmd/auto.go", "internal/auto/prompts_test.go"]
key_decisions: ["Opened separate db.Connect in CLI command rather than exposing db.Queries from App, matching existing session.go/stats.go pattern", "Set IsSubAgent: true on SessionAgent to avoid UI notification side effects during non-interactive planning", "Used dedicated pubsub.Broker for notifications scoped to init session lifetime"]
patterns_established: []
drill_down_paths: []
observability_surfaces: []
duration: ""
verification_result: "go build . — compiles clean. go test ./internal/auto/... -count=1 — 66 tests pass (64 existing + 2 new). go vet ./internal/auto/... ./internal/cmd/... — clean."
completed_at: 2026-03-27T22:16:04.989Z
blocker_discovered: false
---

# T02: Wired end-to-end init flow: init.md.tpl prompt template, BuildInitPrompt() renderer, RunInit() with planning-only SessionAgent, and crush auto init cobra command

> Wired end-to-end init flow: init.md.tpl prompt template, BuildInitPrompt() renderer, RunInit() with planning-only SessionAgent, and crush auto init cobra command

## What Happened
---
id: T02
parent: S02
milestone: M002
key_files:
  - internal/auto/templates/init.md.tpl
  - internal/auto/prompts.go
  - internal/auto/init.go
  - internal/cmd/auto.go
  - internal/auto/prompts_test.go
key_decisions:
  - Opened separate db.Connect in CLI command rather than exposing db.Queries from App, matching existing session.go/stats.go pattern
  - Set IsSubAgent: true on SessionAgent to avoid UI notification side effects during non-interactive planning
  - Used dedicated pubsub.Broker for notifications scoped to init session lifetime
duration: ""
verification_result: passed
completed_at: 2026-03-27T22:16:04.989Z
blocker_discovered: false
---

# T02: Wired end-to-end init flow: init.md.tpl prompt template, BuildInitPrompt() renderer, RunInit() with planning-only SessionAgent, and crush auto init cobra command

**Wired end-to-end init flow: init.md.tpl prompt template, BuildInitPrompt() renderer, RunInit() with planning-only SessionAgent, and crush auto init cobra command**

## What Happened

Created four deliverables: (1) init.md.tpl planning prompt template with ID/status/sort conventions, (2) BuildInitPrompt() and InitPromptContext in prompts.go, (3) RunInit() in init.go that constructs a SessionAgent with only the three planning tools and dispatches non-interactively, (4) autoInitCmd in cmd/auto.go that wires App dependencies and calls RunInit. Opened a separate DB connection in the CLI command matching existing patterns in session.go/stats.go.

## Verification

go build . — compiles clean. go test ./internal/auto/... -count=1 — 66 tests pass (64 existing + 2 new). go vet ./internal/auto/... ./internal/cmd/... — clean.

## Verification Evidence

| # | Command | Exit Code | Verdict | Duration |
|---|---------|-----------|---------|----------|
| 1 | `go build .` | 0 | ✅ pass | 3000ms |
| 2 | `go test ./internal/auto/... -count=1` | 0 | ✅ pass | 600ms |
| 3 | `go vet ./internal/auto/... ./internal/cmd/...` | 0 | ✅ pass | 1000ms |


## Deviations

None.

## Known Issues

None.

## Files Created/Modified

- `internal/auto/templates/init.md.tpl`
- `internal/auto/prompts.go`
- `internal/auto/init.go`
- `internal/cmd/auto.go`
- `internal/auto/prompts_test.go`


## Deviations
None.

## Known Issues
None.
