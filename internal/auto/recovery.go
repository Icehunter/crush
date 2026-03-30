package auto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// CrashRecoveryInfo describes the state at time of crash, reconstructed
// from the lock file payload and DB state.
type CrashRecoveryInfo struct {
	// CrashedUnit is the unit that was active when the crash occurred.
	// Zero-value if no unit info was recorded in the lock.
	CrashedUnit Unit `json:"crashed_unit"`
	// CrashedAt is the timestamp from the lock file.
	CrashedAt time.Time `json:"crashed_at"`
	// UnitCompleted is true if the DB shows the unit's status was
	// advanced before the crash (i.e., the crash happened after dispatch
	// succeeded). In this case the unit should be skipped on recovery.
	UnitCompleted bool `json:"unit_completed"`
	// Action describes what the recovery recommends.
	Action RecoveryAction `json:"action"`
}

// RecoveryAction describes what to do on recovery.
type RecoveryAction string

const (
	// RecoverySkip means the unit was already completed — skip it.
	RecoverySkip RecoveryAction = "skip"
	// RecoveryRedispatch means the unit did not complete — re-dispatch it.
	RecoveryRedispatch RecoveryAction = "redispatch"
	// RecoveryNone means no crash was detected.
	RecoveryNone RecoveryAction = "none"
)

// UpdateLockUnit rewrites the lock file to include the currently active unit.
// This is called before each dispatch so that crash recovery knows what was
// running. Only updates if the lock is already held (the file exists and we
// own it).
func UpdateLockUnit(lockPath string, unit Unit) error {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return fmt.Errorf("read lock for update: %w", err)
	}

	var p lockPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("unmarshal lock for update: %w", err)
	}

	p.UnitType = string(unit.Type)
	p.UnitID = unit.String()
	p.MilestoneID = unit.MilestoneID

	updated, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal updated lock: %w", err)
	}

	return os.WriteFile(lockPath, updated, 0o644)
}

// RecoverFromCrash inspects the lock file (if it exists from a dead process)
// and compares against the DB state to determine recovery action. This should
// be called before the main loop starts.
//
// Returns RecoveryNone info if no crash state is detected.
func RecoverFromCrash(ctx context.Context, lockPath string, querier StateQuerier) (*CrashRecoveryInfo, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &CrashRecoveryInfo{Action: RecoveryNone}, nil
		}
		return nil, fmt.Errorf("read crash lock: %w", err)
	}

	var p lockPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return &CrashRecoveryInfo{Action: RecoveryNone}, nil
	}

	// If the process is still alive, this isn't a crash — it's a concurrent run.
	if isProcessAlive(p.PID) {
		return &CrashRecoveryInfo{Action: RecoveryNone}, nil
	}

	// Dead process — this is crash recovery territory.
	info := &CrashRecoveryInfo{
		CrashedAt: p.StartedAt,
	}

	// If no unit info in the lock, we can't determine what was running.
	if p.UnitType == "" {
		info.Action = RecoveryRedispatch
		return info, nil
	}

	// Reconstruct what the unit was.
	info.CrashedUnit = Unit{
		Type:        UnitType(p.UnitType),
		MilestoneID: p.MilestoneID,
		Title:       p.UnitID,
	}

	// Check DB state to see if the unit completed before the crash.
	unit, err := DeriveState(ctx, querier)
	if err != nil {
		return nil, fmt.Errorf("derive state for crash recovery: %w", err)
	}

	// If the derived next unit is the same one that crashed, it didn't complete.
	if unit.String() == info.CrashedUnit.Title {
		info.Action = RecoveryRedispatch
		info.UnitCompleted = false
	} else {
		// The unit isn't the next one anymore, so it must have completed
		// (status was advanced) before the crash.
		info.Action = RecoverySkip
		info.UnitCompleted = true
	}

	return info, nil
}
