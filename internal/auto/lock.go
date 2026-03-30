package auto

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const lockFileName = "auto.lock"

// ErrLockHeld is returned when the lock file is held by a live process.
var ErrLockHeld = errors.New("auto lock is held by another process")

// lockPayload is the JSON content stored inside the lock file.
type lockPayload struct {
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"started_at"`
	UnitType    string    `json:"unit_type,omitempty"`
	UnitID      string    `json:"unit_id,omitempty"`
	MilestoneID string    `json:"milestone_id,omitempty"`
}

// LockFile manages a file-based lock that prevents concurrent auto-mode
// instances. The lock file contains JSON with the owning PID and start
// timestamp. Stale locks (owner PID no longer running) are automatically
// reclaimed.
type LockFile struct {
	path string
	held bool
}

// NewLockFile creates a LockFile targeting dir/auto.lock.
func NewLockFile(dir string) *LockFile {
	return &LockFile{path: filepath.Join(dir, lockFileName)}
}

// Acquire attempts to create and hold the lock file. It returns
// ErrLockHeld when another live process holds the lock. Stale locks
// (dead PID) are removed and re-acquired transparently.
func (l *LockFile) Acquire() error {
	if l.held {
		return nil // Already held by us.
	}

	payload, err := json.Marshal(lockPayload{
		PID:       os.Getpid(),
		StartedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("marshal lock payload: %w", err)
	}

	// Try atomic create first (O_EXCL fails if file exists).
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		// We won the race — write our payload.
		_, writeErr := f.Write(payload)
		f.Close()
		if writeErr != nil {
			_ = os.Remove(l.path)
			return fmt.Errorf("write lock file: %w", writeErr)
		}
		l.held = true
		return nil
	}

	if !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("create lock file: %w", err)
	}

	// Lock file exists — check if owner is alive.
	data, readErr := os.ReadFile(l.path)
	if readErr != nil {
		// File vanished between OpenFile and ReadFile — retry once.
		return l.retryAcquire(payload)
	}

	var p lockPayload
	if err := json.Unmarshal(data, &p); err != nil {
		// File exists but can't be parsed — likely being written right now.
		// Treat as held to avoid racing the writer.
		return fmt.Errorf("%w: lock file exists but is unreadable", ErrLockHeld)
	}
	if isProcessAlive(p.PID) {
		return fmt.Errorf("%w: pid %d (started %s)", ErrLockHeld, p.PID, p.StartedAt.Format(time.RFC3339))
	}

	// Stale lock — remove and retry.
	_ = os.Remove(l.path)
	return l.retryAcquire(payload)
}

// retryAcquire attempts a single retry of the atomic create.
func (l *LockFile) retryAcquire(payload []byte) error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: lost race to acquire lock", ErrLockHeld)
		}
		return fmt.Errorf("create lock file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(payload); err != nil {
		_ = os.Remove(l.path)
		return fmt.Errorf("write lock file: %w", err)
	}
	l.held = true
	return nil
}

// Release removes the lock file if we hold it.
func (l *LockFile) Release() error {
	if !l.held {
		return nil
	}
	err := os.Remove(l.path)
	l.held = false
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove lock file: %w", err)
	}
	return nil
}

// Path returns the absolute path of the lock file.
func (l *LockFile) Path() string {
	return l.path
}

// isProcessAlive checks whether a process with the given PID is running.
// It sends signal 0 which performs the check without actually sending a
// signal.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 tests for existence without affecting the process.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
