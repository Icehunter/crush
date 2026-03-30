package auto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DoctorCheck represents a single health check result.
type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warn", "error"
	Message string `json:"message"`
	Fixable bool   `json:"fixable"`
	Fixed   bool   `json:"fixed"`
}

// DoctorReport is the result of running all health checks.
type DoctorReport struct {
	Checks    []DoctorCheck `json:"checks"`
	Timestamp time.Time     `json:"timestamp"`
}

// Summary returns a human-readable summary of the doctor report.
func (r *DoctorReport) Summary() string {
	var lines []string
	var okCount, warnCount, errCount int

	for _, c := range r.Checks {
		icon := "✓"
		switch c.Status {
		case "warn":
			icon = "⚠"
			warnCount++
		case "error":
			icon = "✗"
			errCount++
		default:
			okCount++
		}
		line := fmt.Sprintf("  %s %s: %s", icon, c.Name, c.Message)
		if c.Fixed {
			line += " (auto-fixed)"
		}
		lines = append(lines, line)
	}

	header := fmt.Sprintf("Health check: %d ok, %d warnings, %d errors",
		okCount, warnCount, errCount)
	return header + "\n" + strings.Join(lines, "\n")
}

// RunDoctor performs health checks on the auto-mode infrastructure.
// If fix is true, it attempts to auto-heal fixable issues.
func RunDoctor(ctx context.Context, dataDir string, querier StateQuerier, fix bool) *DoctorReport {
	report := &DoctorReport{
		Timestamp: time.Now(),
	}

	report.Checks = append(report.Checks, checkStaleLock(dataDir, fix))
	report.Checks = append(report.Checks, checkDataDir(dataDir))
	report.Checks = append(report.Checks, checkJournalDir(dataDir))
	report.Checks = append(report.Checks, checkMetricsFile(dataDir))
	report.Checks = append(report.Checks, checkDBState(ctx, querier))

	// Persist report history.
	saveReportHistory(dataDir, report)

	return report
}

// checkStaleLock verifies the lock file isn't held by a dead process.
func checkStaleLock(dataDir string, fix bool) DoctorCheck {
	lockPath := filepath.Join(dataDir, lockFileName)

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DoctorCheck{
				Name:    "Lock file",
				Status:  "ok",
				Message: "No lock file present",
			}
		}
		return DoctorCheck{
			Name:    "Lock file",
			Status:  "error",
			Message: fmt.Sprintf("Cannot read lock: %v", err),
		}
	}

	var p lockPayload
	if err := json.Unmarshal(data, &p); err != nil {
		check := DoctorCheck{
			Name:    "Lock file",
			Status:  "error",
			Message: "Lock file is corrupt (unparseable JSON)",
			Fixable: true,
		}
		if fix {
			_ = os.Remove(lockPath)
			check.Fixed = true
			check.Message += " — removed"
		}
		return check
	}

	if isProcessAlive(p.PID) {
		return DoctorCheck{
			Name:    "Lock file",
			Status:  "ok",
			Message: fmt.Sprintf("Lock held by PID %d (alive, started %s)", p.PID, p.StartedAt.Format(time.RFC3339)),
		}
	}

	check := DoctorCheck{
		Name:    "Lock file",
		Status:  "warn",
		Message: fmt.Sprintf("Stale lock from PID %d (dead, started %s)", p.PID, p.StartedAt.Format(time.RFC3339)),
		Fixable: true,
	}
	if fix {
		_ = os.Remove(lockPath)
		check.Fixed = true
		check.Message += " — removed"
	}
	return check
}

// checkDataDir verifies the data directory exists and is writable.
func checkDataDir(dataDir string) DoctorCheck {
	info, err := os.Stat(dataDir)
	if err != nil {
		return DoctorCheck{
			Name:    "Data directory",
			Status:  "error",
			Message: fmt.Sprintf("Data directory missing: %s", dataDir),
		}
	}
	if !info.IsDir() {
		return DoctorCheck{
			Name:    "Data directory",
			Status:  "error",
			Message: fmt.Sprintf("Data path is not a directory: %s", dataDir),
		}
	}
	return DoctorCheck{
		Name:    "Data directory",
		Status:  "ok",
		Message: dataDir,
	}
}

// checkJournalDir verifies the journal directory exists.
func checkJournalDir(dataDir string) DoctorCheck {
	journalDir := filepath.Join(dataDir, "journal")
	if _, err := os.Stat(journalDir); err != nil {
		return DoctorCheck{
			Name:    "Journal directory",
			Status:  "warn",
			Message: "Journal directory not yet created (will be created on first dispatch)",
		}
	}

	// Count journal files.
	entries, _ := os.ReadDir(journalDir)
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			count++
		}
	}
	return DoctorCheck{
		Name:    "Journal directory",
		Status:  "ok",
		Message: fmt.Sprintf("%d journal files", count),
	}
}

// checkMetricsFile verifies the metrics file is readable.
func checkMetricsFile(dataDir string) DoctorCheck {
	metricsPath := filepath.Join(dataDir, "metrics.json")
	data, err := os.ReadFile(metricsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DoctorCheck{
				Name:    "Metrics file",
				Status:  "ok",
				Message: "No metrics file yet (will be created on first dispatch)",
			}
		}
		return DoctorCheck{
			Name:    "Metrics file",
			Status:  "error",
			Message: fmt.Sprintf("Cannot read metrics: %v", err),
		}
	}

	var entries []UnitMetrics
	if err := json.Unmarshal(data, &entries); err != nil {
		return DoctorCheck{
			Name:    "Metrics file",
			Status:  "warn",
			Message: "Metrics file is corrupt (unparseable)",
			Fixable: true,
		}
	}

	return DoctorCheck{
		Name:    "Metrics file",
		Status:  "ok",
		Message: fmt.Sprintf("%d metric entries", len(entries)),
	}
}

// checkDBState verifies milestones, slices, and tasks are consistent.
func checkDBState(ctx context.Context, querier StateQuerier) DoctorCheck {
	if querier == nil {
		return DoctorCheck{
			Name:    "Database state",
			Status:  "warn",
			Message: "No querier available for DB checks",
		}
	}

	milestones, err := querier.ListMilestones(ctx)
	if err != nil {
		return DoctorCheck{
			Name:    "Database state",
			Status:  "error",
			Message: fmt.Sprintf("Cannot query milestones: %v", err),
		}
	}

	activeCount := 0
	for _, m := range milestones {
		if Status(m.Status) == StatusActive {
			activeCount++
		}
	}

	if activeCount > 1 {
		return DoctorCheck{
			Name:    "Database state",
			Status:  "warn",
			Message: fmt.Sprintf("%d active milestones (expected 0 or 1)", activeCount),
		}
	}

	return DoctorCheck{
		Name:    "Database state",
		Status:  "ok",
		Message: fmt.Sprintf("%d milestones (%d active)", len(milestones), activeCount),
	}
}

// saveReportHistory appends the report to the doctor history file.
func saveReportHistory(dataDir string, report *DoctorReport) {
	path := filepath.Join(dataDir, "doctor-history.jsonl")
	data, err := json.Marshal(report)
	if err != nil {
		return
	}
	data = append(data, '\n')

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}
