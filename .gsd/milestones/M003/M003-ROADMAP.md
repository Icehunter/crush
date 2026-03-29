# M003: M003: Safety Rails - Context

## Vision
Make auto-mode trustworthy for unattended use by adding four safety rails: verification gates that run configurable commands after task execution (with retry-on-failure), dollar-cost budget ceiling, stuck detection with retry-then-pause escalation, and context pressure monitoring with wrap-up signaling.

## Slice Overview
| ID | Slice | Risk | Depends | Done | After this |
|----|-------|------|---------|------|------------|
| S01 | Auto Config + Verification Gates | high | — | ✅ | Configure auto.verification_commands in crush.json. Engine runs them after each task dispatch. On failure, engine re-dispatches with a diagnostic prompt containing truncated failure output. Tests prove the full verify→retry→succeed and verify→retry→fail paths. |
| S02 | Budget Ceiling | medium | S01 | ✅ | Set auto.budget_ceiling to 0.50 in crush.json. Engine checks cumulative child session costs before each dispatch. When ceiling is reached, engine pauses and publishes EventBudgetExceeded. Tests prove budget enforcement with mock session costs. |
| S03 | Stuck Detection + Context Pressure | medium | S01 | ✅ | Engine tracks dispatch results in a sliding window. When >50% of recent dispatches fail for the same unit, engine retries with diagnostic then pauses if still stuck. Context monitor tracks token usage against model context window and signals wrap-up at configurable threshold. Tests prove stuck detection escalation and context pressure signaling. |
