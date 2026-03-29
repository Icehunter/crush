package auto

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupAdapterTestDB opens a shared-cache in-memory SQLite database with
// migrations applied. Uses a unique DSN per test so parallel tests don't
// collide. Returns *db.Queries for both seeding and adapter construction.
func setupAdapterTestDB(t *testing.T) *db.Queries {
	t.Helper()

	params := url.Values{}
	params.Add("_pragma", "foreign_keys(ON)")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&%s", t.Name(), params.Encode())

	sqlDB, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	goose.SetBaseFS(db.FS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(sqlDB, "migrations"))

	return db.New(sqlDB)
}

// ---------------------------------------------------------------------------
// DB seeding helpers
// ---------------------------------------------------------------------------

// seedMilestone creates a milestone with the given ID, status, and phase.
func seedMilestone(t *testing.T, q *db.Queries, id, status, phase string) {
	t.Helper()
	_, err := q.CreateMilestone(context.Background(), db.CreateMilestoneParams{
		ID: id, Title: "Milestone " + id, Status: status, Phase: phase,
	})
	require.NoError(t, err)
}

// seedSlice creates a slice under a milestone.
func seedSlice(t *testing.T, q *db.Queries, id, milestoneID, status, phase string, sortOrder int64, dependsOn string) {
	t.Helper()
	dep := sql.NullString{}
	if dependsOn != "" {
		dep = sql.NullString{String: dependsOn, Valid: true}
	}
	_, err := q.CreateSlice(context.Background(), db.CreateSliceParams{
		ID: id, MilestoneID: milestoneID, Title: "Slice " + id,
		Status: status, Phase: phase, SortOrder: sortOrder, DependsOn: dep,
	})
	require.NoError(t, err)
}

// seedTask creates a task under a slice.
func seedTask(t *testing.T, q *db.Queries, id, sliceID, milestoneID, status, phase string, sortOrder int64) {
	t.Helper()
	_, err := q.CreateTask(context.Background(), db.CreateTaskParams{
		ID: id, SliceID: sliceID, MilestoneID: milestoneID, Title: "Task " + id,
		Status: status, Phase: phase, SortOrder: sortOrder,
	})
	require.NoError(t, err)
}

// seedSession creates a session with known token counts.
func seedSession(t *testing.T, q *db.Queries, id string, parentID sql.NullString, promptTokens, completionTokens int64) {
	t.Helper()
	_, err := q.CreateSession(context.Background(), db.CreateSessionParams{
		ID: id, ParentSessionID: parentID, Title: "Session " + id,
		PromptTokens: promptTokens, CompletionTokens: completionTokens,
	})
	require.NoError(t, err)
}

// seedUpdateTaskStatus updates a task's status in the DB.
func seedUpdateTaskStatus(t *testing.T, q *db.Queries, id, status string) {
	t.Helper()
	_, err := q.UpdateTaskStatus(context.Background(), db.UpdateTaskStatusParams{
		ID:     id,
		Status: status,
	})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// TestDBStateQuerier
// ---------------------------------------------------------------------------

func TestDBStateQuerier(t *testing.T) {
	t.Parallel()
	q := setupAdapterTestDB(t)
	ctx := context.Background()
	sq := NewDBStateQuerier(q)

	// Seed data: 1 milestone, 2 slices (one with DependsOn), 2 tasks.
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "executing", 1, "")
	seedSlice(t, q, "S02", "M001", "pending", "pre_planning", 2, "S01")
	seedTask(t, q, "T01", "S01", "M001", "pending", "pre_planning", 1)
	seedTask(t, q, "T02", "S01", "M001", "completed", "completed", 2)

	t.Run("ListMilestones", func(t *testing.T) {
		t.Parallel()
		rows, err := sq.ListMilestones(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, "M001", rows[0].ID)
		require.Equal(t, "Milestone M001", rows[0].Title)
		require.Equal(t, "active", rows[0].Status)
		require.Equal(t, "executing", rows[0].Phase)
	})

	t.Run("ListSlicesByMilestone", func(t *testing.T) {
		t.Parallel()
		rows, err := sq.ListSlicesByMilestone(ctx, "M001")
		require.NoError(t, err)
		require.Len(t, rows, 2)

		// Ordered by sort_order.
		require.Equal(t, "S01", rows[0].ID)
		require.Equal(t, int64(1), rows[0].SortOrder)
		require.Equal(t, "", rows[0].DependsOn) // Empty string for no deps.

		require.Equal(t, "S02", rows[1].ID)
		require.Equal(t, int64(2), rows[1].SortOrder)
		require.Equal(t, "S01", rows[1].DependsOn) // Non-empty DependsOn.
	})

	t.Run("ListTasksBySlice", func(t *testing.T) {
		t.Parallel()
		rows, err := sq.ListTasksBySlice(ctx, "S01")
		require.NoError(t, err)
		require.Len(t, rows, 2)

		// Ordered by sort_order.
		require.Equal(t, "T01", rows[0].ID)
		require.Equal(t, "Task T01", rows[0].Title)
		require.Equal(t, "pending", rows[0].Status)
		require.Equal(t, "pre_planning", rows[0].Phase)
		require.Equal(t, int64(1), rows[0].SortOrder)

		require.Equal(t, "T02", rows[1].ID)
		require.Equal(t, "completed", rows[1].Status)
		require.Equal(t, "completed", rows[1].Phase)
	})

	t.Run("ListSlicesByMilestone_Empty", func(t *testing.T) {
		t.Parallel()
		rows, err := sq.ListSlicesByMilestone(ctx, "NONEXISTENT")
		require.NoError(t, err)
		require.Empty(t, rows)
	})

	t.Run("ListTasksBySlice_Empty", func(t *testing.T) {
		t.Parallel()
		rows, err := sq.ListTasksBySlice(ctx, "NONEXISTENT")
		require.NoError(t, err)
		require.Empty(t, rows)
	})
}

// ---------------------------------------------------------------------------
// TestDBStatusAdvancer
// ---------------------------------------------------------------------------

func TestDBStatusAdvancer(t *testing.T) {
	t.Parallel()
	q := setupAdapterTestDB(t)
	ctx := context.Background()
	adv := NewDBStatusAdvancer(q)

	// Seed entities for each transition.
	seedMilestone(t, q, "M001", "active", "executing")
	seedSlice(t, q, "S01", "M001", "active", "pre_planning", 1, "")
	seedSlice(t, q, "S02", "M001", "active", "planning", 2, "")
	seedSlice(t, q, "S03", "M001", "active", "executing", 3, "")
	seedTask(t, q, "T01", "S03", "M001", "active", "executing", 1)

	t.Run("UnitResearch_sets_slice_phase_to_planning", func(t *testing.T) {
		t.Parallel()
		err := adv.AdvanceStatus(ctx, Unit{
			Type: UnitResearch, MilestoneID: "M001", SliceID: "S01",
		})
		require.NoError(t, err)

		s, err := q.GetSlice(ctx, "S01")
		require.NoError(t, err)
		require.Equal(t, string(PhasePlanning), s.Phase)
	})

	t.Run("UnitPlanSlice_sets_slice_phase_to_executing", func(t *testing.T) {
		t.Parallel()
		err := adv.AdvanceStatus(ctx, Unit{
			Type: UnitPlanSlice, MilestoneID: "M001", SliceID: "S02",
		})
		require.NoError(t, err)

		s, err := q.GetSlice(ctx, "S02")
		require.NoError(t, err)
		require.Equal(t, string(PhaseExecuting), s.Phase)
	})

	t.Run("UnitExecuteTask_sets_task_completed", func(t *testing.T) {
		t.Parallel()
		err := adv.AdvanceStatus(ctx, Unit{
			Type: UnitExecuteTask, MilestoneID: "M001", SliceID: "S03", TaskID: "T01",
		})
		require.NoError(t, err)

		task, err := q.GetTask(ctx, "T01")
		require.NoError(t, err)
		require.Equal(t, string(StatusCompleted), task.Status)
		require.Equal(t, string(PhaseCompleted), task.Phase)
	})

	t.Run("UnitSummarizeSlice_sets_slice_completed", func(t *testing.T) {
		t.Parallel()

		// Use a dedicated slice to avoid racing with other subtests.
		seedSlice(t, q, "S04", "M001", "active", "executing", 4, "")
		err := adv.AdvanceStatus(ctx, Unit{
			Type: UnitSummarizeSlice, MilestoneID: "M001", SliceID: "S04",
		})
		require.NoError(t, err)

		s, err := q.GetSlice(ctx, "S04")
		require.NoError(t, err)
		require.Equal(t, string(StatusCompleted), s.Status)
		require.Equal(t, string(PhaseCompleted), s.Phase)
	})

	t.Run("UnitValidateMilestone_sets_milestone_completed", func(t *testing.T) {
		t.Parallel()

		// Use a dedicated milestone to avoid racing with other subtests.
		seedMilestone(t, q, "M002", "active", "validating")
		err := adv.AdvanceStatus(ctx, Unit{
			Type: UnitValidateMilestone, MilestoneID: "M002",
		})
		require.NoError(t, err)

		m, err := q.GetMilestone(ctx, "M002")
		require.NoError(t, err)
		require.Equal(t, string(StatusCompleted), m.Status)
		require.Equal(t, string(PhaseCompleted), m.Phase)
	})

	t.Run("UnknownUnitType_returns_error", func(t *testing.T) {
		t.Parallel()
		err := adv.AdvanceStatus(ctx, Unit{Type: "bogus"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown unit type")
	})
}

// ---------------------------------------------------------------------------
// TestDBTokenQuerier
// ---------------------------------------------------------------------------

func TestDBTokenQuerier(t *testing.T) {
	t.Parallel()
	q := setupAdapterTestDB(t)
	ctx := context.Background()
	tq := NewDBTokenQuerier(q)

	parentNS := sql.NullString{String: "parent-1", Valid: true}

	// Create parent session.
	seedSession(t, q, "parent-1", sql.NullString{}, 0, 0)

	// Create child sessions under that parent.
	seedSession(t, q, "child-1", parentNS, 100, 50)
	seedSession(t, q, "child-2", parentNS, 200, 75)

	t.Run("sums_child_tokens", func(t *testing.T) {
		t.Parallel()
		prompt, completion, err := tq.GetTokenUsage(ctx, "parent-1")
		require.NoError(t, err)
		require.Equal(t, int64(300), prompt)     // 100 + 200
		require.Equal(t, int64(125), completion) // 50 + 75
	})

	t.Run("returns_zero_for_no_children", func(t *testing.T) {
		t.Parallel()
		seedSession(t, q, "lonely", sql.NullString{}, 99, 99)
		prompt, completion, err := tq.GetTokenUsage(ctx, "lonely")
		require.NoError(t, err)
		require.Equal(t, int64(0), prompt)
		require.Equal(t, int64(0), completion)
	})
}

// ---------------------------------------------------------------------------
// TestSessionServiceCreator — mock-based
// ---------------------------------------------------------------------------

// mockSessionService is a minimal fake implementing session.Service methods
// used by SessionServiceCreator.
type mockSessionService struct {
	session.Service // Embed to satisfy unused methods.
	createFn        func(ctx context.Context, title string) (session.Session, error)
	createTaskFn    func(ctx context.Context, id, parentID, title string) (session.Session, error)
}

func (m *mockSessionService) Create(ctx context.Context, title string) (session.Session, error) {
	return m.createFn(ctx, title)
}

func (m *mockSessionService) CreateTaskSession(ctx context.Context, id, parentID, title string) (session.Session, error) {
	return m.createTaskFn(ctx, id, parentID, title)
}

func TestSessionServiceCreator(t *testing.T) {
	t.Parallel()

	t.Run("CreateSession_returns_ID", func(t *testing.T) {
		t.Parallel()
		svc := &mockSessionService{
			createFn: func(_ context.Context, title string) (session.Session, error) {
				return session.Session{ID: "sess-abc", Title: title}, nil
			},
		}
		creator := NewSessionServiceCreator(svc)
		id, err := creator.CreateSession(context.Background(), "test session")
		require.NoError(t, err)
		require.Equal(t, "sess-abc", id)
	})

	t.Run("CreateSession_propagates_error", func(t *testing.T) {
		t.Parallel()
		svc := &mockSessionService{
			createFn: func(_ context.Context, _ string) (session.Session, error) {
				return session.Session{}, errors.New("db down")
			},
		}
		creator := NewSessionServiceCreator(svc)
		_, err := creator.CreateSession(context.Background(), "fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "db down")
	})

	t.Run("CreateChildSession_returns_ID_with_parent", func(t *testing.T) {
		t.Parallel()
		var capturedParent string
		svc := &mockSessionService{
			createTaskFn: func(_ context.Context, id, parentID, title string) (session.Session, error) {
				capturedParent = parentID
				return session.Session{ID: id, ParentSessionID: parentID, Title: title}, nil
			},
		}
		creator := NewSessionServiceCreator(svc)
		id, err := creator.CreateChildSession(context.Background(), "child-1", "parent-1", "child title")
		require.NoError(t, err)
		require.Equal(t, "child-1", id)
		require.Equal(t, "parent-1", capturedParent)
	})

	t.Run("CreateChildSession_propagates_error", func(t *testing.T) {
		t.Parallel()
		svc := &mockSessionService{
			createTaskFn: func(_ context.Context, _, _, _ string) (session.Session, error) {
				return session.Session{}, errors.New("constraint violation")
			},
		}
		creator := NewSessionServiceCreator(svc)
		_, err := creator.CreateChildSession(context.Background(), "c", "p", "t")
		require.Error(t, err)
		require.Contains(t, err.Error(), "constraint violation")
	})
}

// ---------------------------------------------------------------------------
// TestCoordinatorDispatcher — mock-based
// ---------------------------------------------------------------------------

// mockCoordinator records RunWithForcedTier calls and returns configurable
// results. It satisfies agent.Coordinator — unused methods panic.
type mockCoordinator struct {
	calls []mockCoordinatorCall
	err   error
}

// Compile-time check.
var _ agent.Coordinator = (*mockCoordinator)(nil)

type mockCoordinatorCall struct {
	SessionID string
	Prompt    string
	Tier      config.SelectedModelType
}

func (m *mockCoordinator) Run(_ context.Context, _, _ string, _ ...message.Attachment) (*fantasy.AgentResult, error) {
	panic("unexpected call to Run")
}

func (m *mockCoordinator) RunWithForcedTier(_ context.Context, sessionID, prompt string, tier config.SelectedModelType, _ ...message.Attachment) (*fantasy.AgentResult, error) {
	m.calls = append(m.calls, mockCoordinatorCall{
		SessionID: sessionID, Prompt: prompt, Tier: tier,
	})
	return nil, m.err
}

func (m *mockCoordinator) Cancel(_ string)                             {}
func (m *mockCoordinator) CancelAll()                                  {}
func (m *mockCoordinator) IsSessionBusy(_ string) bool                 { return false }
func (m *mockCoordinator) IsBusy() bool                                { return false }
func (m *mockCoordinator) QueuedPrompts(_ string) int                  { return 0 }
func (m *mockCoordinator) QueuedPromptsList(_ string) []string         { return nil }
func (m *mockCoordinator) ClearQueue(_ string)                         {}
func (m *mockCoordinator) Summarize(_ context.Context, _ string) error { return nil }
func (m *mockCoordinator) Model() agent.Model                          { return agent.Model{} }
func (m *mockCoordinator) UpdateModels(_ context.Context) error        { return nil }

func TestCoordinatorDispatcher(t *testing.T) {
	t.Parallel()

	t.Run("passes_through_and_discards_result", func(t *testing.T) {
		t.Parallel()
		mock := &mockCoordinator{}
		d := NewCoordinatorDispatcher(mock)
		err := d.RunWithForcedTier(context.Background(), "sess-1", "hello", config.SelectedModelTypeMain)
		require.NoError(t, err)
		require.Len(t, mock.calls, 1)
		require.Equal(t, "sess-1", mock.calls[0].SessionID)
		require.Equal(t, "hello", mock.calls[0].Prompt)
		require.Equal(t, config.SelectedModelTypeMain, mock.calls[0].Tier)
	})

	t.Run("propagates_error", func(t *testing.T) {
		t.Parallel()
		mock := &mockCoordinator{err: errors.New("timeout")}
		d := NewCoordinatorDispatcher(mock)
		err := d.RunWithForcedTier(context.Background(), "sess-2", "prompt", config.SelectedModelTypeBackground)
		require.Error(t, err)
		require.Contains(t, err.Error(), "timeout")
	})
}
