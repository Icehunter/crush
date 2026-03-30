package auto

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecoverFromCrash_NoLockFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	info, err := RecoverFromCrash(context.Background(), lockPath, &fakeQuerier{})
	require.NoError(t, err)
	require.Equal(t, RecoveryNone, info.Action)
}

func TestRecoverFromCrash_LiveProcess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	// Write a lock with our own PID (alive).
	payload := lockPayload{
		PID:       os.Getpid(),
		StartedAt: time.Now(),
		UnitType:  string(UnitExecuteTask),
		UnitID:    "M001/S01/T01",
	}
	data, _ := json.Marshal(payload)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	info, err := RecoverFromCrash(context.Background(), lockPath, &fakeQuerier{})
	require.NoError(t, err)
	require.Equal(t, RecoveryNone, info.Action, "live process should not trigger recovery")
}

func TestRecoverFromCrash_DeadProcess_NoUnitInfo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	// Write a lock with PID 1 (almost certainly not us, but alive on most systems).
	// Use PID 99999999 which is extremely unlikely to be alive.
	payload := lockPayload{
		PID:       99999999,
		StartedAt: time.Now().Add(-5 * time.Minute),
	}
	data, _ := json.Marshal(payload)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	info, err := RecoverFromCrash(context.Background(), lockPath, &fakeQuerier{})
	require.NoError(t, err)
	require.Equal(t, RecoveryRedispatch, info.Action)
}

func TestRecoverFromCrash_DeadProcess_UnitNotCompleted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {{ID: "S01", Title: "Auth", Status: "active", Phase: "executing", SortOrder: 1}},
		},
		tasks: map[string][]TaskRow{
			"S01": {{ID: "T01", Title: "Login", Status: "active", SortOrder: 1}},
		},
	}

	// The crashed unit matches what DeriveState would return next.
	nextUnit, _ := DeriveState(context.Background(), q)
	payload := lockPayload{
		PID:         99999999,
		StartedAt:   time.Now().Add(-5 * time.Minute),
		UnitType:    string(nextUnit.Type),
		UnitID:      nextUnit.String(),
		MilestoneID: nextUnit.MilestoneID,
	}
	data, _ := json.Marshal(payload)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	info, err := RecoverFromCrash(context.Background(), lockPath, q)
	require.NoError(t, err)
	require.Equal(t, RecoveryRedispatch, info.Action)
	require.False(t, info.UnitCompleted)
}

func TestRecoverFromCrash_DeadProcess_UnitCompleted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	q := &fakeQuerier{
		milestones: []MilestoneRow{{ID: "M001", Title: "Test", Status: "active", Phase: "executing"}},
		slices: map[string][]SliceRow{
			"M001": {{ID: "S01", Title: "Auth", Status: "active", Phase: "executing", SortOrder: 1}},
		},
		tasks: map[string][]TaskRow{
			"S01": {{ID: "T01", Title: "Login", Status: "active", SortOrder: 1}},
		},
	}

	// The crashed unit does NOT match what DeriveState returns — it already advanced.
	payload := lockPayload{
		PID:         99999999,
		StartedAt:   time.Now().Add(-5 * time.Minute),
		UnitType:    string(UnitExecuteTask),
		UnitID:      "some-previous-unit",
		MilestoneID: "M001",
	}
	data, _ := json.Marshal(payload)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	info, err := RecoverFromCrash(context.Background(), lockPath, q)
	require.NoError(t, err)
	require.Equal(t, RecoverySkip, info.Action)
	require.True(t, info.UnitCompleted)
}

func TestUpdateLockUnit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	// Write initial lock.
	initial := lockPayload{
		PID:       os.Getpid(),
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(initial)
	require.NoError(t, os.WriteFile(lockPath, data, 0o644))

	// Update with unit info.
	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Login",
	}
	require.NoError(t, UpdateLockUnit(lockPath, unit))

	// Read back and verify.
	updated, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	var p lockPayload
	require.NoError(t, json.Unmarshal(updated, &p))
	require.Equal(t, string(UnitExecuteTask), p.UnitType)
	require.Equal(t, unit.String(), p.UnitID)
	require.Equal(t, "M001", p.MilestoneID)
	require.Equal(t, os.Getpid(), p.PID)
}

func TestRecoverFromCrash_CorruptLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	// Write garbage to lock file.
	require.NoError(t, os.WriteFile(lockPath, []byte("not json"), 0o644))

	info, err := RecoverFromCrash(context.Background(), lockPath, &fakeQuerier{})
	require.NoError(t, err)
	require.Equal(t, RecoveryNone, info.Action, "corrupt lock should be treated as no crash")
}
