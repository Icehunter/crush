package auto

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// sequentialMockVerifier returns pre-configured VerificationResult slices in
// order. Each call to RunVerification pops the next response. If exhausted it
// returns all-passing.
type sequentialMockVerifier struct {
	mu        sync.Mutex
	responses [][]VerificationResult
	callCount int
}

func (v *sequentialMockVerifier) RunVerification(_ context.Context, _ string) ([]VerificationResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	idx := v.callCount
	v.callCount++
	if idx < len(v.responses) {
		return v.responses[idx], nil
	}
	// Default: all passing.
	return []VerificationResult{{Command: "true", Passed: true}}, nil
}

// neverCalledVerifier panics if RunVerification is invoked.
type neverCalledVerifier struct{}

func (v *neverCalledVerifier) RunVerification(_ context.Context, _ string) ([]VerificationResult, error) {
	panic("verifier should not be called for non-task units")
}

// collectEvents drains events from a broker subscription until count events
// are received or timeout elapses.
func collectEvents(ch <-chan pubsub.Event[AutoEvent], count int, timeout time.Duration) []pubsub.Event[AutoEvent] {
	var events []pubsub.Event[AutoEvent]
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-ch:
			events = append(events, ev)
			if len(events) >= count {
				return events
			}
		case <-deadline:
			return events
		}
	}
}

// TestIntegration_VerifyRetrySucceed proves the dispatch→fail→retry→succeed→advance path.
func TestIntegration_VerifyRetrySucceed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// First verification fails, second succeeds.
	verifier := &sequentialMockVerifier{
		responses: [][]VerificationResult{
			{{Command: "go test ./...", ExitCode: 1, Passed: false, Stderr: "FAIL pkg"}},
			{{Command: "go test ./...", ExitCode: 0, Passed: true}},
		},
	}

	eng := NewEngine(querier, sessions, dispatch, advancer, verifier, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	// Subscribe to events.
	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx := context.Background()
	err := eng.Step(ctx, "M001")
	require.NoError(t, err)

	// Dispatcher called twice: original + retry with diagnostic.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 2, "should dispatch original + retry")
	retryPrompt := dispatch.calls[1].Prompt
	dispatch.mu.Unlock()
	require.Contains(t, retryPrompt, "VERIFICATION FAILED", "retry prompt must contain diagnostic header")
	require.Contains(t, retryPrompt, "FAIL pkg", "retry prompt must contain truncated failure output")

	// Advancer called exactly once — status advanced after successful retry.
	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 1, "should advance once after successful retry")
	advancer.mu.Unlock()

	// Collect events: UnitStarted, VerificationStarted, VerificationFailed,
	// VerificationStarted (retry), VerificationPassed, UnitCompleted = 6.
	collected := collectEvents(events, 6, 3*time.Second)
	types := make([]pubsub.EventType, len(collected))
	for i, ev := range collected {
		types[i] = ev.Type
	}
	require.Equal(t, []pubsub.EventType{
		EventUnitStarted,
		EventVerificationStarted,
		EventVerificationFailed,
		EventVerificationStarted,
		EventVerificationPassed,
		EventUnitCompleted,
	}, types, "events should follow verify→retry→succeed flow")
}

// TestIntegration_VerifyRetryFail proves the dispatch→fail→retry→fail→no-advance path.
func TestIntegration_VerifyRetryFail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unit := Unit{
		Type:        UnitExecuteTask,
		MilestoneID: "M001",
		SliceID:     "S01",
		TaskID:      "T01",
		Title:       "Execute task T01",
	}

	querier := &fixedSequenceQuerier{units: []Unit{unit}}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// Both verifications fail.
	verifier := &sequentialMockVerifier{
		responses: [][]VerificationResult{
			{{Command: "go test ./...", ExitCode: 1, Passed: false, Stderr: "FAIL first"}},
			{{Command: "go test ./...", ExitCode: 1, Passed: false, Stderr: "FAIL second"}},
		},
	}

	eng := NewEngine(querier, sessions, dispatch, advancer, verifier, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events := broker.Subscribe(subCtx)

	ctx := context.Background()
	err := eng.Step(ctx, "M001")
	require.Error(t, err, "Step should return error when verification fails after retry")
	require.Contains(t, err.Error(), "verification failed after retry")

	// Dispatcher called twice: original + retry.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 2, "should dispatch original + retry")
	dispatch.mu.Unlock()

	// Advancer NOT called — status not advanced.
	advancer.mu.Lock()
	require.Empty(t, advancer.advanced, "should not advance when verification fails after retry")
	advancer.mu.Unlock()

	// Collect events: UnitStarted, VerificationStarted, VerificationFailed,
	// VerificationStarted (retry), VerificationFailed (retry) = 5
	// No UnitCompleted.
	collected := collectEvents(events, 5, 3*time.Second)
	types := make([]pubsub.EventType, len(collected))
	for i, ev := range collected {
		types[i] = ev.Type
	}
	require.Equal(t, []pubsub.EventType{
		EventUnitStarted,
		EventVerificationStarted,
		EventVerificationFailed,
		EventVerificationStarted,
		EventVerificationFailed,
	}, types, "events should show two verification failures, no UnitCompleted")
}

// TestIntegration_VerifySkippedForNonTaskUnits proves verification only runs
// for execute_task units.
func TestIntegration_VerifySkippedForNonTaskUnits(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	units := []Unit{
		{Type: UnitResearch, MilestoneID: "M001", SliceID: "S01", Title: "Research S01"},
		{Type: UnitPlanSlice, MilestoneID: "M001", SliceID: "S01", Title: "Plan S01"},
	}

	querier := &fixedSequenceQuerier{units: units}
	sessions := &mockSessionCreator{}
	dispatch := &recordingDispatcher{}
	advancer := &mockAdvancer{querier: querier}
	broker := pubsub.NewBroker[AutoEvent]()

	// This verifier panics if called — proving it is never invoked.
	verifier := &neverCalledVerifier{}

	eng := NewEngine(querier, sessions, dispatch, advancer, verifier, nil, 0, nil, nil, broker, dir, slog.Default(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := eng.Run(ctx, "M001")
	require.NoError(t, err)

	// Both units dispatched normally.
	dispatch.mu.Lock()
	require.Len(t, dispatch.calls, 2, "should dispatch both non-task units")
	dispatch.mu.Unlock()

	// Both units advanced.
	advancer.mu.Lock()
	require.Len(t, advancer.advanced, 2, "both units should be advanced")
	advancer.mu.Unlock()
}
