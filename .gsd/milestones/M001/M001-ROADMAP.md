# M001: M001: Hierarchical Task Model + State Machine

## Vision
M001: Hierarchical Task Model + State Machine

## Slice Overview
| ID | Slice | Risk | Depends | Done | After this |
|----|-------|------|---------|------|------------|
| S01 | DB Schema + SQLC Queries | high | — | ✅ | TBD |
| S02 | Domain Model + Status Enums | low | S01 | ✅ | TBD |
| S03 | State Derivation Engine | high | S02 | ✅ | TBD |
| S04 | Dispatch Rules Table | medium | S03 | ✅ | TBD |
| S05 | Integration Proof | low | S04 | ✅ | # S05: Integration Proof — UAT

**Milestone:** M001
**Written:** 2026-03-27T19:18:27.593Z

# S05: Integration Proof — UAT

**Milestone:** M001
**Written:** 2026-03-27

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: This slice delivers test code only — no runtime behavior, no UI, no API. The tests themselves are the proof artifacts.

## Preconditions

- Go toolchain available (go 1.24+)
- Working directory is the crush repo root (or worktree)
- SQLite available (CGO_ENABLED=0 with modernc driver)

## Smoke Test

Run `go test ./internal/auto/ -run TestIntegration_FullLifecycle -v -count=1` — should pass with 11 lifecycle steps logged.

## Test Cases

### 1. Full Lifecycle Sequence

1. Run `go test ./internal/auto/ -run TestIntegration_FullLifecycle -v -count=1`
2. **Expected:** Test passes. Output shows 11 steps from plan_milestone through none terminal state.

### 2. Empty Database Returns None

1. Run `go test ./internal/auto/ -run TestIntegration_EmptyDB -v -count=1`
2. **Expected:** Test passes. DeriveState on empty DB returns ActionNone.

### 3. Dependency Gating

1. Run `go test ./internal/auto/ -run TestIntegration_DependencyGating -v -count=1`
2. **Expected:** Test passes. S02 (depends on S01) is skipped while S01 is active. After S01 completes, S02 becomes actionable.

### 4. Terminal State

1. Run `go test ./internal/auto/ -run TestIntegration_TerminalState -v -count=1`
2. **Expected:** Test passes. Fully-completed milestone returns ActionNone.

### 5. Full Suite Regression

1. Run `go test ./internal/auto/ -v -count=1`
2. **Expected:** All 45 tests pass (41 existing unit tests + 4 new integration tests). No regressions.

## Edge Cases

### Parallel Safety
- All integration tests use `t.Parallel()`. Each creates its own in-memory SQLite database. No shared state between tests.

### Build and Vet
- `go build ./internal/auto/...` exits 0
- `go vet ./internal/auto/...` exits 0
 |
