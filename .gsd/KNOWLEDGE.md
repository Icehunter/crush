# Knowledge Register

<!-- Append-only. Project-specific rules, patterns, and lessons learned.
     Read at the start of every unit. Append when you discover something future agents need. -->

## K001: 529 errors during milestone planning can corrupt roadmap state

**When:** M001 initial planning
**What happened:** API 529 errors during `gsd_plan_milestone` caused S05 to be marked ✅ from birth (empty plan, no tasks, no execution). The roadmap title/vision were also malformed ("M001:" instead of real text). Additionally, `gsd_complete_slice` was writing UAT content into the slice demo/After-this DB field, causing the roadmap table to contain entire UAT documents in cells.
**Impact:** Milestone appeared closer to completion than it was. S05 needed manual roadmap repair.
**Rule:** After 529 error recovery, verify roadmap checkboxes match actual completion state. Check that uncompleted slices have ⬜ not ✅, and that the "After this" column contains one-liners not multi-page documents.

## K002: gsd_complete_slice uatContent parameter populates the demo field

**When:** M001 S01-S04 completion
**What happened:** The `uatContent` parameter passed to `gsd_complete_slice` was rendered into the roadmap's "After this" column, bloating the roadmap from 15 lines to 465 lines.
**Rule:** This is a known rendering behavior. The roadmap was manually rewritten with proper one-liners. Future closers should be aware the demo field may get overwritten by the tool.

## K003: Worktree isolation requires consolidating untracked files

**When:** M001 S01-S02
**What happened:** Files created in the main working copy don't appear in the worktree. Each slice needed to copy newly-created source files from the main repo into the worktree before they could be used.
**Rule:** When working in a worktree, verify that files from prior slices exist in the worktree. If not, they need to be consolidated in.

## K004: In-memory SQLite with unique DSN per test enables safe t.Parallel()

**When:** M001 S01
**What happened:** Using `file:<test-name>?mode=memory&cache=shared` as the DSN gives each test its own in-memory database. Combined with goose migrations, this enables fully parallel test execution with zero shared state.
**Rule:** Use this pattern for any auto-mode tests that need a real database. Never share a database connection across parallel tests.

## K005: Pre-existing vet failures surface in worktrees

**When:** M001 S01
**What happened:** `go vet ./...` caught a value receiver on a struct containing sync.RWMutex in internal/csync/maps.go. This existed in main but wasn't caught until the worktree build gate ran.
**Rule:** Run `go vet ./...` early in a milestone to catch pre-existing issues before they block slice verification.

## K006: Test querier+advancer coupling needed for multi-phase loop tests

**When:** M002 S01/T04
**What happened:** `fixedSequenceQuerier` returned units in order, but for non-task unit types (research, plan, summarize, validate) `DeriveState` never called `ListTasksBySlice`, so the sequence index never advanced — causing infinite loops in integration tests. Fixed by adding an `Advance()` method on the querier, called by `mockAdvancer.AdvanceStatus()` after each unit completes.
**Rule:** When testing the auto loop with a mock querier, the advancer must advance the querier's index. Couple them via a shared reference so `AdvanceStatus()` triggers `querier.Advance()`.

## K007: O_EXCL lock file race between create and write

**When:** M002 S01/T04
**What happened:** `OpenFile(O_CREATE|O_EXCL)` creates the lock file atomically, but the JSON payload write is a separate operation. A concurrent goroutine could read the empty file, fail unmarshal, treat it as stale, delete it, and acquire. Fixed by treating unmarshal failures as "lock held" rather than "stale".
**Rule:** When using O_EXCL lock files, treat any file that exists but can't be parsed as "lock held" — never assume an unparseable lock is stale.

## K008: Test querier+advancer coupling prevents infinite loops

**When:** M002 S01/T04
**What happened:** Integration tests looped infinitely because the mock advancer wasn't calling the mock querier's Advance() method. DeriveState kept returning the same unit. Fix: mock advancer calls querier.Advance() so state progresses.
**Rule:** In engine loop tests, the mock advancer must mutate the mock querier's state so DeriveState returns fresh results each iteration.

## K009: LLM tool errors should be JSON text, not Go errors

**When:** M002 S02/T01
**What happened:** Planning tools initially returned Go errors on validation failure. The LLM couldn't parse these and couldn't self-correct. Switching to JSON text responses (`{"error": "id is required"}`) lets the LLM read and retry.
**Rule:** LLM-facing tools should return structured JSON error responses, not Go error types. The LLM needs parseable feedback to self-correct.

## K010: Consuming-package interfaces decouple test and production code cleanly

**When:** M002 S01
**What happened:** Defining StateQuerier, SessionCreator, Dispatcher, StatusAdvancer interfaces in internal/auto/ with lightweight Row types (not db.Queries) enabled full mock testing without importing the DB layer. Tests run in-memory with zero DB dependencies.
**Rule:** Define interfaces in the consuming package with minimal types. This avoids circular imports and makes mocking trivial.

## K011: Consistent gate pattern reduces implementation cost per safety rail

**When:** M003 S01-S03
**What happened:** S01 established the safety gate pattern: nil/zero config disables, gate checks at a defined point in step(), exceeded condition pauses engine and publishes a typed event. S02 and S03 each took less time because they followed the same pattern. The verification gate, budget gate, stuck gate, and context gate all share identical structure.
**Rule:** When adding engine gates, follow the established pattern: nil/zero disables, check at defined step() point, pause+publish on exceeded. This makes each new gate predictable and reduces review burden.

## K012: NewEngine parameter growth signals need for options pattern

**When:** M003 S02-S03
**What happened:** Each safety rail added 1-2 parameters to NewEngine (verifier, budgetChecker, budgetCeiling, stuckDetector, contextMonitor). Every addition required updating all NewEngine call sites across 3-5 test files — purely mechanical but time-consuming. By S03/T02, there were 17 call sites to update.
**Rule:** When a constructor exceeds ~8 parameters, consider switching to an options struct or functional options pattern. This is a future refactor candidate for NewEngine.

## K013: Mirror existing event patterns when adding new event subsystems

**When:** M004 S01
**What happened:** Auto-mode events were modeled identically to the existing LSP event pattern — typed string constants, flat structs, package-level broker, setupSubscriber registration in App. This meant zero design decisions for event plumbing and the code reviewed as a natural extension.
**Rule:** When adding a new event type to Crush, copy the existing pattern in internal/app/lsp_events.go verbatim. The consistency makes code review trivial and reduces integration risk.

## K014: Testing git worktree operations requires real repos, not mocks

**When:** M004 S03
**What happened:** WorktreeManager tests use t.TempDir() + real git init to test Create/Merge/Remove against actual git repos. This caught two bugs that mocks would have missed: (1) runGit needed both stdout and stderr in error output for nothing-to-commit detection, (2) git outputs both "nothing to commit" and "nothing added to commit" depending on state.
**Rule:** For git integration code, test against real temporary repos. Git's output varies across states and versions — mocks can't capture this reliably.

## K015: Sidebar section insertion reduces downstream section height

**When:** M004 S01/T03
**What happened:** The auto-mode sidebar section is inserted between header and files. When active, it reduces the space available for files/LSPs/MCPs below it. The sidebar dynamically adjusts — autoModeInfo() returns empty string when autoSnapshot is nil so the sidebar renders normally when auto-mode is inactive.
**Rule:** When adding conditional sidebar sections, always make them no-op (return empty string) when their data source is nil. This preserves existing layout when the feature is inactive.

## K016: Shared builder functions prevent CLI/TUI wiring drift

**When:** M005 S04-S05
**What happened:** `buildAutoEngine()` in `internal/cmd/auto.go` constructs the full adapter graph (DB adapters, verifier, stuck detector, worktree manager, engine, controller) from config. Both `crush auto start` and the TUI path in `cmd/root.go` call the same function. This prevents the two entry points from diverging in how they wire dependencies.
**Rule:** When multiple entry points construct the same dependency graph, extract a shared builder function. Both CLI and TUI paths should call it. Test new components by adding to the builder, not by duplicating wiring.

## K017: Event unification — pick the richer type system as base

**When:** M005 S01
**What happened:** M003's event system had 12 types with `pubsub.EventType` constants and `NewAutoEvent()` constructor. M004's had 8 types with `AutoEventType` string type. Unifying on M003's system required adding one field (`Snapshot *AutoSnapshot`) and 4 constants. Unifying on M004's would have required rewriting every engine publish call and changing the broker type.
**Rule:** When merging overlapping type systems, pick the one with more consumers and richer infrastructure as the base. Fold missing pieces from the other into it.
