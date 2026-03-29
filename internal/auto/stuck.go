package auto

import (
	"fmt"
	"sync"
)

// StuckDetector tracks dispatch outcomes in a per-unit sliding window.
// When more than 50% of results in a full window are failures, the unit
// is considered stuck. A nil StuckDetector or zero windowSize disables
// detection.
type StuckDetector struct {
	windowSize int
	mu         sync.Mutex
	windows    map[string]*ringBuffer
}

// NewStuckDetector creates a StuckDetector with the given window size.
// A windowSize <= 0 disables detection.
func NewStuckDetector(windowSize int) *StuckDetector {
	if windowSize <= 0 {
		return nil
	}
	return &StuckDetector{
		windowSize: windowSize,
		windows:    make(map[string]*ringBuffer),
	}
}

// UnitKey returns the canonical key for a Unit used by stuck detection.
func UnitKey(u Unit) string {
	return fmt.Sprintf("%s/%s/%s", u.MilestoneID, u.SliceID, u.TaskID)
}

// Record appends a pass/fail outcome to the unit's sliding window.
func (d *StuckDetector) Record(unitKey string, passed bool) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	rb, ok := d.windows[unitKey]
	if !ok {
		rb = newRingBuffer(d.windowSize)
		d.windows[unitKey] = rb
	}
	rb.push(passed)
}

// IsStuck returns true when the unit's window is full and strictly more
// than 50% of the entries are failures.
func (d *StuckDetector) IsStuck(unitKey string) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	rb, ok := d.windows[unitKey]
	if !ok {
		return false
	}
	return rb.isStuck()
}

// ringBuffer is a fixed-size circular buffer of booleans (true = passed).
type ringBuffer struct {
	data  []bool
	size  int
	count int // Total entries written (capped at size for fullness check).
	head  int // Next write position.
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		data: make([]bool, size),
		size: size,
	}
}

func (rb *ringBuffer) push(passed bool) {
	rb.data[rb.head] = passed
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// isStuck returns true when the buffer is full and strictly more than
// 50% of entries are failures.
func (rb *ringBuffer) isStuck() bool {
	if rb.count < rb.size {
		return false
	}
	failures := 0
	for i := range rb.size {
		if !rb.data[i] {
			failures++
		}
	}
	// Strictly more than 50%.
	return failures*2 > rb.size
}
