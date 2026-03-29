# S01 — Branch Reconciliation Research

**Date:** 2026-03-28
**Status:** Complete

## Summary

Merging `milestone/M003` into `milestone/M005` is structurally straightforward but has one hard compile-blocking conflict that must be resolved manually: **duplicate `AutoEvent` struct definitions and duplicate event constant names** (`EventUnitStarted`, `EventUnitCompleted`) with different types. M003 defines engine events in `events.go` using `pubsub.EventType` constants and a `{Unit, Error error, Timestamp, Message}` payload. M004 (on M005) defines TUI events in `event.go` using `AutoEventType` string constants and a `{Type, MilestoneID, SliceID, TaskID, Phase, Error string, Snapshot}` payload. After merge, both files land in `internal/auto/` and the package won't compile.

All **source files** merge cleanly via `git merge` — zero source code conflicts. The only conflicts are in `.gsd/` metadata files (DECISIONS.md, KNOWLEDGE.md, REQUIREMENTS.md, etc.) which are managed artifacts and can be resolved by keeping M005's versions. The real work is the post-merge unification of the two event systems.

## Recommendation

**Merge M003 into M005, then unify events in a single reconciliation pass.** The unified `AutoEvent` must satisfy both consumers: the engine (publishes events with Unit context, error, message) and the TUI sidebar (reads `Snapshot` for rendering). The recommended approach:

1. Keep M003's `events.go` as the canonical event file (it has the `pubsub.EventType` constants the engine uses).
2. Extend M003's `AutoEvent` struct to include the `Snapshot` field that the TUI needs.
3. Add M004's missing event type constants (`EventAutoStarted`, `EventAutoPaused`, `EventAutoResumed`, etc.) as `pubsub.EventType` constants alongside M003's existing ones.
4. Delete M004's `event.go` and `event_test.go` — they're superseded.
5. Update TUI consumers in `internal/ui/model/ui.go` and `internal/app/auto_events_test.go` to use the unified struct.

This keeps the engine as the source of truth for events (M003 is more complete: 12 event types vs 8) while preserving the TUI's `AutoSnapshot` rendering capability.

## Implementation Landscape

### Key Files

- `internal/auto/events.go` (from M003) — Engine event constants (`pubsub.EventType`) and `AutoEvent` struct with `{Unit, Error, Timestamp, Message}`. This is the canonical event file; needs `Snapshot *AutoSnapshot` field added.
- `internal/auto/event.go` (from M004/M005) — TUI event types (`AutoEventType`) and `AutoEvent` with `{Type, MilestoneID, SliceID, TaskID, Phase, Error string, Snapshot}`. Must be deleted after merge; its `AutoSnapshot` and `SliceProgress` types move to `events.go`.
- `internal/auto/event_test.go` (from M004/M005) — Tests for TUI event types. Must be rewritten to test the unified struct.
- `internal/auto/engine.go` (from M003) — 505-line engine: `Run()`, `step()`, `publish()`. Uses `NewAutoEvent(unit, err, message)` to publish. No changes needed if event unification preserves `NewAutoEvent`.
- `internal/ui/model/ui.go` — TUI update handler: `case pubsub.Event[auto.AutoEvent]: m.autoSnapshot = msg.Payload.Snapshot`. Needs adjustment to read Snapshot from the unified struct.
- `internal/ui/model/sidebar.go` — Sidebar renderer reads `m.autoSnapshot` (type `*auto.AutoSnapshot`). No change needed if `AutoSnapshot` struct is preserved.
- `internal/app/auto_events.go` — Broker wiring: `pubsub.NewBroker[auto.AutoEvent]()`. No change needed — both sides already use `pubsub.Broker[AutoEvent]`.
- `internal/app/auto_events_test.go` — Tests that create `auto.AutoEvent{Type: ..., Snapshot: ...}`. Must update field references after unification.
- `internal/config/config.go` — M003 moves `WorktreeMode` from `Options` into `AutoConfig` struct. Merges cleanly.
- `internal/agent/coordinator.go` — M003 has a small logic change to `isAnthropicThinking`. Merges cleanly.
- `internal/db/sessions.sql.go` — M003 adds `SumChildSessionCosts` query. Merges cleanly.
- `internal/db/querier.go` — M003 adds `SumChildSessionCosts` to interface. Merges cleanly.
- `internal/db/auto_test.go` — M003 deletes this file (tests moved to `internal/auto/`). Merge handles this.

### New Files from M003 (not on M005 today)

All of these are net-new additions that don't conflict:

| File | Lines | Purpose |
|------|-------|---------|
| `internal/auto/engine.go` | 505 | Core engine loop |
| `internal/auto/engine_test.go` | 517 | Engine unit tests |
| `internal/auto/engine_integration_test.go` | 232 | Full loop lifecycle tests |
| `internal/auto/engine_verify_integration_test.go` | 230 | Verify + retry tests |
| `internal/auto/engine_budget_integration_test.go` | 187 | Budget gate tests |
| `internal/auto/engine_stuck_integration_test.go` | 240 | Stuck detection tests |
| `internal/auto/engine_context_integration_test.go` | 154 | Context pressure tests |
| `internal/auto/state.go` | 215 | DeriveState + StateQuerier interface |
| `internal/auto/state_test.go` | 432 | State derivation tests |
| `internal/auto/budget.go` | 35 | BudgetChecker interface |
| `internal/auto/budget_test.go` | 53 | Budget tests |
| `internal/auto/context.go` | 43 | ContextMonitor + TokenQuerier |
| `internal/auto/context_test.go` | 78 | Context monitor tests |
| `internal/auto/stuck.go` | 104 | StuckDetector |
| `internal/auto/stuck_test.go` | 124 | Stuck detection tests |
| `internal/auto/verify.go` | 137 | ShellVerifier |
| `internal/auto/verify_test.go` | 118 | Verifier tests |
| `internal/auto/lock.go` | 144 | LockFile (O_EXCL) |
| `internal/auto/lock_test.go` | 105 | Lock file tests |
| `internal/auto/prompts.go` | 106 | BuildPrompt + template loading |
| `internal/auto/prompts_test.go` | 201 | Prompt builder tests |
| `internal/auto/init.go` | 111 | InitTool definitions |
| `internal/auto/init_test.go` | 159 | Init tool tests |
| `internal/auto/init_tools.go` | 182 | Planning tool implementations |
| `internal/auto/init_tools_test.go` | 470 | Planning tool tests |
| `internal/auto/unit.go` | 65 | Unit type + UnitType enum |
| `internal/auto/events.go` | 49 | Engine event types (conflict with event.go) |
| `internal/auto/templates/*.md.tpl` | 6 files | Prompt templates |
| `internal/config/auto_test.go` | — | AutoConfig parsing tests |

### Build Order

1. **Git merge + .gsd conflict resolution** — Merge `milestone/M003` into `milestone/M005`. Resolve `.gsd/` metadata conflicts by keeping M005's versions (they're newer and include M004 data). This is mechanical.

2. **Event type unification (compile-critical)** — This is the single riskiest task and must be done immediately after merge. Delete `event.go`, merge `AutoSnapshot`/`SliceProgress` into `events.go`, add a `Snapshot *AutoSnapshot` field to `AutoEvent`, reconcile constant names. Update TUI consumers. This unblocks compilation.

3. **Build verification** — `go build ./...` must pass. The `coordinator.go` change (removing `ReasoningEffort` check) could conflict with upstream changes — verify.

4. **Test verification** — `go test ./internal/auto/...` must pass all ~42+ engine tests plus M004's existing tests.

### Verification Approach

The slice succeeds when:

```bash
# All source compiles
go build ./...

# All auto-mode tests pass (engine + state + safety + events + worktree)
go test ./internal/auto/... -v -count=1

# Config tests pass (AutoConfig struct)
go test ./internal/config/... -run TestAutoConfig -v

# App event broker tests pass
go test ./internal/app/... -run TestAutoEvent -v

# UI model tests pass (sidebar + auto toggle)
go test ./internal/ui/model/... -run "TestAutoMode\|TestSidebar.*Auto\|TestToggleAuto" -v

# DB tests pass (SumChildSessionCosts)
go test ./internal/db/... -v -count=1
```

## Constraints

- `CGO_ENABLED=0` — SQLite driver must be CGO-free. M003's `SumChildSessionCosts` query uses the existing sqlc setup so this is fine.
- The merge must be a single commit (or small series) to keep git history clean. Don't rewrite M003 history.
- `.gsd/` metadata files are managed by the GSD tooling — resolve conflicts by keeping M005's versions since they include M004 completion data.

## Common Pitfalls

- **Duplicate symbol compilation error** — Both `events.go` (M003) and `event.go` (M004) define `EventUnitStarted` and `EventUnitCompleted` with different types. The compiler will refuse to build. Must delete one file before attempting `go build`.
- **AutoEvent field mismatch in TUI** — After unification, `internal/ui/model/ui.go` does `m.autoSnapshot = msg.Payload.Snapshot`. If the unified `AutoEvent` struct doesn't have a `Snapshot` field, the TUI breaks. Add the field during unification.
- **M003's coordinator.go change may regress** — M003 removes the `ReasoningEffort` check in `isAnthropicThinking()`. If upstream added reasoning effort support between claudecode and M005, this removal could regress functionality. Verify the current M005 coordinator.go before applying.
- **Test helper coupling** — M003 engine tests use coupled mock querier/advancer (K006, K008). If any M004 changes touched shared test helpers, tests could break. The mocks are all within `internal/auto/` so isolation should be clean.
- **`internal/db/auto_test.go` deletion** — M003 deletes this file. M005 still has it. After merge, verify the file is actually gone and that its tests moved to `internal/auto/`.

## Open Risks

- The `coordinator.go` change from M003 (removing `ReasoningEffort` check) may conflict with M004's auto-model-tier-switching feature. The M005 commit message mentions "auto model tier switching" — need to verify this doesn't depend on the removed code path.
- If `pubsub.Broker` generic type changed between claudecode and M005, the engine's broker usage pattern may need adjustment. Confirmed: the broker API is stable (`Publish(EventType, T)`, `Subscribe(ctx) <-chan Event[T]`).
