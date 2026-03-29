package auto

import (
	"context"
	"testing"
)

// fakeQuerier is an in-memory StateQuerier for testing DeriveState.
type fakeQuerier struct {
	milestones []MilestoneRow
	slices     map[string][]SliceRow // keyed by milestone ID
	tasks      map[string][]TaskRow  // keyed by slice ID
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		slices: make(map[string][]SliceRow),
		tasks:  make(map[string][]TaskRow),
	}
}

func (f *fakeQuerier) ListMilestones(_ context.Context) ([]MilestoneRow, error) {
	return f.milestones, nil
}

func (f *fakeQuerier) ListSlicesByMilestone(_ context.Context, milestoneID string) ([]SliceRow, error) {
	return f.slices[milestoneID], nil
}

func (f *fakeQuerier) ListTasksBySlice(_ context.Context, sliceID string) ([]TaskRow, error) {
	return f.tasks[sliceID], nil
}

func TestDeriveState_EmptyDB(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unit.IsDone() {
		t.Fatalf("expected done, got %s", unit)
	}
}

func TestDeriveState_NoActiveMilestone(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Pending milestone", Status: "pending", Phase: "pre_planning"},
	}
	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unit.IsDone() {
		t.Fatalf("expected done, got %s", unit)
	}
}

func TestDeriveState_SingleTaskDispatch(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "First milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "Core slice", Status: "active", Phase: "executing", SortOrder: 1},
	}
	q.tasks["S01"] = []TaskRow{
		{ID: "T01", Title: "First task", Status: "pending", Phase: "executing", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Type != UnitExecuteTask {
		t.Fatalf("expected execute_task, got %s", unit.Type)
	}
	if unit.TaskID != "T01" {
		t.Fatalf("expected T01, got %s", unit.TaskID)
	}
	if unit.SliceID != "S01" {
		t.Fatalf("expected S01, got %s", unit.SliceID)
	}
	if unit.MilestoneID != "M001" {
		t.Fatalf("expected M001, got %s", unit.MilestoneID)
	}
}

func TestDeriveState_MultiTaskOrdering(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "Slice", Status: "active", Phase: "executing", SortOrder: 1},
	}
	q.tasks["S01"] = []TaskRow{
		{ID: "T01", Title: "Task 1", Status: "completed", Phase: "completed", SortOrder: 1},
		{ID: "T02", Title: "Task 2", Status: "pending", Phase: "executing", SortOrder: 2},
		{ID: "T03", Title: "Task 3", Status: "pending", Phase: "executing", SortOrder: 3},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Type != UnitExecuteTask {
		t.Fatalf("expected execute_task, got %s", unit.Type)
	}
	if unit.TaskID != "T02" {
		t.Fatalf("expected T02 (first non-completed), got %s", unit.TaskID)
	}
}

func TestDeriveState_SliceDependencyBlocking(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "First", Status: "active", Phase: "executing", SortOrder: 1},
		{ID: "S02", Title: "Second", Status: "pending", Phase: "pre_planning", SortOrder: 2, DependsOn: "S01"},
	}
	q.tasks["S01"] = []TaskRow{
		{ID: "T01", Title: "Task", Status: "pending", Phase: "executing", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// S01 is dispatchable; S02 is blocked.
	if unit.Type != UnitExecuteTask {
		t.Fatalf("expected execute_task from S01, got %s", unit.Type)
	}
	if unit.SliceID != "S01" {
		t.Fatalf("expected S01, got %s", unit.SliceID)
	}
}

func TestDeriveState_BlockedSliceUnblocksAfterCompletion(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "First", Status: "completed", Phase: "completed", SortOrder: 1},
		{ID: "S02", Title: "Second", Status: "active", Phase: "executing", SortOrder: 2, DependsOn: "S01"},
	}
	q.tasks["S02"] = []TaskRow{
		{ID: "T01", Title: "Task in S02", Status: "pending", Phase: "executing", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.SliceID != "S02" {
		t.Fatalf("expected S02 (unblocked), got %s", unit.SliceID)
	}
}

func TestDeriveState_PhaseProgression(t *testing.T) {
	t.Parallel()

	phases := []struct {
		phase    string
		wantType UnitType
	}{
		{"pre_planning", UnitResearch},
		{"researching", UnitResearch},
		{"planning", UnitPlanSlice},
		{"executing", UnitExecuteTask},
		{"summarizing", UnitSummarizeSlice},
	}

	for _, tc := range phases {
		t.Run(tc.phase, func(t *testing.T) {
			t.Parallel()
			q := newFakeQuerier()
			q.milestones = []MilestoneRow{
				{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
			}
			q.slices["M001"] = []SliceRow{
				{ID: "S01", Title: "Slice", Status: "active", Phase: tc.phase, SortOrder: 1},
			}
			if tc.phase == "executing" {
				q.tasks["S01"] = []TaskRow{
					{ID: "T01", Title: "Task", Status: "pending", Phase: "executing", SortOrder: 1},
				}
			}

			unit, err := DeriveState(context.Background(), q)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if unit.Type != tc.wantType {
				t.Fatalf("phase %s: expected %s, got %s", tc.phase, tc.wantType, unit.Type)
			}
		})
	}
}

func TestDeriveState_AllTasksCompletedTriggersSliceSummarize(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "Slice", Status: "active", Phase: "executing", SortOrder: 1},
	}
	q.tasks["S01"] = []TaskRow{
		{ID: "T01", Title: "Task 1", Status: "completed", Phase: "completed", SortOrder: 1},
		{ID: "T02", Title: "Task 2", Status: "completed", Phase: "completed", SortOrder: 2},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Type != UnitSummarizeSlice {
		t.Fatalf("expected summarize_slice when all tasks done, got %s", unit.Type)
	}
}

func TestDeriveState_AllSlicesCompletedTriggersValidation(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "First", Status: "completed", Phase: "completed", SortOrder: 1},
		{ID: "S02", Title: "Second", Status: "completed", Phase: "completed", SortOrder: 2},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Type != UnitValidateMilestone {
		t.Fatalf("expected validate_milestone, got %s", unit.Type)
	}
	if unit.MilestoneID != "M001" {
		t.Fatalf("expected M001, got %s", unit.MilestoneID)
	}
}

func TestDeriveState_AllDone(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Done milestone", Status: "active", Phase: "completed"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "Slice", Status: "completed", Phase: "completed", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unit.IsDone() {
		t.Fatalf("expected done when milestone phase is completed, got %s", unit)
	}
}

func TestDeriveState_FullyBlocked(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	// Both slices depend on each other — fully blocked (degenerate case).
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "A", Status: "pending", Phase: "pre_planning", SortOrder: 1, DependsOn: "S02"},
		{ID: "S02", Title: "B", Status: "pending", Phase: "pre_planning", SortOrder: 2, DependsOn: "S01"},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unit.IsDone() {
		t.Fatalf("expected done (blocked), got %s", unit)
	}
}

func TestDeriveState_MultipleDependencies(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "A", Status: "completed", Phase: "completed", SortOrder: 1},
		{ID: "S02", Title: "B", Status: "completed", Phase: "completed", SortOrder: 2},
		{ID: "S03", Title: "C", Status: "active", Phase: "executing", SortOrder: 3, DependsOn: "S01,S02"},
	}
	q.tasks["S03"] = []TaskRow{
		{ID: "T01", Title: "Task", Status: "pending", Phase: "executing", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.SliceID != "S03" {
		t.Fatalf("expected S03 (both deps met), got %s", unit.SliceID)
	}
}

func TestDeriveState_PartialDependencyBlocking(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "executing"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "A", Status: "completed", Phase: "completed", SortOrder: 1},
		{ID: "S02", Title: "B", Status: "active", Phase: "executing", SortOrder: 2},
		{ID: "S03", Title: "C", Status: "pending", Phase: "pre_planning", SortOrder: 3, DependsOn: "S01,S02"},
	}
	q.tasks["S02"] = []TaskRow{
		{ID: "T01", Title: "Task in S02", Status: "pending", Phase: "executing", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// S02 is not complete, so S03 is blocked. Should dispatch S02 task.
	if unit.SliceID != "S02" {
		t.Fatalf("expected S02 (S03 blocked on S02), got %s", unit.SliceID)
	}
}

func TestDeriveState_ValidatingPhaseOnSlice(t *testing.T) {
	t.Parallel()
	q := newFakeQuerier()
	q.milestones = []MilestoneRow{
		{ID: "M001", Title: "Milestone", Status: "active", Phase: "validating"},
	}
	q.slices["M001"] = []SliceRow{
		{ID: "S01", Title: "Slice", Status: "active", Phase: "validating", SortOrder: 1},
	}

	unit, err := DeriveState(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit.Type != UnitValidateMilestone {
		t.Fatalf("expected validate_milestone, got %s", unit.Type)
	}
}

func TestUnit_IsDone(t *testing.T) {
	t.Parallel()
	if !(Unit{}).IsDone() {
		t.Fatal("zero-value Unit should be done")
	}
	if (Unit{Type: UnitExecuteTask}).IsDone() {
		t.Fatal("non-zero Unit should not be done")
	}
}

func TestUnit_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		unit Unit
		want string
	}{
		{"done", Unit{}, "done"},
		{"task", Unit{Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S01", TaskID: "T01", Title: "Do stuff"}, "execute_task M001/S01/T01: Do stuff"},
		{"slice", Unit{Type: UnitPlanSlice, MilestoneID: "M001", SliceID: "S01", Title: "Plan it"}, "plan_slice M001/S01: Plan it"},
		{"milestone", Unit{Type: UnitValidateMilestone, MilestoneID: "M001", Title: "Validate"}, "validate_milestone M001: Validate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.unit.String()
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUnitType_IsValid(t *testing.T) {
	t.Parallel()
	valid := []UnitType{UnitResearch, UnitPlanSlice, UnitExecuteTask, UnitSummarizeSlice, UnitValidateMilestone}
	for _, v := range valid {
		if !v.IsValid() {
			t.Fatalf("%s should be valid", v)
		}
	}
	if UnitType("bogus").IsValid() {
		t.Fatal("bogus should not be valid")
	}
}

func TestDepsMetFor(t *testing.T) {
	t.Parallel()
	completed := map[string]bool{"S01": true, "S02": true}

	cases := []struct {
		depends string
		want    bool
	}{
		{"", true},
		{"S01", true},
		{"S01,S02", true},
		{"S01, S02", true},
		{"S03", false},
		{"S01,S03", false},
	}
	for _, tc := range cases {
		got := depsMetFor(tc.depends, completed)
		if got != tc.want {
			t.Fatalf("depsMetFor(%q) = %v, want %v", tc.depends, got, tc.want)
		}
	}
}
