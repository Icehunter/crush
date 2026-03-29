package auto

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// failNDispatcher fails the first N dispatches, then succeeds. Thread-safe.
type failNDispatcher struct {
	mu        sync.Mutex
	calls     []dispatchCall
	failCount int // Number of dispatches to fail.
}

func (d *failNDispatcher) RunWithForcedTier(_ context.Context, sessionID, prompt string, tier config.SelectedModelType) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, dispatchCall{SessionID: sessionID, Prompt: prompt, Tier: tier})
	if len(d.calls) <= d.failCount {
		return errSimulated
	}
	return nil
}

var errSimulated = errSentinel("simulated dispatch error")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

// TestIntegration_StuckRetrySucceed verifies: dispatcher fails enough to
// trigger stuck, diagnostic retry succeeds, engine continues.
func TestIntegration_StuckRetrySucceed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// We need the unit to be dispatched multiple times to fill the stuck
	// window. The querier will keep returning the same unit until the
	// advancer marks it done.
	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute T01",
	}

	// Window size 3: need >50% failures (i.e. 2+ out of 3).
	// We'll use a recording dispatcher + verifier. The engine's Run loop
	// retries on step errors. We want:
	//   step 1: dispatch succeeds, verification fails → record fail
	//   step 2: dispatch succeeds, verification fails → record fail
	//   step 3: stuck detected → diagnostic retry dispatched, verification passes → record pass, advance
	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Verifier: first two calls fail, third (diagnostic retry verify) succeeds.
	verifier := &sequentialMockVerifier{
		responses: [][]VerificationResult{
			// Step 1: initial verification fails.
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL"}},
			// Step 1: retry verification within runVerificationGate also fails.
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL"}},
			// Step 2 (retry from Run loop): initial verification fails.
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL"}},
			// Step 2: retry verification within runVerificationGate also fails.
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL"}},
			// Step 3: stuck detected → diagnostic retry verification succeeds.
			{{Command: "go test", ExitCode: 0, Passed: true}},
		},
	}

	detector := NewStuckDetector(2)
	eng := NewEngine(querier, sessions, dispatch, advancer, verifier, nil, 0, detector, nil, broker, dir, slog.Default(), nil)

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Advancer should have been called — the unit completed.
	advancer.mu.Lock()
	require.NotEmpty(t, advancer.advanced, "unit should have been advanced after stuck recovery")
	advancer.mu.Unlock()

	// Should have received EventUnitCompleted eventually.
	var gotCompleted bool
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == EventUnitCompleted {
				gotCompleted = true
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	require.True(t, gotCompleted, "should receive EventUnitCompleted after stuck recovery")
}

// TestIntegration_StuckRetryFail verifies: dispatcher keeps failing, stuck
// detected, diagnostic retry fails, engine pauses and publishes
// EventStuckDetected.
func TestIntegration_StuckRetryFail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// All verification calls fail.
	verifier := &sequentialMockVerifier{
		responses: [][]VerificationResult{
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL 1"}},
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL 2"}},
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL 3"}},
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL 4"}},
			// Stuck diagnostic retry verification also fails.
			{{Command: "go test", ExitCode: 1, Passed: false, Stderr: "FAIL 5"}},
		},
	}

	detector := NewStuckDetector(2)
	eng := NewEngine(querier, sessions, dispatch, advancer, verifier, nil, 0, detector, nil, broker, dir, slog.Default(), nil)

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err, "engine should return nil on pause, not error")

	// Engine should be paused.
	status := eng.Status()
	require.Equal(t, EnginePaused, status.State, "engine should be paused after stuck detection")

	// Advancer should NOT have been called.
	advancer.mu.Lock()
	require.Empty(t, advancer.advanced, "should not advance when stuck cannot be resolved")
	advancer.mu.Unlock()

	// Should have received EventStuckDetected.
	var gotStuck bool
	timeout := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == EventStuckDetected {
				gotStuck = true
				goto done2
			}
			if ev.Type == EventLoopPaused {
				goto done2
			}
		case <-timeout:
			goto done2
		}
	}
done2:
	require.True(t, gotStuck, "should receive EventStuckDetected")
}

// TestIntegration_StuckNotTriggeredBelowThreshold verifies: fewer failures
// than the threshold, engine proceeds normally.
func TestIntegration_StuckNotTriggeredBelowThreshold(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Verification passes on first attempt.
	verifier := &sequentialMockVerifier{
		responses: [][]VerificationResult{
			{{Command: "go test", ExitCode: 0, Passed: true}},
		},
	}

	// Large window — one pass is nowhere near stuck.
	detector := NewStuckDetector(5)
	eng := NewEngine(querier, sessions, dispatch, advancer, verifier, nil, 0, detector, nil, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Engine completed normally.
	status := eng.Status()
	require.Equal(t, EngineIdle, status.State, "engine should be idle after normal completion")

	// Advancer should have been called.
	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 1, "should advance normally")
	advancer.mu.Unlock()
}
