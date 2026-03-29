# S02: Interactive Planning — crush auto init — UAT

**Milestone:** M002
**Written:** 2026-03-27T22:19:50.519Z

# UAT: S02 — Interactive Planning — crush auto init

## Preconditions
- Crush built successfully (`go build .`)
- All auto package tests pass (`go test ./internal/auto/... -count=1`)
- No vet errors (`go vet ./internal/auto/... ./internal/cmd/...`)
- S01 auto tables migration applied (milestones, slices, tasks tables exist)

## Test Cases

### TC1: Planning Tools — Create Milestone with Active Status
1. Set up in-memory SQLite with goose migrations
2. Call create_milestone tool with `{"id": "M001", "title": "Build REST API"}`
3. **Expected:** Milestone M001 created with status=active (first milestone auto-activates)
4. Call create_milestone tool with `{"id": "M002", "title": "Add auth"}`
5. **Expected:** Milestone M002 created with status=pending (second milestone stays pending)

### TC2: Planning Tools — Create Slice with Parent Validation
1. Create milestone M001 first
2. Call create_slice tool with `{"id": "S01", "milestone_id": "M001", "title": "Setup project", "sort_order": 1}`
3. **Expected:** Slice S01 created with milestone_id=M001, status=pending, phase=pre_planning, sort_order=1
4. Call create_slice with `{"id": "S02", "milestone_id": "M001", "title": "Add endpoints", "sort_order": 2, "depends_on": "S01"}`
5. **Expected:** Slice S02 created with depends_on=S01, sort_order=2

### TC3: Planning Tools — Create Task with Full Relationships
1. Create milestone M001 and slice S01 first
2. Call create_task tool with `{"id": "T01", "slice_id": "S01", "milestone_id": "M001", "title": "Init go module", "description": "Run go mod init", "sort_order": 1}`
3. **Expected:** Task T01 created with correct parent relationships, status=pending, phase=pre_planning

### TC4: Planning Tools — Field Validation Errors
1. Call create_milestone with empty id: `{"id": "", "title": "Test"}`
2. **Expected:** JSON error response containing "id" and "required" (not a Go error)
3. Call create_slice with missing milestone_id: `{"id": "S01", "title": "Test"}`
4. **Expected:** JSON error response mentioning missing milestone_id
5. Call create_task with missing slice_id
6. **Expected:** JSON error response mentioning missing slice_id
7. Verify all errors are JSON text responses the LLM can parse and self-correct from

### TC5: BuildInitPrompt — Template Rendering
1. Call BuildInitPrompt with InitPromptContext{Vision: "Build a REST API with auth", WorkingDir: "/tmp/test"}
2. **Expected:** Returned string contains "Build a REST API with auth"
3. **Expected:** Returned string references create_milestone, create_slice, create_task tools
4. **Expected:** Returned string specifies ID conventions (M001, S01, T01)

### TC6: Full Integration — Tool Sequence Creates Structured Plan
1. Set up in-memory SQLite with migrations
2. Execute tool sequence simulating LLM: create_milestone(M001) → create_slice(S01, S02) → create_task(T01, T02, T03)
3. Verify 1 milestone in DB with status=active
4. Verify 2 slices with correct sort_order (1, 2) and status=pending
5. Verify 3 tasks with correct slice_id, milestone_id, and sort_order
6. Verify all phases are pre_planning

### TC7: CLI Registration — crush auto init
1. Run `crush auto init --help`
2. **Expected:** Shows usage with vision positional argument
3. **Expected:** Command is registered under the `crush auto` parent command

## Edge Cases

### EC1: Duplicate Milestone ID
- Call create_milestone twice with same ID
- **Expected:** Second call returns error (DB unique constraint)

### EC2: Very Long Vision String
- Pass 5000+ character vision to BuildInitPrompt
- **Expected:** Template renders without truncation

### EC3: Sort Order Zero
- Create slice with sort_order=0
- **Expected:** Tool accepts it (no minimum constraint)

### EC4: DeriveState Integration
- After init creates milestone with status=active, call DeriveState()
- **Expected:** Returns the first dispatchable unit from the newly-created plan (research unit for first slice)
