package auto

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLockFile_AcquireRelease(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lf := NewLockFile(dir)

	require.NoError(t, lf.Acquire())
	require.FileExists(t, filepath.Join(dir, lockFileName))

	// Read and verify payload.
	data, err := os.ReadFile(filepath.Join(dir, lockFileName))
	require.NoError(t, err)
	var p lockPayload
	require.NoError(t, json.Unmarshal(data, &p))
	require.Equal(t, os.Getpid(), p.PID)
	require.WithinDuration(t, time.Now(), p.StartedAt, 5*time.Second)

	require.NoError(t, lf.Release())
	require.NoFileExists(t, filepath.Join(dir, lockFileName))
}

func TestLockFile_DoubleAcquireSameInstance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lf := NewLockFile(dir)

	require.NoError(t, lf.Acquire())
	// Second acquire on the same instance is a no-op.
	require.NoError(t, lf.Acquire())
	require.NoError(t, lf.Release())
}

func TestLockFile_DoubleAcquireDifferentInstance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lf1 := NewLockFile(dir)
	lf2 := NewLockFile(dir)

	require.NoError(t, lf1.Acquire())
	err := lf2.Acquire()
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrLockHeld), "expected ErrLockHeld, got: %v", err)

	require.NoError(t, lf1.Release())
	// Now lf2 should succeed.
	require.NoError(t, lf2.Acquire())
	require.NoError(t, lf2.Release())
}

func TestLockFile_StaleLockReclaimed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Write a lock with a PID that is definitely not running.
	stalePID := 2147483647 // Max 32-bit PID — almost certainly dead.
	data, err := json.Marshal(lockPayload{PID: stalePID, StartedAt: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, lockFileName), data, 0o644))

	lf := NewLockFile(dir)
	require.NoError(t, lf.Acquire(), "should reclaim stale lock")
	require.NoError(t, lf.Release())
}

func TestLockFile_ReleaseWithoutAcquire(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lf := NewLockFile(dir)
	// Release when not held is a no-op.
	require.NoError(t, lf.Release())
}

func TestLockFile_ConcurrentAcquire(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	const goroutines = 10
	results := make(chan error, goroutines)
	for range goroutines {
		go func() {
			lf := NewLockFile(dir)
			results <- lf.Acquire()
		}()
	}

	var acquired int
	for range goroutines {
		if err := <-results; err == nil {
			acquired++
		}
	}
	require.Equal(t, 1, acquired, "exactly one goroutine should acquire the lock")
}
