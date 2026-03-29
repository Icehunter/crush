package auto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStuckDetector_FullWindowMajorityFailures(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(4)
	key := "M001/S01/T01"

	// 3 failures out of 4 → stuck.
	d.Record(key, false)
	d.Record(key, false)
	d.Record(key, false)
	d.Record(key, true)

	require.True(t, d.IsStuck(key))
}

func TestStuckDetector_FiftyFiftyNotStuck(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(4)
	key := "M001/S01/T01"

	// 2 failures out of 4 → exactly 50%, not stuck (must be strictly >50%).
	d.Record(key, false)
	d.Record(key, false)
	d.Record(key, true)
	d.Record(key, true)

	require.False(t, d.IsStuck(key))
}

func TestStuckDetector_WindowSlides(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(4)
	key := "M001/S01/T01"

	// Fill with all failures → stuck.
	d.Record(key, false)
	d.Record(key, false)
	d.Record(key, false)
	d.Record(key, false)
	require.True(t, d.IsStuck(key))

	// Push two passes → 2 fail + 2 pass = 50% → not stuck.
	d.Record(key, true)
	d.Record(key, true)
	require.False(t, d.IsStuck(key))
}

func TestStuckDetector_EmptyWindowNotStuck(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(4)
	require.False(t, d.IsStuck("M001/S01/T01"))
}

func TestStuckDetector_PartialWindowNotStuck(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(4)
	key := "M001/S01/T01"

	// Only 2 failures in window of size 4 → not full → not stuck.
	d.Record(key, false)
	d.Record(key, false)

	require.False(t, d.IsStuck(key))
}

func TestStuckDetector_NilDetectorSafe(t *testing.T) {
	t.Parallel()
	var d *StuckDetector
	d.Record("M001/S01/T01", false) // Should not panic.
	require.False(t, d.IsStuck("M001/S01/T01"))
}

func TestStuckDetector_ZeroWindowSizeReturnsNil(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(0)
	require.Nil(t, d)
}

func TestStuckDetector_SinglePassClearsStuck(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(3)
	key := "M001/S01/T01"

	// All 3 failures → stuck.
	d.Record(key, false)
	d.Record(key, false)
	d.Record(key, false)
	require.True(t, d.IsStuck(key))

	// One pass pushes out oldest failure → 2 fail + 1 pass.
	// 2/3 > 50% → still stuck.
	d.Record(key, true)
	require.True(t, d.IsStuck(key))

	// Another pass → 1 fail + 2 pass → not stuck.
	d.Record(key, true)
	require.False(t, d.IsStuck(key))
}

func TestStuckDetector_MultipleUnitsIndependent(t *testing.T) {
	t.Parallel()
	d := NewStuckDetector(2)

	d.Record("M001/S01/T01", false)
	d.Record("M001/S01/T01", false)
	d.Record("M001/S01/T02", true)
	d.Record("M001/S01/T02", true)

	require.True(t, d.IsStuck("M001/S01/T01"))
	require.False(t, d.IsStuck("M001/S01/T02"))
}

func TestUnitKey(t *testing.T) {
	t.Parallel()
	u := Unit{MilestoneID: "M001", SliceID: "S01", TaskID: "T01"}
	require.Equal(t, "M001/S01/T01", UnitKey(u))
}
