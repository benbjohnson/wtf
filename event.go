package wtf

import (
	"context"
)

// Event type constants.
const (
	EventTypeDialValueChanged           = "dial:value_changed"
	EventTypeDialMembershipValueChanged = "dial_membership:value_changed"
)

// Event represents an event that occurs in the system. Currently there are only
// events for changes to a dial value or membership value. These events are
// eventually propagated out to connected users via WebSockets whenever changes
// occur so that the UI can update in real-time.
type Event struct {
	// Specifies the type of event that is occurring.
	Type string `json:"type"`

	// The actual data from the event. See related payload types below.
	Payload interface{} `json:"payload"`
}

// DialValueChangedPayload represents the payload for an Event object with a
// type of EventTypeDialValueChanged.
type DialValueChangedPayload struct {
	ID    int `json:"id"`
	Value int `json:"value"`
}

// DialMembershipValueChangedPayload represents the payload for an Event object
// with a type of EventTypeDialMembershipValueChanged.
type DialMembershipValueChangedPayload struct {
	ID    int `json:"id"`
	Value int `json:"value"`
}

// EventService represents a service for managing event dispatch and event
// listeners (aka subscriptions).
//
// Events are user-centric in this implementation although a more generic
// implementation may use a topic-centic model (e.g. "dial_value_changed(id=1)").
// The application has frequent reconnects so it's more efficient to subscribe
// for a single user instead of resubscribing to all their related topics.
type EventService interface {
	// Publishes an event to a user's event listeners.
	// If the user is not currently subscribed then this is a no-op.
	PublishEvent(userID int, event Event)

	// Creates a subscription for the current user's events.
	// Caller must call Subscription.Close() when done with the subscription.
	Subscribe(ctx context.Context) (Subscription, error)
}

// NopEventService returns an event service that does nothing.
func NopEventService() EventService { return &nopEventService{} }

type nopEventService struct{}

func (*nopEventService) PublishEvent(userID int, event Event) {}

func (*nopEventService) Subscribe(ctx context.Context) (Subscription, error) {
	panic("not implemented")
}

// Subscription represents a stream of events for a single user.
type Subscription interface {
	// Event stream for all user's event.
	C() <-chan Event

	// Closes the event stream channel and disconnects from the event service.
	Close() error
}
