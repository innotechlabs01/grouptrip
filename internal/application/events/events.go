package events

import "time"

// Event is a domain event marker.
type Event struct {
	Type       string
	OccurredAt time.Time
	Payload    map[string]any
}

// EventSink receives domain events (projections, notifications, risk detection).
type EventSink interface {
	Publish(Event) error
}

// NoopSink is a do-nothing sink for tests and early scaffolding.
type NoopSink struct{}

func (NoopSink) Publish(Event) error { return nil }
