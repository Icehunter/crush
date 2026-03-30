package auto

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoctor_CleanState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
	}

	report := RunDoctor(context.Background(), dir, q, false)
	require.NotNil(t, report)

	for _, c := range report.Checks {
		require.NotEqual(t, "error", c.Status, "check %s: %s", c.Name, c.Message)
	}
}

func TestDoctor_StaleLock_NoFix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	// Write a stale lock (dead PID).
	payload := lockPayload{PID: 99999999, UnitType: "execute_task"}
	data, _ := json.Marshal(payload)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	report := RunDoctor(context.Background(), dir, nil, false)
	var lockCheck *DoctorCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Lock file" {
			lockCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, lockCheck)
	require.Equal(t, "warn", lockCheck.Status)
	require.True(t, lockCheck.Fixable)
	require.False(t, lockCheck.Fixed)

	// Lock file should still exist.
	_, err := os.Stat(lockPath)
	require.NoError(t, err)
}

func TestDoctor_StaleLock_WithFix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	payload := lockPayload{PID: 99999999}
	data, _ := json.Marshal(payload)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	report := RunDoctor(context.Background(), dir, nil, true)
	var lockCheck *DoctorCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Lock file" {
			lockCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, lockCheck)
	require.True(t, lockCheck.Fixed)

	// Lock file should be removed.
	_, err := os.Stat(lockPath)
	require.True(t, os.IsNotExist(err))
}

func TestDoctor_CorruptLock_WithFix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	require.NoError(t, os.WriteFile(lockPath, []byte("not json"), 0o644))

	report := RunDoctor(context.Background(), dir, nil, true)
	var lockCheck *DoctorCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Lock file" {
			lockCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, lockCheck)
	require.Equal(t, "error", lockCheck.Status)
	require.True(t, lockCheck.Fixed)
}

func TestDoctor_MultipleActiveMilestones(t *testing.T) {
	t.Parallel()
	q := &fakeQuerier{
		milestones: []MilestoneRow{
			{ID: "M001", Title: "First", Status: "active"},
			{ID: "M002", Title: "Second", Status: "active"},
		},
	}

	report := RunDoctor(context.Background(), t.TempDir(), q, false)
	var dbCheck *DoctorCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Database state" {
			dbCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, dbCheck)
	require.Equal(t, "warn", dbCheck.Status)
	require.Contains(t, dbCheck.Message, "2 active milestones")
}

func TestDoctor_Summary(t *testing.T) {
	t.Parallel()
	report := &DoctorReport{
		Checks: []DoctorCheck{
			{Name: "Lock", Status: "ok", Message: "all good"},
			{Name: "DB", Status: "warn", Message: "issues found"},
			{Name: "Metrics", Status: "error", Message: "corrupt"},
		},
	}

	summary := report.Summary()
	require.Contains(t, summary, "1 ok")
	require.Contains(t, summary, "1 warnings")
	require.Contains(t, summary, "1 errors")
	require.Contains(t, summary, "✓ Lock")
	require.Contains(t, summary, "⚠ DB")
	require.Contains(t, summary, "✗ Metrics")
}

func TestDoctor_ReportHistory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	RunDoctor(context.Background(), dir, nil, false)

	// History file should exist.
	historyPath := filepath.Join(dir, "doctor-history.jsonl")
	_, err := os.Stat(historyPath)
	require.NoError(t, err)
}
