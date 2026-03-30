package auto

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetricsLedger_RecordAndAggregate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "metrics.json")
	m := NewMetricsLedger(path)

	m.Record(UnitMetrics{
		MilestoneID: "M001",
		UnitType:    string(UnitExecuteTask),
		Success:     true,
		DurationMs:  1500,
		Cost:        0.01,
		ModelTier:   "main",
	})
	m.Record(UnitMetrics{
		MilestoneID: "M001",
		UnitType:    string(UnitResearch),
		Success:     true,
		DurationMs:  2000,
		Cost:        0.005,
		ModelTier:   "planning",
	})
	m.Record(UnitMetrics{
		MilestoneID: "M001",
		UnitType:    string(UnitExecuteTask),
		Success:     false,
		DurationMs:  800,
		Cost:        0.003,
		ModelTier:   "main",
	})

	agg := m.Aggregate("M001")
	require.Equal(t, 3, agg.TotalUnits)
	require.Equal(t, 2, agg.SuccessCount)
	require.Equal(t, 1, agg.FailureCount)
	require.InDelta(t, 0.018, agg.TotalCost, 0.0001)
	require.Equal(t, int64(4300), agg.TotalDuration)
	require.InDelta(t, 0.006, agg.AvgCostPerUnit, 0.001)
}

func TestMetricsLedger_FilterByMilestone(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "metrics.json")
	m := NewMetricsLedger(path)

	m.Record(UnitMetrics{MilestoneID: "M001", Success: true, Cost: 0.01})
	m.Record(UnitMetrics{MilestoneID: "M002", Success: true, Cost: 0.02})

	agg1 := m.Aggregate("M001")
	require.Equal(t, 1, agg1.TotalUnits)
	require.InDelta(t, 0.01, agg1.TotalCost, 0.0001)

	aggAll := m.Aggregate("")
	require.Equal(t, 2, aggAll.TotalUnits)
	require.InDelta(t, 0.03, aggAll.TotalCost, 0.0001)
}

func TestMetricsLedger_Persistence(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "metrics.json")

	// Write entries.
	m1 := NewMetricsLedger(path)
	m1.Record(UnitMetrics{
		MilestoneID: "M001",
		UnitType:    "execute_task",
		Success:     true,
		DurationMs:  1000,
		Cost:        0.01,
	})

	// Reload from disk.
	m2 := NewMetricsLedger(path)
	agg := m2.Aggregate("")
	require.Equal(t, 1, agg.TotalUnits)
	require.InDelta(t, 0.01, agg.TotalCost, 0.0001)
}

func TestMetricsLedger_TotalCost(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "metrics.json")
	m := NewMetricsLedger(path)

	m.Record(UnitMetrics{MilestoneID: "M001", Cost: 0.01})
	m.Record(UnitMetrics{MilestoneID: "M001", Cost: 0.02})

	require.InDelta(t, 0.03, m.TotalCost("M001"), 0.0001)
}

func TestMetricsLedger_EmptyAggregate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "metrics.json")
	m := NewMetricsLedger(path)

	agg := m.Aggregate("M001")
	require.Equal(t, 0, agg.TotalUnits)
	require.Equal(t, float64(0), agg.TotalCost)
	require.Equal(t, int64(0), agg.AvgDurationMs)
}

func TestMetricsLedger_Entries(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "metrics.json")
	m := NewMetricsLedger(path)

	now := time.Now()
	m.Record(UnitMetrics{
		Timestamp:   now,
		MilestoneID: "M001",
		UnitType:    "research",
		Success:     true,
	})

	entries := m.Entries()
	require.Len(t, entries, 1)
	require.Equal(t, "M001", entries[0].MilestoneID)
}
