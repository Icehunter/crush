# S03: Stuck Detection + Context Pressure — UAT

**Milestone:** M003
**Written:** 2026-03-28T05:34:11.848Z

# S03: Stuck Detection + Context Pressure — UAT

**Milestone:** M003
**Written:** 2026-03-28

## UAT Type

- UAT mode: artifact-driven
- Why this mode is sufficient: Both stuck detection and context pressure are engine-internal safety gates tested via unit and integration tests. No UI or CLI surface to manually exercise — the gates are wired into the engine loop and fire based on dispatch outcomes and token counts.

## Preconditions

- Go toolchain available
- Working directory contains the crush source with M003 S03 changes
- `go test ./internal/auto/` passes

## Smoke Test

Run `go test ./internal/auto/ -run 'TestIntegration_Stuck|TestIntegration_Context' -count=1 -v` — all 6 integration tests pass, confirming both safety rails function end-to-end in the engine loop.

## Test Cases

### 1. Stuck Detection — Retry Succeeds

1. Create an engine with a StuckDetector (window size 3)
2. Dispatcher fails 3 times for the same unit (window full, >50% failures)
3. Engine detects stuck, dispatches diagnostic retry
4. Dispatcher succeeds on retry
5. **Expected:** Engine recovers and continues. Unit advances. No EventStuckDetected published.

### 2. Stuck Detection — Retry Fails, Engine Pauses

1. Create an engine with a StuckDetector (window size 3)
2. Dispatcher fails 3 times for the same unit
3. Engine detects stuck, dispatches diagnostic retry
4. Diagnostic retry also fails
5. **Expected:** Engine pauses. EventStuckDetected is published. Engine status is paused.

### 3. Stuck Detection — Below Threshold, Normal Operation

1. Create an engine with a StuckDetector (window size 5)
2. Dispatcher fails 2 times, succeeds 3 times (below 50% failure threshold)
3. **Expected:** Engine proceeds normally. No stuck detection triggered. All units advance.

### 4. Context Pressure — Exceeds Threshold, Engine Pauses

1. Create an engine with a ContextMonitor (threshold 0.8, context window 10000)
2. Mock TokenQuerier returns 9000 total tokens (90% > 80% threshold)
3. Dispatch one unit successfully
4. **Expected:** After dispatch, context pressure check fires. Engine pauses. EventContextPressure published.

### 5. Context Pressure — Below Threshold, Normal Operation

1. Create an engine with a ContextMonitor (threshold 0.8, context window 10000)
2. Mock TokenQuerier returns 5000 total tokens (50% < 80% threshold)
3. Dispatch units until sequence exhausted
4. **Expected:** Engine completes all units normally. No pause. No EventContextPressure.

### 6. Context Pressure — Nil Monitor Skips

1. Create an engine with nil contextMonitor
2. Dispatch units
3. **Expected:** Engine completes normally. Context pressure check is a no-op.

## Edge Cases

### Stuck detector with nil detector
- Pass nil for stuckDetector in NewEngine
- **Expected:** All stuck checks are no-ops. Engine runs normally.

### Stuck detector 50/50 boundary
- Window size 4, exactly 2 passes and 2 failures
- **Expected:** Not stuck (must be strictly >50% failures)

### Context monitor with zero context window
- NewContextMonitor(0.8, 0, querier) returns nil
- **Expected:** Nil monitor passed to engine, context check is no-op

### Context monitor querier error
- TokenQuerier.GetTokenUsage returns an error
- **Expected:** Error propagated, engine handles gracefully

### Multiple units with independent stuck windows
- Two different unit keys each have their own sliding window
- Unit A is stuck, Unit B is not
- **Expected:** Stuck detection only triggers for Unit A, not Unit B
