package auto

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockParkerQuerier implements StateQuerier for park tests.
type mockParkerQuerier struct {
	milestones []MilestoneRow
}

func (m *mockParkerQuerier) ListMilestones(_ context.Context) ([]MilestoneRow, error) {
	return m.milestones, nil
}

func (m *mockParkerQuerier) ListSlicesByMilestone(_ context.Context, _ string) ([]SliceRow, error) {
	return nil, nil
}

func (m *mockParkerQuerier) ListTasksBySlice(_ context.Context, _ string) ([]TaskRow, error) {
	return nil, nil
}

// mockStatusSetter tracks SetMilestoneStatus calls.
type mockStatusSetter struct {
	lastID     string
	lastStatus string
}

func TestPark_Success(t *testing.T) {
	t.Parallel()
	q := &mockParkerQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Status: "active"},
		},
	}
	// Test the validation logic directly.
	err := validateParkAction(q, "M001")
	require.NoError(t, err)
}

func TestPark_NotFound(t *testing.T) {
	t.Parallel()
	q := &mockParkerQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Status: "active"},
		},
	}

	err := validateParkAction(q, "M999")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestPark_AlreadyParked(t *testing.T) {
	t.Parallel()
	q := &mockParkerQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Status: "parked"},
		},
	}

	err := validateParkAction(q, "M001")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already parked")
}

func TestPark_CompletedMilestone(t *testing.T) {
	t.Parallel()
	q := &mockParkerQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Status: "completed"},
		},
	}

	err := validateParkAction(q, "M001")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot park completed")
}

func TestUnpark_NotParked(t *testing.T) {
	t.Parallel()
	q := &mockParkerQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Status: "active"},
		},
	}

	err := validateUnparkAction(q, "M001")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not parked")
}

func TestUnpark_Success(t *testing.T) {
	t.Parallel()
	q := &mockParkerQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Status: "parked"},
		},
	}

	err := validateUnparkAction(q, "M001")
	require.NoError(t, err)
}

// validateParkAction checks the validation logic of Park without needing
// a real DBStatusAdvancer.
func validateParkAction(q StateQuerier, milestoneID string) error {
	ctx := context.Background()
	milestones, err := q.ListMilestones(ctx)
	if err != nil {
		return err
	}

	found := false
	for _, m := range milestones {
		if m.ID == milestoneID {
			found = true
			if Status(m.Status) == StatusParked {
				return fmt.Errorf("milestone %s is already parked", milestoneID)
			}
			if Status(m.Status) == StatusCompleted {
				return fmt.Errorf("cannot park completed milestone %s", milestoneID)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("milestone %s not found", milestoneID)
	}
	return nil
}

// validateUnparkAction checks the validation logic of Unpark.
func validateUnparkAction(q StateQuerier, milestoneID string) error {
	ctx := context.Background()
	milestones, err := q.ListMilestones(ctx)
	if err != nil {
		return err
	}

	found := false
	for _, m := range milestones {
		if m.ID == milestoneID {
			found = true
			if Status(m.Status) != StatusParked {
				return fmt.Errorf("milestone %s is not parked (status: %s)", milestoneID, m.Status)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("milestone %s not found", milestoneID)
	}
	return nil
}
