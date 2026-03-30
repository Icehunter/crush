package auto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveQueue_ResearchPhase(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "active", Phase: "researching", SortOrder: 1},
			},
		},
	}

	queue, err := DeriveQueue(context.Background(), q, "M001")
	require.NoError(t, err)
	require.Contains(t, queue[0], "research S01")
	require.Contains(t, queue[0], "Auth")
	// Should include follow-up phases.
	require.GreaterOrEqual(t, len(queue), 4) // research + plan + execute + summarize + validate
}

func TestDeriveQueue_ExecutingPhase(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "active", Phase: "executing", SortOrder: 1},
			},
		},
		tasks: map[string][]TaskRow{
			"S01": {
				{ID: "T01", Title: "Login", Status: "active", SortOrder: 1},
				{ID: "T02", Title: "Logout", Status: "pending", SortOrder: 2},
				{ID: "T03", Title: "Token", Status: "completed", SortOrder: 3},
			},
		},
	}

	queue, err := DeriveQueue(context.Background(), q, "M001")
	require.NoError(t, err)
	// Should list 2 incomplete tasks + summarize + validate.
	require.Contains(t, queue[0], "execute T01")
	require.Contains(t, queue[1], "execute T02")
	require.Contains(t, queue[2], "summarize S01")
}

func TestDeriveQueue_AllCompleted(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "completed", Phase: "completed", SortOrder: 1},
			},
		},
	}

	queue, err := DeriveQueue(context.Background(), q, "M001")
	require.NoError(t, err)
	require.Empty(t, queue)
}

func TestDeriveQueue_BlockedSlice(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {
				{ID: "S01", Title: "Auth", Status: "active", Phase: "researching", SortOrder: 1, DependsOn: "S00"},
			},
		},
	}

	queue, err := DeriveQueue(context.Background(), q, "M001")
	require.NoError(t, err)
	require.Contains(t, queue[0], "[blocked]")
}
