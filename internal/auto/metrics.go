package auto

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// UnitMetrics records the outcome of a single unit dispatch for aggregation.
type UnitMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	MilestoneID string    `json:"milestone_id"`
	UnitType    string    `json:"unit_type"`
	Success     bool      `json:"success"`
	DurationMs  int64     `json:"duration_ms"`
	Cost        float64   `json:"cost"`
	ModelTier   string    `json:"model_tier"`
}

// MetricsAggregate holds aggregated stats for display in /gsd status.
type MetricsAggregate struct {
	TotalUnits    int     `json:"total_units"`
	SuccessCount  int     `json:"success_count"`
	FailureCount  int     `json:"failure_count"`
	TotalCost     float64 `json:"total_cost"`
	TotalDuration int64   `json:"total_duration_ms"`
	AvgCostPerUnit float64 `json:"avg_cost_per_unit"`
	AvgDurationMs  int64   `json:"avg_duration_ms"`
}

// MetricsLedger persists unit metrics to an append-only JSON file and
// provides aggregation queries. Thread-safe.
type MetricsLedger struct {
	mu      sync.Mutex
	path    string
	entries []UnitMetrics
}

// NewMetricsLedger creates a ledger backed by the given file path.
// Loads existing entries on creation.
func NewMetricsLedger(path string) *MetricsLedger {
	m := &MetricsLedger{path: path}
	m.load()
	return m
}

// Record appends a metrics entry and persists to disk.
func (m *MetricsLedger) Record(entry UnitMetrics) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	m.mu.Lock()
	m.entries = append(m.entries, entry)
	m.mu.Unlock()

	m.save()
}

// Aggregate returns aggregated metrics, optionally filtered by milestone.
// Pass empty milestoneID to aggregate all entries.
func (m *MetricsLedger) Aggregate(milestoneID string) MetricsAggregate {
	m.mu.Lock()
	defer m.mu.Unlock()

	var agg MetricsAggregate
	for _, e := range m.entries {
		if milestoneID != "" && e.MilestoneID != milestoneID {
			continue
		}
		agg.TotalUnits++
		if e.Success {
			agg.SuccessCount++
		} else {
			agg.FailureCount++
		}
		agg.TotalCost += e.Cost
		agg.TotalDuration += e.DurationMs
	}

	if agg.TotalUnits > 0 {
		agg.AvgCostPerUnit = agg.TotalCost / float64(agg.TotalUnits)
		agg.AvgDurationMs = agg.TotalDuration / int64(agg.TotalUnits)
	}

	return agg
}

// TotalCost returns the cumulative cost across all entries for a milestone.
func (m *MetricsLedger) TotalCost(milestoneID string) float64 {
	return m.Aggregate(milestoneID).TotalCost
}

// Entries returns a copy of all recorded entries.
func (m *MetricsLedger) Entries() []UnitMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]UnitMetrics, len(m.entries))
	copy(out, m.entries)
	return out
}

// load reads existing entries from disk. Errors are silently ignored.
func (m *MetricsLedger) load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}

	var entries []UnitMetrics
	if err := json.Unmarshal(data, &entries); err != nil {
		return
	}

	m.mu.Lock()
	m.entries = entries
	m.mu.Unlock()
}

// save writes all entries to disk. Errors are silently ignored —
// metrics must never block the engine.
func (m *MetricsLedger) save() {
	m.mu.Lock()
	data, err := json.MarshalIndent(m.entries, "", "  ")
	m.mu.Unlock()
	if err != nil {
		return
	}

	dir := metricsDir(m.path)
	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	_ = os.WriteFile(m.path, data, 0o644)
}

// metricsDir returns the directory portion of a path.
func metricsDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}
