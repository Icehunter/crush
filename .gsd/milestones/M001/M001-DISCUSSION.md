# M001 Discussion Log

## Exchange — 2026-03-27T09:39:43.117Z

### M002 Gate

M002 (Auto Loop + Session Management) — it has rich seed material from this conversation. How do you want to handle it?

- **Discuss now (Recommended)** — Conduct a focused discussion for M002 now while context is fresh. Full discuss → context flow.
- **Write draft for later** — M002 has seed material from this conversation. Write a draft context for future discussion.
- **Just queue it** — Just create the directory. Full discussion from scratch later.

**Selected:** Discuss now (Recommended)

---
## Exchange — 2026-03-27T09:41:01.018Z

### Auto agent

Should auto-mode dispatch units through the existing Coordinator.Run() with synthesized prompts (max reuse), or build a separate auto-specific agent with custom prompts and potentially different tools per unit type?

- **Reuse Coordinator (Recommended)** — Auto-mode dispatches via existing Coordinator.Run() with a synthesized prompt. Reuses all existing agent infrastructure.
- **Custom auto agent** — Auto-mode builds its own agent with custom system prompts and a stripped-down tool set per unit type.

**Selected:** Reuse Coordinator (Recommended)

---
## Exchange — 2026-03-27T09:41:25.663Z

### Plan seeding UX

For Crush-native planning — how should the user seed the initial milestone? An explicit `crush auto init` interactive command, or just using `crush run` with a planning prompt?

- **Interactive init command (Recommended)** — `crush auto init` opens an interactive session where the user describes the vision. Crush generates the milestone/slice/task structure.
- **Prompt via crush run** — User runs `crush run 'plan this project: ...'` and the response seeds the DB. More ad-hoc.

**Selected:** Interactive init command (Recommended)

### Model tier

Should auto-mode use different model tiers for different unit types (planning model for research/planning, main for execution, background for summaries)?

- **Tier by unit type (Recommended)** — Research and planning units use the planning model. Execution uses the main model. Background for summaries.
- **Main model for everything** — Always use the main model for everything in auto-mode. Simpler, potentially more expensive.

**Selected:** Tier by unit type (Recommended)

---
## Exchange — 2026-03-27T09:42:08.763Z

### Auto state

What should `crush auto status` show before the TUI dashboard exists (M002-M003)? A rich structured status, or minimal until M004 adds the sidebar?

- **Structured status output (Recommended)** — `crush auto status` shows a structured view: active milestone/slice/task, phase, cost so far, recent actions. Quick and useful.
- **Minimal until M004** — Minimal: just 'running/paused/stopped' and the current unit. Details via the TUI dashboard in M004.

**Selected:** Structured status output (Recommended)

### Pause behavior

When `crush auto pause` is issued, should it finish the current unit then pause, or immediately cancel the in-progress unit?

- **Finish current then pause (Recommended)** — Let the current unit finish, then pause before dispatching the next. Clean boundary.
- **Immediate cancel** — Cancel the current unit immediately. May lose in-progress work.

**Selected:** Finish current then pause (Recommended)

---
## Exchange — 2026-03-27T09:42:53.516Z

### M002 Depth

M002 depth check — does the technical approach (reuse Coordinator with tier-by-unit-type, interactive init, structured status, finish-then-pause) capture it right?

- **Yes, write it (Recommended)** — Proceed to write the full M002 context.
- **Not quite** — I want to clarify or adjust something.

**Selected:** Yes, write it (Recommended)

---
## Exchange — 2026-03-27T09:44:31.510Z

### M003 Gate

M003 (Safety Rails) — verification gates, budget, stuck detection. The decisions are already locked from earlier. How do you want to handle it?

- **Discuss now (Recommended)** — Conduct a focused discussion for M003 now. The seed material from this conversation covers it well.
- **Write draft for later** — M003 has solid seed material. Write a draft context for future refinement.
- **Just queue it** — Just create the directory. Full discussion from scratch later.

**Selected:** Discuss now (Recommended)

---
## Exchange — 2026-03-27T09:45:38.551Z

### M003 Depth

M003 depth check — verification via shell commands, dollar budget via session cost aggregation, stuck detection via sliding window, context pressure via TotalUsage. Capture it right?

- **Yes, write it (Recommended)** — Proceed to write the full M003 context.
- **Not quite** — I want to clarify or adjust something.

**Selected:** Yes, write it (Recommended)

---
## Exchange — 2026-03-27T09:47:03.410Z

### M004 Gate

M004 (TUI + Worktrees) depends heavily on the TUI codebase which may change by the time M003 ships. Draft now with seed material, or defer entirely?

- **Write draft for later (Recommended)** — Write a draft context capturing TUI patterns and worktree approach. Dedicated discussion later when M003 ships.
- **Discuss now** — Conduct a focused discussion for M004 now.
- **Just queue it** — Just create the directory. Full discussion from scratch later.

**Selected:** Discuss now

---
## Exchange — 2026-03-27T09:47:45.536Z

### M004 Depth

M004 depth check — sidebar section within existing 30-char panel, keybindings for auto control, pub/sub event consumption via tea.Cmd, git worktree via shell-out. Capture it right?

- **Yes, write it (Recommended)** — Proceed to write the full M004 context.
- **Not quite** — I want to clarify or adjust something.

**Selected:** Yes, write it (Recommended)

---
