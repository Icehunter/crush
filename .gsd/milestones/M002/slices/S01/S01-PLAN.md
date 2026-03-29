# S01: Core Auto Loop Engine + CLI

**Goal:** Core auto loop engine that derives state from the DB, dispatches units to the LLM with fresh sessions, advances through tasks/slices, publishes typed events, and is controllable via `crush auto start/pause/stop/status` CLI commands. Lock file prevents concurrent instances. Kill + restart resumes from DB state.
**Demo:** After this: TBD

## Tasks
- [x] **T01: Built pure-logic state derivation layer with UnitType enum, Unit struct, AutoEvent types, StateQuerier interface, and DeriveState function — 18 tests pass** — 
