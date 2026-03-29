package app

import (
	"context"

	"github.com/charmbracelet/crush/internal/auto"
	"github.com/charmbracelet/crush/internal/pubsub"
)

var autoBroker = pubsub.NewBroker[auto.AutoEvent]()

// SubscribeAutoEvents returns a channel for auto-mode events.
func SubscribeAutoEvents(ctx context.Context) <-chan pubsub.Event[auto.AutoEvent] {
	return autoBroker.Subscribe(ctx)
}

// PublishAutoEvent publishes an auto-mode event.
func PublishAutoEvent(eventType pubsub.EventType, event auto.AutoEvent) {
	autoBroker.Publish(eventType, event)
}

// AutoBroker returns the auto-mode event broker for engine integration.
func AutoBroker() *pubsub.Broker[auto.AutoEvent] {
	return autoBroker
}
