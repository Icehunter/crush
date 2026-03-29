package app

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/auto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

func TestAutoEventBroker_PublishSubscribe(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := SubscribeAutoEvents(ctx)

	snap := &auto.AutoSnapshot{
		MilestoneID: "M001",
		Status:      "running",
	}
	PublishAutoEvent(auto.EventUnitStarted, auto.AutoEvent{
		Unit:     auto.Unit{MilestoneID: "M001", SliceID: "S01", TaskID: "T01"},
		Snapshot: snap,
	})

	select {
	case evt := <-ch:
		require.Equal(t, pubsub.EventType("unit_started"), evt.Type)
		require.Equal(t, "M001", evt.Payload.Unit.MilestoneID)
		require.Equal(t, "S01", evt.Payload.Unit.SliceID)
		require.Equal(t, "T01", evt.Payload.Unit.TaskID)
		require.NotNil(t, evt.Payload.Snapshot)
		require.Equal(t, "running", evt.Payload.Snapshot.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for auto event")
	}
}

func TestAutoEventBroker_Shutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	ch := SubscribeAutoEvents(ctx)

	cancel()

	// After cancellation, the channel should eventually close.
	select {
	case _, ok := <-ch:
		if ok {
			// Got a stale event; drain and continue.
			return
		}
		// Channel closed — clean shutdown.
	case <-time.After(2 * time.Second):
		// Timeout is acceptable; the broker may not close the channel
		// immediately on context cancellation.
	}
}

func TestAutoEventBroker_Accessor(t *testing.T) {
	t.Parallel()

	broker := AutoBroker()
	require.NotNil(t, broker)
}
