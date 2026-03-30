package auto

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoutingHistory_Rate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")
	rh := NewRoutingHistory(path)

	rh.Rate("execute_task", "main", "ok")
	rh.Rate("execute_task", "main", "over")

	summary := rh.Summary()
	require.Contains(t, summary, "execute_task")
	require.Contains(t, summary, "ok=1")
	require.Contains(t, summary, "over=1")
}

func TestRoutingHistory_SuggestNotEnoughData(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")
	rh := NewRoutingHistory(path)

	rh.Rate("research", "planning", "over")
	require.Equal(t, "", rh.Suggest("research"), "need at least 3 ratings")
}

func TestRoutingHistory_SuggestDown(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")
	rh := NewRoutingHistory(path)

	rh.Rate("execute_task", "main", "over")
	rh.Rate("execute_task", "main", "over")
	rh.Rate("execute_task", "main", "ok")

	require.Equal(t, "down", rh.Suggest("execute_task"))
}

func TestRoutingHistory_SuggestUp(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")
	rh := NewRoutingHistory(path)

	rh.Rate("research", "background", "under")
	rh.Rate("research", "background", "under")
	rh.Rate("research", "background", "ok")

	require.Equal(t, "up", rh.Suggest("research"))
}

func TestRoutingHistory_SuggestNoChange(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")
	rh := NewRoutingHistory(path)

	rh.Rate("plan_slice", "planning", "ok")
	rh.Rate("plan_slice", "planning", "ok")
	rh.Rate("plan_slice", "planning", "ok")

	require.Equal(t, "", rh.Suggest("plan_slice"))
}

func TestRoutingHistory_Persistence(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")

	rh1 := NewRoutingHistory(path)
	rh1.Rate("execute_task", "main", "ok")

	rh2 := NewRoutingHistory(path)
	summary := rh2.Summary()
	require.Contains(t, summary, "execute_task")
}

func TestRoutingHistory_EmptySummary(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "routing.json")
	rh := NewRoutingHistory(path)
	require.Contains(t, rh.Summary(), "No tier ratings")
}
